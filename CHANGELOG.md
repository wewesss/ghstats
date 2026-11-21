# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.0] - 2026-11-21

### Added

- `render.rows`: lay the cards out in `README.snippet.md`. Each row is a `<p align="left|center|right">` holding one or more items (card names, `identity`, `links`) side by side, with a `width` in percent proportional to their natural widths. Without `rows`, the snippet is unchanged.
- `header: false` on a card removes the `login / ghstats` line, the sync date and the rule under them; the card is 40 px shorter.
- `render.languages.multipliers: false` hides the `×<n>` label next to a weighted language; the shares are unchanged.

### Changed

- A `wide` card spreads its blocks over two columns of 495 px: `languages` stays on the right when present, otherwise the blocks are balanced by height.
- Cards placed on the same row by `render.rows` are drawn at the height of the tallest one: the frame and the background are extended and the content stays at the top. `identity` keeps its own height.
- A `wide` card with a single block is now drawn at 495 px instead of 1000 px with an empty column, and the snippet writes `width="495"` for it.

## [1.4.0] - 2026-11-21

### Added

- The Go entry points of `collect`, `compute`, `render`, `config`, `history`, `snapshot` and `report` are listed as stable for the whole v1 series, with runnable examples in each package.
- `ghstats.render(yaml, mode, report?)` in the WASM renderer draws the cards from a real `stats.json`; without the third argument it keeps the sample data.
- The builder loads real data: a GitHub login and path fetch `stats.json` from `raw.githubusercontent.com`, or a local `stats.json` is opened in the browser and never leaves it. The gallery keeps the sample data.
- Releases attach `ghstats.wasm` and `wasm_exec.js`, both covered by `checksums.txt`.
- `examples/server`: a small `net/http` server that serves a `stats.json` and its cards without calling GitHub.
- The guide "Build your own front" documents the fields of `stats.json`, the CLI pipeline, the Go packages, the WASM renderer and the server example, and the FAQ points to it.

## [1.3.0] - 2026-11-21

### Added

- `ghstats init`: sets up a profile repository from your machine through the GitHub CLI. It creates `<user>/<user>` and the private `<user>/ghstats-data` when they are missing, writes the workflow and a minimal `ghstats.yaml`, adds the cards to the profile README between `<!-- ghstats:start -->` and `<!-- ghstats:end -->`, stores the token as the `GHSTATS_TOKEN` secret after checking it, and starts the first run. Every step is idempotent and prints `ok`, `created`, `updated` or `skipped`. Flags: `--user`, `--yes`, `--force`, `--no-run`.
- The getting-started guide opens with the `ghstats init` setup; the manual steps stay under Manual setup.

## [1.2.0] - 2026-11-21

### Added

- `render.cards`: split the render into several images, each grouping blocks and variants as you choose. A card named `overview` writes `overview-dark.svg` and `overview-light.svg`, and `README.snippet.md` lists one picture per card. Without `render.cards`, nothing changes: `render.blocks` gives `stats-<mode>.svg`.
- The builder on the site edits several cards: a name, a layout and its blocks for each one.

## [1.1.1] - 2026-11-18

### Security

- API errors printed by `ghstats collect` no longer carry the request URL or the repository name, so a rate limit hit during the traffic step no longer exposes a private repository name in a public Actions log. The HTTP status, the GitHub message and the reset time are kept.

### Fixed

- `ghstats collect --keep` only removes files named like the snapshots `collect` writes (`snapshot-<timestamp>.json`), and leaves any other file of the folder alone.
- Pre-release tags (`v1.2.0-rc.1`) no longer trigger the release or Pages workflows, no longer move the major tag, are marked as pre-releases by GoReleaser, and are ignored when `@v1` resolves to the newest release.

### Changed

- The Pages workflow grants `pages: write` and `id-token: write` to the deploy job only.
- `go.mod` pins `toolchain go1.27.4`, which fixes GO-2026-6859 for local builds.
- The Action guide states that pinning by commit SHA protects `action.yml`, not the binary, and that the `version` input freezes the binary.

### Added

- Project link buttons at the top of the READMEs and on the home page of the site, generated in `showcase/project/`.

## [1.1.0] - 2026-11-18

### Added

- The `custom` link kind and the `link_style` profile key (`full`, `icon`, `text`).
- `showcase/`: generated SVGs of every card variant, dark and light, kept in sync by CI.
- Documentation in English and French under `docs/`, and illustrated READMEs.
- A GitHub Pages site with a live builder and a live gallery, rendered in the browser by the same Go code compiled to WebAssembly. It is deployed on every `v*.*.*` tag.

### Changed

- Nothing in the CLI or the Action: the binary and the Action behave as in 1.0.0.

## [1.0.0] - 2026-11-17

### Added

- `ghstats collect --keep N` keeps only the N most recent snapshots; `0` keeps them all.
- `keep-snapshots` Action input (default `30`).
- The Action installs the release matching the tag it is pinned to: `@v1` takes the newest v1.x.y, and a branch or SHA ref falls back to the latest release. The `version` input now defaults to empty.
- Stability policy: configuration `version: 1` and `stats.json` `schema_version: 1` are stable for the whole v1 series.

## [0.2.4] - 2026-11-17

- The isometric language stack follows the cycling accent.

## [0.2.3] - 2026-11-17

- The totals show cumulative views instead of the source archives tile.

## [0.2.2] - 2026-11-17

- Tighter identity card layout, with anchored dots.

## [0.2.1] - 2026-11-17

- Link anchors in `README.snippet.md` are free of whitespace.

## [0.2.0] - 2026-11-17

- Profile cards: identity card, wide stats layout, link badges and `README.snippet.md` written by `render`.
- Configuration keys for the profile, the layout and `render.base_path`.

## [0.1.0] - 2026-11-17

- First release of the rewrite: collect, compute and render commands, configurable dark and light SVG cards with JSON output, a traffic hint on 403, release archives and the composite GitHub Action.
