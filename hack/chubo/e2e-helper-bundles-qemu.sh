#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Chubo-first defaults for Wave B; chuboos script still does the heavy lifting.
export ARTIFACTS="${ARTIFACTS:-_out/chubo}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chubo}"
export REGISTRY_NAME="${REGISTRY_NAME:-chubo-helper-registry}"
export RUN_DIR="${RUN_DIR:-$(mktemp -d /tmp/chubo-helper-e2e.XXXXXX)}"

if [[ -z "${TALOSCTL_CHUBO:-}" ]]; then
	HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
	HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"
	export TALOSCTL_CHUBO="${TALOSCTL_CHUBO:-${SCRIPT_DIR}/../../_out/chubo/talosctl-${HOST_GOOS}-${HOST_GOARCH}}"
fi

exec "${SCRIPT_DIR}/../chuboos/e2e-helper-bundles-qemu.sh" "$@"
