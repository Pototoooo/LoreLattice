# RAG / Wiki / Hybrid 四类控制变量评测方法

- 语料：`testdata/wiki_company_internal/` 的 12 篇冻结内部知识文档。
- 题集：16 题；单点事实、局部综合、多跳、全局主题各 4 题；每题每模式重复 2 次，共 96 次。
- 控制变量：同一知识库、同一 `qwen3.7-plus` 模型、同一 `qwen3-rerank`、temperature=0、embedding_top_k=10、rerank_top_k=10、rerank_threshold=0.3、同一容器和时间窗口。
- 自变量：Agent 类型与其原生工具集，分别为 `rag-qa`、`wiki-qa`、`hybrid-rag-wiki`。运行顺序按题号和重复轮次轮转，降低固定顺序偏差。
- Rerank 边界：三个 Agent 均绑定相同 Rerank 模型；`knowledge_search` 执行时进入 Rerank 路径。纯 Wiki 的 `wiki_search/wiki_read_page` 不经过 Chunk Rerank，因此报告另列实际 Rerank 路径命中率，而不把“已配置”误写成“每次都执行”。

## 指标

- Strict accuracy：最终答案非空、执行完整、无重复最终段落，且该题全部冻结事实组均命中。
- Fact completeness：命中的冻结事实组数 / 该题全部事实组数。
- Retrieval recall：实际读取到的期望源文档数 / 该题期望源文档数；Wiki 页面通过冻结 page-to-source 映射归一到源文档。
- Citation recall：最终答案引用覆盖的期望源文档数 / 该题期望源文档数。
- Token cost：服务端完成事件汇总的所有 Agent LLM 轮次 prompt/completion/total token，而非只统计最后一轮。
- Latency：CLI 发出请求到完整 SSE 结束的端到端耗时，报告 P50/P95。
- Pairwise comprehensiveness：同题、同轮下，以 fact completeness 比较 Wiki/Hybrid 与 RAG，报告胜/平/负、全比较胜率、平均百分点差和 bootstrap 95% CI。

## 可解释边界

- 这是一套专门覆盖层级导航、跨文档、多跳、全局综合的内部 Wiki 强项题集，不代表开放域总体分布。
- 事实组采用冻结字符串变体做确定性评分，可复现但可能把语义正确、措辞不同的答案计为未命中，因此 Strict accuracy 是保守指标。
- 每个 category/mode 只有 8 个观测，分类结果适合做项目回归和简历证据，不宜外推为普适产品结论。
