#!/usr/bin/env bash
set -euo pipefail

# End-to-end opengyoza quorum gate flow in QEMU:
# install -> runtime mTLS -> unsafe quorum check (2 peers blocks graceful upgrade)
# -> safe quorum check (3 peers allows graceful upgrade/reboot)
#
# This script uses a one-shot local debug container on the node to mock
# opengyoza `/v1/status/peers` responses.

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
TALOSCTL="${TALOSCTL:-${TALOS_ROOT}/_out/talosctl-${HOST_GOOS}-${HOST_GOARCH}}"

CLUSTER_NAME="${CLUSTER_NAME:-chubo-opengyoza-quorum-e2e}"
STATE_DIR="${STATE_DIR:-/tmp/chubo-opengyoza-quorum-e2e-state}"
WORKDIR="${WORKDIR:-/tmp/chubo-opengyoza-quorum-e2e-work}"
NODE_IP="${NODE_IP:-10.5.0.2}"
CIDR="${CIDR:-10.5.0.0/24}"
INSTALL_DISK="${INSTALL_DISK:-/dev/vda}"

REGISTRY_NAME="${REGISTRY_NAME:-chubo-opengyoza-quorum-e2e-registry}"
REGISTRY_PORT="${REGISTRY_PORT:-5001}"
REGISTRY_LOCAL_ADDR="${REGISTRY_LOCAL_ADDR:-localhost:${REGISTRY_PORT}}"
REGISTRY_NODE_ADDR="${REGISTRY_NODE_ADDR:-10.5.0.1:${REGISTRY_PORT}}"
INSTALLER_BASE_IMAGE_LOCAL="${INSTALLER_BASE_IMAGE_LOCAL:-${REGISTRY_LOCAL_ADDR}/chubo/installer-base:dev}"
INSTALLER_IMAGE_LOCAL="${INSTALLER_IMAGE_LOCAL:-${REGISTRY_LOCAL_ADDR}/chubo/installer:dev}"
INSTALLER_IMAGE_NODE="${INSTALLER_IMAGE_NODE:-${REGISTRY_NODE_ADDR}/chubo/installer:dev}"
REGISTRY_MIRROR_NODE="${REGISTRY_MIRROR_NODE:-${REGISTRY_NODE_ADDR}=http://${REGISTRY_NODE_ADDR}}"

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
MOCK_IMAGE_TAR="${WORKDIR}/opengyoza-peers-mock.tar"
UNSAFE_UPGRADE_OUT="${WORKDIR}/unsafe-upgrade.out"
SAFE_MOCK_LOG="${WORKDIR}/safe-mock.log"
UNSAFE_MOCK_LOG="${WORKDIR}/unsafe-mock.log"
CLUSTER_CREATE_LOG="${WORKDIR}/cluster-create.log"

cluster_created=0
registry_started=0
runtime_config_applied=0

while [[ $# -gt 0 ]]; do
	case "$1" in
	--skip-build)
		SKIP_BUILD=1
		;;
	-h | --help)
		echo "usage: $0 [--skip-build]"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		echo "usage: $0 [--skip-build]" >&2
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
		if "${TALOSCTL}" version --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" >/dev/null 2>&1 &&
			"${TALOSCTL}" --talosconfig "${TALOSCONFIG_FILE}" -e "${NODE_IP}" -n "${NODE_IP}" service machined >/dev/null 2>&1; then
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

wait_for_process_exit() {
	local pid="$1"
	local timeout_seconds="$2"
	local description="$3"
	local deadline=$((SECONDS + timeout_seconds))

	while kill -0 "${pid}" >/dev/null 2>&1; do
		if ((SECONDS >= deadline)); then
			echo "timed out waiting for process ${pid}: ${description}" >&2
			return 1
		fi

		sleep 1
	done

	wait "${pid}"
}

