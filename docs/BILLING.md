# 余额与真实支付

LoreLattice 按 Workspace 共享余额。平台模型按调用扣费，自备 Key / 本地模型只记用量。没有 Trial / Pro、Token 套餐上限、月费或续订。旧账户的套餐字段只保留用于审计，不参与放行判断。

## 首期支付范围

已实现支付宝**电脑网站支付、公钥模式、RSA2、正式网关**，充值固定为 ¥10、¥50、¥100。需签约 `FAST_INSTANT_TRADE_PAY` 对应产品。已有支付宝商户账号不一定代表该应用已开通电脑网站支付。

订单创建、支付结果通知、主动查单、重复通知幂等、工作空间隔离都由服务端完成。浏览器仅打开支付宝页面并轮询本地订单；同步返回页、客户端传入状态、普通充值请求都不能增加余额。微信支付、手机网站支付、自动退款和税务发票不在首期范围。不要把充值记录当成税务发票。

接口：

- `GET /api/v1/billing/overview`：钱包状态。
- `GET /api/v1/billing/usage?view=jobs`：业务消费记录；默认 view 为模型明细。
- `POST /api/v1/billing/payments`：Owner 创建订单，参数 `amount_fen`、UUID `idempotency_key`。
- `GET /api/v1/billing/payments`：Owner 读取最近 50 笔订单。
- `GET /api/v1/billing/payments/:id`：Owner 检查订单，可恢复待付款链接。
- `POST /api/v1/billing/payments/alipay/notify`：公开回调，使用支付宝签名认证，不使用用户 JWT。

旧 `/top-ups`、`/plans`、`/invoices` 和 `/subscription/*` 路由已移除，不能再直接模拟加钱。财务接口在 RBAC 观察模式下仍强制 Owner 权限，API Key 不允许操作。

## 人民币和历史账本

历史账本、模型单价仍以 USD 为内部单位，避免给原始金额换标签。页面统一显示人民币，首次使用时将 `BILLING_CNY_PER_USD` 保存到 `billing_wallet_settings`。默认 **7** 是可配置的商业计价系数，不是实时汇率或汇率建议。上线前应确定该值；首次写入后，更改环境变量不会重估历史余额。不要直接修改这张表来调价；调整新模型销售单价即可。

充值订单保存整数人民币分、计价系数和折算后的内部入账金额，后者向下保留 8 位小数。支付确认必须同时匹配订单、金额、app_id、seller_id。外部交易号全局唯一，订单状态、充值流水、余额增加在同一事务提交。

迁移 **不删除历史资金或模拟充值**。已有 `sandbox_topup` 会作为旧余额保留，因此上线前必须核对并决定是否将其视为赠金；不要把历史模拟入账统计成真实营业收入。旧 Trial / Pro 订阅在远端仍可能存在，切换前需在 MeterForge 停用这些旧订阅/计费任务；新代码不会自动取消或修改外部历史账单。

新工作空间默认赠金为 0，`BILLING_WELCOME_CREDIT_USD` 可设置体验额度；配置赠金前应配合工作空间创建权限，避免重复领用。BYOK 和本地模型不需要余额。内置远程模型默认识别为平台付费，即使配置了 API Key；用户创建且携带自己的 Key 的模型默认 BYOK。管理员可通过模型 `extra_config.billing_mode` 明确覆盖。旧 `included` 配置在新调用中按平台模式处理。

## 配置与部署

将商户私钥和支付宝公钥放在仓库忽略的 `data/payment-secrets/`，文件限制为运行服务账户可读。默认 Compose 将此目录只读挂载到 `/run/payment-secrets`。这里只支持 PEM 公钥模式；使用证书模式的应用需另行适配，不能将证书内容当公钥文件。

```dotenv
BILLING_ENABLED=true
BILLING_CNY_PER_USD=7
BILLING_WELCOME_CREDIT_USD=0
BILLING_RESERVATION_TTL=24h
ALIPAY_ENABLED=true
ALIPAY_APP_ID=你的应用ID
ALIPAY_SELLER_ID=你的支付宝商户PID
ALIPAY_PRIVATE_KEY_FILE=/run/payment-secrets/merchant-private.pem
ALIPAY_PUBLIC_KEY_FILE=/run/payment-secrets/alipay-public.pem
ALIPAY_NOTIFY_URL=https://你的域名/api/v1/billing/payments/alipay/notify
ALIPAY_RETURN_URL=https://你的域名/platform/settings?section=billing
METERFORGE_ENABLED=false
```

