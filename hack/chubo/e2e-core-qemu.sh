#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Chubo-first defaults for Wave B; chuboos script still does the heavy lifting.
export ARTIFACTS="${ARTIFACTS:-_out/chubo}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
export CLUSTER_NAME="${CLUSTER_NAME:-chubo-e2e}"
export STATE_DIR="${STATE_DIR:-/tmp/chubo-e2e-state}"
export WORKDIR="${WORKDIR:-/tmp/chubo-e2e-work}"
export REGISTRY_NAME="${REGISTRY_NAME:-chubo-e2e-registry}"
export SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chubo-support-e2e.zip}"
export CLUSTER_LOGS_OUT="${CLUSTER_LOGS_OUT:-/tmp/logs-chubo-e2e.tar.gz}"
export CLUSTER_SUPPORT_OUT="${CLUSTER_SUPPORT_OUT:-/tmp/support-chubo-e2e.zip}"
exec "${SCRIPT_DIR}/../chuboos/e2e-core-qemu.sh" "$@"
