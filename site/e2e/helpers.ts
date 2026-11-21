import { expect, type Locator, type Page } from "@playwright/test";
import { parseYaml, type Yaml } from "../demos/miniyaml";

export interface Shot {
  src: string;
  width: string | null;
}

export interface SnippetRow {
  tag: string;
  align: string;
  shots: Shot[];
}

export async function open(page: Page, scheme: "light" | "dark" = "dark"): Promise<void> {
  await page.emulateMedia({ reducedMotion: "reduce", colorScheme: scheme });
  await page.goto("");
  await expect(page.getByTestId("builder")).toBeVisible();
  await expect(element(page, "stats")).toBeVisible();
}

export function element(page: Page, item: string): Locator {
  return page.locator(`[data-testid="element"][data-item="${item}"]`);
}

export function canvasOf(page: Page): Locator {
  return page.getByTestId("canvas");
}

export function scrollerOf(page: Page): Locator {
  return page.getByTestId("canvas-scroll");
}

export async function scrollSettingsToBottom(page: Page): Promise<number> {
  const settings = page.getByTestId("settings");
  await settings.evaluate((node) => node.scrollTo({ top: node.scrollHeight }));
  await expect.poll(() => settings.evaluate((node) => node.scrollTop)).toBeGreaterThan(0);
  return settings.evaluate((node) => node.scrollTop);
}

export function image(page: Page, item: string): Locator {
  return element(page, item).locator("img");
}

export async function svgOf(img: Locator): Promise<string> {
  const src = (await img.getAttribute("src")) ?? "";
  return decodeURIComponent(src.slice(src.indexOf(",") + 1));
}

export async function srcOf(img: Locator): Promise<string> {
  return (await img.getAttribute("src")) ?? "";
}

async function textOf(page: Page, tab: string, target: string, read: (node: Locator) => Promise<string>): Promise<string> {
  await page.getByTestId(`tab-${tab}`).click();
  const text = await read(page.getByTestId(target));
  await page.getByTestId("tab-preview").click();
  await expect(canvasOf(page)).toBeVisible();
  return text;
}

export function snippet(page: Page): Promise<string> {
  return textOf(page, "snippet", "snippet-text", (node) => node.innerText());
}

export function yamlText(page: Page): Promise<string> {
  return textOf(page, "yaml", "yaml-editor", (node) => node.inputValue());
}

export async function yamlDoc(page: Page): Promise<Record<string, Yaml>> {
  const parsed = parseYaml(await yamlText(page));
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error("the YAML tab did not hold a mapping");
  return parsed;
}

export async function snippetRows(page: Page): Promise<SnippetRow[]> {
  const text = await snippet(page);
  return page.evaluate((source) => {
    const body = new DOMParser().parseFromString(source, "text/html").body;
    return Array.from(body.children).map((node) => ({
      tag: node.tagName.toLowerCase(),
      align: node.getAttribute("align") ?? "",
      shots: Array.from(node.querySelectorAll("img")).map((img) => ({ src: img.getAttribute("src") ?? "", width: img.getAttribute("width") })),
    }));
  }, text);
}

export function nameOf(shot: Shot): string {
  return shot.src.replace(/^.*?ghstats\//, "").replace(/-(dark|light)\.svg$/, "");
}

export function itemsOf(row: SnippetRow): string[] {
  return row.shots.map(nameOf);
}

export async function drag(source: Locator, target: Locator): Promise<void> {
  const page = source.page();
  await source.scrollIntoViewIfNeeded();
  await source.hover();
  await page.mouse.down();
  const from = await source.boundingBox();
  if (from === null) throw new Error("the dragged element has no box");
  await page.mouse.move(from.x + from.width / 2 + 12, from.y + from.height / 2 + 12, { steps: 6 });
  await expect(page.getByTestId("canvas-scroll")).toHaveAttribute("data-dragging", "true");
  await expect(target).toBeVisible();
  const to = await target.boundingBox();
  if (to === null) throw new Error("the drop zone has no box");
  await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 12 });
  await page.mouse.up();
}

export async function selectCard(page: Page, name: string): Promise<void> {
  await element(page, name).click();
  await expect(page.getByTestId("section-selection")).toBeVisible();
}

export function mapping(value: Yaml | undefined): Record<string, Yaml> {
  if (value === null || value === undefined || typeof value !== "object" || Array.isArray(value)) throw new Error("expected a mapping");
  return value;
}

export function sequence(value: Yaml | undefined): Yaml[] {
  if (!Array.isArray(value)) throw new Error("expected a sequence");
  return value;
}

export async function rect(node: Locator): Promise<{ left: number; top: number; right: number; bottom: number; width: number; height: number }> {
  return node.evaluate((el) => {
    const box = el.getBoundingClientRect();
    return { left: box.left, top: box.top, right: box.right, bottom: box.bottom, width: box.width, height: box.height };
  });
}
