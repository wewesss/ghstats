import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Canvas, itemLabel } from "./canvas";
import { DEFAULT_PATH, SAMPLE, loadFromFile, loadFromGithub, type DataSource } from "./data";
import { fromYaml, linkItem, toConfig, type FormState, type RowState } from "./form";
import { docsHome, loadGhstats, type Ghstats, type Mode, type Options, type RenderResult } from "./ghstats";
import { parseCanvas, step, toRows, type Direction } from "./layout";
import { SettingsPanel } from "./panel";
import { WORKFLOW } from "./workflow";

interface Ready {
  api: Ghstats;
  options: Options;
  initial: FormState;
}

type Tab = "preview" | "yaml" | "workflow" | "snippet";
type Pane = "settings" | "preview" | "files";
type Theme = "" | "light" | "dark";

const TABS: { id: Tab; label: string }[] = [
  { id: "preview", label: "Preview" },
  { id: "yaml", label: "ghstats.yaml" },
  { id: "workflow", label: "Workflow" },
  { id: "snippet", label: "Snippet" },
];

const FILES: { id: Exclude<Tab, "preview">; label: string }[] = [
  { id: "yaml", label: "yaml" },
  { id: "workflow", label: "workflow" },
  { id: "snippet", label: "snippet" },
];

const THEMES: { value: Theme; label: string }[] = [
  { value: "", label: "Auto" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

const ARROWS: { direction: Direction; glyph: string; label: string }[] = [
  { direction: "left", glyph: "←", label: "Move left" },
  { direction: "right", glyph: "→", label: "Move right" },
  { direction: "up", glyph: "↑", label: "Move to the row above" },
  { direction: "down", glyph: "↓", label: "Move to the row below" },
];

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

function useFrame(): { left: number; top: number } {
  const [frame, setFrame] = useState({ left: 0, top: 0 });
  useLayoutEffect(() => {
    const measure = () => {
      const sidebar = document.querySelector<HTMLElement>(".mira-docs > .mira-docs__sidebar");
      const bar = document.querySelector<HTMLElement>(".mira-docs > .mira-docs__bar");
      let left = 0;
      let top = 0;
      if (sidebar !== null) {
        const style = getComputedStyle(sidebar);
        if (style.display !== "none" && style.position === "sticky") left = Math.round(sidebar.getBoundingClientRect().right);
      }
      if (bar !== null && getComputedStyle(bar).display !== "none") top = Math.max(0, Math.round(bar.getBoundingClientRect().bottom));
      setFrame((previous) => (previous.left === left && previous.top === top ? previous : { left, top }));
    };
    measure();
    window.addEventListener("resize", measure);
    const watch = new ResizeObserver(measure);
    watch.observe(document.documentElement);
    const mutations = new MutationObserver(measure);
    mutations.observe(document.documentElement, { attributes: true });
    const bar = document.querySelector(".mira-docs > .mira-docs__bar");
    if (bar !== null) watch.observe(bar);
    return () => {
      window.removeEventListener("resize", measure);
      watch.disconnect();
      mutations.disconnect();
    };
  }, []);
  return frame;
}

interface Language {
  href: string;
  code: string;
  label: string;
  current: boolean;
}

function readTheme(): Theme {
  const value = document.documentElement.getAttribute("data-mira-theme");
  return value === "light" || value === "dark" ? value : "";
}

function readLanguages(): Language[] {
  return Array.from(document.querySelectorAll<HTMLAnchorElement>(".mira-docs__languages a")).map((link) => {
    const code = link.getAttribute("lang") ?? link.textContent ?? "";
    return { href: link.getAttribute("href") ?? "", code, label: code.toUpperCase(), current: link.getAttribute("aria-current") === "page" };
  });
}

function SiteControls() {
  const [theme, setTheme] = useState<Theme>("");
  const [languages, setLanguages] = useState<Language[]>([]);

  useEffect(() => {
    setTheme(readTheme());
    setLanguages(readLanguages());
    const watch = new MutationObserver(() => setTheme(readTheme()));
    watch.observe(document.documentElement, { attributes: true, attributeFilter: ["data-mira-theme"] });
    return () => watch.disconnect();
  }, []);

  return (
    <div className="gs-site" data-testid="site-controls">
      <a className="gs-docslink" href={docsHome()} data-testid="docs-link">
        ← Docs
      </a>
      <span role="group" aria-label="Theme" className="gs-segment gs-compact">
        {THEMES.map((entry) => (
          <button
            key={entry.value}
            type="button"
            className="gs-button"
            data-mira-axis="theme"
            data-mira-axis-value={entry.value}
            data-testid={`theme-${entry.value === "" ? "auto" : entry.value}`}
            aria-pressed={theme === entry.value}
          >
            {entry.label}
          </button>
        ))}
      </span>
      {languages.length > 1 ? (
        <span role="group" aria-label="Language" className="gs-segment gs-compact">
          {languages.map((entry) => (
            <a
              key={entry.code}
              className="gs-button"
              href={entry.href}
              lang={entry.code}
              hrefLang={entry.code}
              data-testid={`lang-${entry.code}`}
              {...(entry.current ? { "aria-current": "page" as const } : {})}
            >
              {entry.label}
            </a>
          ))}
        </span>
      ) : null}
    </div>
  );
}

function DataBar({ source, onChange }: { source: DataSource; onChange: (next: DataSource) => void }) {
  const [kind, setKind] = useState<"sample" | "github" | "file">(source.report === undefined ? "sample" : "github");
  const [login, setLogin] = useState("");
  const [path, setPath] = useState(DEFAULT_PATH);
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState("");

  const run = (load: () => Promise<DataSource>) => {
    setBusy(true);
    setProblem("");
    load()
      .then(onChange)
      .catch((error: unknown) => setProblem(error instanceof Error ? error.message : String(error)))
      .finally(() => setBusy(false));
  };

  return (
    <div className="gs-databar" data-testid="data-bar">
      <label className="gs-inline">
        <span>Data</span>
        <select
          value={kind}
          data-testid="data-kind"
          onChange={(event) => {
            const next = event.target.value as "sample" | "github" | "file";
            setKind(next);
            setProblem("");
            if (next === "sample") onChange(SAMPLE);
          }}
        >
          <option value="sample">Sample data</option>
          <option value="github">GitHub login</option>
          <option value="file">stats.json file</option>
        </select>
      </label>
      {kind === "github" ? (
        <>
          <label className="gs-inline">
            <span>Login</span>
            <input data-testid="data-login" value={login} onChange={(event) => setLogin(event.target.value.trim())} spellCheck={false} autoCapitalize="off" />
          </label>
          <label className="gs-inline">
            <span>Path</span>
            <input data-testid="data-path" value={path} onChange={(event) => setPath(event.target.value.trim())} spellCheck={false} />
          </label>
          <button type="button" className="gs-button" data-testid="data-load" disabled={busy || login === ""} onClick={() => run(() => loadFromGithub(login, path))}>
            Load
          </button>
        </>
      ) : null}
      {kind === "file" ? (
        <label className="gs-inline">
          <span>File</span>
          <input
            type="file"
            data-testid="data-file"
            accept=".json,application/json"
            onChange={(event) => {
              const file = event.target.files?.[0];
              event.target.value = "";
              if (file !== undefined) run(() => loadFromFile(file));
            }}
          />
        </label>
      ) : null}
      {problem !== "" ? (
        <pre className="gs-error gs-span" role="alert">
          {problem}
        </pre>
      ) : null}
    </div>
  );
}

function OutputPane({ name, hint, text, downloadable, children }: { name: string; hint: ReactNode; text: string; downloadable?: boolean; children: ReactNode }) {
  return (
    <section className="gs-output" data-testid={`output-${name}`}>
      <header>
        <code>{name}</code>
        <span className="gs-actions">
          <CopyButton text={text} />
          {downloadable === true ? (
            <button type="button" className="gs-button" onClick={() => download(name, text)}>
              Download
            </button>
          ) : null}
        </span>
      </header>
      <p className="gs-hint">{hint}</p>
      {children}
    </section>
  );
}

function BuilderApp({ api, options, initial }: Ready) {
  const [form, setForm] = useState(initial);
  const [override, setOverride] = useState<string | null>(null);
  const [mode, setMode] = useState<Mode>("dark");
  const [data, setData] = useState<DataSource>(SAMPLE);
  const [tab, setTab] = useState<Tab>("preview");
  const [zoom, setZoom] = useState<"fit" | "100">("fit");
  const [pane, setPane] = useState<Pane>("preview");
  const [selected, setSelected] = useState<string | null>(null);
  const settings = useRef<HTMLDivElement>(null);
  const frame = useFrame();

  useEffect(() => {
    document.documentElement.setAttribute("data-gs-builder", "");
    return () => document.documentElement.removeAttribute("data-gs-builder");
  }, []);

  const formSource = useMemo(() => JSON.stringify(toConfig(form)), [form]);
  const formResult = useMemo(() => api.render(formSource, mode, data.report), [api, formSource, mode, data]);
  const result = useMemo(
    () => (override === null ? formResult : api.render(override, mode, data.report)),
    [api, override, formResult, mode, data],
  );

  const lastGood = useRef<RenderResult>(result);
  if (result.error === "") lastGood.current = result;
  const shown = result.error === "" ? result : lastGood.current;

  const lastYaml = useRef("");
  if (formResult.error === "") lastYaml.current = formResult.yaml;
  const yamlText = override ?? lastYaml.current;

  const canvasRows = useMemo(() => parseCanvas(shown.snippet, mode, shown.cards), [shown, mode]);
  const blank = form.rows.length === 0;
  const rows = useMemo(() => {
    const base = toRows(canvasRows);
    return blank ? base.map((row) => ({ ...row, align: "center" })) : base;
  }, [canvasRows, blank]);
  const onCanvas = useMemo(() => rows.flatMap((row) => row.items), [rows]);
  const editable = override === null && result.error === "";

  const exists =
    selected !== null &&
    (form.cards.some((card) => card.name === selected) || form.links.some((link) => linkItem(link.kind) === selected) || onCanvas.includes(selected));
  const current = exists ? selected : null;

  useEffect(() => {
    if (selected !== null && !exists) setSelected(null);
  }, [selected, exists]);

  useEffect(() => {
    if (current !== null) settings.current?.scrollTo({ top: 0 });
  }, [current]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      const target = event.target as HTMLElement | null;
      if (target !== null && (target.tagName === "TEXTAREA" || target.tagName === "INPUT")) return;
      setSelected(null);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  const arrange = (next: RowState[]) => setForm((previous) => ({ ...previous, rows: next }));
  const moves = ARROWS.map((entry) => ({ ...entry, to: current === null ? null : step(rows, current, entry.direction) }));

  const choose = (next: Tab) => {
    setTab(next);
    if (pane !== "settings") setPane(next === "preview" ? "preview" : "files");
  };

  const editYaml = (value: string) => setOverride(value === lastYaml.current ? null : value);

  return createPortal(
    <div className="gs-frame" style={{ left: frame.left, top: frame.top }}>
    <div className="gs-app" data-testid="builder" data-pane={pane} data-mode={mode}>
      <div className="gs-mobile-tabs" role="tablist" aria-label="Builder panes">
        <button type="button" role="tab" data-testid="pane-settings" aria-selected={pane === "settings"} onClick={() => setPane("settings")}>
          Settings
        </button>
        <button
          type="button"
          role="tab"
          data-testid="pane-preview"
          aria-selected={pane === "preview"}
          onClick={() => {
            setPane("preview");
            setTab("preview");
          }}
        >
          Preview
        </button>
        <button
          type="button"
          role="tab"
          data-testid="pane-files"
          aria-selected={pane === "files"}
          onClick={() => {
            setPane("files");
            if (tab === "preview") setTab("yaml");
          }}
        >
          Files
        </button>
      </div>

      <aside ref={settings} className="gs-settings" data-testid="settings" aria-label="Settings">
        <SiteControls />
        {override !== null ? (
          <div className="gs-lock">
            <span>YAML edited by hand: the form is off.</span>
            <button type="button" className="gs-button" onClick={() => setOverride(null)}>
              Back to the form
            </button>
          </div>
        ) : null}
        <fieldset disabled={override !== null} className="gs-reset" data-locked={override !== null}>
          <SettingsPanel
            form={form}
            options={options}
            rows={rows}
            onCanvas={onCanvas}
            selected={current}
            onSelect={setSelected}
            onChange={setForm}
          />
        </fieldset>
      </aside>

      <div className="gs-workspace" data-testid="workspace">
        <div className="gs-bar">
          <DataBar source={data} onChange={setData} />
          <div className="gs-view">
            <span role="group" aria-label="Preview mode" className="gs-segment">
              <button type="button" className="gs-button" data-testid="mode-light" aria-pressed={mode === "light"} onClick={() => setMode("light")}>
                Light
              </button>
              <button type="button" className="gs-button" data-testid="mode-dark" aria-pressed={mode === "dark"} onClick={() => setMode("dark")}>
                Dark
              </button>
            </span>
            <span role="group" aria-label="Zoom" className="gs-segment">
              <button type="button" className="gs-button" data-testid="zoom-fit" aria-pressed={zoom === "fit"} onClick={() => setZoom("fit")}>
                Fit
              </button>
              <button type="button" className="gs-button" data-testid="zoom-100" aria-pressed={zoom === "100"} onClick={() => setZoom("100")}>
                100%
              </button>
            </span>
          </div>
        </div>

        <div className="gs-tabs" role="tablist" aria-label="Outputs">
          {TABS.map((entry) => (
            <button key={entry.id} type="button" role="tab" data-testid={`tab-${entry.id}`} aria-selected={tab === entry.id} onClick={() => choose(entry.id)}>
              {entry.label}
            </button>
          ))}
          <span className="gs-source" data-testid="data-source">
            Showing <strong>{data.label}</strong>
          </span>
        </div>

        {tab !== "preview" ? (
          <div className="gs-filetabs" role="group" aria-label="Files" data-testid="file-tabs">
            <span className="gs-segment">
              {FILES.map((entry) => (
                <button key={entry.id} type="button" className="gs-button" data-testid={`file-${entry.id}`} aria-pressed={tab === entry.id} onClick={() => choose(entry.id)}>
                  {entry.label}
                </button>
              ))}
            </span>
          </div>
        ) : null}

        {tab === "preview" ? (
          <div className="gs-previewpane" data-testid="preview">
            <div className="gs-selbar" data-testid="selection-bar">
              {override !== null ? (
                <span className="gs-hint">The YAML was edited by hand: the canvas only shows it. Use Back to the form to arrange again.</span>
              ) : current === null ? (
                <span className="gs-hint">Click an element to select it. Drag it, or use the arrow keys, to arrange the README.</span>
              ) : (
                <>
                  <span className="gs-selname">
                    Selected <strong data-testid="selected-name">{itemLabel(current)}</strong>
                  </span>
                  <span className="gs-arrows" role="group" aria-label="Move the selection">
                    {moves.map((entry) => (
                      <button
                        key={entry.direction}
                        type="button"
                        className="gs-button gs-icon-button"
                        data-testid={`move-${entry.direction}`}
                        aria-label={entry.label}
                        title={entry.label}
                        disabled={!editable || entry.to === null}
                        onClick={() => {
                          if (entry.to !== null) arrange(entry.to);
                        }}
                      >
                        {entry.glyph}
                      </button>
                    ))}
                  </span>
                  <button type="button" className="gs-button gs-only-mobile" data-testid="edit-selection" onClick={() => setPane("settings")}>
                    Edit settings
                  </button>
                  <button type="button" className="gs-button" onClick={() => setSelected(null)}>
                    Deselect
                  </button>
                </>
              )}
            </div>
            {result.error !== "" ? (
              <pre className="gs-error gs-banner" role="alert" data-testid="error">
                {result.error}
              </pre>
            ) : null}
            <Canvas
              rows={canvasRows}
              state={rows}
              mode={mode}
              selected={current}
              editable={editable}
              zoom={zoom}
              onSelect={setSelected}
              onArrange={(next) => arrange(next)}
            />
          </div>
        ) : null}

        {tab === "yaml" ? (
          <OutputPane
            name="ghstats.yaml"
            text={yamlText}
            downloadable
            hint="Save it at the root of your profile repository: the Action and the CLI read it from there. Editing it here turns the form off."
          >
            <textarea
              className="gs-yaml"
              data-testid="yaml-editor"
              spellCheck={false}
              value={yamlText}
              onChange={(event) => editYaml(event.target.value)}
              aria-label="ghstats.yaml"
            />
          </OutputPane>
        ) : null}

        {tab === "workflow" ? (
          <OutputPane
            name=".github/workflows/ghstats.yml"
            text={WORKFLOW}
            hint="Add it to your profile repository and replace <user> with your login, as described in Getting started."
          >
            <pre data-testid="workflow-text">{WORKFLOW}</pre>
          </OutputPane>
        ) : null}

        {tab === "snippet" ? (
          <OutputPane name="README.snippet.md" text={shown.snippet} hint="Paste it in your profile README. The Action writes the same file after each run.">
            <pre data-testid="snippet-text">{shown.snippet}</pre>
          </OutputPane>
        ) : null}
      </div>
    </div>
    </div>,
    document.body,
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
