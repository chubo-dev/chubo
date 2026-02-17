#!/usr/bin/env bash
set -euo pipefail

# End-to-end multi-node Chubo module flow in QEMU:
# install -> runtime mTLS -> openwonton/opengyoza healthy -> peers converge
#
# This validates that `bootstrapExpect` + `join` in `modules.chubo` render correctly and
# form a real cluster (not a mocked/quorum-override scenario).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALOS_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${TALOS_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chubo}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL:--tags grpcnotrace,chubo}"
ARCH="${ARCH:-amd64}"
SKIP_BUILD="${SKIP_BUILD:-0}"
HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"
CHUBOCTL="${CHUBOCTL:-${TALOSCTL:-${TALOS_ROOT}/_out/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}}"
BUILDX_BUILDER="${BUILDX_BUILDER:-local}"

CONTROLPLANE_COUNT="${CONTROLPLANE_COUNT:-3}"

RUN_ID="${RUN_ID:-$RANDOM}"
BASE_NET_OCTET="${BASE_NET_OCTET:-$((100 + RANDOM % 100))}"
BASE_NET_SUBNET="${BASE_NET_SUBNET:-$((10 + RANDOM % 200))}"
CONTROL_PLANE_PORT="${CONTROL_PLANE_PORT:-$((7400 + RANDOM % 400))}"
CLUSTER_CREATE_MAX_ATTEMPTS="${CLUSTER_CREATE_MAX_ATTEMPTS:-3}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-cluster-${RUN_ID}}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-cluster-state-${RUN_ID}}"
WORKDIR="${WORKDIR:-/tmp/chubo-cluster-work-${RUN_ID}}"
CIDR="${CIDR:-10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.0/24}"
NET_PREFIX="${NET_PREFIX:-10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}}"
INSTALL_DISK="${INSTALL_DISK:-/dev/vda}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-cluster-reg-${RUN_ID}}"
REGISTRY_PORT="${REGISTRY_PORT:-$((5100 + RANDOM % 300))}"
REGISTRY_LOCAL_ADDR="${REGISTRY_LOCAL_ADDR:-}"
REGISTRY_NODE_ADDR="${REGISTRY_NODE_ADDR:-}"
INSTALLER_BASE_IMAGE_LOCAL="${INSTALLER_BASE_IMAGE_LOCAL:-}"
INSTALLER_IMAGE_LOCAL="${INSTALLER_IMAGE_LOCAL:-}"
INSTALLER_IMAGE_NODE="${INSTALLER_IMAGE_NODE:-}"
REGISTRY_MIRROR_NODE="${REGISTRY_MIRROR_NODE:-}"

TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1800}"
SLEEP_SECONDS="${SLEEP_SECONDS:-3}"
MAINTENANCE_PERSIST_SECONDS="${MAINTENANCE_PERSIST_SECONDS:-30}"
MAINTENANCE_FALLBACK_SECONDS="${MAINTENANCE_FALLBACK_SECONDS:-180}"

SECRETS_FILE="${WORKDIR}/secrets.yaml"
TALOSCONFIG_FILE="${WORKDIR}/talosconfig"
CLUSTER_CREATE_LOG="${WORKDIR}/cluster-create.log"

CRANE_BIN=""

cluster_created=0
registry_started=0

HELPERS_DIR="${WORKDIR}/helpers"
NOMAD_CURL_ARGS=()
CONSUL_CURL_ARGS=()

CLEANUP_STALE_ONLY=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--skip-build)
		SKIP_BUILD=1
		;;
	--cleanup-stale-only)
		CLEANUP_STALE_ONLY=1
		;;
	-h | --help)
		echo "usage: $0 [--skip-build] [--cleanup-stale-only]"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		echo "usage: $0 [--skip-build] [--cleanup-stale-only]" >&2
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
	: "${REGISTRY_NODE_ADDR:=${NET_PREFIX}.1:${REGISTRY_PORT}}"
	: "${INSTALLER_BASE_IMAGE_LOCAL:=${REGISTRY_LOCAL_ADDR}/chubo/installer-base:dev}"
	: "${INSTALLER_IMAGE_LOCAL:=${REGISTRY_LOCAL_ADDR}/chubo/installer:dev}"
	: "${INSTALLER_IMAGE_NODE:=${REGISTRY_NODE_ADDR}/chubo/installer:dev}"
	: "${REGISTRY_MIRROR_NODE:=${REGISTRY_NODE_ADDR}=http://${REGISTRY_NODE_ADDR}}"
}

