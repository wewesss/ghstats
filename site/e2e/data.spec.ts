import { readFileSync } from "node:fs";
import { join } from "node:path";
import { expect, test } from "@playwright/test";
import { image, open, svgOf } from "./helpers";

const fixture = readFileSync(join(import.meta.dirname, "fixtures", "stats.json"));
const fixtureUrl = "https://raw.githubusercontent.com/fixturedev/fixturedev/HEAD/ghstats/stats.json";

test.beforeEach(async ({ page }) => {
  await open(page);
});

test("i. data by login comes from the mocked raw.githubusercontent.com", async ({ page }) => {
  const requested: string[] = [];
  await page.route("https://raw.githubusercontent.com/**", async (route) => {
    requested.push(route.request().url());
    await route.fulfill({ status: 200, contentType: "application/json", body: fixture });
  });

  const sample = await svgOf(image(page, "stats"));
  expect(sample).toContain("128");
  expect(sample).not.toContain("7,421");
  await expect(page.getByTestId("data-source")).toContainText("sample data");

  await page.getByTestId("data-kind").selectOption("github");
  await page.getByTestId("data-login").fill("fixturedev");
  await page.getByTestId("data-load").click();

  await expect(page.getByTestId("data-source")).toContainText("fixturedev (GitHub)");
  await expect.poll(async () => svgOf(image(page, "stats"))).toContain("7,421");
  expect(await svgOf(image(page, "stats"))).not.toContain("128");
  expect(requested).toEqual([fixtureUrl]);
});

test("i. data by login reports a missing stats.json", async ({ page }) => {
  await page.route("https://raw.githubusercontent.com/**", (route) => route.fulfill({ status: 404, body: "404: Not Found" }));
  await page.getByTestId("data-kind").selectOption("github");
  await page.getByTestId("data-login").fill("nobody");
  await page.getByTestId("data-load").click();
  await expect(page.getByRole("alert").filter({ hasText: "No stats.json" })).toBeVisible();
  await expect(page.getByTestId("data-source")).toContainText("sample data");
  expect(await svgOf(image(page, "stats"))).toContain("128");
});

test("i. data by file upload replaces the sample", async ({ page }) => {
  await page.getByTestId("data-kind").selectOption("file");
  await page.getByTestId("data-file").setInputFiles({ name: "my-stats.json", mimeType: "application/json", buffer: fixture });

  await expect(page.getByTestId("data-source")).toContainText("my-stats.json");
  await expect.poll(async () => svgOf(image(page, "stats"))).toContain("7,421");
  expect(await svgOf(image(page, "stats"))).not.toContain("128");

  await page.getByTestId("data-kind").selectOption("sample");
  await expect(page.getByTestId("data-source")).toContainText("sample data");
  await expect.poll(async () => svgOf(image(page, "stats"))).toContain("128");
});
