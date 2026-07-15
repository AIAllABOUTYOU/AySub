export function applyInterceptWarmup(
  credentials: Record<string, unknown>,
  enabled: boolean,
  mode: 'create' | 'edit'
): void {
  if (enabled) {
    credentials.intercept_warmup_requests = true
  } else if (mode === 'edit') {
    delete credentials.intercept_warmup_requests
  }
}

export interface PlanTypeOption {
  value: string
  label: string
  [key: string]: unknown
}

export function planTypeDisplayLabel(value: string): string {
  switch (value.trim().toLowerCase()) {
    case 'plus':
      return 'Plus'
    case 'pro':
    case 'chatgptpro':
      return 'Pro'
    case 'free':
      return 'Free'
    case 'team':
      return 'Team'
    default:
      return value
  }
}

export function readPlanType(credentials: Record<string, unknown> | undefined | null): string {
  const value = credentials?.plan_type
  return typeof value === 'string' ? value : ''
}

export function buildPlanTypeOptions(current: string, clearLabel: string): PlanTypeOption[] {
  const normalized = (current || '').trim()
  const currentLabel = normalized ? planTypeDisplayLabel(normalized) : ''
  const presets: PlanTypeOption[] = [
    { value: 'plus', label: 'Plus' },
    { value: 'pro', label: 'Pro' },
    { value: 'free', label: 'Free' }
  ]
  const options: PlanTypeOption[] = [{ value: '', label: clearLabel }]
  for (const preset of presets) {
    if (normalized && preset.value !== normalized.toLowerCase() && preset.label === currentLabel) {
      options.push({ value: normalized, label: preset.label })
    } else {
      options.push(preset)
    }
  }
  if (normalized && !options.some((option) => option.value.toLowerCase() === normalized.toLowerCase())) {
    options.push({ value: normalized, label: planTypeDisplayLabel(normalized) })
  }
  return options
}

export function applyPlanType(
  credentials: Record<string, unknown>,
  planType: string
): Record<string, unknown> {
  const normalized = (planType || '').trim()
  if (normalized) {
    credentials.plan_type = normalized
  } else {
    delete credentials.plan_type
  }
  return credentials
}
