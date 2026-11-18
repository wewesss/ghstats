---
title: Builder
description: Construire une configuration ghstats dans le navigateur et voir les cartes produites.
sidebar:
  order: 1
---

# Builder

Le builder produit les mêmes cartes que l'Action, avec le même code Go compilé en WebAssembly, à partir d'un rapport d'exemple fixe. Changez les options à gauche, vérifiez les cartes à droite, puis copiez les fichiers en bas. L'interface du builder reste en anglais.

```demo builder
```

Ouvrez le [builder en ligne](https://wewesss.github.io/ghstats/fr/builder/) si vous lisez ceci sur GitHub : la démo ci-dessus ne fonctionne que sur le site.

## Utiliser les sorties

1. Enregistrez `ghstats.yaml` à la racine de votre dépôt de profil. L'Action et la CLI le lisent à cet endroit ; les clés sont dans [Configuration](../guide/configuration.md).
2. Ajoutez le workflow dans `.github/workflows/ghstats.yml` et remplacez `<user>` par votre identifiant, comme décrit dans [Prise en main](../guide/getting-started.md).
3. Collez `README.snippet.md` dans le README de votre profil. L'Action écrit le même fichier à chaque exécution, vous pouvez donc aussi le copier depuis le dossier `ghstats/` généré.

## Modifier le YAML à la main

La zone `ghstats.yaml` est éditable. Dès qu'elle diffère de ce que produit le formulaire, le formulaire est désactivé et l'aperçu suit le texte ; « Back to the form » ramène aux valeurs du formulaire. Une erreur de validation est affichée telle quelle au-dessus de l'aperçu, les dernières cartes valides restant en dessous.

L'aperçu utilise des données d'exemple : les chiffres ne sont pas les vôtres. La [Galerie](../cards/gallery.md) montre chaque variante.
