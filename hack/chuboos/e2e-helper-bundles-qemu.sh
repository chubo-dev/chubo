#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Legacy wrapper kept for one transition cycle: forward to the chubo-primary script.
export ARTIFACTS="${ARTIFACTS:-_out/chubo}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"

exec "${SCRIPT_DIR}/../chubo/e2e-helper-bundles-qemu.sh" "$@"
