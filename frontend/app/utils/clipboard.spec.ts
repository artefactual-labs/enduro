import { expect, test } from 'vitest'

import { copyTextToClipboard } from './clipboard'

test('copyTextToClipboard uses navigator clipboard when available', async () => {
  const originalNavigator = globalThis.navigator
  const calls: string[] = []

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: {
      clipboard: {
        async writeText(value: string) {
          calls.push(value)
        }
      }
    }
  })

  try {
    const copied = await copyTextToClipboard('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
    expect(copied).toBe(true)
    expect(calls).toEqual(['31ceb5d5-a9c1-488b-b4ee-40910e54109e'])
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: originalNavigator
    })
  }
})

test('copyTextToClipboard falls back to document.execCommand', async () => {
  const originalNavigator = globalThis.navigator
  const originalDocument = globalThis.document

  const textarea = {
    value: '',
    style: {} as Record<string, string>,
    setAttribute() {},
    focus() {},
    select() {}
  }

  const appended: unknown[] = []
  const removed: unknown[] = []

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: {}
  })

  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      body: {
        appendChild(node: unknown) {
          appended.push(node)
        },
        removeChild(node: unknown) {
          removed.push(node)
        }
      },
      createElement(tag: string) {
        expect(tag).toBe('textarea')
        return textarea
      },
      execCommand(command: string) {
        expect(command).toBe('copy')
        return true
      }
    }
  })

  try {
    const copied = await copyTextToClipboard('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
    expect(copied).toBe(true)
    expect(textarea.value).toBe('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
    expect(appended).toEqual([textarea])
    expect(removed).toEqual([textarea])
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: originalNavigator
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: originalDocument
    })
  }
})

test('copyTextToClipboard falls back when navigator clipboard rejects', async () => {
  const originalNavigator = globalThis.navigator
  const originalDocument = globalThis.document

  const textarea = {
    value: '',
    style: {} as Record<string, string>,
    setAttribute() {},
    focus() {},
    select() {}
  }

  const calls: string[] = []

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: {
      clipboard: {
        async writeText(value: string) {
          calls.push(value)
          throw new Error('Clipboard permission denied')
        }
      }
    }
  })

  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: {
      body: {
        appendChild() {},
        removeChild() {}
      },
      createElement(tag: string) {
        expect(tag).toBe('textarea')
        return textarea
      },
      execCommand(command: string) {
        expect(command).toBe('copy')
        return true
      }
    }
  })

  try {
    const copied = await copyTextToClipboard('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
    expect(copied).toBe(true)
    expect(calls).toEqual(['31ceb5d5-a9c1-488b-b4ee-40910e54109e'])
    expect(textarea.value).toBe('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: originalNavigator
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: originalDocument
    })
  }
})

test('copyTextToClipboard returns false for empty values', async () => {
  const copied = await copyTextToClipboard('')
  expect(copied).toBe(false)
})

test('copyTextToClipboard returns false when no clipboard API or document body exists', async () => {
  const originalNavigator = globalThis.navigator
  const originalDocument = globalThis.document

  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: {}
  })

  Object.defineProperty(globalThis, 'document', {
    configurable: true,
    value: undefined
  })

  try {
    const copied = await copyTextToClipboard('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
    expect(copied).toBe(false)
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: originalNavigator
    })
    Object.defineProperty(globalThis, 'document', {
      configurable: true,
      value: originalDocument
    })
  }
})
