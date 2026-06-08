# Sub2API+NewAPI+Grok2API三合一魔改整合方案
## 文档版本：V1.0｜项目名称：AySub｜官网：aiaay.com｜GitHub：https://github.com/AIAllABOUTYOU/AySub｜开源协议：MIT

## 一、基础信息对照表
|项目|开发语言|存储引擎|产品核心定位|
| ---- | ---- | ---- | ---- |
|Sub2API|Go|PostgreSQL+Redis|订阅Cookie聚合网关，主打会话粘连、订阅账号池调度、合租Token记账|
|NewAPI|Go|SQLite/MySQL/PostgreSQL|API商用管理网关，用户系统、模型管理、分组倍率、密钥权限管控|
|Grok2API|Python(FastAPI)|SQLite/JSON文件|Grok X会员Cookie专用转OpenAI中间件，仅限Grok网页逆向|

## 二、上游接入能力对比&改造方案
|功能|Sub2API|NewAPI|Grok2API|魔改处置规则|
| ---- |:----:|:----:|:----:|----|
|GPT+/Claude/Gemini网页Cookie录入|✅原生|❌不支持|❌不支持|完整保留Sub原有解析代码|
|X/Grok网页Cookie接入|❌无适配器|❌不支持|✅原生逆向|Grok2核心逻辑转Go，新增Grok渠道适配器内置进Sub|
|各厂商官方OpenAI Key接入|✅自定义渠道|✅全品类原生渠道|✅仅Grok官方Key|UI复用New渠道配置表单，接入逻辑统一|
|第三方中转API对接|✅支持|✅完善|✅支持|沿用New渠道配置参数规范|

## 三、调度引擎能力对比（核心保留Sub）
|功能|Sub2API|NewAPI|Grok2API|魔改处置规则|
| ---- |:----:|:----:|:----:|----|
|Sticky会话绑定（同对话固定账号）|✅原生|❌轮询换号|✅单池粘连|**全盘沿用Sub调度内核，不替换**|
|账号失效自动剔除、健康检测|✅|❌仅报错切换|✅|保留Sub检测逻辑|
|账号额度耗尽自动切备用池、熔断|✅|❌|✅|原生不动|
|精细化渠道权重、分组路由|✅简易配置|✅完善权重|✅简易|移植New权重配置UI至Sub后台|

## 四、模型&定价管理（全量移植NewAPI）
|功能|Sub2API|NewAPI|Grok2API|魔改处置规则|
| ---- |:----:|:----:|:----:|----|
|全局模型库、自定义模型别名|❌简陋|✅批量管理|❌固定模型|Sub新增New同款模型管理页面|
|分模型独立倍率定价|❌基础计价|✅精细化倍率|❌无|移植New定价计算逻辑|
|渠道绑定指定可用模型、启停管控|❌|✅|❌|并入Sub渠道编辑弹窗|
|模型分组归类管理|❌|✅|❌|新增模型分组菜单|

## 五、用户&密钥系统（全量移植NewAPI）
|功能|Sub2API|NewAPI|Grok2API|魔改处置规则|
| ---- |:----:|:----:|:----:|----|
|多用户注册登录、角色权限分组|❌仅单管理员|✅完整体系|❌无|Sub新增用户数据表与后台页面|
|用户独立额度、分组配额、黑名单|❌|✅|❌无|接入New额度管控逻辑|
|密钥绑定用户/分组、批量生成密钥|✅简单生成|✅精细化分组密钥|❌单密钥|融合两者密钥生成规则|
|用户余额、充值、消费账单|❌合租账单|✅商用账单|❌无|Sub账单+New消费统计合并|

## 六、数据大盘与统计
|功能|Sub2API|NewAPI|Grok2API|魔改处置规则|
| ---- |:----:|:----:|:----:|----|
|上游订阅账号用量监控|✅订阅维度|❌Key维度|✅Grok账号|仪表盘分两块：左侧上游账号(Sub)、右侧用户消费(New)|
|全量Token统计、可视化图表|✅|✅完善大盘|❌极简|统一一套统计面板|

## 七、UI界面整合规划
|项目|前端框架|原有菜单结构|整合方案|
| ---- | ---- | ---- | ---- |
|Sub2API|Vue3+Element Plus|渠道/密钥/账单|**作为主体框架，保留侧边栏基座**|
|NewAPI|Vue3+Element Plus|仪表盘/渠道/模型/用户/账单|页面功能全移植进Sub前端，新增对应侧边菜单|
|Grok2API|简易前端|仅Grok账号配置|废弃独立页面，配置内嵌至Sub渠道-Grok分类|

> 最终侧边菜单结构
> 1. 仪表盘（合并Sub上游监控+New用户消费）
> 2. 渠道管理
>    - GPT/Claude/Gemini(订阅Cookie)
>    - Grok(X会员Cookie)
>    - 第三方API Key渠道
> 3. 模型管理（New全套功能）
> 4. 用户管理（New全套功能）
> 5. API密钥管理（Sub+New融合）
> 6. 账单统计
> 7. 系统设置

