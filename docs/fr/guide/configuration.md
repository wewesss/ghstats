---
title: Configuration
description: Toutes les clés de ghstats.yaml, avec leurs valeurs autorisées et leurs défauts.
sidebar:
  order: 3
---

# Configuration

ghstats lit un seul fichier YAML, `ghstats.yaml` par défaut. Toutes les clés sauf `version` sont facultatives, et un fichier vide donne les valeurs par défaut. Les clés inconnues et les valeurs invalides sont refusées avec un message qui nomme la clé. Le fichier complet et annoté est [`config/example.yaml`](../../../config/example.yaml), et un exemple complet clôt cette page.

```yaml
version: 1
```

`version` doit valoir `1`.

## scope

Quels dépôts sont comptés.

| Clé | Défaut | Description |
| --- | --- | --- |
| `scope.owners` | vide | Ne compter que les dépôts de ces propriétaires (insensible à la casse). Vide : tous les propriétaires renvoyés par la collecte. |
| `scope.include_forks` | `false` | Compter les forks. |
| `scope.include_archived` | `false` | Compter les dépôts archivés. |
| `scope.include_private` | `true` | Compter les dépôts privés dans les agrégats. Leurs noms ne sont jamais écrits. |
| `scope.exclude_repos` | vide | `owner/name` ou un glob comme `owner/*-old` (insensible à la casse). |
| `scope.repo_weights` | vide | Table de `owner/name` vers un multiplicateur (>= 0) appliqué à tous les langages de ce dépôt, en mode adjusted uniquement. |

## languages

Comment les parts de langages sont calculées et affichées.

| Clé | Défaut | Description |
| --- | --- | --- |
| `languages.mode` | `raw` | `raw` ou `adjusted`. |
| `languages.weights` | vide | Table de langage vers un multiplicateur (>= 0, 1 par défaut, noms insensibles à la casse). |
| `languages.exclude` | vide | Langages retirés du calcul ajusté, comme un poids de 0. Les parts brutes ne sont pas touchées. |
| `languages.hide` | vide | Langages retirés de l'affichage uniquement. |
| `languages.hidden_share` | `other` | Ce que devient la part des langages masqués : `other` la fusionne dans l'entrée Other, `renormalize` remet les parts visibles à l'échelle pour que leur somme vaille 1. |
| `languages.max_visible` | `6` | Langages affichés avant de regrouper le reste dans Other. `0` pour illimité. |
| `languages.other_label` | `Other` | Libellé de l'entrée regroupée. Ne doit pas être vide. |

### Brut et ajusté

En mode `raw`, les parts sont les octets de code que GitHub rapporte pour chaque langage, tels quels. En mode `adjusted`, chaque langage est d'abord multiplié par son poids dans `languages.weights`, et chaque dépôt par son poids dans `scope.repo_weights`, puis les parts sont calculées. Servez-vous-en pour réduire un langage surtout généré ou importé (un poids de `0.35` compte 35 pour cent des octets) ou pour l'écarter entièrement (`0`).

`stats.json` garde côte à côte les parts brutes et les parts ajustées.

## distribution

Interrupteurs des entrées `distribution` du rapport : le nombre de téléchargements des assets de release, et le trafic de clones et de vues.

| Clé | Défaut |
| --- | --- |
| `distribution.release_assets` | `true` |
| `distribution.clones` | `true` |
| `distribution.views` | `true` |

## render

L'apparence des cartes. Voir la [galerie](../cards/gallery.md) pour chaque bloc et chaque variante.

| Clé | Valeurs autorisées |
| --- | --- |
| `render.blocks` | Liste de `block: [variantes]`, empilés dans cet ordre. `stats` : `totals`, `distribution`. `languages` : `bars`, `iso`, `cells`, `treemap`, `shift`, `terminal`. `history` : `views`, `clones`. |
| `render.private_bar` | `true` (défaut) ou `false`. Remplace les noms des dépôts privés par une barre noire. |
| `render.languages.colors` | `accent` (défaut), `github`, `mono`, `spectrum`, `duotone`, ou une table de langage vers `"#rrggbb"`. |
| `render.theme.preset` | `noir` (défaut), `carbon`, `graphite`, `midnight`, `slate`, `dune`, `transparent`. |
| `render.theme.accent` | `ice` (défaut), `mint`, `amber`, `silver`, `violet`, `rose`, `lime`, `coral`, `cycle`, ou `{dark: "#rrggbb", light: "#rrggbb"}`. |
| `render.theme.overrides` | Clés `bg`, `border`, `fg`, `muted`, `faint` en `"#rrggbb"` ; `bg` accepte aussi `none`. |
| `render.motion` | `true` (défaut) ou `false`. `false` coupe toutes les animations. Le mouvement réduit est toujours respecté. |
| `render.layout` | `card` (495 px, défaut) ou `wide` (1000 px, `languages` dans une colonne de droite, les autres blocs empilés à gauche). |
| `render.base_path` | Préfixe de chaque chemin écrit dans `README.snippet.md`, terminé par `/` (défaut `ghstats/`). |

### Blocs et variantes

Chaque bloc apparaît une seule fois. Un bloc à une variante l'affiche fixe ; un bloc à plusieurs variantes les fait alterner sur place :

```yaml
render:
  blocks:
    - stats: [totals, distribution]
    - languages: [bars, treemap]
    - history: [views]
```

### Thème

`preset` fixe les couleurs de fond et de texte, `accent` la couleur d'accent. `cycle` fait changer l'accent au fil du temps, synchronisé sur toutes les cartes. Un accent personnalisé exige `dark` et `light`. `overrides` remplace des couleurs isolées du thème :

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

`render.languages.colors` est un nom de palette ou votre propre table. Les langages absents d'une table utilisent leur couleur GitHub.

```yaml
render:
  languages:
    colors: {TypeScript: "#3178c6", Go: "#00add8"}
```

## profile

Une section facultative qui ajoute une carte identité et un badge par lien. Tout suit `render.theme`. Voir [Cartes de profil](../cards/profile.md) pour les fichiers produits.

| Clé | Description |
| --- | --- |
| `profile.name` | Nom affiché sur la carte. Par défaut : le login du rapport. |
| `profile.tagline` | Une ligne sous le nom. |
| `profile.location` | Facultatif. |
| `profile.note` | Facultatif. |
| `profile.focus` | 1 à 6 entrées, chacune `{name, building}` où `building` contient 1 à 4 mots. |
| `profile.links` | Liste de `kind: valeur`, gardée dans cet ordre ; une valeur vide est ignorée. |

La carte identité est générée quand `profile` contient au moins `focus` ou `tagline`.

### Liens

Types de liens : `youtube`, `twitch`, `kick`, `discord`, `tiktok`, `instagram`, `x`, `bluesky`, `threads`, `mastodon`, `reddit`, `linkedin`, `steam`, `itchio`, `website`, `github_sponsors`, `open_collective`, `buy_me_a_coffee`, `ko_fi`, `patreon`. Chaque type ne peut être listé qu'une fois.

Une valeur est un identifiant (le `@` initial est facultatif) ou une URL `https://` complète ; `mastodon`, `steam` et `website` exigent une URL complète. La forme `{url, label}` remplace le libellé :

```yaml
profile:
  links:
    - youtube: "@octocat"
    - website: https://example.com
    - discord: {url: "https://discord.com/invite/xyz", label: discord}
```

## Exemple complet

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
