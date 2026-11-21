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
  style: string;
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
  weighting: boolean;
  cards: CardState[];
  rows: RowState[];
  name: string;
  tagline: string;
  location: string;
  note: string;
  focus: FocusState[];
  links: LinkState[];
  linkStyle: string;
  identityShow: string[];
  identityLayout: string;
}

export const CUSTOM = "custom";
export const DEFAULT_CARD = "stats";
export const MAX_CARDS = 12;
export const MAX_ROWS = 12;
export const ITEM_IDENTITY = "identity";
export const ITEM_LINKS = "links";
export const LINK_PREFIX = "links/";
export const ALIGNS = ["left", "center", "right"] as const;
export const IDENTITY_PARTS = ["header", "name", "tagline", "location", "note", "building", "focus", "footer"] as const;
export const CARD_NAME = /^[a-z0-9][a-z0-9-]{0,39}$/;

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

export function isLinkItem(item: string): boolean {
  return item === ITEM_LINKS || item.startsWith(LINK_PREFIX);
}

export function linkItem(kind: string): string {
  return `${LINK_PREFIX}${kind}`;
}

function blockStates(value: Yaml | undefined): BlockState[] {
  return list(value).map((item) => {
    const [name = "", variants] = Object.entries(dict(item))[0] ?? [];
    return { name, variants: list(variants).map(text) };
  });
}

export const SHIFT = "shift";
export const SHIFT_REASON = "shift shows the weighting; turn on Show weighting to use it.";

export function withoutShift(card: CardState): CardState {
  return {
    ...card,
    blocks: card.blocks.map((block) => {
      if (block.name !== "languages" || !block.variants.includes(SHIFT)) return block;
      const variants = block.variants.filter((variant) => variant !== SHIFT);
      return { ...block, variants: variants.length > 0 ? variants : ["bars"] };
    }),
  };
}

export function setWeighting(form: FormState, weighting: boolean): FormState {
  return weighting ? { ...form, weighting } : { ...form, weighting, cards: form.cards.map(withoutShift) };
}

export function cardNameProblem(name: string, cards: CardState[], self: number): string {
  const value = name.trim();
  if (value === "") return "A card needs a name.";
  if (value === ITEM_IDENTITY || value === ITEM_LINKS) return `"${value}" is reserved.`;
  if (!CARD_NAME.test(value)) return "Use lowercase letters, digits and hyphens, up to 40 characters.";
  if (cards.some((card, at) => at !== self && card.name === value)) return "Another card already has this name.";
  return "";
}

export function uniqueName(base: string, taken: Iterable<string>): string {
  const used = new Set(taken);
  if (!used.has(base)) return base;
  let at = 2;
  while (used.has(`${base}-${at}`)) at += 1;
  return `${base}-${at}`;
}

function takenNames(form: FormState): string[] {
  return [...form.cards.map((card) => card.name), ITEM_IDENTITY, ITEM_LINKS];
}

export function effectiveLayout(form: FormState, card: CardState): string {
  return card.layout !== "" ? card.layout : form.layout !== "" ? form.layout : "card";
}

export function newCard(form: FormState, options: Options): CardState {
  const shown = new Set(form.cards.flatMap((card) => card.blocks.flatMap((block) => block.variants.map((variant) => `${block.name}/${variant}`))));
  const pairs = options.blocks
    .flatMap((block) => block.variants.map((variant) => ({ block: block.name, variant })))
    .filter((pair) => form.weighting || pair.variant !== SHIFT);
  const pick = pairs.find((pair) => !shown.has(`${pair.block}/${pair.variant}`)) ?? pairs[0];
  const block = pick?.block ?? DEFAULT_CARD;
  return {
    name: uniqueName(block, takenNames(form)),
    layout: "",
    header: true,
    blocks: pick === undefined ? [] : [{ name: pick.block, variants: [pick.variant] }],
  };
}

export function addCard(form: FormState, options: Options): { form: FormState; name: string } {
  const card = newCard(form, options);
  return { form: { ...form, cards: [...form.cards, card] }, name: card.name };
}

export function renameItem(rows: RowState[], from: string, to: string): RowState[] {
  return rows.map((row) => ({ ...row, items: row.items.map((item) => (item === from ? to : item)) }));
}

export function dropItem(rows: RowState[], item: string): RowState[] {
  return rows.map((row) => ({ ...row, items: row.items.filter((entry) => entry !== item) })).filter((row) => row.items.length > 0);
}

export function removeCard(form: FormState, name: string): FormState {
  if (form.cards.length <= 1) return form;
  return { ...form, cards: form.cards.filter((card) => card.name !== name), rows: dropItem(form.rows, name) };
}

export function makeCard(form: FormState, rows: RowState[], from: string, block: string): { form: FormState; name: string } | null {
  const source = form.cards.find((card) => card.name === from);
  const moved = source?.blocks.find((entry) => entry.name === block);
  if (source === undefined || moved === undefined || source.blocks.length < 2) return null;
  const name = uniqueName(moved.name, takenNames(form));
  const created: CardState = { name, layout: "", header: !source.header, blocks: [moved] };
  const cards = form.cards.flatMap((card) =>
    card === source ? [{ ...card, blocks: card.blocks.filter((entry) => entry !== moved) }, created] : [card],
  );
  const sourceRow = rows.findIndex((row) => row.items.includes(from));
  const fits = effectiveLayout(form, source) === "card" && effectiveLayout(form, created) === "card";
  const next = rows.map((row) => ({ ...row, items: [...row.items] }));
  const only = sourceRow >= 0 ? next[sourceRow] : undefined;
  if (only !== undefined && fits && !only.items.some(isLinkItem)) {
    only.items.splice(only.items.indexOf(from) + 1, 0, name);
  } else if (only !== undefined) {
    next.splice(sourceRow + 1, 0, { align: only.align, items: [name] });
  } else {
    next.push({ align: "center", items: [name] });
  }
  return { form: { ...form, cards, rows: next.length > MAX_ROWS ? rows : next }, name };
}

