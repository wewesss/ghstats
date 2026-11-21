---
title: Prise en main
description: Installer ghstats sur le dépôt de votre profil, du token au premier lancement.
sidebar:
  order: 1
---

Cette page vous mène de zéro à des cartes publiées dans le README de votre profil.

## Installation rapide avec ghstats init

`ghstats init` configure le dépôt de votre profil depuis votre machine, par la CLI GitHub. Le token est la seule chose que vous créez à la main.

Avant de commencer :

1. Installez la [CLI GitHub](https://cli.github.com) et connectez-vous avec `gh auth login`.
2. Donnez-lui le scope `workflow`, nécessaire pour écrire sous `.github/workflows/` et que `gh auth login` n'accorde pas par défaut : `gh auth refresh -h github.com -s workflow`.
3. Installez ghstats avec `go install github.com/wewesss/ghstats/cmd/ghstats@latest`, ou téléchargez un binaire depuis la [page des releases](https://github.com/wewesss/ghstats/releases).
4. Créez le token fine-grained décrit dans [Créer le token](#1-créer-le-token). Donnez-lui accès à tous les dépôts, pour qu'il couvre aussi `<user>/ghstats-data`, que `init` crée.

Lancez ensuite la commande :

```sh
export GHSTATS_TOKEN=<token>
ghstats init
```

Sans `GHSTATS_TOKEN` dans l'environnement, `init` demande le token quand le secret manque ; la saisie est masquée dans un terminal. `init` enchaîne ces étapes dans l'ordre et affiche `ok`, `created`, `updated` ou `skipped` pour chacune :

1. Vérifie que `gh` est installé et possède le scope `workflow`, et lit votre login (remplaçable avec `--user`).
2. Vérifie que le dépôt de profil `<user>/<user>` existe, et le crée après confirmation s'il manque.
3. Vérifie que le dépôt de données privé `<user>/ghstats-data` existe, et le crée s'il manque. Il s'arrête sur une erreur si ce dépôt est public, même avec `--force`, car il contient des noms de dépôts privés.
4. Écrit `.github/workflows/ghstats.yml` et un `ghstats.yaml` minimal dans le dépôt de profil.
5. Ajoute le code des cartes au `README.md`, entre les marqueurs `<!-- ghstats:start -->` et `<!-- ghstats:end -->`, après l'avoir affiché et demandé une confirmation.
6. Vérifie que le token appartient à votre compte et peut lire `<user>/ghstats-data`, puis l'enregistre comme secret `GHSTATS_TOKEN` du dépôt de profil. Le token passe à `gh` par son entrée standard et n'est jamais affiché.
7. Lance le premier run du workflow et affiche l'URL de l'onglet Actions. Passez cette étape avec `--no-run`.

Relancer `init` est sans risque pour vos fichiers, mais pas sans effet. Un workflow et un `ghstats.yaml` existants sont laissés tels quels sauf avec `--force`. Seul le texte entre les marqueurs du README est remplacé, et rien n'est écrit quand il est déjà à jour. Le secret est laissé tel quel quand il existe et que `GHSTATS_TOKEN` n'est pas défini ; avec `GHSTATS_TOKEN` défini, il est réécrit. Sans `--no-run`, un nouveau run du workflow démarre à chaque fois. Les flags sont dans [Ligne de commande](./cli.md#init).

## Installation manuelle

Les étapes ci-dessous font à la main ce que fait `ghstats init`.

## Prérequis

- Un compte GitHub avec un dépôt de profil nommé `<user>/<user>`.
- Un dépôt privé nommé `<user>/ghstats-data`. Il contient les snapshots et l'historique de trafic, qui comportent des noms de dépôts privés : il doit donc rester privé. Voir [Confidentialité](./privacy.md).
- Un runner Linux ou macOS : l'Action est une Action composite pour ces runners.

## 1. Créer le token

Utilisez un token d'accès personnel fine-grained :

- Propriétaire de la ressource : votre compte, avec accès à tous les dépôts.
- Permissions des dépôts, en lecture seule : Metadata, Contents, Administration (nécessaire pour le trafic).
- En plus, Contents en lecture et écriture sur `<user>/ghstats-data`, pour que le workflow puisse y pousser les données.

Un token fine-grained ne couvre qu'un seul propriétaire : les dépôts d'une organisation n'ont donc aucune donnée de trafic. Utilisez un token par organisation, ou un token classique avec le scope `repo`.

## 2. Enregistrer le secret

Dans les paramètres du dépôt de votre profil, ouvrez Secrets and variables, Actions, et créez un secret de dépôt nommé `GHSTATS_TOKEN` contenant le token.

## 3. Ajouter le workflow

Créez `.github/workflows/ghstats.yml` dans le dépôt de votre profil. Il récupère le dépôt de données, lance l'Action, puis commite les données et les cartes :

```yaml
name: ghstats

on:
  schedule:
    - cron: "17 3 * * *"
  workflow_dispatch:

permissions:
  contents: write

jobs:
  stats:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - uses: actions/checkout@v7
        with:
          repository: <user>/ghstats-data
          path: ghstats-data
          token: ${{ secrets.GHSTATS_TOKEN }}

      - uses: wewesss/ghstats@v1
        with:
          token: ${{ secrets.GHSTATS_TOKEN }}

      - name: Commit the data repository
        working-directory: ghstats-data
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add -A
          if ! git diff --cached --quiet; then
            git commit -m "chore: update ghstats data"
            git push
          fi

      - name: Commit the cards
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add ghstats
          if ! git diff --cached --quiet; then
            git commit -m "chore: update ghstats cards"
            git push
          fi
```

Remplacez `<user>` par votre login. `wewesss/ghstats@v1` suit la dernière version v1.x.y ; voir [Action GitHub](./action.md) pour tous les inputs.

L'Action fonctionne sans fichier de configuration. Pour changer les blocs, le thème ou les cartes de profil, ajoutez un `ghstats.yaml` à la racine du dépôt de profil ; les clés sont dans [Configuration](./configuration.md).

## 4. Référencer les cartes dans le README du profil

L'Action écrit les cartes dans `ghstats/` et les commite. Collez ceci dans le README de votre profil :

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ghstats/stats-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="ghstats/stats-light.svg">
  <img alt="GitHub stats" src="ghstats/stats-dark.svg">
</picture>
```

L'Action écrit aussi `ghstats/README.snippet.md` avec le code prêt à coller ; avec une section `profile`, il inclut la carte identité et les badges de liens. Voir [Cartes de profil](../cards/profile.md).

## 5. Lancer une première fois

Ouvrez l'onglet Actions du dépôt de profil, sélectionnez le workflow `ghstats` et choisissez Run workflow. Quand le run est vert, `ghstats/` apparaît dans le dépôt et la carte s'affiche sur votre profil. Ensuite le workflow se lance chaque jour à 03:17 UTC.

Si les chiffres de trafic restent vides, le token ne peut pas lire le trafic d'un des propriétaires : le run affiche une indication, et la [section sur le token](#1-créer-le-token) ci-dessus explique comment couvrir les organisations.
