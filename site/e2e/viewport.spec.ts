import { expect, test, type Locator } from "@playwright/test";
import { canvasOf, image, mapping, open, rect, scrollSettingsToBottom, scrollerOf, sequence, srcOf, yamlDoc } from "./helpers";

test("a. editing a link at the bottom of the settings keeps the preview inside the window", async ({ page }) => {
  await open(page);
  const viewport = page.viewportSize();
  expect(viewport).not.toBeNull();
  const width = viewport?.width ?? 0;
  const height = viewport?.height ?? 0;

  const scrolled = await scrollSettingsToBottom(page);

  const before = await srcOf(image(page, "links/youtube"));
  const field = page.getByLabel("Link 1 value");
  await field.fill("@ghstats-e2e");
  await expect.poll(() => srcOf(image(page, "links/youtube"))).not.toBe(before);

  const canvas = await rect(canvasOf(page));
  expect(canvas.top).toBeGreaterThanOrEqual(0);
  expect(canvas.left).toBeGreaterThanOrEqual(0);
  expect(canvas.right).toBeLessThanOrEqual(width);
  expect(canvas.bottom).toBeLessThanOrEqual(height);
  expect(await page.evaluate(() => window.scrollY)).toBe(0);
  expect(scrolled).toBeGreaterThan(0);

  const profile = mapping(mapping(await yamlDoc(page)).profile);
  expect(sequence(profile.links).map(mapping)[0]).toEqual({ youtube: "@ghstats-e2e" });
});

const viewports = [
  { name: "desktop", width: 1440, height: 900 },
  { name: "mobile", width: 390, height: 844 },
];

async function covered(node: Locator): Promise<string[]> {
  return node.evaluate((node) => {
    const box = node.getBoundingClientRect();
    const inset = 10;
    const points: [string, number, number][] = [
      ["top-left", box.left + inset, box.top + inset],
      ["top-right", box.right - inset, box.top + inset],
      ["bottom-left", box.left + inset, box.bottom - inset],
      ["bottom-right", box.right - inset, box.bottom - inset],
      ["center", box.left + box.width / 2, box.top + box.height / 2],
    ];
    return points.flatMap(([name, x, y]) => {
      const hit = document.elementFromPoint(x, y);
      return hit !== null && node.contains(hit) ? [] : [`${name} at ${Math.round(x)},${Math.round(y)}: ${hit === null ? "nothing" : hit.tagName.toLowerCase() + "." + hit.className}`];
    });
  });
}

for (const viewport of viewports) {
  for (const scheme of ["light", "dark"] as const) {
    test(`j. ${viewport.name} ${viewport.width}x${viewport.height} ${scheme}: no horizontal scroll and nothing over the canvas`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await open(page, scheme);
      await page.getByRole("group", { name: "Preview mode" }).getByRole("button", { name: scheme === "light" ? "Light" : "Dark", exact: true }).click();
      await expect(canvasOf(page)).toHaveAttribute("data-mode", scheme);

      const widths = await page.evaluate(() => ({
        viewport: window.innerWidth,
        root: document.documentElement.scrollWidth,
        body: document.body.scrollWidth,
      }));
      expect(widths.root).toBe(widths.viewport);
      expect(widths.body).toBeLessThanOrEqual(widths.viewport);

      const readme = await rect(canvasOf(page));
      expect.soft(readme.left).toBeGreaterThanOrEqual(0);
      expect.soft(readme.right).toBeLessThanOrEqual(viewport.width);
      expect.soft(readme.top).toBeGreaterThanOrEqual(0);
      expect.soft(readme.bottom).toBeLessThanOrEqual(viewport.height);

      if (viewport.name === "mobile") {
        const holder = await rect(scrollerOf(page));
        expect.soft(readme.top - holder.top).toBeLessThanOrEqual(17);
      }

      expect(await covered(scrollerOf(page))).toEqual([]);
      expect(await covered(canvasOf(page))).toEqual([]);
    });
  }
}
