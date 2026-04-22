export default defineAppConfig({
  ui: {
    colors: {
      primary: 'brand',
      neutral: 'slate'
    },
    button: {
      slots: {
        base: 'cursor-pointer'
      }
    },
    formField: {
      slots: {
        description: 'text-xs text-muted',
        hint: 'text-xs text-muted',
        help: 'mt-2 text-xs text-muted'
      }
    },
    tabs: {
      slots: {
        trigger: 'cursor-pointer'
      }
    }
  }
})
