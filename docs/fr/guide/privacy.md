---
title: Confidentialité
description: Ce qui contient des noms de dépôts privés, ce qui n'en contient pas, et la limite du trafic des organisations.
sidebar:
  order: 5
---

## Ce qui est privé

Les snapshots bruts contiennent des noms de dépôts privés. Gardez-les, ainsi que le fichier d'historique, dans un dépôt privé uniquement. C'est pourquoi le workflow de [Prise en main](./getting-started.md) récupère un dépôt privé distinct `<user>/ghstats-data` dans `data-dir` et y commite `snapshots/` et `history.json`.

Les fichiers `stats.json`, SVG et PNG générés ne contiennent jamais de nom de dépôt privé : le dossier de sortie peut donc être commité dans un dépôt de profil public.

| Chemin | Noms de dépôts privés | Où le garder |
| --- | --- | --- |
| `<data-dir>/snapshots/` | oui | dépôt privé |
| `<data-dir>/history.json` | oui | dépôt privé |
| `<output-dir>/` (`stats.json`, fichiers SVG et PNG, `README.snippet.md`) | non | dépôt de profil public |

## Dépôts privés dans les chiffres

Les dépôts privés sont comptés dans les agrégats quand `scope.include_private` vaut `true` (le défaut) ; mettez `false` pour les écarter. Leurs noms ne sont jamais écrits. Quand `render.private_bar` vaut `true` et qu'au moins un dépôt privé est compté, la variante `totals` de la carte stats ajoute une ligne `PRIVATE <n>`, quatre barres pleines et `names hidden`. Aucun nom n'est dans le rapport : les barres ne remplacent rien.

## Trafic des organisations

Le trafic exige un accès en écriture avec un token qui couvre le propriétaire de chaque dépôt. Un token fine-grained ne couvre qu'un seul propriétaire : les dépôts d'une organisation n'ont donc aucune donnée de trafic, et `collect` affiche une indication quand GitHub répond 403. Utilisez un token par organisation, ou un token classique avec le scope `repo`.

## Logos de marque

Les icônes des badges viennent de [Simple Icons](https://simpleicons.org) (CC0 1.0). Les logos de marque restent la propriété de leurs détenteurs respectifs.
