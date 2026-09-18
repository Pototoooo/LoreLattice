#!/usr/bin/env python3
from __future__ import annotations
import argparse,csv,json,math,random,statistics
from collections import defaultdict
from pathlib import Path

def pct(x):return None if x is None else round(100*x,2)
def pctl(xs,q):
 if not xs:return None
 ys=sorted(xs);pos=(len(ys)-1)*q;lo=int(pos);hi=min(lo+1,len(ys)-1);return ys[lo]+(ys[hi]-ys[lo])*(pos-lo)
def agg(rows):
 def mean(k):return statistics.mean(float(r[k]) for r in rows) if rows else None
 return {'runs':len(rows),'execution_complete_rate':mean('execution_complete'),'strict_accuracy':mean('strict_accuracy'),'mean_completeness':mean('fact_completeness'),'mean_retrieval_recall':mean('retrieval_recall'),'mean_citation_recall':mean('citation_recall'),'tool_error_free_rate':statistics.mean(not r['tool_errors'] for r in rows),'mean_tool_calls':mean('tool_call_count'),'rerank_path_rate':mean('rerank_path_used'),'mean_prompt_tokens':mean('prompt_tokens'),'mean_completion_tokens':mean('completion_tokens'),'mean_total_tokens':mean('total_tokens'),'median_total_tokens':pctl([r['total_tokens'] for r in rows],.5),'p95_total_tokens':pctl([r['total_tokens'] for r in rows],.95),'mean_cached_tokens':mean('cached_tokens'),'latency_ms':{'mean':mean('latency_ms'),'p50':pctl([r['latency_ms'] for r in rows],.5),'p95':pctl([r['latency_ms'] for r in rows],.95),'max':max((r['latency_ms'] for r in rows),default=None)}}
def paired(rows,a,b,category=None):
 by={(r['case_id'],r['repeat'],r['mode']):r for r in rows if category is None or r['category']==category};pairs=[]
 for case,rep,mode in list(by):
  if mode!=a:continue
  ra=by.get((case,rep,a));rb=by.get((case,rep,b))
  if ra and rb:pairs.append((ra,rb))
 deltas=[x['fact_completeness']-y['fact_completeness'] for x,y in pairs];wins=sum(d>1e-12 for d in deltas);losses=sum(d<-1e-12 for d in deltas);ties=len(deltas)-wins-losses
 random.seed(20260902);boots=[]
 if deltas:
  for _ in range(10000):boots.append(statistics.mean(random.choice(deltas) for _ in deltas))
  boots.sort();ci=[boots[int(.025*(len(boots)-1))],boots[int(.975*(len(boots)-1))]]
 else:ci=[None,None]
 non_tie=wins+losses
 return {'comparisons':len(pairs),'wins':wins,'ties':ties,'losses':losses,'win_rate_all':wins/len(pairs) if pairs else None,'win_rate_non_tie':wins/non_tie if non_tie else None,'mean_completeness_delta':statistics.mean(deltas) if deltas else None,'bootstrap_95ci':ci}
def main():
 ap=argparse.ArgumentParser();ap.add_argument('--input',type=Path,required=True);ap.add_argument('--output-dir',type=Path,required=True);a=ap.parse_args();a.output_dir.mkdir(parents=True,exist_ok=True);rows=[json.loads(x) for x in a.input.read_text().splitlines() if x.strip()];modes=['rag','wiki','hybrid'];cats=['single_hop_factual','local_synthesis','multi_hop','global_thematic']
 summary={'protocol':{'dataset':'four_class_v1','raw_runs':len(rows),'cases':len(set(r['case_id'] for r in rows)),'repeats':len(set(r['repeat'] for r in rows)),'modes':modes,'categories':cats,'accuracy':'all frozen fact groups present','completeness':'mean frozen fact-group coverage','retrieval_recall':'expected source documents retrieved / expected source documents','citation_recall':'expected source documents represented by final citations / expected source documents','tokens':'cumulative API-reported usage across every Agent LLM round'},'overall':{},'category':{},'pairwise':{}}
 for m in modes:summary['overall'][m]=agg([r for r in rows if r['mode']==m])
 for c in cats:summary['category'][c]={m:agg([r for r in rows if r['mode']==m and r['category']==c]) for m in modes}
 for c in [None]+cats:
  k='overall' if c is None else c;summary['pairwise'][k]={'wiki_vs_rag':paired(rows,'wiki','rag',c),'hybrid_vs_rag':paired(rows,'hybrid','rag',c)}
 (a.output_dir/'SUMMARY.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
 with (a.output_dir/'METRICS.csv').open('w',newline='') as f:
  w=csv.writer(f);w.writerow(['category','mode','runs','execution_complete_rate','strict_accuracy','fact_completeness','retrieval_recall','citation_recall','rerank_path_rate','tool_error_free_rate','mean_prompt_tokens','mean_completion_tokens','mean_total_tokens','median_total_tokens','p95_total_tokens','p50_latency_ms','p95_latency_ms','mean_tool_calls'])
  for c in ['overall']+cats:
   block=summary['overall'] if c=='overall' else summary['category'][c]
   for m in modes:
    x=block[m];w.writerow([c,m,x['runs'],x['execution_complete_rate'],x['strict_accuracy'],x['mean_completeness'],x['mean_retrieval_recall'],x['mean_citation_recall'],x['rerank_path_rate'],x['tool_error_free_rate'],x['mean_prompt_tokens'],x['mean_completion_tokens'],x['mean_total_tokens'],x['median_total_tokens'],x['p95_total_tokens'],x['latency_ms']['p50'],x['latency_ms']['p95'],x['mean_tool_calls']])
 lines=['# Four-class RAG / Wiki / Hybrid benchmark','',f"Runs: {len(rows)}; cases: {summary['protocol']['cases']}; repeats: {summary['protocol']['repeats']}",'','| Category | Mode | Accuracy | Completeness | Retrieval recall | Citation recall | Rerank path | Mean / median tokens | P50 / P95 latency |','|---|---|---:|---:|---:|---:|---:|---:|---:|']
 for c in ['overall']+cats:
  block=summary['overall'] if c=='overall' else summary['category'][c]
  for m in modes:
   x=block[m];lines.append(f"| {c} | {m} | {pct(x['strict_accuracy'])}% | {pct(x['mean_completeness'])}% | {pct(x['mean_retrieval_recall'])}% | {pct(x['mean_citation_recall'])}% | {pct(x['rerank_path_rate'])}% | {x['mean_total_tokens']:.0f} / {x['median_total_tokens']:.0f} | {x['latency_ms']['p50']/1000:.2f}s / {x['latency_ms']['p95']/1000:.2f}s |")
 lines+=['','## Pairwise completeness']
 for c,v in summary['pairwise'].items():
  for cmp,x in v.items():lines.append(f"- {c} {cmp}: {x['wins']}W/{x['ties']}T/{x['losses']}L, all-comparison win rate={pct(x['win_rate_all'])}%, delta={pct(x['mean_completeness_delta'])} pp, bootstrap 95% CI=[{pct(x['bootstrap_95ci'][0])}, {pct(x['bootstrap_95ci'][1])}] pp")
 (a.output_dir/'REPORT.md').write_text('\n'.join(lines)+'\n')
 print(f"ANALYZE_RESULT=PASS rows={len(rows)} output={a.output_dir/'SUMMARY.json'}")
if __name__=='__main__':main()
