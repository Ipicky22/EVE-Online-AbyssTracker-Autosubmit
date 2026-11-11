# EVE Online AbyssTracker Autosubmit

[English](README.md) | [Français](README.fr.md) | **Deutsch** | [Русский](README.ru.md) | [中文](README.zh.md)

> Diese Übersetzung wurde nicht von einer Person mit Deutsch als Muttersprache geprüft. Maßgeblich ist die [englische Fassung](README.md).

Sendet deine Abyssal-Deadspace-Runs aus EVE Online automatisch an [Abyss Tracker](https://abysstracker.com).
Das Programm heißt `abyssup`.
Es greift nie in den EVE-Client ein: Es liest nur das Lokal-Chatlog, das EVE auf die Festplatte schreibt, und die Windows-Zwischenablage.

> **Status: in Arbeit.** Das Hochladen funktioniert. Das Lesen der Windows-Zwischenablage wurde noch nicht auf einem echten Rechner getestet, siehe [Prüfstand](#prüfstand).

## Verwendung

1. Starte `abyssup.exe`: Die Seite `http://localhost:47813` öffnet sich.
2. Wähle Wetter, Stufe, Schiffsklasse und Fitting und klicke auf **Sitzung starten**.
3. Kopiere deinen Frachtraum (`Strg+A`, `Strg+C`) vor dem ersten Run.
4. Kopiere nach jedem Run deinen Frachtraum erneut: Der Run wird an Abyss Tracker gesendet.
5. Kopiere nach einem Systemwechsel oder dem Andocken deinen Frachtraum erneut, bevor du den nächsten Run startest.

Runs werden öffentlich veröffentlicht.
Ein Run ohne verlässliche Frachtraum-Kopie wird nicht gesendet: Er wird in `incomplete/` abgelegt.

## Einrichtung (einmalig)

1. Erstelle ein Token unter `abysstracker.com/my-account/personal-access-tokens`.
2. Öffne auf der Seite **Einstellungen**, füge das Token ein und speichere.

Es wird weder eine EVE-Entwickleranwendung noch eine Anmeldung des Charakters benötigt.
EVE muss Chats in Dateien protokollieren (das ist die Voreinstellung).
Liegen deine Logs nicht in `Documents\EVE\logs\Chatlogs`, trage den Ordner in den **Einstellungen** ein.

## Sprachen

| | Englisch | Französisch | Deutsch | Russisch | Chinesisch |
|---|---|---|---|---|---|
| Oberfläche (**Einstellungen → Sprache**) | ja | ja | ja | ja | ja |
| EVE-Client: Gegenstandsnamen | ja | ja | ja | ja | ja |
| EVE-Client: Lokal-Chatlog | angenommen | geprüft | angenommen | angenommen | angenommen |

„Geprüft“ heißt: an einer echten Logdatei getestet. „Angenommen“ heißt: Der Wortlaut stammt aus öffentlichen
EVE-Log-Parsern und wurde nicht an einem echten Log dieser Sprache getestet. Die Run-Erkennung funktioniert
dann möglicherweise nicht.
Eine Lokal-Logdatei in einer dieser Sprachen ist willkommen, siehe [Bekannte Grenzen](#bekannte-grenzen).
Die deutschen, russischen und chinesischen Texte wurden nicht von Muttersprachlern geprüft.
Gegenstandsnamen werden immer auf Englisch an Abyss Tracker gesendet.

## Optionen

- `-dry-run`: schreibt Runs ins Programmlog, statt sie zu senden.
- `-no-open`: öffnet beim Start keinen Browser.

## Daten

In `%APPDATA%\abyssup\` (`~/.config/abyssup/` unter Linux):
`config.json`, `names.json` (Cache der Gegenstandsnamen), `abyssup.log`
und die Ordner `pending/`, `sent/`, `incomplete/`, `rejected/`.

Das Token wird im Klartext in deinem Benutzerprofil gespeichert, nie in diesem Repository.

## Funktionsweise

- **Charakter**: aus der neuesten Datei `Local_<Datum>_<Uhrzeit>_<Charakter-ID>.txt` gelesen.
- **Beginn, Ende und Dauer des Runs**: EVE schreibt bei jedem Systemwechsel eine Zeile ins Lokal-Chatlog. Beim Eintritt in den Abgrund erscheint das System als `Unknown` (auf einem französischen Client `Inconnu`). Sekundengenau, ohne Netzwerkanfrage.
- **Tod**: Verlassen des Abgrunds in ein anderes System als das, in dem das Filament benutzt wurde (Aufwachen im Klon).
- **Beute**: Unterschied zwischen der Frachtraum-Kopie vor dem Run und der danach. Gegenstandsnamen werden über die öffentliche EVE-API (ESI) in Typ-IDs aufgelöst, mit einem Cache auf der Festplatte.
- **Wetter, Stufe, Schiffsklasse, Fitting**: aus den Sitzungseinstellungen.

## Bekannte Grenzen

- Das Log sagt nicht, wann du andockst. Ein Systemwechsel verwirft den bekannten Frachtraum und verlangt eine neue Kopie. Dockst du aber an, **ohne das System zu verlassen**, und vergisst die neue Kopie, ist die Beute des nächsten Runs falsch.
- Ein Tod wird nicht erkannt, wenn dein Klon in dem System liegt, in dem das Filament benutzt wurde.
- Nur ein Charakter gleichzeitig.
- Das Wort, das im Lokal-Log für den Abgrund steht (`Unknown`, `Unbekannt`, `Неизвестно`, `未知`), ist nur für Französisch (`Inconnu`) geprüft. Ist es für deine Sprache falsch, werden Runs schlicht nicht erkannt und nichts wird gesendet.

## Entwicklung

```
go test ./...
GOOS=windows go build -ldflags "-H=windowsgui" -o abyssup.exe .
```

## Prüfstand

Geprüft:

- Run-Erkennung an einem echten französischen Lokal-Chatlog: Charakter, Eintritt, Austritt, Dauer sekundengenau;
- das Abyss-Tracker-Token wird im Header `X-API-KEY` akzeptiert;
- `fits/list` liefert die Fittings des Kontos mit ihrem Schiff;
- ein Test-Run, der über den Upload-Code des Programms gesendet wurde, erscheint auf Abyss Tracker mit richtigem Wetter, richtiger Stufe, Schiffsklasse, Fitting, Dauer und Beute;
- die verbrauchten Filamente, die in den verlorenen Gegenständen gesendet werden, werden nur einmal gezählt.

Noch zu bestätigen:

- `cansSpawned` / `cansLooted` werden als 1 / 0 gesendet, was die Website als „Bioadaptive and some cans“ anzeigt. Welche Werte „nicht angegeben“ ergeben, ist noch nicht bekannt;
- Lesen der Windows-Zwischenablage und Benachrichtigungen: kompiliert, nie auf einem echten Windows ausgeführt.
