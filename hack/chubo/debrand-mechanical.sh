#!/usr/bin/env bash
set -euo pipefail

# Applies low-risk comment-only debranding rewrites.
#
# Scope:
# - single-line comments: // ...
# - block comments: /* ... */
#
# Rewrites:
# - legacy product name -> Chubo
# - legacy CLI name -> chuboctl
#
# Usage:
#   ./hack/chubo/debrand-mechanical.sh path/to/file.go [...]

if [[ "$#" -lt 1 ]]; then
	echo "usage: $0 <go-file> [<go-file> ...]" >&2
	exit 1
fi

python3 - "$@" <<'PY'
import re
import sys
from pathlib import Path

single_line = re.compile(r"(^[ \t]*//.*$)", re.M)
block = re.compile(r"/\*.*?\*/", re.S)

def rewrite_comment_text(text: str) -> str:
	# Preserve explicitly historical compatibility references.
	legacy_marker = "legacy " + "T" + "alos"
	previous_marker = "previous " + "T" + "alos versions"
	if legacy_marker in text or previous_marker in text:
		return re.sub(r"\btalosctl\b", "chuboctl", text)

	text = re.sub(r"\bTalos\b", "Chubo", text)
	text = re.sub(r"\btalosctl\b", "chuboctl", text)
	return text

def rewrite_comments(content: str) -> str:
	content = single_line.sub(lambda m: rewrite_comment_text(m.group(0)), content)
	content = block.sub(lambda m: rewrite_comment_text(m.group(0)), content)
	return content

for raw in sys.argv[1:]:
	path = Path(raw)
	if path.suffix != ".go":
		continue

	original = path.read_text()
	updated = rewrite_comments(original)

	if updated != original:
		path.write_text(updated)
		print(path)
PY
