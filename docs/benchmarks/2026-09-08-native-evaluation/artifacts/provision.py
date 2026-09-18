import requests,json,time,pathlib,pyarrow.parquet as pq
p=pathlib.Path(__file__).parent;out=pathlib.Path('docs/benchmarks/2026-09-08-native-evaluation');s=requests.Session();s.headers['Authorization']='Bearer '+(p/'token').read_text();base='http://127.0.0.1:8080/api/v1'
def call(method,path,**kw):
 r=s.request(method,base+path,timeout=120,**kw)
 if not r.ok:raise RuntimeError((r.status_code,r.text))
 return r.json()
def save(n,x): (out/n).write_text(json.dumps(x,ensure_ascii=False,indent=2))
body={'name':'native-eval-controlled-20260908','type':'document','embedding_model_id':'60a033b0-81bb-4f1c-8d4c-66ff3c3ad993','summary_model_id':'cb425145-111f-4150-a02d-f5a67ca61bd4','indexing_strategy':{'vector_enabled':True,'keyword_enabled':True,'wiki_enabled':True,'graph_enabled':False},'wiki_config':{'synthesis_model_id':'cb425145-111f-4150-a02d-f5a67ca61bd4','max_pages_per_ingest':20,'ingest_map_parallel':1,'ingest_reduce_parallel':1},'chunking_config':{'chunk_size':1024,'chunk_overlap':0},'question_generation_config':{'enabled':False}}
kb=call('POST','/knowledge-bases',json=body);save('kb_created.json',kb);kid=kb['data']['id'];(p/'kb_id').write_text(kid);print('KB_CREATED',kid,flush=True)
manifest=[]
for row in pq.read_table('dataset/samples/corpus.parquet').to_pylist():
 d=call('POST',f'/knowledge-bases/{kid}/knowledge/manual',json={'title':f'native-passage-{row["id"]}','content':row['text'],'status':'publish'})['data'];manifest.append({'pid':int(row['id']),'knowledge_id':d['id'],'title':d['title'],'source_text':row['text']});print('PASSAGE_CREATED',row['id'],d['id'],flush=True);save('source_manifest.json',manifest)
presets=call('GET','/agents/type-presets');save('agent_presets.json',presets);print('PRESET_KEYS',str(presets)[:300],flush=True)
