import json,pathlib,re,statistics,subprocess,hashlib
root=pathlib.Path.cwd();p=root/'.work/native-eval-20260908';o=root/'docs/benchmarks/2026-09-08-native-evaluation';rows=list(map(json.loads,(o/'runs.jsonl').read_text().splitlines()));ds=json.load(open(o/'dataset.json'));reference=ds['answers'][0]['text'];manifest=json.load(open(o/'source_manifest.json'));docs={x['knowledge_id']:x['pid'] for x in manifest};titles={x['title']:x['pid'] for x in manifest};pages={x['slug']:[docs[d] for d in (x.get('source_refs') or []) if d in docs] for x in json.load(open(o/'wiki_pages.json'))['pages']}
inputs=[]
for r in rows:
 retrieved=set();cited=set();read=[];ambiguous=0
 for e in r['events']:
  if e.get('response_type')!='tool_result':continue
  d=e.get('data') or {};name=d.get('tool_name');blob=json.dumps(d,ensure_ascii=False)
  if d.get('ambiguous_slugs'):ambiguous+=1
  if name=='wiki_read_page':
   slugs=re.findall(r'(?s)<wiki_page>.*?<metadata>.*?<link>\[\[([^]|]+)',d.get('output',''));read+=slugs
   for slug in slugs:retrieved.update(pages.get(slug,[]))
  elif name in ['knowledge_search','grep_chunks','list_knowledge_chunks','get_document_info','wiki_read_source_doc']:
   retrieved.update(pid for doc,pid in docs.items() if doc in blob)
 for name in re.findall(r'<kb\b[^>]*\bdoc="([^"]+)"',r['final_answer']):
  if name in titles:cited.add(titles[name])
 for slug in re.findall(r'\[\[([^]|]+)(?:\|[^]]*)?\]\]',r['final_answer']):cited.update(pages.get(slug,[]))
 norm=re.sub(r'\s','',r['final_answer']).lower();hits=[any(x in norm for x in group) for group in [['unix'],['linux'],['freebsd'],['macos','macintosh','mac操作系统'],['dos']]]
 r.update({'retrieved_pids':sorted(retrieved),'cited_pids':sorted(cited),'source_coverage':len(retrieved)/4,'citation_source_coverage':len(cited)/4,'reference_entity_coverage':sum(hits)/5,'all_five_reference_entities':all(hits),'wiki_pages_read':read,'ambiguous_page_events':ambiguous,'rerank_path_used':'knowledge_search' in r['tools']})
 inputs.append({'key':r['key'],'answer':r['final_answer'],'reference':reference,'retrieved':[],'relevant':[1,2,3,4]})
(o/'metric_input.json').write_text(json.dumps(inputs,ensure_ascii=False,indent=2));cmd=[str(p/'native-metrics'),str(o/'native_scores.json')]
with open(o/'METRICS_RUN.log','w') as log:subprocess.run(cmd,input=json.dumps(inputs),text=True,stdout=log,stderr=log,check=True)
scores={x['key']:x['metrics']['generation_metrics'] for x in json.load(open(o/'native_scores.json'))}
summary={}
for r in rows:r['native_generation_metrics']=scores[r['key']];r.pop('events')
for m in ['rag','wiki','hybrid']:
 a=[r for r in rows if r['mode']==m];mean=lambda k:statistics.mean(r[k] for r in a)
 summary[m]={'runs':len(a),'execution_complete_rate':mean('execution_complete'),'source_coverage':mean('source_coverage'),'citation_source_coverage':mean('citation_source_coverage'),'reference_entity_coverage':mean('reference_entity_coverage'),'all_five_reference_entities_rate':mean('all_five_reference_entities'),'latency_p50_ms':statistics.median(r['latency_ms'] for r in a),'mean_total_tokens':statistics.mean(r['complete'].get('total_tokens',0) for r in a),'mean_tool_calls':statistics.mean(len(r['tools']) for r in a),'rerank_path_runs':sum(r['rerank_path_used'] for r in a),'ambiguous_page_events':sum(r['ambiguous_page_events'] for r in a),'tool_error_events':sum(len(r['tool_errors']) for r in a),'native_generation_metrics':{k:statistics.mean(r['native_generation_metrics'][k] for r in a) for k in scores[a[0]['key']]}}
(o/'SCORED_RUNS.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2));(o/'SUMMARY.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2));print(json.dumps(summary,indent=2))
