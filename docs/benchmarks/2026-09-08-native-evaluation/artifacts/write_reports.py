import json,pathlib,hashlib,shutil
r=pathlib.Path.cwd();p=r/'.work/native-eval-20260908';o=r/'docs/benchmarks/2026-09-08-native-evaluation';s=json.load(open(o/'SUMMARY.json'));old=json.load(open(r/'docs/benchmarks/2026-09-02-four-class-benchmark/SUMMARY.json'));rows=json.load(open(o/'SCORED_RUNS.json'))
def table(keys):
 text='| 指标 | RAG | Wiki | Hybrid |\n|---|---:|---:|---:|\n'
 for label,key,fmt in keys:
  vals=[s[m]['native_generation_metrics'][key] if key in s[m]['native_generation_metrics'] else s[m][key] for m in ['rag','wiki','hybrid']];text+='| '+label+' | '+' | '.join(format(x,fmt) for x in vals)+' |\n'
 return text
report='''# 原生数据集 RAG / Wiki / Hybrid 评测报告

日期：2026-09-08。结论：本次完整执行原生 Evaluation，并用原生题集及评分函数完成三模式适配评测。**这不是三个 Embedding 模型对比，也不是未改动原生 API 原本就支持三模式。**

## 1. 数据与样本量

项目自带 `dataset/samples` 五个 Parquet 文件原样读取，哈希见 ORIGINAL_HASHES.txt。只有 **1 道独立问题、4 段原文、1 条参考答案、4 条相关文档标签**。问题为“计算机的操作系统有哪些”；参考答案为“计算机的操作系统有 UNIX、Linux、FreeBSD、Mac OS 和 DOS。”

三模式各重复 3 次，共 9 次正式调用；循环轮换 RAG→Wiki→Hybrid、Wiki→Hybrid→RAG、Hybrid→RAG→Wiki。重复次数不是独立问题数，不计算显著性或泛化胜率。题目和参考答案未扩充、未自行埋点。4 段原文通过原生手工文档发布 API 原样入库，各段独立为一篇文档；标题仅为 native-passage-N。Wiki 通过项目原生异步流程自动生成，固定同一份 21 页快照，未人工填写答案页。

## 2. 控制变量

| 项目 | 固定值 |
|---|---|
| 代码基线 | main a0a27ed4d21335bfa3a83e4f163787c307e492f0 |
| Embedding | text-embedding-v4 |
| 回答 / Wiki 合成模型 | qwen3.7-plus |
| Rerank 配置 | qwen3-rerank；阈值 0.3；Top-K 10 |
| 向量召回 | Top-K 10；阈值 0.5 |
| 关键词阈值 | 0.3 |
| 回答温度 / 上限 | 0 / 4096 tokens |
| Agent 迭代上限 / 单轮超时 | 30 / 120 秒 |
| 会话 | 每次新会话；多轮、历史检索复用关闭 |
| 其他 | 改写、扩写、联网、MCP、Skills 关闭；引用开启 |
| 数据与服务 | 同一 KB、同一冻结 Wiki、同一隔离服务、串行执行 |

变化项是原生 Agent 类型及其自带提示词和工具集合，这是“模式整体”的对照，不是单独证明知识结构的因果效应。RAG 使用知识搜索/原文工具；Wiki 使用 Wiki 搜索、页面及源文档工具；Hybrid 使用 Wiki + 原文工具。Rerank 配置相同，但 **Wiki 页面检索不调用向量 Rerank**；Hybrid 也可选 grep/list 而绕过 knowledge_search，因此“已配置”不等于每次实际调用。实际路径计数见下表及 RERANK_EVIDENCE.log。

## 3. 三模式正式结果

'''
report+=table([('完成率','execution_complete_rate','.1%'),('BLEU-1','bleu1','.4f'),('BLEU-2','bleu2','.4f'),('BLEU-4','bleu4','.4f'),('ROUGE-1','rouge1','.4f'),('ROUGE-2','rouge2','.4f'),('ROUGE-L','rougel','.4f'),('参考答案五实体覆盖率（补充）','reference_entity_coverage','.1%'),('五实体全覆盖运行占比（补充）','all_five_reference_entities_rate','.1%'),('源文档覆盖率（补充）','source_coverage','.1%'),('引用映射源文档覆盖率（补充）','citation_source_coverage','.1%'),('端到端 P50 毫秒','latency_p50_ms','.1f'),('平均累计 tokens','mean_total_tokens','.1f'),('平均工具调用数','mean_tool_calls','.2f'),('进入 knowledge_search 的次数 / 3','rerank_path_runs','d'),('页面歧义事件数','ambiguous_page_events','d'),('工具显式错误数','tool_error_events','d')])
report+='\n### 相对 RAG 的变化（仅本题）\n\n| 模式 | ROUGE-L 差值 | P50 耗时变化 | 平均 tokens 变化 |\n|---|---:|---:|---:|\n'
for m in ['wiki','hybrid']:
 report+=f"| {m} | {s[m]['native_generation_metrics']['rougel']-s['rag']['native_generation_metrics']['rougel']:+.4f} | {(s[m]['latency_p50_ms']/s['rag']['latency_p50_ms']-1):+.1%} | {(s[m]['mean_total_tokens']/s['rag']['mean_total_tokens']-1):+.1%} |\n"
