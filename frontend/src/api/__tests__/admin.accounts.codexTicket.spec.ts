import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))

vi.mock('../client', () => ({
  apiClient: { get, post }
}))

import { exportData, getCodexTicketProbeProgress, probeCodexTicket } from '@/api/admin/accounts'

describe('admin accounts Codex ticket probe API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('posts the current model policy and allows the full backend probe window', async () => {
    const policy = {
      enabled: true,
      target_mode: 'manual' as const,
      target_length: 332,
      missing_policy: 'pause' as const
    }
    post.mockResolvedValue({
      data: {
        model: 'gpt-6-astra',
        result: { attempted: true, outcome: 'non_target', target_length: 332, ready: false },
        tickets: []
      }
    })

    const operationID = '11111111-1111-4111-8111-111111111111'
    const result = await probeCodexTicket(41, 'gpt-6-astra', operationID, policy)

    expect(post).toHaveBeenCalledWith(
      '/admin/accounts/41/codex-ticket/probe',
      { model: 'gpt-6-astra', policy, operation_id: operationID },
      { timeout: 90000 }
    )
    expect(result.result?.outcome).toBe('non_target')
  })

  it('loads server-observed progress and logs for the operation', async () => {
    const progress = {
      operation_id: '11111111-1111-4111-8111-111111111111',
      account_id: 41,
      model: 'gpt-6-astra',
      status: 'running',
      stage: 'requesting_upstream',
      percent: 56,
      started_at: '2028-01-01T00:00:00Z',
      updated_at: '2028-01-01T00:00:01Z',
      elapsed_ms: 1000,
      ready: false,
      logs: []
    }
    get.mockResolvedValue({ data: progress })

    await expect(getCodexTicketProbeProgress(41, progress.operation_id)).resolves.toEqual(progress)
    expect(get).toHaveBeenCalledWith('/admin/accounts/41/codex-ticket/probe/11111111-1111-4111-8111-111111111111')
  })

  it('requests original ticket values only through the protected backup endpoint', async () => {
    get.mockResolvedValue({
      data: { exported_at: '2026-09-20T00:00:00Z', proxies: [], accounts: [], ticket_info: [] }
    })

    await exportData({ ids: [41], includeProxies: false, includeTicketInfo: true })

    expect(get).toHaveBeenCalledWith('/admin/accounts/data', {
      params: {
        ids: '41',
        include_proxies: 'false',
        include_ticket_info: 'true'
      }
    })
  })

  it('includes original ticket values in account backups by default', async () => {
    get.mockResolvedValue({
      data: { exported_at: '2026-09-20T00:00:00Z', proxies: [], accounts: [], ticket_info: [] }
    })

    await exportData()

    expect(get).toHaveBeenCalledWith('/admin/accounts/data', {
      params: { include_ticket_info: 'true' }
    })
  })
})
