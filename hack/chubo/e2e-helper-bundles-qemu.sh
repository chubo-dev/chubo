#!/usr/bin/env bash
set -euo pipefail

# End-to-end smoke for Chubo workload helper bundles in QEMU/vmnet:
# install -> runtime mTLS -> nomadconfig/consulconfig/openbaoconfig extraction.
#
# This is optimized for fast local validation on macOS arm64.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHUBO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${CHUBO_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chubo}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"

CHUBOCTL_BASE="${CHUBOCTL_BASE:-${TALOSCTL_BASE:-${CHUBO_ROOT}/_out/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}}"
CHUBOCTL_CHUBO="${CHUBOCTL_CHUBO:-${TALOSCTL_CHUBO:-${CHUBO_ROOT}/_out/chubo/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-helper-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"
REGISTRY_LOCAL_ADDR="${REGISTRY_LOCAL_ADDR:-localhost:${REGISTRY_PORT}}"
REGISTRY_BUILD_ADDR="${REGISTRY_BUILD_ADDR:-host.docker.internal:${REGISTRY_PORT}}"
USERNAME="${USERNAME:-chubo}"

INSTALLER_TAG="${INSTALLER_TAG:-helper$(date +%s)}"
INSTALLER_IMAGE_NODE="${INSTALLER_IMAGE_NODE:-10.0.2.2:${REGISTRY_PORT}/${USERNAME}/installer:${INSTALLER_TAG}-arm64}"
REGISTRY_MIRROR_NODE="${REGISTRY_MIRROR_NODE:-10.0.2.2:${REGISTRY_PORT}=http://10.0.2.2:${REGISTRY_PORT}}"
SKIP_BUILD="${SKIP_BUILD:-0}"
OPENGYOZA_ARTIFACT_URL="${OPENGYOZA_ARTIFACT_URL:-}"
OPENGYOZA_VERSION="${OPENGYOZA_VERSION:-1.6.4}"
OPENGYOZA_MIRROR_PORT="${OPENGYOZA_MIRROR_PORT:-5010}"
OPENGYOZA_MIRROR_PID=0
BUILDX_BUILDER="${BUILDX_BUILDER:-local}"
OPENBAO_MODE="${OPENBAO_MODE:-nomadJob}"
WITH_OPENBAO="${WITH_OPENBAO:-1}"
REQUESTED_OPENBAO_MODE="${OPENBAO_MODE}"

HOST_PORT_ENV_SET=0
if [[ -n "${HOST_PORT+x}" ]]; then
	HOST_PORT_ENV_SET=1
fi

HOST_PORT="${HOST_PORT:-50000}"
VMNET_ENABLE="${VMNET_ENABLE:-0}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-900}"
SLEEP_SECONDS="${SLEEP_SECONDS:-2}"
RETRY_ATTEMPTS="${RETRY_ATTEMPTS:-5}"
RETRY_SLEEP_SECONDS="${RETRY_SLEEP_SECONDS:-5}"
RUN_DIR="${RUN_DIR:-$(mktemp -d /tmp/chubo-helper-e2e.XXXXXX)}"
KEEP_VM="${KEEP_VM:-0}"
PRUNE_KEEP="${PRUNE_KEEP:-3}"
CLEANUP_ONLY="${CLEANUP_ONLY:-0}"

SECRETS_FILE="${RUN_DIR}/secrets.yaml"
MACHINECONFIG_INSTALL="${RUN_DIR}/machineconfig-install.yaml"
MACHINECONFIG_RUNTIME="${RUN_DIR}/machineconfig-runtime.yaml"
CHUBOCONFIG_FILE="${RUN_DIR}/chuboconfig"
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

port_in_use() {
	local port="$1"

	# Best-effort: lsof is available by default on macOS and keeps this script dependency-light.
	if command -v lsof >/dev/null 2>&1; then
		lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
		return $?
	fi

	return 1
}

pick_free_port() {
	local port="$1"
	local attempts=256

	for _ in $(seq 1 "${attempts}"); do
		if ! port_in_use "${port}"; then
			echo "${port}"
			return 0
		fi

		port=$((port + 1))
	done

	echo "failed to find a free TCP port starting at ${1}" >&2
	return 1
}

