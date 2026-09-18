# LoreLattice Benchmark 决策日志

- 测试日期：2026-08-25
- 基准分支：`newBee/benchmark-evidence`
- 基准代码：`f0eca422ed78f396d3f9a8c5dd26d0bf100028a7`
- 线上后端镜像：`sha256:8bc652f3780f6550d9a8075dd096e72a24adb77fe9c486a962b2b21aa9e34f7d`
- 目标：产生可复现、可解释、可用于简历且能经受追问的质量与性能数据。

## 决策记录

### D001：使用独立 worktree

- 决策：从本地 `main` 建立 `newBee/benchmark-evidence`，合并最新 Agent 输出修复后执行测试。
- 原因：主工作目录存在未提交的 ASR、文档和测试数据；直接测试或修改会混淆测试对象与个人改动。
- 备选：直接在当前工作目录执行。未采用，因为无法形成干净的 commit 到结果映射。
- 证据：基准 HEAD 为 `f0eca422ed78f396d3f9a8c5dd26d0bf100028a7`；原工作目录保持不变。

### D002：复制而不是移动企业文档

- 决策：把 12 份企业知识文档复制到版本化 benchmark 数据目录。
- 原因：保留原始文件及来源，测试数据可独立校验哈希并支持回滚。
- 数据目录：`benchmarks/datasets/enterprise_knowledge_v1/source_docs`。

### D003：测试分层

- 决策：测试分为现有代码回归、RAG 检索与回答质量、Agent 任务、HTTP/SSE 性能四层。
- 原因：单一分数无法同时证明正确性、检索质量、生成可信度和系统性能。
- 报告原则：任何简历数字必须带数据集规模、硬件、模型、并发、重复次数和 commit。

## 执行日志

后续每一步记录命令、输入、字面结果、退出状态、异常和修正。

### D004：先跑项目现有回归测试

- 决策：先执行 Go 全仓测试、CLI race/coverage/vet、前端 test/type-check/build、DocReader unittest，再进行模型与性能 benchmark。
- 原因：质量或性能数字必须建立在代码回归通过的版本上；否则 benchmark 可能测到已知功能错误。
- 命令范围：
  - `go test -count=1 ./...`
  - `cd cli && go test -race -coverprofile=coverage.out ./... && go vet ./... && go build ./...`
  - `npm ci && npm test && npm run type-check && npm run build-only`
  - DocReader 运行镜像挂载当前源码后执行 `python -m unittest discover -s docreader/tests -v`
- 环境快照：`benchmarks/results/2026-08-25/environment.json`。

### D005：保留首次失败，并纠正测试环境而不是删减测试

- 观察：首次 Go 全量测试失败，原因包含 `localhost:50051` 未映射、外部
  `LOG_FORMAT=json` 污染日志断言、SSRF 白名单在进程内缓存后未重新读取；首次
  DocReader 测试还发现 3 个二进制样本从未进入 Git 历史。
- 决策：保留首次失败日志；用临时本机 TCP 代理连接正在运行的 DocReader；在
  第二次 Go 测试中显式固定 `LOG_FORMAT` 和 `SSRF_WHITELIST`；从可审查脚本生成
  最小 DOCX/PPTX/PPT 样本，不下载来源不明的二进制文件。
- 边界：这些动作只修复测试前置条件，不修改被测业务逻辑。首次失败仍计入
  “测试基础设施缺口”，第二次结果用于判断当前代码回归状态。
- 生成器第一次执行：LibreOffice 成功退出但按输入文件名输出
  `en_marker.ppt`，脚本错误地只检查目标名 `en_38256.ppt`，因此退出 1。
  修正：验证 LibreOffice 实际输出后原子重命名到测试所需文件名，再次生成。
- 目标测试随后证明该 legacy PPT 测试不仅检查格式，还要求恰好提取 1 张图片；
  第一版最小演示文稿只有文字，得到 `0 != 1`。修正：生成器加入一张确定性
  PNG 图形再转换为 legacy PPT，继续保留这次失败作为样本语义缺口证据。

### D006：按测试语义拆分 SSRF 环境

- 第二次 Go 测试给所有包统一加入 loopback 白名单，修复了本地 HTTP 正向样本，
  但使 `web_fetch` 和 `web_search` 的 SSRF 负向测试失去意义并失败。
- 决策：不修改安全断言，也不把失败包排除。默认包在无白名单的干净环境运行；
  仅 `notion` connector 与 `docparser` 两个依赖本机 HTTP fixture、且读取一次缓存的
  包在进程启动时获得 `127.0.0.1,localhost`。包装脚本汇总两个阶段退出码。
- 含义：这是测试环境分层，不是用全局放宽安全策略换取绿色结果。
- 第三次运行进一步发现同一 Go package 内同时存在“允许本机 fixture”的正向测试
  和“拒绝 loopback”的负向测试，包级环境仍过粗；此外 `localhost` 被解析到
  IPv6 `::1`，而第一版代理只监听 IPv4。
- 最终修正：测试 helper 每次设置环境后显式重置生产代码的 `sync.Once` 白名单
  缓存，并在 cleanup 再重置；负向测试显式使用空白名单。代理同时监听
  `127.0.0.1` 与 `::1`。这样每个安全断言都保留原始语义。
