#!/usr/bin/env bash
set -euo pipefail

# End-to-end chubo core flow in QEMU:
# install -> runtime mTLS -> upgrade -> rollback -> support bundle
#
# This script is Linux/amd64 oriented for CI. It uses `talosctl cluster create dev`
# to get a host-reachable node IP and drives the rest over the OS API.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALOS_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${TALOS_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chubo}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL:--tags grpcnotrace,chubo}"
ARCH="${ARCH:-amd64}"
SKIP_BUILD="${SKIP_BUILD:-0}"
WITH_HELPERS="${WITH_HELPERS:-0}"
BUILDX_BUILDER="${BUILDX_BUILDER:-local}"
HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"
TALOSCTL="${TALOSCTL:-${TALOS_ROOT}/_out/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}"

RUN_ID="${RUN_ID:-$RANDOM}"
BASE_NET_OCTET="${BASE_NET_OCTET:-$((100 + RANDOM % 100))}"
BASE_NET_SUBNET="${BASE_NET_SUBNET:-$((10 + RANDOM % 200))}"
CONTROL_PLANE_PORT="${CONTROL_PLANE_PORT:-$((7400 + RANDOM % 400))}"
CLUSTER_CREATE_MAX_ATTEMPTS="${CLUSTER_CREATE_MAX_ATTEMPTS:-3}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-core-${RUN_ID}}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-core-state-${RUN_ID}}"
WORKDIR="${WORKDIR:-/tmp/chubo-core-work-${RUN_ID}}"
NODE_IP="${NODE_IP:-10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.2}"
CIDR="${CIDR:-10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.0/24}"
INSTALL_DISK="${INSTALL_DISK:-/dev/vda}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-core-reg-${RUN_ID}}"
REGISTRY_PORT="${REGISTRY_PORT:-$((5100 + RANDOM % 300))}"
REGISTRY_LOCAL_ADDR="${REGISTRY_LOCAL_ADDR:-}"
REGISTRY_NODE_ADDR="${REGISTRY_NODE_ADDR:-}"
INSTALLER_BASE_IMAGE_LOCAL="${INSTALLER_BASE_IMAGE_LOCAL:-}"
INSTALLER_IMAGE_LOCAL="${INSTALLER_IMAGE_LOCAL:-}"
INSTALLER_IMAGE_NODE="${INSTALLER_IMAGE_NODE:-}"
REGISTRY_MIRROR_NODE="${REGISTRY_MIRROR_NODE:-}"

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1200}"
SLEEP_SECONDS="${SLEEP_SECONDS:-3}"
MAINTENANCE_PERSIST_SECONDS="${MAINTENANCE_PERSIST_SECONDS:-30}"
MAINTENANCE_FALLBACK_SECONDS="${MAINTENANCE_FALLBACK_SECONDS:-180}"
ACTION_REBOOT_WAIT_SECONDS="${ACTION_REBOOT_WAIT_SECONDS:-600}"
PROBE_TIMEOUT_SECONDS="${PROBE_TIMEOUT_SECONDS:-8}"
TIMEOUT_BIN="${TIMEOUT_BIN:-$(command -v timeout || command -v gtimeout || true)}"
SUPPORT_OUT="${SUPPORT_OUT:-${WORKDIR}/support.zip}"
CLUSTER_LOGS_OUT="${CLUSTER_LOGS_OUT:-${WORKDIR}/cluster-logs.tar.gz}"
CLUSTER_SUPPORT_OUT="${CLUSTER_SUPPORT_OUT:-${WORKDIR}/cluster-support.zip}"
HELPERS_DIR="${HELPERS_DIR:-${WORKDIR}/helpers}"
HELPERS_VALIDATION_OUT="${HELPERS_VALIDATION_OUT:-${WORKDIR}/helpers-validation.txt}"

SECRETS_FILE="${WORKDIR}/secrets.yaml"
MACHINECONFIG_INSTALL="${WORKDIR}/machineconfig-install.yaml"
MACHINECONFIG_RUNTIME="${WORKDIR}/machineconfig-runtime.yaml"
TALOSCONFIG_FILE="${WORKDIR}/talosconfig"
SUPPORT_LISTING="${WORKDIR}/support-listing.txt"
CLUSTER_CREATE_LOG="${WORKDIR}/cluster-create.log"

