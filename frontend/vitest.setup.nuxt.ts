import { vi } from 'vitest'

class MockEventSource {
  onerror: ((event?: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onopen: ((event?: Event) => void) | null = null

  constructor(public url: string) {}

  close() {}
}

vi.stubGlobal('EventSource', MockEventSource)
