---
title: Build your own front
description: Read stats.json, call the Go packages, load the WASM renderer or serve the cards yourself.
sidebar:
  order: 6
---

ghstats computes the numbers once and writes them to `stats.json`. Everything that draws the cards, the site builder included, reads that file or the Go packages behind it. You can do the same in your own page, script or server.

## Read stats.json

The workflow commits `stats.json` into the output directory of your profile repository. GitHub serves it from a fixed URL, with open CORS, so a web page on any origin can read it:

```text
https://raw.githubusercontent.com/<user>/<user>/HEAD/ghstats/stats.json
```

`HEAD` is the default branch, and `ghstats/` is the `output-dir` of the Action (see [GitHub Action](./action.md)).

```js
const url = "https://raw.githubusercontent.com/<user>/<user>/HEAD/ghstats/stats.json";
const response = await fetch(url);
if (!response.ok) throw new Error(`stats.json: ${response.status}`);
const stats = await response.json();
console.log(stats.login, stats.totals.stars);
```

`stats.json` never holds a private repository name, and `schema_version: 1` is stable for the whole v1 series: an added field breaks nothing, so ignore the fields you do not know. See [Privacy](./privacy.md).

| Field | Type | Meaning |
| --- | --- | --- |
| `schema_version` | integer | Version of this document, `1`. |
| `generated_at` | RFC 3339 time | When `compute` ran. |
| `collected_at` | RFC 3339 time | When `collect` fetched the data. |
| `login` | string | The GitHub login the token belongs to. |
| `totals.repos` | integer | Repositories kept by the `scope` of the configuration. |
| `totals.public_repos` | integer | Public ones among them. |
| `totals.private_repos` | integer | Private ones among them; `0` unless `scope.include_private` is on. |
| `totals.stars` | integer | Stars summed across those repositories. |
| `totals.forks` | integer | Forks summed across those repositories. |
| `totals.followers` | integer | Followers of the account. |
| `languages.mode` | string | `raw` or `adjusted`, from `languages.mode`. |
| `languages.raw` | list of language shares | Every language, by bytes of code. |
| `languages.adjusted` | list of language shares | The same, after the weights and exclusions of the configuration. |
| `languages.display` | list of language shares | What the cards show: the shares of the chosen mode, after `hide` and `max_visible`, with the folded languages summed under the `other_label` entry. |
| `languages.multipliers` | object | Language name to the weight applied; only in `adjusted` mode and only for weights other than `1`. |
| `languages.hidden_share` | string | `other` or `renormalize`: where the hidden languages went. |
| `distribution` | list of metrics | Download and traffic counters, one entry per channel and window. |
| `trends` | list of trends | Weekly and daily series for `clones` and `views`; only the channels the configuration enables. |
| `warnings` | list of strings | Collection problems, as counts. Empty when everything worked. |

A language share holds `name`, `bytes` (bytes of code), `weighted` (bytes after the weights) and `share` (a fraction between 0 and 1 of the total).

A metric holds:

| Field | Type | Meaning |
| --- | --- | --- |
| `channel` | string | `release_assets`, `clones`, `views` or `source_archives`. |
| `count` | integer or `null` | The value; `null` when the channel is unavailable. |
| `window` | string | `all_time`, `last_14_days` or `since:<YYYY-MM-DD>`, the first day of the stored history. |
| `reliability` | string | `observed`, `partial` (some repositories have no data) or `unavailable`. |
| `note` | string, optional | A short explanation, such as why a channel is unavailable. |

`source_archives` is always `unavailable`: GitHub exposes no counter for source downloads. Traffic counts are summed across repositories, so unique visitors are not summed.

A trend holds `channel`, `weeks` (26 entries, `start` is the Monday of the week and `count` its total) and `days` (the last 28 days, `day` and `count`).

## Run the pipeline yourself

The CLI runs the three steps on your machine, and the Action does the same on a runner. Collect, compute and render are described in [Command line](./cli.md):

```sh
export GHSTATS_TOKEN=<token>
ghstats collect --out data/snapshots --history data/history.json
ghstats compute --snapshot data/snapshots/snapshot-<timestamp>.json --history data/history.json --out report.json
ghstats render --report report.json --out-dir out
```

`report.json` has the same content as `stats.json`: you can stop after `compute` and draw the cards yourself.

## Use the Go packages

```sh
go get github.com/wewesss/ghstats@v1
```

These entry points are stable for the whole v1 series, as is `report` (its types and constants):

