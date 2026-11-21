import type { Options } from "./ghstats";
import { parseYaml, type Yaml } from "./miniyaml";

export interface BlockState {
  name: string;
  variants: string[];
}

export interface CardState {
  name: string;
  layout: string;
  header: boolean;
  blocks: BlockState[];
}

export interface RowState {
  align: string;
  items: string[];
}

export interface FocusState {
  name: string;
  building: string;
}

export interface LinkState {
  kind: string;
  value: string;
  label: string;
  icon: string;
}

export interface FormState {
  preset: string;
  accent: string;
  accentDark: string;
  accentLight: string;
  palette: string;
  paletteColors: string;
  layout: string;
  motion: boolean;
  multipliers: boolean;
  cards: CardState[];
  rows: RowState[];
  name: string;
  tagline: string;
  location: string;
  note: string;
  focus: FocusState[];
  links: LinkState[];
  linkStyle: string;
}

export const CUSTOM = "custom";
export const DEFAULT_CARD = "stats";
export const MAX_CARDS = 12;
export const MAX_ROWS = 12;
export const ITEM_IDENTITY = "identity";
export const ITEM_LINKS = "links";
export const ALIGNS = ["left", "center", "right"] as const;

type Dict = { [key: string]: Yaml };

function dict(value: Yaml | undefined): Dict {
  return value !== null && typeof value === "object" && !Array.isArray(value) ? value : {};
}

function list(value: Yaml | undefined): Yaml[] {
  return Array.isArray(value) ? value : [];
}

function text(value: Yaml | undefined): string {
  return typeof value === "string" ? value : typeof value === "number" ? String(value) : "";
}

function blockStates(value: Yaml | undefined, options: Options): BlockState[] {
  const blocks: BlockState[] = list(value).map((item) => {
    const [name = "", variants] = Object.entries(dict(item))[0] ?? [];
    return { name, variants: list(variants).map(text) };
  });
  for (const known of options.blocks) {
    if (!blocks.some((block) => block.name === known.name)) blocks.push({ name: known.name, variants: [] });
  }
  return blocks;
}

export function isSplit(cards: CardState[]): boolean {
  return cards.length > 1 || (cards[0]?.name ?? "") !== "";
}

export function cardNames(cards: CardState[]): string[] {
  return isSplit(cards) ? cards.map((card) => card.name) : [DEFAULT_CARD];
}

export function newCard(name: string, options: Options): CardState {
  const blocks = options.blocks.map((block, at) => ({ name: block.name, variants: at === 0 ? block.variants.slice(0, 1) : [] }));
  return { name, layout: "", header: true, blocks };
}

export function hasIdentity(form: FormState): boolean {
  return form.tagline.trim() !== "" || form.focus.some((entry) => entry.name.trim() !== "" || entry.building.trim() !== "");
}

export function hasLinks(form: FormState): boolean {
  return form.links.some((link) => link.value.trim() !== "");
}

export function splitCards(form: FormState, options: Options): { cards: CardState[]; rows: RowState[] } {
  const first = form.cards[0];
  const checked = (first?.blocks ?? []).filter((block) => block.variants.length > 0);
  const cards: CardState[] = checked.map((block, at) => ({
    name: block.name,
    layout: "",
    header: at === 0,
    blocks: options.blocks.map((known) => (known.name === block.name ? block : { name: known.name, variants: [] })),
  }));
  const rows: RowState[] = cards.map((card) => ({ align: "center", items: [card.name] }));
  if (hasIdentity(form)) rows.unshift({ align: "center", items: [ITEM_IDENTITY] });
  if (hasLinks(form)) rows.push({ align: "center", items: [ITEM_LINKS] });
  return { cards, rows };
}

export function reconcileRows(rows: RowState[], before: CardState[], after: CardState[]): RowState[] {
  const known = new Set(after.map((card) => card.name));
  const renamed = new Map<string, string>();
  if (before.length === after.length) {
    before.forEach((card, at) => {
      const to = after[at]?.name;
      if (to !== undefined && to !== card.name) renamed.set(card.name, to);
    });
  }
  return rows.map((row) => ({
    ...row,
    items: row.items.map((item) => renamed.get(item) ?? item).filter((item) => item === ITEM_IDENTITY || item === ITEM_LINKS || known.has(item)),
  }));
}