report+='''
### 指标定义与解释

- BLEU/ROUGE 直接调用项目 Go `service.MetricList`，未重写公式；文本为最后一组 answer event 的拼接，保留实际 Markdown/引用。同文 sanity 得到 BLEU=1、ROUGE≈1，空回答=0。
- 参考答案只有一句话，实际回答很长，添加分节、解释和引用。这会压低词面匹配分，**低 BLEU/ROUGE 不等于同样低的事实准确率**。参考答案未涵盖原文提到的 Ubuntu、Mint 等信息，也不能把所有额外项直接判为幻觉。
- 补充实体覆盖只检查参考答案的五个系统名及明确别名，并非原生严格准确率；它不检查矛盾、额外陈述或来源忠实性。没有 LLM judge 胜率或人工全面事实核验。
- 补充源覆盖通过成功工具结果中的原文 ID，以及实际读到的 Wiki 页 source_refs 映射为 PID。引用覆盖使用最终答案引用进行同样映射。它是来源级代理指标，不证明页面保留了原文每个事实。
- Agent 轨迹是多轮混合检索，未硬造统一 Top-K 排序；三模式不公布伪造的 NDCG/MRR。native_scores.json 中 retrieval 字段是显式空输入产生的占位 0，仅使用其中 generation_metrics，来源覆盖另算。
- tokens 是 complete 事件累计 Agent 模型用量，不包含 Wiki 建库、Embedding、Rerank、自动标题生成等费用；耗时从发送问答请求到 SSE 结束，不包括建库时间。两者均非总系统成本。
- temperature=0 仍有服务端和模型波动，3 次重复仅能观察本题运行稳定性。

## 4. 原生入口与实现问题

原生 `/api/v1/evaluation` 的请求只有 dataset_id、knowledge_base_id、chat_id、rerank_id，内部固定走普通 RAG pipeline；datasetID 实际仍读取 samples。原生指标任务仅保存在进程内存，本报告已导出响应。

实际运行了两次原生入口：未修改版本检索六项均为 0；隔离修复后 Precision、Recall、NDCG@3、NDCG@10、MRR、MAP 均为 1。生成指标完整见 native_result.json 与 native_corrected_result.json。这两次的启动配置亦不同（原服务 Top-K=30、向量阈值0.2、温度0.3；隔离服务为本报告参数），**不得把 0→1 写成受控模型收益**。

确认的两个修复：
1. `getPassageList` 按 PID 作数组下标，却使用 maxPID 长度及 `< maxPID`，遗漏 PID=4 的 DOS 原文。副本改为 maxPID+1、<=maxPID；专门回归测试原版失败、修复后通过。
2. 原生评测异步导入后立刻问答并删除临时库，存在索引竞态。副本改为现有同步导入方法，执行日志确认 4 个 chunk 后再检索。

主工作区业务代码与原服务配置保持原状；修复放在 newBee/native-evaluation-20260908 隔离工作树，保留快照、补丁及回滚证据。

另外，初次预跑只传 system_prompt_id，运行层读取 system_prompt 导致通用提示词回退。正式实验显式填入同一代码版本自带的三种模板内容。此前预跑保存在 pilot-unresolved-prompts，**不纳入正式统计**。部分正式 Wiki 工具返回 ambiguous_slugs，且同一 KB ID 重复出现在候选列表；记录事件数作为导航诊断，不因 success=true 而忽略。未在实验中途继续改导航来追求更好分数。

## 5. 结论及可复核证据

原生样本适合确认接入、检索链路和评分器能跑通，但单题在实体覆盖上容易饱和，既不证明 Wiki 全局优势，也不推翻此前多跳实验。优先修好评测的导入/等待机制，再用独立多题集评估能力。不要将本题变化包装为普遍“准确率提升”。

文件：SUMMARY.json 汇总；SCORED_RUNS.json 逐轮答案与指标；runs.jsonl / raw/*.sse 为原始事件；dataset.json / source_manifest.json / wiki_pages.json 为数据及索引快照；三个 *_agent.json 为完整参数；native_* 为两次原生入口；METRIC_SANITY.log 为指标自检；VERIFICATION.txt 为测试和回滚记录；COMPARISON.md 为旧实验对比。
'''
(o/'REPORT.md').write_text(report)
comp='''# 原生题集与自建四类题集对比

## 结论

两套结果是互补证据，不应合并计算一个“提升”。本次原生样本是接入冒烟；此前自建四类题集更适合诊断单跳、局部综合、多跳和全局问题差异，但仍是人工构造的离线实验，不等于外部 benchmark 或真实用户效果。

| 维度 | 本次原生样本 | 2026-09-02 自建 four_class_v1 |
|---|---|---|
| 独立题数 | 1 | 16，四类各4题 |
| 数据规模 | 4段原文，自动生成21 Wiki页 | 12篇内部文档 |
| 正式运行数 | 9：1题×3模式×3重复 | 96：16题×3模式×2重复 |
| 参考标准 | 原生一句参考答案、4相关PID | 冻结的 required_fact_groups 与预期来源 |
| 主要生成指标 | 原生 BLEU/ROUGE | 全事实组匹配严格率、事实完整率 |
| 检索指标 | 原生API排名指标；Agent另算来源覆盖 | Agent来源覆盖、引用覆盖 |
| 公共模型 | text-embedding-v4 / qwen3.7-plus / qwen3-rerank | 见旧配置文件，不仅凭模型名判断同条件 |
| 解读 | 简单列举题的链路验证 | 多类题目的诊断及成本权衡 |

## 已有自建题集真实汇总（重新读取原报告）

| 模式 | 严格率 | 事实完整率 | 引用覆盖 | P50 秒 | 平均 tokens |
|---|---:|---:|---:|---:|---:|
'''
for m,v in old['overall'].items():comp+=f"| {m} | {v['strict_accuracy']:.2%} | {v['mean_completeness']:.2%} | {v['mean_citation_recall']:.2%} | {v['latency_ms']['p50']/1000:.2f} | {v['mean_total_tokens']:.1f} |\n"
comp+='\n## 本次原生题集结果（不同口径，禁止纵向当作提升）\n\n'+table([('ROUGE-L','rougel','.4f'),('五实体覆盖（非事实完整率）','reference_entity_coverage','.1%'),('来源覆盖','source_coverage','.1%'),('引用映射覆盖','citation_source_coverage','.1%'),('P50 毫秒','latency_p50_ms','.1f'),('平均 tokens','mean_total_tokens','.1f')])
comp+='''
## 为什么这些数字看起来可能冲突

1. 原生题目的五实体覆盖即使100%，也不等于旧评测严格率100%。旧严格率要求该题所有冻结事实组命中，题目更长、约束更多；本次五实体词面覆盖忽略额外错误陈述。
2. 旧 Wiki 多跳收益应与同题 RAG 比较，不能拿本次单题的 ROUGE 或短时耗时佐证或否认。全局/多跳在原生题集中没有独立覆盖。
3. 旧检索近饱和但严格率不足，说明“拿到源文档”与“答案表达全部要求”不同；本次 Wiki source_refs 覆盖也是代理指标，不是细粒度证据召回。
4. 两次的输入长度、索引内容、缓存命中、提示词及运行时间不同，跨实验 tokens/延迟只能描述负载差异，不给出优化收益百分比。
5. 旧自建题集有人工选题和关键词判分偏差，本次原生只有1题且参考答案过短；两者都不支持 GraphRAG 论文级总体胜率。重复运行不增加独立题目多样性。

## 使用建议

- 项目验收：原生样本用于 CI/冒烟与回归；此次发现的漏文档、异步索引竞态比单题分数更值得先修复。
- 项目选型：继续使用旧四类分层结果观察优势及代价，保留弱项，随后用未参与调参的独立题集验证。
- 简历：旧数据可写“自建离线四类题集”，本次写“复用原生评测完成回归并定位评测缺陷”；不要把两个实验的分数拼成一个进步曲线。

旧报告：docs/benchmarks/2026-09-02-four-class-benchmark/REPORT.md；原始96轮：benchmarks/results/2026-09-02-four-class-v1/formal/results.jsonl。对比所用旧 SUMMARY 的快照及 SHA256 随本目录保留。
'''
(o/'COMPARISON.md').write_text(comp);shutil.copy(r/'docs/benchmarks/2026-09-02-four-class-benchmark/SUMMARY.json',o/'PREVIOUS_SUMMARY.json')
for f in ['analyze.py','run_modes.py','write_reports.py','provision.py','agents_setup.py','fix_prompts.py','native_baseline.py','native_corrected.py']:
 shutil.copy(p/f,o/'artifacts'/f)
print('REPORTS_WRITTEN=2')
