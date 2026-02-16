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

TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/chubo-active-refs.XXXXXX")"
trap 'rm -rf "${TMPDIR}"' EXIT

core_pattern='(kubernetes|k8s|kube|etcd)'
cri_pattern='(\bcri\b)'

scan_core="${TMPDIR}/core.raw"
scan_cri="${TMPDIR}/cri.raw"
core_norm="${TMPDIR}/core.normalized"
cri_norm="${TMPDIR}/cri.normalized"

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
	--glob '!hack/chubo/check-active-refs.sh'
	--glob '!hack/chubo/active-refs-baseline.txt'
	--glob '!hack/chubo/active-cri-refs-baseline.txt'
)

normalize_refs() {
	# Normalize away line numbers so non-semantic code movement doesn't trigger
	# baseline churn.
	sed -E 's#^([^:]+):[0-9]+:#\1:#' "$1" | sort -u >"$2"
}

run_scan() {
	rg -n -i "${core_pattern}" . "${rg_common_excludes[@]}" >"${scan_core}" || true
	rg -n -i "${cri_pattern}" . "${rg_common_excludes[@]}" >"${scan_cri}" || true

	normalize_refs "${scan_core}" "${core_norm}"
	normalize_refs "${scan_cri}" "${cri_norm}"
}

write_baselines() {
	cp "${core_norm}" "${BASELINE_CORE}"
	cp "${cri_norm}" "${BASELINE_CRI}"

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

run_scan

if [[ "${1:-}" == "--update-baseline" ]]; then
	write_baselines
	exit 0
fi

require_baseline "${BASELINE_CORE}"
require_baseline "${BASELINE_CRI}"

check_one "core" "${BASELINE_CORE}" "${core_norm}"
check_one "cri" "${BASELINE_CRI}" "${cri_norm}"

echo "active refs guardrail passed"
