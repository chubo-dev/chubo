#!/usr/bin/env bash
set -euo pipefail

# End-to-end opengyoza quorum gate flow in QEMU:
# install -> runtime mTLS -> unsafe quorum check (2 peers blocks graceful upgrade)
# -> safe quorum check (3 peers allows graceful upgrade/reboot)
#
# This script overrides the peers list via a META tag to make the quorum
# scenarios deterministic (no reliance on network namespaces or debug containers).

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
TALOSCTL="${TALOSCTL:-${TALOS_ROOT}/_out/chuboctl-${HOST_GOOS}-${HOST_GOARCH}}"
BUILDX_BUILDER="${BUILDX_BUILDER:-local}"
RUN_ID="${RUN_ID:-$RANDOM}"
BASE_NET_OCTET="${BASE_NET_OCTET:-$((100 + RANDOM % 100))}"
BASE_NET_SUBNET="${BASE_NET_SUBNET:-$((10 + RANDOM % 200))}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-ogy-${RUN_ID}}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-ogy-state-${RUN_ID}}"
WORKDIR="${WORKDIR:-/tmp/chubo-ogy-work-${RUN_ID}}"
NODE_IP="${NODE_IP:-10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.2}"
CIDR="${CIDR:-10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.0/24}"
INSTALL_DISK="${INSTALL_DISK:-/dev/vda}"
CONTROL_PLANE_PORT="${CONTROL_PLANE_PORT:-7443}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-ogy-reg-${RUN_ID}}"
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
CLUSTER_CREATE_MAX_ATTEMPTS="${CLUSTER_CREATE_MAX_ATTEMPTS:-3}"

SECRETS_FILE="${WORKDIR}/secrets.yaml"
MACHINECONFIG_INSTALL="${WORKDIR}/machineconfig-install.yaml"
MACHINECONFIG_RUNTIME="${WORKDIR}/machineconfig-runtime.yaml"
TALOSCONFIG_FILE="${WORKDIR}/talosconfig"
ROLE_PATCH_FILE="${WORKDIR}/opengyoza-role-patch.yaml"
UNSAFE_UPGRADE_OUT="${WORKDIR}/unsafe-upgrade.out"
UNSAFE_MACHINED_LOG="${WORKDIR}/unsafe-machined.log"
CLUSTER_CREATE_LOG="${WORKDIR}/cluster-create.log"

# Keep in sync with meta.ChuboOpenGyozaPeersOverride (talos/pkg/machinery/meta/constants.go).
OPENGYOZA_PEERS_OVERRIDE_META_KEY="${OPENGYOZA_PEERS_OVERRIDE_META_KEY:-18}"

cluster_created=0
registry_started=0
runtime_config_applied=0
CLEANUP_STALE_ONLY=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--skip-build)
		SKIP_BUILD=1
		;;
	--cleanup-stale | --cleanup-stale-only)
		CLEANUP_STALE_ONLY=1
		;;
	-h | --help)
		echo "usage: $0 [--skip-build] [--cleanup-stale]"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		echo "usage: $0 [--skip-build] [--cleanup-stale]" >&2
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
		return 0
	fi

	echo "creating buildx builder: ${BUILDX_BUILDER}"
	docker buildx create --name "${BUILDX_BUILDER}" --driver docker-container >/dev/null
	docker buildx inspect --builder "${BUILDX_BUILDER}" --bootstrap >/dev/null
}

wait_for_maintenance() {
	wait_until "maintenance API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" get addresses --insecure -e "${NODE_IP}" -n "${NODE_IP}"
}

wait_for_runtime() {
	wait_until "runtime mTLS API on ${NODE_IP}" "${TIMEOUT_SECONDS}" \
		"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"
}

service_is_up() {
	local service_name="$1"

	"${TALOSCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" service "${service_name}" 2>/dev/null |
		grep -qi "Health check successful"
}

