import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type DragEvent, type KeyboardEvent } from "react";
import { ALIGNS, type RowState } from "./form";
import { svgUrl, type Mode } from "./ghstats";
import { t, type Key } from "./i18n";
import { canPlace, place, same, step, type CanvasEntry, type CanvasRow, type Direction, type Target } from "./layout";

export const COLUMN = 896;
const GUTTER = 16;

const urls = new Map<string, string>();

function imageUrl(svg: string): string {
  let url = urls.get(svg);
  if (url === undefined) {
    if (urls.size > 120) urls.clear();
    url = svgUrl(svg);
    urls.set(svg, url);
  }
  return url;
}

export function itemLabel(item: string): string {
  if (item === "identity") return t("identityCard");
  if (item.startsWith("links/")) return t("linkNamed", { name: item.slice(6) });
  if (item === "links") return t("links");
  return t("cardNamed", { name: item });
}

function widthStyle(width: string | null): CSSProperties {
  if (width !== null && /^\d+(\.\d+)?%$/.test(width)) return { width, flex: "0 0 auto" };
  if (width !== null && /^\d+$/.test(width)) return { width: `${width}px`, maxWidth: "100%", flex: "0 1 auto" };
  return { flex: "0 1 auto", maxWidth: "100%" };
}

const ALIGN_SIDE: Record<string, Key> = { left: "alignLeft", center: "alignCenter", right: "alignRight" };

const JUSTIFY: Record<string, string> = { left: "flex-start", center: "center", right: "flex-end" };

function AlignGlyph({ align }: { align: string }) {
  const lines = align === "left" ? [[1, 11], [1, 7], [1, 9]] : align === "right" ? [[5, 15], [9, 15], [7, 15]] : [[3, 13], [5, 11], [4, 12]];
  return (
    <svg viewBox="0 0 16 12" width="14" height="11" aria-hidden="true" focusable="false">
      {lines.map(([from, to], at) => (
        <line key={at} x1={from} x2={to} y1={2 + at * 4} y2={2 + at * 4} stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      ))}
    </svg>
  );
}

interface Over {
  key: string;
  ok: boolean;
}

