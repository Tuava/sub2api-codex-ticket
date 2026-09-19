<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    :close-on-click-outside="!showBulkConfig"
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <div class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportProfilesTitle') }}
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.dataImportProfilesHint') }}
        </p>

        <div v-if="profiles.length" class="mt-3 flex flex-wrap gap-2" data-testid="import-profile-list">
          <button
            v-for="profile in profiles"
            :key="profile.id"
            type="button"
            class="rounded-full border px-3 py-1.5 text-xs font-medium transition-colors"
            :class="activeProfileId === profile.id
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300'"
            :title="profileScopeLabel(profile)"
            :data-testid="`import-profile-${profile.id}`"
            @click="applyProfile(profile)"
          >
            {{ profile.name }}
          </button>
        </div>
        <div v-else class="mt-3 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.dataImportProfilesEmpty') }}
        </div>

        <div class="mt-3 flex flex-col gap-2 sm:flex-row">
          <input
            v-model="profileName"
            data-testid="import-profile-name"
            class="input flex-1"
            :placeholder="t('admin.accounts.dataImportProfileNamePlaceholder')"
          />
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            data-testid="save-import-profile"
            @click="saveProfileAsNew"
          >
            {{ t('admin.accounts.dataImportProfileSaveAs') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            data-testid="overwrite-import-profile"
            :disabled="!activeProfileId"
            @click="overwriteActiveProfile"
          >
            {{ t('admin.accounts.dataImportProfileOverwrite') }}
          </button>
        </div>

        <div class="mt-2 flex flex-wrap gap-2">
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="!activeProfileId"
            @click="exportActiveProfile"
          >
            {{ t('admin.accounts.dataImportProfileExport') }}
          </button>
          <button type="button" class="btn btn-secondary btn-sm" @click="profileFileInput?.click()">
            {{ t('admin.accounts.dataImportProfileImport') }}
          </button>
          <button
            type="button"
            class="btn btn-danger btn-sm"
            :disabled="!activeProfileId"
            data-testid="delete-import-profile"
            @click="deleteActiveProfile"
          >
            {{ t('common.delete') }}
          </button>
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed px-4 py-3 transition-colors"
          :class="dragActive
            ? 'border-primary-400 bg-primary-50/70 dark:border-primary-500 dark:bg-primary-900/20'
            : 'border-gray-300 bg-gray-50 dark:border-dark-600 dark:bg-dark-800'"
          @dragenter.prevent="handleDragEnter"
          @dragover.prevent
          @dragleave.prevent="handleDragLeave"
          @drop.prevent="handleDrop"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200" :title="fileListTitle">
              {{ selectedFilesLabel || t('admin.accounts.dataImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">
              JSON (.json)
              <span v-if="files.length > 1"> · {{ fileListTitle }}</span>
            </div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="openFilePicker">
            {{ t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          data-testid="account-import-file-input"
          type="file"
          class="hidden"
          accept="application/json,.json"
          multiple
          @change="handleFileChange"
        />
        <input
          ref="profileFileInput"
          data-testid="profile-import-file-input"
          type="file"
          class="hidden"
          accept="application/json,.json"
          @change="handleProfileFileImport"
        />
      </div>

      <SmartProxyAssignmentPanel
        v-model="smartProxyOptions"
        :show-action="false"
        show-enable
      />

      <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.accounts.dataImportBulkConfigTitle') }}
            </div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.dataImportBulkConfigHint') }}
            </p>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            data-testid="open-import-bulk-config"
            :disabled="files.length === 0"
            @click="showBulkConfig = true"
          >
            {{ postImportUpdates ? t('common.edit') : t('admin.accounts.dataImportConfigure') }}
          </button>
        </div>
        <div
          v-if="postImportUpdates"
          class="mt-3 rounded-lg bg-emerald-50 px-3 py-2 text-xs text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300"
          data-testid="import-bulk-config-ready"
        >
          {{ t('admin.accounts.dataImportBulkConfigReady', { count: Object.keys(postImportUpdates).length }) }}
          <button type="button" class="ml-2 underline" @click="postImportUpdates = null">
            {{ t('common.clear') }}
          </button>
        </div>
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <BulkEditAccountModal
    :show="showBulkConfig"
    :account-ids="[]"
    :selected-platforms="importPlatforms"
    :selected-types="importTypes"
    :target="{
      mode: 'selected',
      previewCount: importAccountCount,
      selectedPlatforms: importPlatforms,
      selectedTypes: importTypes
    }"
    :proxies="proxies"
    :groups="groups"
    config-only
    @close="showBulkConfig = false"
    @configured="handleBulkConfigured"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SmartProxyAssignmentPanel from '@/components/account/SmartProxyAssignmentPanel.vue'
import BulkEditAccountModal from '@/components/account/BulkEditAccountModal.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type {
  AccountPlatform,
  AccountType,
  AdminDataImportResult,
  AdminDataPayload,
  AdminGroup,
  Proxy,
  SmartProxyAssignmentOptions
} from '@/types'

interface Props {
  show: boolean
  proxies?: Proxy[]
  groups?: AdminGroup[]
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

interface ImportBatchProfile {
  id: string
  name: string
  post_import_updates: Record<string, unknown> | null
  smart_proxy_assignment: SmartProxyAssignmentOptions
  platforms: AccountPlatform[]
  account_types: AccountType[]
  updated_at: string
}

interface ImportBatchProfileFile {
  type: 'sub2api-account-import-profile'
  version: 1
  exported_at: string
  profile: ImportBatchProfile
}

const props = withDefaults(defineProps<Props>(), {
  proxies: () => [],
  groups: () => []
})
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const IMPORT_PROFILE_STORAGE_KEY = 'sub2api:admin:account-import-profiles:v1'
const files = ref<File[]>([])
const dragDepth = ref(0)
const dragActive = computed(() => dragDepth.value > 0)
const hasCreatedData = ref(false)
const result = ref<AdminDataImportResult | null>(null)
const parsedPayloads = ref<AdminDataPayload[]>([])
const showBulkConfig = ref(false)
const postImportUpdates = ref<Record<string, unknown> | null>(null)
const defaultSmartProxyOptions = (): SmartProxyAssignmentOptions => ({
  enabled: false,
  proxy_count: 2,
  test_latency: true,
  prefer_low_latency: true,
  low_latency_limit: 0,
  weighted_by_load: true
})
const smartProxyOptions = ref<SmartProxyAssignmentOptions>(defaultSmartProxyOptions())
const profiles = ref<ImportBatchProfile[]>([])
const activeProfileId = ref('')
const profileName = ref('')

const fileInput = ref<HTMLInputElement | null>(null)
const profileFileInput = ref<HTMLInputElement | null>(null)
const importedAccounts = computed(() => parsedPayloads.value.flatMap((payload) => payload.accounts))
const importAccountCount = computed(() => importedAccounts.value.length)
const importPlatforms = computed<AccountPlatform[]>(() => Array.from(new Set(
  importedAccounts.value.map((account) => account.platform).filter(Boolean)
)))
const importTypes = computed<AccountType[]>(() => Array.from(new Set(
  importedAccounts.value.map((account) => account.type).filter(Boolean)
)))
const selectedFilesLabel = computed(() => {
  if (files.value.length === 0) return ''
  if (files.value.length === 1) return files.value[0]?.name || ''
  return t('admin.accounts.selectedCount', { count: files.value.length })
})
const fileListTitle = computed(() => files.value.map((item) => item.name).join(', '))

const errorItems = computed(() => result.value?.errors || [])

const cloneJSON = <T,>(value: T): T => JSON.parse(JSON.stringify(value)) as T

const makeProfileID = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `profile-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const normalizeSmartProxyOptions = (raw: unknown): SmartProxyAssignmentOptions => {
  const value = raw && typeof raw === 'object' ? raw as Partial<SmartProxyAssignmentOptions> : {}
  return {
    enabled: value.enabled === true,
    proxy_count: Math.min(32, Math.max(1, Math.trunc(Number(value.proxy_count) || 2))),
    test_latency: value.test_latency !== false,
    prefer_low_latency: value.prefer_low_latency !== false,
    low_latency_limit: Math.max(0, Math.trunc(Number(value.low_latency_limit) || 0)),
    weighted_by_load: value.weighted_by_load !== false
  }
}

const normalizeImportProfile = (raw: unknown): ImportBatchProfile | null => {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null
  const value = raw as Partial<ImportBatchProfile>
  const name = typeof value.name === 'string' ? value.name.trim() : ''
  if (!name) return null
  const updates = value.post_import_updates && typeof value.post_import_updates === 'object' && !Array.isArray(value.post_import_updates)
    ? cloneJSON(value.post_import_updates)
    : null
  return {
    id: typeof value.id === 'string' && value.id.trim() ? value.id.trim() : makeProfileID(),
    name,
    post_import_updates: updates,
    smart_proxy_assignment: normalizeSmartProxyOptions(value.smart_proxy_assignment),
    platforms: Array.isArray(value.platforms) ? Array.from(new Set(value.platforms)) : [],
    account_types: Array.isArray(value.account_types) ? Array.from(new Set(value.account_types)) : [],
    updated_at: typeof value.updated_at === 'string' ? value.updated_at : new Date().toISOString()
  }
}

const loadProfiles = () => {
  try {
    const raw = localStorage.getItem(IMPORT_PROFILE_STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    profiles.value = Array.isArray(parsed)
      ? parsed.map(normalizeImportProfile).filter((profile): profile is ImportBatchProfile => profile !== null)
      : []
  } catch {
    profiles.value = []
  }
}

const persistProfiles = () => {
  try {
    localStorage.setItem(IMPORT_PROFILE_STORAGE_KEY, JSON.stringify(profiles.value))
  } catch {
    appStore.showWarning(t('admin.accounts.dataImportProfileStorageFailed'))
  }
}

const profileScopeLabel = (profile: ImportBatchProfile) => {
  const platforms = profile.platforms.length ? profile.platforms.join(', ') : t('common.all')
  const types = profile.account_types.length ? profile.account_types.join(', ') : t('common.all')
  return `${platforms} / ${types}`
}

const currentProfileSnapshot = (name: string, id = makeProfileID()): ImportBatchProfile => ({
  id,
  name,
  post_import_updates: postImportUpdates.value ? cloneJSON(postImportUpdates.value) : null,
  smart_proxy_assignment: cloneJSON(smartProxyOptions.value),
  platforms: [...importPlatforms.value],
  account_types: [...importTypes.value],
  updated_at: new Date().toISOString()
})

const hasCurrentProfileConfig = () => postImportUpdates.value !== null || smartProxyOptions.value.enabled === true

const uniqueImportedProfileName = (baseName: string) => {
  const names = new Set(profiles.value.map((profile) => profile.name))
  if (!names.has(baseName)) return baseName
  for (let i = 2; ; i += 1) {
    const candidate = `${baseName} (${i})`
    if (!names.has(candidate)) return candidate
  }
}

const saveProfileAsNew = () => {
  const name = profileName.value.trim()
  if (!name) {
    appStore.showError(t('admin.accounts.dataImportProfileNameRequired'))
    return
  }
  if (!hasCurrentProfileConfig()) {
    appStore.showError(t('admin.accounts.dataImportProfileEmptyConfig'))
    return
  }
  if (profiles.value.some((profile) => profile.name === name)) {
    appStore.showError(t('admin.accounts.dataImportProfileNameExists'))
    return
  }
  const profile = currentProfileSnapshot(name)
  profiles.value.push(profile)
  activeProfileId.value = profile.id
  persistProfiles()
  appStore.showSuccess(t('admin.accounts.dataImportProfileSaved', { name }))
}

const overwriteActiveProfile = () => {
  const index = profiles.value.findIndex((profile) => profile.id === activeProfileId.value)
  if (index < 0) return
  const name = profileName.value.trim()
  if (!name) {
    appStore.showError(t('admin.accounts.dataImportProfileNameRequired'))
    return
  }
  if (profiles.value.some((profile, profileIndex) => profileIndex !== index && profile.name === name)) {
    appStore.showError(t('admin.accounts.dataImportProfileNameExists'))
    return
  }
  profiles.value[index] = currentProfileSnapshot(name, activeProfileId.value)
  persistProfiles()
  appStore.showSuccess(t('admin.accounts.dataImportProfileUpdated', { name }))
}

const applyProfile = (profile: ImportBatchProfile) => {
  activeProfileId.value = profile.id
  profileName.value = profile.name
  postImportUpdates.value = profile.post_import_updates ? cloneJSON(profile.post_import_updates) : null
  smartProxyOptions.value = cloneJSON(profile.smart_proxy_assignment)
  const currentPlatforms = new Set(importPlatforms.value)
  const currentTypes = new Set(importTypes.value)
  const scopeMismatch = files.value.length > 0 && (
    profile.platforms.some((platform) => !currentPlatforms.has(platform)) ||
    profile.account_types.some((accountType) => !currentTypes.has(accountType))
  )
  if (scopeMismatch) {
    appStore.showWarning(t('admin.accounts.dataImportProfileScopeWarning'))
  }
  appStore.showSuccess(t('admin.accounts.dataImportProfileApplied', { name: profile.name }))
}

const deleteActiveProfile = () => {
  const profile = profiles.value.find((item) => item.id === activeProfileId.value)
  if (!profile) return
  if (!confirm(t('admin.accounts.dataImportProfileDeleteConfirm', { name: profile.name }))) return
  profiles.value = profiles.value.filter((item) => item.id !== profile.id)
  activeProfileId.value = ''
  profileName.value = ''
  persistProfiles()
}

const exportActiveProfile = () => {
  const profile = profiles.value.find((item) => item.id === activeProfileId.value)
  if (!profile) return
  const payload: ImportBatchProfileFile = {
    type: 'sub2api-account-import-profile',
    version: 1,
    exported_at: new Date().toISOString(),
    profile
  }
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `sub2api-import-profile-${profile.name.replace(/[^\w\-.\u4e00-\u9fff]+/g, '-')}.json`
  anchor.click()
  URL.revokeObjectURL(url)
}

const handleProfileFileImport = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const sourceFile = input.files?.[0]
  input.value = ''
  if (!sourceFile) return
  try {
    const parsed = JSON.parse(await readFileAsText(sourceFile)) as Partial<ImportBatchProfileFile>
    if (parsed.type !== 'sub2api-account-import-profile' || parsed.version !== 1) {
      throw new Error('unsupported profile file')
    }
    const imported = normalizeImportProfile(parsed.profile)
    if (!imported) throw new Error('invalid profile')
    imported.id = makeProfileID()
    imported.name = uniqueImportedProfileName(imported.name)
    profiles.value.push(imported)
    persistProfiles()
    applyProfile(imported)
    appStore.showSuccess(t('admin.accounts.dataImportProfileImported', { name: imported.name }))
  } catch {
    appStore.showError(t('admin.accounts.dataImportProfileImportFailed'))
  }
}

watch(
  () => props.show,
  (open) => {
    if (open) {
      files.value = []
      dragDepth.value = 0
      hasCreatedData.value = false
      result.value = null
      parsedPayloads.value = []
      showBulkConfig.value = false
      postImportUpdates.value = null
      smartProxyOptions.value = defaultSmartProxyOptions()
      activeProfileId.value = ''
      profileName.value = ''
      loadProfiles()
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
  },
  { immediate: true }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const handleFileChange = async (event: Event) => {
  const target = event.target as HTMLInputElement
  await setSelectedFiles(target.files)
  target.value = ''
}

const handleClose = () => {
  if (importing.value) return
  if (hasCreatedData.value) {
    hasCreatedData.value = false
    emit('imported')
  }
  emit('close')
}

const isJsonFile = (sourceFile: File) => {
  const name = sourceFile.name.toLowerCase()
  return name.endsWith('.json') || sourceFile.type === 'application/json'
}

const setSelectedFiles = async (sourceFiles: FileList | File[] | null | undefined) => {
  if (importing.value) return
  const incoming = Array.from(sourceFiles || [])
  const picked = incoming.filter(isJsonFile)
  if (!picked.length) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }
  if (picked.length < incoming.length) {
    appStore.showWarning(
      t('admin.accounts.dataImportIgnoredFiles', { count: incoming.length - picked.length })
    )
  }
  const activeProfile = profiles.value.find((profile) => profile.id === activeProfileId.value)
  files.value = picked
  result.value = null
  parsedPayloads.value = []
  for (const sourceFile of picked) {
    try {
      const parsed = JSON.parse(await readFileAsText(sourceFile))
      if (isValidDataPayload(parsed)) parsedPayloads.value.push(parsed)
    } catch {
      // Detailed validation remains in handleImport so the selected file is
      // still visible and the user receives the existing localized error.
    }
  }
  if (activeProfile) {
    postImportUpdates.value = activeProfile.post_import_updates
      ? cloneJSON(activeProfile.post_import_updates)
      : null
    smartProxyOptions.value = cloneJSON(activeProfile.smart_proxy_assignment)
  } else {
    postImportUpdates.value = null
  }
}

const handleDragEnter = () => {
  if (importing.value) return
  dragDepth.value += 1
}

const handleDragLeave = () => {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}

const handleDrop = async (event: DragEvent) => {
  dragDepth.value = 0
  if (importing.value) return
  await setSelectedFiles(event.dataTransfer?.files)
}

const readFileAsText = async (sourceFile: File): Promise<string> => {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }

  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }

  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

const SUPPORTED_DATA_TYPES = ['sub2api-data', 'sub2api-bundle']
const SUPPORTED_DATA_VERSION = 1

// 与后端 validateDataHeader 对齐:合并前逐文件校验,避免坏文件混入合并 payload 后
// 报错无法定位来源,或绕过后端本会对单文件做的 type/version 检查。
const isValidDataPayload = (payload: unknown): payload is AdminDataPayload => {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return false
  const candidate = payload as Record<string, unknown>
  if (
    candidate.type !== undefined &&
    candidate.type !== '' &&
    !SUPPORTED_DATA_TYPES.includes(candidate.type as string)
  ) {
    return false
  }
  if (
    candidate.version !== undefined &&
    candidate.version !== 0 &&
    candidate.version !== SUPPORTED_DATA_VERSION
  ) {
    return false
  }
  return Array.isArray(candidate.proxies) && Array.isArray(candidate.accounts)
}

const mergeDataPayloads = (payloads: AdminDataPayload[]): AdminDataPayload => {
  const [firstPayload] = payloads
  if (payloads.length === 1 && firstPayload) return firstPayload

  return {
    type: payloads.find((item) => typeof item.type === 'string')?.type,
    version: payloads.find((item) => typeof item.version === 'number')?.version,
    exported_at: new Date().toISOString(),
    proxies: payloads.flatMap((item) => item.proxies),
    accounts: payloads.flatMap((item) => item.accounts),
    skipped_shadows: payloads.reduce((sum, item) => {
      const count = Number(item.skipped_shadows || 0)
      return Number.isFinite(count) ? sum + count : sum
    }, 0)
  }
}

const handleBulkConfigured = (updates: Record<string, unknown>) => {
  postImportUpdates.value = updates
  showBulkConfig.value = false
}

const handleImport = async () => {
  if (files.value.length === 0) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  try {
    const dataPayloads: AdminDataPayload[] = []
    for (const sourceFile of files.value) {
      let parsed: unknown
      try {
        parsed = JSON.parse(await readFileAsText(sourceFile))
      } catch {
        appStore.showError(
          t('admin.accounts.dataImportParseFailedFile', { name: sourceFile.name })
        )
        return
      }
      if (!isValidDataPayload(parsed)) {
        appStore.showError(t('admin.accounts.dataImportInvalidFile', { name: sourceFile.name }))
        return
      }
      dataPayloads.push(parsed)
    }
    const dataPayload = mergeDataPayloads(dataPayloads)

    const res = await adminAPI.accounts.importData({
      data: dataPayload,
      skip_default_group_bind: true,
      post_import_updates: postImportUpdates.value ?? undefined,
      smart_proxy_assignment: smartProxyOptions.value.enabled
        ? smartProxyOptions.value
        : undefined
    })

    result.value = res

    const msgParams: Record<string, unknown> = {
      account_created: res.account_created,
      account_failed: res.account_failed,
      proxy_created: res.proxy_created,
      proxy_reused: res.proxy_reused,
      proxy_failed: res.proxy_failed,
      proxy_assigned: res.proxy_assigned || 0,
      proxy_assign_failed: res.proxy_assign_failed || 0,
      post_import_updated: res.post_import_updated || 0,
      post_import_failed: res.post_import_failed || 0,
    }
    if (
      res.account_failed > 0 ||
      res.proxy_failed > 0 ||
      (res.proxy_assign_failed || 0) > 0 ||
      (res.post_import_failed || 0) > 0
    ) {
      // 部分成功也创建了数据;弹窗关闭时通过 imported 通知父组件刷新列表
      if (res.account_created > 0 || res.proxy_created > 0) {
        hasCreatedData.value = true
      }
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.dataImportSuccess', msgParams))
      emit('imported')
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.dataImportFailed'))
  } finally {
    importing.value = false
  }
}
</script>
