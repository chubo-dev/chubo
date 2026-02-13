#!/usr/bin/env bash
set -euo pipefail

# End-to-end smoke for Chubo workload helper bundles in QEMU/vmnet:
# install -> runtime mTLS -> nomadconfig/consulconfig/openbaoconfig extraction.
#
# This is optimized for fast local validation on macOS arm64.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALOS_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${TALOS_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chubo}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"

TALOSCTL_BASE="${TALOSCTL_BASE:-${TALOS_ROOT}/_out/talosctl-${HOST_GOOS}-${HOST_GOARCH}}"
TALOSCTL_CHUBO="${TALOSCTL_CHUBO:-${TALOS_ROOT}/_out/chubo/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-helper-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"
REGISTRY_LOCAL_ADDR="${REGISTRY_LOCAL_ADDR:-localhost:${REGISTRY_PORT}}"
REGISTRY_BUILD_ADDR="${REGISTRY_BUILD_ADDR:-host.docker.internal:${REGISTRY_PORT}}"
USERNAME="${USERNAME:-chubo}"

INSTALLER_TAG="${INSTALLER_TAG:-helper$(date +%s)}"
INSTALLER_IMAGE_NODE="${INSTALLER_IMAGE_NODE:-10.0.2.2:${REGISTRY_PORT}/${USERNAME}/installer:${INSTALLER_TAG}-arm64}"
REGISTRY_MIRROR_NODE="${REGISTRY_MIRROR_NODE:-10.0.2.2:${REGISTRY_PORT}=http://10.0.2.2:${REGISTRY_PORT}}"
SKIP_BUILD="${SKIP_BUILD:-0}"

HOST_PORT="${HOST_PORT:-50000}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-900}"
SLEEP_SECONDS="${SLEEP_SECONDS:-2}"
RUN_DIR="${RUN_DIR:-$(mktemp -d /tmp/chubo-helper-e2e.XXXXXX)}"
KEEP_VM="${KEEP_VM:-0}"
PRUNE_KEEP="${PRUNE_KEEP:-3}"

SECRETS_FILE="${RUN_DIR}/secrets.yaml"
MACHINECONFIG_INSTALL="${RUN_DIR}/machineconfig-install.yaml"
MACHINECONFIG_RUNTIME="${RUN_DIR}/machineconfig-runtime.yaml"
TALOSCONFIG_FILE="${RUN_DIR}/talosconfig"
HELPERS_DIR="${RUN_DIR}/helpers"
LOG_INSTALL="${RUN_DIR}/qemu-install.log"
LOG_DISK="${RUN_DIR}/qemu-disk.log"
VALIDATION_OUT="${RUN_DIR}/helper-validation.txt"

QEMU_INSTALL_PID=0
QEMU_DISK_PID=0
NODE_IP=""

require_cmd() {
	local cmd="$1"

	if ! command -v "${cmd}" >/dev/null 2>&1; then
		echo "required command not found: ${cmd}" >&2

		exit 1
	fi
}

prune_old_run_dirs() {
	# These QEMU runs can allocate multi-GB qcow2 images. Keep only the most recent
	# runs to avoid filling the host disk during tight iteration loops.
	local keep="${PRUNE_KEEP}"
	local current=""

	if current="$(cd "${RUN_DIR}" 2>/dev/null && pwd -P)"; then
		:
	else
		current="${RUN_DIR}"
	fi

	# Sort newest -> oldest. /tmp is /private/tmp on macOS, use the canonical path.
	mapfile -t dirs < <(ls -1td /private/tmp/chubo-helper-e2e.* 2>/dev/null || true)

	local kept=0
	for dir in "${dirs[@]}"; do
		# Never delete the active run dir.
		if [[ "${dir}" == "${current}" || "${dir}" == "${RUN_DIR}" ]]; then
			continue
		fi

		if ((kept < keep)); then
			kept=$((kept + 1))
			continue
		fi

		rm -rf -- "${dir}" >/dev/null 2>&1 || true
	done
}