export function identityPartsOn(form: FormState): string[] {
  return form.identityShow.length === 0 ? [...IDENTITY_PARTS] : IDENTITY_PARTS.filter((part) => form.identityShow.includes(part));
}

export function toggleIdentityPart(form: FormState, part: string, on: boolean): FormState {
  const current = identityPartsOn(form);
  const next = IDENTITY_PARTS.filter((entry) => (entry === part ? on : current.includes(entry)));
  if (next.length === 0) return form;
  return { ...form, identityShow: next.length === IDENTITY_PARTS.length ? [] : next };
}

export function partHasValue(form: FormState, part: string): boolean {
  switch (part) {
    case "tagline":
      return form.tagline.trim() !== "";
    case "location":
      return form.location.trim() !== "";
    case "note":
      return form.note.trim() !== "";
    case "building":
    case "focus":
      return form.focus.some((entry) => entry.name.trim() !== "" || words(entry.building).length > 0);
    default:
      return true;
  }
}

export function identityFromProfile(form: FormState): boolean {
  return form.tagline.trim() !== "" || form.focus.some((entry) => entry.name.trim() !== "" || entry.building.trim() !== "");
}

export function linkReady(link: LinkState): boolean {
  return link.value.trim() !== "";
}

export function fromYaml(source: string, options: Options): FormState {
  const root = dict(parseYaml(source));
  const render = dict(root.render);
  const theme = dict(render.theme);
  const profile = dict(root.profile);
  const identity = dict(profile.identity);
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
          blocks: blockStates(dict(entry).blocks),
        }))
      : [{ name: DEFAULT_CARD, layout: "", header: true, blocks: blockStates(render.blocks) }];
  const first = options.blocks[0];
  if (cards.length === 1 && cards[0]?.blocks.length === 0 && first !== undefined) {
    cards[0].blocks = [{ name: first.name, variants: first.variants.slice(0, 1) }];
  }

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
    weighting: dict(render.languages).weighting !== false,
    cards: dict(render.languages).weighting === false ? cards.map(withoutShift) : cards,
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
      if (value !== null && typeof value === "object" && !Array.isArray(value)) {
        return { kind, value: text(value.url), label: text(value.label), icon: text(value.icon), style: text(value.style) };
      }
      return { kind, value: text(value), label: "", icon: "", style: "" };
    }),
    linkStyle: text(profile.link_style),
    identityShow: list(identity.show).map(text),
    identityLayout: text(identity.layout),
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
  const names = new Set(form.cards.map((card) => card.name.trim()).filter((name) => name !== ""));
  const kinds = new Set(form.links.map((link) => link.kind));
  const placed = new Set<string>();
  const out: { align: string; items: string[] }[] = [];
  for (const row of form.rows) {
    const items: string[] = [];
    for (const raw of row.items) {
      const item = raw.trim();
      const known =
        item === ITEM_IDENTITY || item === ITEM_LINKS || names.has(item) || (item.startsWith(LINK_PREFIX) && kinds.has(item.slice(LINK_PREFIX.length)));
      if (placed.has(item) || !known) continue;
      placed.add(item);
      items.push(item);
    }
    if (items.length > 0) out.push({ align: row.align, items });
  }
  return out;
}

export function isPlain(form: FormState): boolean {
  const card = form.cards[0];
  return form.cards.length === 1 && card?.name === DEFAULT_CARD && card.header && card.layout === "" && outputRows(form).length === 0;
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
      const value = link.value.trim();
      if (link.kind !== CUSTOM && link.label.trim() === "" && link.style === "") return { [link.kind]: value };
      const detail: Record<string, string> = { url: value };
      if (link.label.trim() !== "") detail.label = link.label.trim();
      if (link.kind === CUSTOM && link.icon.trim() !== "") detail.icon = link.icon.trim();
      if (link.style !== "") detail.style = link.style;
      return { [link.kind]: detail };
    });
  if (links.length > 0) profile.links = links;
  if (form.linkStyle !== "") profile.link_style = form.linkStyle;
  const identity: Record<string, unknown> = {};
  if (form.identityShow.length > 0) identity.show = IDENTITY_PARTS.filter((part) => form.identityShow.includes(part));
  if (form.identityLayout !== "") identity.layout = form.identityLayout;
  if (Object.keys(identity).length > 0) profile.identity = identity;

  const blocksOf = (blocks: BlockState[]) =>
    blocks.filter((block) => block.variants.length > 0).map((block) => ({ [block.name]: block.variants }));
  const rows = outputRows(form);
  const content = isPlain(form)
    ? { blocks: blocksOf(form.cards[0]?.blocks ?? []) }
    : {
        cards: form.cards.map((card) => ({
          name: card.name.trim(),
          ...(card.layout !== "" ? { layout: card.layout } : {}),
          ...(card.header ? {} : { header: false }),
          blocks: blocksOf(card.blocks),
        })),
        ...(rows.length > 0 ? { rows } : {}),
      };

  const config: Record<string, unknown> = {
    version: 1,
    render: {
      ...content,
      languages: {
        colors: form.palette === CUSTOM ? paletteMap(form.paletteColors) : form.palette,
        ...(form.weighting ? {} : { weighting: false }),
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
