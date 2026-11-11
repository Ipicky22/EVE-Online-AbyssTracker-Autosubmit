package main

// German interface texts.
// Not reviewed by a native speaker.
var messagesDE = map[string]string{
	// Page
	"dryRunBanner":  "dry-run: es wird nichts gesendet",
	"sessionOn":     "Sitzung läuft",
	"sessionOff":    "Keine Sitzung",
	"weather":       "Wetter",
	"tier":          "Stufe",
	"hull":          "Schiffsklasse",
	"fit":           "Fitting",
	"fitNeedsToken": "trage dein Abyss-Tracker-Token ein, um deine Fittings aufzulisten",
	"none":          "Keines",
	"startSession":  "Sitzung starten",
	"updateSession": "Sitzung aktualisieren",
	"endSession":    "Sitzung beenden",
	"settings":      "Einstellungen",
	"tokenLabel":    "Abyss-Tracker-Token",
	"tokenSaved":    "gespeichert; leer lassen, um es zu behalten",
	"tokenHowTo":    "erstellen unter abysstracker.com → My account → Personal access tokens",
	"logDirLabel":   "Ordner der EVE-Chatlogs",
	"logDirHelp":    "dort schreibt EVE seine Dateien Local_….txt",
	"language":      "Sprache",
	"save":          "Speichern",

	// Session
	"settingsSaved":  "Einstellungen gespeichert.",
	"sessionStarted": "Sitzung läuft: %s T%d. Kopiere deinen Frachtraum vor dem ersten Run.",
	"sessionEnded":   "Sitzung beendet.",
	"errNoLogDir":    "Ordner der EVE-Chatlogs nicht gefunden: %s (Einstellungen)",
	"errNoToken":     "Trage zuerst dein Abyss-Tracker-Token ein (Einstellungen)",

	// Local log
	"logFollowed":   "Lokal-Log wird verfolgt: %s.",
	"noLocalLog":    "Kein Lokal-Log in %s: starte EVE.",
	"logUnreadable": "Lokal-Log nicht lesbar: %v",

	// Runs
	"baseSaved":           "Ausgangs-Frachtraum gespeichert.",
	"noBaseAtStart":       "Run ohne Frachtraum-Kopie zu Beginn erkannt: er wird nicht gesendet.",
	"midRunIgnored":       "Beim Start der Sitzung lief bereits ein Run: ignoriert. Kopiere deinen Frachtraum für den nächsten.",
	"runDoneCopy":         "Run beendet. Kopiere deinen Frachtraum (Strg+A, Strg+C).",
	"runLost":             "Run verloren (im Abgrund gestorben).",
	"movedRecopy":         "Systemwechsel: kopiere deinen Frachtraum vor dem nächsten Run erneut.",
	"notSentNoBase":       "Run nicht gesendet: keine Frachtraum-Kopie zu Beginn. Kopiere deinen Frachtraum jetzt für den nächsten.",
	"notSentNewRun":       "Run nicht gesendet: ein neuer Run wurde ohne Frachtraum-Kopie gestartet.",
	"notSentMoved":        "Run nicht gesendet: Systemwechsel vor der Frachtraum-Kopie.",
	"notSentStopped":      "Run nicht gesendet: Sitzung vor der Frachtraum-Kopie beendet.",
	"notSentUnknownItems": "Run nicht gesendet: EVE kennt diese Gegenstände nicht (%s). Die Kopie war vielleicht kein Frachtraum.",
	"notSentError":        "Run nicht gesendet: %v",

	// Upload
	"dryRunStatus": "[dry-run] Run von %v nicht gesendet:\n%s",
	"dryRunNotify": "[dry-run] Run von %v erkannt.",
	"sentStatus":   "Run gesendet: %s",
	"sentNotify":   "Run an Abyss Tracker gesendet.",
	"rejected":     "Run von Abyss Tracker abgelehnt: %s",
	"sendFailed":   "Senden nicht möglich, der Run bleibt in der Warteschlange: %v",
}
