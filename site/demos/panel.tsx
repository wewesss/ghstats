import { useState, type ReactNode } from "react";
import {
  CUSTOM,
  IDENTITY_PARTS,
  ITEM_IDENTITY,
  MAX_CARDS,
  MAX_ROWS,
  addCard,
  cardNameProblem,
  dropItem,
  effectiveLayout,
  identityFromProfile,
  identityPartsOn,
  linkItem,
  linkReady,
  makeCard,
  partHasValue,
  removeCard,
  renameItem,
  SHIFT,
  setWeighting,
  toggleIdentityPart,
  type BlockState,
  type CardState,
  type FocusState,
  type FormState,
  type LinkState,
  type RowState,
} from "./form";
import type { Options } from "./ghstats";
import { t, type Key } from "./i18n";

const PART_HELP: Record<string, Key> = {
  header: "partHelpHeader",
  name: "partHelpName",
  tagline: "partHelpTagline",
  location: "partHelpLocation",
  note: "partHelpNote",
  building: "partHelpBuilding",
  focus: "partHelpFocus",
  footer: "partHelpFooter",
};

function Field({ label, children, hint }: { label: string; children: ReactNode; hint?: string }) {
  return (
    <label className="gs-field">
      <span>{label}</span>
      {children}
      {hint !== undefined && hint !== "" ? <small className="gs-hint">{hint}</small> : null}
    </label>
  );
}

function Select({
  value,
  values,
  onChange,
  names,
  disabled,
  label,
}: {
  value: string;
  values: readonly string[];
  onChange: (value: string) => void;
  names?: Record<string, string>;
  disabled?: (value: string) => boolean;
  label?: string;
}) {
  return (
    <select value={value} aria-label={label} onChange={(event) => onChange(event.target.value)}>
      {values.map((entry) => (
        <option key={entry} value={entry} disabled={disabled?.(entry) === true}>
          {names?.[entry] ?? (entry === "" ? t("defaultName") : entry)}
        </option>
      ))}
    </select>
  );
}

function Section({ id, title, children, open }: { id: string; title: string; children: ReactNode; open?: boolean }) {
  return (
    <details className="gs-section" data-testid={`section-${id}`} open={open}>
      <summary>{title}</summary>
      <div className="gs-section-body">{children}</div>
    </details>
  );
}

function Check({ checked, onChange, children, disabled, testid, title }: { checked: boolean; onChange: (value: boolean) => void; children: ReactNode; disabled?: boolean; testid?: string; title?: string }) {
  return (
    <label className="gs-check" title={title}>
      <input type="checkbox" data-testid={testid} checked={checked} disabled={disabled} onChange={(event) => onChange(event.target.checked)} />
      <span>{children}</span>
    </label>
  );
}

interface Props {
  form: FormState;
  options: Options;
  rows: RowState[];
  onCanvas: string[];
  selected: string | null;
  onSelect: (item: string | null) => void;
  onChange: (next: FormState) => void;
}

