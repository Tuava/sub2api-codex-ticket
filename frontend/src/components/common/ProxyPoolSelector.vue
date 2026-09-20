<template>
  <div ref="containerRef" class="space-y-2">
    <button
      ref="triggerRef"
      type="button"
      class="select-trigger"
      :class="[
        isOpen && 'select-trigger-open',
        disabled && 'select-trigger-disabled'
      ]"
      :disabled="disabled"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      @click="toggleDropdown"
      @keydown.down.prevent="openDropdown"
      @keydown.up.prevent="openDropdown"
    >
      <span class="select-value">
        {{ selectedLabel }}
      </span>
      <span class="select-icon">
        <Icon
          name="chevronDown"
          size="md"
          :class="['transition-transform duration-200', isOpen && 'rotate-180']"
        />
      </span>
    </button>

    <Teleport to="body">
      <Transition name="select-dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          :class="['proxy-pool-dropdown-portal', instanceId]"
          :style="dropdownStyle"
          role="listbox"
          aria-multiselectable="true"
          @click.stop
          @mousedown.stop
          @keydown.esc.prevent="closeDropdown(true)"
        >
          <div class="proxy-pool-header">
            <div class="proxy-pool-search">
              <Icon name="search" size="sm" class="shrink-0 text-gray-400" />
              <input
                ref="searchInputRef"
                v-model="searchQuery"
                type="text"
                class="proxy-pool-search-input"
                :placeholder="t('admin.proxies.searchProxies')"
                @click.stop
              />
            </div>
            <span class="shrink-0 text-xs text-gray-400 dark:text-gray-500">
              {{ t('common.selectedCount', { count: modelValue.length }) }}
            </span>
          </div>

          <div class="proxy-pool-options">
            <label
              role="option"
              :aria-selected="modelValue.length === 0"
              data-testid="proxy-pool-none-option"
              :class="[
                'proxy-pool-option',
                modelValue.length === 0 && 'proxy-pool-option-selected'
              ]"
            >
              <input
                type="checkbox"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="modelValue.length === 0"
                data-testid="proxy-pool-none-checkbox"
                @change="selectNoProxy($event)"
              />
              <span class="min-w-0 flex-1 text-sm font-medium text-gray-800 dark:text-gray-100">
                {{ t('admin.accounts.noProxy') }}
              </span>
              <Icon
                v-if="modelValue.length === 0"
                name="check"
                size="sm"
                class="shrink-0 text-primary-500"
              />
            </label>

            <div class="mx-3 border-t border-gray-100 dark:border-dark-700" />

            <label
              v-for="proxy in filteredProxies"
              :key="proxy.id"
              role="option"
              :data-testid="`proxy-pool-option-${proxy.id}`"
              :aria-selected="modelValue.includes(proxy.id)"
              :class="[
                'proxy-pool-option',
                modelValue.includes(proxy.id) && 'proxy-pool-option-selected'
              ]"
            >
              <input
                type="checkbox"
                class="h-4 w-4 shrink-0 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="modelValue.includes(proxy.id)"
                :data-testid="`proxy-pool-checkbox-${proxy.id}`"
                @change="toggleProxy(proxy.id)"
              />
              <span class="min-w-0 flex-1">
                <span class="flex min-w-0 items-center gap-2">
                  <span class="truncate text-sm font-medium text-gray-800 dark:text-gray-100">
                    {{ proxy.name }}
                  </span>
                  <span
                    v-if="proxy.latency_ms != null"
                    class="shrink-0 rounded bg-emerald-50 px-1.5 py-0.5 text-[10px] text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                  >
                    {{ proxy.latency_ms }}ms
                  </span>
                  <span
                    v-if="proxy.account_count != null"
                    class="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-500 dark:bg-dark-600 dark:text-gray-400"
                  >
                    {{ proxy.account_count }}
                  </span>
                </span>
                <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ proxy.protocol }}://{{ proxy.host }}:{{ proxy.port }}
                </span>
              </span>
              <Icon
                v-if="modelValue.includes(proxy.id)"
                name="check"
                size="sm"
                class="shrink-0 text-primary-500"
              />
            </label>

            <p v-if="filteredProxies.length === 0" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('common.noOptionsFound') }}
            </p>
          </div>

          <div class="proxy-pool-footer">
            <button type="button" class="proxy-pool-action" @click="selectAllFiltered">
              {{ t('common.selectAll') }}
            </button>
            <button
              type="button"
              class="proxy-pool-action"
              :disabled="modelValue.length === 0"
              @click="clearSelection"
            >
              {{ t('common.clear') }}
            </button>
            <button type="button" class="btn btn-primary btn-sm ml-auto" @click="closeDropdown(true)">
              {{ t('common.confirm') }}
            </button>
          </div>
        </div>
      </Transition>
    </Teleport>

    <div v-if="selectedProxies.length > 0" class="flex flex-wrap gap-1.5">
      <button
        v-for="proxy in selectedProxies"
        :key="proxy.id"
        type="button"
        class="inline-flex items-center gap-1 rounded-full bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
        :title="t('common.remove')"
        @click="toggleProxy(proxy.id)"
      >
        {{ proxy.name }}
        <span aria-hidden="true">×</span>
      </button>
    </div>

    <div v-if="laneProxies.length > 0" class="mt-3 space-y-3 rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800/60">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.accounts.proxyLanes.title') }}
          </div>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.proxyLanes.hint') }}
          </p>
        </div>
        <select
          :value="laneStrategy"
          class="input min-w-44 py-1.5 text-xs"
          data-testid="proxy-lane-strategy"
          @change="updateLaneStrategy(($event.target as HTMLSelectElement).value as ProxyLaneStrategy)"
        >
          <option value="round_robin">{{ t('admin.accounts.proxyLanes.roundRobin') }}</option>
          <option value="least_connections">{{ t('admin.accounts.proxyLanes.leastConnections') }}</option>
          <option value="weighted">{{ t('admin.accounts.proxyLanes.weighted') }}</option>
        </select>
      </div>

      <div class="rounded-lg border border-primary-200 bg-white p-3 dark:border-primary-800/50 dark:bg-dark-700" data-testid="proxy-lanes-unified-config">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <div class="text-xs font-semibold text-gray-900 dark:text-white">
              {{ t('admin.accounts.proxyLanes.unifiedSettings') }}
            </div>
            <p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.proxyLanes.unifiedSettingsHint') }}
            </p>
          </div>
          <label class="inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input
              :checked="unifiedLaneConfig.enabled"
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              data-testid="proxy-lanes-unified-enabled"
              @change="updateUnified('enabled', ($event.target as HTMLInputElement).checked)"
            />
            {{ t('common.enabled') }}
          </label>
        </div>

        <div class="mt-3 grid grid-cols-2 gap-2 lg:grid-cols-5">
          <label class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.proxyLanes.accountConcurrency') }}
            <div
              class="mt-1 flex h-[34px] items-center rounded-xl border border-gray-200 bg-gray-50 px-3 text-sm font-semibold text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200"
              data-testid="proxy-lanes-unified-max-concurrency"
            >
              {{ accountConcurrency }}
            </div>
            <span class="mt-1 block text-[10px] text-gray-400 dark:text-gray-500">
              {{ t('admin.accounts.proxyLanes.accountConcurrencyHint') }}
            </span>
          </label>
          <label class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.proxyLanes.weight') }}
            <input
              :value="unifiedLaneConfig.weight"
              type="number"
              min="1"
              max="1000"
              class="input mt-1 py-1.5 text-xs"
              data-testid="proxy-lanes-unified-weight"
              @input="updateUnifiedNumber('weight', $event, 1)"
            />
          </label>
          <label class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.proxyLanes.timeout') }}
            <input
              :value="unifiedLaneConfig.timeout_seconds"
              type="number"
              min="0"
              max="86400"
              class="input mt-1 py-1.5 text-xs"
              data-testid="proxy-lanes-unified-timeout"
              @input="updateUnifiedNumber('timeout_seconds', $event, 0)"
            />
          </label>
          <label class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.proxyLanes.circuitThreshold') }}
            <input
              :value="unifiedLaneConfig.error_circuit_threshold"
              type="number"
              min="0"
              max="1000"
              class="input mt-1 py-1.5 text-xs"
              data-testid="proxy-lanes-unified-circuit-threshold"
              @input="updateUnifiedNumber('error_circuit_threshold', $event, 0)"
            />
          </label>
          <label class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.proxyLanes.circuitCooldown') }}
            <input
              :value="unifiedLaneConfig.circuit_cooldown_seconds"
              type="number"
              min="0"
              max="86400"
              class="input mt-1 py-1.5 text-xs"
              data-testid="proxy-lanes-unified-circuit-cooldown"
              @input="updateUnifiedNumber('circuit_cooldown_seconds', $event, 0)"
            />
          </label>
        </div>
      </div>

      <div class="space-y-2" data-testid="proxy-lanes-list">
        <div
          v-for="(proxy, index) in laneProxies"
          :key="proxy.id"
          class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 dark:border-dark-600 dark:bg-dark-700"
          :data-testid="`proxy-lane-${proxy.id}`"
        >
          <div class="min-w-0 flex items-center gap-2">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gray-100 text-[11px] font-semibold text-gray-500 dark:bg-dark-600 dark:text-gray-300">
              {{ index + 1 }}
            </span>
            <div class="min-w-0">
              <div class="flex min-w-0 items-center gap-2">
                <span class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ proxy.name }}</span>
                <span
                  v-if="proxy.id === primaryProxyId"
                  class="shrink-0 rounded-full bg-blue-50 px-2 py-0.5 text-[11px] text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                >
                  {{ t('admin.accounts.proxyLanes.primary') }}
                </span>
              </div>
              <div class="mt-0.5 flex flex-wrap items-center gap-x-2 text-[11px] text-gray-500 dark:text-gray-400">
                <span>{{ proxy.protocol }}://{{ proxy.host }}:{{ proxy.port }}</span>
                <span v-if="proxy.latency_ms != null">{{ proxy.latency_ms }}ms</span>
              </div>
            </div>
          </div>
          <span class="shrink-0 text-[11px] text-gray-500 dark:text-gray-400">
            {{ unifiedLaneConfig.enabled ? t('admin.accounts.proxyLanes.ready') : t('admin.accounts.proxyLanes.disabled') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Proxy, ProxyLaneConfig, ProxyLaneStrategy } from '@/types'

const props = withDefaults(defineProps<{
  modelValue: number[]
  proxies: Proxy[]
  primaryProxyId?: number | null
  laneConfigs?: ProxyLaneConfig[]
  laneStrategy?: ProxyLaneStrategy
  accountConcurrency?: number
  disabled?: boolean
}>(), {
  primaryProxyId: null,
  laneConfigs: () => [],
  laneStrategy: 'round_robin',
  accountConcurrency: 1,
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
  'update:laneConfigs': [value: ProxyLaneConfig[]]
  'update:laneStrategy': [value: ProxyLaneStrategy]
}>()

const { t } = useI18n()
const instanceId = `proxy-pool-${Math.random().toString(36).slice(2, 9)}`
const isOpen = ref(false)
const searchQuery = ref('')
const containerRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const triggerRect = ref<DOMRect | null>(null)
const dropdownPosition = ref<'bottom' | 'top'>('bottom')

const dropdownStyle = computed(() => {
  const rect = triggerRect.value
  if (!rect) return {}
  const padding = 8
  const viewportWidth = window.innerWidth
  const availableWidth = Math.max(280, viewportWidth - padding * 2)
  const width = Math.min(Math.max(rect.width, 360), Math.min(520, availableWidth))
  const left = Math.min(Math.max(padding, rect.left), Math.max(padding, viewportWidth - width - padding))
  const style: Record<string, string> = {
    position: 'fixed',
    left: `${left}px`,
    width: `${width}px`,
    zIndex: '100000030'
  }
  if (dropdownPosition.value === 'top') {
    style.bottom = `${window.innerHeight - rect.top + 4}px`
  } else {
    style.top = `${rect.bottom + 4}px`
  }
  return style
})

const updateTriggerRect = () => {
  if (triggerRef.value) triggerRect.value = triggerRef.value.getBoundingClientRect()
}

const calculateDropdownPosition = () => {
  updateTriggerRect()
  nextTick(() => {
    const rect = triggerRect.value
    const dropdown = dropdownRef.value
    if (!rect || !dropdown) return
    const height = dropdown.offsetHeight || 320
    const below = window.innerHeight - rect.bottom
    const above = rect.top
    dropdownPosition.value = below < height && above > below ? 'top' : 'bottom'
  })
}

const openDropdown = () => {
  if (props.disabled || isOpen.value) return
  isOpen.value = true
}

const closeDropdown = (focusTrigger = false) => {
  if (!isOpen.value) return
  isOpen.value = false
  searchQuery.value = ''
  if (focusTrigger) nextTick(() => triggerRef.value?.focus())
}

const toggleDropdown = () => {
  if (isOpen.value) closeDropdown()
  else openDropdown()
}

watch(isOpen, (open) => {
  if (open) {
    calculateDropdownPosition()
    window.addEventListener('scroll', updateTriggerRect, { capture: true, passive: true })
    window.addEventListener('resize', calculateDropdownPosition)
    nextTick(() => searchInputRef.value?.focus())
  } else {
    window.removeEventListener('scroll', updateTriggerRect, { capture: true })
    window.removeEventListener('resize', calculateDropdownPosition)
  }
})

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const insideTrigger = containerRef.value?.contains(target)
  const insideDropdown = Boolean(target.closest(`.${instanceId}`))
  if (!insideTrigger && !insideDropdown) closeDropdown()
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', updateTriggerRect, { capture: true })
  window.removeEventListener('resize', calculateDropdownPosition)
})

