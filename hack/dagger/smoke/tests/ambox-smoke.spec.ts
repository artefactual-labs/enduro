import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { execFile as execFileCallback } from "node:child_process";
import { createHash } from "node:crypto";
import { promises as fs } from "node:fs";
import path from "node:path";
import { promisify } from "node:util";

const execFile = promisify(execFileCallback);

const enduroURL = process.env.ENDURO_URL ?? "http://enduro:9000";
const lgtmURL = process.env.LGTM_URL ?? "http://lgtm:3000";
const tempoURL = `${lgtmURL}/api/datasources/proxy/uid/tempo`;
const temporalAddress = process.env.TEMPORAL_ADDRESS ?? "temporal:7233";
const watchedDir = process.env.WATCHED_DIR ?? "/runtime/watched";
const batchDir = process.env.BATCH_DIR ?? "/runtime/batch";
const receiptsDir = process.env.RECEIPTS_DIR ?? "/runtime/receipts/out";
const artifactsDir = process.env.ARTIFACTS_DIR ?? "/artifacts";
const cacheBuster = process.env.E2E_CACHE_BUSTER ?? `${Date.now()}`;
const expectedProcessingWorkflows = Number(
  process.env.EXPECTED_PROCESSING_WORKFLOWS ?? "1",
);
const verifyBatchWorkflow = process.env.VERIFY_BATCH_WORKFLOW === "true";
const batchProcessingConfig = "enduro-batch";
const batchOriginal = {
  name: "hello.txt",
  bytes: Buffer.from("Hello from the unzipped BagIt transfer.\n", "utf8"),
};
const batchIdentifier = `2.16.578.1.39.100.11.${cacheBuster}`;
const batchTransferName = `dpj-sip-${batchIdentifier}-20260721`;
const batchParent = "2.16.578.1.39.100.11.234";
const journal =
  "<avlxml><avleveringsbeskrivelse>Synthetic journal</avleveringsbeskrivelse></avlxml>";

type Collection = {
  id: number;
  status: string;
};

type TraceAttribute = {
  key: string;
  value: { stringValue?: string };
};

type TempoTrace = {
  batches?: {
    resource?: { attributes?: TraceAttribute[] };
    scopeSpans?: {
      spans?: { name: string; attributes?: TraceAttribute[] }[];
    }[];
  }[];
};

test.describe.configure({ mode: "serial" });

test.beforeAll(async () => {
  await fs.mkdir(artifactsDir, { recursive: true });
});

