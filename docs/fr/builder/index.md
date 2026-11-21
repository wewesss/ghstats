---
title: Builder
description: Construire une configuration ghstats dans le navigateur, agencer le README sur l'aperçu et copier les fichiers.
sidebar:
  order: 1
---

Le builder produit les mêmes cartes que l'Action, avec le même code Go compilé en WebAssembly, à partir de données d'exemple, d'un identifiant GitHub ou d'un fichier `stats.json`. Il occupe toute la fenêtre : les réglages sont à gauche et défilent seuls, l'aperçu du README est à droite et ne bouge jamais pendant que vous modifiez. L'interface du builder reste en anglais.

```demo builder
```

Ouvrez le [builder en ligne](https://wewesss.github.io/ghstats/fr/builder/) si vous lisez ceci sur GitHub : la démo ci-dessus ne fonctionne que sur le site.

## Agencer le README

L'aperçu est le README tel que GitHub le dessine, construit à partir du snippet que l'Action écrit.

- Cliquez sur une carte, sur la carte d'identité ou sur un badge de lien pour le sélectionner : la section Selection des réglages n'affiche que ses options. Échap, Deselect ou un clic dans le vide annule la sélection.
- Glissez un élément avant ou après un autre pour le mettre dans la même ligne, ou sur une bande New row pour lui donner sa propre ligne. Une ligne de liens n'accepte que des liens ; un dépôt refusé est entouré de rouge.
- Avec un élément sélectionné, les boutons fléchés ou les touches fléchées le déplacent dans sa ligne et vers la ligne du dessus ou du dessous.
- Survolez une ligne, ou sélectionnez l'un de ses éléments, pour afficher ses boutons d'alignement à gauche, au centre et à droite.
- Sur une carte, Make it a card sort un bloc dans une nouvelle carte placée à côté.
- Avec un élément sélectionné, Download SVG et Download PNG l'enregistrent pour le mode d'aperçu courant ; Download all (.zip) regroupe chaque élément en SVG et en PNG, sombre et clair. Le PNG est dessiné sans animation en double taille, comme `ghstats render --png`, et son moteur ne se charge qu'au premier téléchargement PNG.

## Utiliser les sorties

Les onglets sous la barre d'outils contiennent les trois fichiers.

1. Enregistrez `ghstats.yaml` à la racine de votre dépôt de profil. L'Action et la CLI le lisent à cet endroit ; les clés sont dans [Configuration](../guide/configuration.md).
2. Ajoutez le workflow dans `.github/workflows/ghstats.yml` et remplacez `<user>` par votre identifiant, comme décrit dans [Prise en main](../guide/getting-started.md).
3. Collez `README.snippet.md` dans le README de votre profil. L'Action écrit le même fichier à chaque exécution, vous pouvez donc aussi le copier depuis le dossier `ghstats/` généré.

## Modifier le YAML à la main

L'onglet `ghstats.yaml` est éditable. Dès qu'il diffère de ce que produit le formulaire, le formulaire est désactivé et l'aperçu suit le texte ; « Back to the form » ramène aux valeurs du formulaire. Une erreur de validation est affichée telle quelle au-dessus de l'aperçu, les dernières cartes valides restant en dessous.

L'aperçu utilise des données d'exemple tant que vous ne chargez pas un identifiant ou un fichier : avec l'exemple, les chiffres ne sont pas les vôtres. La [Galerie](../cards/gallery.md) montre chaque variante.