cluster_created=0
registry_started=0
runtime_config_applied=0
chubo_status_configured=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--skip-build)
		SKIP_BUILD=1
		;;
	--with-helpers)
		WITH_HELPERS=1
		;;
	-h | --help)
		echo "usage: $0 [--skip-build] [--with-helpers]"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		echo "usage: $0 [--skip-build] [--with-helpers]" >&2
		exit 2
		;;
	esac

	shift
done

require_cmd() {
	local cmd="$1"

	if ! command -v "${cmd}" >/dev/null 2>&1; then
		echo "required command not found: ${cmd}" >&2

		exit 1
	fi
}

ensure_buildx_builder() {
	# Buildx "docker" driver (default with colima) can hang after large builds.
	# Prefer a docker-container builder for deterministic, non-hanging local runs.
	if docker buildx inspect --builder "${BUILDX_BUILDER}" --bootstrap >/dev/null 2>&1; then
		# Talos build relies on `RUN --security=insecure`; require the builder to allow that entitlement.
		if docker buildx inspect --builder "${BUILDX_BUILDER}" 2>/dev/null | grep -q 'BuildKit daemon flags:.*security.insecure'; then
			return 0
		fi

		echo "buildx builder ${BUILDX_BUILDER} missing security.insecure entitlement, recreating"
		docker buildx rm "${BUILDX_BUILDER}" >/dev/null 2>&1 || true
	fi

	echo "creating buildx builder: ${BUILDX_BUILDER}"
	docker buildx create --name "${BUILDX_BUILDER}" --driver docker-container \
		--buildkitd-flags '--allow-insecure-entitlement security.insecure' >/dev/null
	docker buildx inspect --builder "${BUILDX_BUILDER}" --bootstrap >/dev/null
}

refresh_registry_refs() {
	: "${REGISTRY_LOCAL_ADDR:=localhost:${REGISTRY_PORT}}"
	: "${REGISTRY_NODE_ADDR:=10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.1:${REGISTRY_PORT}}"
	: "${INSTALLER_BASE_IMAGE_LOCAL:=${REGISTRY_LOCAL_ADDR}/chubo/installer-base:dev}"
	: "${INSTALLER_IMAGE_LOCAL:=${REGISTRY_LOCAL_ADDR}/chubo/installer:dev}"
	: "${INSTALLER_IMAGE_NODE:=${REGISTRY_NODE_ADDR}/chubo/installer:dev}"
	: "${REGISTRY_MIRROR_NODE:=${REGISTRY_NODE_ADDR}=http://${REGISTRY_NODE_ADDR}}"
}

start_registry() {
	local attempt=1

	while true; do
		echo "starting local OCI registry on :${REGISTRY_PORT} (attempt ${attempt}/20)"
		docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1 || true

		if docker run -d --rm --name "${REGISTRY_NAME}" -p "${REGISTRY_PORT}:5000" registry:2 >/dev/null 2>&1; then
			registry_started=1
			refresh_registry_refs
			return 0
		fi

		if ((attempt >= 20)); then
			echo "failed to start local registry after ${attempt} attempts" >&2
			return 1
		fi

		REGISTRY_PORT=$((5400 + RANDOM % 500))
		attempt=$((attempt + 1))
	done
}

wait_until() {
	local description="$1"
	local timeout_seconds="$2"
	shift 2

	local deadline=$((SECONDS + timeout_seconds))

	while true; do
		if run_probe "$@" >/dev/null 2>&1; then
			return 0
		fi

		if ((SECONDS >= deadline)); then
			echo "timed out waiting for: ${description}" >&2

			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
}

run_probe() {
	if [[ -n "${TIMEOUT_BIN}" ]]; then
		"${TIMEOUT_BIN}" "${PROBE_TIMEOUT_SECONDS}" "$@"
	else
		"$@"
	fi
}

