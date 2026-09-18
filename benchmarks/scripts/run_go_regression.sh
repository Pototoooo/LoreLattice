#!/bin/sh
set -u

# Every package starts without shell contamination. Tests that need loopback
# access now explicitly reset the production sync.Once cache around themselves.
packages=$(go list ./...)

printf 'PHASE=isolated packages=%s\n' "$(printf '%s\n' "$packages" | wc -l | tr -d ' ')"
env -u LOG_FORMAT -u SSRF_WHITELIST -u SSRF_WHITELIST_EXTRA \
  go test -count=1 $packages
default_status=$?

printf 'RESULT suite_exit=%s\n' "$default_status"
if [ "$default_status" -ne 0 ]; then
  exit 1
fi