test("filesystem transfer resumes after a prod receipt decision", async ({
  request,
}) => {
  await test.step("wait for LGTM's Tempo datasource", async () => {
    await expect(async () => {
      const response = await request.get(`${tempoURL}/ready`, { timeout: 5_000 });
      expect(response.ok(), await response.text()).toBeTruthy();
    }).toPass({ timeout: 120_000, intervals: [1_000, 2_000, 5_000] });
  });

  const runID = `e2e-${Math.floor(Date.now() / 1000)}`;
  const receiptIdentifier = `2.16.578.1.39.100.11.${Date.now()}`;
  const transferName = `avl-other-sip-${receiptIdentifier}-20260721.xml.zip`;

  await test.step("submit a zipped transfer through the watched directory", async () => {
    await fs.rm(receiptsDir, { recursive: true, force: true });

    const stageDir = `/tmp/${runID}`;
    await fs.rm(stageDir, { recursive: true, force: true });
    await fs.mkdir(path.join(stageDir, "objects"), { recursive: true });
    await fs.writeFile(
      path.join(stageDir, "objects", "hello.txt"),
      `hello from ${runID}\n`,
    );
    await fs.mkdir(path.join(stageDir, "AVLXML", "objekter"), { recursive: true });
    await fs.writeFile(
      path.join(stageDir, "AVLXML", "objekter", "avlxml.xml"),
      `<avlxml><avleveringsidentifikator>${receiptIdentifier}</avleveringsidentifikator><pasientjournal>synthetic-private-marker</pasientjournal></avlxml>`,
    );
    await execFile("zip", ["-qr", `/tmp/${transferName}`, "objects", "AVLXML"], {
      cwd: stageDir,
    });

    // Give Archivematica's worker services time to settle before the first
    // transfer arrives. This keeps the smoke focused on Enduro behavior.
    await sleep(45_000);
    await fs.copyFile(`/tmp/${transferName}`, path.join(watchedDir, transferName));
    await fs.unlink(`/tmp/${transferName}`);
    console.log(`submitted ${transferName}`);
  });

  const pendingCollection = await waitForCollectionStatus(
    request,
    transferName,
    "pending",
    {
      artifactPrefix: "",
      logPrefix: "collection",
    },
  );

  await test.step("repair prod delivery and submit a retry decision", async () => {
    await fs.mkdir(receiptsDir, { recursive: true });
    await fs.chown(receiptsDir, 65532, 65532);

    const response = await request.post(
      `${enduroURL}/collection/${pendingCollection.id}/decision`,
      { data: { option: "RETRY_ONCE" } },
    );
    const body = await response.text();
    await writeTextArtifact(
      "decision-response.txt",
      [
        `collection_id=${pendingCollection.id}`,
        "option=RETRY_ONCE",
        `status=${response.status()}`,
        `status_text=${response.statusText()}`,
        "body=",
        body,
      ].join("\n") + "\n",
    );
    expect(response.ok(), body).toBeTruthy();
  });

  const collection = await waitForCollectionDone(request, transferName, {
    artifactPrefix: "",
    logPrefix: "collection",
  });

  const receiptFiles = await fs.readdir(receiptsDir);
  const matches = receiptFiles.filter(
    (name) =>
      name.startsWith(`Receipt_${receiptIdentifier}_`) &&
      name.endsWith(".mft"),
  );
  expect(matches).toHaveLength(1);
  const [receiptName] = matches;

  const receiptPath = path.join(receiptsDir, receiptName);
  const receipt = JSON.parse(await fs.readFile(receiptPath, "utf8"));
  expect(receipt).toMatchObject({
    identifier: receiptIdentifier,
    type: "avlxml",
    accepted: true,
    message: "Package was processed by Archivematica pipeline ambox",
  });
  await fs.copyFile(
    receiptPath,
    path.join(artifactsDir, "production-receipt.mft"),
  );
  await writeJSONArtifact("production-receipt.json", receipt);

  const detail = await downloadAndInspectAIP(request, {
    collection,
    transferName,
    artifactPrefix: "",
    extractDir: "/tmp/aip-extract",
  });
  expect(detail.processing_config).toBe("automated");
  const deliveries = await hariDeliveries(request, detail.aip_id);
  expect(deliveries).toHaveLength(1);
  const [delivery] = deliveries;
  expect(delivery).toMatchObject({
    status: 200,
    contentType: "application/json",
    receipt: { type: "avlxml", message: "AVLXML was processed by Archivematica pipeline ambox" },
  });
  expect(delivery.receipt.parent).toBeUndefined();
  expectSameStorageTime(delivery.receipt.timestamp, receipt.timestamp);
  const xml = Buffer.from(delivery.receipt.xml, "base64").toString();
  expect(xml).toContain(`<avleveringsidentifikator>${receiptIdentifier}</avleveringsidentifikator>`);
  expect(xml).not.toMatch(/pasientjournal|synthetic-private-marker/);

  await writeTextArtifact(
    "report.txt",
    [
      `run_id=${runID}`,
      `transfer_name=${transferName}`,
      `collection_id=${collection.id}`,
      `status=${collection.status}`,
      `receipt=${receiptName}`,
    ].join("\n") + "\n",
  );
});