run_cluster_create() {
	local monitor_path="${STATE_DIR}/${CLUSTER_NAME}/${CLUSTER_NAME}-controlplane-1.monitor"

	if ((${#monitor_path} >= 104)); then
		echo "qemu monitor path too long (${#monitor_path} bytes): ${monitor_path}" >&2
		echo "set shorter STATE_DIR and/or CLUSTER_NAME" >&2
		return 1
	fi

	"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster create dev \
		--arch "${ARCH}" \
		--cidr "${CIDR}" \
		--control-plane-port "${CONTROL_PLANE_PORT}" \
		--controlplanes 1 \
		--workers 0 \
		--disk 12288 \
		--cpus 2.0 \
		--memory 2.0GiB \
		--skip-injecting-config \
		--with-cluster-discovery=false \
		--with-init-node=false \
		--wait=false \
		--vmlinuz-path "${ARTIFACTS}/vmlinuz-${ARCH}" \
		--initrd-path "${ARTIFACTS}/initramfs-${ARCH}.xz" \
		--install-image "${INSTALLER_IMAGE_NODE}" \
		--registry-mirror "${REGISTRY_MIRROR_NODE}"
}

create_cluster_with_retry() {
	local attempt=1

	while true; do
		echo "creating single-node QEMU cluster in maintenance mode (attempt ${attempt}/${CLUSTER_CREATE_MAX_ATTEMPTS})"
		if run_cluster_create 2>&1 | tee "${CLUSTER_CREATE_LOG}"; then
			cluster_created=1
			return 0
		fi

		if grep -Eq 'interface bridge[0-9]+ not found' "${CLUSTER_CREATE_LOG}" && ((attempt < CLUSTER_CREATE_MAX_ATTEMPTS)); then
			echo "cluster create hit bridge bring-up race; destroying partial state and retrying"
			"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy --provisioner qemu >/dev/null 2>&1 || true
			rm -rf "${STATE_DIR:?}/${CLUSTER_NAME}"
			attempt=$((attempt + 1))
			sleep 2
			continue
		fi

		echo "cluster create failed; see ${CLUSTER_CREATE_LOG}" >&2
		return 1
	done
}

wait_for_maintenance() {
	wait_until "maintenance API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" get addresses --insecure -e "${NODE_IP}" -n "${NODE_IP}"
}

wait_for_runtime() {
	wait_until "runtime mTLS API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
}

wait_for_runtime_stable() {
	local consecutive=0
	local required_consecutive=5
	local deadline=$((SECONDS + TIMEOUT_SECONDS))

	while true; do
		if run_probe "${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1 &&
			run_probe "${TALOSCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" service machined >/dev/null 2>&1; then
			consecutive=$((consecutive + 1))

			if ((consecutive >= required_consecutive)); then
				return 0
			fi
		else
			consecutive=0
		fi

		if ((SECONDS >= deadline)); then
			echo "timed out waiting for stable runtime API on ${NODE_IP}" >&2

			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
}

check_binary_mode_artifact() {
	local resource="$1"
	local output

	if ! output="$("${TALOSCTL}" get "${resource}" --namespace chubo -o yaml --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>&1)"; then
		echo "failed to query ${resource} status" >&2
		echo "${output}" >&2
		return 1
	fi

	if grep -q 'configured: false' <<<"${output}"; then
		echo "${resource}: configured=false (skipping binaryMode assertion in this flow)"
		return 0
	fi

	chubo_status_configured=1

	if ! grep -q 'binaryMode: artifact' <<<"${output}"; then
		echo "expected ${resource} binaryMode=artifact" >&2
		echo "${output}" >&2
		return 1
	fi

	echo "${resource}: binaryMode=artifact"
}

download_helper_bundles() {
	local bundle

	echo "downloading helper bundles"
	rm -rf "${HELPERS_DIR}"
	mkdir -p "${HELPERS_DIR}"

	"${TALOSCTL}" nomadconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
	"${TALOSCTL}" consulconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
	"${TALOSCTL}" openbaoconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"

	for bundle in nomadconfig consulconfig openbaoconfig; do
		local dir="${HELPERS_DIR}/${bundle}"
		test -d "${dir}"

		case "${bundle}" in
		nomadconfig)
			test -f "${dir}/nomad.env"
			test -f "${dir}/nomad.hcl"
			;;
		consulconfig)
			test -f "${dir}/consul.env"
			test -f "${dir}/consul.hcl"
			;;
		openbaoconfig)
			test -f "${dir}/openbao.env"
			test -f "${dir}/openbao.hcl"
			;;
		esac

		test -f "${dir}/ca.pem"
		test -f "${dir}/client.pem"
		test -f "${dir}/client-key.pem"
		test -f "${dir}/acl.token"
		test -f "${dir}/README"
	done

	{
		echo "helpers_dir=${HELPERS_DIR}"
		echo "nomad_addr=$(awk -F= '/^NOMAD_ADDR=/{print $2}' "${HELPERS_DIR}/nomadconfig/nomad.env")"
		echo "consul_addr=$(awk -F= '/^CONSUL_HTTP_ADDR=/{print $2}' "${HELPERS_DIR}/consulconfig/consul.env")"
		echo "openbao_addr=$(awk -F= '/^VAULT_ADDR=/{print $2}' "${HELPERS_DIR}/openbaoconfig/openbao.env")"
		echo "nomad_token_len=$(tr -d '\n' < "${HELPERS_DIR}/nomadconfig/acl.token" | wc -c | tr -d ' ')"
		echo "consul_token_len=$(tr -d '\n' < "${HELPERS_DIR}/consulconfig/acl.token" | wc -c | tr -d ' ')"
		echo "openbao_token_len=$(tr -d '\n' < "${HELPERS_DIR}/openbaoconfig/acl.token" | wc -c | tr -d ' ')"
	} >"${HELPERS_VALIDATION_OUT}"

	cat "${HELPERS_VALIDATION_OUT}"
}

