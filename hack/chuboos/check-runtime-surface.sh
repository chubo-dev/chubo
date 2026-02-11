#!/usr/bin/env bash
set -euo pipefail

# Verifies runtime service and listener surface for chuboos.
#
# Usage:
#   hack/chuboos/check-runtime-surface.sh \
#     --talosconfig /tmp/chuboos-talosconfig \
#     --endpoint 192.168.0.139 \
#     --node 192.168.0.139

TALOSCTL="${TALOSCTL:-./_out/talosctl-darwin-arm64}"
TALOSCONFIG=""
ENDPOINT=""
NODE=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--talosctl)
		TALOSCTL="$2"
		shift 2
		;;
	--talosconfig)
		TALOSCONFIG="$2"
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

if [[ -z "${TALOSCONFIG}" || -z "${ENDPOINT}" || -z "${NODE}" ]]; then
	echo "missing required args: --talosconfig, --endpoint, --node" >&2
	exit 2
fi

if [[ ! -x "${TALOSCTL}" ]]; then
	echo "talosctl not executable: ${TALOSCTL}" >&2
	exit 2
fi

common_args=(--talosconfig "${TALOSCONFIG}" -e "${ENDPOINT}" -n "${NODE}")

check_running() {
	local service_id="$1"
	local output

	if ! output="$("${TALOSCTL}" "${common_args[@]}" service "${service_id}" 2>&1)"; then
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

	output="$("${TALOSCTL}" "${common_args[@]}" service "${service_id}" 2>&1 || true)"

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
	ext-chubo-agent
)

for svc in "${required_services[@]}"; do
	check_running "${svc}"
done

forbidden_services=(
	dashboard
	kubelet
	etcd
	cri
)

for svc in "${forbidden_services[@]}"; do
	check_not_running "${svc}"
done

echo "Checking listener ports"

netstat_output="$("${TALOSCTL}" "${common_args[@]}" netstat --listening --all)"
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

# Kubernetes/etcd listeners must stay absent in chuboos baseline.
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
