import requests,json,time,pathlib
p=pathlib.Path(__file__).parent; out=pathlib.Path('docs/benchmarks/2026-09-08-native-evaluation'); s=requests.Session();s.headers['Authorization']='Bearer '+(p/'token').read_text()
body={'dataset_id':'default','knowledge_base_id':'b68cc4f9-71d8-4f25-93a6-d8c1f90c3dae','chat_id':'cb425145-111f-4150-a02d-f5a67ca61bd4','rerank_id':'266160fe-ec66-42de-a138-1a2950c1d867'}
(out/'native_request.json').write_text(json.dumps(body,indent=2));r=s.post('http://127.0.0.1:8080/api/v1/evaluation',json=body,timeout=120)
print('CREATE_HTTP',r.status_code,flush=True);(out/'native_create.json').write_text(r.text);r.raise_for_status(); task=r.json()['data']['task']['id'];print('TASK',task,flush=True)
for i in range(90):
 time.sleep(2);r=s.get('http://127.0.0.1:8080/api/v1/evaluation',params={'task_id':task},timeout=30);(out/'native_result.json').write_text(r.text);r.raise_for_status();d=r.json()['data'];print('POLL',d['task'],flush=True)
 if d['task']['status'] in (2,3):print('NATIVE_RESULT',json.dumps(d.get('metric')),flush=True);break
else:raise SystemExit('TIMEOUT')
