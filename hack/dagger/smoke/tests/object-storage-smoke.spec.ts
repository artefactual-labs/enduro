import { expect, test, type APIRequestContext } from "@playwright/test";
import { execFile as execFileCallback } from "node:child_process";
import { promises as fs } from "node:fs";
import path from "node:path";
import { promisify } from "node:util";

const execFile = promisify(execFileCallback);

const enduroURL = process.env.ENDURO_URL ?? "http://enduro:9000";
const temporalAddress = process.env.TEMPORAL_ADDRESS ?? "temporal:7233";
const artifactsDir = process.env.ARTIFACTS_DIR ?? "/artifacts";
const cacheBuster = process.env.E2E_CACHE_BUSTER ?? `${Date.now()}`;
const scenario = process.env.OBJECT_STORAGE_SCENARIO ?? "object-storage";
const transferName =
  process.env.OBJECT_STORAGE_TRANSFER_NAME ?? `${scenario}-${cacheBuster}.zip`;
const s3Endpoint = process.env.S3_ENDPOINT ?? "";
const s3Bucket = process.env.S3_BUCKET ?? "sips";
const s3AccessKey = process.env.S3_ACCESS_KEY_ID ?? "minio";
const s3SecretKey = process.env.S3_SECRET_ACCESS_KEY ?? "minio123";
const s3Region = process.env.S3_REGION ?? "us-west-1";
const redisList = process.env.REDIS_LIST ?? "";
const uploadDelayMS = Number(process.env.OBJECT_STORAGE_UPLOAD_DELAY_MS ?? "45000");

type Collection = {
  id: number;
  status: string;
  workflow_id?: string;
  run_id?: string;
};

test.beforeAll(async () => {
  await fs.mkdir(artifactsDir, { recursive: true });
});

test(`${scenario} S3 watcher transfer completes and produces an AIP`, async ({
  request,
}) => {
  const stageDir = `/tmp/${scenario}-${cacheBuster}`;
  const zipPath = `/tmp/${transferName}`;

  await test.step("wait for Enduro API", async () => {
    await waitForEnduro(request);
  });

  await test.step("prepare a zipped transfer", async () => {
    await fs.rm(stageDir, { recursive: true, force: true });
    await fs.mkdir(path.join(stageDir, "objects"), { recursive: true });
    await fs.writeFile(
      path.join(stageDir, "objects", "hello.txt"),
      `hello from ${scenario} ${cacheBuster}\n`,
    );
    await exec("zip", ["-qr", zipPath, "objects"], { cwd: stageDir });
  });

  await test.step("upload the transfer through S3", async () => {
    // Match the filesystem watcher smoke: ambox can accept connections before
    // Archivematica workers have fully settled.
    if (uploadDelayMS > 0) {
      await sleep(uploadDelayMS);
    }
    await retry(async () => {
      await exec("s3put", [
        "-endpoint",
        s3Endpoint,
        "-region",
        s3Region,
        "-bucket",
        s3Bucket,
        "-key",
        transferName,
        "-access-key",
        s3AccessKey,
        "-secret-key",
        s3SecretKey,
        "-file",
        zipPath,
      ]);
    });
    console.log(`uploaded ${transferName} to ${s3Endpoint}/${s3Bucket}`);
    await logRedisListLength("after upload");
  });

  const collection = await waitForCollectionDone(request, transferName);
  await downloadAndInspectAIP(request, collection, transferName);
  await inspectTemporalHistory(collection, transferName);
});

async function waitForCollectionDone(
  request: APIRequestContext,
  name: string,
): Promise<Collection> {
  let lastStatus = "";
  let lastCollection: Collection | undefined;
  let sawCollection = false;
  let lastError = "";

  for (let attempt = 0; attempt < 720; attempt++) {
    try {
      const response = await request.get(
        `${enduroURL}/collection?name=${encodeURIComponent(name)}`,
      );
      if (response.ok()) {
        lastError = "";
        const body = await response.json();
        const collection = body.items?.[0];
        if (collection?.id) {
          sawCollection = true;
          lastCollection = collection;
          lastStatus = collection.status;
          console.log(`collection=${lastCollection.id} status=${lastStatus}`);
        }
      } else {
        lastError = `status=${response.status()}`;
      }
    } catch (err) {
      lastError = String(err);
    }

    if (!sawCollection && attempt > 0 && attempt % 6 === 0) {
      console.log(`collection missing for ${name} after ${attempt * 10}s`);
      await logRedisListLength("while waiting for collection");
    }

    if (!sawCollection && attempt === 30) {
      await writeTextArtifact(
        "collection-missing-report.txt",
        [
          `transfer_name=${name}`,
          `scenario=${scenario}`,
          `s3_endpoint=${s3Endpoint}`,
          `s3_bucket=${s3Bucket}`,
          `redis_list=${redisList}`,
          "collection_seen=false",
        ].join("\n") + "\n",
      );
      throw new Error(`timed out waiting for ${name} collection to be created`);
    }

    if (lastError && attempt > 0 && attempt % 6 === 0) {
      console.log(`collection poll error for ${name}: ${lastError}`);
    }

    if (lastStatus === "done") {
      expect(lastCollection).toBeTruthy();
      return lastCollection!;
    }

    if (lastStatus === "error") {
      await saveCollection(request, lastCollection?.id ?? 0, "collection-error.json");
      throw new Error(`${name} entered error state`);
    }

    await sleep(10_000);
  }

  throw new Error(
    `timed out waiting for ${name}; last status=${lastStatus || "missing"} last error=${lastError || "none"}`,
  );
}