- 主机无法直连 Docker Desktop 的 `172.20.0.4` bridge 地址，主机 Python 代理
  因上游连接失败退出。最终使用同一 DocReader 镜像启动临时代理容器，加入项目
  Docker network，并把 50051 同时发布到 IPv4/IPv6 主机；单独 client 测试结果
  为 2/2 通过。该临时容器在测试结束后删除。

### D007：RAG 数据集按独立事实与问法变体分别计数

- 决策：从 12 份合成企业制度文档人工定义 60 个有来源的事实组；每组提供
  canonical 与 paraphrase 两种问法，共 120 条查询，其中单文档 96 条、多文档
  24 条。简历和报告不得把它写成 120 个独立事实。
- 校验：120 个唯一 query ID、60 个 fact ID、每组 2 个变体；所有预期证据词
  都能在声明的源文档中找到，缺失数为 0。

### D008：检索通道做消融，不用单一 Recall 掩盖系统特征

- 决策：同一 120 条查询分别运行 hybrid、vector-only、keyword-only，Top-K
  固定为 8，并同时记录 Hit@K、来源 Recall@K、MRR、全部来源命中、证据词覆盖
  和延迟。并发固定为 4，仅用于缩短质量基准墙钟时间。
- 结果：三种模式 360 次请求均成功。Hybrid Hit@8 99.17%、MRR@8 93.13%；
  keyword-only Hit@8 100%、MRR@8 93.16%，且延迟明显更低。该 12 文档精确术语
  数据集偏向关键词检索，不能据此泛化到自然开放问答。

### D009：回答基准的 4 并发首轮视为稳定性探针，不覆盖为质量成绩

- 观察：首轮 120 次 KnowledgeQA 在 workers=4 时，前 65 条正常，随后 55 条
  出现无引用的固定英文拒答，其中 2 条还遇到 DashScope EOF；测试结束后单次
  探针又恢复正常。该时间分布符合突发并发触发上游/本地保护，而不是后半段
  问题全部不可回答。
- 决策：保留首轮原始结果作为并发稳定性证据；仅对错误或无引用的 55 条在
  workers=1、冷却后重跑，并与首轮正常条目合并为“稳定单路质量集”。不把重跑
  后结果冒充 4 并发稳定性结果。

### D010：人工逐答审定与自动字符串断言分开报告

- 观察：自动断言器按标点和字面字符串切分，无法把“三次”识别为“3次”，也会
  把“10:42–13:26”误判为未回答“2小时44分钟”；因此其低分主要是评测器假阴性。
- 决策：冻结题目、预期答案和来源声明后，由 Codex 逐条对 120 个最终回答做
  定性审定，并公开审定表；审定不是独立盲测人工标注，报告必须明确这一限制。
- 结果：118/120 完全通过，2 条部分通过。一条事实正确但生成了不存在的
  `example.com` 占位图片链接；另一条正确说明决议优先和转拨 20 万元，但未完整
  给出预期的 180/52/50 万元口径。120/120 都有相关来源引用，115/120 覆盖了
  数据集声明的全部来源；后一个指标会保守惩罚“单一文档已包含完整事实”的情况。

### D011：Agent 同时测“已知重复题”和“未知恢复题”

- 决策：使用已部署的 `builtin-smart-reasoning` 顺序运行同一事实题 20 次，再运行
  5 个知识库中不存在的 Zephyr-99 预算问题；保留 verbose 事件流，按最终 answer
  event 分组检查正确事实、重复段落、工具调用与错误。
- 结果：20/20 包含 286 人且最终段落无重复，5/5 明确未找到。共观察到 79 个工具
  调用，投影流返回 77 个 tool_result，返回的 77 个均无 error；两个
  `get_document_info` 缺少配对结果。25 次中 1 次耗时 128.6 秒。
- 解释：重复输出修复在该场景有效；但内置 Agent 的 `max_iterations=50` 和
  `llm_call_timeout=120` 没有落实治理文档中的总计 12 次/90 秒门控，长尾不能隐藏。

### D012：把无节流压力失败与受控吞吐分开

- 首次 k6 使用 1/10/50 VU 且不加 pacing，52,070 次请求中出现 23,839 次 502。
  Nginx 日志显示 `connect() ... failed (99: Address not available)`，不是应用返回
  业务错误，而是本机代理到上游的新连接压垮临时端口/地址资源。
- 决策：保留该次为 stress-to-failure；另用 constant-arrival-rate 运行
  10/100/300 target RPS 各 15 秒，作为可解释的受控容量样本。
- 结果：完成 6,134 次 HTTP 200，18 次由 k6 调度丢弃，P95 6.42 ms；资源峰值为
  App 43.34% CPU / 234.5 MiB、Frontend 5.89% / 9.77 MiB、Postgres 16.15% /
  91.93 MiB、Redis 1.33% / 21.39 MiB。

### D013：只清理本轮创建的会话

- 决策：从所有 JSON/JSONL 中递归提取字段名严格等于 `session_id` 的 UUID，与
  `session list --since 2d` 交集核对，并先执行 delete dry-run；不按时间范围或标题
  模糊删除，避免碰到用户原有会话。
- 结果：收集并命中 202/202 个 benchmark 会话，批量删除返回 successes=202、
  failures=0；删除后近两天仍有 5 个原有会话，benchmark_remaining=0。