export function fromYaml(source: string, options: Options): FormState {
  const root = dict(parseYaml(source));
  const render = dict(root.render);
  const theme = dict(render.theme);
  const profile = dict(root.profile);
  const accent = theme.accent;
  const accentMap = dict(accent);
  const palette = dict(render.languages).colors;

  const declared = list(render.cards);
  const cards: CardState[] =
    declared.length > 0
      ? declared.map((entry) => ({
          name: text(dict(entry).name),
          layout: text(dict(entry).layout),
          header: dict(entry).header !== false,
          blocks: blockStates(dict(entry).blocks, options),
        }))
      : [{ name: "", layout: "", header: true, blocks: blockStates(render.blocks, options) }];

  return {
    preset: text(theme.preset) || (options.presets[0] ?? ""),
    accent: typeof accent === "string" ? accent : CUSTOM,
    accentDark: text(accentMap.dark) || "#9ec5ff",
    accentLight: text(accentMap.light) || "#2457c5",
    palette: typeof palette === "string" ? palette : palette === undefined ? "accent" : CUSTOM,
    paletteColors: Object.entries(dict(palette))
      .map(([language, color]) => `${language}: ${text(color)}`)
      .join("\n"),
    layout: text(render.layout) || "card",
    motion: render.motion !== false,
    multipliers: dict(render.languages).multipliers !== false,
    cards,
    rows: list(render.rows).map((entry) => ({
      align: text(dict(entry).align) || "center",
      items: list(dict(entry).items).map(text),
    })),
    name: text(profile.name),
    tagline: text(profile.tagline),
    location: text(profile.location),
    note: text(profile.note),
    focus: list(profile.focus).map((entry) => ({
      name: text(dict(entry).name),
      building: list(dict(entry).building).map(text).join(", "),
    })),
    links: list(profile.links).map((entry) => {
      const [kind = "", value] = Object.entries(dict(entry))[0] ?? [];
      const detail = dict(value);
      if (kind === CUSTOM) return { kind, value: text(detail.url), label: text(detail.label), icon: text(detail.icon) };
      return { kind, value: text(value), label: "", icon: "" };
    }),
    linkStyle: text(profile.link_style),
  };
}

function words(value: string): string[] {
  return value
    .split(",")
    .map((word) => word.trim())
    .filter((word) => word !== "");
}

function paletteMap(value: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const line of value.split("\n")) {
    const at = line.lastIndexOf(":");
    if (at <= 0) continue;
    const language = line.slice(0, at).trim();
    if (language !== "") out[language] = line.slice(at + 1).trim();
  }
  return out;
}

function outputRows(form: FormState): { align: string; items: string[] }[] {
  if (!isSplit(form.cards)) return [];
  const names = new Set(form.cards.map((card) => card.name.trim()).filter((name) => name !== ""));
  const placed = new Set<string>();
  const out: { align: string; items: string[] }[] = [];
  for (const row of form.rows) {
    const items: string[] = [];
    for (const raw of row.items) {
      const item = raw.trim();
      if (placed.has(item) || !(item === ITEM_IDENTITY || item === ITEM_LINKS || names.has(item))) continue;
      placed.add(item);
      items.push(item);
    }
    if (items.length > 0) out.push({ align: row.align, items });
  }
  return out;
}

export function toConfig(form: FormState): Record<string, unknown> {
  const profile: Record<string, unknown> = {};
  if (form.name.trim() !== "") profile.name = form.name.trim();
  if (form.tagline.trim() !== "") profile.tagline = form.tagline.trim();
  if (form.location.trim() !== "") profile.location = form.location.trim();
  if (form.note.trim() !== "") profile.note = form.note.trim();
  const focus = form.focus
    .filter((entry) => entry.name.trim() !== "" || words(entry.building).length > 0)
    .map((entry) => ({ name: entry.name.trim(), building: words(entry.building) }));
  if (focus.length > 0) profile.focus = focus;
  const links = form.links
    .filter((link) => link.value.trim() !== "")
    .map((link) => {
      if (link.kind !== CUSTOM) return { [link.kind]: link.value.trim() };
      const detail: Record<string, string> = { url: link.value.trim() };
      if (link.label.trim() !== "") detail.label = link.label.trim();
      if (link.icon.trim() !== "") detail.icon = link.icon.trim();
      return { [CUSTOM]: detail };
    });
  if (links.length > 0) profile.links = links;
  if (form.linkStyle !== "") profile.link_style = form.linkStyle;

  const blocksOf = (blocks: BlockState[]) =>
    blocks.filter((block) => block.variants.length > 0).map((block) => ({ [block.name]: block.variants }));
  const rows = outputRows(form);
  const content = isSplit(form.cards)
    ? {
        cards: form.cards.map((card) => ({
          name: card.name.trim(),
          ...(card.layout !== "" ? { layout: card.layout } : {}),
          ...(card.header ? {} : { header: false }),
          blocks: blocksOf(card.blocks),
        })),
        ...(rows.length > 0 ? { rows } : {}),
      }
    : { blocks: blocksOf(form.cards[0]?.blocks ?? []) };

  const config: Record<string, unknown> = {
    version: 1,
    render: {
      ...content,
      languages: {
        colors: form.palette === CUSTOM ? paletteMap(form.paletteColors) : form.palette,
        ...(form.multipliers ? {} : { multipliers: false }),
      },
      theme: {
        preset: form.preset,
        accent: form.accent === CUSTOM ? { dark: form.accentDark, light: form.accentLight } : form.accent,
      },
      motion: form.motion,
      layout: form.layout,
    },
  };
  if (Object.keys(profile).length > 0) config.profile = profile;
  return config;
}

export function presetRender(
  block: string,
  variant: string,
  preset: string,
  accent: string,
  palette: string,
): Record<string, unknown> {
  return {
    version: 1,
    render: {
      blocks: [{ [block]: [variant] }],
      languages: { colors: palette },
      theme: { preset, accent },
      motion: false,
    },
  };
}