wait_for_runtime_stable() {
	local consecutive=0
	local required_consecutive=2
	local deadline=$((SECONDS + TIMEOUT_SECONDS))
	local attempts=0

	while true; do
		local has_version=0
		local has_machined=0
		local has_containerd=0

		if "${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1; then
			has_version=1
		fi

		if service_is_up "machined"; then
			has_machined=1
		fi

		if service_is_up "containerd"; then
			has_containerd=1
		fi

		if ((has_version == 1 && has_machined == 1 && has_containerd == 1)); then
			consecutive=$((consecutive + 1))

			if ((consecutive >= required_consecutive)); then
				return 0
			fi
		else
			consecutive=0
		fi

		attempts=$((attempts + 1))
		if ((attempts % 10 == 0)); then
			echo "waiting for stable runtime: version=${has_version} machined=${has_machined} containerd=${has_containerd}"
		fi

		if ((SECONDS >= deadline)); then
			echo "timed out waiting for stable runtime API on ${NODE_IP}" >&2

			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
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
			echo "${context}: no boot ID change observed within ${ACTION_REBOOT_WAIT_SECONDS}s" >&2
			return 1
		fi

		sleep "${SLEEP_SECONDS}"
	done
}

dump_machined_logs() {
	local label="$1"
	local out_path="$2"

	echo "capturing machined logs (${label}) to ${out_path}"

	if wait_until "runtime mTLS API on ${NODE_IP} for log capture" 300 \
		"${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}"; then
		"${TALOSCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" logs machined --tail 4000 >"${out_path}" 2>&1 || true
	else
		echo "runtime mTLS API not available for log capture" >"${out_path}"
	fi
}

pick_control_plane_port() {
	local candidate="${CONTROL_PLANE_PORT}"
	local attempts=0

	while lsof -nP -iTCP:"${candidate}" -sTCP:LISTEN >/dev/null 2>&1; do
		attempts=$((attempts + 1))
		if ((attempts > 20)); then
			echo "failed to find a free control-plane port (starting at ${CONTROL_PLANE_PORT})" >&2
			return 1
		fi

		candidate=$((20000 + RANDOM % 20000))
	done

	if [[ "${candidate}" != "${CONTROL_PLANE_PORT}" ]]; then
		echo "control-plane port ${CONTROL_PLANE_PORT} is busy, switching to ${candidate}"
	fi

	CONTROL_PLANE_PORT="${candidate}"
	return 0
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

	if [[ "${candidate}" != "${REGISTRY_PORT}" ]]; then
		echo "registry port ${REGISTRY_PORT} is busy, switching to ${candidate}"
	fi

	REGISTRY_PORT="${candidate}"
	return 0
}

refresh_registry_refs() {
	: "${REGISTRY_LOCAL_ADDR:=localhost:${REGISTRY_PORT}}"
	: "${REGISTRY_NODE_ADDR:=10.${BASE_NET_OCTET}.${BASE_NET_SUBNET}.1:${REGISTRY_PORT}}"
	: "${INSTALLER_BASE_IMAGE_LOCAL:=${REGISTRY_LOCAL_ADDR}/chubo/installer-base:dev}"
	: "${INSTALLER_IMAGE_LOCAL:=${REGISTRY_LOCAL_ADDR}/chubo/installer:dev}"
	: "${INSTALLER_IMAGE_NODE:=${REGISTRY_NODE_ADDR}/chubo/installer:dev}"
	: "${REGISTRY_MIRROR_NODE:=${REGISTRY_NODE_ADDR}=http://${REGISTRY_NODE_ADDR}}"
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

		return 1
	done
}

cleanup() {
	set +e

	if ((cluster_created == 1)); then
		"${TALOSCTL}" --state "${STATE_DIR}" --name "${CLUSTER_NAME}" cluster destroy \
			--provisioner qemu >/dev/null 2>&1
	fi

	if ((registry_started == 1)); then
		docker rm -f "${REGISTRY_NAME}" >/dev/null 2>&1
	fi
}

trap cleanup EXIT

cleanup_stale_clusters() {
	local state_root

	shopt -s nullglob

	# Prefer a graceful destroy, but also force-kill any orphaned QEMU processes
	# that reference our chubo-ogy state dirs (these can be left behind after aborted runs).
	local stale_qemu_pids=""
	stale_qemu_pids="$(/bin/ps -ax -o pid=,command= | /usr/bin/awk '$0 ~ /qemu-system-/ && index($0, "/tmp/chubo-ogy-state-") { print $1 }' 2>/dev/null || true)"
	if [[ -n "${stale_qemu_pids}" ]]; then
		echo "killing stale QEMU processes: ${stale_qemu_pids}"
		/bin/kill ${stale_qemu_pids} >/dev/null 2>&1 || true
		/bin/sleep 1
		for pid in ${stale_qemu_pids}; do
			/bin/kill -0 "${pid}" >/dev/null 2>&1 || continue
			/bin/kill -9 "${pid}" >/dev/null 2>&1 || true
		done
	fi

	for state_root in /tmp/chubo-ogy-state-*; do
		[[ -d "${state_root}" ]] || continue

		local cluster_dir
		for cluster_dir in "${state_root}"/chubo-ogy-*; do
			[[ -d "${cluster_dir}" ]] || continue

			local cluster_name
			cluster_name="$(basename "${cluster_dir}")"

			echo "destroying stale cluster ${cluster_name} (state dir ${state_root})"
			"${TALOSCTL}" --state "${state_root}" --name "${cluster_name}" cluster destroy --provisioner qemu >/dev/null 2>&1 || true
		done
	done

	for state_root in /tmp/chubo-ogy-state-* /tmp/chubo-ogy-work-*; do
		[[ -e "${state_root}" ]] || continue

		rm -rf "${state_root}"
	done
}

