---
title: Faire son propre front
description: Lire stats.json, appeler les paquets Go, charger le moteur WASM ou servir soi-même les cartes.
sidebar:
  order: 6
---

ghstats calcule les chiffres une seule fois et les écrit dans `stats.json`. Tout ce qui dessine les cartes, le builder du site compris, lit ce fichier ou les paquets Go qui sont derrière. Vous pouvez faire la même chose dans votre propre page, script ou serveur.

## Lire stats.json

Le workflow commite `stats.json` dans le dossier de sortie du dépôt de votre profil. GitHub le sert à une URL fixe, avec un CORS ouvert : une page web de n'importe quelle origine peut donc le lire :

```text
https://raw.githubusercontent.com/<user>/<user>/HEAD/ghstats/stats.json
```

`HEAD` désigne la branche par défaut, et `ghstats/` est le `output-dir` de l'Action (voir [Action GitHub](./action.md)).

```js
const url = "https://raw.githubusercontent.com/<user>/<user>/HEAD/ghstats/stats.json";
const response = await fetch(url);
if (!response.ok) throw new Error(`stats.json: ${response.status}`);
const stats = await response.json();
console.log(stats.login, stats.totals.stars);
```

`stats.json` ne contient jamais le nom d'un dépôt privé, et `schema_version: 1` est stable pour toute la série v1 : un champ ajouté ne casse rien, ignorez donc les champs que vous ne connaissez pas. Voir [Confidentialité](./privacy.md).

| Champ | Type | Sens |
| --- | --- | --- |
| `schema_version` | entier | Version de ce document, `1`. |
| `generated_at` | date RFC 3339 | Moment où `compute` a tourné. |
| `collected_at` | date RFC 3339 | Moment où `collect` a récupéré les données. |
| `login` | chaîne | Le login GitHub auquel appartient le jeton. |
| `totals.repos` | entier | Dépôts retenus par le `scope` de la configuration. |
| `totals.public_repos` | entier | Parmi eux, les dépôts publics. |
| `totals.private_repos` | entier | Parmi eux, les dépôts privés ; `0` sauf si `scope.include_private` est actif. |
| `totals.stars` | entier | Étoiles additionnées sur ces dépôts. |
| `totals.forks` | entier | Forks additionnés sur ces dépôts. |
| `totals.followers` | entier | Abonnés du compte. |
| `languages.mode` | chaîne | `raw` ou `adjusted`, d'après `languages.mode`. |
| `languages.raw` | liste de parts de langage | Tous les langages, en octets de code. |
| `languages.adjusted` | liste de parts de langage | La même chose, après les poids et les exclusions de la configuration. |
| `languages.display` | liste de parts de langage | Ce que montrent les cartes : les parts du mode choisi, après `hide` et `max_visible`, les langages repliés étant additionnés dans l'entrée `other_label`. |
| `languages.multipliers` | objet | Nom du langage vers le poids appliqué ; seulement en mode `adjusted` et seulement pour les poids différents de `1`. |
| `languages.hidden_share` | chaîne | `other` ou `renormalize` : où sont passés les langages masqués. |
| `distribution` | liste de métriques | Compteurs de téléchargements et de trafic, une entrée par canal et par fenêtre. |
| `trends` | liste de tendances | Séries hebdomadaires et quotidiennes pour `clones` et `views` ; seulement les canaux que la configuration active. |
| `warnings` | liste de chaînes | Problèmes de collecte, sous forme de comptes. Vide quand tout s'est bien passé. |

Une part de langage contient `name`, `bytes` (octets de code), `weighted` (octets après les poids) et `share` (une fraction entre 0 et 1 du total).

Une métrique contient :

| Champ | Type | Sens |
| --- | --- | --- |
| `channel` | chaîne | `release_assets`, `clones`, `views` ou `source_archives`. |
| `count` | entier ou `null` | La valeur ; `null` quand le canal est indisponible. |
| `window` | chaîne | `all_time`, `last_14_days` ou `since:<AAAA-MM-JJ>`, le premier jour de l'historique conservé. |
| `reliability` | chaîne | `observed`, `partial` (certains dépôts n'ont pas de données) ou `unavailable`. |
| `note` | chaîne, facultative | Une courte explication, par exemple pourquoi un canal est indisponible. |

`source_archives` est toujours `unavailable` : GitHub n'expose aucun compteur pour les téléchargements des sources. Les comptes de trafic sont additionnés sur les dépôts, donc les visiteurs uniques ne le sont pas.

Une tendance contient `channel`, `weeks` (26 entrées, `start` est le lundi de la semaine et `count` son total) et `days` (les 28 derniers jours, `day` et `count`).

## Lancer soi-même le pipeline

La CLI fait les trois étapes sur votre machine, et l'Action fait pareil sur un runner. `collect`, `compute` et `render` sont décrits dans [Ligne de commande](./cli.md) :

```sh
export GHSTATS_TOKEN=<jeton>
ghstats collect --out data/snapshots --history data/history.json
ghstats compute --snapshot data/snapshots/snapshot-<horodatage>.json --history data/history.json --out report.json
ghstats render --report report.json --out-dir out
```

`report.json` a le même contenu que `stats.json` : vous pouvez vous arrêter après `compute` et dessiner les cartes vous-même.

## Utiliser les paquets Go

```sh
go get github.com/wewesss/ghstats@v1
```

Ces points d'entrée sont stables pour toute la série v1, comme `report` (ses types et ses constantes) :

| Paquet | Points d'entrée |
| --- | --- |
| `collect` | `Collect`, `Options` |
| `compute` | `Compute` |
| `render` | `Cards`, `Snippet`, `JSON`, `Mode`, `Dark`, `Light` |
| `config` | `Config`, `Parse`, `Load`, `Default` |
| `history` | `History`, `Read`, `ReadFile`, `Write`, `WriteFile`, `Merge` |
| `snapshot` | `Snapshot`, `Read`, `ReadFile`, `Write`, `WriteFile` |
| `report` | `Report` et les autres types, et les constantes |

Tout le reste peut changer dans n'importe quelle version 1.x. Un changement de signature dans la liste ci-dessus est un changement incompatible et sort en v2. Chaque paquet a des exemples exécutables dans son `example_test.go`.

Du jeton aux cartes :

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/wewesss/ghstats/collect"
	"github.com/wewesss/ghstats/compute"
	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/history"
	"github.com/wewesss/ghstats/render"
)

