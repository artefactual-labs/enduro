import {
  defaultDashboardUiOptions,
  parseSavedDashboardUiOptions,
  type DashboardUiOptions
} from './useDashboardUiOptions.helpers'

const dashboardUiOptionsStorageKey = 'enduroDashboardUiOptions'

type SetDashboardUiOptionOptions = {
  persist?: boolean
}

function cloneOptions(options: DashboardUiOptions): DashboardUiOptions {
  return { ...options }
}

export function useDashboardUiOptions() {
  const options = useState<DashboardUiOptions>('dashboard-ui-options', () => cloneOptions(defaultDashboardUiOptions))
  const hasLoaded = useState('dashboard-ui-options-loaded', () => false)

  function persistOptions() {
    if (!import.meta.client || !hasLoaded.value) return
    localStorage.setItem(dashboardUiOptionsStorageKey, JSON.stringify(options.value))
  }

  function loadOptions() {
    if (!import.meta.client || hasLoaded.value) return

    const saved = localStorage.getItem(dashboardUiOptionsStorageKey)
    const parsed = parseSavedDashboardUiOptions(saved)

    if (saved !== null && parsed === null) {
      localStorage.removeItem(dashboardUiOptionsStorageKey)
    }

    if (parsed) {
      options.value = {
        ...defaultDashboardUiOptions,
        ...parsed
      }
    }

    hasLoaded.value = true
  }

  function setCollectionsSearchOpen(value: boolean, setOptions: SetDashboardUiOptionOptions = {}) {
    options.value = {
      ...options.value,
      collectionsSearchOpen: value
    }

    if (setOptions.persist !== false) {
      persistOptions()
    }
  }

  const collectionsSearchOpen = computed({
    get: () => options.value.collectionsSearchOpen,
    set: setCollectionsSearchOpen
  })

  onMounted(loadOptions)

  return {
    collectionsSearchOpen,
    setCollectionsSearchOpen
  }
}
