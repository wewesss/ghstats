---
title: Configuration
description: Every key of ghstats.yaml, with its allowed values and defaults.
sidebar:
  order: 3
---

ghstats reads one YAML file, `ghstats.yaml` by default. Every key except `version` is optional, and an empty file means the defaults. Unknown keys and invalid values are refused with a message naming the key. The complete annotated file is [`config/example.yaml`](../../../config/example.yaml), and a full example closes this page.

```yaml
version: 1
```

`version` must be `1`.

## scope

Which repositories are counted.

| Key | Default | Description |
| --- | --- | --- |
| `scope.owners` | empty | Only count repositories of these owners (case-insensitive). Empty means every owner returned by the collection. |
| `scope.include_forks` | `false` | Count forks. |
| `scope.include_archived` | `false` | Count archived repositories. |
| `scope.include_private` | `true` | Count private repositories in the aggregates. Their names are never output. |
| `scope.exclude_repos` | empty | `owner/name` or a glob such as `owner/*-old` (case-insensitive). |
| `scope.repo_weights` | empty | Map of `owner/name` to a multiplier (>= 0) applied to every language of that repository, in adjusted mode only. |

## languages

How the language shares are computed and displayed.

| Key | Default | Description |
| --- | --- | --- |
| `languages.mode` | `raw` | `raw` or `adjusted`. |
| `languages.weights` | empty | Map of language to a multiplier (>= 0, default 1, names are case-insensitive). |
| `languages.exclude` | empty | Languages removed from the adjusted calculation, same as a weight of 0. The raw shares are untouched. |
| `languages.hide` | empty | Languages removed from the display only. |
| `languages.hidden_share` | `other` | What happens to the share of hidden languages: `other` folds it into the Other entry, `renormalize` rescales the visible shares so they sum to 1. |
| `languages.max_visible` | `6` | Languages shown before the rest is folded into Other. `0` is unlimited. |
| `languages.other_label` | `Other` | Label of the folded entry. Must not be empty. |

### Raw and adjusted

In `raw` mode the shares are the bytes of code GitHub reports for each language, as they are. In `adjusted` mode each language is first multiplied by its weight in `languages.weights`, and each repository by its weight in `scope.repo_weights`, then the shares are computed. Use it to lower a language that is mostly generated or vendored (a weight of `0.35` counts 35 percent of the bytes) or to drop one entirely (`0`).

`stats.json` keeps the raw and the adjusted shares side by side.

## distribution

Switches for the `distribution` entries of the report: the download count of release assets, and the clones and views traffic.

| Key | Default |
| --- | --- |
| `distribution.release_assets` | `true` |
| `distribution.clones` | `true` |
| `distribution.views` | `true` |
| `distribution.hide_partial` | `false` |

`distribution.hide_partial`: Shows n/a on the cards instead of a traffic total that misses some repositories (reliability `partial`). stats.json keeps the count.

## render

How the cards look. See the [gallery](../cards/gallery.md) for every block and variant.