export function Canvas({
  rows,
  state,
  mode,
  selected,
  editable,
  zoom,
  onSelect,
  onArrange,
}: {
  rows: CanvasRow[];
  state: RowState[];
  mode: Mode;
  selected: string | null;
  editable: boolean;
  zoom: "fit" | "100";
  onSelect: (item: string | null) => void;
  onArrange: (rows: RowState[], item: string) => void;
}) {
  const scroller = useRef<HTMLDivElement>(null);
  const column = useRef<HTMLDivElement>(null);
  const focusAfter = useRef<string | null>(null);
  const [available, setAvailable] = useState(COLUMN);
  const [room, setRoom] = useState(0);
  const [height, setHeight] = useState(0);
  const [drag, setDrag] = useState<string | null>(null);
  const [over, setOver] = useState<Over | null>(null);

  useLayoutEffect(() => {
    const node = scroller.current;
    if (node === null) return;
    const measure = () => {
      setAvailable(Math.max(120, node.clientWidth - GUTTER * 2));
      setRoom(Math.max(120, node.clientHeight - GUTTER * 2));
    };
    measure();
    const watch = new ResizeObserver(measure);
    watch.observe(node);
    return () => watch.disconnect();
  }, []);

  useLayoutEffect(() => {
    const node = column.current;
    if (node === null) return;
    const measure = () => setHeight(node.offsetHeight);
    measure();
    const watch = new ResizeObserver(measure);
    watch.observe(node);
    return () => watch.disconnect();
  }, []);

  useEffect(() => {
    const item = focusAfter.current;
    if (item === null) return;
    focusAfter.current = null;
    const nodes = scroller.current?.querySelectorAll<HTMLElement>('[data-testid="element"]');
    Array.from(nodes ?? []).find((node) => node.dataset.item === item)?.focus({ preventScroll: true });
  }, [rows]);

  const fit = Math.min(1, available / COLUMN, height > 0 ? room / height : 1);
  const scale = zoom === "fit" ? Math.max(0.1, Math.floor(fit * 1000) / 1000) : 1;
  const byWidth = zoom === "fit" && available < COLUMN && (height === 0 || available / COLUMN <= room / height);

  const apply = (next: RowState[] | null, item: string) => {
    if (next === null || same(next, state)) return;
    focusAfter.current = item;
    onArrange(next, item);
  };

  const finish = () => {
    setDrag(null);
    setOver(null);
  };

  const zoneProps = (key: string, target: Target) => ({
    "data-over": over?.key === key ? (over.ok ? "ok" : "deny") : undefined,
    onDragOver: (event: DragEvent) => {
      if (drag === null) return;
      const ok = canPlace(state, drag, target);
      if (over?.key !== key || over.ok !== ok) setOver({ key, ok });
      if (ok) {
        event.preventDefault();
        event.dataTransfer.dropEffect = "move";
      }
    },
    onDragLeave: () => {
      if (over?.key === key) setOver(null);
    },
    onDrop: (event: DragEvent) => {
      event.preventDefault();
      const item = drag ?? event.dataTransfer.getData("text/plain");
      finish();
      if (item !== "") apply(place(state, item, target), item);
    },
  });

  const move = (item: string, direction: Direction) => apply(step(state, item, direction), item);

  const onKey = (event: KeyboardEvent, item: string) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onSelect(item);
      return;
    }
    if (!editable || event.altKey || event.ctrlKey || event.metaKey) return;
    const direction: Direction | undefined = { ArrowLeft: "left", ArrowRight: "right", ArrowUp: "up", ArrowDown: "down" }[event.key] as Direction | undefined;
    if (direction === undefined) return;
    event.preventDefault();
    onSelect(item);
    move(item, direction);
  };

  const band = (at: number) => {
    const key = `row:${at}`;
    return (
      <div key={key} className="gs-newrow" data-testid="drop-newrow" data-at={at} aria-hidden="true" {...zoneProps(key, { kind: "row", at })}>
        <span>{t("newRow")}</span>
      </div>
    );
  };

  const element = (entry: CanvasEntry, row: number, index: number) => {
    const isSelected = selected === entry.item;
    const before = `slot:${row}:${index}`;
    const after = `slot:${row}:${index + 1}`;
    return (
      <div
        key={entry.item}
        className="gs-el"
        data-testid="element"
        data-item={entry.item}
        data-selected={isSelected}
        data-dragging={drag === entry.item}
        data-over={over?.key === before ? "before" : over?.key === after ? "after" : undefined}
        data-deny={over !== null && !over.ok && (over.key === before || over.key === after)}
        style={widthStyle(entry.width)}
        tabIndex={0}
        role="button"
        aria-pressed={isSelected}
        aria-label={itemLabel(entry.item)}
        draggable={editable}
        onClick={() => onSelect(entry.item)}
        onKeyDown={(event) => onKey(event, entry.item)}
        onDragStart={(event) => {
          event.dataTransfer.setData("text/plain", entry.item);
          event.dataTransfer.effectAllowed = "move";
          const item = entry.item;
          setTimeout(() => setDrag(item), 0);
        }}
        onDragEnd={finish}
      >
        <img src={imageUrl(entry.svg)} alt="" draggable={false} style={{ width: entry.width === null ? "auto" : "100%" }} />
        <div className="gs-zone gs-zone-before" data-testid="drop-before" aria-hidden="true" {...zoneProps(before, { kind: "slot", row, at: index })} />
        <div className="gs-zone gs-zone-after" data-testid="drop-after" aria-hidden="true" {...zoneProps(after, { kind: "slot", row, at: index + 1 })} />
      </div>
    );
  };

  return (
    <div
      ref={scroller}
      className="gs-canvas"
      data-testid="canvas-scroll"
      data-dragging={drag !== null}
      onClick={(event) => {
        const target = event.target as Element;
        if (target.closest('[data-testid="element"]') === null && target.closest("button") === null) onSelect(null);
      }}
    >
      <div className="gs-sizer" style={{ width: COLUMN * scale, height: height * scale, ...(byWidth ? { marginTop: 0 } : {}) }}>
        <div
          ref={column}
          className="gs-readme"
          data-testid="canvas"
          data-mode={mode}
          data-scale={scale.toFixed(3)}
          style={{ width: COLUMN, transform: `scale(${scale})` }}
        >
          {rows.map((row, r) => (
            <div key={r} className="gs-line" data-testid="row" data-index={r} data-align={row.align}>
              {band(r)}
              <div className="gs-line-body" style={{ justifyContent: JUSTIFY[row.align] ?? "flex-start" }}>
                {row.entries.map((entry, i) => element(entry, r, i))}
                {editable ? (
                  <div className="gs-tools" role="group" aria-label={t("rowAlignment", { n: r + 1 })} data-testid="row-tools">
                    {ALIGNS.map((align) => (
                      <button
                        key={align}
                        type="button"
                        data-testid={`align-${align}`}
                        aria-label={t("alignRow", { n: r + 1, side: t(ALIGN_SIDE[align] ?? "alignLeft") })}
                        title={t("alignTitle", { side: t(ALIGN_SIDE[align] ?? "alignLeft") })}
                        aria-pressed={row.align === align}
                        onClick={() => {
                          const next = state.map((entry, at) => (at === r ? { ...entry, align } : entry));
                          if (!same(next, state)) onArrange(next, row.entries[0]?.item ?? "");
                        }}
                      >
                        <AlignGlyph align={align} />
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>
            </div>
          ))}
          {band(rows.length)}
        </div>
      </div>
    </div>
  );
}
