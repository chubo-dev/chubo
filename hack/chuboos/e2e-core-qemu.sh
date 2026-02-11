#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Legacy chuboos defaults preserved for compatibility.
export ARTIFACTS="${ARTIFACTS:-_out/chuboos}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chuboos}"
export CLUSTER_NAME="${CLUSTER_NAME:-chuboos-e2e}"
export STATE_DIR="${STATE_DIR:-/tmp/chuboos-e2e-state}"
export WORKDIR="${WORKDIR:-/tmp/chuboos-e2e-work}"
export REGISTRY_NAME="${REGISTRY_NAME:-chuboos-e2e-registry}"
export SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chuboos-support-e2e.zip}"
export CLUSTER_LOGS_OUT="${CLUSTER_LOGS_OUT:-/tmp/logs-chuboos-e2e.tar.gz}"
export CLUSTER_SUPPORT_OUT="${CLUSTER_SUPPORT_OUT:-/tmp/support-chuboos-e2e.zip}"
exec "${SCRIPT_DIR}/../chubo/e2e-core-qemu.sh" "$@"
