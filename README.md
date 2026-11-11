# abyssup

Envoie tout seul les runs Abyss d'EVE Online sur [Abyss Tracker](https://abysstracker.com).
Il ne touche jamais au client EVE : il lit la position du personnage via l'API EVE (ESI) et le presse-papiers Windows.

## Usage

1. Lancer `abyssup.exe` : la page `http://localhost:47813` s'ouvre.
2. Choisir météo, tier, coque, fit, visibilité, puis « Démarrer la session ».
3. Copier la soute (`Ctrl+A`, `Ctrl+C`) avant le premier run.
4. Après chaque run, recopier la soute : le run part sur Abyss Tracker.
5. Après un passage en station, recopier la soute avant le run suivant.

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

## À confirmer avant le premier envoi réel

Les valeurs ci-dessous ne sont pas documentées par l'API ; elles sont regroupées en tête de `upload.go`.

- `privacy` : `public` / `anonymous` / `private`.
- `shipType` : `cruiser` / `destroyer` / `frigate`.
- `cansSpawned` / `cansLooted` : envoyés à 0.
- Mort détectée par « sortie de l'Abysse directement amarré » : non testé en jeu.
- Lecture du presse-papiers et notifications Windows : compilées, jamais exécutées sur un vrai Windows.
