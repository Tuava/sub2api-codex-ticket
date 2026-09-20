import { DOMWrapper, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

import ProxyPoolSelector from '../ProxyPoolSelector.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

const proxy = (id: number, status: 'active' | 'inactive' | 'expired' = 'active') => ({
  id,
  name: `proxy-${id}`,
  protocol: 'http' as const,
  host: '127.0.0.1',
  port: 8000 + id,
  username: null,
  status,
  expires_at: null,
  fallback_mode: 'none' as const,
  expiry_warn_days: 0,
  created_at: '',
  updated_at: ''
})

describe('ProxyPoolSelector', () => {
  it('excludes the primary and unavailable proxies, then emits selected pool ids', async () => {
    const wrapper = mount(ProxyPoolSelector, {
      attachTo: document.body,
      props: {
        modelValue: [],
        primaryProxyId: 1,
        proxies: [proxy(1), proxy(2), proxy(3, 'inactive')]
      },
      global: { stubs: { Icon: true } }
    })

    await wrapper.get('button.select-trigger').trigger('click')
    const portal = document.body.querySelector('.proxy-pool-dropdown-portal')
    expect(portal).not.toBeNull()
    expect(wrapper.find('.proxy-pool-dropdown-portal').exists()).toBe(false)
    const options = new DOMWrapper(portal as Element)
    expect(options.text()).not.toContain('proxy-1')
    expect(options.text()).toContain('proxy-2')
    expect(options.text()).not.toContain('proxy-3')
    await options.get('[data-testid="proxy-pool-checkbox-2"]').setValue(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([2])
    expect(document.body.querySelector('.proxy-pool-dropdown-portal')).not.toBeNull()

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    await new Promise((resolve) => setTimeout(resolve, 200))
    expect(document.body.querySelector('.proxy-pool-dropdown-portal')).toBeNull()
    wrapper.unmount()
  })

  it('does not erase persisted ids while the proxy list is still loading', () => {
    const wrapper = mount(ProxyPoolSelector, {
      props: { modelValue: [9], proxies: [] },
      global: { stubs: { Icon: true } }
    })
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('keeps the floating panel open while selecting multiple proxies', async () => {
    const wrapper = mount(ProxyPoolSelector, {
      attachTo: document.body,
      props: {
        modelValue: [],
        proxies: [proxy(1), proxy(2), proxy(3)]
      },
      global: { stubs: { Icon: true } }
    })

    await wrapper.get('button.select-trigger').trigger('click')
    const portals = document.body.querySelectorAll('.proxy-pool-dropdown-portal')
    const portal = new DOMWrapper(portals[portals.length - 1] as Element)
    await portal.get('[data-testid="proxy-pool-checkbox-1"]').setValue(true)
    await wrapper.setProps({ modelValue: [1] })
    await portal.get('[data-testid="proxy-pool-checkbox-2"]').setValue(true)

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([1, 2])
    expect(document.body.querySelector('.proxy-pool-dropdown-portal')).not.toBeNull()
    wrapper.unmount()
  })

  it('keeps no-proxy mutually exclusive with every concrete proxy', async () => {
    const wrapper = mount(ProxyPoolSelector, {
      attachTo: document.body,
      props: {
        modelValue: [],
        proxies: [proxy(1), proxy(2)]
      },
      global: { stubs: { Icon: true } }
    })

    await wrapper.get('button.select-trigger').trigger('click')
    const portal = new DOMWrapper(document.body.querySelector('.proxy-pool-dropdown-portal') as Element)
    expect(portal.get<HTMLInputElement>('[data-testid="proxy-pool-none-checkbox"]').element.checked).toBe(true)

    await portal.get('[data-testid="proxy-pool-none-checkbox"]').setValue(false)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([])
    await wrapper.setProps({ modelValue: [] })
    expect(portal.get<HTMLInputElement>('[data-testid="proxy-pool-none-checkbox"]').element.checked).toBe(true)

    await portal.get('[data-testid="proxy-pool-checkbox-1"]').setValue(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([1])
    await wrapper.setProps({ modelValue: [1] })
    expect(portal.get<HTMLInputElement>('[data-testid="proxy-pool-none-checkbox"]').element.checked).toBe(false)

    await portal.get('[data-testid="proxy-pool-checkbox-2"]').setValue(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([1, 2])
    await wrapper.setProps({ modelValue: [1, 2] })

    await portal.get('[data-testid="proxy-pool-none-checkbox"]').setValue(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([])
    await wrapper.setProps({ modelValue: [] })
    expect(portal.get<HTMLInputElement>('[data-testid="proxy-pool-none-checkbox"]').element.checked).toBe(true)
    expect(portal.get<HTMLInputElement>('[data-testid="proxy-pool-checkbox-1"]').element.checked).toBe(false)
    expect(portal.get<HTMLInputElement>('[data-testid="proxy-pool-checkbox-2"]').element.checked).toBe(false)
    wrapper.unmount()
  })

  it('shows one unified settings panel and applies shared values to every lane', async () => {
    const wrapper = mount(ProxyPoolSelector, {
      props: {
        modelValue: [2],
        primaryProxyId: 1,
        accountConcurrency: 12,
        proxies: [proxy(1), proxy(2)],
        laneConfigs: [
          {
            proxy_id: 1,
            enabled: true,
            max_concurrency: 6,
            weight: 1,
            timeout_seconds: 0,
            error_circuit_threshold: 0,
            circuit_cooldown_seconds: 60,
            fallback_order: 0
          },
          {
            proxy_id: 2,
            enabled: true,
            max_concurrency: 6,
            weight: 1,
            timeout_seconds: 0,
            error_circuit_threshold: 0,
            circuit_cooldown_seconds: 60,
            fallback_order: 1
          }
        ]
      },
      global: { stubs: { Icon: true } }
    })

    expect(wrapper.find('[data-testid="proxy-lanes-unified-config"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid^="proxy-lane-config-"]')).toHaveLength(0)
    expect(wrapper.get('[data-testid="proxy-lanes-unified-max-concurrency"]').text()).toContain('12')

    await wrapper.get('[data-testid="proxy-lanes-unified-weight"]').setValue('3')
    const emitted = wrapper.emitted('update:laneConfigs')?.at(-1)?.[0] as Array<{ proxy_id: number; weight: number }>
    expect(emitted).toHaveLength(2)
    expect(emitted.every((lane) => lane.weight === 3)).toBe(true)
    expect(emitted.map((lane) => lane.proxy_id)).toEqual([1, 2])
    wrapper.unmount()
  })
})
