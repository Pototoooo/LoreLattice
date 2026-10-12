# 主应用质量门禁

`.github/workflows/app-ci.yml` 在每个 PR、main push 和手动触发时执行：

- Go 主应用、独立 client 模块和 DocReader Go 客户端回归，关键模块竞态检测及 API 构建。
  按现有 regression runner 清理 LOG_FORMAT/SSRF 白名单环境污染；不关闭 SSRF 防护。
  DocReader Go 客户端默认使用本地 gRPC fixture；真实服务测试需 DOCREADER_TEST_ADDR。
- 独立 PostgreSQL 17 服务：提供 `BILLING_TEST_POSTGRES_DSN`，不使用生产凭证。
  钱包测试创建唯一 schema，并在退出时清理，验证并发预留和支付幂等性。
- Node 24：`npm ci`，测试、类型检查及生产构建；普通 build 仍可单独使用。
- Python 3.12 + DocReader uv.lock：解析器、SSRF 等 unittest。

应在仓库 branch protection 中将 backend/frontend/docreader 设为 required checks。
新增工作流不会自动更改 GitHub 保护规则。离线 fixture 和接口 mock 测试不等于
真实模型端到端评测，现有 CLI E2E 仍需显式配置测试服务凭证并触发。

开发环境：Go 版本以 go.mod 为准，Node 最低 22.19。macOS SDK 与 linker
不兼容属于本机工具链问题，应匹配 Xcode/CommandLineTools 版本，不能通过
关闭 CGO 绕过 pg_query、SQLite、Jieba 等实际依赖。