ensure_opengyoza_artifact_url() {
	# opengyoza is currently a private repo. The guest can't fetch release assets without
	# a token, so we mirror the selected asset via a local HTTP server and point the node
	# at 10.0.2.2 (slirp host gateway).
	if [[ -n "${OPENGYOZA_ARTIFACT_URL}" ]]; then
		return 0
	fi

	local tag="v${OPENGYOZA_VERSION}"
	local asset="gyoza_${OPENGYOZA_VERSION}_linux_arm64.zip"
	local public_url="https://github.com/opengyoza/opengyoza/releases/download/${tag}/${asset}"

	if curl -fsSLI "${public_url}" >/dev/null 2>&1; then
		OPENGYOZA_ARTIFACT_URL="${public_url}"
		return 0
	fi

	if ! command -v gh >/dev/null 2>&1; then
		echo "OPENGYOZA_ARTIFACT_URL is unset, and ${public_url} is not reachable. Install gh or set OPENGYOZA_ARTIFACT_URL." >&2
		return 1
	fi

	if ! command -v python3 >/dev/null 2>&1; then
		echo "python3 is required to mirror private opengyoza assets. Install python3 or set OPENGYOZA_ARTIFACT_URL." >&2
		return 1
	fi

	local dir="${RUN_DIR}/opengyoza-artifacts"
	mkdir -p "${dir}"
	local local_path="${dir}/${asset}"

	if [[ ! -f "${local_path}" ]]; then
		echo "downloading opengyoza release asset via gh (${tag}/${asset})"
		if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
			# The run dir is created by root when invoked under sudo. Since we run `gh` as the
			# invoker (to reuse its auth/token config), ensure the invoker can write into it.
			chown_run_dir_to_invoker
			su - "${SUDO_USER}" -c "gh release download \"${tag}\" -R opengyoza/opengyoza -p \"${asset}\" -D \"${dir}\" --clobber"
		else
			gh release download "${tag}" -R opengyoza/opengyoza -p "${asset}" -D "${dir}" --clobber
		fi
	fi

	local port="${OPENGYOZA_MIRROR_PORT}"
	if port_in_use "${port}"; then
		port="$(pick_free_port "${port}")"
	fi

	echo "serving opengyoza artifact on :${port}"
	python3 -m http.server "${port}" --directory "${dir}" --bind 0.0.0.0 >"${RUN_DIR}/opengyoza-mirror.log" 2>&1 &
	OPENGYOZA_MIRROR_PID=$!

	# Wait for the server to come up so the guest doesn't race.
	retry curl -fsS --range 0-0 "http://localhost:${port}/${asset}" -o /dev/null
	OPENGYOZA_ARTIFACT_URL="http://10.0.2.2:${port}/${asset}"
}

ensure_local_api_sans() {
	# For slirp/usernet QEMU runs, the guest is only reachable via hostfwd (127.0.0.1:$HOST_PORT).
	# Add 127.0.0.1/localhost to API cert SANs so runtime mTLS can work without a bridged NIC.
	local file="$1"

	if rg -q '^[[:space:]]*certSANs:' "${file}"; then
		return 0
	fi

	local tmp="${file}.tmp"
	awk '
		/^machine:[[:space:]]*$/ && !done {
			print
			print "  certSANs:"
			print "    - 127.0.0.1"
			print "    - localhost"
			done = 1
			next
		}
		{ print }
	' "${file}" >"${tmp}"
	mv "${tmp}" "${file}"
}

chown_run_dir_to_invoker() {
	# When running under sudo, keep artifacts readable by the invoker (debugging, iteration).
	if [[ "$(id -u)" -ne 0 ]]; then
		return 0
	fi

	if [[ -n "${SUDO_UID:-}" && -n "${SUDO_GID:-}" ]]; then
		chown -R "${SUDO_UID}:${SUDO_GID}" "${RUN_DIR}" >/dev/null 2>&1 || true
	fi
}

