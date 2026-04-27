export function useCollectionWorkflowAutoReload() {
  return useState<boolean>('collection-workflow-auto-reload', () => true)
}
