const FALLBACK_VERSION_LABEL = '(development snapshot)'
const SEMVER_LIKE_REGEX = /^\d+\.\d+\.\d+(?:[-+].+)?$/

export default defineNuxtPlugin({
  name: 'enduro-version',
  dependsOn: ['enduro-api'],
  async setup() {
    const versionLabel = useState<string>('enduroVersion', () => '')

    try {
      const enduroApi = useEnduroApi()
      const headerVersion = await enduroApi.system.versionHeader()

      if (!headerVersion) {
        versionLabel.value = FALLBACK_VERSION_LABEL
        return
      }

      const normalizedVersion = headerVersion.replace(/^v/, '')
      const isSemverLike = SEMVER_LIKE_REGEX.test(normalizedVersion)

      versionLabel.value = isSemverLike
        ? `v${normalizedVersion}`
        : FALLBACK_VERSION_LABEL
    } catch {
      versionLabel.value = FALLBACK_VERSION_LABEL
    }
  }
})
