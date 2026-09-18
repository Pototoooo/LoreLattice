#!/bin/sh
set -eu
HERE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TARGET=${1:?Usage: ROLLBACK.sh TARGET_COPY_OF_evaluation.go}
cp "$HERE/artifacts/evaluation.go.original" "$TARGET"
cmp -s "$HERE/artifacts/evaluation.go.original" "$TARGET"
echo 'ROLLBACK_HASH_MATCH=YES'
