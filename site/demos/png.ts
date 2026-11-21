import { baseUrl } from "./ghstats";
import { t } from "./i18n";

interface Resvg {
  memory: WebAssembly.Memory;
  alloc(len: number): number;
  dealloc(ptr: number, len: number): void;
  render(ptr: number, len: number, scale: number): number;
  resultPtr(): number;
  resultLen(): number;
}

let loading: Promise<Resvg> | undefined;

async function instantiate(url: string): Promise<WebAssembly.Instance> {
  if (typeof WebAssembly.instantiateStreaming === "function") {
    try {
      const streamed = await WebAssembly.instantiateStreaming(fetch(url), {});
      return streamed.instance;
    } catch {
      // falls through to the buffered path, for a host that serves the wrong MIME type
    }
  }
  const response = await fetch(url);
  if (!response.ok) throw new Error(t("fetchFailed", { url, status: response.status }));
  const buffered = await WebAssembly.instantiate(await response.arrayBuffer(), {});
  return buffered.instance;
}

async function boot(): Promise<Resvg> {
  const instance = await instantiate(baseUrl("/resvg.wasm"));
  const exports = instance.exports as Record<string, unknown>;
  const memory = exports.memory as WebAssembly.Memory | undefined;
  const alloc = exports.alloc as Resvg["alloc"] | undefined;
  const dealloc = exports.dealloc as Resvg["dealloc"] | undefined;
  const render = exports.render as Resvg["render"] | undefined;
  const resultPtr = exports.result_ptr as Resvg["resultPtr"] | undefined;
  const resultLen = exports.result_len as Resvg["resultLen"] | undefined;
  if (!memory || !alloc || !dealloc || !render || !resultPtr || !resultLen) throw new Error(t("resvgExports"));
  return { memory, alloc, dealloc, render, resultPtr, resultLen };
}

function load(): Promise<Resvg> {
  loading ??= boot().catch((error: unknown) => {
    loading = undefined;
    throw error;
  });
  return loading;
}

export async function svgToPng(svg: string, scale = 2): Promise<Blob> {
  const engine = await load();
  const input = new TextEncoder().encode(svg);
  if (input.length === 0) throw new Error(t("pngEmpty"));
  const ptr = engine.alloc(input.length);
  try {
    new Uint8Array(engine.memory.buffer, ptr, input.length).set(input);
    const code = engine.render(ptr, input.length, scale);
    const out = new Uint8Array(engine.memory.buffer, engine.resultPtr(), engine.resultLen()).slice();
    if (code !== 0) throw new Error(t("pngFailed", { message: new TextDecoder().decode(out) }));
    return new Blob([out], { type: "image/png" });
  } finally {
    engine.dealloc(ptr, input.length);
  }
}
