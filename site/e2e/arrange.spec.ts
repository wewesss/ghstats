import { expect, test } from "@playwright/test";
import { drag, element, image, itemsOf, mapping, open, rect, selectCard, sequence, snippetRows, yamlDoc } from "./helpers";

test.beforeEach(async ({ page }) => {
  await open(page);
});

test("d. dragging the languages card next to stats puts both in one paragraph at 49%", async ({ page }) => {
  await selectCard(page, "stats");
  await page.locator('[data-testid="block"][data-block="languages"]').getByTestId("make-card").click();
  await expect(element(page, "languages")).toBeVisible();

  await drag(element(page, "languages"), page.getByTestId("drop-newrow").last());
  await expect
    .poll(async () => (await snippetRows(page)).find((row) => itemsOf(row).includes("languages"))?.shots.length)
    .toBe(1);
  const apart = await snippetRows(page);
  expect(apart.find((row) => itemsOf(row).includes("stats"))?.shots).toHaveLength(1);

  await drag(element(page, "languages"), element(page, "stats").getByTestId("drop-after"));
  await expect
    .poll(async () => (await snippetRows(page)).find((row) => itemsOf(row).includes("languages"))?.shots.length)
    .toBe(2);
  const rows = await snippetRows(page);
  const shared = rows.find((row) => itemsOf(row).includes("languages"));
  expect(itemsOf(shared ?? { tag: "p", align: "", shots: [] })).toEqual(["stats", "languages"]);
  expect(shared?.tag).toBe("p");
  expect(shared?.shots.map((shot) => shot.width)).toEqual(["49%", "49%"]);

  const stats = await rect(image(page, "stats"));
  const languages = await rect(image(page, "languages"));
  expect(Math.abs(stats.height - languages.height)).toBeLessThan(1);
  expect(Math.abs(stats.top - languages.top)).toBeLessThan(1);
  expect(languages.left).toBeGreaterThanOrEqual(stats.right - 1);

  const render = mapping(mapping(await yamlDoc(page)).render);
  const row = sequence(render.rows).map(mapping).find((entry) => sequence(entry.items).includes("languages"));
  expect(sequence(row?.items)).toEqual(["stats", "languages"]);
});

test("e. aligning a row right writes align=right and sticks the card to the right edge", async ({ page }) => {
  const row = page.getByTestId("row").filter({ has: element(page, "stats") });
  await row.hover();
  await row.getByTestId("align-right").click();

  await expect.poll(async () => (await snippetRows(page)).find((entry) => itemsOf(entry).includes("stats"))?.align).toBe("right");
  const rows = await snippetRows(page);
  expect(rows.filter((entry) => entry.align === "right")).toHaveLength(1);
  expect(rows.filter((entry) => entry.align === "center").length).toBeGreaterThan(0);

  const doc = mapping(mapping(await yamlDoc(page)).render);
  const entry = sequence(doc.rows).map(mapping).find((candidate) => sequence(candidate.items).includes("stats"));
  expect(entry?.align).toBe("right");

  const column = await rect(page.getByTestId("canvas"));
  const card = await rect(image(page, "stats"));
  const scale = Number(await page.getByTestId("canvas").getAttribute("data-scale"));
  const padding = 36 * scale;
  expect(Math.abs(column.right - card.right - padding)).toBeLessThan(4);
  expect(card.left - column.left).toBeGreaterThan(column.width / 3);
});

test("h. a link badge dragged to a new row sits alone in its own paragraph", async ({ page }) => {
  await drag(element(page, "links/youtube"), page.getByTestId("drop-newrow").nth(1));
  await expect
    .poll(async () => (await snippetRows(page)).some((row) => row.shots.length === 1 && itemsOf(row)[0] === "links/youtube"))
    .toBe(true);
  const rows = await snippetRows(page);
  const alone = rows.filter((row) => row.tag === "p" && itemsOf(row).length === 1 && itemsOf(row)[0] === "links/youtube");
  expect(alone).toHaveLength(1);
  const others = rows.find((row) => itemsOf(row).includes("links/website"));
  expect(itemsOf(others ?? { tag: "p", align: "", shots: [] })).toEqual(["links/website", "links/github_sponsors"]);

  const doc = mapping(mapping(await yamlDoc(page)).render);
  const single = sequence(doc.rows).map(mapping).find((entry) => sequence(entry.items).includes("links/youtube"));
  expect(sequence(single?.items)).toEqual(["links/youtube"]);
});
