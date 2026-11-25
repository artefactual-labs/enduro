import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { basename, join } from "node:path";

const transferID = "71bd0ab3-e496-4ec0-9146-048abf471df4";
const siblingID = "71bd0ab3-e496-4ec0-9146-048abf471df5";
const siblingSIP = "71bd0ab3-e496-4ec0-9146-048abf471df6";
const headers = {
  "content-type": "application/json",
  "x-archivematica-id": process.env.PIPELINE_ID ?? "e1b9a096-c3ca-4ee2-9010-7b76ba37d318",
  "x-archivematica-version": "1.18.0",
};
const submissions = [];

createServer(async (req, res) => {
  const reply = (status, body) => {
    res.writeHead(status, headers);
    res.end(JSON.stringify(body));
  };
  try {
    if (req.method === "GET" && req.url === "/api/processing-configuration/") {
      return reply(200, { processing_configurations: [] });
    }
    if (req.method === "GET" && req.url === "/api/v2beta/package/") {
      return reply(501, {});
    }
    if (req.method === "GET" && req.url === "/submissions") {
      return reply(200, submissions);
    }
    if (req.method === "POST" && req.url === "/api/v2beta/package/") {
      const chunks = [];
      for await (const chunk of req) chunks.push(chunk);
      const { name, type, path } = JSON.parse(Buffer.concat(chunks).toString());
      const relative = Buffer.from(path, "base64").toString();
      const submission = { name, type };
      submissions.push(submission);
      if (!relative.startsWith("enduro") || basename(relative) !== relative) {
        return reply(422, { error: "expected a staged transfer directory" });
      }
      const dir = join("/runtime/transfers", relative);
      if (name === "converted") {
        submission.checksum = await readFile(join(dir, "metadata/checksum.sha256"), "utf8");
        submission.payload = await readFile(join(dir, "objects/payload.txt"), "utf8");
        submission.declaration = await readFile(join(dir, "metadata/submissionDocumentation/bagit.txt"), "utf8");
        return reply(201, { id: "02603572-36a1-43a4-a785-3812fd4fdef0" });
      }
      submission.declaration = await readFile(join(dir, "bagit.txt"), "utf8");
      submission.manifest = await readFile(join(dir, "manifest-sha256.txt"), "utf8");
      if (name === "sibling") {
        await readFile(join(dir, "data", "sibling.txt"));
        return reply(201, { id: siblingID });
      }
      submission.hiddenFile = await readFile(join(dir, "data", ".DS_Store"), "utf8");
      return reply(201, { id: transferID });
    }
    if (req.method === "DELETE" && [`/api/transfer/${siblingID}/delete/`, `/api/ingest/${siblingSIP}/delete/`].includes(req.url)) {
      return reply(200, { removed: true });
    }
    if (req.url === `/api/transfer/status/${siblingID}`) {
      return reply(200, { status: "COMPLETE", sip_uuid: siblingSIP });
    }
    if (req.url === `/api/ingest/status/${siblingSIP}`) {
      return reply(200, { status: "COMPLETE" });
    }
    if (req.method === "GET" && [transferID, "02603572-36a1-43a4-a785-3812fd4fdef0"].some((id) => req.url === `/api/transfer/status/${id}`)) {
      return reply(200, { status: "PROCESSING" });
    }
    return reply(500, { error: `unexpected request ${req.method} ${req.url}` });
  } catch (error) {
    return reply(422, { error: String(error) });
  }
}).listen(8080, "0.0.0.0");
