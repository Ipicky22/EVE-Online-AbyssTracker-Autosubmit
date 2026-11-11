# EVE Online AbyssTracker Autosubmit

Envoie tout seul les runs Abyss d'EVE Online sur [Abyss Tracker](https://abysstracker.com).
Le programme s'appelle `abyssup`.
Il ne touche jamais au client EVE : il lit la position du personnage via l'API EVE (ESI) et le presse-papiers Windows.

## Usage

1. Lancer `abyssup.exe` : la page `http://localhost:47813` s'ouvre.
2. Choisir météo, tier, coque, fit, puis « Démarrer la session ».
3. Copier la soute (`Ctrl+A`, `Ctrl+C`) avant le premier run.
4. Après chaque run, recopier la soute : le run part sur Abyss Tracker.
5. Après un passage en station, recopier la soute avant le run suivant.

Les runs sont publiés en public.
Un run sans copie de soute fiable n'est pas envoyé : il est rangé dans `incomplete/`.

## Installation (une fois)

1. Créer un jeton sur `abysstracker.com/my-account/personal-access-tokens`.
2. Créer une application sur `developers.eveonline.com` :
   callback `http://localhost:47813/callback`, scope `esi-location.read_location.v1`.
3. Dans la page, ouvrir « Réglages », coller le jeton et l'identifiant de l'application, enregistrer.
4. Cliquer « Connecter le personnage EVE ».

## Options

- `-dry-run` : écrit les runs dans le journal au lieu de les envoyer.
- `-no-open` : n'ouvre pas le navigateur au lancement.

## Données

Dans `%APPDATA%\abyssup\` (`~/.config/abyssup/` sous Linux) :
`config.json`, `token.json`, `names.json` (cache des noms d'objets), `abyssup.log`,
et les dossiers `pending/`, `sent/`, `incomplete/`, `rejected/`.

## Développement

```
go test ./...
GOOS=windows go build -ldflags "-H=windowsgui" -o abyssup.exe .
```

## État des vérifications

Vérifié avec un vrai jeton (2026-11-11) :

- le jeton passe dans l'en-tête `X-API-KEY` ;
- `fits/list` rend les fits du compte avec leur vaisseau ;
- `POST /v1/abyss-tracker/telemetry` répond `Error: false` avec un `RunId`.

**Bloquant** : les deux runs de test ainsi créés (`PRIVATE` puis `PUBLIC`) n'apparaissent ni sur le site,
ni dans `get-runs`, ni dans `runs/{id}`. Tant que la cause n'est pas connue, l'envoi n'est pas prouvé de bout en bout.

À confirmer ensuite :

- `cansSpawned` / `cansLooted` envoyés à 1 / 0 pour ne pas déclarer « tous les conteneurs pillés » ;
- mort détectée par « sortie de l'Abysse directement amarré » : non testé en jeu ;
- lecture du presse-papiers et notifications Windows : compilées, jamais exécutées sur un vrai Windows.
