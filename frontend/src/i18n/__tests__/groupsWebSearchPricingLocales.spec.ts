import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('group web search pricing locales', () => {
  it('exposes matching Chinese and English pricing keys', () => {
    expect(zh.admin.groups.webSearchPricing).toMatchObject({
      title: 'Codex 网页搜索计费',
      pricePerCall: '每次搜索价格（USD）',
      pricePerCallHint: expect.stringContaining('$0.01'),
      finalPricePreview: expect.stringContaining('{price}')
    })
    expect(en.admin.groups.webSearchPricing).toMatchObject({
      title: 'Codex Web Search Pricing',
      pricePerCall: 'Price per search call (USD)',
      pricePerCallHint: expect.stringContaining('$0.01'),
      finalPricePreview: expect.stringContaining('{price}')
    })
  })
})