ensure_registry() {
	if curl -fsS "http://localhost:${REGISTRY_PORT}/v2/" >/dev/null 2>&1; then
		echo "reusing existing registry on :${REGISTRY_PORT}"

		return 0
	fi

	docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1 || true

	if docker run -d --restart=always --name "${REGISTRY_NAME}" -p "${REGISTRY_PORT}:5000" registry:2 >/dev/null; then
		return 0
	fi

	if curl -fsS "http://localhost:${REGISTRY_PORT}/v2/" >/dev/null 2>&1; then
		echo "registry port :${REGISTRY_PORT} is already in use, reusing running registry"

		return 0
	fi

	echo "failed to start or reuse local registry on :${REGISTRY_PORT}" >&2

	return 1
}

wait_until() {
	local description="$1"
	local timeout_seconds="$2"
	shift 2

	local deadline=$((SECONDS + timeout_seconds))

	while true; do
		if "$@" >/dev/null 2>&1; then
			return 0
		fi

		if ((SECONDS >= deadline)); then
			echo "timed out waiting for: ${description}" >&2

			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
}

parse_bridged_ip() {
	local addresses_file="$1"

	awk '
		{
			for (i = 1; i <= NF; i++) {
				# talosctl prints addresses as either "<ip>/<cidr>" or "<iface>/<ip>/<cidr>".
				if ($i ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\/[0-9]+$/) {
					ip = $i
					sub(/\/.*/, "", ip)
					if (ip !~ /^10\.0\.2\./ && ip != "127.0.0.1") {
						print ip
						exit
					}
				} else if ($i ~ /^[^\/]+\/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\/[0-9]+$/) {
					n = split($i, parts, "/")
					if (n == 3) {
						ip = parts[2]
					} else {
						next
					}
					if (ip !~ /^10\.0\.2\./ && ip != "127.0.0.1") {
						print ip
						exit
					}
				}
			}
		}
	' "${addresses_file}"
}

openwonton_ready() {
	local output

	if ! output="$("${TALOSCTL_CHUBO}" get openwontonstatus --namespace chubo -o json --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>/dev/null)"; then
		return 1
	fi

	jq -e '.spec.configured == true and .spec.healthy == true and .spec.aclReady == true and .spec.binaryMode == "artifact"' <<<"${output}" >/dev/null 2>&1 || return 1
}

opengyoza_ready() {
	local output

	if ! output="$("${TALOSCTL_CHUBO}" get opengyozastatus --namespace chubo -o json --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>/dev/null)"; then
		return 1
	fi

	jq -e '.spec.configured == true and .spec.healthy == true and .spec.aclReady == true and .spec.binaryMode == "artifact"' <<<"${output}" >/dev/null 2>&1 || return 1
}

openbao_job_ready() {
	local output

	if ! output="$("${TALOSCTL_CHUBO}" get openbaojobstatus --namespace chubo -o json --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>/dev/null)"; then
		return 1
	fi

	jq -e '.spec.configured == true and .spec.nomadReachable == true and .spec.present == true and (.spec.lastError == "" or .spec.lastError == null)' <<<"${output}" >/dev/null 2>&1 || return 1
}

dump_chubo_debug() {
	set +e

	echo
	echo "==== chubo debug ===="
	echo "run_dir=${RUN_DIR}"
	echo "node_ip=${NODE_IP}"
	echo

	echo "-- openwontonstatus (json)"
	"${TALOSCTL_CHUBO}" get openwontonstatus --namespace chubo -o json --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- openwontonstatus (yaml)"
	"${TALOSCTL_CHUBO}" get openwontonstatus --namespace chubo -o yaml --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- opengyozastatus (json)"
	"${TALOSCTL_CHUBO}" get opengyozastatus --namespace chubo -o json --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- opengyozastatus (yaml)"
	"${TALOSCTL_CHUBO}" get opengyozastatus --namespace chubo -o yaml --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- v1alpha1 service openwonton (yaml)"
	"${TALOSCTL_CHUBO}" get svc --namespace v1alpha1 openwonton -o yaml --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- v1alpha1 service opengyoza (yaml)"
	"${TALOSCTL_CHUBO}" get svc --namespace v1alpha1 opengyoza -o yaml --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- qemu logs (tail)"
	echo "install_log=${LOG_INSTALL}"
	tail -n 120 "${LOG_INSTALL}" 2>/dev/null || true
	echo
	echo "disk_log=${LOG_DISK}"
	tail -n 160 "${LOG_DISK}" 2>/dev/null || true
	echo "==== end chubo debug ===="
	echo
}

