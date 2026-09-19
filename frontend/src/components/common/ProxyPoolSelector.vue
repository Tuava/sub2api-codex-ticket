<template>
  <div class="space-y-2">
    <button
      type="button"
      class="select-trigger"
      :class="disabled && 'select-trigger-disabled'"
      :disabled="disabled"
      @click="isOpen = !isOpen"
    >
      <span class="select-value">
        {{ selectedLabel }}
      </span>
      <Icon
        name="chevronDown"
        size="md"
        :class="['transition-transform duration-200', isOpen && 'rotate-180']"
      />
    </button>

    <div
      v-if="isOpen"
      class="max-h-64 overflow-y-auto rounded-lg border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-500 dark:bg-dark-700"
    >
      <div class="mb-2 flex items-center gap-2 rounded-md bg-gray-50 px-2 dark:bg-dark-600">
        <Icon name="search" size="sm" class="text-gray-400" />
        <input
          v-model="searchQuery"
          type="text"
          class="min-w-0 flex-1 bg-transparent py-2 text-sm outline-none dark:text-white"
          :placeholder="t('admin.proxies.searchProxies')"
        />
      </div>

      <label
        v-for="proxy in filteredProxies"
        :key="proxy.id"
        class="flex cursor-pointer items-center gap-3 rounded-md px-2 py-2 hover:bg-gray-50 dark:hover:bg-dark-600"
      >
        <input
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="modelValue.includes(proxy.id)"
          @change="toggleProxy(proxy.id)"
        />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-medium text-gray-800 dark:text-gray-100">
            {{ proxy.name }}
          </span>
          <span class="block truncate text-xs text-gray-500 dark:text-gray-400">
            {{ proxy.protocol }}://{{ proxy.host }}:{{ proxy.port }}
          </span>
        </span>
      </label>

      <p v-if="filteredProxies.length === 0" class="px-2 py-3 text-center text-sm text-gray-500">
        {{ t('common.noOptionsFound') }}
      </p>
    </div>

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
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Proxy } from '@/types'

const props = withDefaults(defineProps<{
  modelValue: number[]
  proxies: Proxy[]
  primaryProxyId?: number | null
  disabled?: boolean
}>(), {
  primaryProxyId: null,
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const { t } = useI18n()
const isOpen = ref(false)
const searchQuery = ref('')

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

const selectedLabel = computed(() => selectedProxies.value.length > 0
  ? t('admin.accounts.proxyPoolSelected', { count: selectedProxies.value.length })
  : t('admin.accounts.proxyPoolEmpty'))

const toggleProxy = (id: number) => {
  const selected = new Set(props.modelValue)
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
  emit('update:modelValue', Array.from(selected))
}

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
</script>

<style scoped>
.select-trigger {
  @apply flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm text-gray-900 transition-all duration-200 hover:border-gray-300 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-100 dark:hover:border-dark-500;
}

.select-trigger-disabled {
  @apply cursor-not-allowed bg-gray-100 opacity-60 dark:bg-dark-900;
}

.select-value {
  @apply flex-1 truncate text-left;
}
</style>
