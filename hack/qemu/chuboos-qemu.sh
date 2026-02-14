#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Legacy wrapper kept for one transition cycle.
# Forward to the chubo-primary script (prefer chubo naming by default).
export ARTIFACTS="${ARTIFACTS:-_out/chubo}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
export SOCKET_VMNET_LOG="${SOCKET_VMNET_LOG:-/tmp/chubo-socket_vmnet.log}"
export RUNDIR_FILE="${RUNDIR_FILE:-/tmp/chubo-qemu.last}"
if [[ -z "${QEMU_RUNDIR:-}" ]]; then
  export QEMU_RUNDIR="${QEMU_RUNDIR:-$(mktemp -d /tmp/chubo-qemu.XXXXXX)}"
fi
exec "${SCRIPT_DIR}/chubo-qemu.sh" "$@"
