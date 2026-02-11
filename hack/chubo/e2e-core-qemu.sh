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
ARCH="${ARCH:-amd64}"
TALOSCTL="${TALOSCTL:-${TALOS_ROOT}/_out/talosctl-linux-amd64}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-e2e}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-e2e-state}"
WORKDIR="${WORKDIR:-/tmp/chubo-e2e-work}"
NODE_IP="${NODE_IP:-10.5.0.2}"
CIDR="${CIDR:-10.5.0.0/24}"
INSTALL_DISK="${INSTALL_DISK:-/dev/vda}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-e2e-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"
REGISTRY_LOCAL_ADDR="${REGISTRY_LOCAL_ADDR:-localhost:${REGISTRY_PORT}}"
REGISTRY_NODE_ADDR="${REGISTRY_NODE_ADDR:-10.5.0.1:${REGISTRY_PORT}}"
INSTALLER_BASE_IMAGE_LOCAL="${INSTALLER_BASE_IMAGE_LOCAL:-${REGISTRY_LOCAL_ADDR}/chubo/installer-base:dev}"
INSTALLER_IMAGE_LOCAL="${INSTALLER_IMAGE_LOCAL:-${REGISTRY_LOCAL_ADDR}/chubo/installer:dev}"
INSTALLER_IMAGE_NODE="${INSTALLER_IMAGE_NODE:-${REGISTRY_NODE_ADDR}/chubo/installer:dev}"
REGISTRY_MIRROR_NODE="${REGISTRY_MIRROR_NODE:-${REGISTRY_NODE_ADDR}=http://${REGISTRY_NODE_ADDR}}"

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1200}"
SLEEP_SECONDS="${SLEEP_SECONDS:-3}"
SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chubo-support-e2e.zip}"
CLUSTER_LOGS_OUT="${CLUSTER_LOGS_OUT:-/tmp/logs-chubo-e2e.tar.gz}"
CLUSTER_SUPPORT_OUT="${CLUSTER_SUPPORT_OUT:-/tmp/support-chubo-e2e.zip}"

SECRETS_FILE="${WORKDIR}/secrets.yaml"
MACHINECONFIG_INSTALL="${WORKDIR}/machineconfig-install.yaml"
MACHINECONFIG_RUNTIME="${WORKDIR}/machineconfig-runtime.yaml"
TALOSCONFIG_FILE="${WORKDIR}/talosconfig"
SUPPORT_LISTING="${WORKDIR}/support-listing.txt"

cluster_created=0
registry_started=0

require_cmd() {
	local cmd="$1"

	if ! command -v "${cmd}" >/dev/null 2>&1; then
		echo "required command not found: ${cmd}" >&2

		exit 1
	fi
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

wait_for_maintenance() {
	wait_until "maintenance API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" get addresses --insecure -e "${NODE_IP}" -n "${NODE_IP}"
}

wait_for_runtime() {
	wait_until "runtime mTLS API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
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

if [[ ! -x "${TALOSCTL}" ]]; then
	make talosctl-linux-amd64
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

echo "building chubo boot artifacts"
make initramfs kernel sd-boot ARTIFACTS="${ARTIFACTS}" GO_BUILDTAGS="${GO_BUILDTAGS}" PLATFORM="linux/${ARCH}"

echo "building chubo installer-base and imager docker images"
make docker-installer-base docker-imager \
	DEST="${ARTIFACTS}" \
	GO_BUILDTAGS="${GO_BUILDTAGS}" \
	PLATFORM="linux/${ARCH}" \
	INSTALLER_ARCH=targetarch \
	IMAGE_REGISTRY=localhost \
	USERNAME=chubo \
	IMAGE_TAG_OUT=dev

docker load -i "${ARTIFACTS}/installer-base.tar" >/dev/null
docker load -i "${ARTIFACTS}/imager.tar" >/dev/null

echo "starting local OCI registry on :${REGISTRY_PORT}"
docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1 || true
docker run -d --rm --name "${REGISTRY_NAME}" -p "${REGISTRY_PORT}:5000" registry:2 >/dev/null
registry_started=1

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

echo "creating single-node QEMU cluster in maintenance mode"
"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster create dev \
	--arch "${ARCH}" \
	--cidr "${CIDR}" \
	--controlplanes 1 \
	--workers 0 \
	--disk 12288 \
	--cpus 2.0 \
	--memory 2.0GiB \
	--skip-injecting-config \
	--skip-kubeconfig \
	--skip-k8s-node-readiness-check \
	--with-cluster-discovery=false \
	--with-init-node=false \
	--kubeprism-port=0 \
	--wait=false \
	--vmlinuz-path "${ARTIFACTS}/vmlinuz-${ARCH}" \
	--initrd-path "${ARTIFACTS}/initramfs-${ARCH}.xz" \
	--install-image "${INSTALLER_IMAGE_NODE}" \
	--registry-mirror "${REGISTRY_MIRROR_NODE}"
cluster_created=1

echo "waiting for maintenance API"
wait_for_maintenance

echo "applying install config"
"${TALOSCTL}" apply-config --insecure -m reboot -e "${NODE_IP}" -n "${NODE_IP}" -f "${MACHINECONFIG_INSTALL}"

echo "waiting for node to leave maintenance mode after install apply"
maintenance_deadline=$((SECONDS + 180))
while "${TALOSCTL}" get addresses --insecure -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1; do
	if ((SECONDS >= maintenance_deadline)); then
		echo "maintenance API is still up after install apply; continuing with runtime wait/fallback path"
		break
	fi

	sleep "${SLEEP_SECONDS}"
done

if ! wait_for_runtime; then
	echo "runtime mTLS did not come up after install, applying runtime config and rebooting"
	wait_for_maintenance
	"${TALOSCTL}" apply-config --insecure -m reboot -e "${NODE_IP}" -n "${NODE_IP}" -f "${MACHINECONFIG_RUNTIME}"
	wait_for_runtime
fi

echo "validating runtime mTLS and runtime surface"
"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
./hack/chubo/check-runtime-surface.sh \
	--talosconfig "${TALOSCONFIG_FILE}" \
	--endpoint "${NODE_IP}" \
	--node "${NODE_IP}"

echo "running upgrade flow"
"${TALOSCTL}" upgrade \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-i "${INSTALLER_IMAGE_NODE}" \
	--wait=false
wait_for_runtime

echo "running rollback flow"
"${TALOSCTL}" rollback \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}"
wait_for_runtime

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

echo "chubo core E2E flow completed"
echo "support bundle: ${SUPPORT_OUT}"
echo "cluster logs:   ${CLUSTER_LOGS_OUT}"
echo "cluster support:${CLUSTER_SUPPORT_OUT}"