cleanup() {
	set +e

	if [[ "${QEMU_INSTALL_PID}" -gt 0 ]]; then
		kill "${QEMU_INSTALL_PID}" >/dev/null 2>&1 || true
	fi

	if [[ "${QEMU_DISK_PID}" -gt 0 ]] && [[ "${KEEP_VM}" -eq 0 ]]; then
		kill "${QEMU_DISK_PID}" >/dev/null 2>&1 || true
	fi

	if [[ "${KEEP_VM}" -eq 0 ]]; then
		pkill -f "qemu-system-aarch64.*${RUN_DIR}" >/dev/null 2>&1 || true
	fi
}

trap cleanup EXIT

prune_old_run_dirs

require_cmd docker
require_cmd go
require_cmd make
require_cmd openssl
require_cmd rg
require_cmd curl
require_cmd jq

if [[ ! -x "${TALOSCTL_BASE}" ]]; then
	make talosctl
fi

rebuild_chuboctl=0
if [[ ! -x "${TALOSCTL_CHUBO}" ]]; then
	rebuild_chuboctl=1
else
	# The helper-bundles E2E depends on newer `gen machineconfig` flags.
	# If the cached binary doesn't have them, rebuild it from source.
	if ! "${TALOSCTL_CHUBO}" gen machineconfig --help 2>&1 | rg -q -- '--with-chubo'; then
		rebuild_chuboctl=1
	fi
fi

if [[ "${rebuild_chuboctl}" -eq 1 ]]; then
	mkdir -p "$(dirname "${TALOSCTL_CHUBO}")"

	GOOS="${HOST_GOOS}" GOARCH="${HOST_GOARCH}" CGO_ENABLED=0 go build \
		-tags grpcnotrace,chubo \
		-o "${TALOSCTL_CHUBO}" \
		./cmd/chuboctl
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

echo "using run dir: ${RUN_DIR}"
echo "installer tag: ${INSTALLER_TAG}"

echo "starting local registry on :${REGISTRY_PORT}"
ensure_registry