function BlockRow({
  card,
  block,
  index,
  options,
  weighting,
  onBlocks,
  onMake,
}: {
  card: CardState;
  block: BlockState;
  index: number;
  options: Options;
  weighting: boolean;
  onBlocks: (blocks: BlockState[]) => void;
  onMake: () => void;
}) {
  const known = options.blocks.find((entry) => entry.name === block.name)?.variants ?? [];
  const last = card.blocks.length === 1;
  const shiftBlocked = block.name === "languages" && !weighting;
  const swap = (by: number) => {
    const next = [...card.blocks];
    const target = index + by;
    const [moved] = next.splice(index, 1);
    if (moved !== undefined) next.splice(target, 0, moved);
    onBlocks(next);
  };
  const toggle = (variant: string) => {
    const on = block.variants.includes(variant);
    if (on && block.variants.length === 1) return;
    const variants = known.filter((entry) => (entry === variant ? !on : block.variants.includes(entry)));
    onBlocks(card.blocks.map((entry, at) => (at === index ? { ...entry, variants } : entry)));
  };
  return (
    <div className="gs-chip-block" data-testid="block" data-block={block.name}>
      <div className="gs-chip-head">
        <strong>{block.name}</strong>
        <span className="gs-actions">
          <button type="button" className="gs-button gs-icon-button" aria-label={t("moveBlockUp", { name: block.name })} disabled={index === 0} onClick={() => swap(-1)}>
            ↑
          </button>
          <button type="button" className="gs-button gs-icon-button" aria-label={t("moveBlockDown", { name: block.name })} disabled={index === card.blocks.length - 1} onClick={() => swap(1)}>
            ↓
          </button>
          <button type="button" className="gs-button gs-icon-button" aria-label={t("removeBlock", { name: block.name })} disabled={last} title={last ? t("keepOneBlock") : undefined} onClick={() => onBlocks(card.blocks.filter((_, at) => at !== index))}>
            ×
          </button>
        </span>
      </div>
      <div className="gs-variants" role="group" aria-label={t("blockVariants", { name: block.name })}>
        {known.map((variant) => (
          <button
            key={variant}
            type="button"
            className="gs-toggle"
            data-testid="variant"
            data-variant={variant}
            aria-pressed={block.variants.includes(variant)}
            disabled={shiftBlocked && variant === SHIFT}
            title={shiftBlocked && variant === SHIFT ? t("shiftReason") : undefined}
            onClick={() => toggle(variant)}
          >
            {variant}
          </button>
        ))}
      </div>
      {shiftBlocked && known.includes(SHIFT) ? (
        <small className="gs-hint" data-testid="shift-reason">
          {t("shiftReason")}
        </small>
      ) : null}
      <button type="button" className="gs-button gs-wide" data-testid="make-card" disabled={last} title={last ? t("noOtherBlock") : undefined} onClick={onMake}>
        {t("makeCard")}
      </button>
    </div>
  );
}

function CardSettings({ form, options, rows, card, index, onSelect, onChange }: Props & { card: CardState; index: number }) {
  const update = (patch: Partial<CardState>) => onChange({ ...form, cards: form.cards.map((entry, at) => (at === index ? { ...entry, ...patch } : entry)) });
  const problem = cardNameProblem(card.name, form.cards, index);
  const absent = options.blocks.filter((known) => !card.blocks.some((block) => block.name === known.name));
  const rename = (name: string) => {
    onChange({
      ...form,
      cards: form.cards.map((entry, at) => (at === index ? { ...entry, name } : entry)),
      rows: renameItem(form.rows, card.name, name),
    });
    onSelect(name);
  };
  return (
    <>
      <Field label={t("fieldName")} hint={problem}>
        <input data-testid="card-name" value={card.name} spellCheck={false} aria-invalid={problem !== ""} onChange={(event) => rename(event.target.value)} />
      </Field>
      <Field label={t("layout")}>
        <Select
          value={card.layout}
          values={["", ...options.layouts]}
          names={{ "": t("defaultOf", { value: form.layout || "card" }) }}
          onChange={(layout) => update({ layout })}
        />
      </Field>
      <Check checked={card.header} onChange={(header) => update({ header })} testid="card-header">
        {t("header")}
      </Check>
      <div className="gs-sub">{t("blocks")}</div>
      {card.blocks.map((block, at) => (
        <BlockRow
          key={block.name}
          card={card}
          block={block}
          index={at}
          options={options}
          weighting={form.weighting}
          onBlocks={(blocks) => update({ blocks })}
          onMake={() => {
            const made = makeCard(form, rows, card.name, block.name);
            if (made !== null) {
              onChange(made.form);
              onSelect(made.name);
            }
          }}
        />
      ))}
      <select
        className="gs-add"
        data-testid="add-block"
        aria-label={t("addBlockLabel")}
        value=""
        disabled={absent.length === 0}
        onChange={(event) => {
          const known = options.blocks.find((entry) => entry.name === event.target.value);
          if (known !== undefined) update({ blocks: [...card.blocks, { name: known.name, variants: known.variants.slice(0, 1) }] });
        }}
      >
        <option value="">{absent.length === 0 ? t("everyBlock") : t("addBlock")}</option>
        {absent.map((known) => (
          <option key={known.name} value={known.name}>
            {known.name}
          </option>
        ))}
      </select>
      <button
        type="button"
        className="gs-button gs-danger"
        data-testid="delete-card"
        disabled={form.cards.length <= 1}
        title={form.cards.length <= 1 ? t("lastCard") : undefined}
        onClick={() => {
          onChange(removeCard(form, card.name));
          onSelect(null);
        }}
      >
        {t("deleteCard")}
      </button>
    </>
  );
}

