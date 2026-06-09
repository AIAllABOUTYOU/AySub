const DEFAULT_HOME_TEXTS = new Set([
  '首页', '功能', '模型', '价格', '扩展', '资料', 'Home', 'Features', 'Models', 'Pricing', 'Extensions', 'Info',
  '稳定运行中', 'Operational', 'AI 能力', 'AI Access', '一站接入', 'One Gateway',
  '聚合 Claude、GPT、Gemini、Grok 等主流大模型，统一 API 接口，按量计费，智能调度账号与渠道。',
  'Aggregate Claude, GPT, Gemini, Grok and other mainstream models behind one API, with usage-based billing and smart account routing.',
  '立即开始', 'Get Started', '进入控制台', 'Go to Dashboard', '了解能力', 'Explore Features',
  '可配置模型', 'Configurable models', '目标可用性', 'Target availability', '调度响应', 'Routing latency',
  '核心能力', 'Capabilities', '模型支持', '资料信息', 'Information',
  '把多模型接入、账号调度和用量管理放到一个入口',
  'One entry for model access, account routing and usage control',
  '默认首页展示的是 AySub 当前已经落地的网关、权限、日志、模型和计费能力。',
  'The default home page reflects the gateway, permissions, logging, model and billing capabilities available in AySub.',
  '统一网关', 'Unified Gateway', '一个 API 入口兼容多类模型和端点，统一鉴权、转发、错误处理和响应格式。',
  'One API entry handles multiple model families and endpoints with unified auth, forwarding, error handling and response format.',
  '多账号调度', 'Multi-account Routing', '按分组、模型、额度和健康状态选择账号，失败时自动切换可用渠道。',
  'Route by group, model, quota and health. Failures can automatically switch to available channels.',
  '余额与额度', 'Balance & Quota', '用户余额、订阅额度、分组倍率和模型定价统一计算，账务边界清楚。',
  'Balance, subscriptions, group multipliers and model pricing are calculated in one place.',
  'Key 级权限', 'Key-Level Permissions', '按 API Key 控制模型、端点、额度和限流，拒绝请求由后端统一执行。',
  'Restrict models, endpoints, quota and rate limits per API key. Denials are enforced by the backend.',
  '请求日志', 'Request Logs', '成功、失败、成本、渠道、耗时和错误摘要统一记录，方便排查和报表统计。',
  'Track success, failure, cost, channel, latency and error summaries in one place.',
  '协议兼容', 'Protocol Compatible', '兼容 OpenAI 风格接口，同时保留 Claude、Gemini、Grok 等入口能力。',
  'OpenAI-style APIs with Claude, Gemini, Grok and media endpoint coverage.',
  '统一展示可用模型和计费口径', 'Show available models and pricing notes clearly',
  '管理员可以在首页配置里维护展示模型、供应商、说明和价格文字，不影响真实网关定价。',
  'Admins can edit displayed models, providers, descriptions and pricing text without changing real gateway billing rules.',
  '长上下文、代码和复杂推理场景。', 'Long-context, coding and complex reasoning workloads.',
  'OpenAI-compatible 对话、工具调用和多模态入口。', 'OpenAI-compatible chat, tool calls and multimodal access.',
  'Google Gemini 与 Code Assist 相关能力。', 'Google Gemini and Code Assist related access.',
  'xAI / Grok Web、Console 与媒体任务入口。', 'xAI / Grok Web, Console and media task access.',
  '按量计费', 'Pay as you go', '已支持', 'Supported',
  '按模型和套餐清楚展示价格', 'Present model and plan pricing',
  '这里是首页展示价格，可由后台自定义；真实扣费仍以渠道、分组和模型定价为准。',
  'These are home-page display prices. Actual billing still follows backend model, group and channel pricing.',
  '个人使用', 'Personal', '按量', 'Metered', '无固定月费', 'no fixed monthly fee', '适合个人或小工具接入。', 'For individuals and small tools.',
  '团队协作', 'Team', '配额', 'Quota', '可控预算', 'controlled budget', '适合多成员共用 Key、分组和报表。', 'For shared keys, groups and reporting.',
  '专属部署', 'Dedicated', '定制', 'Custom', '适合需要私有化、代理、审计和专属渠道策略的场景。',
  'For private deployment, proxies, audit and dedicated channel policy.', '查看文档', 'View Docs',
  '统一 API Key', 'Unified API key', '请求日志和用量查询', 'Request logs and usage query', '多模型切换', 'Multi-model switching',
  '额度和限流控制', 'Quota and rate limits', '模型与端点权限', 'Model and endpoint permissions', '运营报表', 'Operational reports',
  '私有化部署', 'Private deployment', '渠道策略配置', 'Channel policy', '安全审计', 'Security audit',
  '公开资料可以直接在后台维护', 'Public information managed from admin',
  '首页底部适合放 API 地址、客服、计费说明、安全承诺或业务联系信息。',
  'Use this area for API base URL, contact details, billing notes, security boundaries or business information.',
  'API 地址', 'API Endpoint', '用户复制 Key 后调用的基础地址。', 'Base URL users call after creating a key.',
  '计费方式', 'Billing', '按量扣费', 'Metered billing', '最终以后台模型、分组和渠道定价为准。',
  'Final cost follows backend model, group and channel pricing.',
  '安全边界', 'Security', '后端校验', 'Backend enforced', '权限拒绝、内容审核和风控事件不依赖前端隐藏。',
  'Permission denial, moderation and risk events do not rely on frontend hiding.',
  '联系信息', 'Contact', '后台未配置', 'Not configured', '可在首页配置模块中填写客服或商务联系。',
  'Set support or business contact information in Home Config.',
  '可扩展能力', 'Extensible', '按业务继续扩展首页内容', 'Keep extending the home page by business need',
  '这个区块默认隐藏，开启后可展示更多能力、场景或运营信息。',
  'This section is hidden by default. Enable it to show more capabilities, use cases or operational information.',
  'aysub-api', 'curl https://api.example.com/v1/chat/completions', 'model: claude-sonnet-4-5',
  'route: group/pro -> healthy channel', 'usage: logged, billed, audited'
])

export function isDefaultHomeText(value: unknown): value is string {
  return typeof value === 'string' && DEFAULT_HOME_TEXTS.has(value.trim())
}

export function stripDefaultHomeTextStrings<T>(value: T): T {
  if (typeof value === 'string') {
    return (isDefaultHomeText(value) ? '' : value) as T
  }

  if (Array.isArray(value)) {
    return value.map((item) => stripDefaultHomeTextStrings(item)) as T
  }

  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, stripDefaultHomeTextStrings(item)])
    ) as T
  }

  return value
}
