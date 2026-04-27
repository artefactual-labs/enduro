# Enduro Frontend

Nuxt 4 frontend for the new Enduro UI.

## Development

Install dependencies:

```bash
npm install
```

Start the dev server:

```bash
npm run dev
```

Useful commands:

```bash
npm run typecheck
npm run lint
npm test
npm run build
```

## Application structure

This is a Nuxt 4 single-page application. Server-side rendering is disabled in
[`nuxt.config.ts`](./nuxt.config.ts), so routes render in the browser and API
requests are made from the client.

The app follows Nuxt's `app/` directory conventions:

- `app/pages` defines file-based routes.
- `app/components` contains auto-imported Vue components.
- `app/composables` contains UI state and interaction logic.
- `app/loaders` contains route data loaders.
- `app/plugins` wires app-level services such as the Enduro API client.

Route-owned read data should be loaded with Vue Router Data Loaders from
`app/loaders` and exported by the page that uses them. Mutations and explicit
user actions should stay in composables or component event handlers and then
reload the relevant loader data when needed. Pass the loader `AbortSignal` to
API calls so navigation changes can cancel stale requests.

## Theming

The frontend uses Nuxt UI v4 and Tailwind CSS. Treat theme colors as semantic roles, not as fixed brand names.

- Use Nuxt UI semantic colors in components: `primary`, `neutral`, `success`, `warning`, `error`, `info`.
- Do not introduce new component styles tied directly to `purple` or another temporary brand hue.
- The current branded palette lives in [`app/assets/css/main.css`](./app/assets/css/main.css) as `--color-brand-*`.
- Nuxt UI `primary` is mapped to `brand` in [`app/app.config.ts`](./app/app.config.ts).

### Preferred pattern

1. Use Nuxt UI props and semantic utilities first.
2. Derive any custom app tokens from Nuxt UI tokens.
3. Add one-off CSS only when the component cannot be expressed cleanly through Nuxt UI.

Examples:

- Prefer `color="primary"` over custom button colors.
- Prefer `text-primary` or `border-default` over raw hex values.
- For branded custom surfaces, derive from app tokens such as `--app-brand-strong` and `--app-on-brand`.

### Dark mode

Dark mode should be token-driven, not component-by-component.

- Use Nuxt UI tokens such as `--ui-primary`, `--ui-bg`, and `--ui-text-inverted` as the mode-aware seam.
- Keep app-specific tokens in `main.css` as a thin wrapper around Nuxt UI tokens.
- Avoid separate ad hoc dark-mode color choices in components unless the component has a strong visual reason.

This keeps future brand changes localized to the theme layer instead of requiring broad component rewrites.

### Rules of thumb

- In Vue templates, prefer semantic props and classes.
- In CSS, prefer `brand` or app-level semantic tokens, never `purple`.
- Avoid hardcoded hex values for branded UI.
- If a future rebrand happens, the goal should be to change tokens, not component code.
