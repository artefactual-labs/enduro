export default defineNuxtPlugin({
  name: 'enduro-monitor',
  dependsOn: ['enduro-api'],
  setup() {
    const monitor = useEnduroMonitor()
    monitor.start()
  }
})
