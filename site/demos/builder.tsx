import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  CUSTOM,
  MAX_CARDS,
  cardNames,
  fromYaml,
  isSplit,
  newCard,
  toConfig,
  type BlockState,
  type CardState,
  type FocusState,
  type FormState,
  type LinkState,
} from "./form";
import { loadGhstats, svgUrl, type Ghstats, type Mode, type Options, type RenderResult } from "./ghstats";
import { WORKFLOW } from "./workflow";

interface Ready {
  api: Ghstats;
  options: Options;
  initial: FormState;
}

function download(name: string, content: string): void {
  const url = URL.createObjectURL(new Blob([content], { type: "text/plain" }));
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = name;
  anchor.click();
  URL.revokeObjectURL(url);
}

function CopyButton({ text }: { text: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      className="gs-button"
      onClick={() => {
        void navigator.clipboard.writeText(text).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        });
      }}
    >
      {done ? "Copied" : "Copy"}
    </button>
  );
}

function Output({ name, text, downloadable }: { name: string; text: string; downloadable?: boolean }) {
  return (
    <section className="gs-output">
      <header>
        <code>{name}</code>
        <span>
          <CopyButton text={text} />
          {downloadable === true ? (
            <button type="button" className="gs-button" onClick={() => download(name, text)}>
              Download
            </button>
          ) : null}
        </span>
      </header>
      <pre>{text}</pre>
    </section>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="gs-field">
      <span>{label}</span>
      {children}
    </label>
  );
}

function Select({ value, values, onChange }: { value: string; values: readonly string[]; onChange: (value: string) => void }) {
  return (
    <select value={value} onChange={(event) => onChange(event.target.value)}>
      {values.map((entry) => (
        <option key={entry} value={entry}>
          {entry === "" ? "default" : entry}
        </option>
      ))}
    </select>
  );
}

function Preview({ result, mode, order }: { result: RenderResult; mode: Mode; order: string[] }) {
  const names = Object.keys(result.cards);
  const identity = names.filter((name) => name.startsWith("identity-"));
  const links = names.filter((name) => name.startsWith("links/"));
  const rest = names.filter((name) => !name.startsWith("identity-") && !name.startsWith("links/"));
  const ordered = order.map((card) => `${card}-${mode}.svg`).filter((name) => rest.includes(name));
  const stats = [...ordered, ...rest.filter((name) => !ordered.includes(name)).sort()];
  const image = (name: string) => (
    <img key={name} alt={name} src={svgUrl(result.cards[name] ?? "")} className="gs-card" />
  );
  return (
    <div className="gs-stage" data-mode={mode}>
      {identity.map(image)}
      {stats.map(image)}
      {links.length > 0 ? <div className="gs-badges">{links.map(image)}</div> : null}
    </div>
  );
}

function BlocksEditor({ blocks, options, onChange }: { blocks: BlockState[]; options: Options; onChange: (next: BlockState[]) => void }) {
  const updateBlock = (index: number, variants: string[]) =>
    onChange(blocks.map((block, at) => (at === index ? { ...block, variants } : block)));
  const moveBlock = (index: number, by: number) => {
    const target = index + by;
    if (target < 0 || target >= blocks.length) return;
    const next = [...blocks];
    const [moved] = next.splice(index, 1);
    if (moved !== undefined) next.splice(target, 0, moved);
    onChange(next);
  };
  return (
    <>
      {blocks.map((block, index) => {
        const known = options.blocks.find((entry) => entry.name === block.name);
        return (
          <div className="gs-block" key={block.name}>
            <div className="gs-block-head">
              <strong>{block.name}</strong>
              <span>
                <button type="button" className="gs-button" aria-label={`Move ${block.name} up`} disabled={index === 0} onClick={() => moveBlock(index, -1)}>
                  Up
                </button>
                <button type="button" className="gs-button" aria-label={`Move ${block.name} down`} disabled={index === blocks.length - 1} onClick={() => moveBlock(index, 1)}>
                  Down
                </button>
              </span>
            </div>
            <div className="gs-variants">
              {(known?.variants ?? []).map((variant) => (
                <label className="gs-check" key={variant}>
                  <input
                    type="checkbox"
                    checked={block.variants.includes(variant)}
                    onChange={(event) =>
                      updateBlock(
                        index,
                        (known?.variants ?? []).filter((entry) => (entry === variant ? event.target.checked : block.variants.includes(entry))),
                      )
                    }
                  />
                  <span>{variant}</span>
                </label>
              ))}
            </div>
          </div>
        );
      })}
    </>
  );
}