test("LGTM captures the completed transfer's workflow trace", async ({ request }) => {
  test.setTimeout(90_000);
  const collection = JSON.parse(
    await fs.readFile(path.join(artifactsDir, "collection.json"), "utf8"),
  );
  const workflowID: string = collection.workflow_id;
  expect(workflowID).toMatch(/^processing-workflow-/);

  // Both Enduro and the collector batch exports. Wait for this transfer's
  // completed workflow and activity spans, not unrelated health-check traffic.
  await expect(async () => {
    const searchResponse = await request.get(`${tempoURL}/api/search`, {
      params: {
        q: `{ resource.service.name = "enduro" && span.temporalWorkflowID = ${JSON.stringify(workflowID)} && name = "RunWorkflow:processing-workflow" }`,
        limit: 1,
      },
      timeout: 10_000,
    });
    expect(searchResponse.ok(), await searchResponse.text()).toBeTruthy();
    const search = await searchResponse.json();
    await writeJSONArtifact("tempo-search.json", search);
    const traceID: string = search.traces?.[0]?.traceID;
    expect(traceID, `No processing trace found for ${workflowID}`).toBeTruthy();

    const traceResponse = await request.get(`${tempoURL}/api/traces/${traceID}`, {
      timeout: 10_000,
    });
    expect(traceResponse.ok(), await traceResponse.text()).toBeTruthy();
    const trace: TempoTrace = await traceResponse.json();
    await writeJSONArtifact("tempo-trace.json", trace);
    const spans = (trace.batches ?? [])
      .filter((batch) =>
        batch.resource?.attributes?.some(
          (attribute) =>
            attribute.key === "service.name" &&
            attribute.value.stringValue === "enduro",
        ),
      )
      .flatMap((batch) => batch.scopeSpans ?? [])
      .flatMap((scope) => scope.spans ?? [])
      .filter((span) =>
        span.attributes?.some(
          (attribute) =>
            attribute.key === "temporalWorkflowID" &&
            attribute.value.stringValue === workflowID,
        ),
      );
    const spanNames = spans.map((span) => span.name);
    expect(spanNames).toEqual(
      expect.arrayContaining([
        "RunWorkflow:processing-workflow",
        "RunActivity:publish-transfer-activity",
        "RunActivity:transfer-activity",
      ]),
    );
    await writeTextArtifact(
      "telemetry-report.txt",
      `workflow_id=${workflowID}\ntrace_id=${traceID}\nspans=${spanNames.join(",")}\n`,
    );
  }).toPass({ timeout: 60_000, intervals: [1_000, 2_000, 5_000] });
});

