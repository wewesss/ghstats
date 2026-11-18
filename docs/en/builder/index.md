---
title: Builder
description: Build a ghstats configuration in the browser and see the cards it renders.
sidebar:
  order: 1
---

The builder renders the same cards as the Action, with the same Go code compiled to WebAssembly, from a fixed sample report. Change the options on the left, check the cards on the right, then copy the files at the bottom.

```demo builder
```

Open the [live builder](https://wewesss.github.io/ghstats/en/builder/) if you are reading this on GitHub: the demo above only runs on the site.

## Use the outputs

1. Save `ghstats.yaml` at the root of your profile repository. The Action and the CLI read it from there; the keys are in [Configuration](../guide/configuration.md).
2. Add the workflow to `.github/workflows/ghstats.yml` and replace `<user>` with your login, as described in [Getting started](../guide/getting-started.md).
3. Paste `README.snippet.md` in your profile README. The Action writes the same file after each run, so you can also copy it from the generated `ghstats/` folder.

## Edit the YAML by hand

The `ghstats.yaml` box is editable. As soon as it differs from what the form produces, the form is disabled and the preview follows the text; use Back to the form to return to the form values. A validation error is shown as is, above the preview, with the last valid cards kept below it.

The preview uses sample data: the numbers are not yours. See [Gallery](../cards/gallery.md) for every variant.