回调地址必须通过公网 HTTPS 到达 app；不能被登录页、WAF 验证页或前端 SPA 回退页替代。签名验证失败返回 `failure`；事务成功后返回纯文本 `success`。商户私钥用于签请求，**支付宝公钥**用于验通知和查询响应，两者不同。

更新代码后运行数据库迁移至 `000077`，构建后端与前端并重建 app 容器。使用 work 环境时，必须同时指定正确 Compose 文件和插值来源，例如 `docker compose --env-file .env.work -f docker-compose.work.yml ...`；只改 `.env.work` 不会更新既有容器。启动后检查容器实际 `BILLING_ENABLED`，不能只看文件。

支付宝未配置时页面明确显示充值未开放，不回退到模拟支付；错误价格 JSON 会阻止调用，不会默默改用其他售价。上线前需要在已签约的正式应用完成一笔实际充值验收：确认支付宝交易、订单 `paid`、一条充值流水及余额增加一致，然后重复投递通知验证不重复入账。私钥不要发到聊天或提交到 Git。

## 结算与故障处理

本地余额是唯一放行依据：已预留、已完成及恢复暂结的费用都会计入占用，PostgreSQL 对账户行加锁，防止并发透支。每次调用保留单价版本，实际扣款不超过调用前授权的预留金额；供应商实际用量高于估算时，超出预留的成本由平台承担。上线应设置合适的最大输出量和模型售价。

流式结算失败会有界重试并记录错误。进程中断遗留预留超过默认 24 小时后按预留金额暂结，标记 `recovered` / 页面“暂结待核查”，不把无法证明免费的调用自动退款。迟到的实际结算或失败释放可纠正暂结；运营需检查这些记录并按供应商证据处理。

支付通知丢失时每分钟主动查待确认订单，支付窗口结束后每小时复查；本地超时不代表未付款，只有验签通过的支付成功才入账。可查看 `billing_payment_orders.last_error`。换商户配置时，旧订单需要原商户配置继续对账。首期没有自动退款接口；如在支付宝侧人工退款，必须同时人工核对本地余额，不能只退外部资金。

MeterForge 可选开启，只接收新事件 `lorelattice.wallet.usage.v1`、subject `tenant:wallet:<id>`。旧定价事件不会重发，新事件不得关联旧套餐价格。可在 MeterForge 建立基于 `data.quantity`、`data.feature_key`、`data.cost_usd` 的分析视图；它不再负责钱包扣费或用量额度。旧 pending 事件标记为 `legacy` 留待核查。Outbox 稳定 event ID、重试与本地账本不受远端可用性影响。

## 验证

```sh
go test ./internal/billing ./internal/models/chat ./internal/models/embedding ./internal/models/rerank ./internal/models/asr ./internal/models/vlm ./internal/router
# 仅对可丢弃的 PostgreSQL 测试库运行；测试建立独立 schema 并清理。
BILLING_TEST_POSTGRES_DSN='host=127.0.0.1 port=5432 user=postgres password=... dbname=... sslmode=disable' go test ./internal/billing -run TestPostgresWalletConcurrency -v
cd frontend
npm run type-check
npm run build
```

覆盖支付签名与金额篡改、重复参数、重复通知、未支付不入账、事务回滚、跨空间访问、迟到通知、签名查单、系数锁定、并发预留、预留恢复和独立遥测命名空间。自动测试使用临时密钥和模拟支付宝 HTTP 服务，没有发生真实交易。正式网关联调需要部署方提供商户配置和公网域名。

官方协议依据：[电脑网站支付接入](https://aipay.alipay.com/docs/vibe-pay/ai-web-app-payment-qianyi/ai-web-app-payment-integration-guide.html)、[异步通知验签和业务校验](https://aipay.alipay.com/docs/ai-web-app-payment-qianyi/api-list/async-notify-verify.html)、[统一收单查询](https://developer.alibaba.com/docs/doc.htm?articleId=757&docType=4&treeId=180)。