test("Nuxt batch import form submits an unzipped bag", async ({ page }) => {
  test.setTimeout(120_000);

  await test.step("prepare a bag with a manifest-listed hidden file", async () => {
    const transferDir = path.join(batchDir, batchTransferName);
    await fs.rm(transferDir, { recursive: true, force: true });
    await fs.mkdir(path.join(transferDir, "data"), { recursive: true });
    await fs.writeFile(
      path.join(transferDir, "bagit.txt"),
      "BagIt-Version: 1.0\nTag-File-Character-Encoding: UTF-8\n",
    );
    const identifiers = [{
      file: "objects/DPJ/journal/avlxml.xml",
      identifiers: [{ identifierType: "avleveringsidentifikator", identifier: batchParent }],
    }];
    const payload = [
      batchOriginal,
      { name: ".DS_Store", bytes: Buffer.alloc(0) },
      { name: "DPJ/journal/avlxml.xml", bytes: Buffer.from(journal) },
      { name: "metadata/identifiers.json", bytes: Buffer.from(JSON.stringify(identifiers)) },
    ];
    const manifest = [];
    for (const { name, bytes } of payload) {
      const target = path.join(transferDir, "data", name);
      await fs.mkdir(path.dirname(target), { recursive: true });
      await fs.writeFile(target, bytes);
      const digest = createHash("sha256").update(bytes).digest("hex");
      manifest.push(`${digest}  data/${name}`);
    }
    await fs.writeFile(
      path.join(transferDir, "manifest-sha256.txt"),
      manifest.join("\n") + "\n",
    );
    await execFile("chown", ["-R", "65532:65532", transferDir]);
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
    await page
      .getByRole("combobox", { name: "Processing configuration" })
      .click();
    await page
      .getByRole("option", { name: batchProcessingConfig, exact: true })
      .click();
    await page.getByRole("combobox", { name: "Transfer type" }).click();
    await page.getByRole("option", { name: "Unzipped bag", exact: true }).click();
    await page.getByRole("checkbox", { name: "Exclude hidden files" }).uncheck();

    await page.getByRole("button", { name: "Submit" }).click();
    const confirmation = page
      .getByRole("dialog")
      .filter({ hasText: "Submit batch import?" });
    await expect(confirmation).toBeVisible();
    await expect(
      confirmation.getByText(batchProcessingConfig, { exact: true }),
    ).toBeVisible();
    const [submission] = await Promise.all([
      page.waitForRequest(
        (request) =>
          request.method() === "POST" &&
          /^\/batch\/?$/.test(new URL(request.url()).pathname),
      ),
      confirmation.getByRole("button", { name: "Submit" }).click(),
    ]);
    expect(submission.postDataJSON()).toMatchObject({
      processing_config: batchProcessingConfig,
      transfer_type: "unzipped bag",
      exclude_hidden_files: false,
    });

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

test("batch bag preserves its original and records its processing configuration", async ({
  page,
}) => {
  const request = page.request;
  const pending = await waitForCollectionStatus(request, batchTransferName, "pending", {
    artifactPrefix: "batch-", logPrefix: "batch HARI retry",
  });
  const pendingResponse = await request.get(`${enduroURL}/collection/${pending.id}`);
  expect(pendingResponse.ok()).toBeTruthy();
  const { aip_id } = await pendingResponse.json();
  const attempts = await hariDeliveries(request, aip_id);
  expect(attempts.map((entry) => entry.status)).toEqual([503]);
  expect(attempts[0].receipt).toMatchObject({
    type: "dpj",
    parent: batchParent,
    xml: Buffer.from(journal).toString("base64"),
    message: "AVLXML was processed by Archivematica pipeline ambox",
  });
  const prodFiles = async () => (await fs.readdir(receiptsDir))
    .filter((name) => name.startsWith(`Receipt_${batchIdentifier}_`));
  expect(await prodFiles()).toEqual([]);
  const retry = await request.post(`${enduroURL}/collection/${pending.id}/decision`, {
    data: { option: "RETRY_ONCE" },
  });
  expect(retry.ok(), await retry.text()).toBeTruthy();
  const collection = await waitForCollectionDone(request, batchTransferName, {
    artifactPrefix: "batch-",
    logPrefix: "batch collection",
  });

  const detail = await downloadAndInspectAIP(request, {
    collection,
    transferName: batchTransferName,
    artifactPrefix: "batch-",
    extractDir: "/tmp/batch-aip-extract",
    expectedOriginal: batchOriginal,
  });
  expect(detail.processing_config).toBe(batchProcessingConfig);
  expect(detail.aip_id).toBe(aip_id);
  expect(await hariDeliveries(request, aip_id)).toEqual([
    attempts[0], { ...attempts[0], status: 200 },
  ]);
  const files = await prodFiles();
  expect(files).toHaveLength(1);
  expect(files[0]).toMatch(/\.mft$/);
  const prod = JSON.parse(await fs.readFile(path.join(receiptsDir, files[0]), "utf8"));
  expect(prod).toMatchObject({
    identifier: batchIdentifier,
    type: "dpj",
    parent: batchParent,
    accepted: true,
    message: "Package was processed by Archivematica pipeline ambox",
  });
  expectSameStorageTime(attempts[0].receipt.timestamp, prod.timestamp);

  await page.goto(`${enduroURL}/collections/${collection.id}`, {
    waitUntil: "domcontentloaded",
  });
  const processingConfig = page
    .locator("dl > div")
    .filter({
      has: page.getByText("Processing configuration", { exact: true }),
    })
    .locator("dd");
  await expect(processingConfig).toHaveText(batchProcessingConfig);

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

  let hasDecisionUpdate = false;
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

    const hasAcceptedUpdate =
      /WorkflowExecutionUpdateAccepted|EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED/.test(
        history,
      );
    const hasCompletedUpdate =
      /WorkflowExecutionUpdateCompleted|EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED/.test(
        history,
      );
    hasDecisionUpdate ||= hasAcceptedUpdate && hasCompletedUpdate;
  }

  expect(hasDecisionUpdate).toBeTruthy();

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
    `has_decision_update=${hasDecisionUpdate}`,
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
  return waitForCollectionStatus(request, transferName, "done", opts);
}

async function waitForCollectionStatus(
  request: APIRequestContext,
  transferName: string,
  expectedStatus: string,
  opts: { artifactPrefix: string; logPrefix: string },
): Promise<Collection> {
  let lastStatus = "";
  let lastCollectionID = 0;
  let sawCollection = false;
  let lastError = "";

  for (let attempt = 0; attempt < 720; attempt++) {
    try {
      const response = await request.get(
        `${enduroURL}/collection?name=${encodeURIComponent(transferName)}`,
      );
      if (response.ok()) {
        lastError = "";
        const body = await response.json();
        const collection = body.items?.[0];
        if (collection?.id) {
          sawCollection = true;
          lastCollectionID = collection.id;
          lastStatus = collection.status;
          console.log(`${opts.logPrefix}=${lastCollectionID} status=${lastStatus}`);
        }
      } else {
        lastError = `status=${response.status()}`;
      }
    } catch (err) {
      lastError = String(err);
    }

    if (!sawCollection && attempt > 0 && attempt % 6 === 0) {
      console.log(
        `${opts.logPrefix} missing for ${transferName} after ${attempt * 10}s`,
      );
    }

    if (!sawCollection && attempt === 30) {
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
      await writeTextArtifact(
        `${opts.artifactPrefix}collection-missing-report.txt`,
        [
          `transfer_name=${transferName}`,
          `log_prefix=${opts.logPrefix}`,
          `last_error=${lastError || "none"}`,
          "collection_seen=false",
          "",
          "temporal_workflows=",
          workflows,
        ].join("\n") + "\n",
      );
      throw new Error(
        `timed out waiting for ${transferName} collection to be created`,
      );
    }

    if (lastError && attempt > 0 && attempt % 6 === 0) {
      console.log(`${opts.logPrefix} poll error for ${transferName}: ${lastError}`);
    }

    if (lastStatus === expectedStatus) {
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
    `timed out waiting for ${transferName} to become ${expectedStatus}; ` +
      `last status=${lastStatus || "missing"}`,
  );
}

async function downloadAndInspectAIP(
  request: APIRequestContext,
  opts: {
    collection: Collection;
    transferName: string;
    artifactPrefix: string;
    extractDir: string;
    expectedOriginal?: { name: string; bytes: Buffer };
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
  if (!response.ok()) {
    await saveCollection(request, opts.collection.id, collectionName);
    await writeTextArtifact(
      `${opts.artifactPrefix}download-error.txt`,
      [
        `transfer_name=${opts.transferName}`,
        `collection_id=${opts.collection.id}`,
        `download_status=${response.status()}`,
        `download_status_text=${response.statusText()}`,
        "download_headers=",
        JSON.stringify(response.headers(), null, 2),
        "",
        "download_body=",
        await response.text(),
      ].join("\n") + "\n",
    );
  }
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
  if (opts.expectedOriginal) {
    const originalPath = path.join(
      path.dirname(metsPath),
      "objects",
      opts.expectedOriginal.name,
    );
    expect(await fs.readFile(originalPath)).toEqual(opts.expectedOriginal.bytes);
  }

  await writeTextArtifact(
    inspectionName,
    [
      `mets_path=${path.relative(opts.extractDir, metsPath)}`,
      `archivematica_mentions=${countOccurrences(mets, "Archivematica")}`,
      "has_preservation_system_agent=true",
      "has_archivematica_agent=true",
    ].join("\n") + "\n",
  );

  const detailResponse = await request.get(
    `${enduroURL}/collection/${opts.collection.id}`,
  );
  expect(detailResponse.ok(), await detailResponse.text()).toBeTruthy();
  const detail = await detailResponse.json();
  expect(detail).toMatchObject({
    id: opts.collection.id,
    status: "done",
    workflow_id: expect.stringMatching(/^processing-workflow-/),
  });
  await writeJSONArtifact(collectionName, detail);
  return detail;
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

async function hariDeliveries(request: APIRequestContext, aipID: string) {
  expect(aipID).toBeTruthy();
  const response = await request.get("http://hari:8080/deliveries");
  expect(response.ok()).toBeTruthy();
  type Delivery = {
    status: number;
    contentType: string;
    receipt: { aip_id: string; xml: string; timestamp: string; parent?: string };
  };
  const deliveries: Delivery[] = await response.json();
  const attempts = deliveries.filter((entry) => entry.receipt.aip_id === aipID);
  await writeJSONArtifact(`hari-${aipID}.json`, attempts);
  return attempts;
}

function expectSameStorageTime(hari: string, prod: string) {
  // HARI encodes the storage timestamp to whole seconds.
  expect(Date.parse(hari)).toBe(Math.floor(Date.parse(prod) / 1000) * 1000);
}