function CardsEditor({ cards, options, onChange }: { cards: CardState[]; options: Options; onChange: (next: CardState[]) => void }) {
  const update = (index: number, patch: Partial<CardState>) =>
    onChange(cards.map((card, at) => (at === index ? { ...card, ...patch } : card)));
  const unnamed = (list: CardState[]) => list.map((card) => ({ ...card, name: "", layout: "" }));
  const remove = (index: number) => {
    const next = cards.filter((_, at) => at !== index);
    onChange(next.length <= 1 ? unnamed(next.length === 0 ? cards : next) : next);
  };
  const add = () => {
    const taken = new Set(cards.map((card) => card.name));
    let at = cards.length + 1;
    while (taken.has(`card-${at}`)) at += 1;
    onChange([...cards, newCard(`card-${at}`, options)]);
  };
  const first = cards[0];
  if (!isSplit(cards)) {
    return (
      <fieldset>
        <legend>Blocks</legend>
        <BlocksEditor blocks={first?.blocks ?? []} options={options} onChange={(blocks) => update(0, { blocks })} />
        <button
          type="button"
          className="gs-button"
          onClick={() => onChange(first === undefined ? [newCard("overview", options)] : [{ ...first, name: "overview" }, newCard("card-2", options)])}
        >
          Split into cards
        </button>
      </fieldset>
    );
  }
  return (
    <>
      {cards.map((card, index) => (
        <fieldset key={index}>
          <legend>Card {index + 1}</legend>
          <div className="gs-row">
            <input aria-label={`Card ${index + 1} name`} placeholder="Name" value={card.name} onChange={(event) => update(index, { name: event.target.value })} />
            <select aria-label={`Card ${index + 1} layout`} value={card.layout} onChange={(event) => update(index, { layout: event.target.value })}>
              {["", ...options.layouts].map((entry) => (
                <option key={entry} value={entry}>
                  {entry === "" ? "default" : entry}
                </option>
              ))}
            </select>
            <button type="button" className="gs-button" onClick={() => remove(index)}>
              Remove
            </button>
          </div>
          <BlocksEditor blocks={card.blocks} options={options} onChange={(blocks) => update(index, { blocks })} />
        </fieldset>
      ))}
      <button type="button" className="gs-button" disabled={cards.length >= MAX_CARDS} onClick={add}>
        Add card ({cards.length}/{MAX_CARDS})
      </button>
    </>
  );
}

