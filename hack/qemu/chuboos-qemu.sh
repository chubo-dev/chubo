#!/usr/bin/env bash
set -euo pipefail

# Quick, no-sudo QEMU loop for the chuboos build-tag variant.
#
# Why this exists:
# - `talosctl cluster create` can require sudo (vmnet/socket_vmnet) on macOS.
# - slirp/usernet + hostfwd works without privileges and is "good enough" for fast iteration.
#
# This script boots Talos via systemd-boot from a FAT "EFI drive" directory and forwards the
# maintenance API to 127.0.0.1:$HOST_PORT (default 50000).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TALOS_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
cd "${TALOS_ROOT}"

ARTIFACTS="${ARTIFACTS:-_out/chuboos}"
GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chuboos}"

QEMU_BIN="${QEMU_BIN:-qemu-system-aarch64}"
QEMU_ACCEL="${QEMU_ACCEL:-hvf}"
EDK2_CODE="${EDK2_CODE:-/opt/homebrew/share/qemu/edk2-aarch64-code.fd}"
EDK2_VARS_TEMPLATE="${EDK2_VARS_TEMPLATE:-/opt/homebrew/share/qemu/edk2-arm-vars.fd}"

HOST_PORT="${HOST_PORT:-50000}"
DISK_SIZE="${DISK_SIZE:-10G}"

# Keep this on by default: after installation completes, the next boot from the "EFI dir" will halt
# and prompt you to boot from the installed disk instead (prevents accidental "still on ISO" loops).
HALT_IF_INSTALLED="${HALT_IF_INSTALLED:-1}"

# Optional extra kernel args, e.g. 'talos.platform=qemu' or debugging flags.
EXTRA_KERNEL_ARGS="${EXTRA_KERNEL_ARGS:-}"

need_build=0
for f in \
  "${ARTIFACTS}/initramfs-arm64.xz" \
  "${ARTIFACTS}/vmlinuz-arm64" \
  "${ARTIFACTS}/sd-boot-arm64.efi"; do
  if [[ ! -f "${f}" ]]; then
    need_build=1
  fi
done

if [[ "${need_build}" -eq 1 ]]; then
  make initramfs kernel sd-boot ARTIFACTS="${ARTIFACTS}" GO_BUILDTAGS="${GO_BUILDTAGS}"
fi

RUNDIR="$(mktemp -d /tmp/chuboos-qemu.XXXXXX)"
EFIDIR="${RUNDIR}/efi"

mkdir -p "${EFIDIR}/EFI/BOOT" "${EFIDIR}/EFI/Linux" "${EFIDIR}/loader/entries"

cp -f "${ARTIFACTS}/sd-boot-arm64.efi" "${EFIDIR}/EFI/BOOT/BOOTAA64.EFI"
cp -f "${ARTIFACTS}/vmlinuz-arm64" "${EFIDIR}/EFI/Linux/vmlinuz.efi"
cp -f "${ARTIFACTS}/initramfs-arm64.xz" "${EFIDIR}/EFI/Linux/initramfs.xz"

cat >"${EFIDIR}/loader/loader.conf" <<EOF
default chuboos.conf
timeout 1
editor 1
EOF

cat >"${EFIDIR}/loader/entries/chuboos.conf" <<EOF
title   ChuboOS (chuboos)
linux   /EFI/Linux/vmlinuz.efi
initrd  /EFI/Linux/initramfs.xz
options talos.platform=metal talos.halt_if_installed=${HALT_IF_INSTALLED} net.ifnames=0 slab_nomerge pti=on talos.dashboard.disabled=1 console=tty0 console=ttyAMA0 printk.devkmsg=on consoleblank=0 ${EXTRA_KERNEL_ARGS}
EOF

cp -f "${EDK2_VARS_TEMPLATE}" "${RUNDIR}/edk2-vars.fd"
qemu-img create -f qcow2 "${RUNDIR}/disk.qcow2" "${DISK_SIZE}" >/dev/null

cat <<EOF
RUNDIR: ${RUNDIR}
EFI dir drive: ${EFIDIR} (appears as /dev/vda inside the VM)
Data disk: ${RUNDIR}/disk.qcow2 (appears as /dev/vdb inside the VM)
Maintenance API: 127.0.0.1:${HOST_PORT} -> guest :50000

Apply config example (maintenance mode):
  talosctl apply-config -i -e 127.0.0.1 -n 127.0.0.1 -f <config.yaml>

NOTE: for this QEMU layout, set machine.install.disk to /dev/vdb (not /dev/vda).
EOF

exec "${QEMU_BIN}" \
  -machine virt,accel="${QEMU_ACCEL}" \
  -cpu host \
  -smp 4 \
  -m 2048 \
  -object rng-random,filename=/dev/urandom,id=rng0 \
  -device virtio-rng-pci,rng=rng0 \
  -drive if=pflash,format=raw,readonly=on,file="${EDK2_CODE}" \
  -drive if=pflash,format=raw,file="${RUNDIR}/edk2-vars.fd" \
  -drive if=virtio,format=raw,file=fat:rw:"${EFIDIR}" \
  -drive if=virtio,format=qcow2,file="${RUNDIR}/disk.qcow2" \
  -device virtio-net-pci,netdev=net0 \
  -netdev user,id=net0,hostfwd=tcp::"${HOST_PORT}"-:50000 \
  -nographic

