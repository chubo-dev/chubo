#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Legacy chuboos defaults preserved for compatibility.
export ARTIFACTS="${ARTIFACTS:-_out/chuboos}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chuboos}"
export CLUSTER_NAME="${CLUSTER_NAME:-chuboos-e2e-docker}"
export STATE_DIR="${STATE_DIR:-/tmp/chuboos-e2e-docker-state}"
export WORKDIR="${WORKDIR:-/tmp/chuboos-e2e-docker-work}"
export SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chuboos-support-e2e-docker.zip}"
exec "${SCRIPT_DIR}/../chubo/e2e-core-docker.sh" "$@"