if [[ "${SKIP_BUILD}" -eq 0 ]]; then
	IMAGES_DIR="${RUN_DIR}/images"
	mkdir -p "${IMAGES_DIR}"

	echo "building installer-base + imager tarballs (${INSTALLER_TAG})"
	GO_BUILDTAGS="${GO_BUILDTAGS}" make docker-installer-base docker-imager \
		DEST="${IMAGES_DIR}" \
		IMAGE_REGISTRY=localhost \
		USERNAME="${USERNAME}" \
		IMAGE_TAG_OUT="${INSTALLER_TAG}" \
		PLATFORM=linux/arm64 \
		INSTALLER_ARCH=targetarch

	echo "loading imager image into local docker (for installer build)"
	docker load -i "${IMAGES_DIR}/imager.tar" >/dev/null

	echo "pushing installer-base image to local registry (${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer-base:${INSTALLER_TAG})"
	"${CRANE_BIN}" --insecure push "${IMAGES_DIR}/installer-base.tar" "${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer-base:${INSTALLER_TAG}" >/dev/null

	echo "building installer tarball via imager"
	SOURCE_DATE_EPOCH="$(git log -1 --pretty=%ct)"
	docker run --rm -t \
		--network=host \
		--user "$(id -u):$(id -g)" \
		-v "${PWD}/${ARTIFACTS}:/secureboot:ro" \
		-v "${PWD}/${ARTIFACTS}:/out" \
		-e SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH}" \
		-e DETERMINISTIC_SEED=1 \
		localhost/${USERNAME}/imager:"${INSTALLER_TAG}" installer \
		--arch arm64 \
		--insecure \
		--base-installer-image "${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer-base:${INSTALLER_TAG}"

	echo "pushing installer tarball (${INSTALLER_TAG}-arm64)"
	installer_arch_ref="$("${CRANE_BIN}" --insecure push "${ARTIFACTS}/installer-arm64.tar" "${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer:${INSTALLER_TAG}-arm64")"
	"${CRANE_BIN}" --insecure index append -t "${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer:${INSTALLER_TAG}" -m "${installer_arch_ref}" >/dev/null
	rm -f "${ARTIFACTS}/installer-arm64.tar"
else
	echo "SKIP_BUILD=1, reusing existing installer image tag: ${INSTALLER_TAG}"

	if ! "${CRANE_BIN}" --insecure manifest "${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer:${INSTALLER_TAG}-arm64" >/dev/null 2>&1; then
		echo "installer image ${REGISTRY_LOCAL_ADDR}/${USERNAME}/installer:${INSTALLER_TAG}-arm64 not found" >&2

		exit 1
	fi
fi

echo "generating secrets + machine config"
"${TALOSCTL_BASE}" gen secrets -o "${SECRETS_FILE}"
"${TALOSCTL_CHUBO}" gen machineconfig \
	--with-secrets "${SECRETS_FILE}" \
	--install-disk /dev/vdb \
	--install-image "${INSTALLER_IMAGE_NODE}" \
	--registry-mirror "${REGISTRY_MIRROR_NODE}" \
	--with-chubo \
	--chubo-role server \
	--with-openbao \
	--openbao-mode nomadJob \
	-o "${MACHINECONFIG_INSTALL}"

cp "${MACHINECONFIG_INSTALL}" "${MACHINECONFIG_RUNTIME}"
runtime_tmp="${MACHINECONFIG_RUNTIME}.tmp"
sed \
	-e 's|^\([[:space:]]*disk:[[:space:]]*\).*$|\1"/dev/vda"|' \
	-e 's/^\([[:space:]]*wipe:[[:space:]]*\)true$/\1false/' \
	-e 's|^\([[:space:]]*image:[[:space:]]*\).*$|\1""|' \
	"${MACHINECONFIG_RUNTIME}" >"${runtime_tmp}"
mv "${runtime_tmp}" "${MACHINECONFIG_RUNTIME}"

"${TALOSCTL_BASE}" gen config chubo https://0.0.0.0:6443 \
	--with-secrets "${SECRETS_FILE}" \
	-t talosconfig \
	-o "${TALOSCONFIG_FILE}"

VMNET_MAC="52:54:00:$(openssl rand -hex 3 | sed -E 's/(..)(..)(..)/\1:\2:\3/')"
echo "booting install media (vmnet mac: ${VMNET_MAC})"
VMNET_ENABLE=1 VMNET_MAC="${VMNET_MAC}" QEMU_RUNDIR="${RUN_DIR}" BOOT_FROM_DISK=0 HOST_PORT="${HOST_PORT}" \
	./hack/qemu/chubo-qemu.sh >"${LOG_INSTALL}" 2>&1 &
QEMU_INSTALL_PID=$!

wait_until "maintenance API (install media)" "${TIMEOUT_SECONDS}" \
	"${TALOSCTL_CHUBO}" get addresses --insecure -e 127.0.0.1 -n 127.0.0.1

"${TALOSCTL_CHUBO}" get addresses --insecure -e 127.0.0.1 -n 127.0.0.1 >"${RUN_DIR}/addresses-install.txt"
NODE_IP="$(parse_bridged_ip "${RUN_DIR}/addresses-install.txt")"
if [[ -z "${NODE_IP}" ]]; then
	echo "failed to discover bridged node IP from install phase" >&2

	exit 1
fi

echo "applying install config"
"${TALOSCTL_CHUBO}" apply-config -i -e 127.0.0.1 -n 127.0.0.1 -f "${MACHINECONFIG_INSTALL}" || true

wait_until "installer completion marker" "${TIMEOUT_SECONDS}" \
	rg -q "installation of .* complete" "${LOG_INSTALL}"

kill "${QEMU_INSTALL_PID}" >/dev/null 2>&1 || true
QEMU_INSTALL_PID=0
sleep 1
pkill -f "qemu-system-aarch64.*${RUN_DIR}" >/dev/null 2>&1 || true
sleep 1

echo "booting installed disk"
cp -f /opt/homebrew/share/qemu/edk2-arm-vars.fd "${RUN_DIR}/edk2-vars.fd"
VMNET_ENABLE=1 VMNET_MAC="${VMNET_MAC}" QEMU_RUNDIR="${RUN_DIR}" BOOT_FROM_DISK=1 HOST_PORT="${HOST_PORT}" \
	./hack/qemu/chubo-qemu.sh >"${LOG_DISK}" 2>&1 &
QEMU_DISK_PID=$!

wait_until "maintenance API (installed disk)" "${TIMEOUT_SECONDS}" \
	"${TALOSCTL_CHUBO}" get addresses --insecure -e 127.0.0.1 -n 127.0.0.1

echo "applying runtime config and rebooting into runtime API"
"${TALOSCTL_CHUBO}" apply-config -i -m reboot -e 127.0.0.1 -n 127.0.0.1 -f "${MACHINECONFIG_RUNTIME}" || true

wait_until "runtime mTLS API (${NODE_IP})" "${TIMEOUT_SECONDS}" \
	"${TALOSCTL_CHUBO}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"

echo "waiting for openwonton/opengyoza health + ACL bootstrap"
if ! wait_until "openwontonstatus (healthy + aclReady)" "${TIMEOUT_SECONDS}" openwonton_ready; then
	dump_chubo_debug
	exit 1
fi

if ! wait_until "opengyozastatus (healthy + aclReady)" "${TIMEOUT_SECONDS}" opengyoza_ready; then
	dump_chubo_debug
	exit 1
fi

echo "waiting for openbao Nomad job controller"
wait_until "openbaojobstatus (present)" "${TIMEOUT_SECONDS}" openbao_job_ready

echo "downloading helper bundles"
mkdir -p "${HELPERS_DIR}"
"${TALOSCTL_CHUBO}" nomadconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
"${TALOSCTL_CHUBO}" consulconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
"${TALOSCTL_CHUBO}" openbaoconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"

for bundle in nomadconfig consulconfig openbaoconfig; do
	dir="${HELPERS_DIR}/${bundle}"
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
	echo "run=${RUN_DIR}"
	echo "node_ip=${NODE_IP}"
	echo "installer_tag=${INSTALLER_TAG}"
	"${TALOSCTL_CHUBO}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" \
		| awk 'BEGIN{s=0} /^Server:/{s=1; next} s && /^\tTag:/{print "server_tag=" $2} s && /^\tSHA:/{print "server_sha=" $2}'
	echo "nomad_addr=$(awk -F= '/^NOMAD_ADDR=/{print $2}' "${HELPERS_DIR}/nomadconfig/nomad.env")"
	echo "consul_addr=$(awk -F= '/^CONSUL_HTTP_ADDR=/{print $2}' "${HELPERS_DIR}/consulconfig/consul.env")"
	echo "openbao_addr=$(awk -F= '/^VAULT_ADDR=/{print $2}' "${HELPERS_DIR}/openbaoconfig/openbao.env")"
	echo "nomad_token_len=$(tr -d '\n' < "${HELPERS_DIR}/nomadconfig/acl.token" | wc -c | tr -d ' ')"
	echo "consul_token_len=$(tr -d '\n' < "${HELPERS_DIR}/consulconfig/acl.token" | wc -c | tr -d ' ')"
	echo "openbao_token=$(tr -d '\n' < "${HELPERS_DIR}/openbaoconfig/acl.token")"
} >"${VALIDATION_OUT}"

cat "${VALIDATION_OUT}"
echo
echo "helper bundle smoke complete"
echo "artifacts:"
echo "  ${VALIDATION_OUT}"
echo "  ${HELPERS_DIR}"
echo "  ${LOG_INSTALL}"
echo "  ${LOG_DISK}"
