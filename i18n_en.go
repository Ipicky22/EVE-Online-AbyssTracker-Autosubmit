package main

// English interface texts.
var messagesEN = map[string]string{
	// Page
	"dryRunBanner":  "dry-run: nothing is sent",
	"sessionOn":     "Session running",
	"sessionOff":    "No session",
	"weather":       "Weather",
	"tier":          "Tier",
	"hull":          "Hull",
	"fit":           "Fit",
	"fitNeedsToken": "set your Abyss Tracker token to list your fits",
	"none":          "None",
	"startSession":  "Start session",
	"updateSession": "Update session",
	"endSession":    "End session",
	"settings":      "Settings",
	"tokenLabel":    "Abyss Tracker token",
	"tokenSaved":    "saved; leave empty to keep it",
	"tokenHowTo":    "create one at abysstracker.com → My account → Personal access tokens",
	"logDirLabel":   "EVE chat logs folder",
	"logDirHelp":    "where EVE writes its Local_….txt files",
	"language":      "Language",
	"save":          "Save",

	// Session
	"settingsSaved":  "Settings saved.",
	"sessionStarted": "Session running: %s T%d. Copy your cargo hold before the first run.",
	"sessionEnded":   "Session ended.",
	"errNoLogDir":    "EVE chat logs folder not found: %s (Settings)",
	"errNoToken":     "Set your Abyss Tracker token first (Settings)",

	// Local log
	"logFollowed":   "Following Local log: %s.",
	"noLocalLog":    "No Local log in %s: start EVE.",
	"logUnreadable": "Local log unreadable: %v",

	// Runs
	"baseSaved":           "Starting cargo saved.",
	"noBaseAtStart":       "Run detected without a cargo copy at the start: it will not be sent.",
	"midRunIgnored":       "A run was already in progress when the session started: ignored. Copy your cargo hold for the next one.",
	"runDoneCopy":         "Run finished. Copy your cargo hold (Ctrl+A, Ctrl+C).",
	"runLost":             "Run lost (died in the Abyss).",
	"movedRecopy":         "System change: copy your cargo hold again before the next run.",
	"notSentNoBase":       "Run not sent: no cargo copy at the start. Copy your cargo hold now for the next one.",
	"notSentNewRun":       "Run not sent: a new run was started without a cargo copy.",
	"notSentMoved":        "Run not sent: system change before the cargo copy.",
	"notSentStopped":      "Run not sent: session ended before the cargo copy.",
	"notSentUnknownItems": "Run not sent: items unknown to EVE (%s). The copy may not have been a cargo hold.",
	"notSentError":        "Run not sent: %v",

	// Upload
	"dryRunStatus": "[dry-run] %v run not sent:\n%s",
	"dryRunNotify": "[dry-run] %v run detected.",
	"sentStatus":   "Run sent: %s",
	"sentNotify":   "Run sent to Abyss Tracker.",
	"rejected":     "Run rejected by Abyss Tracker: %s",
	"sendFailed":   "Could not send, the run stays queued: %v",
}
