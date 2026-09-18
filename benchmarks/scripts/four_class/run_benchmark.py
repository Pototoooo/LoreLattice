#!/usr/bin/env python3
from __future__ import annotations
import argparse,json,os,re,statistics,subprocess,time
from datetime import datetime,timezone
from pathlib import Path

DOC_IDS={"93639dd6-f7d6-4d29-bc4f-013c7d5726bf","deebd8ee-b757-468a-8649-8edde1fb728d","63cd98a5-7250-476c-9a42-06bdda60dea0","2b4c4809-3fc8-4ea2-8f8e-f91e5a28b9b2","3b99bbfe-72a0-496f-9a4e-53b599695170","40cb6e26-0345-44b8-a021-d819d232f619","0173c370-1a56-421e-be61-71df39284cd5","fb2bb21e-1769-4de1-9e41-e4eb9217ceb9","01e15f76-5984-4dba-8c6a-34e5c950ed09","59808dd7-f818-4bc8-9bcc-223df7a6e19e","2b6265dd-c6c2-48c6-a047-f7088c178978","686f8b78-34c0-4aea-bf22-18a6276c23d8"}

def norm(s:str)->str:return re.sub(r"[\s,，。；;：:、*`#_\-（）()\[\]{}]","",s or "").lower()
def final_answer(events):
 groups={};order=[]
 for e in events:
  if e.get('response_type')!='answer':continue
  k=(e.get('data') or {}).get('event_id','unknown')
  if k not in groups:groups[k]=[];order.append(k)
  if e.get('content'):groups[k].append(e['content'])
 return (''.join(groups[order[-1]]) if order else '',len(order))
def find_doc_ids(x):
 out=set()
 if isinstance(x,str): out|={d for d in DOC_IDS if d in x}
 elif isinstance(x,list):
  for v in x:out|=find_doc_ids(v)
 elif isinstance(x,dict):
  for v in x.values():out|=find_doc_ids(v)
 return out

def project(mode,case,repeat,payload,latency,exit_status,stderr,page_map,doc_map):
 events=payload.get('data',{}).get('events',[]) if isinstance(payload,dict) else []
 answer,stages=final_answer(events); tools=[];errors=[];read_pages=set();retrieved=set();complete=False;steps=None;server_ms=None;usage={'prompt_tokens':0,'completion_tokens':0,'total_tokens':0,'cached_tokens':0}
 for e in events:
  d=e.get('data') or {};t=e.get('response_type')
  if t=='tool_call' and d.get('arguments') is not None:tools.append(d.get('tool_name','unknown'))
  elif t=='tool_result':
   name=d.get('tool_name','')
   if d.get('error'):errors.append(str(d['error']))
   if name=='wiki_read_page':
    pages=set(re.findall(r'(?s)<wiki_page>.*?<metadata>.*?<link>\[\[([^]|]+)',str(d.get('output') or '')));read_pages|=pages
    for p in pages:retrieved.update(page_map.get(p) or [])
   elif name in {'knowledge_search','grep_chunks','list_knowledge_chunks','get_document_info'}:retrieved|=find_doc_ids(d)
  elif t in {'error','agent_error'}:errors.append(str(d.get('error') or e.get('content') or 'agent_error'))
  elif t=='complete':
   complete=True;steps=d.get('total_steps');server_ms=d.get('total_duration_ms')
   for k in usage:usage[k]=int(d.get(k) or 0)
 # citations normalized to source-document IDs
 cited=set()
 for name in re.findall(r'<kb\b[^>]*\bdoc="([^"]+)"',answer):
  if name in doc_map:cited.add(doc_map[name])
 for slug in re.findall(r'\[\[([^]|]+)(?:\|[^]]*)?\]\]',answer):cited.update(page_map.get(slug) or [])
 n=norm(answer);hits=[any(norm(term) in n for term in g) for g in case['required_fact_groups']];comp=statistics.mean(hits) if hits else 0
 expected=set(case['expected_doc_ids']);retrieval=len(expected&retrieved)/len(expected);citation=len(expected&cited)/len(expected)
 paras=[norm(x) for x in re.split(r'\n\s*\n',answer) if norm(x)];dups=len(paras)-len(set(paras))
 return {'case_id':case['id'],'category':case['category'],'mode':mode,'repeat':repeat,'question':case['question'],'exit_status':exit_status,'latency_ms':round(latency,3),'server_duration_ms':server_ms,'server_steps':steps,'event_count':len(events),'answer_stage_count':stages,'answer_chars':len(answer),'tools':tools,'tool_call_count':len(tools),'tool_errors':errors,'tool_error_count':len(errors),'execution_complete':exit_status==0 and complete and bool(answer.strip()),'fact_group_hits':hits,'fact_completeness':comp,'strict_accuracy':exit_status==0 and complete and bool(answer.strip()) and comp==1 and dups==0,'duplicate_paragraphs':dups,'expected_doc_ids':sorted(expected),'retrieved_doc_ids':sorted(retrieved),'cited_doc_ids':sorted(cited),'retrieval_recall':retrieval,'citation_recall':citation,'read_wiki_pages':sorted(read_pages),'expected_wiki_pages':case['expected_wiki_pages'],'rerank_path_used':'knowledge_search' in tools,**usage,'final_answer':answer,'stderr':stderr,'raw':payload}