retry() {
	local max="${RETRY_ATTEMPTS}"
	local sleep_seconds="${RETRY_SLEEP_SECONDS}"
	local attempt=1

	while true; do
		if "$@"; then
			return 0
		fi

		if ((attempt >= max)); then
			echo "command failed after ${attempt}/${max} attempts: $*" >&2
			return 1
		fi

		echo "command failed (attempt ${attempt}/${max}), retrying in ${sleep_seconds}s: $*" >&2
		sleep "${sleep_seconds}"

		attempt=$((attempt + 1))
		sleep_seconds=$((sleep_seconds * 2))
		if ((sleep_seconds > 60)); then
			sleep_seconds=60
		fi
	done
}

make_with_tags() {
	GO_BUILDTAGS="${GO_BUILDTAGS}" make "$@"
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
	# macOS ships Bash 3.2 by default, which doesn't have `mapfile`.
	local dirs=()
	while IFS= read -r dir; do
		dirs+=("${dir}")
	done < <(ls -1td /private/tmp/chubo-helper-e2e.* 2>/dev/null || true)

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

ensure_buildx_builder() {
	# Buildx "docker" driver (default with colima) can hang after large builds.
	# Prefer a docker-container builder for deterministic, non-hanging local runs.
	if docker buildx inspect --builder "${BUILDX_BUILDER}" --bootstrap >/dev/null 2>&1; then
		# Chubo builds rely on `RUN --security=insecure`; require the builder to allow that entitlement.
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
				# chuboctl prints addresses as either "<ip>/<cidr>" or "<iface>/<ip>/<cidr>".
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

	if ! output="$("${CHUBOCTL_CHUBO}" get openwontonstatus --namespace chubo -o json --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>/dev/null)"; then
		return 1
	fi

	jq -e '.spec.configured == true and .spec.healthy == true and .spec.aclReady == true and .spec.binaryMode == "artifact"' <<<"${output}" >/dev/null 2>&1 || return 1
}

opengyoza_ready() {
	local output

	if ! output="$("${CHUBOCTL_CHUBO}" get opengyozastatus --namespace chubo -o json --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>/dev/null)"; then
		return 1
	fi

	jq -e '.spec.configured == true and .spec.healthy == true and .spec.aclReady == true and .spec.binaryMode == "artifact"' <<<"${output}" >/dev/null 2>&1 || return 1
}

openbao_job_ready() {
	local output

	if ! output="$("${CHUBOCTL_CHUBO}" get openbaojobstatus --namespace chubo -o json --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" 2>/dev/null)"; then
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
	"${CHUBOCTL_CHUBO}" get openwontonstatus --namespace chubo -o json --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- openwontonstatus (yaml)"
	"${CHUBOCTL_CHUBO}" get openwontonstatus --namespace chubo -o yaml --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- opengyozastatus (json)"
	"${CHUBOCTL_CHUBO}" get opengyozastatus --namespace chubo -o json --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- opengyozastatus (yaml)"
	"${CHUBOCTL_CHUBO}" get opengyozastatus --namespace chubo -o yaml --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- node time (runtime API)"
	"${CHUBOCTL_CHUBO}" time --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- openwonton TLS cert (subject/dates)"
	"${CHUBOCTL_CHUBO}" read /var/lib/chubo/certs/openwonton/server.pem --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" \
		| openssl x509 -noout -subject -dates 2>/dev/null || true
	echo

	echo "-- opengyoza TLS cert (subject/dates)"
	"${CHUBOCTL_CHUBO}" read /var/lib/chubo/certs/opengyoza/server.pem --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" \
		| openssl x509 -noout -subject -dates 2>/dev/null || true
	echo

	echo "-- v1alpha1 service openwonton (yaml)"
	"${CHUBOCTL_CHUBO}" get svc --namespace v1alpha1 openwonton -o yaml --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
	echo

	echo "-- v1alpha1 service opengyoza (yaml)"
	"${CHUBOCTL_CHUBO}" get svc --namespace v1alpha1 opengyoza -o yaml --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" || true
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

	stop_opengyoza_mirror

	if [[ "${QEMU_INSTALL_PID}" -gt 0 ]]; then
		kill "${QEMU_INSTALL_PID}" >/dev/null 2>&1 || true
	fi

	if [[ "${QEMU_DISK_PID}" -gt 0 ]] && [[ "${KEEP_VM}" -eq 0 ]]; then
		kill "${QEMU_DISK_PID}" >/dev/null 2>&1 || true
	fi

	if [[ "${KEEP_VM}" -eq 0 ]]; then
		pkill -f "qemu-system-aarch64.*${RUN_DIR}" >/dev/null 2>&1 || true
	fi

	chown_run_dir_to_invoker
}

