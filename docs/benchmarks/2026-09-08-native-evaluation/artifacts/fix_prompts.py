import pathlib,json,requests,yaml,shutil
p=pathlib.Path(__file__).parent;o=pathlib.Path('docs/benchmarks/2026-09-08-native-evaluation');pilot=o/'pilot-unresolved-prompts';pilot.mkdir(exist_ok=True)
for n in ['raw','runs.jsonl','MODES_RUN.log']:
 if (o/n).exists():shutil.move(str(o/n),str(pilot/n))
s=requests.Session();s.headers['Authorization']='Bearer '+(p/'token').read_text();templates={x['id']:x['content'] for x in yaml.safe_load(pathlib.Path('config/prompt_templates/agent_system_prompt.yaml').read_text())['templates']}
for m,aid in json.load(open(p/'agent_ids.json')).items():
 d=json.load(open(o/(m+'_agent.json')))['data'];shutil.copy(o/(m+'_agent.json'),pilot/(m+'_agent.json'));c=d['config'];c['system_prompt']=templates[c['system_prompt_id']]
 r=s.put('http://127.0.0.1:18080/api/v1/agents/'+aid,json={'name':d['name'],'config':c},timeout=30);r.raise_for_status();(o/(m+'_agent.json')).write_text(json.dumps(r.json(),ensure_ascii=False,indent=2));print(m,len(c['system_prompt']))