const isUsable = (proxy: Proxy) => {
  if (proxy.status !== 'active') return false
  if (!proxy.expires_at) return true
  const expiresAt = Date.parse(proxy.expires_at)
  return Number.isNaN(expiresAt) || expiresAt > Date.now()
}

const selectableProxies = computed(() => props.proxies.filter((proxy) =>
  proxy.id !== props.primaryProxyId && isUsable(proxy)
))

const filteredProxies = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return selectableProxies.value
  return selectableProxies.value.filter((proxy) =>
    proxy.name.toLowerCase().includes(query) || proxy.host.toLowerCase().includes(query)
  )
})

const selectedProxies = computed(() => {
  const selected = new Set(props.modelValue)
  return selectableProxies.value.filter((proxy) => selected.has(proxy.id))
})

const laneProxies = computed(() => {
  const selected = new Set(props.modelValue)
  return props.proxies.filter((proxy) =>
    isUsable(proxy) && (proxy.id === props.primaryProxyId || selected.has(proxy.id))
  )
})

const laneConcurrency = (index: number): number => {
  const laneCount = Math.max(1, laneProxies.value.length)
  const total = Math.max(1, props.accountConcurrency)
  const base = Math.floor(total / laneCount)
  return Math.max(1, base + (index < total % laneCount ? 1 : 0))
}