stop_opengyoza_mirror() {
	if [[ "${OPENGYOZA_MIRROR_PID}" -le 0 ]]; then
		return 0
	fi

	kill "${OPENGYOZA_MIRROR_PID}" >/dev/null 2>&1 || true
	wait "${OPENGYOZA_MIRROR_PID}" >/dev/null 2>&1 || true
	OPENGYOZA_MIRROR_PID=0
}

read_openbao_helper_env() {
	local key="$1"
	local env_file="${HELPERS_DIR}/openbaoconfig/openbao.env"

	[[ -f "${env_file}" ]] || return 1

	awk -F= -v key="${key}" '$1 == key {print substr($0, index($0, "=") + 1)}' "${env_file}"
}

generate_machineconfig() {
	local mode="$1"
	local output="$2"
	local config_image="$3"
	local wipe="$4"
	local disk="$5"
	local openbao_address="${6:-}"
	local openbao_token="${7:-}"
	local args=(
		--with-secrets "${SECRETS_FILE}"
		--install-disk "${disk}"
		--install-image "${config_image}"
		--registry-mirror "${REGISTRY_MIRROR_NODE}"
		--with-chubo
		--chubo-role server
		-o "${output}"
	)

	if [[ "${wipe}" == "false" ]]; then
		args+=(--wipe=false)
	fi

	if [[ "${WITH_OPENBAO}" == "1" ]]; then
		args+=(
			--with-openbao
			--openbao-mode "${mode}"
		)
		if [[ -n "${openbao_address}" ]]; then
			args+=(--openbao-vault-address "${openbao_address}")
		fi
		if [[ -n "${openbao_token}" ]]; then
			args+=(--openbao-vault-token "${openbao_token}")
		fi
	fi

	if [[ -n "${OPENGYOZA_ARTIFACT_URL}" ]]; then
		args+=(--opengyoza-artifact-url "${OPENGYOZA_ARTIFACT_URL}")
	fi

	"${CHUBOCTL_CHUBO}" gen machineconfig "${args[@]}"
}

