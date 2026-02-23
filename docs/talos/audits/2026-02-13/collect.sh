#!/usr/bin/env bash
set -euo pipefail

# Collect baseline "kube/etcd/cri" inventories for the Talos fork and store them
# under this date-stamped directory. This is intentionally kept simple and
# reproducible (no external deps besides rg/go/docker).

audit_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${audit_dir}/../../../../.." && pwd)"

talos_dir="${repo_root}/talos"
chubo_dir="${repo_root}/chubo"

if ! command -v rg >/dev/null 2>&1; then
	echo "missing required command: rg" >&2
	exit 2
fi

if ! command -v go >/dev/null 2>&1; then
	echo "missing required command: go" >&2
	exit 2
fi

pattern='(kubernetes|k8s|kube|etcd|\bcri\b)'

{
	echo "date: $(date -Iseconds)"
	echo "talos_head: $(git -C "${talos_dir}" rev-parse HEAD)"
	echo "chubo_head: $(git -C "${chubo_dir}" rev-parse HEAD)"
	echo "go: $(go version)"
	echo "uname: $(uname -a)"
	echo "rg: $(rg --version | head -n1)"
	echo "docker_context: $(docker context show 2>/dev/null || true)"
	echo "docker_host: ${DOCKER_HOST:-}"
} >"${audit_dir}/meta.txt"

cd "${talos_dir}"

rg -n -i "${pattern}" . >"${audit_dir}/raw-refs.txt" || true

rg -n -i "${pattern}" . \
	--glob '!**/*.md' \
	--glob '!**/docs/**' \
	--glob '!**/website/**' \
	--glob '!**/CHANGELOG.md' \
	--glob '!**/*_test.go' \
	--glob '!**/testdata/**' \
	--glob '!**/hack/test/**' \
	--glob '!**/internal/integration/**' \
	--glob '!**/vendor/**' \
	--glob '!**/*.pb.go' \
	--glob '!**/*_vtproto.pb.go' \
	--glob '!**/*.binpb' \
	>"${audit_dir}/active-refs.txt" || true

rg -n -l '^//go:build.*\bchubo\b' --glob '*.go' . | sort >"${audit_dir}/chubo-go-buildtag-files.txt" || true

if [[ -s "${audit_dir}/chubo-go-buildtag-files.txt" ]]; then
	# ripgrep doesn't support reading file lists directly; use xargs for a stable macOS/Linux path.
	xargs rg -n -i "${pattern}" <"${audit_dir}/chubo-go-buildtag-files.txt" >"${audit_dir}/chubo-go-buildtag-refs.txt" || true
else
	: >"${audit_dir}/chubo-go-buildtag-refs.txt"
fi

tags="tcell_minimal,grpcnotrace,chubo"

GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
	go list -deps -tags "${tags}" ./internal/app/machined | sort -u >"${audit_dir}/deps-machined-linux-arm64.txt"

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go list -deps -tags "${tags}" ./cmd/talosctl | sort -u >"${audit_dir}/deps-talosctl-linux-amd64.txt"

grep -E '^(k8s\\.io/|go\\.etcd\\.io/etcd)' "${audit_dir}/deps-machined-linux-arm64.txt" >"${audit_dir}/deps-machined-forbidden.txt" || true
grep -E '^(k8s\\.io/|go\\.etcd\\.io/etcd)' "${audit_dir}/deps-talosctl-linux-amd64.txt" >"${audit_dir}/deps-talosctl-forbidden.txt" || true

(cd "${audit_dir}" && ls -1) >"${audit_dir}/raw-files.txt"

echo "wrote inventories under: ${audit_dir}"
