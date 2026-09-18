import requests,json,pathlib,time,pyarrow.parquet as pq
p=pathlib.Path(__file__).parent;out=pathlib.Path('docs/benchmarks/2026-09-08-native-evaluation');(out/'raw').mkdir(exist_ok=True)
s=requests.Session();s.headers['Authorization']='Bearer '+(p/'token').read_text();base='http://127.0.0.1:18080/api/v1';kb=(p/'kb_id').read_text();agents=json.load(open(p/'agent_ids.json'))
def save(name,obj): (out/name).write_text(json.dumps(obj,ensure_ascii=False,indent=2))
def call(method,path,**kw):
 r=s.request(method,base+path,timeout=90,**kw);r.raise_for_status();return r.json()
queries=pq.read_table('dataset/samples/queries.parquet').to_pylist();query=queries[0]['text'];save('dataset.json',{name:pq.read_table('dataset/samples/'+name+'.parquet').to_pylist() for name in ['queries','corpus','answers','qrels','qas']})
pages=call('GET',f'/knowledgebase/{kb}/wiki/pages',params={'page_size':200});save('wiki_pages.json',pages);assert pages.get('total',len(pages.get('pages',[])))>0,pages
save('wiki_stats.json',call('GET',f'/knowledgebase/{kb}/wiki/stats'))
done={}
if (out/'runs.jsonl').exists():done={r['key']:r for r in map(json.loads,(out/'runs.jsonl').read_text().splitlines())}
for rep in range(3):
 order=['rag','wiki','hybrid'];order=order[rep:]+order[:rep]
 for mode in order:
  key=f'{mode}-{rep+1}'
  if key in done:continue
  session=call('POST','/sessions',json={'title':'native-eval-'+key,'knowledge_base_id':kb});sid=session['data']['id'];save('raw/'+key+'-session.json',session)
  body={'query':query,'agent_id':agents[mode],'agent_enabled':True,'knowledge_base_ids':[kb],'disable_title':True};save('raw/'+key+'-request.json',body)
  events=[];start=time.monotonic();error=None;status=None
  try:
   with s.post(base+'/agent-chat/'+sid,json=body,stream=True,timeout=(20,240)) as r:
    status=r.status_code;r.raise_for_status()
    with (out/'raw'/f'{key}.sse').open('w') as f:
     for line in r.iter_lines():
      line=line.decode('utf-8');f.write(line+'\n');f.flush()
      if line.startswith('data:'):
       content=line[5:].strip()
       if content and content!='[DONE]':events.append(json.loads(content))
  except Exception as ex:error=str(ex)
  groups={};order=[];tools=[];complete={};errors=[]
  for e in events:
   d=e.get('data') or {};typ=e.get('response_type')
   if typ=='answer':
    eid=d.get('event_id','unknown')
    if eid not in groups:groups[eid]=[];order.append(eid)
    if e.get('content'):groups[eid].append(e['content'])
   elif typ=='tool_call' and d.get('arguments') is not None:tools.append(d.get('tool_name'))
   elif typ=='complete':complete=d
   elif typ in ['error','agent_error'] or (typ=='tool_result' and d.get('error')):errors.append(e)
  answer=''.join(groups[order[-1]]) if order else ''
  row={'key':key,'mode':mode,'repeat':rep+1,'question':query,'session_id':sid,'http_status':status,'latency_ms':round((time.monotonic()-start)*1000,3),'final_answer':answer,'tools':tools,'complete':complete,'error':error,'tool_errors':errors,'execution_complete':bool(complete and answer and not error),'events':events}
  with (out/'runs.jsonl').open('a') as f:f.write(json.dumps(row,ensure_ascii=False)+'\n')
  print(key,'HTTP',status,'COMPLETE',row['execution_complete'],'MS',row['latency_ms'],'TOOLS',tools,'ANSWER',answer[:90],flush=True)