function IdentitySettings({ form, onChange, canRemove, onRemove }: { form: FormState; onChange: (next: FormState) => void; canRemove: boolean; onRemove: () => void }) {
  const on = identityPartsOn(form);
  return (
    <>
      <div className="gs-sub">{t("parts")}</div>
      <div className="gs-parts" role="group" aria-label={t("identityParts")}>
        {IDENTITY_PARTS.map((part) => {
          const empty = !partHasValue(form, part);
          return (
            <Check
              key={part}
              checked={on.includes(part)}
              disabled={on.length === 1 && on.includes(part)}
              onChange={(checked) => onChange(toggleIdentityPart(form, part, checked))}
              testid={`part-${part}`}
              title={t(PART_HELP[part] ?? "partHelpName")}
            >
              {part}
              {empty ? <em>{t("noValue")}</em> : null}
            </Check>
          );
        })}
      </div>
      <small className="gs-hint">{on.length === IDENTITY_PARTS.length ? t("allParts") : t("someParts")}</small>
      <Field label={t("width")}>
        <Select
          value={form.identityLayout}
          values={["", "wide", "card"]}
          names={{ "": t("widthDefault"), wide: t("widthWide"), card: t("widthCard") }}
          onChange={(identityLayout) => onChange({ ...form, identityLayout })}
        />
      </Field>
      <button
        type="button"
        className="gs-button gs-danger"
        data-testid="remove-identity"
        disabled={!canRemove}
        onClick={onRemove}
      >
        {t("removeIdentity")}
      </button>
      {canRemove ? null : <small className="gs-hint">{t("identityKept")}</small>}
    </>
  );
}

function LinkSettings({ form, options, link, index, onChange, onSelect }: { form: FormState; options: Options; link: LinkState; index: number; onChange: (next: FormState) => void; onSelect: (item: string | null) => void }) {
  const kinds = options.link_kinds.map((entry) => entry.kind);
  const info = options.link_kinds.find((entry) => entry.kind === link.kind);
  const patch = (change: Partial<LinkState>) => onChange({ ...form, links: form.links.map((entry, at) => (at === index ? { ...entry, ...change } : entry)) });
  const taken = (kind: string) => kind !== link.kind && form.links.some((entry) => entry.kind === kind);
  return (
    <>
      <Field label={t("kind")}>
        <Select
          value={link.kind}
          values={kinds}
          disabled={taken}
          onChange={(kind) => {
            onChange({
              ...form,
              links: form.links.map((entry, at) => (at === index ? { ...entry, kind } : entry)),
              rows: renameItem(form.rows, linkItem(link.kind), linkItem(kind)),
            });
            onSelect(linkItem(kind));
          }}
        />
      </Field>
      <Field label={info?.url_only === true ? t("url") : t("handleOrUrl")}>
        <input
          data-testid="link-value"
          value={link.value}
          placeholder={info?.url_only === true ? "https://example.com" : t("handlePlaceholder")}
          spellCheck={false}
          onChange={(event) => patch({ value: event.target.value })}
        />
      </Field>
      <Field label={link.kind === CUSTOM ? t("labelRequired") : t("labelOptional")}>
        <input value={link.label} onChange={(event) => patch({ label: event.target.value })} />
      </Field>
      {link.kind === CUSTOM ? (
        <Field label={t("iconOptional")}>
          <input value={link.icon} spellCheck={false} onChange={(event) => patch({ icon: event.target.value })} />
        </Field>
      ) : null}
      <Field label={t("style")}>
        <Select
          value={link.style}
          values={["", ...options.link_styles]}
          names={{ "": t("defaultOf", { value: form.linkStyle || "full" }) }}
          onChange={(style) => patch({ style })}
        />
      </Field>
      <button
        type="button"
        className="gs-button gs-danger"
        onClick={() => {
          onChange({ ...form, links: form.links.filter((_, at) => at !== index), rows: dropItem(form.rows, linkItem(link.kind)) });
          onSelect(null);
        }}
      >
        {t("removeLink")}
      </button>
    </>
  );
}

