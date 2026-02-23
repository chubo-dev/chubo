#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHUBO_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
WORKSPACE_DIR="$(cd "${CHUBO_DIR}/.." && pwd)"
TALOS_DIR="${TALOS_DIR:-${WORKSPACE_DIR}/talos}"

if [[ ! -d "${TALOS_DIR}/api/resource/definitions" ]]; then
	echo "talos repo not found at ${TALOS_DIR}" >&2
	exit 1
fi

OUT_DIR="${SCRIPT_DIR}"

{
	echo "generated_at_utc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
	echo "workspace_dir=${WORKSPACE_DIR}"
	echo "talos_dir=${TALOS_DIR}"
} > "${OUT_DIR}/meta.txt"

pushd "${TALOS_DIR}" >/dev/null

rg -n "^package talos\\.resource\\.definitions\\." api/resource/definitions --glob '*.proto' \
	> "${OUT_DIR}/proto-package-lines.txt"

cut -d: -f1 "${OUT_DIR}/proto-package-lines.txt" | sort -u \
	> "${OUT_DIR}/proto-package-files.txt"

rg -n "talos\\.resource\\.definitions" tools/structprotogen \
	> "${OUT_DIR}/structprotogen-talospackage-refs.txt"

rg -n "talos\\.resource\\.definitions" pkg/machinery/api/resource/definitions \
	--glob '*.pb.go' --glob '*.vtproto.pb.go' \
	> "${OUT_DIR}/generated-stub-talospackage-refs.txt"

rg -n "talos\\.resource\\.definitions" . \
	--glob '!_out/**' \
	--glob '!**/*.pb.go' \
	--glob '!**/*.vtproto.go' \
	--glob '!**/*.md' \
	--glob '!api/resource/definitions/**/*.proto' \
	--glob '!hack/chubo/talos-refs-baseline.txt' \
	> "${OUT_DIR}/non-generated-talospackage-refs.txt"

popd >/dev/null

PROTO_FILE_COUNT="$(wc -l < "${OUT_DIR}/proto-package-files.txt" | tr -d ' ')"
PROTO_LINE_COUNT="$(wc -l < "${OUT_DIR}/proto-package-lines.txt" | tr -d ' ')"
GENERATED_FILE_COUNT="$(cut -d: -f1 "${OUT_DIR}/generated-stub-talospackage-refs.txt" | sort -u | wc -l | tr -d ' ')"
GENERATED_LINE_COUNT="$(wc -l < "${OUT_DIR}/generated-stub-talospackage-refs.txt" | tr -d ' ')"
STRUCTPROTOGEN_LINE_COUNT="$(wc -l < "${OUT_DIR}/structprotogen-talospackage-refs.txt" | tr -d ' ')"
NON_GENERATED_LINE_COUNT="$(wc -l < "${OUT_DIR}/non-generated-talospackage-refs.txt" | tr -d ' ')"

{
	echo "proto_package_files=${PROTO_FILE_COUNT}"
	echo "proto_package_lines=${PROTO_LINE_COUNT}"
	echo "generated_stub_files_with_talos_namespace=${GENERATED_FILE_COUNT}"
	echo "generated_stub_lines_with_talos_namespace=${GENERATED_LINE_COUNT}"
	echo "structprotogen_talos_namespace_lines=${STRUCTPROTOGEN_LINE_COUNT}"
	echo "other_non_generated_talos_namespace_lines=${NON_GENERATED_LINE_COUNT}"
} > "${OUT_DIR}/summary.txt"

echo "wrote proto namespace audit to ${OUT_DIR}"
