# Contribuer à ghstats

Merci d'envisager une contribution. Quelques règles gardent le projet sain, merci de les lire avant d'ouvrir une PR.

> English version : [CONTRIBUTING.md](./CONTRIBUTING.md) (la version faisant foi).

## Langue

**Les titres de PR, les descriptions, les messages de commit, les commentaires de code et les discussions doivent être en anglais.** Je suis francophone moi-même, mais l'anglais garde le projet accessible à tout le monde. Les issues aussi. La documentation existe en anglais et en français sous [`docs/`](./docs) ; les fichiers `*.fr.md` et `docs/fr` sont les seuls fichiers en français.

Cette page est traduite pour faciliter la lecture, mais **la version anglaise fait foi** en cas de divergence, et **votre PR doit être rédigée en anglais**.

## Avant d'ouvrir une PR

1. **Forkez** le dépôt et créez une branche à partir de `main`.
2. **Installez Go** à la version de [`go.mod`](./go.mod). Bun n'est nécessaire que pour le site de documentation dans `site/`, jamais pour le code Go ; si vous utilisez un gestionnaire de paquets JavaScript, utilisez `bun`, pas `npm`, `pnpm` ni `yarn`.
3. **Lancez les mêmes vérifications que la CI** avant de pousser :

   ```sh
   test -z "$(gofmt -l .)"                      # échoue si un fichier n'est pas formaté ; gofmt -l . les liste
   go vet ./...
   go test -race ./...
   GOOS=js GOARCH=wasm go vet ./cmd/wasm
   GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/wasm
   go run ./tools/showcase
   git add -N showcase/ && git diff --exit-code showcase/
   go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
   go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
   bash -n scripts/install.sh
   ```

   La CI lance les mêmes portes sur chaque PR, plus les deux tests de résolution de tag de `scripts/install.sh` (les étapes `Resolve the release tag` et `Ignore pre-release tags` du job `action` de [`ci.yml`](./.github/workflows/ci.yml)) ; si l'une échoue, la PR n'est pas fusionnée. La vérification du showcase échoue quand les fichiers de `showcase/` ne sont pas la sortie de `go run ./tools/showcase` : relancez le générateur et commitez le résultat dès que vous modifiez le moteur de rendu ou les données d'exemple.

## Le site de documentation

Le site de [`site/`](./site) est construit depuis [`docs/`](./docs) avec [`@standarx/mdocs`](https://www.npmjs.com/package/@standarx/mdocs). Le builder et la galerie vivante rendent les cartes dans le navigateur avec le moteur de rendu Go compilé en WebAssembly : la construction a donc besoin de Go en plus de Bun :

```sh
cd site
bun install --frozen-lockfile
bun run typecheck
bun run build   # compile le WebAssembly, puis écrit site/dist
bunx playwright install --with-deps chromium
bun run test:e2e
bun run dev     # rechargement des pages, sans le moteur de rendu : voir site/README.md
```

La CI lance les mêmes `typecheck`, `build`, installation de Playwright et `test:e2e`. Le site est déployé sur GitHub Pages par `.github/workflows/pages.yml` à chaque tag qui correspond à `v[0-9]+.[0-9]+.[0-9]+` (les tags de préversion sont exclus), ou à la main.

## Le moteur PNG

Les cartes PNG sont dessinées par resvg compilé en WebAssembly, versionné sous `png/resvg.wasm`. Quand vous modifiez `tools/resvgwasm`, reconstruisez-le avec `tools/resvgwasm/build.sh` (il demande Rust et la cible `wasm32-unknown-unknown`) et commitez le `png/resvg.wasm` régénéré.

## Fichiers golden

Les cartes SVG sont comparées à des fichiers golden dans `render/testdata`. Quand une modification du rendu est voulue, régénérez-les et relisez le diff avant de commiter :

```sh
go test ./render -update
git diff render/testdata
```

Un golden qui change sans raison dans votre PR est une régression, pas une mise à jour. Ajouter de nouveaux golden est normal ; modifier les existants demande une explication dans le corps de la PR.

## Portée d'une PR

- **Un sujet par PR.** Ne mélangez pas un refactor sans rapport avec un correctif.
- **Petit et ciblé vaut mieux que gros et dispersé.** Si un changement touche plus d'environ 300 lignes réparties sur des fichiers sans lien, découpez-le.
- **Décrivez ce qui change et pourquoi** dans le corps de la PR. Un test qui reproduit le bug accélère beaucoup la relecture.
- **Pas de reformatage au passage.** `gofmt` fait partie des vérifications ; ne mélangez pas un reformatage avec des changements de logique.

## Tests

- Chaque changement de comportement demande un test à côté du code modifié.
- Chaque nouvelle clé de configuration demande un cas accepté et un cas refusé dans les tests de `config`, avec le message d'erreur attendu.
- N'affaiblissez pas les tests existants pour faire passer un changement. Si un test est faux, expliquez pourquoi dans la PR et corrigez-le explicitement.

## Style de commit

Suivez [Conventional Commits](https://www.conventionalcommits.org/fr/v1.0.0/), en anglais, comme l'historique existant :

```
<type>: <résumé court>

[corps facultatif]
```

- **Types** utilisés dans ce dépôt : `feat`, `fix`, `docs`, `ci`, `chore`, `refactor`, `test`.
- **Résumé** : impératif présent, en minuscules, sans point final, 72 caractères au plus. `feat: add the custom link kind`, pas `added` ni `Added.`.
- **Pas d'emoji** et aucun trailer d'IA ou de co-auteur dans un message de commit.
- **Changements incompatibles** : ajoutez `!` après le type et un pied `BREAKING CHANGE:` qui explique la migration. La configuration `version: 1` et `schema_version: 1` restent stables pour toute la série v1, comme les points d'entrée Go listés dans le guide pour faire son propre front : changer la signature de l'un d'eux est un changement incompatible, donc réservé à la v2.

## Ce que je refuserai

- Ajouter des dépendances. Le module a quatre dépendances directes (voir [`go.mod`](./go.mod)) et je voudrais que ce nombre reste petit.
- Une sortie qui contient le nom d'un dépôt privé. Les fichiers rendus doivent rester sans danger pour un dépôt public.
- Les changements incompatibles sans note de migration.

## Signaler un bug

Ouvrez une issue avec :

- La version de ghstats (ou la référence `wewesss/ghstats@`) et l'OS du runner.
- Votre `ghstats.yaml`, sans les noms privés.
- Ce que vous attendiez et ce qui s'est passé, avec le journal de l'étape en échec.

## Sécurité

Si vous trouvez un problème de sécurité, n'ouvrez pas d'issue publique. Suivez [SECURITY.md](./SECURITY.md).

---

Merci d'avoir lu jusqu'ici. Les PR qui suivent ces règles sont en général fusionnées vite.
