import pathlib,json,hashlib,subprocess
root=pathlib.Path.cwd();p=root/'.work/native-eval-20260908';o=root/'docs/benchmarks/2026-09-08-native-evaluation'
rows=list(map(json.loads,(o/'runs.jsonl').read_text().splitlines()));assert len(rows)==9 and len({r['key'] for r in rows})==9 and all(r['execution_complete'] for r in rows)
configs=[json.load(open(o/(m+'_agent.json')))['data']['config'] for m in ['rag','wiki','hybrid']]
diff=[k for k in set().union(*(c.keys() for c in configs)) if not all(c.get(k)==configs[0].get(k) for c in configs)]
assert set(diff)=={'agent_type','allowed_tools','system_prompt','system_prompt_id'},diff
for line in (o/'ORIGINAL_HASHES.txt').read_text().splitlines():
 h,f=line.split(None,1);assert hashlib.sha256((root/f.strip()).read_bytes()).hexdigest()==h,f
assert (p/'rollback-copy/evaluation.go').read_bytes()==(p/'evaluation.go.original').read_bytes()
assert (p/'repo/internal/application/service/evaluation.go').read_bytes()==(o/'artifacts/MODIFIED_FILE.go').read_bytes()
assert (o/'artifacts/MODIFIED_FILE.go').read_bytes()!=(p/'evaluation.go.original').read_bytes()
print('FORMAL_RUNS=9/9 COMPLETE; UNIQUE_QUESTIONS=1; MODES=3; REPEATS=3')
print('CONFIG_DIFF='+','.join(sorted(diff)))
print('ORIGINAL_HASHES=UNCHANGED; ROLLBACK_HASH_MATCH=YES; MODIFIED_COPY=RETAINED')
for name in ['REPORT.md','COMPARISON.md','DIFF_FILE.patch','ROLLBACK.sh']:
 assert (o/name).read_text().strip();print('REOPENED='+str(o/name))
