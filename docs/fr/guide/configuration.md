---
title: Configuration
description: Toutes les clés de ghstats.yaml, avec leurs valeurs autorisées et leurs défauts.
sidebar:
  order: 3
---

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
| `distribution.hide_partial` | `false` |

`distribution.hide_partial` : Affiche n/a sur les cartes au lieu d'un total de trafic auquel manquent des dépôts (fiabilité `partial`). stats.json garde le chiffre.

## render

L'apparence des cartes. Voir la [galerie](../cards/gallery.md) pour chaque bloc et chaque variante.

| Clé | Valeurs autorisées |
| --- | --- |
| `render.blocks` | Liste de `block: [variantes]`, empilés dans cet ordre. `stats` : `totals`, `distribution`. `languages` : `bars`, `iso`, `cells`, `treemap`, `shift`, `terminal`. `history` : `views`, `clones`. |
| `render.cards` | Liste facultative de cartes, chacune avec `name`, `blocks` et un `layout` facultatif. Découpe le rendu en plusieurs images ; `render.blocks` est alors ignoré. Voir [Plusieurs cartes](#plusieurs-cartes). |
| `render.rows` | Liste facultative de lignes du README, chacune avec un `align` et les `items` qu'elle contient. Règle l'agencement des cartes dans `README.snippet.md`. Voir [Agencement dans le README](#agencement-dans-le-readme). |
| `render.private_bar` | `true` (défaut) ou `false`. Remplace les noms des dépôts privés par une barre noire. |
| `render.languages.colors` | `accent` (défaut), `github`, `mono`, `spectrum`, `duotone`, ou une table de langage vers `"#rrggbb"`. |
| `render.languages.weighting` | `true` (défaut) ou `false`. `false` masque toute trace de la pondération des langages : l'étiquette `×<n>` à côté d'un langage qui a un poids, l'étiquette `adjusted` / `raw` en haut à droite des blocs de langages, la bande RAW / ADJ sous la variante `bars` (la carte est plus courte) et l'option `--adjusted` de la commande `terminal`. Les parts ne changent pas. Avec `false`, lister la variante `shift` est une erreur, car elle montre la pondération. |
| `render.theme.preset` | `noir` (défaut), `carbon`, `graphite`, `midnight`, `slate`, `dune`, `transparent`. |
| `render.theme.accent` | `ice` (défaut), `mint`, `amber`, `silver`, `violet`, `rose`, `lime`, `coral`, `cycle`, ou `{dark: "#rrggbb", light: "#rrggbb"}`. |
| `render.theme.overrides` | Clés `bg`, `border`, `fg`, `muted`, `faint` en `"#rrggbb"` ; `bg` accepte aussi `none`. |
| `render.motion` | `true` (défaut) ou `false`. `false` coupe toutes les animations. Le mouvement réduit est toujours respecté. |
| `render.png` | `false` (défaut) ou `true`. `true` écrit aussi `<nom>-dark.png` et `<nom>-light.png` à côté de chaque SVG (les cartes, la carte identité et les badges de liens), rendus en 2x sans aucune animation. Le drapeau `--png` de `ghstats render` et l'entrée `png` de l'Action font de même. |
| `render.layout` | `card` (495 px, défaut) ou `wide` (1000 px, deux colonnes de 495 px). Avec `languages`, il va dans la colonne de droite et les autres blocs à gauche ; sans lui, les blocs sont répartis pour équilibrer les deux colonnes. Une carte `wide` à un seul bloc est dessinée en 495 px. |
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

### Plusieurs cartes

Sans `render.cards`, `render.blocks` donne une seule carte, `stats-dark.svg` et `stats-light.svg`. Avec `render.cards`, chaque carte est sa propre paire d'images et regroupe librement des blocs et des variantes :

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

Ce fichier écrit `overview-dark.svg`, `overview-light.svg`, `langs-dark.svg` et `langs-light.svg`. `stats.json` et les badges de liens ne changent pas, et la carte identité garde son nom.

- `name` est obligatoire, unique, en minuscules, chiffres et `-` (1 à 40 caractères, commençant par une lettre ou un chiffre). `identity` et `links` sont réservés.
- `blocks` suit les mêmes règles que `render.blocks`, avec au moins un bloc par carte. Un bloc peut apparaître dans plusieurs cartes.
- `layout` vaut `card` ou `wide`. Absent, la carte hérite de `render.layout`. Une carte `wide` à un seul bloc est dessinée comme une `card`.
- `header` vaut `true` (défaut) ou `false`. `false` retire la ligne `login / ghstats`, la date de synchronisation et le trait en dessous, et la carte est plus courte de 40 px.
- 12 cartes au plus.
- Quand `render.cards` est défini, `render.blocks` est ignoré.

`README.snippet.md` liste une balise `<picture>` par carte, dans l'ordre, de largeur `495` pour un layout `card` et `100%` pour un layout `wide`. Le texte alternatif est `GitHub stats: <name>` :

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

Une carte par variante :

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

### Agencement dans le README

Sans `render.rows`, `README.snippet.md` empile une image par élément : la carte identité, chaque carte dans l'ordre, puis les liens. Avec `render.rows`, chaque ligne devient un `<p align>` et contient un ou plusieurs éléments côte à côte :

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

Le snippet devient :

```html
<p align="center">
  <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/langs-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/langs-light.svg"><img alt="GitHub stats: langs" src="ghstats/langs-dark.svg" width="49%"></picture> <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/views-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/views-light.svg"><img alt="GitHub stats: views" src="ghstats/views-dark.svg" width="49%"></picture>
</p>

<p align="right">
  <picture><source media="(prefers-color-scheme: dark)" srcset="ghstats/overview-dark.svg"><source media="(prefers-color-scheme: light)" srcset="ghstats/overview-light.svg"><img alt="GitHub stats: overview" src="ghstats/overview-dark.svg" width="98%"></picture>
</p>
```

GitHub garde `align` sur `<p>` et `width` sur `<img>` : les deux cartes de la première ligne se partagent la ligne.

- `align` vaut `left`, `center` ou `right` ; `center` par défaut.
- `items` liste des noms de cartes, `identity`, `links` et `links/<kind>` (le badge d'un seul lien, voir [Placer les badges de liens](#placer-les-badges-de-liens)). Un élément se place une seule fois dans l'ensemble des lignes. Une ligne qui contient un élément de lien ne contient que des éléments de lien.
- Quand une ligne a plusieurs éléments, chaque `width` est un pourcentage proportionnel aux largeurs naturelles de la ligne (495 pour une `card`, 1000 pour la carte identité et une carte `wide`), avec 2 % laissés libres. Un élément seul dans sa ligne est dessiné à 98 % quand c'est une carte `wide` ou la carte identité, et garde 495 quand c'est une `card`.
- Les cartes d'une même ligne sont dessinées à la hauteur de la plus haute : le cadre et le fond sont prolongés, l'en-tête reste en haut et les blocs sont centrés verticalement dans l'espace ajouté. La carte identité garde sa hauteur.
- `links` sans lien et un `links/<kind>` dont le lien n'existe pas ou est vide sont ignorés, et une ligne devenue vide n'est pas écrite. `identity` placé dans une ligne dessine toujours la carte identité (voir [Carte identité](#carte-identité)).
- Les éléments absents de toutes les lignes sont ajoutés après les lignes, dans l'ordre par défaut : identité, cartes, liens, avec les largeurs habituelles (495 pour une `card`, `100%` pour une carte `wide`, 495 pour une carte identité en disposition `card`, `100%` pour les autres cartes identité). Les liens qu'aucune ligne ne place sont ajoutés en dernier dans une ligne centrée.
- 12 lignes au plus.

### Placer les badges de liens

`links/<kind>` place le badge d'un seul lien, pour n'importe quel type de lien du profil. `links` place tous les badges qu'aucun `links/<kind>` ne place ailleurs. Les badges d'une ligne sont écrits l'un après l'autre dans un seul `<p align>` : une ligne contient des badges seulement, jamais des badges avec des cartes.

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

Le snippet devient :

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

La carte identité fait ici 495 de large (`identity.layout: card`) : elle partage la première ligne avec la carte de statistiques, à 49 % chacune.

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
| `profile.link_style` | Apparence de chaque bouton de lien : `full` (par défaut, icône et libellé), `icon` (icône seule) ou `text` (libellé seul). Un lien peut le remplacer par son propre `style`. |
| `profile.identity.show` | Parties de la carte identité à dessiner, parmi `header`, `name`, `tagline`, `location`, `note`, `building`, `focus` et `footer`. Par défaut : toutes celles qui ont une valeur. |
| `profile.identity.layout` | `wide` (par défaut, 1000 px) ou `card` (495 px, tout dans une seule colonne). |

La carte identité est générée quand `profile` contient au moins `focus` ou `tagline`, quand `profile.identity.show` est donné, ou quand `identity` est placé dans `render.rows`.

### Carte identité

Sans `profile.identity`, la carte est dessinée comme avant. `identity.show` liste les parties à dessiner ; une partie absente de la liste n'est pas dessinée, la hauteur de la carte suit les parties restantes et aucun trou n'est laissé :

- `header` : le petit nom, `/ profile` et le point qui pulse, avec le trait en dessous.
- `name` : le grand nom.
- `tagline`, `note` : les lignes du même nom.
- `location` : le lieu, à droite de la bande du haut. La bande, le point et le trait n'existent que si `header` est dessiné ; sans `header`, le lieu se place en haut à droite de la première ligne de contenu.
- `building` : la ligne `$ building` avec les mots qui tournent de `focus`.
- `focus` : la liste numérotée du focus. Sans elle, la colonne de droite et son trait vertical disparaissent.
- `footer` : le trait et `github.com/<login>` en bas.

Une partie sans valeur (une `tagline` non renseignée, par exemple) n'est pas dessinée. Une partie inconnue ou répétée est une erreur de validation.

`identity.layout: card` dessine la carte en 495 de large sur une seule colonne : en-tête, nom (sa taille est réduite pour tenir en 447 px), tagline, ligne `building`, note, puis la liste du focus sous le texte, puis le pied de page. Dans `render.rows` et dans le snippet, cette carte compte pour 495 de large.

```yaml
profile:
  name: octocat
  location: caen, fr
  identity:
    show: [name, location]
    layout: card
```

### Liens

Types de liens : `youtube`, `twitch`, `kick`, `discord`, `tiktok`, `instagram`, `x`, `bluesky`, `threads`, `mastodon`, `reddit`, `linkedin`, `steam`, `itchio`, `website`, `github_sponsors`, `open_collective`, `buy_me_a_coffee`, `ko_fi`, `patreon`, `custom`. Chaque type ne peut être listé qu'une fois.

Une valeur est un identifiant (le `@` initial est facultatif) ou une URL `https://` complète ; `mastodon`, `steam` et `website` exigent une URL complète. La forme `{url, label, style}` remplace le libellé et le style :

```yaml
profile:
  links:
    - youtube: "@octocat"
    - website: https://example.com
    - discord: {url: "https://discord.com/invite/xyz", label: discord}
```

Le type `custom` est un bouton vers n'importe quelle URL et n'existe qu'en forme map `{url, label, icon}` : `url` (une URL `https://` complète) et `label` sont obligatoires, `icon` est facultatif. `icon` est soit le nom d'un type connu (`youtube`, `discord`...), qui réutilise son icône embarquée, soit un tracé SVG commençant par `M` ou `m` (l'attribut `d` d'un `<path>` en viewBox 24x24 ; `<`, `>`, `"` et `&` sont refusés). Sans `icon`, le bouton porte un globe.

```yaml
profile:
  links:
    - custom: {url: "https://example.com/blog", label: my blog, icon: youtube}
```

La forme scalaire `custom: https://...`, un `url` ou un `label` manquant, un `icon` sur un autre type et une icône qui n'est ni un nom de type ni un tracé valide sont des erreurs de validation.

### Style des liens

`profile.link_style` vaut `full` (par défaut : icône et libellé), `icon` (un carré avec l'icône seule ; le libellé reste dans `aria-label` et `<title>`) ou `text` (le libellé seul). Toute autre valeur est une erreur de validation qui liste les trois.

```yaml
profile:
  link_style: icon
```

Un lien écrit en forme map peut porter son propre `style`, qui remplace `link_style` pour ce lien seulement :

```yaml
profile:
  link_style: full
  links:
    - youtube: "@octocat"
    - website: {url: "https://example.com", style: icon}
```

Voir [Cartes de profil](../cards/profile.md#styles-de-bouton) pour le rendu de chaque style.

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
  png: false
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