def main():
 ap=argparse.ArgumentParser();ap.add_argument('--cli',type=Path,required=True);ap.add_argument('--token-file',type=Path,required=True);ap.add_argument('--agents',type=Path,required=True);ap.add_argument('--dataset',type=Path,required=True);ap.add_argument('--page-map',type=Path,required=True);ap.add_argument('--doc-map',type=Path,required=True);ap.add_argument('--output-dir',type=Path,required=True);ap.add_argument('--repeats',type=int,default=1);ap.add_argument('--base-url',default='http://127.0.0.1:8080');ap.add_argument('--timeout',type=int,default=150);ap.add_argument('--cooldown',type=float,default=.5);a=ap.parse_args();a.output_dir.mkdir(parents=True,exist_ok=True)
 agents=dict(x.split('\t',1) for x in a.agents.read_text().splitlines() if x.strip());cases=[json.loads(x) for x in a.dataset.read_text().splitlines()];page_map=json.load(open(a.page_map));doc_map=json.load(open(a.doc_map));modes=['rag','wiki','hybrid'];out=a.output_dir/'results.jsonl';done=set()
 if out.exists():
  existing=[json.loads(line) for line in out.read_text().splitlines() if line.strip()]
  valid=[r for r in existing if r.get('exit_status')==0 and r.get('execution_complete')]
  invalid=[r for r in existing if r not in valid]
  if invalid:
   stamp=datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')
   failed=out.with_name(f'results.failed-{stamp}.jsonl')
   failed.write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in invalid))
   out.write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in valid))
   print(f'RESUME_REPAIR kept={len(valid)} quarantined={len(invalid)} path={failed}',flush=True)
  done={(r['case_id'],r['mode'],r['repeat']) for r in valid}
 env=os.environ.copy();env.update({'LORELATTICE_TOKEN':a.token_file.read_text().strip(),'LORELATTICE_HOST':a.base_url,'LORELATTICE_LOG_LEVEL':'error'});total=len(cases)*len(modes)*a.repeats;completed=len(done)
 for rep in range(1,a.repeats+1):
  for idx,case in enumerate(cases):
   order=modes[(idx+rep-1)%3:]+modes[:(idx+rep-1)%3]
   for mode in order:
    key=(case['id'],mode,rep)
    if key in done:continue
    cmd=[str(a.cli),'session','ask','--agent',agents[mode],'--reference','--verbose','--format','json',case['question']];start=time.perf_counter()
    try:p=subprocess.run(cmd,env=env,text=True,capture_output=True,timeout=a.timeout);status,stdout,stderr=p.returncode,p.stdout,p.stderr
    except subprocess.TimeoutExpired as e:status,stdout,stderr=124,e.stdout or '',(e.stderr or '')+f'\nTIMEOUT after {a.timeout}s'
    latency=(time.perf_counter()-start)*1000
    try:payload=json.loads(stdout)
    except Exception:payload={'ok':False,'parse_error':stdout}
    row=project(mode,case,rep,payload,latency,status,stderr,page_map,doc_map)
    with out.open('a') as f:f.write(json.dumps(row,ensure_ascii=False)+'\n')
    completed+=1;print(f"RUN_RESULT {completed}/{total} repeat={rep} case={row['case_id']} category={row['category']} mode={mode} exit={status} accuracy={row['strict_accuracy']} completeness={row['fact_completeness']:.3f} retrieval={row['retrieval_recall']:.3f} citation={row['citation_recall']:.3f} tokens={row['total_tokens']} latency_ms={latency:.1f}",flush=True);time.sleep(a.cooldown)
    if status in {3,7} and not row['execution_complete']:
     print(f"RUN_ABORT exit={status} case={row['case_id']} mode={mode} stderr={stderr.strip()}",flush=True)
     return status
 print(f'RUN_COMPLETE rows={completed} expected={total}')
if __name__=='__main__':raise SystemExit(main())
