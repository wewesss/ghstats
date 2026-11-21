# FAQ : ghstats

> _English version : [FAQ.md](./FAQ.md)_

Questions fréquentes sur la conception et les compromis. Si quelque chose ici est faux ou peu clair, ouvrez une issue ; je préfère corriger plutôt que laisser une réponse périmée.

---

## Installation et confidentialité

### Pourquoi les données brutes vivent-elles dans un dépôt privé ?

Les snapshots et l'historique de trafic contiennent des noms de dépôts privés. Les fichiers rendus (`stats.json`, les cartes SVG, les badges de liens, `README.snippet.md`) n'en contiennent jamais, donc seul le dossier de sortie est commité dans le dépôt de profil public. Le workflow de [Prise en main](./docs/fr/guide/getting-started.md) récupère `<user>/ghstats-data` dans `data-dir` et y pousse. Voir [Confidentialité](./docs/fr/guide/privacy.md).

### Pourquoi un token fine-grained ne donne-t-il pas le trafic de mes organisations ?

Un token fine-grained couvre un seul propriétaire de ressources. Les dépôts d'une organisation sont hors de ce propriétaire, donc GitHub répond 403 sur leurs points d'accès de trafic ; `collect` affiche une indication et continue avec les autres chiffres. Utilisez un token par organisation, ou un token classique avec le scope `repo`. Le trafic demande aussi la permission Administration en lecture seule, c'est pourquoi la section du token de [Prise en main](./docs/fr/guide/getting-started.md#1-créer-le-token) la liste.

### Combien l'Action coûte-t-elle en appels d'API ?

Une requête GraphQL paginée liste les dépôts avec leurs langages et leurs versions récentes (50 dépôts par page), et un dépôt qui a beaucoup de versions obtient des pages en plus. Le trafic ajoute deux appels REST par dépôt, un pour les vues et un pour les clones. Une exécution qui épuise la limite de débit s'arrête avec une erreur au lieu d'écrire des données partielles. Mettez `traffic: false` sur l'Action pour sauter les appels de trafic.

## Les chiffres

### Pourquoi n'y a-t-il plus de tuile `n/a` pour les archives de sources ?

GitHub n'expose aucun compteur de téléchargements pour les archives de sources, donc le chiffre ne peut jamais être rempli. Le rapport liste toujours le canal `source_archives` comme indisponible, avec une note qui dit pourquoi, mais la variante `distribution` de la carte stats montre désormais les vues cumulées (`VIEWS TOTAL`) à la place d'une tuile qui ne pouvait que dire `n/a`.

### Quelle différence entre langages bruts et ajustés ?

`raw` montre les octets de code que GitHub rapporte pour chaque langage, tels quels. `adjusted` multiplie chaque langage par son poids dans `languages.weights` et chaque dépôt par son poids dans `scope.repo_weights` avant de calculer les parts, ce qui permet de compter moins un langage surtout généré ou vendorisé. `stats.json` garde les deux séries de parts côte à côte. Voir [Configuration](./docs/fr/guide/configuration.md#brut-et-ajusté).

### Pourquoi TypeScript monte-t-il quand j'exclus HTML ?

Les parts totalisent toujours 100 pour cent. `languages.exclude` retire un langage du calcul ajusté, comme le ferait un poids de 0, donc le total diminue et chaque autre langage prend une part plus grande de ce total. Rien n'a changé pour TypeScript, seul le dénominateur a changé. `exclude` n'a aucun effet en mode `raw`, et `languages.hide` est différent : il ne retire un langage que de l'affichage.

### Pourquoi mes chiffres de trafic dépassent-ils les 14 derniers jours ?

GitHub conserve 14 jours de vues et de clones. ghstats fusionne les comptes journaliers de chaque exécution dans `history.json`, donc l'historique dépasse cette fenêtre tant que le workflow continue de tourner. Un jour qu'aucune exécution n'a vu avant qu'il sorte de la fenêtre de 14 jours est perdu pour de bon, donc lancez-le au moins tous les 14 jours ; le cron quotidien des exemples laisse beaucoup de marge.

## Fonctionnement

### Combien de snapshots sont conservés, et l'élagage perd-il de l'historique ?

L'entrée `keep-snapshots` vaut `30` par défaut, et `0` garde tous les snapshots. L'élagage ne supprime que les anciens fichiers de `snapshots/` ; l'historique de trafic vit dans `history.json`, fusionné à chaque exécution et jamais élagué.

### Pourquoi mon workflow planifié a-t-il cessé de tourner ?

GitHub désactive les workflows planifiés d'un dépôt public après 60 jours sans activité sur le dépôt. Le workflow ne commite les cartes que lorsqu'elles changent, donc une période calme peut laisser le dépôt sans commit récent. Si les exécutions s'arrêtent, ouvrez l'onglet Actions, réactivez le workflow et lancez-le une fois avec Run workflow.

## Les cartes

### Que fait l'accent `cycle`, et le mouvement réduit ?

`cycle` fait changer l'accent au fil du temps, synchronisé sur toutes les cartes. Quand le lecteur demande le mouvement réduit, ou quand vous mettez `render.motion: false`, toute animation s'arrête et la carte garde sa première couleur d'accent. Les badges sont statiques dans tous les cas.

### Pourquoi le README de mon profil montre-t-il une ancienne carte après l'exécution du workflow ?

GitHub passe par un proxy et met en cache les images affichées dans un README, donc une nouvelle carte peut mettre quelques minutes à apparaître, et un navigateur peut garder l'ancienne. Rechargez la page sans le cache. Si l'ancienne carte reste, vérifiez que le workflow a bien commité les nouveaux fichiers du dossier de sortie.

### Puis-je faire mon propre front ou serveur ?

Oui. `stats.json` est public et lisible depuis n'importe quelle page web, les paquets Go et le moteur WASM sont documentés, et un petit exemple de serveur est fourni. Voir [Faire son propre front](./docs/fr/guide/extend.md).
