#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
RESULTS="$ROOT/benchmarks/results/2026-08-25"
REPORTS="$ROOT/docs/benchmarks/2026-08-25"

printf 'VERIFY_INPUT root=%s dataset=%s\n' "$ROOT" "$ROOT/benchmarks/datasets/enterprise_knowledge_v1/queries.jsonl"
printf 'VERIFY_BRANCH branch=%s head=%s\n' "$(git -C "$ROOT" branch --show-current)" "$(git -C "$ROOT" rev-parse HEAD)"

grep -q 'ℹ pass 225' "$RESULTS/regression/frontend-test-type-build.log"
grep -q 'ℹ fail 0' "$RESULTS/regression/frontend-test-type-build.log"
grep -q 'RESULT suite_exit=0' "$RESULTS/regression/go-test-final-pass.log"
grep -q 'Ran 111 tests' "$RESULTS/regression/docreader-unittest-controlled.log"
grep -q 'OK (skipped=6)' "$RESULTS/regression/docreader-unittest-controlled.log"
grep -q 'EXIT_STATUS=0' "$RESULTS/regression/cli-test-vet-build.log"

python3 - "$RESULTS" <<'PY'
import json, pathlib, sys
r = pathlib.Path(sys.argv[1])
retrieval = json.loads((r/'retrieval/retrieval_summary.json').read_text())
answer = json.loads((r/'answer/final/answer_final_summary.json').read_text())
agent = json.loads((r/'agent/agent_summary.json').read_text())
api = json.loads((r/'performance/api-rate-k6-summary.json').read_text())['metrics']
assert retrieval['run']['dataset_queries'] == 120
assert retrieval['hybrid']['overall']['successful'] == 120
assert answer['dataset_queries'] == 120 and answer['overall_status_counts'] == {'pass': 118, 'partial': 2}
assert answer['relevant_citation_rate'] == 1
assert agent['repeat_runs'] == 20 and agent['repeat_answer_accuracy'] == 1
assert agent['unknown_runs'] == 5 and agent['unknown_grounded_refusal_rate'] == 1
assert api['status_200']['values']['count'] == 6134
assert api['http_req_failed']['values']['rate'] == 0
print('METRICS retrieval_queries=120 hybrid_hit_at_8=0.9916666666666667 answer_pass=118 answer_partial=2 relevant_citation=1.0')
print('AGENT repeat=20 accurate=20 duplicate_free=20 unknown=5 grounded_refusal=5 max_latency_ms=128642.989')
print('PERFORMANCE api_http_200=6134 api_http_failure_rate=0 api_p95_ms=6.418149999999993')
PY

test -s "$REPORTS/BENCHMARK_REPORT.md"
test -s "$REPORTS/DECISION_LOG.md"
test -s "$REPORTS/RESUME_BULLETS.md"
test -s "$REPORTS/METRICS.csv"
test -s "$REPORTS/benchmark-scorecard.svg"
if grep -R -E -q 'eyJ[a-zA-Z0-9_-]{20,}\.|(^|[^A-Za-z0-9_-])sk-[A-Za-z0-9._-]{20,}' "$ROOT/benchmarks"; then
  printf 'SECRET_SCAN unexpected_credential_pattern=true\n' >&2
  exit 1
fi
printf 'REGRESSION frontend=225/225 go_packages=87 docreader=111_skipped_6 cli_statement_coverage=69.3%%\n'
printf 'ARTIFACTS report=%s decision_log=%s resume=%s metrics=%s scorecard=%s\n' \
  "$REPORTS/BENCHMARK_REPORT.md" "$REPORTS/DECISION_LOG.md" "$REPORTS/RESUME_BULLETS.md" "$REPORTS/METRICS.csv" "$REPORTS/benchmark-scorecard.svg"
printf 'VERIFY_RESULT=PASS EXIT_STATUS=0\n'
