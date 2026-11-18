import { useEffect, useState } from "react";
import { presetRender } from "./form";
import { loadGhstats, svgUrl, type Ghstats, type Mode, type Options } from "./ghstats";

function card(api: Ghstats, config: Record<string, unknown>, mode: Mode): { svg: string; error: string } {
  const result = api.render(JSON.stringify(config), mode);
  return { svg: result.cards[`stats-${mode}.svg`] ?? "", error: result.error };
}

function GalleryApp({ api, options }: { api: Ghstats; options: Options }) {
  const [preset, setPreset] = useState(options.presets[0] ?? "noir");
  const [accent, setAccent] = useState("cycle");
  const [palette, setPalette] = useState("accent");

  return (
    <div className="gs-gallery">
      <div className="gs-toolbar">
        <label className="gs-field">
          <span>Preset</span>
          <select value={preset} onChange={(event) => setPreset(event.target.value)}>
            {options.presets.map((entry) => (
              <option key={entry}>{entry}</option>
            ))}
          </select>
        </label>
        <label className="gs-field">
          <span>Accent</span>
          <select value={accent} onChange={(event) => setAccent(event.target.value)}>
            {options.accents.map((entry) => (
              <option key={entry}>{entry}</option>
            ))}
          </select>
        </label>
        <label className="gs-field">
          <span>Language colors</span>
          <select value={palette} onChange={(event) => setPalette(event.target.value)}>
            {options.palettes.map((entry) => (
              <option key={entry}>{entry}</option>
            ))}
          </select>
        </label>
      </div>
      {options.blocks.flatMap((block) =>
        block.variants.map((variant) => {
          const config = presetRender(block.name, variant, preset, accent, palette);
          return (
            <section className="gs-variant" key={`${block.name}-${variant}`}>
              <h4>
                {block.name}: {variant}
              </h4>
              <div className="gs-pair">
                {(["dark", "light"] as const).map((mode) => {
                  const { svg, error } = card(api, config, mode);
                  return error !== "" ? (
                    <pre className="gs-error" key={mode} role="alert">
                      {error}
                    </pre>
                  ) : (
                    <img key={mode} className="gs-card" alt={`${block.name}: ${variant}, ${mode}`} src={svgUrl(svg)} />
                  );
                })}
              </div>
            </section>
          );
        }),
      )}
    </div>
  );
}

export function Gallery() {
  const [ready, setReady] = useState<{ api: Ghstats; options: Options } | null>(null);
  const [failure, setFailure] = useState("");

  useEffect(() => {
    let live = true;
    loadGhstats()
      .then((api) => {
        if (live) setReady({ api, options: api.options() });
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
  return <GalleryApp api={ready.api} options={ready.options} />;
}
