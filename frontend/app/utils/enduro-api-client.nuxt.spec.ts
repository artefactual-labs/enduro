import { afterEach, describe, expect, it, vi } from 'vitest'
import { createEnduroApiClient } from './enduro-api-client'

describe('collection decisions', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('serializes the decision through the generated client', async () => {
    const fetch = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }))
    const client = createEnduroApiClient('http://localhost:9000')

    await client.collections.decide(77, 'ABANDON')

    expect(fetch).toHaveBeenCalledWith('http://localhost:9000/collection/77/decision', expect.objectContaining({
      method: 'POST',
      headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ option: 'ABANDON' })
    }))
  })
})