switch_to_external_openbao_mode() {
	local helper_addr helper_token runtime_external

	helper_addr="$(read_openbao_helper_env "VAULT_ADDR" || true)"
	helper_token="$(tr -d ' \r\n' < "${HELPERS_DIR}/openbaoconfig/acl.token" 2>/dev/null || true)"

	if [[ -z "${helper_addr}" || -z "${helper_token}" ]]; then
		echo "missing OpenBao helper bundle address/token; cannot switch to external mode" >&2
		return 1
	fi

	runtime_external="${RUN_DIR}/machineconfig-runtime-external.yaml"
	generate_machineconfig "external" "${runtime_external}" "" "false" "/dev/vda" "${helper_addr}" "${helper_token}"

	if [[ "${NODE_IP}" == "127.0.0.1" ]]; then
		ensure_local_api_sans "${runtime_external}"
	fi

	echo "reapplying runtime config in external OpenBao mode"
	"${CHUBOCTL_CHUBO}" apply-config --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" -f "${runtime_external}"

	wait_until "runtime mTLS API (${NODE_IP}) after external OpenBao reapply" "${TIMEOUT_SECONDS}" \
		"${CHUBOCTL_CHUBO}" version --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"

	wait_until "openwontonstatus (healthy + aclReady) after external OpenBao reapply" "${TIMEOUT_SECONDS}" openwonton_ready
	wait_until "opengyozastatus (healthy + aclReady) after external OpenBao reapply" "${TIMEOUT_SECONDS}" opengyoza_ready
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

# Prevent accidental reuse of a stale VM when the default hostfwd port is taken.
if port_in_use "${HOST_PORT}"; then
	if [[ "${HOST_PORT_ENV_SET}" -eq 1 ]]; then
		echo "HOST_PORT=${HOST_PORT} is already in use; pick a free port and retry." >&2
		exit 1
	fi

	HOST_PORT="$(pick_free_port "${HOST_PORT}")"
	echo "HOST_PORT is in use, using free port: ${HOST_PORT}" >&2
fi

if [[ "${CLEANUP_ONLY}" -eq 1 ]]; then
	echo "CLEANUP_ONLY=1: killing any QEMU processes for run dir: ${RUN_DIR}" >&2
	pkill -f "qemu-system-aarch64.*${RUN_DIR}" >/dev/null 2>&1 || true
	exit 0
fi

ensure_opengyoza_artifact_url

if [[ "${SKIP_BUILD}" -eq 1 && -x "${CHUBOCTL_BASE}" ]]; then
	echo "SKIP_BUILD=1, reusing existing chuboctl: ${CHUBOCTL_BASE}"
else
	# chuboctl is used for PKI/secrets generation; rebuild when not explicitly reusing artifacts.
	make chuboctl
fi

# When iterating locally we want the CLI to reflect the current worktree (config rendering,
# helper bundle surfaces, etc). The rebuild cost is small compared to the QEMU flow.
rebuild_chuboctl=0
if [[ ! -x "${CHUBOCTL_CHUBO}" ]]; then
	rebuild_chuboctl=1
elif ! "${CHUBOCTL_CHUBO}" gen machineconfig --help 2>&1 | rg -q -- '--with-chubo'; then
	# The helper-bundles E2E depends on newer `gen machineconfig` flags.
	rebuild_chuboctl=1
elif ! "${CHUBOCTL_CHUBO}" gen machineconfig --help 2>&1 | rg -q -- '--opengyoza-artifact-url'; then
	rebuild_chuboctl=1
fi

if [[ "${rebuild_chuboctl}" -eq 1 ]]; then
	mkdir -p "$(dirname "${CHUBOCTL_CHUBO}")"

	GOOS="${HOST_GOOS}" GOARCH="${HOST_GOARCH}" CGO_ENABLED=0 go build \
		-tags grpcnotrace,chubo \
		-o "${CHUBOCTL_CHUBO}" \
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

	ensure_buildx_builder

	echo "building chubo boot artifacts"
	retry make_with_tags initramfs kernel sd-boot \
		ARTIFACTS="${ARTIFACTS}" \
		GO_BUILDTAGS="${GO_BUILDTAGS}" \
		TARGET_ARGS="--builder=${BUILDX_BUILDER} ${TARGET_ARGS:-}" \
		PLATFORM=linux/arm64

	echo "building installer-base + imager tarballs (${INSTALLER_TAG})"
	retry make_with_tags docker-installer-base docker-imager \
		TARGET_ARGS="--builder=${BUILDX_BUILDER} ${TARGET_ARGS:-}" \
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
"${CHUBOCTL_BASE}" gen secrets -o "${SECRETS_FILE}"
BOOTSTRAP_OPENBAO_MODE="${OPENBAO_MODE}"
if [[ "${WITH_OPENBAO}" == "1" && "${REQUESTED_OPENBAO_MODE}" == "external" ]]; then
	BOOTSTRAP_OPENBAO_MODE="nomadJob"
fi

generate_machineconfig "${BOOTSTRAP_OPENBAO_MODE}" "${MACHINECONFIG_INSTALL}" "${INSTALLER_IMAGE_NODE}" "true" "/dev/vdb"
generate_machineconfig "${BOOTSTRAP_OPENBAO_MODE}" "${MACHINECONFIG_RUNTIME}" "" "false" "/dev/vda"

"${CHUBOCTL_BASE}" gen config chubo https://0.0.0.0:6443 \
	--with-secrets "${SECRETS_FILE}" \
	-t chuboconfig \
	-o "${CHUBOCONFIG_FILE}"

VMNET_MAC=""
if [[ "${VMNET_ENABLE}" -eq 1 ]]; then
	VMNET_MAC="52:54:00:$(openssl rand -hex 3 | sed -E 's/(..)(..)(..)/\1:\2:\3/')"
	echo "booting install media (vmnet mac: ${VMNET_MAC})"
else
	echo "booting install media (slirp only)"
fi

VMNET_ENABLE="${VMNET_ENABLE}" VMNET_MAC="${VMNET_MAC}" QEMU_RUNDIR="${RUN_DIR}" BOOT_FROM_DISK=0 HOST_PORT="${HOST_PORT}" \
	./hack/qemu/chubo-qemu.sh >"${LOG_INSTALL}" 2>&1 &
QEMU_INSTALL_PID=$!

wait_until "maintenance API (install media)" "${TIMEOUT_SECONDS}" \
	"${CHUBOCTL_CHUBO}" get addresses --insecure -e 127.0.0.1 -n 127.0.0.1

"${CHUBOCTL_CHUBO}" get addresses --insecure -e 127.0.0.1 -n 127.0.0.1 >"${RUN_DIR}/addresses-install.txt"
NODE_IP="$(parse_bridged_ip "${RUN_DIR}/addresses-install.txt")"
if [[ -z "${NODE_IP}" ]]; then
	echo "no bridged node IP discovered; falling back to hostfwd runtime API (127.0.0.1:${HOST_PORT})" >&2
	NODE_IP="127.0.0.1"
	ensure_local_api_sans "${MACHINECONFIG_INSTALL}"
	ensure_local_api_sans "${MACHINECONFIG_RUNTIME}"
fi

echo "applying install config"
"${CHUBOCTL_CHUBO}" apply-config -i -e 127.0.0.1 -n 127.0.0.1 -f "${MACHINECONFIG_INSTALL}" || true

wait_until "installer completion marker" "${TIMEOUT_SECONDS}" \
	rg -q "installation of .* complete" "${LOG_INSTALL}"

kill "${QEMU_INSTALL_PID}" >/dev/null 2>&1 || true
QEMU_INSTALL_PID=0
sleep 1
pkill -f "qemu-system-aarch64.*${RUN_DIR}" >/dev/null 2>&1 || true
sleep 1

echo "booting installed disk"
cp -f /opt/homebrew/share/qemu/edk2-arm-vars.fd "${RUN_DIR}/edk2-vars.fd"
VMNET_ENABLE="${VMNET_ENABLE}" VMNET_MAC="${VMNET_MAC}" QEMU_RUNDIR="${RUN_DIR}" BOOT_FROM_DISK=1 HOST_PORT="${HOST_PORT}" \
	./hack/qemu/chubo-qemu.sh >"${LOG_DISK}" 2>&1 &
QEMU_DISK_PID=$!

wait_until "maintenance API (installed disk)" "${TIMEOUT_SECONDS}" \
	"${CHUBOCTL_CHUBO}" get addresses --insecure -e 127.0.0.1 -n 127.0.0.1

echo "applying runtime config and rebooting into runtime API"
"${CHUBOCTL_CHUBO}" apply-config -i -m reboot -e 127.0.0.1 -n 127.0.0.1 -f "${MACHINECONFIG_RUNTIME}" || true

wait_until "runtime mTLS API (${NODE_IP})" "${TIMEOUT_SECONDS}" \
	"${CHUBOCTL_CHUBO}" version --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"

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

# Artifact download is complete once services are healthy; stop mirror to avoid hanging on shell exit.
stop_opengyoza_mirror

echo "downloading helper bundles"
mkdir -p "${HELPERS_DIR}"
"${CHUBOCTL_CHUBO}" nomadconfig "${HELPERS_DIR}" --force --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
"${CHUBOCTL_CHUBO}" consulconfig "${HELPERS_DIR}" --force --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
"${CHUBOCTL_CHUBO}" openbaoconfig "${HELPERS_DIR}" --force --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"

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

if [[ "${WITH_OPENBAO}" == "1" && "${REQUESTED_OPENBAO_MODE}" == "external" ]]; then
	switch_to_external_openbao_mode
fi

{
	echo "run=${RUN_DIR}"
	echo "node_ip=${NODE_IP}"
	echo "installer_tag=${INSTALLER_TAG}"
	"${CHUBOCTL_CHUBO}" version --chuboconfig "${CHUBOCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" \
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