function FocusEditor({ form, options, onChange }: { form: FormState; options: Options; onChange: (next: FormState) => void }) {
  const update = (index: number, patch: Partial<FocusState>) =>
    onChange({ ...form, focus: form.focus.map((entry, at) => (at === index ? { ...entry, ...patch } : entry)) });
  return (
    <>
      <div className="gs-sub">{t("focus")}</div>
      {form.focus.map((entry, index) => (
        <div className="gs-box" key={index} data-testid="focus-entry">
          <Field label={t("focusN", { n: index + 1 })}>
            <input value={entry.name} onChange={(event) => update(index, { name: event.target.value })} />
          </Field>
          <Field label={t("building", { max: options.limits.building })}>
            <input value={entry.building} onChange={(event) => update(index, { building: event.target.value })} />
          </Field>
          <button type="button" className="gs-button" onClick={() => onChange({ ...form, focus: form.focus.filter((_, at) => at !== index) })}>
            {t("removeFocus", { n: index + 1 })}
          </button>
        </div>
      ))}
      <button
        type="button"
        className="gs-button"
        disabled={form.focus.length >= options.limits.focus}
        onClick={() => onChange({ ...form, focus: [...form.focus, { name: "", building: "" }] })}
      >
        {t("addFocus", { n: form.focus.length, max: options.limits.focus })}
      </button>
    </>
  );
}

