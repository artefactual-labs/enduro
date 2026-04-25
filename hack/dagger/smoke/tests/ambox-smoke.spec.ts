import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { execFile as execFileCallback } from "node:child_process";
import { promises as fs } from "node:fs";
import path from "node:path";
import { promisify } from "node:util";

const execFile = promisify(execFileCallback);

const enduroURL = process.env.ENDURO_URL ?? "http://enduro:9000";
const temporalAddress = process.env.TEMPORAL_ADDRESS ?? "temporal:7233";
const watchedDir = process.env.WATCHED_DIR ?? "/runtime/watched";
const batchDir = process.env.BATCH_DIR ?? "/runtime/batch";
const artifactsDir = process.env.ARTIFACTS_DIR ?? "/artifacts";
const cacheBuster = process.env.E2E_CACHE_BUSTER ?? `${Date.now()}`;
const expectedProcessingWorkflows = Number(
  process.env.EXPECTED_PROCESSING_WORKFLOWS ?? "1",
);
const verifyBatchWorkflow = process.env.VERIFY_BATCH_WORKFLOW === "true";
const batchTransferName =
  process.env.BATCH_TRANSFER_NAME ?? `batch-${cacheBuster}`;

type Collection = {
  id: number;
  status: string;
};

test.describe.configure({ mode: "serial" });

test.beforeAll(async () => {
  await fs.mkdir(artifactsDir, { recursive: true });
});

test("filesystem watcher transfer completes and produces an AIP", async ({
  request,
}) => {
  const runID = `e2e-${Math.floor(Date.now() / 1000)}`;
  const transferName = `${runID}.zip`;

  await test.step("submit a zipped transfer through the watched directory", async () => {
    const stageDir = `/tmp/${runID}`;
    await fs.rm(stageDir, { recursive: true, force: true });
    await fs.mkdir(path.join(stageDir, "objects"), { recursive: true });
    await fs.writeFile(
      path.join(stageDir, "objects", "hello.txt"),
      `hello from ${runID}\n`,
    );
    await execFile("zip", ["-qr", `/tmp/${transferName}`, "objects"], {
      cwd: stageDir,
    });

    // Give Archivematica's worker services time to settle before the first
    // transfer arrives. This keeps the smoke focused on Enduro behavior.
    await sleep(45_000);
    await fs.copyFile(`/tmp/${transferName}`, path.join(watchedDir, transferName));
    await fs.unlink(`/tmp/${transferName}`);
    console.log(`submitted ${transferName}`);
  });

  const collection = await waitForCollectionDone(request, transferName, {
    artifactPrefix: "",
    logPrefix: "collection",
  });

  await downloadAndInspectAIP(request, {
    collection,
    transferName,
    artifactPrefix: "",
    extractDir: "/tmp/aip-extract",
  });

  await writeTextArtifact(
    "report.txt",
    [
      `run_id=${runID}`,
      `transfer_name=${transferName}`,
      `collection_id=${collection.id}`,
      `status=${collection.status}`,
    ].join("\n") + "\n",
  );
});

test("Nuxt batch import form submits a directory transfer", async ({ page }) => {
  await test.step("prepare a batch transfer visible to Enduro", async () => {
    const transferDir = path.join(batchDir, batchTransferName);
    await fs.rm(transferDir, { recursive: true, force: true });
    await fs.mkdir(path.join(transferDir, "objects"), { recursive: true });
    await fs.writeFile(
      path.join(transferDir, "objects", "hello.txt"),
      `hello from ${batchTransferName}\n`,
    );
  });

  await test.step("submit the batch import form in the dashboard", async () => {
    const pipelinesResponse = await page.request.get(
      `${enduroURL}/pipeline?status=true`,
    );
    expect(pipelinesResponse.ok()).toBeTruthy();

    const pipelines = await pipelinesResponse.json();
    await writeJSONArtifact("batch-pipelines.json", pipelines);

    const pipeline = pipelines.find((item: { name: string }) => {
      return item.name === "ambox";
    });
    expect(pipeline?.id).toBeTruthy();

    await page.goto(
      `${enduroURL}/collections/batch?pipeline=${encodeURIComponent(
        pipeline.id,
      )}`,
      { waitUntil: "domcontentloaded" },
    );
    await page.getByText("Start a new batch").waitFor({ timeout: 60_000 });
    await page.locator('input[placeholder="/path/to/transfers"]').fill(batchDir);

    await page.getByRole("button", { name: "Submit" }).click();
    await Promise.race([
      page.getByText("Batch submitted", { exact: true }).waitFor(),
      page.getByText("Batch running", { exact: true }).waitFor(),
    ]);

    await page.screenshot({
      path: path.join(artifactsDir, "batch-import-submitted.png"),
      fullPage: true,
    });

    await writeTextArtifact(
      "batch-submit-report.txt",
      [
        `transfer_name=${batchTransferName}`,
        `batch_path=${batchDir}`,
        `pipeline_id=${pipeline.id}`,
        `pipeline_name=${pipeline.name}`,
      ].join("\n") + "\n",
    );
  });
});

