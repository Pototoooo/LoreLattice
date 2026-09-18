import requests,json,pathlib
p=pathlib.Path(__file__).parent;out=pathlib.Path('docs/benchmarks/2026-09-08-native-evaluation');s=requests.Session();s.headers['Authorization']='Bearer '+(p/'token').read_text();kb=(p/'kb_id').read_text()
presets=json.load(open(out/'agent_presets.json'))['data']; ids={}
for mode,typ in [('rag','rag-qa'),('wiki','wiki-qa'),('hybrid','hybrid-rag-wiki')]:
 config=next(x['config'].copy() for x in presets if x['id']==typ)
 config.update({'agent_mode':'smart-reasoning','agent_type':typ,'model_id':'cb425145-111f-4150-a02d-f5a67ca61bd4','rerank_model_id':'266160fe-ec66-42de-a138-1a2950c1d867','temperature':0,'max_completion_tokens':4096,'max_iterations':30,'llm_call_timeout':120,'citation_enabled':True,'thinking':False,'kb_selection_mode':'selected','knowledge_bases':[kb],'mcp_selection_mode':'none','skills_selection_mode':'none','web_search_enabled':False,'web_fetch_enabled':False,'multi_turn_enabled':False,'retain_retrieval_history':False,'embedding_top_k':10,'keyword_threshold':0.3,'vector_threshold':0.5,'rerank_top_k':10,'rerank_threshold':0.3,'faq_priority_enabled':False,'enable_query_expansion':False,'enable_rewrite':False,'fallback_strategy':'fixed'})
 r=s.post('http://127.0.0.1:8080/api/v1/agents',json={'name':'native-eval-'+mode+'-20260908','config':config},timeout=60)
 r.raise_for_status();d=r.json();(out/(mode+'_agent.json')).write_text(json.dumps(d,ensure_ascii=False,indent=2));ids[mode]=d['data']['id'];print(mode,ids[mode],flush=True)
(p/'agent_ids.json').write_text(json.dumps(ids))
r=s.get(f'http://127.0.0.1:8080/api/v1/knowledgebase/{kb}/wiki/pages',params={'page_size':200},timeout=60);r.raise_for_status();(out/'wiki_pages.json').write_text(r.text);print('WIKI_PAGES',r.text[:400],flush=True)
