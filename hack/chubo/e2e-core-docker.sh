#!/usr/bin/env bash
set -euo pipefail

# End-to-end chubo core flow in Docker provisioner (non-root fallback):
# runtime mTLS -> runtime surface validation -> support bundle.
#
# This target is intended for fast local iteration where QEMU/HVF root access
# is not available. Keep `e2e-core-qemu.sh` as the strict install/upgrade path.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALOS_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${TALOS_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chubo}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
ARCH="${ARCH:-amd64}"

HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"
TALOSCTL="${TALOSCTL:-${TALOS_ROOT}/_out/talosctl-${HOST_GOOS}-${HOST_GOARCH}}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-e2e-docker}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-e2e-docker-state}"
WORKDIR="${WORKDIR:-/tmp/chubo-e2e-docker-work}"
SUBNET="${SUBNET:-10.5.0.0/24}"
NODE_CONTAINER="${NODE_CONTAINER:-${CLUSTER_NAME}-controlplane-1}"

TALOS_IMAGE_LOCAL="${TALOS_IMAGE_LOCAL:-localhost/chubo/talos:dev}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-600}"
SLEEP_SECONDS="${SLEEP_SECONDS:-3}"

SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chubo-support-e2e-docker.zip}"
SUPPORT_LISTING="${WORKDIR}/support-listing.txt"
TALOSCONFIG_FILE="${WORKDIR}/talosconfig"

cluster_created=0
NODE_IP=""

require_cmd() {
	local cmd="$1"

	if ! command -v "${cmd}" >/dev/null 2>&1; then
		echo "required command not found: ${cmd}" >&2

		exit 1
	fi
}

check_host_support() {
	local host_os

	host_os="$(uname -s | tr '[:upper:]' '[:lower:]')"

	if [[ "${host_os}" != "linux" ]] && [[ "${ALLOW_UNSUPPORTED_DOCKER:-0}" != "1" ]]; then
		echo "docker provisioner fallback is only supported on Linux hosts." >&2
		echo "host=${host_os} is known to fail with Talos container runtime requirements (seccomp/mount attrs)." >&2
		echo "use 'make chubo-e2e-qemu' for authoritative validation, or set ALLOW_UNSUPPORTED_DOCKER=1 to bypass this guard." >&2

		exit 2
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

wait_for_runtime() {
	wait_until "runtime mTLS API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
}

cleanup() {
	set +e

	if ((cluster_created == 1)); then
		"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy >/dev/null 2>&1
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
check_host_support

if [[ ! -x "${TALOSCTL}" ]]; then
	make "talosctl-${HOST_GOOS}-${HOST_GOARCH}"
fi

mkdir -p "${WORKDIR}" "${ARTIFACTS}" "${STATE_DIR}"
rm -f "${TALOSCONFIG_FILE}" "${SUPPORT_OUT}" "${SUPPORT_LISTING}"
"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy >/dev/null 2>&1 || true

echo "building chubo talos docker image"
make docker-talos \
	DEST="${ARTIFACTS}" \
	GO_BUILDTAGS="${GO_BUILDTAGS}" \
	PLATFORM="linux/${ARCH}" \
	INSTALLER_ARCH=targetarch \
	IMAGE_REGISTRY=localhost \
	USERNAME=chubo \
	IMAGE_TAG_OUT=dev

docker load -i "${ARTIFACTS}/talos.tar" >/dev/null

echo "creating single-node Docker provisioner cluster"
cluster_created=1
if ! "${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster create docker \
	--image "${TALOS_IMAGE_LOCAL}" \
	--workers 0 \
	--subnet "${SUBNET}" \
	--talosconfig-destination "${TALOSCONFIG_FILE}"; then
	echo "cluster create docker failed; recent node logs:" >&2
	if docker ps -a --format '{{.Names}}' | grep -qx "${NODE_CONTAINER}"; then
		docker logs --tail 120 "${NODE_CONTAINER}" >&2 || true
	fi
	echo "hint: docker fallback requires host kernel support for Talos container runtime features." >&2

	exit 1
fi

NODE_IP="$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${NODE_CONTAINER}")"
if [[ -z "${NODE_IP}" ]]; then
	echo "failed to resolve node IP for container ${NODE_CONTAINER}" >&2

	exit 1
fi

echo "waiting for runtime mTLS API on ${NODE_IP}"
if ! wait_for_runtime; then
	echo "runtime mTLS did not come up; recent node logs:" >&2
	docker logs --tail 120 "${NODE_CONTAINER}" >&2 || true

	exit 1
fi

echo "validating runtime mTLS and runtime surface"
"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
./hack/chubo/check-runtime-surface.sh \
	--talosctl "${TALOSCTL}" \
	--talosconfig "${TALOSCONFIG_FILE}" \
	--endpoint "${NODE_IP}" \
	--node "${NODE_IP}"

echo "collecting support bundle"
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

echo "chubo docker fallback E2E flow completed"
echo "support bundle: ${SUPPORT_OUT}"