test("batch transfer completes and produces an AIP", async ({ request }) => {
  const collection = await waitForCollectionDone(request, batchTransferName, {
    artifactPrefix: "batch-",
    logPrefix: "batch collection",
  });

  await downloadAndInspectAIP(request, {
    collection,
    transferName: batchTransferName,
    artifactPrefix: "batch-",
    extractDir: "/tmp/batch-aip-extract",
  });

  await writeTextArtifact(
    "batch-report.txt",
    [
      `transfer_name=${batchTransferName}`,
      `collection_id=${collection.id}`,
      `status=${collection.status}`,
    ].join("\n") + "\n",
  );
});

test("Temporal histories include the expected workflow activities", async () => {
  const temporalArtifactsDir = "/tmp/temporal-artifacts";
  await fs.mkdir(path.join(temporalArtifactsDir, "processing"), {
    recursive: true,
  });

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

  const processingIDs = Array.from(
    new Set(workflows.match(/processing-workflow-[0-9a-f-]+/g) ?? []),
  ).sort();
  expect(processingIDs.length).toBeGreaterThanOrEqual(
    expectedProcessingWorkflows,
  );

  for (const workflowID of processingIDs) {
    const history = await runTemporal([
      "workflow",
      "show",
      "--namespace",
      "default",
      "--address",
      temporalAddress,
      "--workflow-id",
      workflowID,
      "--output",
      "json",
    ]);
    await fs.writeFile(
      path.join(temporalArtifactsDir, "processing", `${workflowID}.json`),
      history,
    );

    expect(history).toContain('"name": "publish-transfer-activity"');
    expect(history).toContain('"name": "clean-up-published-transfer-activity"');
    expect(history).toContain('"name": "transfer-activity"');
    expect(history).toMatch(
      /WorkflowExecutionCompleted|EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED/,
    );
  }

  const firstHistoryPath = path.join(
    temporalArtifactsDir,
    "processing",
    `${processingIDs[0]}.json`,
  );
  await fs.copyFile(
    firstHistoryPath,
    path.join(temporalArtifactsDir, "temporal-history.json"),
  );

  let hasBatchActivity = false;
  if (verifyBatchWorkflow) {
    const batchHistory = await runTemporal([
      "workflow",
      "show",
      "--namespace",
      "default",
      "--address",
      temporalAddress,
      "--workflow-id",
      "batch-workflow",
      "--output",
      "json",
    ]);
    await fs.writeFile(
      path.join(temporalArtifactsDir, "batch-history.json"),
      batchHistory,
    );
    expect(batchHistory).toContain('"name": "batch-activity"');
    expect(batchHistory).toMatch(
      /WorkflowExecutionCompleted|EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED/,
    );
    hasBatchActivity = true;
  }

  const report = [
    `workflow_id=${processingIDs[0]}`,
    `expected_processing_workflow_count=${expectedProcessingWorkflows}`,
    `processing_workflow_count=${processingIDs.length}`,
    `verified_processing_workflow_count=${processingIDs.length}`,
    "has_publish_transfer_activity=true",
    "has_cleanup_published_transfer_activity=true",
    "has_transfer_activity=true",
    `has_batch_activity=${hasBatchActivity}`,
    "workflow_completed=true",
  ].join("\n") + "\n";

  await fs.writeFile(path.join(temporalArtifactsDir, "temporal-report.txt"), report);
  console.log(report);
});

