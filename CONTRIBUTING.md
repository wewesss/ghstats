# Contributing to ghstats

Thanks for considering a contribution. A few guidelines keep the project healthy, please read them before opening a PR.

> Version française : [CONTRIBUTING.fr.md](./CONTRIBUTING.fr.md). PRs must still be written in English (see below).

## Language

**PR titles, descriptions, commit messages, code comments and discussion must be in English.** I'm francophone myself, but English keeps the project accessible to everyone. Open issues in English too. The documentation exists in English and French under [`docs/`](./docs); the `*.fr.md` files and `docs/fr` are the only French files.

## Before you open a PR

1. **Fork** the repo and create a branch off `main`.
2. **Install Go** at the version in [`go.mod`](./go.mod). Bun is only needed for the documentation site in `site/`, never for the Go code; if you use a JavaScript package manager at all, use `bun`, not `npm`, `pnpm` or `yarn`.
3. **Run the same checks as CI** before you push:

   ```sh
   gofmt -l .                                   # must print nothing
   go vet ./...
   go test -race ./...
   GOOS=js GOARCH=wasm go vet ./cmd/wasm
   GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/wasm
   go run ./tools/showcase
   git add -N showcase/ && git diff --exit-code showcase/
   go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
   ```

   CI runs the same gates on every PR; if one fails, the PR won't merge. The showcase check fails when the files under `showcase/` are not the output of `go run ./tools/showcase`: run the generator and commit the result whenever you change the renderer or the sample data.

## The documentation site

The site in [`site/`](./site) is built from [`docs/`](./docs) with [`@standarx/mdocs`](https://www.npmjs.com/package/@standarx/mdocs). The builder and the live gallery render the cards in the browser with the Go renderer compiled to WebAssembly, so the build needs Go as well as Bun:

```sh
cd site
bun install --frozen-lockfile
bun run typecheck
bun run build   # compiles the WebAssembly, then writes site/dist
bun run dev     # live reload on the pages, without the renderer: see site/README.md
```

CI runs the same `typecheck` and `build`. The site is deployed to GitHub Pages by `.github/workflows/pages.yml` on every `v*.*.*` tag, or by hand.

## Golden files

The SVG cards are compared with golden files in `render/testdata`. When a change to the renderer is intended, regenerate them and read the diff before you commit:

```sh
go test ./render -update
git diff render/testdata
```

A golden file that changes without a reason in your PR is a regression, not an update. Adding new golden files is fine; changing existing ones needs an explanation in the PR body.

## PR scope

- **One topic per PR.** Don't bundle an unrelated refactor into a bug fix.
- **Small and focused beats big and sprawling.** If a change touches more than about 300 lines across unrelated files, split it.
- **Describe what changed and why** in the PR body. A test that reproduces the bug makes review much faster.
- **No drive-by reformatting.** `gofmt` is part of the checks; don't mix a reformat with logic changes.

## Tests

- Every behavior change needs a test next to the code it changes.
- Every new configuration key needs an accepted case and a rejected case in the `config` tests, with the error message it must produce.
- Don't weaken existing tests to make a change pass. If a test is wrong, explain why in the PR and fix it explicitly.

## Commit style

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), in English, like the existing history:

```
<type>: <short summary>

[optional body]
```

- **Types** used in this repository: `feat`, `fix`, `docs`, `ci`, `chore`, `refactor`, `test`.
- **Summary**: present-tense imperative, lowercase, no trailing period, at most 72 characters. `feat: add the custom link kind`, not `added` or `Added.`.
- **No emoji** and no AI or co-author trailer in a commit message.
- **Breaking changes**: add `!` after the type and a `BREAKING CHANGE:` footer explaining the migration. The configuration `version: 1` and `schema_version: 1` stay stable for the whole v1 series, and so do the Go entry points listed in the extend guide: changing the signature of one of them is a breaking change, so it ships in v2.

## What I'll push back on

- Adding dependencies. The module has two direct dependencies and I'd like to keep it that way.
- Output that contains a private repository name. The rendered files must stay safe for a public repository.
- Breaking changes without a migration note.

## Reporting bugs

Open an issue with:

- The ghstats version (or the `wewesss/ghstats@` ref) and the runner OS.
- Your `ghstats.yaml`, with private names removed.
- What you expected versus what happened, and the log of the failing step.

## Security

If you find something security-relevant, don't open a public issue. Follow [SECURITY.md](./SECURITY.md).

---

Thanks for reading this far. PRs that follow the above tend to merge quickly.
