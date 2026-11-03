import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";

const base = "http://enduro:9000";
async function get(path) {
  const response = await fetch(new URL(path, base), {
    signal: AbortSignal.timeout(10_000),
  });
  assert.equal(response.status, 200, `${path}: ${await response.clone().text()}`);
  return response;
}

// Listing collections exercises the API and the schema migrated at startup.
await (await get("/collection")).json();
const response = await get("/");
assert.match(response.headers.get("content-type"), /text\/html/);
const html = await response.text();
const assets = [...html.matchAll(/(?:src|href)=["']([^"']*\/_nuxt\/[^"']+)["']/g)];
assert.ok(assets.length > 0, "dashboard must reference embedded Nuxt assets");
for (const [, asset] of assets) {
  const response = await get(asset);
  assert.doesNotMatch(response.headers.get("content-type") ?? "", /text\/html/);
  assert.ok((await response.arrayBuffer()).byteLength > 0, asset);
}
await mkdir("/artifacts", { recursive: true });
await writeFile("/artifacts/startup.json", JSON.stringify({
  collectionAPI: "ok",
  dashboard: "ok",
  assets: assets.map(([, asset]) => asset),
}, null, 2) + "\n");
