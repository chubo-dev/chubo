#!/usr/bin/env bash
set -euo pipefail

# Verifies the `chubo` rootfs doesn't accidentally ship Kubernetes/etcd bits.
#
# This is a developer guardrail for Phase 4 (image composition cleanup).

INITRAMFS_PATH="${1:-_out/chubo/initramfs-arm64.xz}"
TOOLS_IMAGE="${TOOLS_IMAGE:-ghcr.io/siderolabs/tools:v1.13.0-alpha.0-13-gdecb988}"

if [[ "${INITRAMFS_PATH}" != /* ]]; then
  INITRAMFS_PATH="$(pwd)/${INITRAMFS_PATH}"
fi

if [[ ! -f "${INITRAMFS_PATH}" ]]; then
  echo "initramfs not found: ${INITRAMFS_PATH}" >&2
  echo "usage: $0 /path/to/initramfs-<arch>.xz" >&2
  exit 2
fi

for bin in zstd cpio docker; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "missing required command: ${bin}" >&2
    exit 2
  fi
done

# On macOS with colima/lima, the Docker daemon typically runs in a VM and can't
# bind-mount host paths under /tmp (they don't exist inside the VM). Use a
# directory next to the initramfs (usually under /Users, which is shared).
tmp_base="/tmp"
if [[ "$(uname -s)" == "Darwin" ]]; then
  tmp_base="$(dirname "${INITRAMFS_PATH}")"
fi

mkdir -p "${tmp_base}"
tmpdir="$(mktemp -d "${tmp_base}/chubo-rootfs-check.XXXXXX")"
trap 'rm -rf "${tmpdir}"' EXIT

(
  cd "${tmpdir}"
  zstd -dc "${INITRAMFS_PATH}" | cpio -idmu >/dev/null 2>&1
)

if [[ ! -f "${tmpdir}/rootfs.sqsh" ]]; then
  echo "rootfs.sqsh not found in initramfs: ${INITRAMFS_PATH}" >&2
  exit 1
fi

echo "Checking ${INITRAMFS_PATH} (TOOLS_IMAGE=${TOOLS_IMAGE})"

matches="$(docker run --rm -v "${tmpdir}":/work -w /work "${TOOLS_IMAGE}" sh -lc \
  'unsquashfs -l rootfs.sqsh | grep -Ei "kube|kubernetes|etcd|opt/cni/bin/flannel" || true' \
)"

if [[ -n "${matches}" ]]; then
  echo "FAIL: found forbidden paths in rootfs:"
  echo "${matches}"
  exit 1
fi

echo "OK: no kube/etcd/flannel artifacts found in rootfs"

# Extra guardrail: ensure the main OS binary (machined, installed as /usr/bin/init)
# does not link Kubernetes/etcd client libraries in the `chubo` build.
#
# We check Go module build info instead of symbols, since binaries are stripped (-s -w).

init_mods="$(docker run --rm -v "${tmpdir}":/work -w /work "${TOOLS_IMAGE}" sh -lc \
  'unsquashfs -cat rootfs.sqsh usr/bin/init > /tmp/chubo-init && go version -m /tmp/chubo-init' \
)"

forbidden_mods="$(printf '%s\n' "${init_mods}" | grep -E '^dep (k8s\\.io/|go\\.etcd\\.io/)' || true)"

if [[ -n "${forbidden_mods}" ]]; then
  echo "FAIL: init binary links forbidden modules:"
  echo "${forbidden_mods}"
  exit 1
fi

echo "OK: init binary doesn't link k8s.io/go.etcd.io modules"
