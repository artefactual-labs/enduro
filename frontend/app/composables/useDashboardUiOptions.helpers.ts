export type DashboardUiOptions = {
  collectionsSearchOpen: boolean
}

export const defaultDashboardUiOptions: DashboardUiOptions = {
  collectionsSearchOpen: false
}

export function normalizeDashboardUiOptions(value: unknown): DashboardUiOptions {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return { ...defaultDashboardUiOptions }
  }

  const options = value as Partial<DashboardUiOptions>

  return {
    collectionsSearchOpen: typeof options.collectionsSearchOpen === 'boolean'
      ? options.collectionsSearchOpen
      : defaultDashboardUiOptions.collectionsSearchOpen
  }
}

export function parseSavedDashboardUiOptions(value: string | null): DashboardUiOptions | null {
  if (value === null) return null

  try {
    return normalizeDashboardUiOptions(JSON.parse(value))
  } catch {
    return null
  }
}
