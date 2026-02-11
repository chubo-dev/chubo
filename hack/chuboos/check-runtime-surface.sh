#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -z "${TALOSCTL:-}" ]] && [[ -x "${SCRIPT_DIR}/../../_out/talosctl-$(go env GOOS)-$(go env GOARCH)" ]]; then
	export TALOSCTL="${SCRIPT_DIR}/../../_out/talosctl-$(go env GOOS)-$(go env GOARCH)"
fi

exec "${SCRIPT_DIR}/../chubo/check-runtime-surface.sh" "$@"