## 八、特色附加功能
|功能|Sub2API|NewAPI|Grok2API|整合方案|
| ---- |:----:|:----:|:----:|----|
|Grok联网搜索、X内置搜索|❌|❌|✅|Grok渠道配置页新增搜索开关配置|
|多模态生图兼容|部分支持|全兼容|Grok专属生图|模型配置页开启多模态选项|

## 九、最终整合成品架构
1. **底层调度内核：完全保留Sub2API原生代码（会话粘连、订阅池、熔断）**
2. **上层运营模块：全量移植NewAPI【用户+模型+定价+分组+UI】**
3. **Grok网页逆向：Grok2API转Go适配器内置，消除独立Python服务**
4. **部署形态：单一Go二进制+单套Vue前端+PostgreSQL+Redis，一个Docker镜像完成三合一**

## 十、当前实现进度
> 更新时间：2026-06-03<br>
> 当前落点：`/Volumes/llovky/AitchTey/code/AySub`

### 1. 已完成
|模块|进度|说明|
| ---- |:----:| ---- |
|Sub2API 主体落地|✅|代码已落在 AySub 根目录，保留 Sub 调度内核、账号池、粘性会话、熔断与用量记录主路径。|
|xAI 官方 API Key|✅|新增 `xai` 平台；默认 base URL 为 `https://api.x.ai`；Chat Completions、Responses fallback、Messages fallback 已接入。|
|xAI `/v1/responses` fallback|✅|xAI API Key 默认不走上游 `/v1/responses`，会转换到 `/v1/chat/completions`，再转换回 Responses 响应。|
|xAI `/v1/messages` fallback|✅|Anthropic Messages 入口已支持 xAI API Key，非流式/流式均转上游 `/v1/chat/completions`，再转回 Anthropic Messages 协议。|
|Grok Cookie 账号|✅|新增 `platform=xai,type=cookie`，支持后台创建/编辑；支持 `sso_token`、完整 cookie、Cloudflare cookie、搜索开关等配置。|
|Grok Cookie Chat Completions|✅|Grok Web reverse 已支持 `/v1/chat/completions` 非流式/流式，含 thinking、搜索引用、图片 markdown、多模态 data URI 上传。|
|Grok Cookie Responses|✅|Grok Web reverse 已支持 `/v1/responses` 非流式/流式，含 Responses 事件转换、usage 估算、搜索引用和图片输出。|
|Grok Cookie Anthropic Messages|✅|新增 `/v1/messages` 支持，Claude/Anthropic 兼容客户端可走 Grok Cookie；支持文本、thinking 流、base64 image block 上传。|
|Grok Cookie Images|✅|支持 `/v1/images/generations` 与 `/v1/images/edits`，可返回本地 URL 或 `b64_json`，edit 可上传 image/mask。|
|Grok Cookie Videos|✅|新增 OpenAI-style `/v1/videos`、`/v1/videos/{video_id}`、`/v1/videos/{video_id}/content`；按 Grok Web `media/post/create` + `app-chat` 视频流协议生成真实视频 URL，并尽量缓存为本地视频文件，失败时暴露上游错误，不做假成功。|
|Grok 本地文件缓存|✅|结合 `jiujiu532/grok2api` 二开能力，新增生成图片/视频本地落盘与 `/v1/files/image?id=...`、`/v1/files/video?id=...` 读取端点；文件位于 `DATA_DIR/files/images|videos`。|
|Grok LiveKit token|✅|新增 `/v1/livekit/tokens`，按 Grok Web `/rest/livekit/tokens` 协议获取短期 LiveKit token，并返回 `wss://livekit.grok.com/rtc` 连接 URL。|
|Grok LiveKit RTC WS|✅/⏳|新增 `/v1/livekit/rtc` 与 `/livekit/rtc` WebSocket 透明代理，客户端传入 LiveKit `access_token` 后双向桥接到 `wss://livekit.grok.com/rtc`；代理层已落地，真实音频协议仍需账号环境验收。|
|Grok Console 免费模型|✅/⏳|已结合 `jiujiu532/grok2api` 二开协议，支持 Cookie 账号通过 `console.x.ai/v1/responses` 调用 `grok-4.3-console`、`grok-4.3-low/medium/high`、`grok-4.20-*-console`、`grok-4.20-multi-agent-*`、`grok-build-console`；Chat Completions、Responses、Anthropic Messages 三类入口已转换，真实免费账号仍需端到端验收。|
|xAI/Grok 模型列表|✅|`/v1/models` 对 xAI 分组返回 Grok2API 对齐模型：`grok-4.20-*`、`grok-imagine-image*`、`grok-imagine-video`、`grok-4.3-console`、`grok-build-console` 等 console 免费模型，支持自定义 models list 过滤。|
|Grok Cookie 用量查询|✅|后台 `/admin/accounts/:id/usage` 对 xAI Cookie 调用 Grok Web `/rest/rate-limits`，展示 fast/expert/heavy/auto 等 quota。|
|前端账号配置|✅|创建/编辑账号表单已支持 Grok Cookie 与 xAI API Key，敏感字段脱敏，账号用量单元格支持 Grok quota 展示。|
|xAI 用户额度与分组|✅|用户平台额度、分组过滤、平台白名单已加入 xAI，避免 xAI 配额配置被清理。|
|xAI Channel Monitor|✅|渠道监控支持 xAI provider；xAI 监控使用 OpenAI Chat 请求模板，Responses 监控仍仅限 OpenAI。|
|端点统计归因|✅|xAI `/v1/responses` 与 `/v1/messages` 的实际上游端点归因到 `/v1/chat/completions`。|

