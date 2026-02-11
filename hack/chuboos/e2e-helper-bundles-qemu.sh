#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Legacy chuboos defaults preserved for compatibility.
export ARTIFACTS="${ARTIFACTS:-_out/chuboos}"
export GO_BUILDTAGS="${GO_BUILDTAGS:-tcell_minimal,grpcnotrace,chuboos}"
export REGISTRY_NAME="${REGISTRY_NAME:-chuboos-helper-registry}"
export RUN_DIR="${RUN_DIR:-$(mktemp -d /tmp/chuboos-helper-e2e.XXXXXX)}"

if [[ -z "${TALOSCTL_CHUBO:-}" ]]; then
	HOST_GOOS="${HOST_GOOS:-$(go env GOOS)}"
	HOST_GOARCH="${HOST_GOARCH:-$(go env GOARCH)}"
	export TALOSCTL_CHUBO="${TALOSCTL_CHUBO:-${SCRIPT_DIR}/../../_out/chuboos/talosctl-${HOST_GOOS}-${HOST_GOARCH}}"
fi

exec "${SCRIPT_DIR}/../chubo/e2e-helper-bundles-qemu.sh" "$@"
