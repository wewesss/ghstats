---
title: Getting started
description: Set up ghstats on your profile repository, from the token to the first run.
sidebar:
  order: 1
---

This page takes you from nothing to cards published in your profile README.

## Quick setup with ghstats init

`ghstats init` sets up your profile repository from your machine, through the GitHub CLI. The token is the only thing you create by hand.

Before you start:

1. Install the [GitHub CLI](https://cli.github.com) and sign in with `gh auth login`.
2. Give it the `workflow` scope, which it needs to write under `.github/workflows/` and which `gh auth login` does not grant by default: `gh auth refresh -h github.com -s workflow`.
3. Install ghstats with `go install github.com/wewesss/ghstats/cmd/ghstats@latest`, or download a binary from the [releases page](https://github.com/wewesss/ghstats/releases).
4. Create the fine-grained token described in [Create the token](#1-create-the-token). Give it access to all repositories, so it also covers `<user>/ghstats-data`, which `init` creates.

Then run it:

```sh
export GHSTATS_TOKEN=<token>
ghstats init
```

Without `GHSTATS_TOKEN` in the environment, `init` asks for the token when the secret is missing; the typed value is hidden in a terminal. `init` does these steps in order, and prints `ok`, `created`, `updated` or `skipped` for each:

1. Checks that `gh` is installed and has the `workflow` scope, and reads your login (override it with `--user`).
2. Makes sure the profile repository `<user>/<user>` exists, and creates it after a confirmation if it does not.
3. Makes sure the private data repository `<user>/ghstats-data` exists, and creates it if it does not. It stops with an error if that repository is public, even with `--force`, because it holds private repository names.
4. Writes `.github/workflows/ghstats.yml` and a minimal `ghstats.yaml` in the profile repository.
5. Adds the card markup to `README.md` between the markers `<!-- ghstats:start -->` and `<!-- ghstats:end -->`, after showing it and asking for a confirmation.
6. Checks that the token belongs to your account and can read `<user>/ghstats-data`, then stores it as the `GHSTATS_TOKEN` secret of the profile repository. The token goes to `gh` on its standard input and is never printed.
7. Starts the first run of the workflow and prints the URL of the Actions tab. Skip this step with `--no-run`.

Running `init` again is safe. An existing workflow and `ghstats.yaml` are left as they are unless you pass `--force`. Only the text between the README markers is replaced, and nothing is written when it is already up to date. The secret is left alone when it exists and no token is given. The flags are in [Command line](./cli.md#init).

## Manual setup

The steps below do by hand what `ghstats init` does.

## Prerequisites

- A GitHub account with a profile repository named `<user>/<user>`.
- A private repository named `<user>/ghstats-data`. It holds the snapshots and the traffic history, which contain private repository names, so it must stay private. See [Privacy](./privacy.md).
- A Linux or macOS runner: the Action is a composite Action for those runners.

## 1. Create the token

Use a fine-grained personal access token:

- Resource owner: your account, with access to all repositories.
- Repository permissions, read-only: Metadata, Contents, Administration (needed for traffic).
- On top of that, Contents read and write on `<user>/ghstats-data`, so the workflow can push the data.

A fine-grained token covers a single owner, so organization repositories get no traffic data. Use one token per organization, or a classic token with the `repo` scope.

## 2. Store the secret

In the settings of your profile repository, open Secrets and variables, Actions, and create a repository secret named `GHSTATS_TOKEN` with the token.

## 3. Add the workflow

Create `.github/workflows/ghstats.yml` in your profile repository. It checks out the data repository, runs the Action, then commits the data and the cards:

```yaml
name: ghstats

on:
  schedule:
    - cron: "17 3 * * *"
  workflow_dispatch:

permissions:
  contents: write

jobs:
  stats:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - uses: actions/checkout@v7
        with:
          repository: <user>/ghstats-data
          path: ghstats-data
          token: ${{ secrets.GHSTATS_TOKEN }}

      - uses: wewesss/ghstats@v1
        with:
          token: ${{ secrets.GHSTATS_TOKEN }}

      - name: Commit the data repository
        working-directory: ghstats-data
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add -A
          if ! git diff --cached --quiet; then
            git commit -m "chore: update ghstats data"
            git push
          fi

      - name: Commit the cards
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add ghstats
          if ! git diff --cached --quiet; then
            git commit -m "chore: update ghstats cards"
            git push
          fi
```

Replace `<user>` with your login. `wewesss/ghstats@v1` follows the newest v1.x.y release; see [GitHub Action](./action.md) for every input.

The Action runs without a configuration file. To change the blocks, the theme or the profile cards, add a `ghstats.yaml` at the root of the profile repository; the keys are in [Configuration](./configuration.md).

## 4. Reference the cards in your profile README

The Action writes the cards to `ghstats/` and commits them. Paste this in your profile README:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ghstats/stats-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ghstats/stats-light.svg">
  <img alt="GitHub stats" src="ghstats/stats-dark.svg">
</picture>
```

The Action also writes `ghstats/README.snippet.md` with the ready-to-paste markup; with a `profile` section it includes the identity card and the link badges. See [Profile cards](../cards/profile.md).

## 5. Run it once

Open the Actions tab of the profile repository, select the `ghstats` workflow and choose Run workflow. When the run is green, `ghstats/` appears in the repository and the card shows in your profile. After that the workflow runs every day at 03:17 UTC.

If the traffic numbers stay empty, the token cannot read traffic for some owner: the run prints a hint, and the [token section](#1-create-the-token) above explains how to cover organizations.