run_cluster_create() {
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

start_peers_mock() {
	local peers_json="$1"
	local log_file="$2"

	"${TALOSCTL}" debug "${MOCK_IMAGE_TAR}" \
		--namespace system \
		--talosconfig "${TALOSCONFIG_FILE}" \
		-e "${NODE_IP}" -n "${NODE_IP}" \
		--args "${peers_json}" >"${log_file}" 2>&1 &
	echo $!
}

build_mock_image_tar() {
	local mock_dir="${WORKDIR}/opengyoza-peers-mock"

	rm -rf "${mock_dir}"
	mkdir -p "${mock_dir}"

	cat >"${mock_dir}/main.go" <<'EOF'
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	peers := `["peer-a","peer-b"]`
	if len(os.Args) > 1 {
		peers = os.Args[1]
	}

	ln, err := net.Listen("tcp", "127.0.0.1:8500")
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	conn, err := ln.Accept()
	if err != nil {
		log.Fatalf("accept failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	_, _ = io.CopyN(io.Discard, conn, 4096)

	resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(peers), peers)
	if _, err := conn.Write([]byte(resp)); err != nil {
		log.Fatalf("write failed: %v", err)
	}
}
EOF

	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${mock_dir}/opengyoza-peers-mock" "${mock_dir}/main.go"

	cat >"${mock_dir}/Dockerfile" <<'EOF'
FROM scratch
COPY opengyoza-peers-mock /opengyoza-peers-mock
ENTRYPOINT ["/opengyoza-peers-mock"]
EOF

	local image_ref="chubo/opengyoza-peers-mock:dev"
	docker build -t "${image_ref}" "${mock_dir}" >/dev/null
	docker save "${image_ref}" -o "${MOCK_IMAGE_TAR}"
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

if [[ "${ARCH}" != "amd64" ]]; then
	echo "unsupported ARCH=${ARCH} (only amd64 is supported by this script)" >&2
	exit 2
fi

if [[ "${EUID}" -ne 0 ]]; then
	echo "error: please run as root user (CNI, qemu hvf requirement), we recommend running with \`sudo -E\`" >&2
	exit 1
fi

require_cmd docker
require_cmd go
require_cmd make

if [[ ! -x "${TALOSCTL}" ]]; then
	make "talosctl-${HOST_GOOS}-${HOST_GOARCH}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
elif ! "${TALOSCTL}" support --help 2>/dev/null | grep -q "Chubo module config snapshots"; then
	echo "existing talosctl binary is not chubo-tagged; rebuilding"
	make "talosctl-${HOST_GOOS}-${HOST_GOARCH}" GO_BUILDFLAGS_TALOSCTL="${GO_BUILDFLAGS_TALOSCTL}"
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

echo "building one-shot opengyoza peers mock image"
build_mock_image_tar

echo "running unsafe quorum scenario (2 peers): graceful upgrade must be blocked"
pre_unsafe_boot_id="$(read_boot_id || true)"
unsafe_mock_pid="$(start_peers_mock '["peer-a","peer-b"]' "${UNSAFE_MOCK_LOG}")"

set +e
"${TALOSCTL}" upgrade \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-i "${INSTALLER_IMAGE_NODE}" \
	--wait \
	--timeout 5m >"${UNSAFE_UPGRADE_OUT}" 2>&1
unsafe_upgrade_rc=$?
set -e

wait_for_process_exit "${unsafe_mock_pid}" 30 "unsafe quorum mock request"

if ((unsafe_upgrade_rc == 0)); then
	echo "expected unsafe quorum upgrade to fail, but it succeeded" >&2
	cat "${UNSAFE_UPGRADE_OUT}" >&2
	exit 1
fi

if ! grep -qi "opengyoza server stop would break quorum" "${UNSAFE_UPGRADE_OUT}"; then
	echo "unsafe quorum failure did not include expected reason" >&2
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

safe_mock_pid="$(start_peers_mock '["peer-a","peer-b","peer-c"]' "${SAFE_MOCK_LOG}")"

"${TALOSCTL}" upgrade \
	--talosconfig "${TALOSCONFIG_FILE}" \
	-e "${NODE_IP}" -n "${NODE_IP}" \
	-i "${INSTALLER_IMAGE_NODE}" \
	--wait=false

wait_for_process_exit "${safe_mock_pid}" 30 "safe quorum mock request"
wait_for_boot_id_change "${pre_safe_boot_id}" "safe quorum upgrade"
wait_for_runtime_stable

echo "opengyoza quorum gate E2E passed"
