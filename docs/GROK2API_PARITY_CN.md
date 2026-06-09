# AySub Grok2API Parity 路线

本文档记录 AySub 当前对 Grok2API / xAI / Grok Web 能力的兼容进度、实现边界和未验收事项。它用于约束 README 中 Grok/xAI 相关公开声明，避免把未完成或未生产验收的功能写成已完成。

## 当前结论

当前代码库已实现 AySub 内部的 Grok/xAI 适配能力：

- xAI/Grok 账号作为 AySub 账号池的一部分参与调度、计费、日志和错误处理。
- 支持 Grok 反代聊天、Console/Reverse 相关服务和用量解析。
- 支持 Grok Cookie 请求的 FlareSolverr / WARP 辅助部署说明。
- 支持可选动态 `x-statsig-id` 兼容头，可通过全局配置或账号级配置启用。
- 网关层已接入 OpenAI-compatible、Responses、Images、Videos、Audio、LiveKit 等统一入口。

README 不把以下内容写成已完成：

- Grok2API Masonry 画廊完整 parity。
- ChatKit 语音 UI 完整 parity。
- 真实 xAI/Grok Cookie、Console、媒体和 LiveKit 账号的生产级逐项验收。
- 对所有 Grok Web 前端私有协议变动的长期稳定承诺。

## 已实现能力

### Grok/xAI 账号与调度

已实现能力：

- xAI/Grok 账号纳入统一账号池。
- 支持账号分组、优先级、负载因子、失败切换、并发控制、临时禁调、冷却和错误清理。
- 支持账号测试、刷新、quota/tier 刷新和用量统计。
- 支持账号级代理绑定，与 WARP/FlareSolverr 部署配合使用。

主要入口：

- 管理侧：`/admin/accounts`
- 管理 API：`/api/v1/admin/accounts/*`

### Grok Cookie / Reverse 兼容

已实现能力：

- Grok reverse chat 服务。
- Grok console chat 服务。
- Grok reverse usage 解析。
- Grok 兼容请求头处理。
- 可选动态 `x-statsig-id` 兼容头。

配置口径：

- 全局配置：`grok.dynamic_statsig_enabled`
- 账号凭据或 extra 覆盖字段：`dynamic_statsig_enabled`、`grok_dynamic_statsig_enabled`

### Cloudflare 与出口代理辅助

已实现能力：

- Docker compose proxy profile 提供 FlareSolverr / WARP 辅助栈。
- 该部署思路参考 [Chenyme/grok2api](https://github.com/chenyme/grok2api) 原项目，并结合 [jiujiu532/grok2api](https://github.com/jiujiu532/grok2api) 二开实践按 AySub 代理体系重写。
- FlareSolverr 可用于刷新 Grok Cookie 账号遇到的 `cf_clearance`。
- WARP 可作为 Grok Cookie 请求的 SOCKS5 出口代理。
- AySub 调用 FlareSolverr 时会复用账号绑定的代理，保证 clearance cookie 与实际请求出口一致。

主要文档：

- `deploy/DOCKER.md`
- README 中的 “Grok WARP / FlareSolverr 可选防封栈”

## 实现边界

- FlareSolverr / WARP 只处理 Cloudflare challenge 和出口 IP 问题。
- token 失效、账号风控、地区限制或上游拒绝访问仍需要更换账号、Cookie 或代理。
- 动态 `x-statsig-id` 是兼容开关，不应默认假定对所有账号都安全或必要。
- Grok Web 私有协议可能随时变化，生产环境需要保留账号测试、错误日志和快速回滚能力。
- `grok2api-new/` 等外部参考目录不应被当作 AySub 运行时代码，除非明确被集成和引用。

## 后续 parity 项

- Masonry 画廊体验与媒体资产链路逐项对齐。
- ChatKit 语音 UI 与实时链路逐项对齐。
- Grok Web 新模型、quota 模式和账号池信号持续跟进。
- 真实 Cookie/Console/媒体/LiveKit 账号的端到端生产验收。
- 对 Grok reverse 协议变更补充更窄的回归测试。

## 验收标准

- Grok/xAI 账号可以在账号池中创建、测试、调度和统计。
- Grok 请求失败时能进入请求日志、账号错误和运营错误视图。
- 动态 `x-statsig-id` 默认关闭，启用路径明确，可账号级覆盖。
- FlareSolverr / WARP 部署说明能指导同出口 clearance 与实际请求。
- 未生产验收的 Grok2API parity 项不在 README 中声明为已完成。

## 发布前建议验证

- Grok Cookie 账号测试和真实请求烟测。
- Grok reverse chat / console chat 目标测试。
- Grok 用量解析目标测试。
- 动态 `x-statsig-id` 开关默认关闭、全局启用、账号级覆盖三种路径检查。
- FlareSolverr / WARP compose 静态配置检查和可选服务器烟测。
- `cd backend && go test ./internal/service -run Grok`
- `pnpm --dir frontend build`
