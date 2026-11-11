# EVE Online AbyssTracker Autosubmit

[English](README.md) | **Français** | [Deutsch](README.de.md) | [Русский](README.ru.md) | [中文](README.zh.md)

Envoie automatiquement vos runs d'Abysse d'EVE Online sur [Abyss Tracker](https://abysstracker.com).
Le programme s'appelle `abyssup`.
Il ne touche jamais au client EVE : il lit seulement le journal de discussion Local qu'EVE écrit sur le disque, et le presse-papiers de Windows.

> **État : en cours de développement.** L'envoi fonctionne. La lecture du presse-papiers Windows n'a pas encore été testée sur une vraie machine, voir [État des vérifications](#état-des-vérifications).

## Utilisation

1. Lancez `abyssup.exe` : la page `http://localhost:47813` s'ouvre.
2. Choisissez la météo, le tier, la coque et le fit, puis cliquez sur **Démarrer la session**.
3. Copiez votre soute (`Ctrl+A`, `Ctrl+C`) avant le premier run.
4. Après chaque run, copiez de nouveau votre soute : le run est envoyé sur Abyss Tracker.
5. Après un changement de système ou un amarrage, recopiez votre soute avant le run suivant.

Les runs sont publiés en public.
Un run sans copie de soute fiable n'est pas envoyé : il est rangé dans `incomplete/`.

## Installation (une fois)

1. Créez un jeton sur `abysstracker.com/my-account/personal-access-tokens`.
2. Dans la page, ouvrez **Réglages**, collez le jeton et enregistrez.

Aucune application de développeur EVE ni connexion du personnage n'est nécessaire.
EVE doit enregistrer les discussions dans un fichier (c'est le cas par défaut).
Si vos journaux ne sont pas dans `Documents\EVE\logs\Chatlogs`, indiquez le dossier dans **Réglages**.

## Langues

| | Anglais | Français | Allemand | Russe | Chinois |
|---|---|---|---|---|---|
| Interface (**Réglages → Langue**) | oui | oui | oui | oui | oui |
| Client EVE : noms d'objets | oui | oui | oui | oui | oui |
| Client EVE : journal Local | supposé | vérifié | supposé | supposé | supposé |

« Vérifié » veut dire testé sur un vrai fichier de journal. « Supposé » veut dire que les mots viennent
d'outils publics qui lisent les journaux d'EVE, sans test sur un vrai journal de cette langue : la
détection des runs peut ne pas fonctionner.
Un fichier de journal Local dans une de ces langues est le bienvenu, voir [Limites connues](#limites-connues).
Les textes allemands, russes et chinois n'ont pas été relus par une personne dont c'est la langue.
Les noms d'objets sont toujours envoyés à Abyss Tracker en anglais.

## Options

- `-dry-run` : écrit les runs dans le journal du programme au lieu de les envoyer.
- `-no-open` : n'ouvre pas le navigateur au lancement.

## Données

Dans `%APPDATA%\abyssup\` (`~/.config/abyssup/` sous Linux) :
`config.json`, `names.json` (cache des noms d'objets), `abyssup.log`,
et les dossiers `pending/`, `sent/`, `incomplete/`, `rejected/`.

Le jeton est stocké en clair dans votre profil utilisateur, jamais dans ce dépôt.

## Fonctionnement

- **Personnage** : lu dans le fichier `Local_<date>_<heure>_<identifiant du personnage>.txt` le plus récent.
- **Début, fin et durée du run** : EVE écrit une ligne dans le journal Local à chaque changement de système. À l'entrée dans l'Abysse, le système apparaît comme `Inconnu` (`Unknown` sur un client anglais). Précis à la seconde, sans aucune requête réseau.
- **Mort** : sortie de l'Abysse dans un autre système que celui où le filament a été utilisé (réveil dans le clone).
- **Butin** : différence entre la copie de soute faite avant le run et celle faite après. Les noms d'objets sont convertis en identifiants par l'API publique d'EVE (ESI), avec un cache sur disque.
- **Météo, tier, coque, fit** : pris dans les réglages de la session.

## Limites connues

- Le journal ne dit pas quand vous vous amarrez. Un changement de système efface la soute connue et demande une nouvelle copie, mais si vous vous amarrez **sans quitter le système** et oubliez de recopier votre soute, le butin du run suivant sera faux.
- Une mort n'est pas détectée quand votre clone est dans le système où le filament a été utilisé.
- Un seul personnage à la fois.
- Le mot affiché pour l'Abysse dans le journal Local (`Unknown`, `Unbekannt`, `Неизвестно`, `未知`) n'est vérifié que pour le français (`Inconnu`). S'il est faux pour votre langue, les runs ne sont simplement pas détectés et rien n'est envoyé.

## Développement

```
go test ./...
GOOS=windows go build -ldflags "-H=windowsgui" -o abyssup.exe .
```

## État des vérifications

Vérifié :

- détection d'un run sur un vrai journal Local français : personnage, entrée, sortie, durée à la seconde ;
- le jeton Abyss Tracker est accepté dans l'en-tête `X-API-KEY` ;
- `fits/list` renvoie les fits du compte avec leur vaisseau ;
- un run de test envoyé par le code d'envoi du programme apparaît sur Abyss Tracker avec la bonne météo, le bon tier, la bonne coque, le bon fit, la bonne durée et le bon butin ;
- les filaments consommés, envoyés dans les objets perdus, sont comptés une seule fois.

Reste à confirmer :

- Abyss Tracker ne classe les runs envoyés que dans deux types de butin : « all cans » ou « Bioadaptive and some cans ». Une case de la session choisit lequel. Que le premier s'obtienne en envoyant deux nombres de conteneurs égaux est supposé, pas encore constaté sur un envoi ;
- lecture du presse-papiers et notifications Windows : compilées, jamais exécutées sur un vrai Windows.