function BuilderForm({ form, options, onChange }: { form: FormState; options: Options; onChange: (next: FormState) => void }) {
  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => onChange({ ...form, [key]: value });
  const kinds = options.link_kinds.map((entry) => entry.kind);
  const updateFocus = (index: number, patch: Partial<FocusState>) =>
    set(
      "focus",
      form.focus.map((entry, at) => (at === index ? { ...entry, ...patch } : entry)),
    );
  const updateLink = (index: number, patch: Partial<LinkState>) =>
    set(
      "links",
      form.links.map((entry, at) => (at === index ? { ...entry, ...patch } : entry)),
    );

  return (
    <div className="gs-form">
      <fieldset>
        <legend>Theme</legend>
        <Field label="Preset">
          <Select value={form.preset} values={options.presets} onChange={(value) => set("preset", value)} />
        </Field>
        <Field label="Accent">
          <Select value={form.accent} values={[...options.accents, CUSTOM]} onChange={(value) => set("accent", value)} />
        </Field>
        {form.accent === CUSTOM ? (
          <div className="gs-pair">
            <Field label="Accent, dark">
              <input type="color" value={form.accentDark} onChange={(event) => set("accentDark", event.target.value)} />
            </Field>
            <Field label="Accent, light">
              <input type="color" value={form.accentLight} onChange={(event) => set("accentLight", event.target.value)} />
            </Field>
          </div>
        ) : null}
        <Field label="Language colors">
          <Select value={form.palette} values={[...options.palettes, CUSTOM]} onChange={(value) => set("palette", value)} />
        </Field>
        {form.palette === CUSTOM ? (
          <Field label="One language per line, Name: #rrggbb">
            <textarea rows={4} value={form.paletteColors} onChange={(event) => set("paletteColors", event.target.value)} spellCheck={false} />
          </Field>
        ) : null}
        <Field label="Layout">
          <Select value={form.layout} values={options.layouts} onChange={(value) => set("layout", value)} />
        </Field>
        <label className="gs-check">
          <input type="checkbox" checked={form.motion} onChange={(event) => set("motion", event.target.checked)} />
          <span>Motion</span>
        </label>
      </fieldset>

      <CardsEditor cards={form.cards} options={options} onChange={(cards) => set("cards", cards)} />

      <fieldset>
        <legend>Profile</legend>
        <Field label="Name">
          <input value={form.name} onChange={(event) => set("name", event.target.value)} />
        </Field>
        <Field label="Tagline">
          <input value={form.tagline} onChange={(event) => set("tagline", event.target.value)} />
        </Field>
        <Field label="Location">
          <input value={form.location} onChange={(event) => set("location", event.target.value)} />
        </Field>
        <Field label="Note">
          <input value={form.note} onChange={(event) => set("note", event.target.value)} />
        </Field>
      </fieldset>

      <fieldset>
        <legend>Focus</legend>
        {form.focus.map((entry, index) => (
          <div className="gs-row" key={index}>
            <input aria-label={`Focus ${index + 1} name`} placeholder="Name" value={entry.name} onChange={(event) => updateFocus(index, { name: event.target.value })} />
            <input
              aria-label={`Focus ${index + 1} building`}
              placeholder={`Up to ${options.limits.building} words, comma separated`}
              value={entry.building}
              onChange={(event) => updateFocus(index, { building: event.target.value })}
            />
            <button type="button" className="gs-button" onClick={() => set("focus", form.focus.filter((_, at) => at !== index))}>
              Remove
            </button>
          </div>
        ))}
        <button
          type="button"
          className="gs-button"
          disabled={form.focus.length >= options.limits.focus}
          onClick={() => set("focus", [...form.focus, { name: "", building: "" }])}
        >
          Add a focus ({form.focus.length}/{options.limits.focus})
        </button>
      </fieldset>

      <fieldset>
        <legend>Links</legend>
        <Field label="Link style">
          <Select value={form.linkStyle} values={["", ...options.link_styles]} onChange={(value) => set("linkStyle", value)} />
        </Field>
        {form.links.map((link, index) => (
          <div className="gs-row" key={index}>
            <Select value={link.kind} values={kinds} onChange={(value) => updateLink(index, { kind: value })} />
            <input
              aria-label={`Link ${index + 1} ${link.kind === CUSTOM ? "url" : "value"}`}
              placeholder={link.kind === CUSTOM ? "https://example.com" : "handle or url"}
              value={link.value}
              onChange={(event) => updateLink(index, { value: event.target.value })}
            />
            {link.kind === CUSTOM ? (
              <>
                <input aria-label={`Link ${index + 1} label`} placeholder="label" value={link.label} onChange={(event) => updateLink(index, { label: event.target.value })} />
                <input aria-label={`Link ${index + 1} icon`} placeholder="icon kind or path" value={link.icon} onChange={(event) => updateLink(index, { icon: event.target.value })} />
              </>
            ) : null}
            <button type="button" className="gs-button" onClick={() => set("links", form.links.filter((_, at) => at !== index))}>
              Remove
            </button>
          </div>
        ))}
        <button type="button" className="gs-button" onClick={() => set("links", [...form.links, { kind: kinds[0] ?? "", value: "", label: "", icon: "" }])}>
          Add a link
        </button>
      </fieldset>
    </div>
  );
}

