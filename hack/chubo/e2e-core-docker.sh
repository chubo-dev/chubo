#!/usr/bin/env bash
set -euo pipefail

# End-to-end chubo core flow in Docker provisioner (non-root fallback):
# runtime mTLS -> runtime surface validation -> support bundle.
#
# This target is intended for fast local iteration where QEMU/HVF root access
# is not available. Keep `e2e-core-qemu.sh` as the strict install/upgrade path.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHUBO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${CHUBO_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chubo}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
ARCH="${ARCH:-amd64}"

HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"
CHUBOCTL="${CHUBOCTL:-${TALOSCTL:-${CHUBO_ROOT}/_out/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-e2e-docker}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-e2e-docker-state}"
WORKDIR="${WORKDIR:-/tmp/chubo-e2e-docker-work}"
SUBNET="${SUBNET:-10.5.0.0/24}"
NODE_CONTAINER="${NODE_CONTAINER:-${CLUSTER_NAME}-controlplane-1}"

CHUBO_IMAGE_LOCAL="${CHUBO_IMAGE_LOCAL:-${TALOS_IMAGE_LOCAL:-localhost/chubo/chubo:dev}}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-600}"
SLEEP_SECONDS="${SLEEP_SECONDS:-3}"

SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chubo-support-e2e-docker.zip}"
SUPPORT_LISTING="${WORKDIR}/support-listing.txt"
CHUBOCONFIG_FILE="${WORKDIR}/chuboconfig"

cluster_created=0
NODE_IP=""

require_cmd() {
	local cmd="$1"

	if ! command -v "${cmd}" >/dev/null 2>&1; then
		echo "required command not found: ${cmd}" >&2

		exit 1
	fi
}

ensure_docker_host() {
	if [[ -n "${DOCKER_HOST:-}" ]]; then
		return 0
	fi

	local context_name context_host

	context_name="$(docker context show 2>/dev/null || true)"
	if [[ -z "${context_name}" ]]; then
		return 0
	fi

	context_host="$(docker context inspect "${context_name}" --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)"
	if [[ -z "${context_host}" || "${context_host}" == "<no value>" ]]; then
		return 0
	fi

	export DOCKER_HOST="${context_host}"
	echo "using DOCKER_HOST from docker context ${context_name}: ${DOCKER_HOST}"
}

check_host_support() {
	local host_os

	host_os="$(uname -s | tr '[:upper:]' '[:lower:]')"

	if [[ "${host_os}" != "linux" ]] && [[ "${ALLOW_UNSUPPORTED_DOCKER:-0}" != "1" ]]; then
		echo "docker provisioner fallback is only supported on Linux hosts." >&2
		echo "host=${host_os} is known to fail with Chubo OS container runtime requirements (seccomp/mount attrs)." >&2
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
		"${CHUBOCTL}" version --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
}

cleanup() {
	set +e

	if ((cluster_created == 1)); then
		"${CHUBOCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy >/dev/null 2>&1
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
ensure_docker_host

if ! docker version >/dev/null 2>&1; then
	echo "docker CLI is available but cannot connect to a daemon." >&2
	echo "hint: set DOCKER_HOST explicitly or select a working docker context before running this script." >&2
	exit 1
fi

if [[ ! -x "${CHUBOCTL}" ]]; then
	ctl_target="chuboctl-${HOST_GOOS}-${HOST_GOARCH}"
	if [[ "${CHUBOCTL##*/}" == talosctl-* ]]; then
		ctl_target="talosctl-${HOST_GOOS}-${HOST_GOARCH}"
	fi

	make "${ctl_target}"
fi

mkdir -p "${WORKDIR}" "${ARTIFACTS}" "${STATE_DIR}"
rm -f "${CHUBOCONFIG_FILE}" "${SUPPORT_OUT}" "${SUPPORT_LISTING}"
"${CHUBOCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy >/dev/null 2>&1 || true

echo "building chubo OS docker image"
make docker-chubo \
	DEST="${ARTIFACTS}" \
	GO_BUILDTAGS="${GO_BUILDTAGS}" \
	PLATFORM="linux/${ARCH}" \
	INSTALLER_ARCH=targetarch \
	IMAGE_REGISTRY=localhost \
	USERNAME=chubo \
	IMAGE_TAG_OUT=dev

docker load -i "${ARTIFACTS}/chubo.tar" >/dev/null

echo "creating single-node Docker provisioner cluster"
cluster_created=1
if ! "${CHUBOCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster create docker \
	--image "${CHUBO_IMAGE_LOCAL}" \
	--workers 0 \
	--subnet "${SUBNET}" \
	--chuboconfig-destination "${CHUBOCONFIG_FILE}"; then
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
"${CHUBOCTL}" version --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
./hack/chubo/check-runtime-surface.sh \
	--chuboctl "${CHUBOCTL}" \
	--chuboconfig "${CHUBOCONFIG_FILE}" \
	--endpoint "${NODE_IP}" \
	--node "${NODE_IP}"

echo "collecting support bundle"
"${CHUBOCTL}" support \
	--chuboconfig "${CHUBOCONFIG_FILE}" \
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
