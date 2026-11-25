import { createServer } from "node:http";

const deliveries = [];
createServer(async (req, res) => {
  res.setHeader("Content-Type", "application/json");
  if (req.method === "GET" && req.url === "/deliveries") {
    return res.end(JSON.stringify(deliveries));
  }
  if (req.method !== "POST" || req.url !== "/v1/hari/avlxml") {
    res.writeHead(404);
    return res.end("{}");
  }
  try {
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const receipt = JSON.parse(Buffer.concat(chunks).toString());
    // Make the first DPJ delivery fail so the test can exercise operator retry.
    const status = receipt.type === "dpj" &&
      !deliveries.some((entry) => entry.receipt.aip_id === receipt.aip_id)
      ? 503 : 200;
    deliveries.push({ receipt, status, contentType: req.headers["content-type"] });
    res.writeHead(status);
    res.end("{}");
  } catch {
    res.writeHead(400);
    res.end("{}");
  }
}).listen(8080, "0.0.0.0");
