#!/usr/bin/env bash
set -euo pipefail

# Verifies we don't introduce new kube/etcd or CRI source references in active
# (non-doc/test/generated/vendor) code paths.
#
# Usage:
#   ./hack/chubo/check-active-refs.sh
#   ./hack/chubo/check-active-refs.sh --update-baseline

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${REPO_ROOT}"

BASELINE_CORE="${BASELINE_CORE:-hack/chubo/active-refs-baseline.txt}"
BASELINE_CRI="${BASELINE_CRI:-hack/chubo/active-cri-refs-baseline.txt}"
COMPAT_PATHS_FILE="${COMPAT_PATHS_FILE:-hack/chubo/active-refs-compat-paths.txt}"

TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/chubo-active-refs.XXXXXX")"
trap 'rm -rf "${TMPDIR}"' EXIT

core_pattern='(kubernetes|k8s|kube|etcd)'
cri_pattern='(\bcri\b)'

scan_core_compat="${TMPDIR}/core.compat.raw"
scan_cri_compat="${TMPDIR}/cri.compat.raw"
core_compat_norm="${TMPDIR}/core.compat.normalized"
cri_compat_norm="${TMPDIR}/cri.compat.normalized"

scan_core_forbidden="${TMPDIR}/core.forbidden.raw"
scan_cri_forbidden="${TMPDIR}/cri.forbidden.raw"
core_forbidden_norm="${TMPDIR}/core.forbidden.normalized"
cri_forbidden_norm="${TMPDIR}/cri.forbidden.normalized"

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
	--glob '!internal/app/machined/pkg/controllers/secrets/data/ca-certificates'
	--glob '!hack/chubo/check-active-refs.sh'
	--glob '!hack/chubo/active-refs-compat-paths.txt'
	--glob '!hack/chubo/active-refs-baseline.txt'
	--glob '!hack/chubo/active-cri-refs-baseline.txt'
)

declare -a compat_paths
declare -a compat_excludes

normalize_refs() {
	# Normalize away line numbers so non-semantic code movement doesn't trigger
	# baseline churn.
	sed -E 's#^([^:]+):[0-9]+:#\1:#' "$1" | sort -u >"$2"
}

trim() {
	local value="$1"
	value="${value#"${value%%[![:space:]]*}"}"
	value="${value%"${value##*[![:space:]]}"}"
	printf '%s' "${value}"
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
	: >"${scan_core_compat}"
	: >"${scan_cri_compat}"

	for path in "${compat_paths[@]}"; do
		if [[ ! -e "${path}" ]]; then
			echo "warning: compat path not found: ${path}" >&2
			continue
		fi

		rg --with-filename -n -i "${core_pattern}" "${path}" "${rg_common_excludes[@]}" >>"${scan_core_compat}" || true
		rg --with-filename -n -i "${cri_pattern}" "${path}" "${rg_common_excludes[@]}" >>"${scan_cri_compat}" || true
	done

	normalize_refs "${scan_core_compat}" "${core_compat_norm}"
	normalize_refs "${scan_cri_compat}" "${cri_compat_norm}"
}

run_forbidden_scan() {
	rg -n -i "${core_pattern}" . "${rg_common_excludes[@]}" "${compat_excludes[@]}" >"${scan_core_forbidden}" || true
	rg -n -i "${cri_pattern}" . "${rg_common_excludes[@]}" "${compat_excludes[@]}" >"${scan_cri_forbidden}" || true

	normalize_refs "${scan_core_forbidden}" "${core_forbidden_norm}"
	normalize_refs "${scan_cri_forbidden}" "${cri_forbidden_norm}"
}

run_scan() {
	load_compat_paths
	build_compat_excludes
	run_compat_scan
	run_forbidden_scan
}

write_baselines() {
	cp "${core_compat_norm}" "${BASELINE_CORE}"
	cp "${cri_compat_norm}" "${BASELINE_CRI}"

	echo "updated baselines:"
	echo "  ${BASELINE_CORE}"
	echo "  ${BASELINE_CRI}"
}

require_baseline() {
	local file="$1"
	if [[ ! -f "${file}" ]]; then
		echo "missing baseline: ${file}" >&2
		echo "run with --update-baseline to initialize it" >&2
		exit 1
	fi
}

check_one() {
	local label="$1"
	local baseline="$2"
	local current="$3"
	local added="${TMPDIR}/${label}.added"
	local removed="${TMPDIR}/${label}.removed"

	comm -13 <(sort -u "${baseline}") <(sort -u "${current}") >"${added}" || true
	comm -23 <(sort -u "${baseline}") <(sort -u "${current}") >"${removed}" || true

	if [[ -s "${added}" ]]; then
		echo "new ${label} references detected:" >&2
		cat "${added}" >&2
		return 1
	fi

	if [[ -s "${removed}" ]]; then
		echo "note: ${label} references were removed (baseline can be refreshed):"
		cat "${removed}"
	fi
}

check_forbidden_empty() {
	local label="$1"
	local file="$2"

	if [[ -s "${file}" ]]; then
		echo "forbidden ${label} references detected outside compat paths:" >&2
		cat "${file}" >&2
		return 1
	fi
}

run_scan

if [[ "${1:-}" == "--update-baseline" ]]; then
	check_forbidden_empty "core" "${core_forbidden_norm}"
	write_baselines
	exit 0
fi

require_baseline "${BASELINE_CORE}"
require_baseline "${BASELINE_CRI}"

check_forbidden_empty "core" "${core_forbidden_norm}"

check_one "compat-core" "${BASELINE_CORE}" "${core_compat_norm}"
check_one "compat-cri" "${BASELINE_CRI}" "${cri_compat_norm}"

echo "active refs guardrail passed"
