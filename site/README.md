# ghstats site

The documentation site, built from [`../docs`](../docs) with [`@standarx/mdocs`](https://www.npmjs.com/package/@standarx/mdocs). Pages are prerendered; the builder and the live gallery are React demos in [`demos/`](./demos) that run the Go renderer, compiled to WebAssembly, in the browser.

## Commands

Run them in this folder. Bun and Go are required.

```sh
bun install --frozen-lockfile
bun run typecheck
bun run build
```

`bun run build` runs three steps:

1. `bun run wasm` compiles `../cmd/wasm` to `public/ghstats.wasm` and copies Go's `wasm_exec.js` next to it. Both files are git-ignored.
2. `mdocs build` writes the pages to `dist/`, under the base path `/ghstats/`.
3. `bun run assets` copies `public/` and `../showcase/` into `dist/`.

## Static files

mdocs copies stylesheets and bundles the demos, but it has no folder of static files, and it empties `dist/` on every build. That is why step 3 exists. `dist/showcase/` is also what the relative image paths of the docs (`../../../showcase/...`) resolve to on the site, so the docs read the same on GitHub and on Pages.

## Testing the builder locally

`bun run dev` serves the pages with live reload at the site root, but it cannot serve the WebAssembly files: the demos then show a load error. To try the builder, build the site and serve `dist/` under `/ghstats/`:

```sh
bun run build
bun -e 'Bun.serve({ port: 4500, fetch(req) { const url = new URL(req.url); if (!url.pathname.startsWith("/ghstats/")) return new Response("not found", { status: 404 }); let path = "dist/" + url.pathname.slice(9); if (path.endsWith("/")) path += "index.html"; const file = Bun.file(path); return file.size === 0 ? new Response("not found", { status: 404 }) : new Response(file); } })'
```

Then open `http://127.0.0.1:4500/ghstats/en/builder/`.

## Deployment

`.github/workflows/pages.yml` builds and deploys the site on every `v*.*.*` tag and on demand. In the repository settings, Pages must use GitHub Actions as its source.