async function waitForCollectionDone(
  request: APIRequestContext,
  transferName: string,
  opts: { artifactPrefix: string; logPrefix: string },
): Promise<Collection> {
  let lastStatus = "";
  let lastCollectionID = 0;

  for (let attempt = 0; attempt < 720; attempt++) {
    const response = await request.get(
      `${enduroURL}/collection?name=${encodeURIComponent(transferName)}`,
    );
    if (response.ok()) {
      const body = await response.json();
      const collection = body.items?.[0];
      if (collection?.id) {
        lastCollectionID = collection.id;
        lastStatus = collection.status;
        console.log(`${opts.logPrefix}=${lastCollectionID} status=${lastStatus}`);
      }
    }

    if (lastStatus === "done") {
      return { id: lastCollectionID, status: lastStatus };
    }

    if (lastStatus === "error") {
      await saveCollection(
        request,
        lastCollectionID,
        `${opts.artifactPrefix}collection-error.json`,
      );
      throw new Error(`${opts.logPrefix} entered error state`);
    }

    await sleep(10_000);
  }

  throw new Error(
    `timed out waiting for ${transferName}; last status=${lastStatus || "missing"}`,
  );
}

async function downloadAndInspectAIP(
  request: APIRequestContext,
  opts: {
    collection: Collection;
    transferName: string;
    artifactPrefix: string;
    extractDir: string;
  },
) {
  const aipName = `${opts.artifactPrefix}aip.7z`;
  const headersName = `${opts.artifactPrefix}aip.headers`;
  const listingName = `${opts.artifactPrefix}aip-listing.txt`;
  const extractLogName = `${opts.artifactPrefix}aip-extract.log`;
  const inspectionName = `${opts.artifactPrefix}aip-inspection.txt`;
  const collectionName = `${opts.artifactPrefix}collection.json`;

  const response = await request.get(
    `${enduroURL}/collection/${opts.collection.id}/download`,
  );
  expect(response.ok()).toBeTruthy();

  await fs.writeFile(path.join(artifactsDir, aipName), await response.body());
  await fs.writeFile(
    path.join(artifactsDir, headersName),
    JSON.stringify(response.headers(), null, 2),
  );

  const aipPath = path.join(artifactsDir, aipName);
  const stat = await fs.stat(aipPath);
  expect(stat.size).toBeGreaterThan(0);

  const listing = await exec("7z", ["l", aipPath]);
  await writeTextArtifact(listingName, listing);
  expect(listing).toContain(opts.transferName);

  await fs.rm(opts.extractDir, { recursive: true, force: true });
  await fs.mkdir(opts.extractDir, { recursive: true });
  const extractLog = await exec("7z", ["x", "-y", `-o${opts.extractDir}`, aipPath]);
  await writeTextArtifact(extractLogName, extractLog);

  const metsPath = await findFirstMETS(opts.extractDir);
  expect(metsPath).toBeTruthy();

  const mets = await fs.readFile(metsPath, "utf8");
  expect(mets).toContain("Archivematica");
  expect(mets).toContain("preservation system");
  expect(mets).toMatch(
    /Archivematica-[0-9]+\.[0-9]+|<[^>]*agentName[^>]*>Archivematica</,
  );

  await writeTextArtifact(
    inspectionName,
    [
      `mets_path=${path.relative(opts.extractDir, metsPath)}`,
      `archivematica_mentions=${countOccurrences(mets, "Archivematica")}`,
      "has_preservation_system_agent=true",
      "has_archivematica_agent=true",
    ].join("\n") + "\n",
  );

  await saveCollection(request, opts.collection.id, collectionName);
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
    await writeJSONArtifact(artifactName, await response.json());
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

async function writeJSONArtifact(name: string, value: unknown) {
  await writeTextArtifact(name, `${JSON.stringify(value, null, 2)}\n`);
}

function countOccurrences(value: string, pattern: string): number {
  return value.split(pattern).length - 1;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
