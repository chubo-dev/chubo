#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Legacy chuboos wrapper kept for one transition cycle.
# Preserve old defaults while forwarding to chubo-primary script.
export ARTIFACTS="${ARTIFACTS:-_out/chuboos}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chuboos}"
export SOCKET_VMNET_LOG="${SOCKET_VMNET_LOG:-/tmp/chuboos-socket_vmnet.log}"
export RUNDIR_FILE="${RUNDIR_FILE:-/tmp/chuboos-qemu.last}"
if [[ -z "${QEMU_RUNDIR:-}" ]]; then
  export QEMU_RUNDIR="${QEMU_RUNDIR:-$(mktemp -d /tmp/chuboos-qemu.XXXXXX)}"
fi
exec "${SCRIPT_DIR}/chubo-qemu.sh" "$@"
