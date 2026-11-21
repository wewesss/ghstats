---
title: Builder
description: Build a ghstats configuration in the browser, arrange the README on the preview and copy the files.
sidebar:
  order: 1
---

The builder renders the same cards as the Action, with the same Go code compiled to WebAssembly, from sample data, a GitHub login or a `stats.json` file. It takes the whole window: the settings are on the left and scroll on their own, the README preview is on the right and never moves while you edit.

```demo builder
```

Open the [live builder](https://wewesss.github.io/ghstats/en/builder/) if you are reading this on GitHub: the demo above only runs on the site.

## Arrange the README

The preview is the README as GitHub draws it, built from the snippet the Action writes.

- Click a card, the identity card or a link badge to select it: the Selection section of the settings shows only its options. Escape, Deselect or a click on the empty canvas clears the selection.
- Drag an element before or after another one to put it in the same row, or onto a New row band to give it a row of its own. A row of links only accepts links; a refused drop is outlined in red.
- With an element selected, the arrow buttons or the arrow keys move it in its row and to the row above or below.
- Hover a row, or select one of its elements, to get its left, center and right alignment buttons.
- On a card, Make it a card moves a block into a new card. The new card goes next to the current one only when both are in the `card` layout and the row holds no link; otherwise it goes in a new row below.
- With an element selected, Download SVG and Download PNG save it for the current preview mode; Download all (.zip) packs every element in SVG and PNG, dark and light. The PNG is drawn without animation at twice the size, like `ghstats render --png`, and its renderer loads on the first PNG download.

## Use the outputs

The tabs under the toolbar hold the three files.

1. Save `ghstats.yaml` at the root of your profile repository. The Action and the CLI read it from there; the keys are in [Configuration](../guide/configuration.md).
2. Add the workflow to `.github/workflows/ghstats.yml` and replace `<user>` with your login, as described in [Getting started](../guide/getting-started.md).
3. Paste `README.snippet.md` in your profile README. The Action writes the same file after each run, so you can also copy it from the generated `ghstats/` folder.

## Edit the YAML by hand

The `ghstats.yaml` tab is editable. As soon as it differs from what the form produces, the form is disabled and the preview follows the text; use Back to the form to return to the form values. A validation error is shown as is, above the preview, with the last valid cards kept below it.

The preview uses sample data until you load a login or a file: with the sample the numbers are not yours. See [Gallery](../cards/gallery.md) for every variant.
