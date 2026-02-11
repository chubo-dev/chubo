#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Chubo-first defaults for Wave B; chuboos script remains as compatibility backend.
export ARTIFACTS="${ARTIFACTS:-_out/chubo}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
export RUNDIR_FILE="${RUNDIR_FILE:-/tmp/chubo-qemu.last}"
export SOCKET_VMNET_LOG="${SOCKET_VMNET_LOG:-/tmp/chubo-socket_vmnet.log}"
exec "${SCRIPT_DIR}/chuboos-qemu.sh" "$@"
