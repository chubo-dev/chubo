#!/usr/bin/env bash
set -euo pipefail

# Verifies the `chuboos` rootfs doesn't accidentally ship Kubernetes/etcd bits.
#
# This is a developer guardrail for Phase 4 (image composition cleanup).

INITRAMFS_PATH="${1:-_out/chuboos/initramfs-arm64.xz}"
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

tmpdir="$(mktemp -d /tmp/chuboos-rootfs-check.XXXXXX)"
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
