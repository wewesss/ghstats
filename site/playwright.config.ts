import { defineConfig, devices } from "@playwright/test";

const port = Number(process.env.PORT ?? 4317);

export default defineConfig({
  testDir: "e2e",
  fullyParallel: true,
  forbidOnly: process.env.CI !== undefined,
  retries: 0,
  workers: process.env.CI !== undefined ? 2 : undefined,
  reporter: [["list"]],
  timeout: 60_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: `http://127.0.0.1:${port}/ghstats/en/builder/`,
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 900 } } }],
  webServer: {
    command: "bun run scripts/serve-dist.ts",
    url: `http://127.0.0.1:${port}/ghstats/en/builder/`,
    reuseExistingServer: false,
    timeout: 30_000,
    env: { PORT: String(port), ...(process.env.GHSTATS_DIST === undefined ? {} : { GHSTATS_DIST: process.env.GHSTATS_DIST }) },
  },
});
