# EVE Online AbyssTracker Autosubmit

Submits your EVE Online Abyssal Deadspace runs to [Abyss Tracker](https://abysstracker.com) automatically.
The program is called `abyssup`.
It never touches the EVE client: it only reads the character's location through the EVE API (ESI) and the Windows clipboard.

> **Status: work in progress.** Uploading is not proven end to end yet, see [Verification status](#verification-status).

## Usage

1. Run `abyssup.exe`: the page `http://localhost:47813` opens. Its labels are in French.
2. Pick the weather, tier, hull and fit, then click « Démarrer la session » (start the session).
3. Copy your cargo hold (`Ctrl+A`, `Ctrl+C`) before the first run.
4. After each run, copy your cargo hold again: the run is sent to Abyss Tracker.
5. After docking, copy your cargo hold again before the next run.

Runs are published as public.
A run without a reliable cargo copy is not sent: it is stored in `incomplete/`.

Both English and French EVE clients are supported; item names are sent in English.

## Setup (once)

1. Create a token at `abysstracker.com/my-account/personal-access-tokens`.
2. Create an application at `developers.eveonline.com`:
   callback `http://localhost:47813/callback`, scope `esi-location.read_location.v1`.
3. On the page, open « Réglages » (settings), paste the token and the application's client ID, and save.
4. Click « Connecter le personnage EVE » (connect the EVE character).

## Options

- `-dry-run`: writes runs to the log instead of sending them.
- `-no-open`: does not open the browser at startup.

## Data

In `%APPDATA%\abyssup\` (`~/.config/abyssup/` on Linux):
`config.json`, `token.json`, `names.json` (item name cache), `abyssup.log`,
and the folders `pending/`, `sent/`, `incomplete/`, `rejected/`.

Tokens are stored in plain text in your user profile, never in this repository.

## How it works

- **Run start, end and duration**: ESI location polled every 5 seconds during a session. A solar system ID between 32000001 and 32000200 means the character is in the Abyss. Accuracy is about 5 to 10 seconds.
- **Death**: leaving the Abyss directly into a docked state.
- **Loot**: difference between the cargo copy made before the run and the one made after it.
- **Weather, tier, hull, fit**: taken from the session settings.

## Development

```
go test ./...
GOOS=windows go build -ldflags "-H=windowsgui" -o abyssup.exe .
```

## Verification status

Verified with a real token (2026-11-11):

- the token is accepted in the `X-API-KEY` header;
- `fits/list` returns the account's fits with their ship;
- `POST /v1/abyss-tracker/telemetry` answers `Error: false` with a `RunId`.

**Blocking**: the two test runs created this way (`PRIVATE`, then `PUBLIC`) show up neither on the site,
nor in `get-runs`, nor in `runs/{id}`. Until the cause is known, uploading is not proven end to end.

Still to confirm after that:

- `cansSpawned` / `cansLooted` are sent as 1 / 0 so the run is not declared as "all cans looted";
- death detected as "left the Abyss directly docked": not tested in game;
- Windows clipboard reading and notifications: compiled, never run on a real Windows machine.