read_boot_id() {
	local boot_id

	if ! boot_id="$("${TALOSCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" read /proc/sys/kernel/random/boot_id 2>/dev/null)"; then
		return 1
	fi

	boot_id="$(printf '%s' "${boot_id}" | tr -d '\r\n[:space:]')"
	if [[ -z "${boot_id}" ]]; then
		return 1
	fi

	printf '%s\n' "${boot_id}"
}

wait_for_boot_id_change() {
	local previous_boot_id="$1"
	local context="$2"
	local deadline=$((SECONDS + ACTION_REBOOT_WAIT_SECONDS))

	while true; do
		local current_boot_id

		if current_boot_id="$(read_boot_id)"; then
			if [[ "${current_boot_id}" != "${previous_boot_id}" ]]; then
				echo "${context}: observed boot ID change (${previous_boot_id} -> ${current_boot_id})"
				return 0
			fi
		fi

		if ((SECONDS >= deadline)); then
			echo "${context}: no boot ID change observed within ${ACTION_REBOOT_WAIT_SECONDS}s"
			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
}

cleanup() {
	set +e

	if ((cluster_created == 1)); then
		"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy \
			--provisioner qemu \
			--save-cluster-logs-archive-path "${CLUSTER_LOGS_OUT}" \
			--save-support-archive-path "${CLUSTER_SUPPORT_OUT}" >/dev/null 2>&1
	fi

	if ((registry_started == 1)); then
		docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1
	fi
}

trap cleanup EXIT

if [[ "${ARCH}" != "amd64" ]]; then
	echo "unsupported ARCH=${ARCH} (only amd64 is supported by this script)" >&2

	exit 2
fi

require_cmd docker
require_cmd go
require_cmd make
require_cmd unzip

ctl_target="chuboctl-${HOST_GOOS}-${HOST_GOARCH}"
if [[ "${TALOSCTL##*/}" == talosctl-* ]]; then
	ctl_target="talosctl-${HOST_GOOS}-${HOST_GOARCH}"
fi

if [[ ! -x "${TALOSCTL}" ]]; then
	make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
elif ! "${TALOSCTL}" support --help 2>/dev/null | grep -q "Chubo module config snapshots"; then
	echo "existing CLI binary is not chubo-tagged; rebuilding"
	make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
elif command -v strings >/dev/null 2>&1 && strings "${TALOSCTL}" 2>/dev/null | grep -q "KUBERNETES ENDPOINT"; then
	# We recently renamed this UX to "CONTROL PLANE ENDPOINT". Seeing the old string
	# means the host binary is stale (even if it was built with the right tags).
	echo "existing CLI binary is stale (contains 'KUBERNETES ENDPOINT'); rebuilding"
	make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
fi

if ! command -v crane >/dev/null 2>&1; then
	go install github.com/google/go-containerregistry/cmd/crane@latest
fi

CRANE_BIN="${CRANE_BIN:-$(command -v crane || true)}"
if [[ -z "${CRANE_BIN}" ]]; then
	CRANE_BIN="$(go env GOPATH)/bin/crane"
fi

if [[ ! -x "${CRANE_BIN}" ]]; then
	echo "crane binary not found after installation attempt" >&2

	exit 1
fi

rm -rf "${WORKDIR}" "${STATE_DIR}"
mkdir -p "${WORKDIR}" "${ARTIFACTS}"

if [[ "${SKIP_BUILD}" != "1" ]]; then
	ensure_buildx_builder

	echo "building chubo boot artifacts"
	make initramfs kernel sd-boot \
		ARTIFACTS="${ARTIFACTS}" \
		GO_BUILDTAGS="${GO_BUILDTAGS}" \
		TARGET_ARGS="--builder=${BUILDX_BUILDER} ${TARGET_ARGS:-}" \
		PLATFORM="linux/${ARCH}"

	echo "building chubo installer-base and imager docker images"
	make docker-installer-base docker-imager \
		DEST="${ARTIFACTS}" \
		GO_BUILDTAGS="${GO_BUILDTAGS}" \
		TARGET_ARGS="--builder=${BUILDX_BUILDER} ${TARGET_ARGS:-}" \
		PLATFORM="linux/${ARCH}" \
		INSTALLER_ARCH=targetarch \
		IMAGE_REGISTRY=localhost \
		USERNAME=chubo \
		IMAGE_TAG_OUT=dev
else
	echo "SKIP_BUILD=1: reusing existing artifacts in ${ARTIFACTS}"
fi

docker load -i "${ARTIFACTS}/installer-base.tar" >/dev/null
docker load -i "${ARTIFACTS}/imager.tar" >/dev/null

start_registry

echo "pushing installer-base image to local registry (${INSTALLER_BASE_IMAGE_LOCAL})"
"${CRANE_BIN}" --insecure push "${ARTIFACTS}/installer-base.tar" "${INSTALLER_BASE_IMAGE_LOCAL}" >/dev/null

echo "building installer image tar via imager"
SOURCE_DATE_EPOCH="$(git log -1 --pretty=%ct)"
docker run --rm -t \
	--network=host \
	-v "${PWD}/${ARTIFACTS}:/secureboot:ro" \
	-v "${PWD}/${ARTIFACTS}:/out" \
	-e SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH}" \
	-e DETERMINISTIC_SEED=1 \
	localhost/chubo/imager:dev installer \
	--arch "${ARCH}" \
	--base-installer-image "${INSTALLER_BASE_IMAGE_LOCAL}"

