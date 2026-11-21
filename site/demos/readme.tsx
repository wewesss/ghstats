import { createPortal } from "react-dom";
import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { svgUrl, type Ghstats, type Mode } from "./ghstats";

const ALLOWED = new Set(["p", "picture", "source", "img", "a"]);
const ALIGNS = new Set(["left", "center", "right"]);
const FOCUSABLE = "button:not(:disabled), a[href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled)";

export interface PreviewEntry {
  badges: boolean;
  keys: string[];
}

function parseSnippet(snippet: string): Document {
  return new DOMParser().parseFromString(snippet, "text/html");
}

export function cardKey(src: string, mode: Mode, keys: string[]): string | undefined {
  const wanted = src.replace(/-(dark|light)\.svg$/, `-${mode}.svg`);
  return [...keys].sort((a, b) => b.length - a.length).find((key) => wanted === key || wanted.endsWith(`/${key}`));
}

export function previewOrder(snippet: string, mode: Mode, keys: string[]): PreviewEntry[] {
  const seen = new Set<string>();
  const ordered: string[] = [];
  for (const pic of parseSnippet(snippet).querySelectorAll("picture")) {
    const key = cardKey(pic.querySelector("img")?.getAttribute("src") ?? "", mode, keys);
    if (key !== undefined && !seen.has(key)) {
      seen.add(key);
      ordered.push(key);
    }
  }
  for (const key of [...keys].sort()) {
    if (!seen.has(key)) ordered.push(key);
  }
  const entries: PreviewEntry[] = [];
  for (const key of ordered) {
    const badge = key.startsWith("links/");
    const last = entries[entries.length - 1];
    if (badge && last?.badges === true) last.keys.push(key);
    else entries.push({ badges: badge, keys: [key] });
  }
  return entries;
}

function imageStyle(width: string | null): CSSProperties {
  const style: CSSProperties = { maxWidth: "100%", height: "auto", display: "inline", boxSizing: "content-box" };
  if (width !== null && /^\d+(\.\d+)?%$/.test(width)) style.width = width;
  else if (width !== null && /^\d+$/.test(width)) style.width = `${width}px`;
  return style;
}

function convert(node: Node, mode: Mode, cards: Record<string, string>, path: string): ReactNode {
  if (node.nodeType === Node.TEXT_NODE) return node.textContent;
  if (!(node instanceof Element)) return null;
  const tag = node.tagName.toLowerCase();
  const children = (from: Element) => Array.from(from.childNodes).map((child, at) => convert(child, mode, cards, `${path}.${at}`));
  if (!ALLOWED.has(tag)) return <span key={path}>{children(node)}</span>;
  switch (tag) {
    case "source":
      return null;
    case "picture":
    case "img": {
      const img = tag === "img" ? node : node.querySelector("img");
      if (img === null) return null;
      const key = cardKey(img.getAttribute("src") ?? "", mode, Object.keys(cards));
      const svg = key === undefined ? undefined : cards[key];
      if (svg === undefined) return null;
      return <img key={path} alt={img.getAttribute("alt") ?? ""} src={svgUrl(svg)} style={imageStyle(img.getAttribute("width"))} />;
    }
    case "p": {
      const align = node.getAttribute("align") ?? "";
      const style: CSSProperties = ALIGNS.has(align) ? { textAlign: align as CSSProperties["textAlign"] } : {};
      return (
        <p key={path} style={style}>
          {children(node)}
        </p>
      );
    }
    default: {
      const href = node.getAttribute("href") ?? "";
      return (
        <a key={path} href={/^https?:\/\//i.test(href) ? href : undefined} onClick={(event) => event.preventDefault()}>
          {children(node)}
        </a>
      );
    }
  }
}

function ReadmeModal({
  api,
  source,
  report,
  initial,
  onClose,
}: {
  api: Ghstats;
  source: string;
  report: string | undefined;
  initial: Mode;
  onClose: () => void;
}) {
  const [mode, setMode] = useState<Mode>(initial);
  const closeButton = useRef<HTMLButtonElement>(null);
  const dialog = useRef<HTMLDivElement>(null);
  const result = useMemo(() => api.render(source, mode, report), [api, source, mode, report]);
  const content = useMemo(() => {
    if (result.error !== "") return null;
    const body = parseSnippet(result.snippet).body;
    return Array.from(body.childNodes).map((node, at) => convert(node, mode, result.cards, String(at)));
  }, [result, mode]);

  useEffect(() => {
    closeButton.current?.focus();
  }, []);

  useEffect(() => {
    const html = document.documentElement;
    const previous = [html.style.overflow, document.body.style.overflow];
    html.style.overflow = "hidden";
    document.body.style.overflow = "hidden";
    return () => {
      html.style.overflow = previous[0] ?? "";
      document.body.style.overflow = previous[1] ?? "";
    };
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key !== "Tab" || dialog.current === null) return;
      const items = Array.from(dialog.current.querySelectorAll<HTMLElement>(FOCUSABLE));
      const first = items[0];
      const last = items[items.length - 1];
      if (first === undefined || last === undefined) return;
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return createPortal(
    <div
      ref={dialog}
      className="gs-modal"
      data-mode={mode}
      role="dialog"
      aria-modal="true"
      aria-label="README preview"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className="gs-modal-bar">
        <span>
          README preview <small>Approximate GitHub rendering</small>
        </span>
        <span>
          <span role="group" aria-label="README preview mode">
            <button type="button" className="gs-button" aria-pressed={mode === "light"} onClick={() => setMode("light")}>
              Light
            </button>
            <button type="button" className="gs-button" aria-pressed={mode === "dark"} onClick={() => setMode("dark")}>
              Dark
            </button>
          </span>
          <button ref={closeButton} type="button" className="gs-button" onClick={onClose}>
            Close
          </button>
        </span>
      </div>
      <div className="gs-modal-page">
        {result.error !== "" ? (
          <pre className="gs-error" role="alert">
            {result.error}
          </pre>
        ) : (
          content
        )}
      </div>
    </div>,
    document.body,
  );
}

export function ReadmePreviewButton({ api, source, report, mode }: { api: Ghstats; source: string; report: string | undefined; mode: Mode }) {
  const [open, setOpen] = useState(false);
  const opener = useRef<HTMLButtonElement>(null);
  const wasOpen = useRef(false);
  const close = useCallback(() => setOpen(false), []);

  useEffect(() => {
    if (wasOpen.current && !open) opener.current?.focus();
    wasOpen.current = open;
  }, [open]);

  return (
    <>
      <button ref={opener} type="button" className="gs-button" onClick={() => setOpen(true)}>
        README preview
      </button>
      {open ? <ReadmeModal api={api} source={source} report={report} initial={mode} onClose={close} /> : null}
    </>
  );
}