pick_registry_port() {
	local candidate="${REGISTRY_PORT}"
	local attempts=0

	while lsof -nP -iTCP:"${candidate}" -sTCP:LISTEN >/dev/null 2>&1; do
		attempts=$((attempts + 1))
		if ((attempts > 20)); then
			echo "failed to find a free registry port (starting at ${REGISTRY_PORT})" >&2
			return 1
		fi

		candidate=$((5400 + RANDOM % 500))
	done

	REGISTRY_PORT="${candidate}"
	return 0
}

start_registry() {
	echo "starting local OCI registry on :${REGISTRY_PORT}"
	docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1 || true

	if docker run -d --rm --name "${REGISTRY_NAME}" -p "${REGISTRY_PORT}:5000" registry:2 >/dev/null 2>&1; then
		registry_started=1
		refresh_registry_refs
		return 0
	fi

	echo "failed to start local registry on :${REGISTRY_PORT}" >&2
	return 1
}

cleanup() {
	set +e

	if [[ -n "${NOMAD_API_NODE_IP:-}" && ${#NOMAD_CURL_ARGS[@]} -gt 0 ]]; then
		curl -fsS --max-time 5 "${NOMAD_CURL_ARGS[@]}" -X DELETE "https://${NOMAD_API_NODE_IP}:4646/v1/job/chubo-cluster-e2e-${RUN_ID}?purge=true" >/dev/null 2>&1 || true
	fi

	if ((cluster_created == 1)); then
		"${CHUBOCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy \
			--provisioner qemu >/dev/null 2>&1
	fi

	if ((registry_started == 1)); then
		docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1
	fi
}

trap cleanup EXIT

cleanup_stale_clusters() {
	shopt -s nullglob

	local stale_qemu_pids=""
	stale_qemu_pids="$(/bin/ps -ax -o pid=,command= | /usr/bin/awk '$0 ~ /qemu-system-/ && index($0, "/tmp/chubo-cluster-state-") { print $1 }' 2>/dev/null || true)"
	if [[ -n "${stale_qemu_pids}" ]]; then
		echo "killing stale QEMU processes: ${stale_qemu_pids}"
		/bin/kill ${stale_qemu_pids} >/dev/null 2>&1 || true
		/bin/sleep 1
		for pid in ${stale_qemu_pids}; do
			/bin/kill -0 "${pid}" >/dev/null 2>&1 || continue
			/bin/kill -9 "${pid}" >/dev/null 2>&1 || true
		done
	fi

	for state_root in /tmp/chubo-cluster-state-*; do
		[[ -d "${state_root}" ]] || continue

		local cluster_dir
		for cluster_dir in "${state_root}"/chubo-cluster-*; do
			[[ -d "${cluster_dir}" ]] || continue

			local cluster_name
			cluster_name="$(basename "${cluster_dir}")"

			echo "destroying stale cluster ${cluster_name} (state dir ${state_root})"
			"${CHUBOCTL}" --state "${state_root}" --name "${cluster_name}" cluster destroy --provisioner qemu >/dev/null 2>&1 || true
		done
	done

	for state_root in /tmp/chubo-cluster-state-* /tmp/chubo-cluster-work-*; do
		[[ -e "${state_root}" ]] || continue
		rm -rf "${state_root}"
	done
}

service_is_up() {
	local node_ip="$1"
	local service_name="$2"

	"${CHUBOCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${node_ip}" -n "${node_ip}" service "${service_name}" 2>/dev/null |
		grep -qi "Health check successful"
}

wait_for_service_up() {
	local node_ip="$1"
	local service_name="$2"

	wait_until "${service_name} healthy on ${node_ip}" "${TIMEOUT_SECONDS}" \
		service_is_up "${node_ip}" "${service_name}"
}

wait_for_maintenance() {
	local node_ip="$1"

	wait_until "maintenance API on ${node_ip}" "${TIMEOUT_SECONDS}" \
		"${CHUBOCTL}" get addresses --insecure -e "${node_ip}" -n "${node_ip}"
}

wait_for_runtime() {
	local node_ip="$1"

	wait_until "runtime mTLS API on ${node_ip}" "${TIMEOUT_SECONDS}" \
		"${CHUBOCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${node_ip}" -n "${node_ip}"
}

apply_install_and_wait() {
	local node_ip="$1"
	local install_cfg="$2"
	local runtime_cfg="$3"

	echo "applying install config to ${node_ip}"
	"${CHUBOCTL}" apply-config --insecure -m reboot -e "${node_ip}" -n "${node_ip}" -f "${install_cfg}"

	echo "waiting for post-install transition on ${node_ip}"
	local transition_deadline=$((SECONDS + TIMEOUT_SECONDS))
	local saw_maintenance_down=0
	local maintenance_reentered_at=0
	local maintenance_up_since=0
	local runtime_config_applied=0

	while true; do
		if "${CHUBOCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${node_ip}" -n "${node_ip}" >/dev/null 2>&1; then
			echo "${node_ip}: runtime mTLS became available after install apply"
			break
		fi

		if "${CHUBOCTL}" get addresses --insecure -e "${node_ip}" -n "${node_ip}" >/dev/null 2>&1; then
			if ((maintenance_up_since == 0)); then
				maintenance_up_since="${SECONDS}"
			fi

			if ((saw_maintenance_down == 1)); then
				if ((maintenance_reentered_at == 0)); then
					maintenance_reentered_at="${SECONDS}"
				fi

				if ((SECONDS - maintenance_reentered_at >= MAINTENANCE_PERSIST_SECONDS)); then
					echo "${node_ip}: maintenance persisted after reboot; applying runtime config and rebooting"
					"${CHUBOCTL}" apply-config --insecure -m reboot -e "${node_ip}" -n "${node_ip}" -f "${runtime_cfg}"
					runtime_config_applied=1
					break
				fi
			elif ((runtime_config_applied == 0)) && ((SECONDS - maintenance_up_since >= MAINTENANCE_FALLBACK_SECONDS)); then
				echo "${node_ip}: maintenance persisted for ${MAINTENANCE_FALLBACK_SECONDS}s; forcing runtime config and reboot"
				"${CHUBOCTL}" apply-config --insecure -m reboot -e "${node_ip}" -n "${node_ip}" -f "${runtime_cfg}"
				runtime_config_applied=1
				break
			fi
		else
			saw_maintenance_down=1
			maintenance_reentered_at=0
			maintenance_up_since=0
		fi

		if ((SECONDS >= transition_deadline)); then
			echo "${node_ip}: timed out waiting for post-install transition" >&2
			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done

	if ! wait_for_runtime "${node_ip}"; then
		if ((runtime_config_applied == 0)); then
			echo "${node_ip}: runtime mTLS did not come up after install; applying runtime config and rebooting"
			wait_for_maintenance "${node_ip}"
			"${CHUBOCTL}" apply-config --insecure -m reboot -e "${node_ip}" -n "${node_ip}" -f "${runtime_cfg}"
		else
			echo "${node_ip}: runtime mTLS did not come up after runtime config apply; retrying runtime wait"
		fi

		wait_for_runtime "${node_ip}"
	fi
}

download_helper_bundles() {
	local node_ip="$1"

	mkdir -p "${HELPERS_DIR}"

	# Download once and reuse for curl-based TLS probes across the cluster.
	"${CHUBOCTL}" nomadconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${node_ip}" -n "${node_ip}"
	"${CHUBOCTL}" consulconfig "${HELPERS_DIR}" --force --talosconfig "${TALOSCONFIG_FILE}" -e "${node_ip}" -n "${node_ip}"

	local nomad_token consul_token
	nomad_token="$(tr -d '\r\n' <"${HELPERS_DIR}/nomadconfig/acl.token")"
	consul_token="$(tr -d '\r\n' <"${HELPERS_DIR}/consulconfig/acl.token")"

	NOMAD_CURL_ARGS=(--cacert "${HELPERS_DIR}/nomadconfig/ca.pem" --cert "${HELPERS_DIR}/nomadconfig/client.pem" --key "${HELPERS_DIR}/nomadconfig/client-key.pem" -H "X-Nomad-Token: ${nomad_token}")
	CONSUL_CURL_ARGS=(--cacert "${HELPERS_DIR}/consulconfig/ca.pem" --cert "${HELPERS_DIR}/consulconfig/client.pem" --key "${HELPERS_DIR}/consulconfig/client-key.pem" -H "X-Consul-Token: ${consul_token}")
	NOMAD_API_NODE_IP="${node_ip}"
}

nomad_job_reaches_terminal_success() {
	local node_ip="$1"
	local job_id="$2"

	local allocs
	allocs="$(curl -fsS --max-time 5 "${NOMAD_CURL_ARGS[@]}" "https://${node_ip}:4646/v1/job/${job_id}/allocations" 2>/dev/null || true)"
	if [[ -z "${allocs}" ]]; then
		return 1
	fi

	jq -e 'length > 0 and any(.[]; .ClientStatus == "complete") and all(.[]; .ClientStatus != "failed" and .ClientStatus != "lost")' <<<"${allocs}" >/dev/null
}

submit_and_verify_nomad_probe_job() {
	local node_ip="$1"
	local job_id="chubo-cluster-e2e-${RUN_ID}"
	local payload_file="${WORKDIR}/${job_id}.json"

	cat >"${payload_file}" <<EOF
{
  "Job": {
    "ID": "${job_id}",
    "Name": "${job_id}",
    "Type": "batch",
    "Datacenters": ["dc1"],
    "TaskGroups": [
      {
        "Name": "probe",
        "Count": 1,
        "Tasks": [
          {
            "Name": "probe",
            "Driver": "exec",
            "Config": {
              "command": "/bin/sh",
              "args": ["-c", "echo chubo cluster e2e > /tmp/chubo-cluster-e2e.txt"]
            },
            "Resources": {
              "CPU": 100,
              "MemoryMB": 64
            }
          }
        ]
      }
    ]
  }
}
EOF

	echo "submitting Nomad probe job (${job_id})"
	curl -fsS --max-time 10 "${NOMAD_CURL_ARGS[@]}" \
		-H "Content-Type: application/json" \
		-X POST \
		--data @"${payload_file}" \
		"https://${node_ip}:4646/v1/jobs" >/dev/null

	wait_until "nomad probe job ${job_id} complete" "${TIMEOUT_SECONDS}" \
		nomad_job_reaches_terminal_success "${node_ip}" "${job_id}"

	echo "purging Nomad probe job (${job_id})"
	curl -fsS --max-time 5 "${NOMAD_CURL_ARGS[@]}" -X DELETE "https://${node_ip}:4646/v1/job/${job_id}?purge=true" >/dev/null || true
}

nomad_peers_ok() {
	local node_ip="$1"
	local expected="$2"

	local count
	count="$(curl -fsS --max-time 2 "${NOMAD_CURL_ARGS[@]}" "https://${node_ip}:4646/v1/status/peers" | jq -r 'length' 2>/dev/null || true)"
	[[ "${count}" == "${expected}" ]]
}

consul_peers_ok() {
	local node_ip="$1"
	local expected="$2"

	local count
	count="$(curl -fsS --max-time 2 "${CONSUL_CURL_ARGS[@]}" "https://${node_ip}:8500/v1/status/peers" | jq -r 'length' 2>/dev/null || true)"
	[[ "${count}" == "${expected}" ]]
}

wait_for_peers() {
	local name="$1"
	local expected="$2"
	shift 2

	local deadline=$((SECONDS + TIMEOUT_SECONDS))
	local attempts=0

	while true; do
		local ok=1
		for ip in "$@"; do
			if [[ "${name}" == "nomad" ]]; then
				nomad_peers_ok "${ip}" "${expected}" || ok=0
			else
				consul_peers_ok "${ip}" "${expected}" || ok=0
			fi
		done

		if ((ok == 1)); then
			return 0
		fi

		attempts=$((attempts + 1))
		if ((attempts % 10 == 0)); then
			echo "waiting for ${name} peers=${expected} on: $*"
		fi

		if ((SECONDS >= deadline)); then
			echo "timed out waiting for ${name} peers=${expected}" >&2
			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
}

run_cluster_create() {
	local monitor_path="${STATE_DIR}/${CLUSTER_NAME}/${CLUSTER_NAME}-controlplane-1.monitor"

	if ((${#monitor_path} >= 104)); then
		echo "qemu monitor path too long (${#monitor_path} bytes): ${monitor_path}" >&2
		echo "set shorter STATE_DIR and/or CLUSTER_NAME" >&2
		return 1
	fi

	"${CHUBOCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster create dev \
		--arch "${ARCH}" \
		--cidr "${CIDR}" \
		--control-plane-port "${CONTROL_PLANE_PORT}" \
		--controlplanes "${CONTROLPLANE_COUNT}" \
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
		echo "creating ${CONTROLPLANE_COUNT}-node QEMU cluster in maintenance mode (attempt ${attempt}/${CLUSTER_CREATE_MAX_ATTEMPTS})"
		if run_cluster_create 2>&1 | tee "${CLUSTER_CREATE_LOG}"; then
			cluster_created=1
			return 0
		fi

			if grep -Eq 'interface bridge[0-9]+ not found' "${CLUSTER_CREATE_LOG}" && ((attempt < CLUSTER_CREATE_MAX_ATTEMPTS)); then
				echo "cluster create hit bridge bring-up race; destroying partial state and retrying"
				"${CHUBOCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy --provisioner qemu >/dev/null 2>&1 || true
				rm -rf "${STATE_DIR:?}/${CLUSTER_NAME}"
				attempt=$((attempt + 1))
				sleep 2
			continue
		fi

		echo "cluster create failed; see ${CLUSTER_CREATE_LOG}" >&2
		return 1
	done
}

if [[ "${ARCH}" != "amd64" ]]; then
	echo "unsupported ARCH=${ARCH} (only amd64 is supported by this script)" >&2
	exit 2
fi

if [[ "${EUID}" -ne 0 ]]; then
	echo "error: qemu cluster fixture requires root; run with \`sudo -E\`" >&2
	exit 1
fi

if ((CLEANUP_STALE_ONLY == 1)); then
	cleanup_stale_clusters
	exit 0
fi

require_cmd docker
require_cmd go
require_cmd make
require_cmd lsof
require_cmd curl
require_cmd jq

rm -rf "${WORKDIR}" "${STATE_DIR}"
mkdir -p "${WORKDIR}" "${ARTIFACTS}"

# On macOS with Colima, Docker is typically exposed via a per-user unix socket.
# This script runs under sudo and uses an isolated DOCKER_CONFIG (below), so
# Docker context resolution won't work unless DOCKER_HOST is set. Best-effort
# auto-detect the default Colima socket for the invoking user.
if [[ -z "${DOCKER_HOST:-}" && "$(uname -s)" == "Darwin" && -n "${SUDO_USER:-}" ]]; then
	colima_home="$(eval echo "~${SUDO_USER}" 2>/dev/null || true)"

	if [[ -n "${colima_home}" && -S "${colima_home}/.colima/default/docker.sock" ]]; then
		export DOCKER_HOST="unix://${colima_home}/.colima/default/docker.sock"
		echo "detected Colima docker socket for ${SUDO_USER}, using DOCKER_HOST=${DOCKER_HOST}"
	fi
fi

export DOCKER_CONFIG="${WORKDIR}/.docker"
mkdir -p "${DOCKER_CONFIG}"
cat >"${DOCKER_CONFIG}/config.json" <<'EOF'
{"auths":{}}
EOF
mkdir -p "${DOCKER_CONFIG}/cli-plugins"
if [[ -x /Applications/Docker.app/Contents/Resources/cli-plugins/docker-buildx ]]; then
	ln -sf /Applications/Docker.app/Contents/Resources/cli-plugins/docker-buildx "${DOCKER_CONFIG}/cli-plugins/docker-buildx"
fi

if ! docker version >/dev/null 2>&1; then
	echo "docker CLI is available but cannot connect to a daemon." >&2
	echo "hint: set DOCKER_HOST before running under sudo (this script uses an isolated DOCKER_CONFIG)." >&2
	exit 1
fi

ctl_target="chuboctl-${HOST_GOOS}-${HOST_GOARCH}"
if [[ "${CHUBOCTL##*/}" == talosctl-* ]]; then
	ctl_target="talosctl-${HOST_GOOS}-${HOST_GOARCH}"
fi

if [[ ! -x "${CHUBOCTL}" ]]; then
	make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
elif ! "${CHUBOCTL}" gen machineconfig --help 2>/dev/null | grep -q -- '--with-chubo'; then
	echo "existing CLI binary is missing --with-chubo; rebuilding"
	make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
fi

if ! command -v crane >/dev/null 2>&1; then
	go install github.com/google/go-containerregistry/cmd/crane@latest
fi

CRANE_BIN="$(command -v crane || true)"
if [[ -z "${CRANE_BIN}" ]]; then
	CRANE_BIN="$(go env GOPATH)/bin/crane"
fi

if [[ ! -x "${CRANE_BIN}" ]]; then
	echo "crane binary not found after installation attempt" >&2
	exit 1
fi

pick_registry_port
refresh_registry_refs
echo "run configuration:"
echo "  cluster: ${CLUSTER_NAME}"
echo "  state dir: ${STATE_DIR}"
echo "  work dir: ${WORKDIR}"
echo "  cidr: ${CIDR}"
	echo "  controlplanes: ${CONTROLPLANE_COUNT}"
	echo "  registry: ${REGISTRY_LOCAL_ADDR} -> ${REGISTRY_NODE_ADDR}"

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
"${CRANE_BIN}" --insecure index append -t "${INSTALLER_IMAGE_LOCAL}" -m "${installer_arch_ref}" >/dev/null

echo "generating secrets and talosconfig"
"${CHUBOCTL}" gen secrets -o "${SECRETS_FILE}"
"${CHUBOCTL}" gen config chubo https://0.0.0.0:6443 \
	--with-secrets "${SECRETS_FILE}" \
	-t talosconfig \
	-o "${TALOSCONFIG_FILE}"

controlplane_ips=()
for ((i = 0; i < CONTROLPLANE_COUNT; i++)); do
	controlplane_ips+=("${NET_PREFIX}.$((2 + i))")
done

echo "expected controlplane IPs: ${controlplane_ips[*]}"

create_cluster_with_retry

install_cfgs=()
runtime_cfgs=()

echo "generating per-node machine configs (modules.chubo enabled, join/bootstrapExpect set)"
for idx in "${!controlplane_ips[@]}"; do
	node_ip="${controlplane_ips[$idx]}"

	join=()
	for other in "${controlplane_ips[@]}"; do
		if [[ "${other}" == "${node_ip}" ]]; then
			continue
		fi
		join+=("${other}")
	done

	join_csv=""
	if ((${#join[@]} > 0)); then
		join_csv="$(IFS=','; echo "${join[*]}")"
	fi

	install_cfg="${WORKDIR}/machineconfig-${idx}-install.yaml"
	runtime_cfg="${WORKDIR}/machineconfig-${idx}-runtime.yaml"

		"${CHUBOCTL}" gen machineconfig \
			--with-secrets "${SECRETS_FILE}" \
			--install-disk "${INSTALL_DISK}" \
			--install-image "${INSTALLER_IMAGE_NODE}" \
		--registry-mirror "${REGISTRY_MIRROR_NODE}" \
		--with-chubo \
		--chubo-role server \
		--chubo-bootstrap-expect "${CONTROLPLANE_COUNT}" \
		${join_csv:+--chubo-join "${join_csv}"} \
		-o "${install_cfg}"

	cp "${install_cfg}" "${runtime_cfg}"
	runtime_tmp="${runtime_cfg}.tmp"
	sed \
		-e 's/^\([[:space:]]*wipe:[[:space:]]*\)true$/\1false/' \
		-e 's|^\([[:space:]]*image:[[:space:]]*\).*$|\1""|' \
		"${runtime_cfg}" >"${runtime_tmp}"
	mv "${runtime_tmp}" "${runtime_cfg}"

	install_cfgs+=("${install_cfg}")
	runtime_cfgs+=("${runtime_cfg}")
done

echo "waiting for maintenance API on all nodes"
for node_ip in "${controlplane_ips[@]}"; do
	wait_for_maintenance "${node_ip}"
done

echo "applying install configs"
for idx in "${!controlplane_ips[@]}"; do
	apply_install_and_wait "${controlplane_ips[$idx]}" "${install_cfgs[$idx]}" "${runtime_cfgs[$idx]}"
done

echo "waiting for openwonton/opengyoza health on all nodes"
for node_ip in "${controlplane_ips[@]}"; do
	wait_for_service_up "${node_ip}" openwonton
	wait_for_service_up "${node_ip}" opengyoza
done

echo "downloading helper bundles for TLS probes"
download_helper_bundles "${controlplane_ips[0]}"

echo "waiting for nomad peers convergence"
wait_for_peers nomad "${CONTROLPLANE_COUNT}" "${controlplane_ips[@]}"

echo "waiting for consul peers convergence"
wait_for_peers consul "${CONTROLPLANE_COUNT}" "${controlplane_ips[@]}"

submit_and_verify_nomad_probe_job "${controlplane_ips[0]}"

echo "leaders:"
for node_ip in "${controlplane_ips[@]}"; do
	nomad_leader="$(curl -fsS --max-time 2 "${NOMAD_CURL_ARGS[@]}" "https://${node_ip}:4646/v1/status/leader" || true)"
	consul_leader="$(curl -fsS --max-time 2 "${CONSUL_CURL_ARGS[@]}" "https://${node_ip}:8500/v1/status/leader" || true)"
	echo "  ${node_ip}: nomad leader=${nomad_leader} consul leader=${consul_leader}"
done

echo "chubo cluster E2E passed"
echo "work dir: ${WORKDIR}"
echo "state dir: ${STATE_DIR}"
