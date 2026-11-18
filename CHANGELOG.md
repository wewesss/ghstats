# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
