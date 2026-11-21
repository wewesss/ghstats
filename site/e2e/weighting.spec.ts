import { expect, test, type Page } from "@playwright/test";
import { image, mapping, open, selectCard, sequence, svgOf, yamlDoc } from "./helpers";

test.beforeEach(async ({ page }) => {
  await open(page);
  await selectCard(page, "stats");
  await page.locator('[data-testid="block"][data-block="languages"]').getByTestId("make-card").click();
  await selectCard(page, "languages");
});

async function texts(page: Page): Promise<string[]> {
  const svg = await svgOf(image(page, "languages"));
  return page.evaluate(
    (source) => Array.from(new DOMParser().parseFromString(source, "image/svg+xml").querySelectorAll("text")).map((node) => (node.textContent ?? "").trim()),
    svg,
  );
}

function traces(list: string[]): string[] {
  return list.filter((text) => text.includes("×") || ["adjusted", "raw"].includes(text.toLowerCase()) || text === "RAW" || text === "ADJ");
}

const shiftToggle = (page: Page) => page.locator('[data-testid="block"][data-block="languages"] [data-testid="variant"][data-variant="shift"]');

test("k. turning Show weighting off hides every trace and disables shift, turning it on restores them", async ({ page }) => {
  await shiftToggle(page).click();
  await expect(shiftToggle(page)).toHaveAttribute("aria-pressed", "true");
  await expect.poll(async () => traces(await texts(page)).length).toBeGreaterThan(0);
  await expect(page.getByTestId("shift-reason")).toHaveCount(0);

  await page.getByTestId("show-weighting").uncheck();

  await expect(shiftToggle(page)).toBeDisabled();
  await expect(shiftToggle(page)).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByTestId("shift-reason")).toBeVisible();
  await expect(page.getByTestId("shift-reason")).toContainText("Show weighting");
  await expect.poll(async () => traces(await texts(page))).toEqual([]);
  const off = await texts(page);
  expect(off.some((text) => text.includes("×"))).toBe(false);
  expect(off.map((text) => text.toLowerCase())).not.toContain("adjusted");
  expect(off).not.toContain("RAW");
  expect(off).not.toContain("ADJ");

  const render = mapping(mapping(await yamlDoc(page)).render);
  expect(mapping(render.languages).weighting).toBe(false);
  const variants = sequence(render.cards)
    .map(mapping)
    .flatMap((card) => sequence(card.blocks).map(mapping))
    .flatMap((block) => (Array.isArray(block.languages) ? block.languages : []));
  expect(variants.length).toBeGreaterThan(0);
  expect(variants).not.toContain("shift");

  await page.getByTestId("show-weighting").check();

  await expect(shiftToggle(page)).toBeEnabled();
  await expect(page.getByTestId("shift-reason")).toHaveCount(0);
  await expect.poll(async () => (await texts(page)).some((text) => text.includes("×"))).toBe(true);
  const on = (await texts(page)).map((text) => text.toLowerCase());
  expect(on).toContain("adjusted");
  expect(on).toContain("raw");
  expect(on).toContain("adj");
});
