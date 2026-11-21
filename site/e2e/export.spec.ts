import { readFileSync } from "node:fs";
import { expect, test, type Download } from "@playwright/test";
import { element, image, open, svgOf } from "./helpers";

const PNG_MAGIC = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];

test.beforeEach(async ({ page }) => {
  await open(page);
});

async function bytesOf(download: Download): Promise<Buffer> {
  const path = await download.path();
  return readFileSync(path);
}

function zipNames(data: Buffer): string[] {
  const end = data.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]));
  expect(end).toBeGreaterThan(0);
  const count = data.readUInt16LE(end + 10);
  let at = data.readUInt32LE(end + 16);
  const names: string[] = [];
  for (let i = 0; i < count; i++) {
    expect(data.readUInt32LE(at)).toBe(0x02014b50);
    const nameLength = data.readUInt16LE(at + 28);
    const extraLength = data.readUInt16LE(at + 30);
    const commentLength = data.readUInt16LE(at + 32);
    names.push(data.subarray(at + 46, at + 46 + nameLength).toString("utf8"));
    at += 46 + nameLength + extraLength + commentLength;
  }
  return names;
}

test("l. Download PNG of the selected element is a 2x PNG named after the element and the mode", async ({ page }) => {
  await element(page, "stats").click();
  const svg = await svgOf(image(page, "stats"));
  const viewBox = /viewBox="0 0 ([\d.]+) ([\d.]+)"/.exec(svg);
  expect(viewBox).not.toBeNull();
  const width = Number(viewBox?.[1]);
  const height = Number(viewBox?.[2]);

  const [download] = await Promise.all([page.waitForEvent("download"), page.getByTestId("download-png").click()]);
  expect(download.suggestedFilename()).toBe("stats-dark.png");
  const data = await bytesOf(download);
  expect(Array.from(data.subarray(0, 8))).toEqual(PNG_MAGIC);
  expect(data.readUInt32BE(16)).toBe(Math.ceil(width) * 2);
  expect(data.readUInt32BE(20)).toBe(Math.ceil(height) * 2);
});

test("l. Download SVG of the selected element follows the preview mode", async ({ page }) => {
  await page.getByTestId("mode-light").click();
  await element(page, "stats").click();
  const [download] = await Promise.all([page.waitForEvent("download"), page.getByTestId("download-svg").click()]);
  expect(download.suggestedFilename()).toBe("stats-light.svg");
  expect((await bytesOf(download)).toString("utf8")).toContain("<svg");
});

test("l. Download all (.zip) holds every element in SVG and PNG, dark and light", async ({ page }) => {
  const items = await page.locator('[data-testid="element"]').evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).dataset.item ?? ""));
  expect(items.length).toBeGreaterThan(0);

  const [download] = await Promise.all([page.waitForEvent("download"), page.getByTestId("download-zip").click()]);
  expect(download.suggestedFilename()).toBe("ghstats.zip");
  const names = zipNames(await bytesOf(download));

  const expected = items.flatMap((item) => ["dark", "light"].flatMap((mode) => [`${item}-${mode}.svg`, `${item}-${mode}.png`]));
  expect([...names].sort()).toEqual([...expected].sort());
});
