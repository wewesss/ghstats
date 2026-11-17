---
title: GitHub Action
description: Every input and output of the ghstats Action, how the version is resolved and which files it produces.
sidebar:
  order: 2
---

# GitHub Action

The repository is a composite Action for Linux and macOS runners. It installs the ghstats release, then runs `collect`, `compute` and `render` in order. A complete workflow is in [Getting started](./getting-started.md).

## Inputs

| Input | Default | Description |
| --- | --- | --- |
| `token` | required | Personal access token used to collect. |
| `version` | empty | Release tag to download. Empty: the tag the action is pinned to (`@v1` takes the newest v1.x.y), or the latest release when the ref is a branch or SHA. `latest` forces the latest release. |
| `config` | `ghstats.yaml` | Configuration file, ignored when the file is missing. |
| `data-dir` | `ghstats-data` | Directory holding `snapshots/` and `history.json`. |
| `output-dir` | `ghstats` | Directory receiving the cards and `README.snippet.md`. |
| `traffic` | `true` | Set to `false` to skip the traffic API calls. |
| `keep-snapshots` | `30` | Number of snapshots kept in `data-dir/snapshots`; `0` keeps them all. |

## Outputs

| Output | Description |
| --- | --- |
| `snapshot` | Path of the snapshot written by `collect`. |
| `report` | Path of `report.json`. |

## Version resolution

The `version` input decides which release of the binary is installed:

- `@v1` (a major tag) installs the newest published v1.x.y release.
- `@v1.0.0` (a full tag) installs exactly that release.
- A branch or a commit SHA installs the latest release and prints a warning.
- `version: latest` installs the latest release whatever the ref.
- `version: v1.0.0` (any release tag) installs that tag whatever the ref.

Pin `wewesss/ghstats@v1` to follow the v1 series: the configuration `version: 1` and the `stats.json` `schema_version: 1` are stable for all of it.

## Files produced

| Path | Written by | Content |
| --- | --- | --- |
| `<data-dir>/snapshots/snapshot-<timestamp>.json` | `collect` | The raw snapshot. It contains private repository names. |
| `<data-dir>/history.json` | `collect` | The traffic history, merged at each run. It contains private repository names. |
| `<output-dir>/stats.json` | `render` | The report, without any private repository name. |
| `<output-dir>/stats-dark.svg`, `stats-light.svg` | `render` | The stats cards. |
| `<output-dir>/identity-dark.svg`, `identity-light.svg` | `render` | The identity card, when `profile` has `focus` or `tagline`. |
| `<output-dir>/links/<kind>-dark.svg`, `<kind>-light.svg` | `render` | One badge per non-empty profile link. |
| `<output-dir>/README.snippet.md` | `render` | The markup to paste in your profile README. |

The report itself is written to the runner temporary directory and returned by the `report` output. Keep `<data-dir>` in a private repository and commit `<output-dir>` to the public one. See [Privacy](./privacy.md).