export function SettingsPanel(props: Props) {
  const { form, options, rows, onCanvas, selected, onSelect, onChange } = props;
  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => onChange({ ...form, [key]: value });
  const kinds = options.link_kinds.map((entry) => entry.kind);
  const [fresh, setFresh] = useState(-1);

  const cardAt = selected === null ? -1 : form.cards.findIndex((card) => card.name === selected);
  const linkAt = selected !== null && selected.startsWith("links/") ? form.links.findIndex((link) => linkItem(link.kind) === selected) : -1;
  const identitySelected = selected === ITEM_IDENTITY;
  const card = form.cards[cardAt];
  const link = form.links[linkAt];
  const title = identitySelected ? t("identityCard") : card !== undefined ? t("cardNamed", { name: card.name }) : link !== undefined ? t("linkNamed", { name: link.kind }) : "";
  const showSelection = (identitySelected && onCanvas.includes(ITEM_IDENTITY)) || card !== undefined || link !== undefined;

  const addLink = () => {
    const kind = kinds.find((entry) => !form.links.some((existing) => existing.kind === entry)) ?? kinds[0] ?? "";
    setFresh(form.links.length);
    onChange({ ...form, links: [...form.links, { kind, value: "", label: "", icon: "", style: "" }] });
  };

  const identityOnCanvas = onCanvas.includes(ITEM_IDENTITY);
  const addIdentity = () => {
    if (rows.length >= MAX_ROWS) return;
    onChange({ ...form, rows: [{ align: "center", items: [ITEM_IDENTITY] }, ...rows] });
    onSelect(ITEM_IDENTITY);
  };
  const removeIdentity = () => {
    onChange({ ...form, identityShow: [], rows: dropItem(form.rows, ITEM_IDENTITY) });
    onSelect(null);
  };

  return (
    <div className="gs-form">
      {showSelection ? (
        <section className="gs-selection" data-testid="section-selection" aria-label={t("selection")}>
          <header>
            <strong>{title}</strong>
            <button type="button" className="gs-button" data-testid="deselect" onClick={() => onSelect(null)}>
              {t("done")}
            </button>
          </header>
          <div className="gs-section-body">
            {card !== undefined ? <CardSettings {...props} card={card} index={cardAt} /> : null}
            {identitySelected ? (
              <IdentitySettings form={form} onChange={onChange} canRemove={!identityFromProfile(form)} onRemove={removeIdentity} />
            ) : null}
            {link !== undefined ? <LinkSettings form={form} options={options} link={link} index={linkAt} onChange={onChange} onSelect={onSelect} /> : null}
          </div>
        </section>
      ) : null}

      <Section id="cards" title={t("sectionCards")} open>
        <div className="gs-list" data-testid="card-list">
          {form.cards.map((entry) => (
            <button
              key={entry.name}
              type="button"
              className="gs-listitem"
              data-testid="card-item"
              aria-pressed={selected === entry.name}
              onClick={() => onSelect(entry.name)}
            >
              <strong>{entry.name === "" ? t("unnamed") : entry.name}</strong>
              <span>{entry.blocks.map((block) => block.name).join(", ")}</span>
              <small>{effectiveLayout(form, entry) === "wide" ? "wide" : "card"}</small>
            </button>
          ))}
          {identityOnCanvas ? (
            <button type="button" className="gs-listitem" data-testid="identity-item" aria-pressed={identitySelected} onClick={() => onSelect(ITEM_IDENTITY)}>
              <strong>identity</strong>
              <span>{t("profileCard")}</span>
              <small>{form.identityLayout === "card" ? "card" : "wide"}</small>
            </button>
          ) : null}
        </div>
        <div className="gs-actions-row">
          <button
            type="button"
            className="gs-button"
            data-testid="add-card"
            disabled={form.cards.length >= MAX_CARDS}
            onClick={() => {
              const added = addCard(form, options);
              onChange(added.form);
              onSelect(added.name);
            }}
          >
            {t("addCard", { n: form.cards.length, max: MAX_CARDS })}
          </button>
          {identityOnCanvas ? null : (
            <button type="button" className="gs-button" data-testid="add-identity" onClick={addIdentity}>
              {t("addIdentity")}
            </button>
          )}
        </div>
      </Section>

      <Section id="theme" title={t("sectionTheme")} open>
        <Field label={t("preset")}>
          <Select value={form.preset} values={options.presets} onChange={(value) => set("preset", value)} />
        </Field>
        <Field label={t("accent")}>
          <Select value={form.accent} values={[...options.accents, CUSTOM]} onChange={(value) => set("accent", value)} />
        </Field>
        {form.accent === CUSTOM ? (
          <div className="gs-pair">
            <Field label={t("accentDark")}>
              <input type="color" value={form.accentDark} onChange={(event) => set("accentDark", event.target.value)} />
            </Field>
            <Field label={t("accentLight")}>
              <input type="color" value={form.accentLight} onChange={(event) => set("accentLight", event.target.value)} />
            </Field>
          </div>
        ) : null}
        <Field label={t("languageColors")}>
          <Select value={form.palette} values={[...options.palettes, CUSTOM]} onChange={(value) => set("palette", value)} />
        </Field>
        {form.palette === CUSTOM ? (
          <Field label={t("paletteLines")}>
            <textarea rows={4} value={form.paletteColors} onChange={(event) => set("paletteColors", event.target.value)} spellCheck={false} />
          </Field>
        ) : null}
        <Field label={t("defaultLayout")}>
          <Select value={form.layout} values={options.layouts} onChange={(value) => set("layout", value)} />
        </Field>
        <Check checked={form.weighting} onChange={(value) => onChange(setWeighting(form, value))} testid="show-weighting">
          {t("showWeighting")}
        </Check>
        <Check checked={form.motion} onChange={(value) => set("motion", value)}>
          {t("motion")}
        </Check>
      </Section>

      <Section id="profile" title={t("sectionProfile")} open>
        <Field label={t("fieldName")}>
          <input data-testid="profile-name" value={form.name} onChange={(event) => set("name", event.target.value)} />
        </Field>
        <Field label={t("tagline")}>
          <input value={form.tagline} onChange={(event) => set("tagline", event.target.value)} />
        </Field>
        <Field label={t("location")}>
          <input value={form.location} onChange={(event) => set("location", event.target.value)} />
        </Field>
        <Field label={t("note")}>
          <input value={form.note} onChange={(event) => set("note", event.target.value)} />
        </Field>
        <FocusEditor form={form} options={options} onChange={onChange} />
      </Section>

      <Section id="links" title={t("sectionLinks")} open>
        <Field label={t("defaultStyle")}>
          <Select value={form.linkStyle} values={["", ...options.link_styles]} names={{ "": t("defaultOf", { value: "full" }) }} onChange={(value) => set("linkStyle", value)} />
        </Field>
        {form.links.map((entry, index) => (
          <div className="gs-linkrow" key={index} data-testid="link-row" data-selected={selected === linkItem(entry.kind)}>
            <select
              aria-label={t("linkKindN", { n: index + 1 })}
              value={entry.kind}
              onChange={(event) => {
                const kind = event.target.value;
                onChange({
                  ...form,
                  links: form.links.map((existing, at) => (at === index ? { ...existing, kind } : existing)),
                  rows: renameItem(form.rows, linkItem(entry.kind), linkItem(kind)),
                });
                if (selected === linkItem(entry.kind)) onSelect(linkItem(kind));
              }}
            >
              {kinds.map((kind) => (
                <option key={kind} value={kind} disabled={kind !== entry.kind && form.links.some((existing) => existing.kind === kind)}>
                  {kind}
                </option>
              ))}
            </select>
            <button
              type="button"
              className="gs-button"
              aria-label={t("editLinkOf", { kind: entry.kind })}
              aria-pressed={selected === linkItem(entry.kind)}
              onClick={() => onSelect(linkItem(entry.kind))}
            >
              {t("edit")}
            </button>
            <button
              type="button"
              className="gs-button gs-icon-button"
              aria-label={t("removeLinkN", { n: index + 1 })}
              onClick={() => {
                onChange({ ...form, links: form.links.filter((_, at) => at !== index), rows: dropItem(form.rows, linkItem(entry.kind)) });
                if (selected === linkItem(entry.kind)) onSelect(null);
              }}
            >
              ×
            </button>
            <label className="gs-field gs-span">
              <span>{t("handleOrUrl")}</span>
              <input
                aria-label={t("linkValueN", { n: index + 1 })}
                data-testid="link-row-value"
                value={entry.value}
                autoFocus={index === fresh}
                placeholder={entry.kind === CUSTOM ? "https://example.com" : t("handlePlaceholder")}
                spellCheck={false}
                onChange={(event) => onChange({ ...form, links: form.links.map((existing, at) => (at === index ? { ...existing, value: event.target.value } : existing)) })}
              />
            </label>
            {linkReady(entry) ? null : <small className="gs-hint gs-span">{t("emptyLink")}</small>}
          </div>
        ))}
        <button type="button" className="gs-button" data-testid="add-link" onClick={addLink}>
          {t("addLink")}
        </button>
      </Section>
    </div>
  );
}
