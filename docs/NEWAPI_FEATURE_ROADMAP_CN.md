# AySub NewAPI 功能融合路线

本文档记录 AySub 当前已融合的 NewAPI 风格功能、实现边界和剩余生产验收项。它是实现进度与验收口径文档，不替代 README、部署文档或支付配置文档。

## 当前结论

NewAPI 全量融合计划中的六个阶段已在当前代码库落地：

- API Key 级模型与端点权限。
- 请求日志中心与运营报表。
- 模型价格计算器与渠道策略可视化。
- 体验中心视频与音频能力。
- 公开状态页。
- 支付生产验收可观测性与文档收口。

剩余事项不是“入口未实现”，而是生产侧真实账号、真实商户和回调链路验收：

- 真实 xAI/Grok Cookie/Console/媒体/LiveKit 账号链路逐项验收。
- 明确支持音频/视频的 OpenAI-compatible 上游账号逐项验收。
- EasyPay、支付宝、微信、Stripe 等真实商户配置、回调和退款链路验收。

## 已完成阶段

### Phase 1：API Key 级模型与端点权限

已实现能力：

- `api_keys` 增加 `allowed_models`、`allowed_endpoints`、`permission_mode`、`permission_updated_at`。
- 空配置继承当前分组/渠道能力；`restrict` 模式按显式白名单严格限制。
- 网关入口统一执行后端权限校验，覆盖 Chat Completions、Responses、Messages、Embeddings、Images、Videos、Audio、LiveKit、Gemini 原生端点。
- 用户侧 API Key 页面支持查看和编辑模型/端点权限。
- 管理侧用户 Key 弹窗支持管理员覆盖权限配置。
- 权限拒绝由后端返回协议兼容错误，并进入 ops/business limited 日志路径。

主要入口：

- 用户侧：`/keys`
- 管理侧：用户管理中的 API Key 弹窗

### Phase 2：请求日志中心与运营报表

已实现能力：

- 基于现有 `usage_logs` 与 `ops_error_logs` 提供统一请求日志视图。
- 请求日志字段包含状态、错误、渠道/账号、端点、请求模型、上游模型、耗时、TTFT、成本、IP、User-Agent 等摘要字段。
- 用户侧展示自己的成功/失败请求。
- 管理侧展示全站请求，支持 API Key、用户、模型、渠道、端点、状态、时间、错误类型等筛选。
- 管理侧运营报表包含请求趋势、错误趋势、模型成本排行、渠道健康排行、用户消费排行、Key 消费排行。
- 不记录完整敏感请求体，只保存 request id、hash、摘要字段和错误摘要。

主要入口：

- 用户侧：`/request-logs`
- 管理侧：请求日志中心

### Phase 3：模型价格计算器与渠道策略可视化

已实现能力：

- 用户侧 `/models` 模型广场按平台、模型、渠道、分组、价格聚合展示能力。
- 模型广场新增价格计算器，可按模型、分组、渠道、计费模式输入 token、图片张数或请求次数，估算原价、倍率后价格、最低/最高渠道价格。
- 管理后台渠道页新增策略视图，展示渠道到分组、模型映射、定价、启停、restrict models、健康状态、最近错误、请求量和成本。
- 复用现有 `account_groups.priority`、账号优先级、渠道模型映射和渠道定价，不引入第二套路由配置数据源。
- 支持批量启停渠道、批量更新模型定价、批量复制渠道策略。

主要入口：

- 用户侧：`/models`
- 管理侧：渠道管理

### Phase 4：体验中心完整化

已实现能力：

- `/playground` 支持聊天与图片生成。
- 视频体验支持 `/v1/videos` 提交任务、`/v1/videos/{id}` 轮询任务、展示状态/失败原因、预览与 `/content` 下载。
- 音频体验支持 OpenAI-compatible REST API：
  - `/v1/audio/speech`
  - `/v1/audio/transcriptions`
  - `/v1/audio/translations`
- 音频只路由到明确支持音频的 OpenAI-compatible 账号/渠道；不支持的平台返回协议兼容错误。
- 体验中心所有 Tab 都使用用户自己的 API Key，受 API Key 权限控制，成功/失败进入日志和计费体系。

主要入口：

- `/playground`
- `/experience`

### Phase 5：公开状态页

已实现能力：

- 新增公开状态页 `/status`。
- 后端提供匿名只读 public API。
- 页面只展示脱敏信息：系统状态、可用模型数量、公开渠道健康摘要、最近 24 小时错误率、延迟区间和近期事件摘要。
- 后台设置支持开关：公开状态页启用、展示模型、展示渠道摘要、展示最近事件。
- 不返回账号、Cookie、真实渠道密钥、内部错误栈或专属分组明细。

主要入口：

- 匿名页面：`/status`
- 管理侧：系统设置中的公开状态页配置

### Phase 6：支付生产验收与文档收口

已实现能力：

- 对订单、套餐、支付实例、回调、退款/关闭订单路径补充运营可观测指标。
- 支付看板新增生产验收指标：
  - `callback_failures`
  - `order_inconsistencies`
  - `provider_unavailable`
  - `refund_requested`
  - `refunding`
  - `refund_failed`
  - `refunded`
  - `fulfillment_failed`
  - `paid_not_completed`
  - `stale_pending`
  - 支付实例启用、禁用、退款启用、用户自助退款启用数量
- 已更新 README 三语文档，按当前视频/音频已接入状态同步说明。
- 新增支付生产验收清单：`docs/PAYMENT_PRODUCTION_ACCEPTANCE_CN.md`。

生产侧仍需验收：

- 使用真实商户号与真实回调地址验证 EasyPay、支付宝、微信、Stripe。
- 使用真实订单验证支付成功、超时关闭、补单、退款、退款失败、回调重复投递、金额/服务商不一致等路径。

## 移动端适配口径

NewAPI 移植页面不照搬 NewAPI 原始桌面宽表布局。当前实现遵循 AySub/Sub2API 现有响应式方式：

- 桌面端可保留高密度表格和策略视图。
- 手机端使用卡片列表、分组信息块和自然流式表单。
- 请求日志、模型广场、渠道策略、运营报表排行、体验中心和 Key 权限表单均避免固定宽度导致的小屏横向溢出。

## 验收标准

- 前后端行为一致，不出现前端显示允许但后端未校验的情况。
- 不新增重复数据源，优先复用现有渠道、模型、分组、价格和用量体系。
- 权限、计费、日志类能力以后端为准，前端只做展示和配置。
- 不记录完整敏感请求体，不在公开状态页暴露账号、Cookie、密钥或内部错误栈。
- 用户侧与管理侧入口清晰，桌面端和移动端均可操作。
- 代码验证至少覆盖后端目标测试、前端构建和必要的集成烟测。

## 当前验证记录

- `cd backend && go test ./internal/service -run TestPaymentDashboardStatsIncludesOpsMetrics -count=1`
- `cd backend && go test ./internal/service ./internal/handler ./internal/server/routes`
- `pnpm --dir frontend build`

完整回归仍建议在发布前执行：

- `cd backend && go test ./...`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`
- Docker 本地构建与服务器 compose 重启烟测
- `/models`、`/playground`、`/status`、`/v1/audio/*`、`/v1/videos` 烟测