echo "pushing installer image to local registry (${INSTALLER_IMAGE_LOCAL})"
installer_arch_ref="$("${CRANE_BIN}" --insecure push "${ARTIFACTS}/installer-${ARCH}.tar" "${INSTALLER_IMAGE_LOCAL}-${ARCH}")"
"${CRANE_BIN}" --insecure index append -t "${INSTALLER_IMAGE_LOCAL}" -m "${installer_arch_ref}"

echo "generating secrets and machine configs"
"${TALOSCTL}" gen secrets -o "${SECRETS_FILE}"
"${TALOSCTL}" gen machineconfig \
	--with-secrets "${SECRETS_FILE}" \
	--install-disk "${INSTALL_DISK}" \
	--install-image "${INSTALLER_IMAGE_NODE}" \
	--registry-mirror "${REGISTRY_MIRROR_NODE}" \
	-o "${MACHINECONFIG_INSTALL}"
"${TALOSCTL}" gen config chubo https://0.0.0.0:6443 \
	--with-secrets "${SECRETS_FILE}" \
	-t talosconfig \
	-o "${TALOSCONFIG_FILE}"

cp "${MACHINECONFIG_INSTALL}" "${MACHINECONFIG_RUNTIME}"
# Use a temp file rewrite so this works on both GNU and BSD/macOS sed.
runtime_tmp="${MACHINECONFIG_RUNTIME}.tmp"
sed \
	-e 's/^\([[:space:]]*wipe:[[:space:]]*\)true$/\1false/' \
	-e 's|^\([[:space:]]*image:[[:space:]]*\).*$|\1""|' \
	"${MACHINECONFIG_RUNTIME}" >"${runtime_tmp}"
mv "${runtime_tmp}" "${MACHINECONFIG_RUNTIME}"

create_cluster_with_retry

echo "waiting for maintenance API"
wait_for_maintenance

echo "applying install config"
"${TALOSCTL}" apply-config --insecure -m reboot -e "${NODE_IP}" -n "${NODE_IP}" -f "${MACHINECONFIG_INSTALL}"

echo "waiting for post-install transition"
transition_deadline=$((SECONDS + TIMEOUT_SECONDS))
saw_maintenance_down=0
maintenance_reentered_at=0
maintenance_up_since=0

