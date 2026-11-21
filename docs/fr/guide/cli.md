---
title: Ligne de commande
description: Les commandes ghstats, leurs flags, les variables d'environnement et les codes de sortie.
sidebar:
  order: 4
---

L'Action GitHub lance ces commandes pour vous. Utilisez la CLI directement pour essayer une configuration en local ou pour faire tourner ghstats en dehors de GitHub Actions.

## Installation

Installez avec Go, ou téléchargez une archive de release depuis la [page des releases](https://github.com/wewesss/ghstats/releases) :

```sh
go install github.com/wewesss/ghstats/cmd/ghstats@latest
```

## Synopsis

```sh
ghstats collect [--out data/snapshots] [--history data/history.json] [--no-traffic] [--concurrency N] [--keep N]
ghstats compute --snapshot <file> --history <file> --out <report.json> [--config <file>]
ghstats render  --report <report.json> --out-dir <dir> [--config <file>]
ghstats init    [--user <login>] [--yes] [--force] [--no-run]
ghstats version
```

`ghstats <command> -h` affiche les flags d'une commande.

## collect

Récupère les dépôts, langages, releases et le trafic dans un snapshot.

| Flag | Défaut | Description |
| --- | --- | --- |
| `--out` | `data/snapshots` | Dossier qui reçoit le fichier snapshot. |
| `--history` | `data/history.json` | Fichier d'historique de trafic, mis à jour sur place. |
| `--no-traffic` | désactivé | Saute les appels à l'API de trafic. |
| `--concurrency` | `0` | Requêtes d'API en parallèle ; `0` utilise la valeur par défaut. |
| `--keep` | `0` | Snapshots conservés dans `--out` ; `0` les garde tous. |

`collect` écrit `snapshot-<timestamp>.json` dans `--out`, le fusionne dans le fichier d'historique et affiche le chemin du snapshot. Avec `--keep N` (N > 0), seuls les N fichiers `snapshot-*.json` les plus récents de `--out` sont conservés.

## compute

Transforme un snapshot et un historique en `report.json`.

| Flag | Requis | Description |
| --- | --- | --- |
| `--snapshot` | oui | Fichier snapshot. |
| `--history` | oui | Fichier d'historique de trafic ; un fichier absent compte comme vide. |
| `--out` | oui | Fichier de rapport à écrire. |
| `--config` | non | Fichier de configuration ; les valeurs par défaut s'appliquent s'il est omis. |

Les avertissements sont affichés sur la sortie d'erreur.

## render

Écrit les fichiers d'un rapport.

| Flag | Requis | Description |
| --- | --- | --- |
| `--report` | oui | Fichier de rapport. |
| `--out-dir` | oui | Dossier qui reçoit les fichiers. |
| `--config` | non | Fichier de configuration ; les valeurs par défaut s'appliquent s'il est omis. |

`render` écrit `stats.json`, `stats-dark.svg` et `stats-light.svg` dans `--out-dir`, plus la carte identité et les badges de liens quand la configuration a une section `profile` (voir [Cartes de profil](../cards/profile.md)), et `README.snippet.md`. Avec `render.cards`, les fichiers de statistiques sont `<name>-dark.svg` et `<name>-light.svg`, une paire par carte (voir [Plusieurs cartes](./configuration.md#plusieurs-cartes)).

## init

Configure un dépôt de profil depuis votre machine, par la CLI GitHub (`gh`), qui doit être installée et connectée avec le scope `workflow`. Les étapes et les prérequis sont dans [Prise en main](./getting-started.md#installation-rapide-avec-ghstats-init).

| Flag | Défaut | Description |
| --- | --- | --- |
| `--user` | compte de `gh` | Login GitHub. |
| `--yes` | désactivé | Répond oui aux confirmations (README, création du dépôt de profil). |
| `--force` | désactivé | Réécrit le workflow et `ghstats.yaml` même s'ils existent. |
| `--no-run` | désactivé | Ne lance pas le premier run. |

Chaque étape affiche `ok` (déjà en place), `created`, `updated` ou `skipped` : relancer `init` ne modifie rien de ce qui est déjà configuré. `init` s'arrête sur une erreur quand `<user>/ghstats-data` est public, même avec `--force`.

## version

Affiche la version.

## Variables d'environnement

`ghstats collect` lit le token dans `GHSTATS_TOKEN`, puis dans `GITHUB_TOKEN`, et échoue si aucune n'est définie. Les permissions nécessaires au token sont dans [Prise en main](./getting-started.md#1-créer-le-token).

`ghstats init` lit `GHSTATS_TOKEN` et l'enregistre comme secret `GHSTATS_TOKEN` du dépôt de profil, après avoir vérifié qu'il appartient à `<user>` et peut lire `<user>/ghstats-data`. Quand la variable est vide et que le secret manque, `init` demande le token ; quand le secret existe déjà, il ne demande rien. `GHSTATS_GRAPHQL_URL` et `GHSTATS_REST_URL` changent les points d'accès de l'API utilisés pour cette vérification.

## Codes de sortie

| Code | Signification |
| --- | --- |
| `0` | Succès. |
| `1` | Erreur d'exécution (pour `init` : `gh` absent, un scope manquant, un dépôt de données public, un token refusé). |
| `2` | Erreur d'usage. |

## Exemple

```sh
export GHSTATS_TOKEN=<token>
snapshot="$(ghstats collect --out data/snapshots --history data/history.json --keep 30)"
ghstats compute --snapshot "$snapshot" --history data/history.json --out report.json --config ghstats.yaml
ghstats render --report report.json --out-dir ghstats --config ghstats.yaml
```
