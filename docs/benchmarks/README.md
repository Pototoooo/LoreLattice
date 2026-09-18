# Benchmark evidence index

当前保留两组可复核交付，旧的 smoke、失败重试和中间产物未合入主工作树。

## 推荐入口

1. **四类 RAG / Wiki / Hybrid 正式评测**
   - 中文结论：[`2026-09-02-four-class-benchmark/FINDINGS_CN.md`](2026-09-02-four-class-benchmark/FINDINGS_CN.md)
   - 指标总表：[`2026-09-02-four-class-benchmark/REPORT.md`](2026-09-02-four-class-benchmark/REPORT.md)
   - 方法说明：[`2026-09-02-four-class-benchmark/METHODOLOGY.md`](2026-09-02-four-class-benchmark/METHODOLOGY.md)
   - 数据验证：[`2026-09-02-four-class-benchmark/VALIDATION_REPORT.md`](2026-09-02-four-class-benchmark/VALIDATION_REPORT.md)
   - 正式原始结果：[`../../benchmarks/results/2026-09-02-four-class-v1/formal/results.jsonl`](../../benchmarks/results/2026-09-02-four-class-v1/formal/results.jsonl)

2. **Wiki 导航修复回归**
   - 报告：[`2026-09-02-wiki-navigation-v2/WIKI_NAVIGATION_REPORT.md`](2026-09-02-wiki-navigation-v2/WIKI_NAVIGATION_REPORT.md)
   - 指标：[`2026-09-02-wiki-navigation-v2/METRICS.csv`](2026-09-02-wiki-navigation-v2/METRICS.csv)
   - 正式原始结果：[`../../benchmarks/results/2026-09-02-wiki-strength-v2/formal/results.jsonl`](../../benchmarks/results/2026-09-02-wiki-strength-v2/formal/results.jsonl)

## 目录约定

- `benchmarks/datasets/`：冻结题集和映射。
- `benchmarks/scripts/`：生成、运行、汇总和验证脚本。
- `benchmarks/results/`：仅保留正式运行原始数据。
- `docs/benchmarks/`：面向阅读和简历取证的汇总材料。