| Key | Allowed values |
| --- | --- |
| `render.blocks` | List of `block: [variants]`, stacked in this order. `stats`: `totals`, `distribution`. `languages`: `bars`, `iso`, `cells`, `treemap`, `shift`, `terminal`. `history`: `views`, `clones`. |
| `render.cards` | Optional list of cards, each with `name`, `blocks` and an optional `layout`. Splits the render into several images; `render.blocks` is then ignored. See [Several cards](#several-cards). |
| `render.rows` | Optional list of README rows, each with an `align` and the `items` it holds. Controls how `README.snippet.md` lays out the cards. See [Layout in the README](#layout-in-the-readme). |
| `render.private_bar` | `true` (default) or `false`. Replaces the names of private repositories with a black bar. |
| `render.languages.colors` | `accent` (default), `github`, `mono`, `spectrum`, `duotone`, or a map of language to `"#rrggbb"`. |
| `render.languages.weighting` | `true` (default) or `false`. `false` hides every trace of language weighting: the `×<n>` label next to a language that has a weight, the `adjusted` / `raw` tag at the top right of the language blocks, the RAW / ADJ strip under the `bars` variant (the card is shorter) and the `--adjusted` flag in the `terminal` command. The shares are unchanged. With `false`, listing the `shift` variant is an error, since it shows the weighting. |
| `render.theme.preset` | `noir` (default), `carbon`, `graphite`, `midnight`, `slate`, `dune`, `transparent`. |
| `render.theme.accent` | `ice` (default), `mint`, `amber`, `silver`, `violet`, `rose`, `lime`, `coral`, `cycle`, or `{dark: "#rrggbb", light: "#rrggbb"}`. |
| `render.theme.overrides` | Keys `bg`, `border`, `fg`, `muted`, `faint` as `"#rrggbb"`; `bg` also accepts `none`. |
| `render.motion` | `true` (default) or `false`. `false` turns every animation off. Reduced motion is always honored. |
| `render.layout` | `card` (495 px, default) or `wide` (1000 px, two columns of 495 px). With `languages`, it sits in the right column and the other blocks on the left; without it, the blocks are spread to keep the two columns level. A `wide` card with a single block is drawn at 495 px. |
| `render.base_path` | Prefix of every path written in `README.snippet.md`, ending with `/` (default `ghstats/`). |

### Blocks and variants

Each block appears once. A block with one variant shows it fixed; a block with several variants rotates them in place:

```yaml
render:
  blocks:
    - stats: [totals, distribution]
    - languages: [bars, treemap]
    - history: [views]
```

### Several cards

Without `render.cards`, `render.blocks` gives one card, `stats-dark.svg` and `stats-light.svg`. With `render.cards`, each card is its own pair of images and groups blocks and variants as you like:

```yaml
version: 1
render:
  cards:
    - name: overview
      blocks:
        - stats: [totals]
        - history: [views]
    - name: langs
      layout: wide
      blocks:
        - languages: [treemap]
        - history: [clones]
```

This writes `overview-dark.svg`, `overview-light.svg`, `langs-dark.svg` and `langs-light.svg`. `stats.json` and the link badges are unchanged, and the identity card keeps its name.

- `name` is required, unique, lowercase letters, digits and `-` (1 to 40 characters, starting with a letter or a digit). `identity` and `links` are reserved.
- `blocks` follows the same rules as `render.blocks`, and needs at least one block per card. A block may appear in several cards.
- `layout` is `card` or `wide`. When it is missing, the card inherits `render.layout`. A `wide` card with a single block is drawn as a `card`.
- `header` is `true` (default) or `false`. `false` removes the `login / ghstats` line, the sync date and the rule under them, and the card is 40 px shorter.
- At most 12 cards.
- When `render.cards` is set, `render.blocks` is ignored.

`README.snippet.md` lists one `<picture>` per card, in order, `495` wide for a `card` layout and `100%` for a `wide` one. The alt text is `GitHub stats: <name>`:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ghstats/overview-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ghstats/overview-light.svg">
  <img alt="GitHub stats: overview" src="ghstats/overview-dark.svg" width="495">
</picture>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ghstats/langs-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ghstats/langs-light.svg">
  <img alt="GitHub stats: langs" src="ghstats/langs-dark.svg" width="100%">
</picture>
```

One card per variant:

```yaml
render:
  cards:
    - name: totals
      blocks:
        - stats: [totals]
    - name: distribution
      blocks:
        - stats: [distribution]
    - name: treemap
      blocks:
        - languages: [treemap]
    - name: views
      blocks:
        - history: [views]
```

### Layout in the README

Without `render.rows`, `README.snippet.md` stacks one picture per item: the identity card, each card in order, then the links. With `render.rows`, each row becomes a `<p align>` and holds one or more items side by side:

```yaml
version: 1
render:
  cards:
    - name: overview
      blocks: [{stats: [totals]}, {history: [views]}]
      layout: wide
    - name: langs
      header: false
      blocks: [{languages: [treemap]}]
    - name: views
      header: false
      blocks: [{history: [clones]}]
  rows:
    - align: center
      items: [langs, views]
    - align: right
      items: [overview]
```

The snippet becomes:

```html
<p align="center">
  <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/langs-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/langs-light.svg"><img alt="GitHub stats: langs" src="ghstats/langs-dark.svg" width="49%"></picture> <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/views-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/views-light.svg"><img alt="GitHub stats: views" src="ghstats/views-dark.svg" width="49%"></picture>
</p>

<p align="right">
  <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/overview-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/overview-light.svg"><img alt="GitHub stats: overview" src="ghstats/overview-dark.svg" width="98%"></picture>
</p>
```

GitHub keeps `align` on `<p>` and `width` on `<img>`, so the two cards of the first row share the line.

- `align` is `left`, `center` or `right`; it defaults to `center`.
- `items` lists card names, `identity`, `links` and `links/<kind>` (a single link badge, see [Placing link badges](#placing-link-badges)). An item can be placed once in all the rows. A row that holds a link item holds only link items.
- With several items in a row, each `width` is a percentage of the natural widths of the row (495 for a `card`, 1000 for the identity card and a `wide` card), with 2 % left free. An item alone in its row is drawn at 98 % when it is a `wide` card or the identity card, and keeps 495 when it is a `card`.
- Cards in the same row are drawn at the height of the tallest one: the frame and background are extended, the header stays at the top and the blocks are centered vertically in the extra space. The identity card keeps its height.
- `links` without links and a `links/<kind>` whose link is missing or empty are skipped, and a row left empty is not written. `identity` placed in a row always draws the identity card (see [Identity card](#identity-card)).
- Items that appear in no row are added after the rows, in the default order: identity, cards, links, with the usual widths (495 for a `card`, `100%` for a `wide` card, 495 for an identity card in the `card` layout, `100%` for the other identity cards). Links that no row places are added last in one centered row.
- At most 12 rows.

### Placing link badges

`links/<kind>` places the badge of one link, for any link kind of the profile. `links` places every badge that no `links/<kind>` places elsewhere. The badges of a row are written one after the other in a single `<p align>`, so a row can hold badges only, never badges and cards.

```yaml
version: 1
render:
  rows:
    - align: center
      items: [identity, stats]
    - align: left
      items: [links/youtube, links/website]
    - align: right
      items: [links]
profile:
  name: octocat
  tagline: SOFTWARE
  identity:
    layout: card
    show: [header, name, tagline, footer]
  links:
    - youtube: "@octocat"
    - website: {url: "https://example.com", style: icon}
    - github_sponsors: octocat
```

The snippet becomes:

```html
<p align="center">
  <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/identity-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/identity-light.svg"><img alt="profile" src="ghstats/identity-dark.svg" width="49%"></picture> <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/stats-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/stats-light.svg"><img alt="GitHub stats" src="ghstats/stats-dark.svg" width="49%"></picture>
</p>

<p align="left">
<a href="https://www.youtube.com/@octocat"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/youtube-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/youtube-light.svg"><img alt="youtube" src="ghstats/links/youtube-dark.svg"></picture></a>
<a href="https://example.com"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/website-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/website-light.svg"><img alt="website" src="ghstats/links/website-dark.svg"></picture></a>
</p>

<p align="right">
<a href="https://github.com/sponsors/octocat"><picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/links/github_sponsors-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/links/github_sponsors-light.svg"><img alt="github_sponsors" src="ghstats/links/github_sponsors-dark.svg"></picture></a>
</p>
```

The identity card is 495 wide here (`identity.layout: card`), so it shares the first row with the stats card at 49 % each.

### Theme

`preset` sets the background and text colors, `accent` the highlight color. `cycle` makes the accent change over time, kept in sync across all the cards. A custom accent needs both `dark` and `light`. `overrides` replaces single colors of the preset:

```yaml
render:
  theme:
    preset: midnight
    accent: {dark: "#9ec5ff", light: "#2457c5"}
    overrides:
      bg: none
      fg: "#e6e8ee"
```

### Palettes

`render.languages.colors` is a palette name or your own map. Languages missing from a map use their GitHub color.

```yaml
render:
  languages:
    colors: {TypeScript: "#3178c6", Go: "#00add8"}
```

## profile

An optional section that adds an identity card and one badge per link. Everything follows `render.theme`. See [Profile cards](../cards/profile.md) for the files it produces.

| Key | Description |
| --- | --- |
| `profile.name` | Name on the card. Default: the report login. |
| `profile.tagline` | One line under the name. |
| `profile.location` | Optional. |
| `profile.note` | Optional. |
| `profile.focus` | 1 to 6 entries, each `{name, building}` where `building` holds 1 to 4 words. |
| `profile.links` | List of `kind: value`, kept in this order; an empty value is skipped. |
| `profile.link_style` | Look of every link button: `full` (default, icon and label), `icon` (icon only) or `text` (label only). A link can override it with its own `style`. |
| `profile.identity.show` | Parts of the identity card to draw, among `header`, `name`, `tagline`, `location`, `note`, `building`, `focus` and `footer`. Default: every part that has a value. |
| `profile.identity.layout` | `wide` (default, 1000 px) or `card` (495 px, everything in one column). |

The identity card is generated when `profile` has at least `focus` or `tagline`, when `profile.identity.show` is set, or when `identity` is placed in `render.rows`.

### Identity card

Without `profile.identity`, the card is drawn as before. `identity.show` lists the parts to draw; a part that is not listed is not drawn, the card height follows the parts that remain and no gap is left:

- `header`: the small name, `/ profile` and the pulsing dot, with the rule under them.
- `name`: the large name.
- `tagline`, `note`: the lines of the same name.
- `location`: the location, at the right of the top strip. The strip, the dot and the rule exist only when `header` is drawn; without `header`, the location sits at the top right of the first content line.
- `building`: the `$ building` line with the rotating words of `focus`.
- `focus`: the numbered focus list. Without it, the right column and its vertical rule disappear.
- `footer`: the rule and `github.com/<login>` at the bottom.

A part with no value (a `tagline` that is not set, for example) is not drawn. An unknown or repeated part is a validation error.

`identity.layout: card` draws the card 495 px wide with one column: header, name (its size is reduced so that it fits 447 px), tagline, `building` line, note, then the focus list under the text, then the footer. In `render.rows` and in the snippet, this card counts as 495 wide.

```yaml
profile:
  name: octocat
  location: caen, fr
  identity:
    show: [name, location]
    layout: card
```

### Links

Link kinds: `youtube`, `twitch`, `kick`, `discord`, `tiktok`, `instagram`, `x`, `bluesky`, `threads`, `mastodon`, `reddit`, `linkedin`, `steam`, `itchio`, `website`, `github_sponsors`, `open_collective`, `buy_me_a_coffee`, `ko_fi`, `patreon`, `custom`. Each kind can be listed once.

A value is a handle (a leading `@` is optional) or a full `https://` URL; `mastodon`, `steam` and `website` need a full URL. The map form `{url, label, style}` overrides the label and the style:

```yaml
profile:
  links:
    - youtube: "@octocat"
    - website: https://example.com
    - discord: {url: "https://discord.com/invite/xyz", label: discord}
```

The `custom` kind is a button to any URL and exists only in the map form `{url, label, icon}`: `url` (a full `https://` URL) and `label` are required, `icon` is optional. `icon` is either a known kind name (`youtube`, `discord`...), which reuses its embedded icon, or an SVG path starting with `M` or `m` (the `d` attribute of a `<path>` in a 24x24 viewBox; `<`, `>`, `"` and `&` are rejected). Without `icon`, the button shows a globe.

```yaml
profile:
  links:
    - custom: {url: "https://example.com/blog", label: my blog, icon: youtube}
```

The scalar form `custom: https://...`, a missing `url` or `label`, an `icon` on another kind and an icon that is neither a kind name nor a valid path are validation errors.

### Link style

`profile.link_style` is `full` (default: icon and label), `icon` (a square with the icon only; the label stays in `aria-label` and `<title>`) or `text` (the label only). Any other value is a validation error that lists the three.

```yaml
profile:
  link_style: icon
```

A link written in the map form can carry its own `style`, which replaces `link_style` for that link only:

```yaml
profile:
  link_style: full
  links:
    - youtube: "@octocat"
    - website: {url: "https://example.com", style: icon}
```

See [Profile cards](../cards/profile.md#button-styles) for the rendering of each style.

## Full example

```yaml
version: 1
scope:
  include_forks: false
  include_archived: false
  include_private: true
languages:
  mode: raw
  hidden_share: other
  max_visible: 6
  other_label: Other
distribution:
  release_assets: true
  clones: true
  views: true
  hide_partial: false
render:
  blocks:
    - stats:
        - totals
    - languages:
        - bars
    - history:
        - views
  private_bar: true
  languages:
    colors: accent
  theme:
    preset: noir
    accent: cycle
  motion: true
  layout: card
  base_path: ghstats/
profile:
  name: wewesss
  tagline: SOFTWARE · GAMES · EXPERIMENTS
  location: caen, fr
  note: no fixed stack — it depends on what I'm making
  focus:
    - name: WEB & DESKTOP
      building:
        - full-stack apps
        - desktop software
    - name: GAMES
      building:
        - gameplay systems
        - game engines
    - name: AI & AUTOMATION
      building:
        - coding agents
        - automated workflows
    - name: SYSTEMS
      building:
        - runtimes & compilers
        - native tooling
  links:
    - youtube: '@wewesss14'
    - website: https://miralabs.tech
    - github_sponsors: wewesss
```
