import { BASE_ATTRIBUTE, normalizeBase, withBase } from "@standarx/mdocs/base";
import { t } from "./i18n";

export type Mode = "dark" | "light";

export interface RenderResult {
  cards: Record<string, string>;
  snippet: string;
  yaml: string;
  error: string;
}

export interface Options {
  blocks: { name: string; variants: string[] }[];
  presets: string[];
  accents: string[];
  palettes: string[];
  layouts: string[];
  link_kinds: { kind: string; url_only: boolean; map_only: boolean }[];
  link_styles: string[];
  limits: { focus: number; building: number };
}

export interface Ghstats {
  render(yaml: string, mode: Mode, report?: string): RenderResult;
  renderStill(yaml: string, mode: Mode, report?: string): RenderResult;
  defaults(): string;
  options(): Options;
}

interface RawGhstats {
  render(yaml: string, mode: string, report?: string, still?: boolean): RenderResult;
  defaults(): string;
  options(): string;
}

declare class Go {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

type Scope = typeof globalThis & { Go?: typeof Go; ghstats?: RawGhstats };

let loading: Promise<Ghstats> | undefined;

export function baseUrl(path: string): string {
  const base = normalizeBase(document.documentElement.getAttribute(BASE_ATTRIBUTE) ?? "/");
  return withBase(base, path);
}

export function docsHome(): string {
  return baseUrl(`/${document.documentElement.lang || "en"}/`);
}

function injectScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = src;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error(t("scriptFailed", { src })));
    document.head.append(script);
  });
}

async function instantiate(go: Go, url: string): Promise<WebAssembly.Instance> {
  if (typeof WebAssembly.instantiateStreaming === "function") {
    try {
      const streamed = await WebAssembly.instantiateStreaming(fetch(url), go.importObject);
      return streamed.instance;
    } catch {
      // falls through to the buffered path, for a host that serves the wrong MIME type
    }
  }
  const response = await fetch(url);
  if (!response.ok) throw new Error(t("fetchFailed", { url, status: response.status }));
  const buffered = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  return buffered.instance;
}

async function boot(): Promise<Ghstats> {
  const scope = globalThis as Scope;
  if (scope.Go === undefined) await injectScript(baseUrl("/wasm_exec.js"));
  if (scope.Go === undefined) throw new Error(t("goMissing"));
  const go = new scope.Go();
  const instance = await instantiate(go, baseUrl("/ghstats.wasm"));
  void go.run(instance);
  const raw = scope.ghstats;
  if (raw === undefined) throw new Error(t("rendererMissing"));
  const options = JSON.parse(raw.options()) as Options;
  return {
    render: (yaml, mode, report) => (report === undefined ? raw.render(yaml, mode) : raw.render(yaml, mode, report)),
    renderStill: (yaml, mode, report) => raw.render(yaml, mode, report ?? "", true),
    defaults: () => raw.defaults(),
    options: () => options,
  };
}

export function loadGhstats(): Promise<Ghstats> {
  loading ??= boot().catch((error: unknown) => {
    loading = undefined;
    throw error;
  });
  return loading;
}

export function svgUrl(svg: string): string {
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}
