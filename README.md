# EVE Online AbyssTracker Autosubmit

**English** | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | [中文](README.zh.md)

Submits your EVE Online Abyssal Deadspace runs to [Abyss Tracker](https://abysstracker.com) automatically.
The program is called `abysstracker-autosubmit`.
It never touches the EVE client: it only reads the Local chat log that EVE writes to disk, and the Windows clipboard.

> **Status: work in progress.** Uploading works. Reading the Windows clipboard has not been tested on a real machine yet, see [Verification status](#verification-status).

## Usage

1. Run `abysstracker-autosubmit.exe`: the page `http://localhost:47813` opens.
2. Pick the weather, tier, hull and fit, then click **Start session**.
3. Copy your cargo hold (`Ctrl+A`, `Ctrl+C`) before the first run.
4. After each run, copy your cargo hold again: the run is sent to Abyss Tracker.
5. After changing system or docking, copy your cargo hold again before the next run.
6. To stop the program, click **Quit** in the page: closing the tab does not stop it.

Runs are published as public.
A run without a reliable cargo copy is not sent: it is stored in `incomplete/`.

The page lists the runs sent during the session, with the duration and loot value Abyss Tracker computed for them.
The list updates by itself and is not kept once a new session starts.

## Setup (once)

1. Create a token at `abysstracker.com/my-account/personal-access-tokens`.
2. On the page, open **Settings**, paste the token and save.

No EVE developer application and no character login are needed.
EVE must log chat to file (it does by default).
If your logs are not in `Documents\EVE\logs\Chatlogs`, set the folder in **Settings**.

## Languages

| | English | French | German | Russian | Chinese |
|---|---|---|---|---|---|
| Interface (**Settings → Language**) | yes | yes | yes | yes | yes |
| EVE client: item names | yes | yes | yes | yes | yes |
| EVE client: Local chat log | assumed | checked | assumed | assumed | assumed |

"Checked" means tested against a real log file. "Assumed" means the wording comes from public EVE log
parsers and has not been tested against a real log of that language: run detection may not work.
A Local log file in one of these languages is welcome, see [Known limits](#known-limits).
The German, Russian and Chinese texts were not reviewed by a native speaker.
Item names are always sent to Abyss Tracker in English.

## Options

- `-dry-run`: writes runs to the log instead of sending them.
- `-no-open`: does not open the browser at startup.

## Data

In `%APPDATA%\abysstracker-autosubmit\` (`~/.config/abysstracker-autosubmit/` on Linux):
`config.json`, `names.json` (item name cache), `abysstracker-autosubmit.log`,
and the folders `pending/`, `sent/`, `incomplete/`, `rejected/`.

The token is stored in plain text in your user profile, never in this repository.

At startup and then every 6 hours, the program asks GitHub for the latest release of this repository, to tell you on the page when a newer version exists.
The request carries no token and no character data.

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
- The word shown for the Abyss in the Local log (`Unknown`, `Unbekannt`, `Неизвестно`, `未知`) is only checked for French (`Inconnu`). If it is wrong for your language, runs are simply not detected and nothing is sent.

## Development

```
go test ./...
GOOS=windows go build -ldflags "-H=windowsgui" -o abysstracker-autosubmit.exe .
```

Before a release, set `version` in `update.go` to the release tag.

## Verification status

Verified:

- run detection from a real French Local chat log: character, entry, exit, duration to the second;
- the Abyss Tracker token is accepted in the `X-API-KEY` header;
- `fits/list` returns the account's fits with their ship;
- a test run sent through the program's own upload code shows up on Abyss Tracker with the right weather, tier, hull, fit, duration and loot;
- the consumed filaments, sent in the lost items, are counted once.

Still to confirm:

- Abyss Tracker files uploaded runs under two loot types only: "all cans", or "Bioadaptive and some cans". A session checkbox picks which. That the first one is obtained by sending equal can counts is assumed, not seen on an upload yet;
- Windows clipboard reading and notifications: compiled, never run on a real Windows machine.
