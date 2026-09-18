# Wiki 导航修复与 RAG / Wiki / Hybrid 控制变量评测

## 结论

Wiki Summary 导航已修复。根因不是 Wiki 检索能力本身，而是 `dN` 同时承担“源文档句柄”和“Summary 页面 slug 别名”，在多轮 ReAct 中发生碰撞。修复后改为两个独立命名空间：源文档继续使用 `dN`，Wiki Summary 页面使用 `pN`；工具执行前把 `summary/pN` 还原为真实页面 slug，最终回答再展开为可点击的公开链接。

完整 v2 评测共 12 道 Wiki 优势题 × 3 种模式 = 36 次调用，36/36 正常完成。Wiki 的导航成功率为 100%（12/12），Summary 导航子集为 100%（2/2），且未再出现 `Wiki page 'summary/dN' not found`。Hybrid 的语义成功率同为 100%，但有 2 题自主选择纯 RAG 路径、没有访问 Wiki 锚点，因此“Wiki 导航成功率”为 83.3%；这两题的答案本身均通过语义判定。

## 控制变量

- 数据集：`wiki_strength_v1`，12 题；覆盖概念关系、实体网络、全局概览导航、Summary 导航。
- 模型：三种模式均绑定同一 LLM：`cb425145-111f-4150-a02d-f5a67ca61bd4`。
- Rerank：三种 Agent 均绑定同一 rerank 模型 `266160fe-ec66-42de-a138-1a2950c1d867`，`top_k=10`，`threshold=0.3`。
- 执行：单 worker，按 Latin-square 方式交错模式顺序，避免总是由同一模式先运行。
- 判定：语义成功 = 正常完成 + 冻结关键词组全部命中 + 无重复最终段落；Wiki 导航成功 = 语义成功 + 命中预设 Wiki 锚点 + 无工具错误。
- 说明：rerank 仅在 Agent 实际调用语义检索路径时生效；Wiki 的 `wiki_search/wiki_read_page` 路径本身不经过 RAG rerank。RAG 有 4/12、Hybrid 有 2/12 调用了可进入 rerank 的 `knowledge_search` 路径。

## v2 正式结果

| 指标 | RAG | Wiki | Hybrid | Wiki vs RAG | Hybrid vs RAG |
|---|---:|---:|---:|---:|---:|
| 执行完成率 | 100% | 100% | 100% | 0 pp | 0 pp |
| 语义成功率 | 91.7% | 100% | 100% | +8.3 pp | +8.3 pp |
| 证据词组召回 | 98.61% | 100% | 100% | +1.39 pp | +1.39 pp |
| 工具无错误率 | 91.7% | 100% | 100% | +8.3 pp | +8.3 pp |
| Wiki 锚点命中率 | N/A | 100% | 83.3% | N/A | N/A |
| Wiki 导航成功率 | N/A | 100% | 83.3% | N/A | N/A |
| 平均工具调用数 | 3.42 | 3.25 | 4.17 | -4.9% | +22.0% |
| P95 延迟 | 41.55 s | 47.44 s | 44.48 s | +14.2% | +7.0% |

### 分类别结果

- 概念关系（4 题）：三种模式语义成功率均 100%；Wiki 导航 4/4，Hybrid 2/4。Hybrid 在另外 2 题选择纯 RAG 路径。
- 实体网络（4 题）：RAG 3/4，Wiki 4/4，Hybrid 4/4；Wiki/Hybrid 在角色关系题上比 RAG 多覆盖一个冻结证据组。
- 全局概览导航（2 题）：三种模式语义均 2/2；Wiki/Hybrid 导航均 2/2。
- Summary 导航（2 题）：三种模式语义均 2/2；Wiki/Hybrid 导航均 2/2，无 Summary slug 错误。

## 对“为什么此前 Wiki/Hybrid 弱于 RAG”的修正判断

首轮数据里 Wiki/Hybrid 弱，主要是导航缺陷和测试集错配叠加，并不能推出 Wiki 架构天然更弱。使用关系型、跨页面和导航型问题后，Wiki 在语义成功率上达到 100%，并以更少的平均工具调用完成结构化导航；代价是 P95 延迟比 RAG 高 14.2%。Hybrid 的优势是答案成功率稳定，但路由不保证一定访问 Wiki，因此不能把“未访问 Wiki 锚点”直接解释成答案失败。

## 可写与不可写的量化结论

可复现地写：在 12 题、36 次离线控制变量评测中，Wiki/Hybrid 语义成功率为 100%，相较 RAG 提升 8.3 个百分点；Wiki 导航成功率 100%，Summary 导航 2/2 无错误。

暂不宜外推：真实用户准确率、线上业务提升、普适性的“Wiki 比 RAG 提升 8.3%”。当前样本只有 12 题且每题每模式只运行一次，未估计随机波动和置信区间；应将结论限定为该冻结测试集。

## 证据入口

- 原始 36 行结果：`benchmarks/results/2026-09-02-wiki-strength-v2/formal/results.jsonl`
- 聚合结果：`docs/benchmarks/2026-09-02-wiki-navigation-v2/SUMMARY.json`
- 正式原始结果：`benchmarks/results/2026-09-02-wiki-strength-v2/formal/results.jsonl`
