import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  outputDir: `${process.env.ARTIFACTS_DIR ?? "/artifacts"}/playwright-results`,
  timeout: 2 * 60 * 60 * 1000,
  expect: {
    timeout: 30 * 1000,
  },
  use: {
    baseURL: process.env.ENDURO_URL ?? "http://enduro:9000",
    trace: "retain-on-failure",
  },
});
