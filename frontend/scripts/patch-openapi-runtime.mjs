import { readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const runtimePath = path.resolve(scriptDir, '../app/openapi-generator/runtime.ts')

// Nuxt enables TypeScript's noImplicitOverride option. OpenAPI Generator's
// typescript-fetch runtime extends Error with a cause parameter property, so it
// needs the explicit override modifier to pass Nuxt type checking.
const replacements = [
  ['constructor(public cause: Error, msg?: string) {', 'constructor(public override cause: Error, msg?: string) {'],
  // These capture groups always participate in their successful regex matches.
  // OpenAPI Generator 7.26 needs assertions for noUncheckedIndexedAccess.
  ['decodeURIComponent(match[1])', 'decodeURIComponent(match[1]!)'],
  ['const name = match[1].toLowerCase();', 'const name = match[1]!.toLowerCase();'],
  ['match[3].trim()', 'match[3]!.trim()']
]

let source = await readFile(runtimePath, 'utf8')
for (const [needle, replacement] of replacements) {
  if (source.includes(replacement)) continue
  if (!source.includes(needle)) {
    throw new Error(`Expected to find ${needle} in ${runtimePath}`)
  }
  source = source.replace(needle, replacement)
}

await writeFile(runtimePath, source)
