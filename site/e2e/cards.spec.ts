import { expect, test } from "@playwright/test";
import { drag, element, image, itemsOf, mapping, open, sequence, selectCard, snippetRows, srcOf, svgOf, yamlDoc, yamlText } from "./helpers";

test.beforeEach(async ({ page }) => {
  await open(page);
});

test("b. changing the theme preset changes the card images", async ({ page }) => {
  const before = await srcOf(image(page, "stats"));
  const select = page.getByLabel("Preset");
  const current = await select.inputValue();
  const values = await select.locator("option").evaluateAll((options) => options.map((option) => (option as HTMLOptionElement).value));
  const next = values.find((value) => value !== current);
  expect(next).toBeDefined();
  await select.selectOption(next ?? "");
  await expect.poll(() => srcOf(image(page, "stats"))).not.toBe(before);
  const render = mapping(mapping(await yamlDoc(page)).render);
  expect(mapping(render.theme).preset).toBe(next);
  expect(await svgOf(image(page, "stats"))).toContain("<svg");
});

test("c. make it a card on the languages block gives two cards without duplicates", async ({ page }) => {
  await selectCard(page, "stats");
  await page.locator('[data-testid="block"][data-block="languages"]').getByTestId("make-card").click();

  await expect(page.getByTestId("card-item")).toHaveCount(2);
  await expect(element(page, "stats")).toBeVisible();
  await expect(element(page, "languages")).toBeVisible();

  const render = mapping(mapping(await yamlDoc(page)).render);
  const cards = sequence(render.cards).map(mapping);
  expect(cards.map((card) => card.name)).toEqual(["stats", "languages"]);
  for (const card of cards) expect(sequence(card.blocks).length).toBeGreaterThan(0);
  const blocks = cards.flatMap((card) => sequence(card.blocks).map((block) => Object.keys(mapping(block))[0]));
  expect(blocks.sort()).toEqual(["history", "languages", "stats"]);
  const placed = sequence(render.rows).flatMap((row) => sequence(mapping(row).items));
  expect(placed.filter((item) => item === "stats")).toHaveLength(1);
  expect(placed.filter((item) => item === "languages")).toHaveLength(1);

  const shots = (await snippetRows(page)).flatMap(itemsOf);
  expect(shots.filter((item) => item === "stats")).toHaveLength(1);
  expect(shots.filter((item) => item === "languages")).toHaveLength(1);
});

test("f. turning the header off removes it from the card", async ({ page }) => {
  expect(await svgOf(image(page, "stats"))).toContain("synced");
  await selectCard(page, "stats");
  await page.getByTestId("card-header").uncheck();
  await expect.poll(async () => svgOf(image(page, "stats"))).not.toContain("synced");
  const render = mapping(mapping(await yamlDoc(page)).render);
  expect(mapping(sequence(render.cards)[0]).header).toBe(false);
  expect((await snippetRows(page)).flatMap(itemsOf)).toContain("stats");
});

test("g. the identity card takes its parts and its width from the selection", async ({ page }) => {
  await selectCard(page, "identity");
  const before = await srcOf(image(page, "identity"));
  for (const part of ["note", "location", "footer"]) await page.getByTestId(`part-${part}`).uncheck();
  await page.getByTestId("section-selection").getByLabel("Width").selectOption("card");
  await expect.poll(() => srcOf(image(page, "identity"))).not.toBe(before);

  const profile = mapping(mapping(await yamlDoc(page)).profile);
  const identity = mapping(profile.identity);
  expect(identity.layout).toBe("card");
  expect(identity.show).toEqual(["header", "name", "tagline", "building", "focus"]);

  const rows = await snippetRows(page);
  const own = rows.find((row) => itemsOf(row).includes("identity"));
  expect(own).toBeDefined();
  expect(own?.shots).toHaveLength(1);
  expect(own?.shots[0]?.width).toBe("495");
});

test("g. the identity card shares a row at 49% each", async ({ page }) => {
  await selectCard(page, "identity");
  await page.getByTestId("section-selection").getByLabel("Width").selectOption("card");
  await expect(yamlText(page)).resolves.toContain("layout: card");
  const rows = await snippetRows(page);
  const alone = rows.find((row) => itemsOf(row).includes("identity"));
  expect(alone?.shots[0]?.width).toBe("495");

  await drag(element(page, "identity"), element(page, "stats").getByTestId("drop-before"));
  await expect.poll(async () => (await snippetRows(page)).find((row) => itemsOf(row).includes("identity"))?.shots.length).toBe(2);
  const shared = (await snippetRows(page)).find((row) => itemsOf(row).includes("identity"));
  expect(itemsOf(shared ?? { tag: "", align: "", shots: [] })).toEqual(["identity", "stats"]);
  expect(shared?.shots.map((shot) => shot.width)).toEqual(["49%", "49%"]);
});