const defaultLane = (proxyID: number, index = 0): ProxyLaneConfig => ({
  proxy_id: proxyID,
  enabled: true,
  max_concurrency: laneConcurrency(index),
  weight: 1,
  timeout_seconds: 0,
  error_circuit_threshold: 0,
  circuit_cooldown_seconds: 60,
  fallback_order: index
})

const laneConfig = (proxyID: number): ProxyLaneConfig => {
  const found = props.laneConfigs.find((item) => item.proxy_id === proxyID)
  return found || defaultLane(proxyID, laneProxies.value.findIndex((proxy) => proxy.id === proxyID))
}

const unifiedLaneConfig = computed<ProxyLaneConfig>(() => {
  const firstProxy = laneProxies.value[0]
  return firstProxy ? laneConfig(firstProxy.id) : defaultLane(0)
})

const normalizedLaneConfigs = (shared = unifiedLaneConfig.value): ProxyLaneConfig[] => laneProxies.value.map((proxy, index) => ({
  ...defaultLane(proxy.id, index),
  enabled: shared.enabled,
  // The account concurrency field is the single capacity setting. The
  // runtime lane capacities are only an even frontend-derived split.
  max_concurrency: laneConcurrency(index),
  weight: shared.weight,
  timeout_seconds: shared.timeout_seconds,
  error_circuit_threshold: shared.error_circuit_threshold,
  circuit_cooldown_seconds: shared.circuit_cooldown_seconds,
  // Fallback order follows the selected proxy order and is never edited per lane.
  fallback_order: index,
  proxy_id: proxy.id
}))