| Package | Entry points |
| --- | --- |
| `collect` | `Collect`, `Options` |
| `compute` | `Compute` |
| `render` | `Cards`, `Snippet`, `JSON`, `Mode`, `Dark`, `Light` |
| `config` | `Config`, `Parse`, `Load`, `Default` |
| `history` | `History`, `Read`, `ReadFile`, `Write`, `WriteFile`, `Merge` |
| `snapshot` | `Snapshot`, `Read`, `ReadFile`, `Write`, `WriteFile` |
| `report` | `Report` and the other types, and the constants |

Everything else can change in any 1.x release. A signature change in the list above is a breaking change and ships in v2. Each package has runnable examples in its `example_test.go`.

From a token to the cards:

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/wewesss/ghstats/collect"
	"github.com/wewesss/ghstats/compute"
	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/render"
)

func main() {
	cfg := config.Default()
	snap, err := collect.Collect(context.Background(), collect.Options{
		Token:   os.Getenv("GHSTATS_TOKEN"),
		Traffic: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	merged := history.Merge(history.History{}, snap)
	rep, err := compute.Compute(snap, merged, cfg, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	cards, err := render.Cards(rep, cfg, render.Dark)
	if err != nil {
		log.Fatal(err)
	}
	for name, svg := range cards {
		if err := os.WriteFile(name, []byte(svg), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
```

`history.Merge` folds the traffic of the snapshot into the history you keep, which is how the traffic outlives the 14 days GitHub exposes. Read your stored history with `history.ReadFile` and save the merged one with `history.WriteFile`. `render.Cards` returns a map from file name to SVG text; the `links/` badges and the identity card are in it when the configuration has a `profile` section, so create the folders before writing the files in that case.

## Use the WASM renderer

Each release attaches the renderer that powers the [builder](https://wewesss.github.io/ghstats/):

- `ghstats.wasm`, the renderer;
- `wasm_exec.js`, the Go loader that matches it;
- `checksums.txt`, the SHA-256 sums of every release file, these two included.

Download them from the [releases page](https://github.com/wewesss/ghstats/releases) and host them yourself. Load `wasm_exec.js`, create a `Go` object, instantiate the module and run it; the renderer then registers `ghstats` on `globalThis`:

```html
<script src="wasm_exec.js"></script>
<img id="card" alt="GitHub stats">
<script type="module">
  const go = new Go();
  const { instance } = await WebAssembly.instantiateStreaming(fetch("ghstats.wasm"), go.importObject);
  go.run(instance);

  const stats = await (await fetch("stats.json")).text();
  const result = ghstats.render("version: 1\n", "dark", stats);
  if (result.error !== "") throw new Error(result.error);
  document.getElementById("card").src =
    "data:image/svg+xml;utf8," + encodeURIComponent(result.cards["stats-dark.svg"]);
</script>
```

The API:

| Call | Returns |
| --- | --- |
| `ghstats.render(yaml, mode, report?, still?)` | `{cards, snippet, yaml, error}`. `yaml` is the configuration (YAML or JSON text), `mode` is `"dark"` or `"light"`, and `report` is the text of a `stats.json`; without it, the renderer draws sample data. `cards` maps file names to SVG text, `snippet` is the README snippet, `yaml` is the normalized configuration, and `error` is empty on success. A `stats.json` whose `schema_version` is not `1` gives an error. `still` is an optional fourth argument: when it is truthy, the cards are drawn with `render.motion` off, whatever the configuration says (pass `undefined` or an empty string as `report` to keep the sample data). |
| `ghstats.defaults()` | The starter configuration, as YAML text. |
| `ghstats.options()` | A JSON string listing what the configuration accepts: blocks and variants, presets, accents, palettes, layouts, link kinds and styles, and limits. |

The renderer never calls GitHub, and `report` never leaves the page.

## Serve it over HTTP

[`examples/server`](https://github.com/wewesss/ghstats/blob/main/examples/server/main.go) is a small `net/http` server built only on the Go packages above. It reads a `stats.json`, renders the cards for both modes at start-up and never calls GitHub:

```sh
go run ./examples/server --report out/stats.json [--config ghstats.yaml]
```

| Route | Answer |
| --- | --- |
| `GET /stats.json` | The report. |
| `GET /cards/<name>-<mode>.svg` | A card, for example `/cards/stats-dark.svg`. |
| `GET /` | The README snippet, pointing at `/cards/`. |

It listens on `127.0.0.1:8080`. It has no cache and no limits: it is an example to copy, not a feature.

## Custom SVG

ghstats has no template language for the cards. To get another look, build your own rendering from `stats.json`, or fork the `render` package: the project is MIT licensed.
