#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Chubo-first defaults for Wave B; chuboos script still does the heavy lifting.
export ARTIFACTS="${ARTIFACTS:-_out/chubo}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
export CLUSTER_NAME="${CLUSTER_NAME:-chubo-e2e-docker}"
export STATE_DIR="${STATE_DIR:-/tmp/chubo-e2e-docker-state}"
export WORKDIR="${WORKDIR:-/tmp/chubo-e2e-docker-work}"
export SUPPORT_OUT="${SUPPORT_OUT:-/tmp/chubo-support-e2e-docker.zip}"
exec "${SCRIPT_DIR}/../chuboos/e2e-core-docker.sh" "$@"
