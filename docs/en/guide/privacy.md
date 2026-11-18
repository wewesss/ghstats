---
title: Privacy
description: What contains private repository names, what does not, and the limit of organization traffic.
sidebar:
  order: 5
---

## What is private

Raw snapshots contain private repository names. Keep them, and the history file, in a private repository only. This is why the workflow in [Getting started](./getting-started.md) checks out a separate private `<user>/ghstats-data` repository into `data-dir` and commits `snapshots/` and `history.json` there.

The rendered `stats.json` and SVG files never contain a private repository name, so the output directory is safe to commit to a public profile repository.

| Path | Private repository names | Where to keep it |
| --- | --- | --- |
| `<data-dir>/snapshots/` | yes | private repository |
| `<data-dir>/history.json` | yes | private repository |
| `<output-dir>/` (`stats.json`, SVG files, `README.snippet.md`) | no | public profile repository |

## Private repositories in the numbers

Private repositories are counted in the aggregates when `scope.include_private` is `true` (the default); set it to `false` to leave them out. Their names are never output. When `render.private_bar` is `true`, a private repository is shown as a black bar instead of a name.

## Organization traffic

Traffic needs push access with a token that covers the owner of each repository. A fine-grained token covers a single owner, so organization repositories get no traffic data and `collect` prints a hint when GitHub answers 403. Use one token per organization, or a classic token with the `repo` scope.

## Brand logos

The badge icons come from [Simple Icons](https://simpleicons.org) (CC0 1.0). The brand logos remain the property of their respective owners.