if [[ "${ARCH}" != "amd64" ]]; then
	echo "unsupported ARCH=${ARCH} (only amd64 is supported by this script)" >&2
	exit 2
fi

if [[ "${EUID}" -ne 0 ]]; then
	echo "error: qemu fixture requires root for cluster provisioning; run with \`sudo -E\`" >&2
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

ctl_target="chuboctl-${HOST_GOOS}-${HOST_GOARCH}"
if [[ "${TALOSCTL##*/}" == talosctl-* ]]; then
	ctl_target="talosctl-${HOST_GOOS}-${HOST_GOARCH}"
fi

if [[ ! -x "${TALOSCTL}" ]]; then
	make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
elif ! "${TALOSCTL}" support --help 2>/dev/null | grep -q "Chubo module config snapshots"; then
	if [[ "${SKIP_BUILD}" == "1" ]]; then
		echo "warning: existing CLI binary is not chubo-tagged; continuing due --skip-build"
	else
		echo "existing CLI binary is not chubo-tagged; rebuilding"
		make "${ctl_target}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
	fi
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

# When running under sudo, the root user's Docker config might reference credential helpers
# (e.g. "docker-credential-desktop") which are not available or not usable in CI/automation.
# Use an isolated Docker config for this E2E run to avoid auth helper lookups (we only talk to
# local registries and public images).
export DOCKER_CONFIG="${WORKDIR}/.docker"
mkdir -p "${DOCKER_CONFIG}"
cat >"${DOCKER_CONFIG}/config.json" <<'EOF'
{"auths":{}}
EOF
mkdir -p "${DOCKER_CONFIG}/cli-plugins"
# Talos build targets use `docker buildx build`. If buildx is only installed via Docker Desktop,
# make it available under the isolated DOCKER_CONFIG.
if [[ -x /Applications/Docker.app/Contents/Resources/cli-plugins/docker-buildx ]]; then
	ln -sf /Applications/Docker.app/Contents/Resources/cli-plugins/docker-buildx "${DOCKER_CONFIG}/cli-plugins/docker-buildx"
fi

if ! docker version >/dev/null 2>&1; then
	echo "docker CLI is available but cannot connect to a daemon." >&2
	echo "hint: set DOCKER_HOST before running under sudo (this script uses an isolated DOCKER_CONFIG)." >&2
	echo "hint: to get the host from your current Docker context, run (as your user):" >&2
	echo "  docker context inspect \"$(docker context show)\" --format '{{.Endpoints.docker.Host}}'" >&2
	exit 1
fi

pick_control_plane_port
pick_registry_port
refresh_registry_refs
echo "run configuration:"
echo "  cluster: ${CLUSTER_NAME}"
echo "  state dir: ${STATE_DIR}"
echo "  work dir: ${WORKDIR}"
echo "  cidr: ${CIDR}"
	echo "  node ip: ${NODE_IP}"
	echo "  control-plane port: ${CONTROL_PLANE_PORT}"
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
runtime_tmp="${MACHINECONFIG_RUNTIME}.tmp"
sed \
	-e 's/^\([[:space:]]*wipe:[[:space:]]*\)true$/\1false/' \
	-e 's|^\([[:space:]]*image:[[:space:]]*\).*$|\1""|' \
	"${MACHINECONFIG_RUNTIME}" >"${runtime_tmp}"
mv "${runtime_tmp}" "${MACHINECONFIG_RUNTIME}"