async function downloadAndInspectAIP(
  request: APIRequestContext,
  collection: Collection,
  name: string,
) {
  const response = await request.get(`${enduroURL}/collection/${collection.id}/download`);
  expect(response.ok()).toBeTruthy();

  await fs.writeFile(path.join(artifactsDir, "aip.7z"), await response.body());
  await fs.writeFile(
    path.join(artifactsDir, "aip.headers"),
    JSON.stringify(response.headers(), null, 2),
  );

  const aipPath = path.join(artifactsDir, "aip.7z");
  const stat = await fs.stat(aipPath);
  expect(stat.size).toBeGreaterThan(0);

  const listing = await exec("7z", ["l", aipPath]);
  await writeTextArtifact("aip-listing.txt", listing);
  expect(listing).toContain(name);

  const extractDir = "/tmp/object-storage-aip-extract";
  await fs.rm(extractDir, { recursive: true, force: true });
  await fs.mkdir(extractDir, { recursive: true });
  const extractLog = await exec("7z", ["x", "-y", `-o${extractDir}`, aipPath]);
  await writeTextArtifact("aip-extract.log", extractLog);

  const metsPath = await findFirstMETS(extractDir);
  expect(metsPath).toBeTruthy();

  const mets = await fs.readFile(metsPath, "utf8");
  expect(mets).toContain("Archivematica");
  expect(mets).toContain("preservation system");

  await saveCollection(request, collection.id, "collection.json");
}

async function inspectTemporalHistory(collection: Collection, name: string) {
  const temporalArtifactsDir = "/tmp/temporal-artifacts";
  await fs.mkdir(temporalArtifactsDir, { recursive: true });

  expect(collection.workflow_id).toBeTruthy();
  const workflowID = collection.workflow_id!;

  const workflows = await runTemporal([
    "workflow",
    "list",
    "--namespace",
    "default",
    "--address",
    temporalAddress,
    "--limit",
    "20",
    "--output",
    "json",
  ]);
  await fs.writeFile(
    path.join(temporalArtifactsDir, "temporal-workflows.json"),
    workflows,
  );

  const history = await runTemporal([
    "workflow",
    "show",
    "--namespace",
    "default",
    "--address",
    temporalAddress,
    "--workflow-id",
    workflowID,
    ...(collection.run_id ? ["--run-id", collection.run_id] : []),
    "--output",
    "json",
  ]);
  await fs.writeFile(path.join(temporalArtifactsDir, "temporal-history.json"), history);

  expect(history).toContain('"name": "publish-transfer-activity"');
  expect(history).toContain('"name": "clean-up-published-transfer-activity"');
  expect(history).toContain('"name": "transfer-activity"');
  expect(history).toMatch(
    /WorkflowExecutionCompleted|EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED/,
  );

  await writeTextArtifact(
    "temporal-report.txt",
    [
      `transfer_name=${name}`,
      `workflow_id=${workflowID}`,
      `run_id=${collection.run_id ?? ""}`,
      "has_publish_transfer_activity=true",
      "has_cleanup_published_transfer_activity=true",
      "has_transfer_activity=true",
      "workflow_completed=true",
    ].join("\n") + "\n",
  );
}

async function saveCollection(
  request: APIRequestContext,
  collectionID: number,
  artifactName: string,
) {
  if (!collectionID) {
    return;
  }
  const response = await request.get(`${enduroURL}/collection/${collectionID}`);
  if (response.ok()) {
    await fs.writeFile(
      path.join(artifactsDir, artifactName),
      JSON.stringify(await response.json(), null, 2),
    );
  }
}

async function waitForEnduro(request: APIRequestContext) {
  let lastError = "";
  for (let attempt = 0; attempt < 90; attempt++) {
    try {
      const response = await request.get(
        `${enduroURL}/collection?name=__enduro_ready__`,
      );
      if (response.ok()) {
        console.log("Enduro API is ready");
        return;
      }
      lastError = `status=${response.status()}`;
    } catch (err) {
      lastError = String(err);
    }
    await sleep(2_000);
  }

  throw new Error(`timed out waiting for Enduro API: ${lastError}`);
}

async function logRedisListLength(label: string) {
  if (!redisList) {
    return;
  }

  try {
    const output = await exec("redis-cli", ["-h", "redis", "LLEN", redisList]);
    console.log(`redis ${redisList} length ${label}: ${output.trim()}`);
  } catch (err) {
    console.log(`redis ${redisList} length ${label}: ${String(err)}`);
  }
}

async function findFirstMETS(root: string): Promise<string> {
  const entries = await fs.readdir(root, { withFileTypes: true });
  for (const entry of entries) {
    const entryPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      const found = await findFirstMETS(entryPath);
      if (found) {
        return found;
      }
    } else if (entry.isFile() && /^METS.*\.xml$/.test(entry.name)) {
      return entryPath;
    }
  }
  return "";
}

async function runTemporal(args: string[]): Promise<string> {
  return exec("temporal", args);
}

async function exec(command: string, args: string[], options = {}): Promise<string> {
  const result = await execFile(command, args, {
    maxBuffer: 10 * 1024 * 1024,
    ...options,
  });
  return `${result.stdout}${result.stderr}`;
}

async function writeTextArtifact(name: string, content: string) {
  await fs.mkdir(artifactsDir, { recursive: true });
  await fs.writeFile(path.join(artifactsDir, name), content);
}

async function sleep(ms: number) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function retry(fn: () => Promise<void>) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 30; attempt++) {
    try {
      await fn();
      return;
    } catch (err) {
      lastError = err;
      await sleep(2_000);
    }
  }
  throw lastError;
}
