#!/usr/bin/env bash
set -euo pipefail

# Verifies runtime service and listener surface for chubo.
#
# Usage:
#   hack/chubo/check-runtime-surface.sh \
#     --chuboctl ./_out/chuboctl-darwin-arm64 \
#     --chuboconfig /tmp/chubo-chuboconfig \
#     --endpoint 192.168.0.139 \
#     --node 192.168.0.139
#
# Notes:
# - `--talosctl` is a legacy alias for `--chuboctl` during the rename wave.
# - `--talosconfig` is a legacy alias for `--chuboconfig` during the rename wave.

CHUBOCTL="${CHUBOCTL:-${TALOSCTL:-./_out/chuboctl-darwin-arm64}}"
CHUBOCONFIG=""
ENDPOINT=""
NODE=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--chuboctl)
		CHUBOCTL="$2"
		shift 2
		;;
	--talosctl)
		CHUBOCTL="$2"
		shift 2
		;;
	--chuboconfig)
		CHUBOCONFIG="$2"
		shift 2
		;;
	--talosconfig)
		CHUBOCONFIG="$2"
		shift 2
		;;
	--endpoint)
		ENDPOINT="$2"
		shift 2
		;;
	--node)
		NODE="$2"
		shift 2
		;;
	-h | --help)
		sed -n '1,24p' "$0"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		exit 2
		;;
	esac
done

if [[ -z "${CHUBOCONFIG}" || -z "${ENDPOINT}" || -z "${NODE}" ]]; then
	echo "missing required args: --chuboconfig (or --talosconfig), --endpoint, --node" >&2
	exit 2
fi

if [[ ! -x "${CHUBOCTL}" ]]; then
	echo "chuboctl not executable: ${CHUBOCTL}" >&2
	exit 2
fi

common_args=(--chuboconfig "${CHUBOCONFIG}" -e "${ENDPOINT}" -n "${NODE}")

check_running() {
	local service_id="$1"
	local output

	if ! output="$("${CHUBOCTL}" "${common_args[@]}" service "${service_id}" 2>&1)"; then
		echo "FAIL: service ${service_id} is not queryable" >&2
		echo "${output}" >&2
		exit 1
	fi

	if ! grep -qE '^STATE[[:space:]]+Running$' <<<"${output}"; then
		echo "FAIL: service ${service_id} is not Running" >&2
		echo "${output}" >&2
		exit 1
	fi

	echo "OK: service ${service_id} is Running"
}

check_not_running() {
	local service_id="$1"
	local output

	output="$("${CHUBOCTL}" "${common_args[@]}" service "${service_id}" 2>&1 || true)"

	if grep -qE '^STATE[[:space:]]+Running$' <<<"${output}"; then
		echo "FAIL: forbidden service ${service_id} is Running" >&2
		echo "${output}" >&2
		exit 1
	fi

	echo "OK: forbidden service ${service_id} is not running"
}

echo "Checking runtime services on ${NODE} (endpoint ${ENDPOINT})"

required_services=(
	udevd
	machined
	containerd
	apid
	auditd
	syslogd
)

for svc in "${required_services[@]}"; do
	check_running "${svc}"
done

forbidden_services=(
	dashboard
	$'\153\165\142\145\154\145\164'
	$'\145\164\143\144'
)

for svc in "${forbidden_services[@]}"; do
	check_not_running "${svc}"
done

echo "Checking listener ports"

netstat_output="$("${CHUBOCTL}" "${common_args[@]}" netstat --listening --all)"
listen_lines="$(printf '%s\n' "${netstat_output}" | grep -E 'LISTEN' || true)"

if [[ -z "${listen_lines}" ]]; then
	echo "FAIL: no listening sockets found in netstat output" >&2
	echo "${netstat_output}" >&2
	exit 1
fi

# Runtime OS API listener must be present.
if ! grep -qE '[:.]50000([[:space:]]|$)' <<<"${listen_lines}"; then
	echo "FAIL: expected API listener on port 50000 is missing" >&2
	echo "${listen_lines}" >&2
	exit 1
fi
echo "OK: found API listener on port 50000"

# Legacy cluster-runtime listeners must stay absent in chubo baseline.
forbidden_ports=(2379 2380 6443 10250 10257 10259)

for port in "${forbidden_ports[@]}"; do
	if grep -qE "[:.]${port}([[:space:]]|$)" <<<"${listen_lines}"; then
		echo "FAIL: forbidden listener port ${port} is active" >&2
		echo "${listen_lines}" >&2
		exit 1
	fi
done

echo "OK: no forbidden listener ports detected"
echo "Runtime surface check passed."
