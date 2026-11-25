import assert from "node:assert/strict";
import { cp, mkdir, readFile, rename, writeFile } from "node:fs/promises";

const base = "http://enduro:9000";
const pipeline = "http://pipeline:8080";
async function get(path, server = base) {
  const response = await fetch(new URL(path, server), {
    signal: AbortSignal.timeout(10_000),
  });
  assert.equal(response.status, 200, `${path}: ${await response.clone().text()}`);
  return response;
}

async function submitBag(directory, excludeHiddenFiles) {
  const response = await fetch(`${base}/batch`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      path: `/runtime/batch/${directory}`,
      pipeline: "startup",
      transfer_type: "unzipped bag",
      exclude_hidden_files: excludeHiddenFiles,
    }),
    signal: AbortSignal.timeout(10_000),
  });
  assert.equal(response.status, 202, await response.text());
}

async function waitFor(description, read, ready) {
  let value;
  for (let attempt = 0; attempt < 120; attempt++) {
    value = await read();
    if (value && ready(value)) return value;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  assert.fail(`timed out waiting for ${description}: ${JSON.stringify(value)}`);
}

function waitForCollection(name, ready) {
  return waitFor(name, async () => {
    const result = await (await get(`/collection?name=${name}`)).json();
    return result.items?.find((item) => item.name === name);
  }, ready);
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

async function assertRejectedBag(name) {
  const collection = await waitForCollection(name, (item) => item.status === "error");
  const history = await (await get(`/collection/${collection.id}/workflow`)).json();
  const historyText = JSON.stringify(history);
  assert.match(historyText, /BagIt validation failed/);
  assert.match(historyText, /data\/\.DS_Store/);
  assert.ok(!collection.transfer_id, "incomplete bag must not be submitted");
  return history;
}

// Rename a complete directory on the shared mount to trigger one watcher event.
await cp("/fixture", "/runtime/watched/drop.part", { recursive: true });
await rename("/runtime/watched/drop.part", "/runtime/watched/drop");
const watcherHistory = await assertRejectedBag("drop");
assert.deepEqual(await (await get("/submissions", pipeline)).json(), []);
assert.equal(await readFile("/runtime/watched/drop/data/.DS_Store", "utf8"), "");

await submitBag("filtered", true);
const history = await assertRejectedBag("incomplete");
const sibling = await waitForCollection("sibling", (item) => ["done", "error"].includes(item.status));
assert.equal(sibling.status, "done");
assert.deepEqual((await (await get("/submissions", pipeline)).json()).map((s) => [s.name, s.type]), [
  ["sibling", "unzipped bag"],
]);

// The pipeline has one slot. Submitting the intact copy proves the rejected
// collection released it and the worker can hand off the original bag layout.
await waitFor("the first batch to finish", async () => (await get("/batch")).json(), (batch) => !batch.running);
await submitBag("unfiltered", false);
const intact = await waitForCollection("intact", (item) => item.transfer_id || item.status === "error");
const submissions = await (await get("/submissions", pipeline)).json();
assert.deepEqual(submissions.map((s) => s.name), ["sibling", "intact"]);
assert.deepEqual(submissions[1], {
  name: "intact",
  type: "unzipped bag",
  declaration: "BagIt-Version: 1.0\nTag-File-Character-Encoding: UTF-8\n",
  manifest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  data/.DS_Store\n",
  hiddenFile: "",
});
assert.equal(intact.transfer_id, "71bd0ab3-e496-4ec0-9146-048abf471df4");

await mkdir("/artifacts", { recursive: true });
await writeFile("/artifacts/watcher-bag-history.json", JSON.stringify(watcherHistory, null, 2) + "\n");
await writeFile("/artifacts/bag-history.json", JSON.stringify(history, null, 2) + "\n");
await writeFile("/artifacts/bag-submission.json", JSON.stringify(submissions[1], null, 2) + "\n");
await writeFile("/artifacts/startup.json", JSON.stringify({
  collectionAPI: "ok",
  dashboard: "ok",
  bagValidation: "ok",
  assets: assets.map(([, asset]) => asset),
}, null, 2) + "\n");