const updateUnified = <K extends keyof Pick<ProxyLaneConfig, 'enabled' | 'weight' | 'timeout_seconds' | 'error_circuit_threshold' | 'circuit_cooldown_seconds'>>(
  key: K,
  value: Pick<ProxyLaneConfig, K>[K]
) => {
  emit('update:laneConfigs', normalizedLaneConfigs({
    ...unifiedLaneConfig.value,
    [key]: value
  }))
}

const updateUnifiedNumber = (
  key: 'weight' | 'timeout_seconds' | 'error_circuit_threshold' | 'circuit_cooldown_seconds',
  event: Event,
  minimum: number
) => {
  const raw = Number((event.target as HTMLInputElement).value)
  updateUnified(key, Math.max(minimum, Number.isFinite(raw) ? Math.trunc(raw) : minimum))
}

const updateLaneStrategy = (value: ProxyLaneStrategy) => emit('update:laneStrategy', value)

const selectedLabel = computed(() => selectedProxies.value.length > 0
  ? t('admin.accounts.proxyPoolSelected', { count: selectedProxies.value.length })
  : t('admin.accounts.proxyPoolEmpty'))

const toggleProxy = (id: number) => {
  const selected = new Set(props.modelValue)
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
  emit('update:modelValue', Array.from(selected))
}

