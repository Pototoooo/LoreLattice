#!/bin/sh
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
baseline="$script_dir/.baseline/LoreLattice_USER_RESEARCH_REHEARSAL.md"
target="${1:-$script_dir/LoreLattice_USER_RESEARCH_REHEARSAL.md}"
cp "$baseline" "$target"
hash=$(shasum -a 256 "$target" | awk '{print $1}')
printf 'ROLLBACK result=PASS restored=%s sha256=%s state=baseline_stub\n' "$target" "$hash"
