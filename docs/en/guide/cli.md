---
title: Command line
description: The ghstats commands, their flags, the environment variables and the exit codes.
sidebar:
  order: 4
---

# Command line

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
ghstats render  --report <report.json> --out-dir <dir> [--config <file>]
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

`render` writes `stats.json`, `stats-dark.svg` and `stats-light.svg` into `--out-dir`, plus the identity card and the link badges when the configuration has a `profile` section (see [Profile cards](../cards/profile.md)), and `README.snippet.md`.

## version

Prints the version.

## Environment variables

`ghstats collect` reads the token from `GHSTATS_TOKEN`, then from `GITHUB_TOKEN`, and fails when neither is set. The permissions the token needs are in [Getting started](./getting-started.md#1-create-the-token).

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | Runtime error. |
| `2` | Usage error. |

## Example

```sh
export GHSTATS_TOKEN=<token>
snapshot="$(ghstats collect --out data/snapshots --history data/history.json --keep 30)"
ghstats compute --snapshot "$snapshot" --history data/history.json --out report.json --config ghstats.yaml
ghstats render --report report.json --out-dir ghstats --config ghstats.yaml
```