cat >"${ROLE_PATCH_FILE}" <<'EOF'
machine:
  files:
    - op: create
      path: /var/lib/chubo/config/opengyoza.role
      permissions: 0o644
      content: |
        server
EOF

"${TALOSCTL}" machineconfig patch "${MACHINECONFIG_INSTALL}" --patch "@${ROLE_PATCH_FILE}" -o "${MACHINECONFIG_INSTALL}"
"${TALOSCTL}" machineconfig patch "${MACHINECONFIG_RUNTIME}" --patch "@${ROLE_PATCH_FILE}" -o "${MACHINECONFIG_RUNTIME}"

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
	if "${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1; then
		echo "runtime mTLS became available after install apply"
		break
	fi

	if "${TALOSCTL}" get addresses --insecure -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1; then
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

wait_for_runtime_stable
echo "runtime mTLS is stable"

echo "verifying opengyoza role file"
role="$("${TALOSCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" read "${openGyozaRolePath:-/var/lib/chubo/config/opengyoza.role}" 2>/dev/null || true)"
role="$(printf '%s' "${role}" | tr -d '\r\n[:space:]')"
if [[ "${role}" != "server" ]]; then
	echo "expected opengyoza role to be 'server', got ${role:-<missing>}" >&2
	echo "role file read failed or returned unexpected content" >&2
	exit 1
fi

echo "running unsafe quorum scenario (2 peers): graceful upgrade must be blocked"
pre_unsafe_boot_id="$(read_boot_id || true)"
echo "writing opengyoza peers override (unsafe) to META key ${OPENGYOZA_PEERS_OVERRIDE_META_KEY}"
"${TALOSCTL}" meta write "${OPENGYOZA_PEERS_OVERRIDE_META_KEY}" '["peer-a","peer-b"]' \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}"

set +e
"${TALOSCTL}" upgrade \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-i "${INSTALLER_IMAGE_NODE}" \
	--wait \
	--timeout 5m >"${UNSAFE_UPGRADE_OUT}" 2>&1
unsafe_upgrade_rc=$?
set -e

if ((unsafe_upgrade_rc == 0)); then
	echo "expected unsafe quorum upgrade to fail, but it succeeded" >&2
	dump_machined_logs "unsafe-upgrade-succeeded" "${UNSAFE_MACHINED_LOG}" || true
	cat "${UNSAFE_UPGRADE_OUT}" >&2
	exit 1
fi

if ! grep -qi "opengyoza server stop would break quorum" "${UNSAFE_UPGRADE_OUT}"; then
	echo "unsafe quorum failure did not include expected reason" >&2
	dump_machined_logs "unsafe-upgrade-missing-reason" "${UNSAFE_MACHINED_LOG}" || true
	cat "${UNSAFE_UPGRADE_OUT}" >&2
	exit 1
fi

post_unsafe_boot_id="$(read_boot_id || true)"
if [[ -n "${pre_unsafe_boot_id}" && -n "${post_unsafe_boot_id}" && "${pre_unsafe_boot_id}" != "${post_unsafe_boot_id}" ]]; then
	echo "unsafe quorum scenario unexpectedly rebooted node (${pre_unsafe_boot_id} -> ${post_unsafe_boot_id})" >&2
	exit 1
fi

wait_for_runtime_stable
echo "unsafe quorum scenario passed (blocked without reboot)"

echo "running safe quorum scenario (3 peers): graceful upgrade must proceed"
pre_safe_boot_id="$(read_boot_id || true)"
if [[ -z "${pre_safe_boot_id}" ]]; then
	echo "failed to read boot ID before safe quorum scenario" >&2
	exit 1
fi

echo "writing opengyoza peers override (safe) to META key ${OPENGYOZA_PEERS_OVERRIDE_META_KEY}"
"${TALOSCTL}" meta write "${OPENGYOZA_PEERS_OVERRIDE_META_KEY}" '["peer-a","peer-b","peer-c"]' \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}"

"${TALOSCTL}" upgrade \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-i "${INSTALLER_IMAGE_NODE}" \
	--wait=false

wait_for_boot_id_change "${pre_safe_boot_id}" "safe quorum upgrade"
wait_for_runtime_stable

echo "clearing opengyoza peers override META key ${OPENGYOZA_PEERS_OVERRIDE_META_KEY}"
"${TALOSCTL}" meta delete "${OPENGYOZA_PEERS_OVERRIDE_META_KEY}" \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" || true

echo "opengyoza quorum gate E2E passed"
