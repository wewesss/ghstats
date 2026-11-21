---
title: Action GitHub
description: Tous les inputs et outputs de l'Action ghstats, la résolution de version et les fichiers produits.
sidebar:
  order: 2
---

Le dépôt est une Action composite pour les runners Linux et macOS. Elle installe la release de ghstats, puis lance `collect`, `compute` et `render` dans l'ordre. Un workflow complet est dans [Prise en main](./getting-started.md).

## Inputs

| Input | Défaut | Description |
| --- | --- | --- |
| `token` | requis | Token d'accès personnel utilisé pour collecter. |
| `version` | vide | Tag de release à télécharger. Vide : le tag sur lequel l'action est épinglée (`@v1` prend la dernière v1.x.y), ou la dernière release quand la ref est une branche ou un SHA. `latest` force la dernière release. |
| `config` | `ghstats.yaml` | Fichier de configuration, ignoré quand le fichier n'existe pas. |
| `data-dir` | `ghstats-data` | Dossier contenant `snapshots/` et `history.json`. |
| `output-dir` | `ghstats` | Dossier qui reçoit les cartes et `README.snippet.md`. |
| `traffic` | `true` | Mettre `false` pour sauter les appels à l'API de trafic. |
| `keep-snapshots` | `30` | Nombre de snapshots conservés dans `data-dir/snapshots` ; `0` les garde tous. |
| `png` | `false` | Mettre `true` pour écrire aussi un `-dark.png` et un `-light.png` à côté de chaque SVG (2x, sans animation : un bloc qui liste plusieurs variantes n'en montre que la première, et l'accent `cycle` s'affiche en `ice`). Équivaut à `render.png: true`. |

## Outputs

| Output | Description |
| --- | --- |
| `snapshot` | Chemin du snapshot écrit par `collect`. |
| `report` | Chemin de `report.json`. |

## Résolution de version

L'input `version` décide quelle release du binaire est installée :

- `@v1` (un tag majeur) installe la dernière release v1.x.y publiée.
- `@v1.0.0` (un tag complet) installe exactement cette release.
- Une branche ou un SHA de commit installe la dernière release et affiche un avertissement.
- `version: latest` installe la dernière release quelle que soit la ref.
- `version: v1.0.0` (n'importe quel tag de release) installe ce tag quelle que soit la ref.

Épinglez `wewesss/ghstats@v1` pour suivre la série v1 : la configuration `version: 1` et le `schema_version: 1` de `stats.json` y sont stables.

Épingler l'Action par SHA de commit protège `action.yml`, pas le binaire : une ref SHA installe la dernière release. Pour figer aussi le binaire, renseignez l'input `version`, par exemple `version: v1.1.1`.

## Fichiers produits

| Chemin | Écrit par | Contenu |
| --- | --- | --- |
| `<data-dir>/snapshots/snapshot-<timestamp>.json` | `collect` | Le snapshot brut. Il contient des noms de dépôts privés. |
| `<data-dir>/history.json` | `collect` | L'historique de trafic, fusionné à chaque exécution. Il contient des noms de dépôts privés. |
| `<output-dir>/stats.json` | `render` | Le rapport, sans aucun nom de dépôt privé. |
| `<output-dir>/stats-dark.svg`, `stats-light.svg` | `render` | La carte de statistiques par défaut. |
| `<output-dir>/<name>-dark.svg`, `<name>-light.svg` | `render` | Une paire par entrée de `render.cards`. |
| `<output-dir>/identity-dark.svg`, `identity-light.svg` | `render` | La carte identité, quand `profile` contient `focus` ou `tagline`, quand `profile.identity.show` est donné, ou quand `identity` est placé dans `render.rows`. |
| `<output-dir>/links/<kind>-dark.svg`, `<kind>-light.svg` | `render` | Un badge par lien de profil non vide. |
| `<output-dir>/<name>-dark.png`, `<name>-light.png` | `render` | Un PNG à côté de chaque SVG ci-dessus (cartes, carte identité, badges de `links/`), seulement quand l'input `png` ou `render.png` est activé. |
| `<output-dir>/README.snippet.md` | `render` | Le code à coller dans le README de votre profil. |

Le rapport lui-même est écrit dans le dossier temporaire du runner et renvoyé par l'output `report`. Gardez `<data-dir>` dans un dépôt privé et commitez `<output-dir>` dans le dépôt public. Voir [Confidentialité](./privacy.md).
