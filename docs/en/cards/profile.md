---
title: Profile cards
description: The identity card, the wide layout, the link badges and README.snippet.md.
sidebar:
  order: 2
---

# Profile cards

An optional `profile` section adds an identity card and one small badge per link. Everything follows the `render.theme`, so presets, accents (including `cycle`, kept in sync across all cards) and overrides apply to every file.

## Configuration

```yaml
profile:
  name: octocat                    # default: the report login
  tagline: SOFTWARE · GAMES · EXPERIMENTS
  location: caen, fr               # optional
  note: no fixed stack, it depends on what I'm making   # optional
  focus:                           # 1 to 6 entries, each with 1 to 4 building words
    - name: WEB & DESKTOP
      building: [full-stack apps, desktop software]
    - name: GAMES
      building: [gameplay systems, game engines]
  links:                           # kept in this order; an empty value is skipped
    - youtube: "@octocat"          # a handle or a full https:// URL
    - website: https://example.com
    - discord: {url: "https://discord.com/invite/xyz", label: discord}
    - github_sponsors: octocat
render:
  layout: wide
  base_path: ghstats/
```

The keys are described in [Configuration](../guide/configuration.md#profile).

## Identity card

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="../../../showcase/identity-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="../../../showcase/identity-light.svg">
  <img alt="identity card" src="../../../showcase/identity-dark.svg" width="100%">
</picture>

The identity card is written as `identity-dark.svg` and `identity-light.svg` when `profile` has at least `focus` or `tagline`. The `building` words rotate every 3 seconds and the matching focus row lights up; with `motion: false` or reduced motion the first word and first row are shown.

## Wide layout

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="../../../showcase/wide-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="../../../showcase/wide-light.svg">
  <img alt="wide layout" src="../../../showcase/wide-dark.svg" width="100%">
</picture>

`render.layout: wide` makes the stats card 1000 px wide, with `languages` in a right column and the other blocks stacked on the left. The identity card has the same width. The default `card` layout is 495 px.

## Link badges

<p>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="../../../showcase/badges/youtube-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="../../../showcase/badges/youtube-light.svg">
    <img alt="youtube" src="../../../showcase/badges/youtube-dark.svg">
  </picture>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="../../../showcase/badges/website-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="../../../showcase/badges/website-light.svg">
    <img alt="website" src="../../../showcase/badges/website-dark.svg">
  </picture>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="../../../showcase/badges/github_sponsors-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="../../../showcase/badges/github_sponsors-light.svg">
    <img alt="github_sponsors" src="../../../showcase/badges/github_sponsors-dark.svg">
  </picture>
</p>

`links/<kind>-dark.svg` and `links/<kind>-light.svg` are written for each non-empty link. ghstats does not delete the files of a link you remove from the configuration.

Link kinds: `youtube`, `twitch`, `kick`, `discord`, `tiktok`, `instagram`, `x`, `bluesky`, `threads`, `mastodon`, `reddit`, `linkedin`, `steam`, `itchio`, `website`, `github_sponsors`, `open_collective`, `buy_me_a_coffee`, `ko_fi`, `patreon`. A value is a handle (a leading `@` is optional) or a full `https://` URL; `mastodon`, `steam` and `website` need a full URL. The map form `{url, label}` overrides the label.

A link is clickable only when its badge is a separate image wrapped in its own `<a>`, which is why each link is its own SVG file and not part of the identity card.

The badge icons come from [Simple Icons](https://simpleicons.org) (CC0 1.0). The brand logos remain the property of their respective owners.

## README.snippet.md

`ghstats render` always writes `README.snippet.md` next to the cards: the identity `<picture>` (when generated), the stats `<picture>`, then a centered row with one `<a href="URL"><picture>...</picture></a>` per link, in configuration order. Every path is prefixed with `render.base_path`. Paste it into your profile README.

For the profile of the gallery, the file reads:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ghstats/identity-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ghstats/identity-light.svg">
  <img alt="profile" src="ghstats/identity-dark.svg" width="100%">
</picture>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ghstats/stats-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ghstats/stats-light.svg">
  <img alt="GitHub stats" src="ghstats/stats-dark.svg" width="495">
</picture>

<p align="center">
<a href="https://www.youtube.com/@wewesss14"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/youtube-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/youtube-light.svg"><img alt="youtube" src="ghstats/links/youtube-dark.svg"></picture></a>
<a href="https://miralabs.tech"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/website-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/website-light.svg"><img alt="website" src="ghstats/links/website-dark.svg"></picture></a>
<a href="https://github.com/sponsors/wewesss"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/github_sponsors-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/github_sponsors-light.svg"><img alt="github_sponsors" src="ghstats/links/github_sponsors-dark.svg"></picture></a>
</p>
```

Commit the whole output directory (`git add ghstats`) so the badges and the identity card are published with the stats cards. The workflow in [Getting started](../guide/getting-started.md) does it.
