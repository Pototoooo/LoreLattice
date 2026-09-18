# 原生数据集 RAG / Wiki / Hybrid 评测报告

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

| 指标 | RAG | Wiki | Hybrid |
|---|---:|---:|---:|
| 完成率 | 100.0% | 100.0% | 100.0% |
| BLEU-1 | 0.0347 | 0.0351 | 0.0429 |
| BLEU-2 | 0.0195 | 0.0242 | 0.0242 |
| BLEU-4 | 0.0089 | 0.0111 | 0.0123 |
| ROUGE-1 | 0.1400 | 0.1283 | 0.1416 |
| ROUGE-2 | 0.0270 | 0.0360 | 0.0263 |
| ROUGE-L | 0.0928 | 0.1090 | 0.1193 |
| 参考答案五实体覆盖率（补充） | 100.0% | 100.0% | 93.3% |
| 五实体全覆盖运行占比（补充） | 100.0% | 100.0% | 66.7% |
| 源文档覆盖率（补充） | 100.0% | 100.0% | 91.7% |
| 引用映射源文档覆盖率（补充） | 100.0% | 100.0% | 58.3% |
| 端到端 P50 毫秒 | 19920.8 | 23208.4 | 24614.6 |
| 平均累计 tokens | 29908.0 | 41440.0 | 68573.7 |
| 平均工具调用数 | 6.33 | 3.33 | 5.33 |
| 进入 knowledge_search 的次数 / 3 | 3 | 0 | 0 |
| 页面歧义事件数 | 0 | 7 | 5 |
| 工具显式错误数 | 0 | 0 | 0 |

### 相对 RAG 的变化（仅本题）

| 模式 | ROUGE-L 差值 | P50 耗时变化 | 平均 tokens 变化 |
|---|---:|---:|---:|
| wiki | +0.0162 | +16.5% | +38.6% |
| hybrid | +0.0265 | +23.6% | +129.3% |

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

## 最终验收状态

9/9 正式运行完整，三份配置仅 Agent 类型、工具集合、提示词内容/ID 不同；原生数据与主工作区源代码哈希未变。临时评测容器已停止，原服务健康。测试知识库、Agent 和会话保留供查看；修复仍只在隔离副本，不自动合入主分支。回滚在另一个副本上验证，恢复后重现 PID4 遗漏，修复副本保持不变。详见 VERIFICATION.txt。

本次 Hybrid 第2次明确漏掉 Mac OS；五实体覆盖低于 RAG/Wiki，但 ROUGE-L 反而更高，说明词面相似度排名不能替代完整性判定。
