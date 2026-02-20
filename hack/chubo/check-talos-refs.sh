#!/usr/bin/env bash
set -euo pipefail

# Verifies Talos/Talosctl naming stays confined to explicit compatibility paths.
#
# Usage:
#   ./hack/chubo/check-talos-refs.sh
#   ./hack/chubo/check-talos-refs.sh --update-baseline
#
# In normal mode:
#   - fails if any talos/talosctl references exist outside compat paths
#   - fails if new talos/talosctl references are added inside compat paths
#
# In update mode:
#   - requires zero references outside compat paths
#   - refreshes the compat baseline with current compat references

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${REPO_ROOT}"

BASELINE_FILE="${BASELINE_FILE:-hack/chubo/talos-refs-baseline.txt}"
COMPAT_PATHS_FILE="${COMPAT_PATHS_FILE:-hack/chubo/talos-refs-compat-paths.txt}"

TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/chubo-talos-refs.XXXXXX")"
trap 'rm -rf "${TMPDIR}"' EXIT

pattern='(\btalosctl\b|\btalos\b)'

scan_compat_raw="${TMPDIR}/compat.raw"
compat_norm="${TMPDIR}/compat.normalized"
scan_forbidden_raw="${TMPDIR}/forbidden.raw"
forbidden_norm="${TMPDIR}/forbidden.normalized"

rg_common_excludes=(
	--glob '!**/*.md'
	--glob '!**/docs/**'
	--glob '!**/website/**'
	--glob '!**/CHANGELOG.md'
	--glob '!**/*_test.go'
	--glob '!**/testdata/**'
	--glob '!**/hack/test/**'
	--glob '!**/internal/integration/**'
	--glob '!**/vendor/**'
	--glob '!**/*.pb.go'
	--glob '!**/*_vtproto.pb.go'
	--glob '!**/*.binpb'
	--glob '!**/go.sum'
	--glob '!tools/go.mod'
	--glob '!tools/go.sum'
	--glob '!hack/chubo/check-talos-refs.sh'
	--glob '!hack/chubo/talos-refs-compat-paths.txt'
	--glob '!hack/chubo/talos-refs-baseline.txt'
)

declare -a compat_paths
declare -a compat_excludes

trim() {
	local value="$1"
	value="${value#"${value%%[![:space:]]*}"}"
	value="${value%"${value##*[![:space:]]}"}"
	printf '%s' "${value}"
}

normalize_refs() {
	# Remove line numbers so baseline is resilient to non-semantic movement.
	sed -E 's#^([^:]+):[0-9]+:#\1:#' "$1" | sort -u >"$2"
}

load_compat_paths() {
	if [[ ! -f "${COMPAT_PATHS_FILE}" ]]; then
		echo "missing compat paths file: ${COMPAT_PATHS_FILE}" >&2
		exit 1
	fi

	while IFS= read -r raw_line || [[ -n "${raw_line}" ]]; do
		local line
		line="$(trim "${raw_line%%#*}")"

		if [[ -z "${line}" ]]; then
			continue
		fi

		compat_paths+=("${line}")
	done <"${COMPAT_PATHS_FILE}"

	if [[ "${#compat_paths[@]}" -eq 0 ]]; then
		echo "no compat paths configured in ${COMPAT_PATHS_FILE}" >&2
		exit 1
	fi
}

build_compat_excludes() {
	for path in "${compat_paths[@]}"; do
		local normalized="${path#./}"

		if [[ -d "${normalized}" ]]; then
			compat_excludes+=(--glob "!${normalized}/**")
		else
			compat_excludes+=(--glob "!${normalized}")
		fi
	done
}

run_compat_scan() {
	: >"${scan_compat_raw}"

	for path in "${compat_paths[@]}"; do
		if [[ ! -e "${path}" ]]; then
			echo "warning: compat path not found: ${path}" >&2
			continue
		fi

		rg --with-filename -n -i "${pattern}" "${path}" "${rg_common_excludes[@]}" >>"${scan_compat_raw}" || true
	done

	normalize_refs "${scan_compat_raw}" "${compat_norm}"
}

run_forbidden_scan() {
	rg -n -i "${pattern}" . "${rg_common_excludes[@]}" "${compat_excludes[@]}" >"${scan_forbidden_raw}" || true
	normalize_refs "${scan_forbidden_raw}" "${forbidden_norm}"
}

run_scan() {
	load_compat_paths
	build_compat_excludes
	run_compat_scan
	run_forbidden_scan
}

require_baseline() {
	if [[ ! -f "${BASELINE_FILE}" ]]; then
		echo "missing baseline: ${BASELINE_FILE}" >&2
		echo "run with --update-baseline to initialize it" >&2
		exit 1
	fi
}

check_forbidden_empty() {
	if [[ -s "${forbidden_norm}" ]]; then
		echo "forbidden talos/talosctl references detected outside compat paths:" >&2
		cat "${forbidden_norm}" >&2
		return 1
	fi
}

check_baseline_delta() {
	local added="${TMPDIR}/compat.added"
	local removed="${TMPDIR}/compat.removed"

	comm -13 <(sort -u "${BASELINE_FILE}") <(sort -u "${compat_norm}") >"${added}" || true
	comm -23 <(sort -u "${BASELINE_FILE}") <(sort -u "${compat_norm}") >"${removed}" || true

	if [[ -s "${added}" ]]; then
		echo "new talos/talosctl references detected in compat paths:" >&2
		cat "${added}" >&2
		return 1
	fi

	if [[ -s "${removed}" ]]; then
		echo "note: talos/talosctl references were removed (baseline can be refreshed):"
		cat "${removed}"
	fi
}

write_baseline() {
	cp "${compat_norm}" "${BASELINE_FILE}"
	echo "updated baseline: ${BASELINE_FILE}"
}

run_scan

if [[ "${1:-}" == "--update-baseline" ]]; then
	check_forbidden_empty
	write_baseline
	exit 0
fi

require_baseline
check_forbidden_empty
check_baseline_delta

echo "talos refs guardrail passed"
