import type { Ghstats, Mode, RenderResult } from "./ghstats";
import { svgToPng } from "./png";
import { zip } from "./zip";

export type Format = "svg" | "png";

export interface Source {
  api: Ghstats;
  config: string;
  report: string | undefined;
}

export interface Exported {
  path: string;
  blob: Blob;
}

export const PNG_SCALE = 2;

function cardsOf(result: RenderResult): Record<string, string> {
  if (result.error !== "") throw new Error(result.error);
  return result.cards;
}

export async function exportItems(source: Source, items: string[], formats: Format[], modes: Mode[]): Promise<Exported[]> {
  const files: Exported[] = [];
  for (const mode of modes) {
    const live = formats.includes("svg") ? cardsOf(source.api.render(source.config, mode, source.report)) : {};
    const still = formats.includes("png") ? cardsOf(source.api.renderStill(source.config, mode, source.report)) : {};
    for (const item of items) {
      const key = `${item}-${mode}.svg`;
      if (formats.includes("svg")) {
        const svg = live[key];
        if (svg === undefined) throw new Error(`no card named ${key}`);
        files.push({ path: key, blob: new Blob([svg], { type: "image/svg+xml" }) });
      }
      if (formats.includes("png")) {
        const svg = still[key];
        if (svg === undefined) throw new Error(`no card named ${key}`);
        files.push({ path: key.replace(/\.svg$/, ".png"), blob: await svgToPng(svg, PNG_SCALE) });
      }
    }
  }
  return files;
}

export async function archive(files: Exported[]): Promise<Blob> {
  const entries = await Promise.all(files.map(async (file) => ({ name: file.path, data: new Uint8Array(await file.blob.arrayBuffer()) })));
  return zip(entries);
}

export function saveBlob(name: string, blob: Blob): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = name;
  anchor.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function flatName(path: string): string {
  return path.replace(/\//g, "-");
}
