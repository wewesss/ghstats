import { MAX_ROWS, isLinkItem, type RowState } from "./form";
import type { Mode } from "./ghstats";

const ALLOWED_ALIGN = new Set(["left", "center", "right"]);

export interface CanvasEntry {
  item: string;
  svg: string;
  alt: string;
  width: string | null;
}

export interface CanvasRow {
  align: string;
  entries: CanvasEntry[];
}

export type Target = { kind: "slot"; row: number; at: number } | { kind: "row"; at: number };

export type Direction = "left" | "right" | "up" | "down";

export function cardKey(src: string, mode: Mode, keys: string[]): string | undefined {
  const wanted = src.replace(/-(dark|light)\.svg$/, `-${mode}.svg`);
  return [...keys].sort((a, b) => b.length - a.length).find((key) => wanted === key || wanted.endsWith(`/${key}`));
}

export function itemOf(key: string): string {
  return key.replace(/-(dark|light)\.svg$/, "");
}

function entryOf(node: Element, mode: Mode, cards: Record<string, string>): CanvasEntry | null {
  const img = node.tagName.toLowerCase() === "img" ? node : node.querySelector("img");
  if (img === null) return null;
  const key = cardKey(img.getAttribute("src") ?? "", mode, Object.keys(cards));
  const svg = key === undefined ? undefined : cards[key];
  if (key === undefined || svg === undefined) return null;
  return { item: itemOf(key), svg, alt: img.getAttribute("alt") ?? "", width: img.getAttribute("width") };
}

export function parseCanvas(snippet: string, mode: Mode, cards: Record<string, string>): CanvasRow[] {
  const body = new DOMParser().parseFromString(snippet, "text/html").body;
  const rows: CanvasRow[] = [];
  const seen = new Set<string>();
  const add = (align: string, entries: (CanvasEntry | null)[]) => {
    const fresh = entries.filter((entry): entry is CanvasEntry => entry !== null && !seen.has(entry.item));
    for (const entry of fresh) seen.add(entry.item);
    if (fresh.length > 0) rows.push({ align, entries: fresh });
  };
  for (const node of Array.from(body.children)) {
    const tag = node.tagName.toLowerCase();
    if (tag === "p") {
      const align = node.getAttribute("align") ?? "";
      add(
        ALLOWED_ALIGN.has(align) ? align : "left",
        Array.from(node.children).map((child) => entryOf(child, mode, cards)),
      );
    } else if (tag === "picture" || tag === "img" || tag === "a") {
      add("left", [entryOf(node, mode, cards)]);
    }
  }
  return rows;
}

export function toRows(canvas: CanvasRow[]): RowState[] {
  return canvas.map((row) => ({ align: row.align, items: row.entries.map((entry) => entry.item) }));
}

function clamp(value: number, low: number, high: number): number {
  return Math.max(low, Math.min(high, value));
}

export function accepts(rows: RowState[], row: number, item: string): boolean {
  const target = rows[row];
  if (target === undefined) return false;
  const link = isLinkItem(item);
  return target.items.every((entry) => entry === item || isLinkItem(entry) === link);
}

export function place(rows: RowState[], item: string, target: Target): RowState[] | null {
  const work = rows.map((row) => ({ align: row.align, items: [...row.items] }));
  const from = work.find((row) => row.items.includes(item));
  let dest: RowState | undefined;
  let at = 0;
  if (target.kind === "row") {
    dest = { align: from?.align ?? "center", items: [] };
    work.splice(clamp(target.at, 0, work.length), 0, dest);
  } else {
    if (!accepts(rows, target.row, item)) return null;
    dest = work[target.row];
    at = target.at;
  }
  if (dest === undefined) return null;
  if (from !== undefined) {
    const index = from.items.indexOf(item);
    from.items.splice(index, 1);
    if (from === dest && index < at) at -= 1;
  }
  dest.items.splice(clamp(at, 0, dest.items.length), 0, item);
  const out = work.filter((row) => row.items.length > 0);
  return out.length > MAX_ROWS ? null : out;
}

export function canPlace(rows: RowState[], item: string, target: Target): boolean {
  return place(rows, item, target) !== null;
}

export function same(a: RowState[], b: RowState[]): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

export function step(rows: RowState[], item: string, direction: Direction): RowState[] | null {
  const row = rows.findIndex((entry) => entry.items.includes(item));
  const current = rows[row];
  if (current === undefined) return null;
  const index = current.items.indexOf(item);
  const alone = current.items.length === 1;
  if (direction === "left") return index === 0 ? null : place(rows, item, { kind: "slot", row, at: index - 1 });
  if (direction === "right") return index === current.items.length - 1 ? null : place(rows, item, { kind: "slot", row, at: index + 2 });
  const delta = direction === "up" ? -1 : 1;
  const neighbour = rows[row + delta];
  if (neighbour !== undefined && accepts(rows, row + delta, item)) {
    return place(rows, item, { kind: "slot", row: row + delta, at: clamp(index, 0, neighbour.items.length) });
  }
  if (alone) {
    if (neighbour === undefined) return null;
    return place(rows, item, { kind: "row", at: direction === "up" ? row - 1 : row + 2 });
  }
  return place(rows, item, { kind: "row", at: direction === "up" ? row : row + 1 });
}

export function setAlign(rows: RowState[], row: number, align: string): RowState[] {
  return rows.map((entry, at) => (at === row ? { ...entry, align } : entry));
}