function BuilderApp({ api, options, initial }: Ready) {
  const [form, setForm] = useState(initial);
  const [override, setOverride] = useState<string | null>(null);
  const [mode, setMode] = useState<Mode>("dark");

  const formSource = useMemo(() => JSON.stringify(toConfig(form)), [form]);
  const formResult = useMemo(() => api.render(formSource, mode), [api, formSource, mode]);
  const result = useMemo(() => (override === null ? formResult : api.render(override, mode)), [api, override, formResult, mode]);

  const lastGood = useRef<RenderResult>(result);
  if (result.error === "") lastGood.current = result;
  const shown = result.error === "" ? result : lastGood.current;

  const lastYaml = useRef("");
  if (formResult.error === "") lastYaml.current = formResult.yaml;
  const yamlText = override ?? lastYaml.current;

  const editYaml = (value: string) => setOverride(value === lastYaml.current ? null : value);

  return (
    <div className="gs-builder">
      <div className="gs-pane gs-pane-form" data-locked={override !== null}>
        {override !== null ? (
          <div className="gs-lock">
            <span>The YAML was edited by hand: the form is off.</span>
            <button type="button" className="gs-button" onClick={() => setOverride(null)}>
              Back to the form
            </button>
          </div>
        ) : null}
        <fieldset disabled={override !== null} className="gs-reset">
          <BuilderForm form={form} options={options} onChange={setForm} />
        </fieldset>
      </div>

      <div className="gs-pane gs-pane-preview">
        <div className="gs-toolbar">
          <span>Preview</span>
          <span role="group" aria-label="Preview mode">
            <button type="button" className="gs-button" aria-pressed={mode === "dark"} onClick={() => setMode("dark")}>
              Dark
            </button>
            <button type="button" className="gs-button" aria-pressed={mode === "light"} onClick={() => setMode("light")}>
              Light
            </button>
          </span>
        </div>
        {result.error !== "" ? (
          <pre className="gs-error" role="alert">
            {result.error}
          </pre>
        ) : null}
        <Preview result={shown} mode={mode} order={override === null ? cardNames(form.cards) : []} />
      </div>

      <div className="gs-pane gs-outputs">
        <section className="gs-output">
          <header>
            <code>ghstats.yaml</code>
            <span>
              <CopyButton text={yamlText} />
              <button type="button" className="gs-button" onClick={() => download("ghstats.yaml", yamlText)}>
                Download
              </button>
            </span>
          </header>
          <textarea className="gs-yaml" spellCheck={false} rows={18} value={yamlText} onChange={(event) => editYaml(event.target.value)} aria-label="ghstats.yaml" />
        </section>
        <Output name=".github/workflows/ghstats.yml" text={WORKFLOW} />
        <Output name="README.snippet.md" text={shown.snippet} />
      </div>
    </div>
  );
}

export function Builder() {
  const [ready, setReady] = useState<Ready | null>(null);
  const [failure, setFailure] = useState("");

  useEffect(() => {
    let live = true;
    loadGhstats()
      .then((api) => {
        if (!live) return;
        const options = api.options();
        setReady({ api, options, initial: fromYaml(api.defaults(), options) });
      })
      .catch((error: unknown) => {
        if (live) setFailure(error instanceof Error ? error.message : String(error));
      });
    return () => {
      live = false;
    };
  }, []);

  if (failure !== "") {
    return (
      <p className="gs-loading" role="alert">
        The renderer failed to load: {failure}
      </p>
    );
  }
  if (ready === null) return <p className="gs-loading">Loading the renderer…</p>;
  return <BuilderApp {...ready} />;
}