### 2. 已验证
- 后端大范围单元测试通过：`go test -tags unit ./internal/service ./internal/handler ./internal/handler/admin ./internal/server/routes ./internal/model`
- 前端类型检查已通过：`pnpm --dir frontend typecheck`
- Grok/xAI 定向用例通过：`/v1/messages` fallback、Grok Chat/Responses/Images/Videos、生成文件缓存、`/v1/files/image|video` 路由、LiveKit token、LiveKit RTC WS 路由、Grok Console Chat/Responses/Messages 转换、xAI `/v1/models`、网关路由注册。
- 前端关键用例已通过：账号用量、账号编辑、设置页、用户平台额度等相关 vitest 用例。

### 3. 剩余缺口
|模块|状态|待完成内容|
| ---- |:----:| ---- |
|Grok 实时音频 WS 桥接|✅/⏳|已完成 `/v1/livekit/tokens` token 获取与 `/v1/livekit/rtc` WebSocket 透明双向代理；仍需真实 Grok Cookie 与真实音频客户端做端到端协议验收。|
|NewAPI 模型管理深度|✅/⏳|模型、倍率、渠道白名单、模型列表候选、定价同步、渠道定价 UI 已具备主路径；后续只剩更细的批量运维体验和模型分组 UI 深度打磨。|
|NewAPI 商用运营能力|✅/⏳|用户、额度、余额、订单、充值、套餐、支付实例、支付回调、运营支付看板已有主路径；后续剩余真实支付商户配置和运营报表细化。|
|二开 Grok2API parity|✅/⏳|`jiujiu532/grok2api` 的 console 免费模型主协议、本地图片/视频缓存与 `/v1/files/image|video` 已转 Go；WARP/FlareSolverr 部署防封栈已按 AySub 代理体系接入：WARP 作为账号代理出口，FlareSolverr 作为 Grok Cookie 的 Cloudflare clearance 自动刷新辅助。图生视频 media link/upscale、Masonry/ChatKit/Admin WebUI 仍未并入，后续应按 AySub 现有后台重新设计，而不是直接整体搬入。|
|Grok2API parity|✅/⏳|Chat、Responses、Messages、Images、Videos、Usage、模型列表、LiveKit token、LiveKit RTC WS、Console 免费模型代理、本地文件缓存已转 Go；剩余为边缘参数、异常策略、真实 Cookie/Console 环境逐项对账。|
|全量端到端验收|⏳|已完成单元测试和前端类型检查；仍需要真实账号环境下的 xAI API Key、Grok Cookie、Console 免费模型、图片、视频、本地文件访问、LiveKit token、quota、搜索链路端到端验证。|
|部署镜像收口|✅/⏳|已有根目录/后端 Dockerfile、compose、standalone/local 部署文档；后续需要按最终域名与生产环境做一次镜像构建和部署演练。|

### 4. 当前结论
当前已完成 xAI/Grok 的核心入口协议链路：`/v1/chat/completions`、`/v1/responses`、`/v1/messages`、图片生成/编辑、视频生成、本地图片/视频文件缓存、LiveKit token、LiveKit RTC WS 代理、Grok Cookie quota、Grok Console 免费模型、xAI/Grok 模型列表与后台配置。<br>
三合一方案的代码主路径已基本补齐；剩余重点是真实账号端到端验收、生产镜像部署演练，以及 NewAPI 运营 UI 的细节打磨。

## 十一、项目品牌与开源说明
### 1.项目信息
- 项目名称：**AySub**
- 官方主站：`aiaay.com`
- API网关域名：`api.aiaay.com`
- 管理后台域名：`api.aiaay.com/login` 
- GitHub仓库：`https://github.com/AIAllABOUTYOU/AySub`
- 项目简介：基于Sub2API内核二次魔改三合一AI聚合网关，融合NewAPI用户、模型、倍率定价整套运营体系，内置Grok2API改写后的Grok Cookie逆向适配器，一站式兼容GPT/Claude/Gemini/Grok网页订阅Cookie以及全厂商官方API Key。

### 2.开源协议
本项目采用 **MIT开源协议**
1. 允许个人/企业自由商用、源码修改、二次分发、闭源打包；
2. 二次衍生项目仅需保留版权声明；
3. 遵循上游Sub2API、NewAPI、Grok2API原版MIT开源协议规范。
> Copyright (c) Aiaay(aiaay.com)

学 AI 上 L 站
https://linux.do/
