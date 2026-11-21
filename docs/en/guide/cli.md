---
title: Command line
description: The ghstats commands, their flags, the environment variables and the exit codes.
sidebar:
  order: 4
---

The GitHub Action runs these commands for you. Use the CLI directly to try a configuration locally or to run ghstats outside GitHub Actions.

## Install

Install with Go or download a release archive from the [releases page](https://github.com/wewesss/ghstats/releases):

```sh
go install github.com/wewesss/ghstats/cmd/ghstats@latest
```

## Synopsis

```sh
ghstats collect [--out data/snapshots] [--history data/history.json] [--no-traffic] [--concurrency N] [--keep N]
ghstats compute --snapshot <file> --history <file> --out <report.json> [--config <file>]
ghstats render  --report <report.json> --out-dir <dir> [--config <file>] [--png]
ghstats init    [--user <login>] [--yes] [--force] [--no-run]
ghstats version
```

`ghstats <command> -h` prints the flags of a command.

## collect

Fetches repositories, languages, releases and traffic into a snapshot.

| Flag | Default | Description |
| --- | --- | --- |
| `--out` | `data/snapshots` | Directory receiving the snapshot file. |
| `--history` | `data/history.json` | Traffic history file, updated in place. |
| `--no-traffic` | off | Skip the traffic API calls. |
| `--concurrency` | `0` | Parallel API requests; `0` uses the default. |
| `--keep` | `0` | Snapshots kept in `--out`; `0` keeps them all. |

`collect` writes `snapshot-<timestamp>.json` into `--out`, merges it into the history file and prints the snapshot path. With `--keep N` (N > 0) only the N most recent `snapshot-*.json` files of `--out` are kept.

## compute

Turns a snapshot and a history into `report.json`.

| Flag | Required | Description |
| --- | --- | --- |
| `--snapshot` | yes | Snapshot file. |
| `--history` | yes | Traffic history file; a missing file counts as empty. |
| `--out` | yes | Report file to write. |
| `--config` | no | Configuration file; the defaults apply when omitted. |

Warnings are printed on the error output.

## render

Writes the files of a report.

| Flag | Required | Description |
| --- | --- | --- |
| `--report` | yes | Report file. |
| `--out-dir` | yes | Directory receiving the files. |
| `--config` | no | Configuration file; the defaults apply when omitted. |
| `--png` | no | Also write `<name>-dark.png` and `<name>-light.png` next to every SVG, at 2x with every animation off. Same as `render.png: true`. |

`render` writes `stats.json`, `stats-dark.svg` and `stats-light.svg` into `--out-dir`, plus the identity card and the link badges when the configuration has a `profile` section (see [Profile cards](../cards/profile.md)), and `README.snippet.md`. With `render.cards`, the stats files are `<name>-dark.svg` and `<name>-light.svg` instead, one pair per card (see [Several cards](./configuration.md#several-cards)).

## init

Sets up a profile repository from your machine, through the GitHub CLI (`gh`), which must be installed and signed in with the `workflow` scope. The steps and the prerequisites are in [Getting started](./getting-started.md#quick-setup-with-ghstats-init).

| Flag | Default | Description |
| --- | --- | --- |
| `--user` | account of `gh` | GitHub login. |
| `--yes` | off | Answer yes to the confirmations (README, creation of the profile repository). |
| `--force` | off | Rewrite the workflow and `ghstats.yaml` even when they exist. |
| `--no-run` | off | Do not start the first run. |

Each step prints `ok` (already in place), `created`, `updated` or `skipped`, so running `init` again changes nothing that is already set up. `init` stops with an error when `<user>/ghstats-data` is public, even with `--force`.

## version

Prints the version.

## Environment variables

`ghstats collect` reads the token from `GHSTATS_TOKEN`, then from `GITHUB_TOKEN`, and fails when neither is set. The permissions the token needs are in [Getting started](./getting-started.md#1-create-the-token).

`ghstats init` reads `GHSTATS_TOKEN` and stores it as the `GHSTATS_TOKEN` secret of the profile repository after checking that it belongs to `<user>` and can read `<user>/ghstats-data`. When the variable is empty and the secret is missing, `init` asks for the token; when the secret already exists, it asks for nothing. `GHSTATS_GRAPHQL_URL` and `GHSTATS_REST_URL` change the API endpoints used for that check.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Runtime error (for `init`: `gh` missing, a missing scope, a public data repository, a refused token). |
| `2` | Usage error. |

## Example

```sh
export GHSTATS_TOKEN=<token>
snapshot="$(ghstats collect --out data/snapshots --history data/history.json --keep 30)"
ghstats compute --snapshot "$snapshot" --history data/history.json --out report.json --config ghstats.yaml
ghstats render --report report.json --out-dir ghstats --config ghstats.yaml
```
