# EVE Online AbyssTracker Autosubmit

Submits your EVE Online Abyssal Deadspace runs to [Abyss Tracker](https://abysstracker.com) automatically.
The program is called `abyssup`.
It never touches the EVE client: it only reads the Local chat log that EVE writes to disk, and the Windows clipboard.

> **Status: work in progress.** Uploading is not proven end to end yet, see [Verification status](#verification-status).

## Usage

1. Run `abyssup.exe`: the page `http://localhost:47813` opens. Its labels are in French.
2. Pick the weather, tier, hull and fit, then click « Démarrer la session » (start the session).
3. Copy your cargo hold (`Ctrl+A`, `Ctrl+C`) before the first run.
4. After each run, copy your cargo hold again: the run is sent to Abyss Tracker.
5. After changing system or docking, copy your cargo hold again before the next run.

Runs are published as public.
A run without a reliable cargo copy is not sent: it is stored in `incomplete/`.

Both English and French EVE clients are supported; item names are sent in English.

## Setup (once)

1. Create a token at `abysstracker.com/my-account/personal-access-tokens`.
2. On the page, open « Réglages » (settings), paste the token and save.

No EVE developer application and no character login are needed.
EVE must log chat to file (it does by default).
If your logs are not in `Documents\EVE\logs\Chatlogs`, set the folder in « Réglages ».

## Options

- `-dry-run`: writes runs to the log instead of sending them.
- `-no-open`: does not open the browser at startup.

## Data

In `%APPDATA%\abyssup\` (`~/.config/abyssup/` on Linux):
`config.json`, `names.json` (item name cache), `abyssup.log`,
and the folders `pending/`, `sent/`, `incomplete/`, `rejected/`.

The token is stored in plain text in your user profile, never in this repository.

## How it works

- **Character**: read from the newest `Local_<date>_<time>_<characterID>.txt` file.
- **Run start, end and duration**: EVE writes a line to the Local chat log on every system change. Entering the Abyss shows the system as `Unknown` (`Inconnu` on a French client). Accurate to the second, no network request.
- **Death**: leaving the Abyss into a different system than the one the filament was used in (waking up in the clone).
- **Loot**: difference between the cargo copy made before the run and the one made after it. Item names are resolved to type IDs through the public EVE API (ESI), with a disk cache.
- **Weather, tier, hull, fit**: taken from the session settings.

## Known limits

- The log does not say when you dock. A system change discards the known cargo and asks for a new copy, but if you dock **without leaving the system** and forget to copy your cargo again, the next run's loot will be wrong.
- A death is not detected when your clone is in the system the filament was used in.
- One character at a time.
- The `Unknown` name on an English client is assumed, not yet checked against a real English log.

## Development

```
go test ./...
GOOS=windows go build -ldflags "-H=windowsgui" -o abyssup.exe .
```

## Verification status

Verified:

- run detection from a real French Local chat log: character, entry, exit, duration to the second;
- the Abyss Tracker token is accepted in the `X-API-KEY` header;
- `fits/list` returns the account's fits with their ship;
- `POST /v1/abyss-tracker/telemetry` answers `Error: false` with a `RunId`.

**Blocking**: the two test runs created this way show up neither on the site, nor in `get-runs`,
nor in `runs/{id}`. The EVE Workbench team confirmed the endpoint is open to third-party tools and that
`PUBLIC` is the right privacy value (any other value publishes anonymously), but does not see the runs
in their database. They are investigating on their side.

Still to confirm:

- `cansSpawned` / `cansLooted` are sent as 1 / 0 so the run is not declared as "all cans looted";
- Windows clipboard reading and notifications: compiled, never run on a real Windows machine.
