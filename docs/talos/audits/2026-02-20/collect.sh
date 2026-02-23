#!/usr/bin/env bash
set -euo pipefail

audit_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${audit_dir}/../../../../.." && pwd)"
talos_dir="${repo_root}/talos"

{
  echo "date: $(date -Iseconds)"
  echo "talos_head: $(git -C "${talos_dir}" rev-parse HEAD)"
  echo "go: $(go version)"
  echo "uname: $(uname -a)"
  echo "rg: $(rg --version | head -n1)"
} >"${audit_dir}/meta.txt"

cd "${talos_dir}"

common=(
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
)

rg -n -i '\btalosctl\b' . "${common[@]}" | sort >"${audit_dir}/active-talosctl-refs.txt" || true
rg -n -i '\btalos\b' . "${common[@]}" | sort >"${audit_dir}/active-talos-refs.txt" || true

AUDIT_DIR="${audit_dir}" python3 - <<'PY' >"${audit_dir}/summary.txt"
from collections import Counter
import os
from pathlib import Path
base = Path(os.environ["AUDIT_DIR"])
for name in ("active-talosctl-refs.txt", "active-talos-refs.txt"):
    c = Counter()
    lines = 0
    with open(base / name) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            lines += 1
            p = line.split(':', 1)[0]
            if p.startswith("./"):
                p = p[2:]
            top = p.split('/', 1)[0]
            c[top] += 1
    print(f"{name}: {lines} matches")
    for k, v in c.most_common(12):
        print(f"  {k}: {v}")
    print()
PY

(cp hack/chubo/talos-refs-compat-paths.txt "${audit_dir}/talos-refs-compat-paths.txt")
(cp hack/chubo/talos-refs-baseline.txt "${audit_dir}/talos-refs-baseline.txt")

(cd "${audit_dir}" && ls -1) >"${audit_dir}/raw-files.txt"

echo "wrote inventories under: ${audit_dir}"
