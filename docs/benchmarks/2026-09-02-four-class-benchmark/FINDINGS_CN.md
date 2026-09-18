# LoreLattice RAG / Wiki / Hybrid 四类评测结论

## 结论先行

这次结果**部分符合预期**：Wiki 在 multi-hop 上确实建立了可复核优势，但当前实现没有复现 GraphRAG 式 global search 的优势。Hybrid 的总体完整性与严格准确率最高，但 token 和工具调用成本明显偏高；RAG 在单点事实和局部综合上已经触顶，仍是更经济的默认路径。

## 可用于项目描述的核心数字

- **Multi-hop（8 个配对观测）**：Wiki strict accuracy 62.5%，RAG 37.5%，即 **+25.0 pp**；事实完整度 94.09% 对 83.75%，即 **+10.34 pp**。同题配对为 **3 胜 / 5 平 / 0 负**，平均完整度差的 bootstrap 95% CI 为 **[+1.14, +26.36] pp**。
- **证据引用**：Wiki 总体 citation recall 88.80%，RAG 64.30%，即 **+24.51 pp**。
- **Hybrid 总体**：strict accuracy 71.88%，RAG 68.75%，即 **+3.12 pp**；完整度 +2.40 pp，但 95% CI 跨 0，不能写成已证实的普适提升。
- **代价**：Hybrid 平均 token 115,950，较 RAG 68,733 **增加 68.7%**；P50 延迟 38.55s，较 RAG 27.41s **增加 40.6%**。在 multi-hop 上 Hybrid token 更是 **增加 162.6%**。

## 分类结果

| 类型 | RAG strict / 完整度 | Wiki strict / 完整度 | Hybrid strict / 完整度 | 判断 |
|---|---:|---:|---:|---|
| 单点事实 | 100% / 100% | 100% / 100% | 100% / 100% | 三者触顶，没有质量提升空间；RAG 最省 |
| 局部综合 | 100% / 100% | 100% / 100% | 100% / 100% | 三者触顶；Wiki 引用覆盖最好 |
| Multi-hop | 37.5% / 83.75% | **62.5% / 94.09%** | 50.0% / 93.18% | Wiki 优势最清楚 |
| Global / thematic | 37.5% / 89.76% | 0% / 84.74% | 37.5% / 89.94% | 当前 Wiki global 能力不足 |

Global 的 Wiki strict accuracy 为 0% 不等于答案完全错误：其平均事实完整度仍为 84.74%，只是 8 次运行每次至少漏掉一个冻结事实组。重复缺失包括 ARR/NRR 量化目标、移动端承诺覆盖关系，以及若干负责人/风险动作。

## 为什么没有达到“GraphRAG 全局问题 70–80% 胜率”

当前 Wiki 主要提供链接页、实体页、概念页和源文档摘要的局部导航；它没有 GraphRAG 的 community detection、分层 community summary 和面向全语料的 map-reduce global search。结构化导航能帮助跨文档跳转，但不自动等于全局聚合。因此本次 global 配对完整度：Wiki 对 RAG 为 **2 胜 / 2 平 / 4 负**（全比较胜率 25%，非平局胜率 33.3%），Hybrid 为 **2 胜 / 4 平 / 2 负**。

## Rerank 与异常

三个 Agent 均绑定同一 `qwen3-rerank`，但 Rerank 只在 `knowledge_search` 实际执行时进入路径：RAG 为 11/32 次运行，Hybrid 为 2/32，纯 Wiki 为 0/32。不能把“Agent 已绑定 Rerank”写成“所有请求都经过 Rerank”。

正式结果文件 96 行均退出为 0、均完成、均有答案和非零 token。运行期间的 DNS/EOF 环境失败被隔离到 `results.failed-*.jsonl`，未进入正式汇总。Hybrid 在 M02 两轮都出现了无效文档别名引发的工具错误，说明 Wiki→Chunk 回退仍需增加 ID 类型校验；这也解释了其 token 尖峰和 tool-error-free rate 93.75%。

## 简历推荐写法

> 设计 16 题四类分层评测集，在固定知识库、LLM、Rerank、Top-K 与温度的 96 次对照实验中，验证 Wiki Agent 在 multi-hop 问答上的严格准确率由 37.5% 提升至 62.5%（+25 pp）、事实完整度提升 10.34 pp，并将引用召回率由 64.3% 提升至 88.8%；同时定位 Hybrid 回退链路的 ID 校验与 token 放大问题。

简历上不要写“全局问题提升 70–80%”或“Wiki 全面优于 RAG”；这两条与本项目实测不符。
