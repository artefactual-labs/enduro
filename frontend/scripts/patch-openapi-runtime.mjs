import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const runtimePath = path.resolve(scriptDir, '../app/openapi-generator/runtime.ts')

// Nuxt enables TypeScript's noImplicitOverride option. OpenAPI Generator's
// typescript-fetch runtime extends Error with a cause parameter property, so it
// needs the explicit override modifier to pass Nuxt type checking.
const needle = 'constructor(public cause: Error, msg?: string) {'
const replacement = 'constructor(public override cause: Error, msg?: string) {'

const source = await readFile(runtimePath, 'utf8')

if (source.includes(replacement)) {
  process.exit(0)
}

if (!source.includes(needle)) {
  throw new Error(`Expected to find FetchError constructor in ${runtimePath}`)
}

await writeFile(runtimePath, source.replace(needle, replacement))
