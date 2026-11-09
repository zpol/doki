#!/usr/bin/env bash
# Fail CI on unexpected govulncheck findings that reach DOKI call sites.
# Allowlisted IDs: .govulncheck-allowlist
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export PATH="$(go env GOPATH)/bin:${PATH}"

ALLOWLIST="${ROOT}/.govulncheck-allowlist"
RAW="$(mktemp)"
trap 'rm -f "$RAW"' EXIT

if ! command -v govulncheck >/dev/null 2>&1; then
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi

set +e
govulncheck -format=json ./... >"$RAW"
gv_exit=$?
set -e

mapfile -t FINDINGS < <(python3 - "$RAW" <<'PY'
import json, sys

path = sys.argv[1]
data = open(path, encoding="utf-8").read()
decoder = json.JSONDecoder()
idx = 0
ids = set()
while idx < len(data):
    while idx < len(data) and data[idx].isspace():
        idx += 1
    if idx >= len(data):
        break
    obj, end = decoder.raw_decode(data, idx)
    idx = end
    finding = obj.get("finding")
    if not isinstance(finding, dict):
        continue
    osv = finding.get("osv")
    if not isinstance(osv, str) or not osv.startswith("GO-"):
        continue
    trace = finding.get("trace") or []
    # Only gate findings that include a frame from this module.
    if any(frame.get("module") == "github.com/zpol/doki" for frame in trace):
        ids.add(osv)
print("\n".join(sorted(ids)))
PY
)

ALLOWED=()
if [[ -f "$ALLOWLIST" ]]; then
  mapfile -t ALLOWED < <(awk '/^GO-[0-9]+-[0-9]+/{print $1}' "$ALLOWLIST")
fi

unexpected=()
for id in "${FINDINGS[@]+"${FINDINGS[@]}"}"; do
  [[ -z "${id:-}" ]] && continue
  allowed=0
  for a in "${ALLOWED[@]+"${ALLOWED[@]}"}"; do
    if [[ "$id" == "$a" ]]; then
      allowed=1
      break
    fi
  done
  if [[ $allowed -eq 0 ]]; then
    unexpected+=("$id")
  fi
done

if [[ ${#FINDINGS[@]} -gt 0 && -n "${FINDINGS[0]:-}" ]]; then
  echo "govulncheck findings reaching DOKI code:"
  for id in "${FINDINGS[@]}"; do
    tag=""
    for a in "${ALLOWED[@]+"${ALLOWED[@]}"}"; do
      [[ "$id" == "$a" ]] && tag=" [allowlisted]"
    done
    echo "  - ${id}${tag}"
  done
else
  echo "govulncheck: no findings reach DOKI call sites"
fi

if [[ ${#unexpected[@]} -gt 0 ]]; then
  echo
  echo "Unexpected vulnerabilities (not allowlisted):"
  printf '  - %s\n' "${unexpected[@]}"
  echo
  govulncheck ./... || true
  exit 1
fi

if [[ "$gv_exit" -ne 0 && "$gv_exit" -ne 3 ]]; then
  echo "govulncheck failed with exit $gv_exit"
  exit "$gv_exit"
fi

echo "govulncheck OK"
