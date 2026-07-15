import { describe, expect, it } from 'vitest'

import { findRowIndexByDomPosition } from '../useSwipeSelect'

function makeScrollElement(rows: Array<{ index: number; top: number; bottom: number }>): Element {
  const elements = rows.map((row) => ({
    getAttribute: (name: string) => (name === 'data-index' ? String(row.index) : null),
    getBoundingClientRect: () => ({
      top: row.top,
      bottom: row.bottom,
      left: 0,
      right: 0,
      width: 0,
      height: row.bottom - row.top,
      x: 0,
      y: row.top,
      toJSON: () => ({})
    })
  })) as unknown as HTMLElement[]

  return {
    querySelectorAll: (selector: string) =>
      (selector === 'tbody tr[data-index]' ? elements : []) as unknown as NodeListOf<Element>
  } as unknown as Element
}

describe('findRowIndexByDomPosition', () => {
  const rows = [
    { index: 0, top: 100, bottom: 200 },
    { index: 1, top: 200, bottom: 300 },
    { index: 2, top: 300, bottom: 450 }
  ]
  const scrollElement = makeScrollElement(rows)

  it('returns -1 when no rows are rendered', () => {
    expect(findRowIndexByDomPosition(makeScrollElement([]), 250)).toBe(-1)
  })

  it('locates the row containing the Y coordinate', () => {
    expect(findRowIndexByDomPosition(scrollElement, 150)).toBe(0)
    expect(findRowIndexByDomPosition(scrollElement, 250)).toBe(1)
    expect(findRowIndexByDomPosition(scrollElement, 400)).toBe(2)
  })

  it('clamps to the first and last rendered rows', () => {
    expect(findRowIndexByDomPosition(scrollElement, 50)).toBe(0)
    expect(findRowIndexByDomPosition(scrollElement, 999)).toBe(2)
  })

  it('picks the closer row when the Y coordinate falls in a gap', () => {
    const gapped = makeScrollElement([
      { index: 0, top: 100, bottom: 180 },
      { index: 1, top: 220, bottom: 300 }
    ])
    expect(findRowIndexByDomPosition(gapped, 190)).toBe(0)
    expect(findRowIndexByDomPosition(gapped, 215)).toBe(1)
  })

  it('returns data-index rather than the DOM array position', () => {
    const remapped = makeScrollElement([
      { index: 5, top: 100, bottom: 200 },
      { index: 9, top: 200, bottom: 300 }
    ])
    expect(findRowIndexByDomPosition(remapped, 150)).toBe(5)
    expect(findRowIndexByDomPosition(remapped, 250)).toBe(9)
  })
})