const selectNoProxy = (event: Event) => {
  // Empty array is the single source of truth for “no additional proxy”.
  // It is mutually exclusive with every concrete proxy selection.
  // A native checkbox toggles itself before change fires. Force it back to the
  // controlled empty-array state so an attempted uncheck cannot create a
  // transient third visual state.
  const input = event.target as HTMLInputElement
  input.checked = true
  emit('update:modelValue', [])
}

const selectAllFiltered = () => {
  const selected = new Set(props.modelValue)
  filteredProxies.value.forEach((proxy) => selected.add(proxy.id))
  emit('update:modelValue', Array.from(selected))
}

const clearSelection = () => emit('update:modelValue', [])

watch(
  () => [props.primaryProxyId, props.proxies, props.modelValue] as const,
  () => {
    // Proxy lists are loaded asynchronously by the account page. Do not erase
    // persisted selections during the brief empty-list phase.
    if (props.proxies.length === 0) return
    const usable = new Set(selectableProxies.value.map((proxy) => proxy.id))
    const normalized = Array.from(new Set(props.modelValue)).filter((id) => usable.has(id))
    if (normalized.length !== props.modelValue.length || normalized.some((id, index) => id !== props.modelValue[index])) {
      emit('update:modelValue', normalized)
    }
  },
  { deep: true, immediate: true }
)

watch(
  () => [props.primaryProxyId, props.modelValue, props.proxies, props.laneConfigs, props.accountConcurrency] as const,
  () => {
    if (props.proxies.length === 0 || laneProxies.value.length === 0) return
    // Older saved accounts may contain different values per lane. Treat the
    // first lane as the canonical unified value and converge the form model
    // immediately, so the legacy independent settings cannot be re-saved.
    const normalized = normalizedLaneConfigs()
    if (JSON.stringify(normalized) !== JSON.stringify(props.laneConfigs)) {
      emit('update:laneConfigs', normalized)
    }
  },
  { deep: true, immediate: true }
)
</script>

<style scoped>
.select-trigger {
  @apply flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm text-gray-900 transition-all duration-200 hover:border-gray-300 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-dark-500;
}

.select-trigger-open {
  @apply border-primary-500 ring-2 ring-primary-500/30;
}

.select-trigger-disabled {
  @apply cursor-not-allowed bg-gray-100 opacity-60 dark:bg-dark-900;
}

.select-value {
  @apply flex-1 truncate text-left;
}

.select-icon {
  @apply shrink-0 text-gray-400 dark:text-dark-400;
}

.proxy-pool-dropdown-portal {
  @apply overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl shadow-black/10 dark:border-dark-700 dark:bg-dark-800 dark:shadow-black/30;
}

.proxy-pool-header {
  @apply flex items-center gap-3 border-b border-gray-100 px-3 py-2 dark:border-dark-700;
}

.proxy-pool-search {
  @apply flex min-w-0 flex-1 items-center gap-2;
}

.proxy-pool-search-input {
  @apply min-w-0 flex-1 bg-transparent text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-gray-100 dark:placeholder:text-dark-400;
}

.proxy-pool-options {
  @apply max-h-64 overflow-y-auto py-1;
}

.proxy-pool-option {
  @apply flex cursor-pointer items-center gap-3 px-3 py-2.5 transition-colors hover:bg-gray-50 dark:hover:bg-dark-700;
}

.proxy-pool-option-selected {
  @apply bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300;
}

.proxy-pool-footer {
  @apply flex items-center gap-2 border-t border-gray-100 bg-gray-50/80 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/50;
}

.proxy-pool-action {
  @apply rounded-lg px-2 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-white hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-primary-300;
}

.select-dropdown-enter-active,
.select-dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.select-dropdown-enter-from,
.select-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