while true; do
	if run_probe "${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1; then
		echo "runtime mTLS became available after install apply"
		break
	fi

	if run_probe "${TALOSCTL}" get addresses --insecure -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1; then
		if ((maintenance_up_since == 0)); then
			maintenance_up_since="${SECONDS}"
		fi

		if ((saw_maintenance_down == 1)); then
			if ((maintenance_reentered_at == 0)); then
				maintenance_reentered_at="${SECONDS}"
			fi

			if ((SECONDS - maintenance_reentered_at >= MAINTENANCE_PERSIST_SECONDS)); then
				echo "maintenance API stayed up after reboot; applying runtime config and rebooting"
				"${TALOSCTL}" apply-config --insecure -m reboot -e "${NODE_IP}" -n "${NODE_IP}" -f "${MACHINECONFIG_RUNTIME}"
				runtime_config_applied=1
				break
			fi
		elif ((runtime_config_applied == 0)) && ((SECONDS - maintenance_up_since >= MAINTENANCE_FALLBACK_SECONDS)); then
			echo "maintenance API stayed up for ${MAINTENANCE_FALLBACK_SECONDS}s after install apply; forcing runtime config and reboot"
			"${TALOSCTL}" apply-config --insecure -m reboot -e "${NODE_IP}" -n "${NODE_IP}" -f "${MACHINECONFIG_RUNTIME}"
			runtime_config_applied=1
			break
		fi
	else
		saw_maintenance_down=1
		maintenance_reentered_at=0
		maintenance_up_since=0
	fi

	if ((SECONDS >= transition_deadline)); then
		echo "timed out waiting for post-install transition" >&2
		exit 1
	fi

	sleep "${SLEEP_SECONDS}"
done

if ! wait_for_runtime; then
	if ((runtime_config_applied == 0)); then
		echo "runtime mTLS did not come up after install, applying runtime config and rebooting"
		wait_for_maintenance
		"${TALOSCTL}" apply-config --insecure -m reboot -e "${NODE_IP}" -n "${NODE_IP}" -f "${MACHINECONFIG_RUNTIME}"
	else
		echo "runtime mTLS did not come up after runtime config apply, retrying runtime wait"
	fi
	wait_for_runtime
fi

echo "validating runtime mTLS and runtime surface"
"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
./hack/chubo/check-runtime-surface.sh \
	--talosconfig "${TALOSCONFIG_FILE}" \
	--endpoint "${NODE_IP}" \
	--node "${NODE_IP}"
check_binary_mode_artifact "openwontonstatus"
check_binary_mode_artifact "opengyozastatus"

if [[ "${WITH_HELPERS}" == "1" ]]; then
	download_helper_bundles
fi

echo "running upgrade flow"
pre_upgrade_boot_id="$(read_boot_id || true)"
"${TALOSCTL}" upgrade \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-i "${INSTALLER_IMAGE_NODE}" \
	--wait=false

if [[ -n "${pre_upgrade_boot_id}" ]]; then
	wait_for_boot_id_change "${pre_upgrade_boot_id}" "upgrade" || true
else
	echo "upgrade: unable to read pre-upgrade boot ID, skipping reboot detection"
fi

wait_for_runtime_stable

echo "running rollback flow"
pre_rollback_boot_id="$(read_boot_id || true)"
if rollback_output="$("${TALOSCTL}" rollback \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" 2>&1)"; then
	if [[ -n "${pre_rollback_boot_id}" ]]; then
		wait_for_boot_id_change "${pre_rollback_boot_id}" "rollback" || true
	else
		echo "rollback: unable to read pre-rollback boot ID, skipping reboot detection"
	fi
elif grep -q "previous UKI not found" <<<"${rollback_output}"; then
	echo "rollback skipped: previous UKI not found for this node/image state"
else
	echo "${rollback_output}" >&2
	exit 1
fi

wait_for_runtime_stable

echo "collecting support bundle"
rm -f "${SUPPORT_OUT}" "${SUPPORT_LISTING}"
"${TALOSCTL}" support \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-O "${SUPPORT_OUT}" -v
unzip -l "${SUPPORT_OUT}" > "${SUPPORT_LISTING}"

grep -q 'summary' "${SUPPORT_LISTING}"
grep -q 'dmesg.log' "${SUPPORT_LISTING}"
grep -q 'dependencies.dot' "${SUPPORT_LISTING}"
grep -q 'mounts' "${SUPPORT_LISTING}"
grep -q 'io' "${SUPPORT_LISTING}"
grep -q 'processes' "${SUPPORT_LISTING}"
grep -q 'service-logs/.*\.state' "${SUPPORT_LISTING}"

if ((chubo_status_configured == 1)); then
	grep -q 'chubo-config/' "${SUPPORT_LISTING}"
else
	echo "support bundle: no configured chubo services in this run; skipping chubo-config snapshot assertion"
fi

echo "chubo core E2E flow completed"
echo "support bundle: ${SUPPORT_OUT}"
echo "cluster logs:   ${CLUSTER_LOGS_OUT}"
echo "cluster support:${CLUSTER_SUPPORT_OUT}"
