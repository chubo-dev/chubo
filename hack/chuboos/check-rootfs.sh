#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ $# -eq 0 ]]; then
	set -- "_out/chuboos/initramfs-arm64.xz"
fi

exec "${SCRIPT_DIR}/../chubo/check-rootfs.sh" "$@"