func main() {
	cfg := config.Default()
	snap, err := collect.Collect(context.Background(), collect.Options{
		Token:   os.Getenv("GHSTATS_TOKEN"),
		Traffic: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	merged := history.Merge(history.History{}, snap)
	rep, err := compute.Compute(snap, merged, cfg, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	cards, err := render.Cards(rep, cfg, render.Dark)
	if err != nil {
		log.Fatal(err)
	}
	for name, svg := range cards {
		if err := os.WriteFile(name, []byte(svg), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
```

`history.Merge` replie le trafic du snapshot dans l'historique que vous conservez, ce qui fait survivre le trafic aux 14 jours qu'expose GitHub. Lisez votre historique avec `history.ReadFile` et enregistrez le résultat fusionné avec `history.WriteFile`. `render.Cards` renvoie une table de nom de fichier vers texte SVG ; les badges `links/` et la carte identité en font partie quand la configuration a une section `profile`, il faut alors créer les dossiers avant d'écrire les fichiers.

## Utiliser le moteur WASM

Chaque release joint le moteur qui fait tourner le [builder](https://wewesss.github.io/ghstats/) :

- `ghstats.wasm`, le moteur ;
- `wasm_exec.js`, le chargeur Go qui lui correspond ;
- `checksums.txt`, les sommes SHA-256 de tous les fichiers de la release, ces deux-là compris.

Téléchargez-les depuis la [page des releases](https://github.com/wewesss/ghstats/releases) et hébergez-les vous-même. Chargez `wasm_exec.js`, créez un objet `Go`, instanciez le module et lancez-le ; le moteur enregistre alors `ghstats` sur `globalThis` :

```html
<script src="wasm_exec.js"></script>
<img id="card" alt="Statistiques GitHub">
<script type="module">
  const go = new Go();
  const { instance } = await WebAssembly.instantiateStreaming(fetch("ghstats.wasm"), go.importObject);
  go.run(instance);

  const stats = await (await fetch("stats.json")).text();
  const result = ghstats.render("version: 1\n", "dark", stats);
  if (result.error !== "") throw new Error(result.error);
  document.getElementById("card").src =
    "data:image/svg+xml;utf8," + encodeURIComponent(result.cards["stats-dark.svg"]);
</script>
```

L'API :

| Appel | Renvoie |
| --- | --- |
| `ghstats.render(yaml, mode, report?, still?)` | `{cards, snippet, yaml, error}`. `yaml` est la configuration (texte YAML ou JSON), `mode` vaut `"dark"` ou `"light"`, et `report` est le texte d'un `stats.json` ; sans lui, le moteur dessine des données d'exemple. `cards` associe des noms de fichiers à du texte SVG, `snippet` est le snippet du README, `yaml` est la configuration normalisée, et `error` est vide en cas de succès. Un `stats.json` dont le `schema_version` n'est pas `1` donne une erreur. `still` est un quatrième argument facultatif : quand il est vrai, les cartes sont dessinées avec `render.motion` coupé, quoi que dise la configuration (passez `undefined` ou une chaîne vide comme `report` pour garder les données d'exemple). |
| `ghstats.defaults()` | La configuration de départ, en texte YAML. |
| `ghstats.options()` | Une chaîne JSON qui liste ce que la configuration accepte : blocs et variantes, préréglages, accents, palettes, mises en page, types et styles de liens, et limites. |

Le moteur n'appelle jamais GitHub, et `report` ne quitte jamais la page.

## Servir en HTTP

[`examples/server`](https://github.com/wewesss/ghstats/blob/main/examples/server/main.go) est un petit serveur `net/http` construit uniquement sur les paquets Go ci-dessus. Il lit un `stats.json`, dessine les cartes des deux modes au démarrage et n'appelle jamais GitHub :

```sh
go run ./examples/server --report out/stats.json [--config ghstats.yaml]
```

| Route | Réponse |
| --- | --- |
| `GET /stats.json` | Le rapport. |
| `GET /cards/<name>-<mode>.svg` | Une carte, par exemple `/cards/stats-dark.svg`. |
| `GET /` | Le snippet du README, qui pointe vers `/cards/`. |

Il écoute sur `127.0.0.1:8080`. Il n'a ni cache ni limite : c'est un exemple à copier, pas une fonctionnalité.

## SVG personnalisé

ghstats n'a pas de langage de gabarit pour les cartes. Pour un autre rendu, faites le vôtre à partir de `stats.json`, ou forkez le paquet `render` : le projet est sous licence MIT.
