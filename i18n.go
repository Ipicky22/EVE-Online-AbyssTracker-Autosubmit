package main

import "fmt"

// Interface languages. The first one is the default.
var langs = []string{"en", "fr"}

var langNames = map[string]string{"en": "English", "fr": "Français"}

// messages holds every text shown to the user: key -> {English, French}.
// Technical errors (network, HTTP) stay in English and are passed in as arguments.
var messages = map[string][2]string{
	// Page
	"dryRunBanner":  {"dry-run: nothing is sent", "dry-run : rien n'est envoyé"},
	"sessionOn":     {"Session running", "Session en cours"},
	"sessionOff":    {"No session", "Pas de session"},
	"weather":       {"Weather", "Météo"},
	"tier":          {"Tier", "Tier"},
	"hull":          {"Hull", "Coque"},
	"fit":           {"Fit", "Fit"},
	"fitNeedsToken": {"set your Abyss Tracker token to list your fits", "renseigne ton jeton Abyss Tracker pour lister tes fits"},
	"none":          {"None", "Aucun"},
	"startSession":  {"Start session", "Démarrer la session"},
	"updateSession": {"Update session", "Mettre à jour la session"},
	"endSession":    {"End session", "Terminer la session"},
	"settings":      {"Settings", "Réglages"},
	"tokenLabel":    {"Abyss Tracker token", "Jeton Abyss Tracker"},
	"tokenSaved":    {"saved; leave empty to keep it", "enregistré ; laisse vide pour le garder"},
	"tokenHowTo":    {"create one at abysstracker.com → My account → Personal access tokens", "à créer sur abysstracker.com → My account → Personal access tokens"},
	"logDirLabel":   {"EVE chat logs folder", "Dossier des journaux de discussion d'EVE"},
	"logDirHelp":    {"where EVE writes its Local_….txt files", "là où EVE écrit ses fichiers Local_….txt"},
	"language":      {"Language", "Langue"},
	"save":          {"Save", "Enregistrer"},

	// Session
	"settingsSaved":  {"Settings saved.", "Réglages enregistrés."},
	"sessionStarted": {"Session running: %s T%d. Copy your cargo hold before the first run.", "Session en cours : %s T%d. Copie ta soute avant le premier run."},
	"sessionEnded":   {"Session ended.", "Session terminée."},
	"errNoLogDir":    {"EVE chat logs folder not found: %s (Settings)", "Dossier des journaux EVE introuvable : %s (Réglages)"},
	"errNoToken":     {"Set your Abyss Tracker token first (Settings)", "Renseigne d'abord ton jeton Abyss Tracker (Réglages)"},

	// Local log
	"logFollowed":   {"Following Local log: %s.", "Journal Local suivi : %s."},
	"noLocalLog":    {"No Local log in %s: start EVE.", "Aucun journal Local dans %s : lance EVE."},
	"logUnreadable": {"Local log unreadable: %v", "Journal Local illisible : %v"},

	// Runs
	"baseSaved":           {"Starting cargo saved.", "Soute de départ enregistrée."},
	"noBaseAtStart":       {"Run detected without a cargo copy at the start: it will not be sent.", "Run détecté sans copie de soute au départ : il ne sera pas envoyé."},
	"midRunIgnored":       {"A run was already in progress when the session started: ignored. Copy your cargo hold for the next one.", "Run déjà en cours au démarrage de la session : ignoré. Copie ta soute pour le prochain."},
	"runDoneCopy":         {"Run finished. Copy your cargo hold (Ctrl+A, Ctrl+C).", "Run terminé. Copie ta soute (Ctrl+A, Ctrl+C)."},
	"runLost":             {"Run lost (died in the Abyss).", "Run perdu (mort dans l'Abysse)."},
	"movedRecopy":         {"System change: copy your cargo hold again before the next run.", "Changement de système : recopie ta soute avant le prochain run."},
	"notSentNoBase":       {"Run not sent: no cargo copy at the start. Copy your cargo hold now for the next one.", "Run non envoyé : pas de copie de soute au départ. Copie ta soute maintenant pour le prochain."},
	"notSentNewRun":       {"Run not sent: a new run was started without a cargo copy.", "Run non envoyé : nouveau run lancé sans copie de soute."},
	"notSentMoved":        {"Run not sent: system change before the cargo copy.", "Run non envoyé : changement de système avant la copie de soute."},
	"notSentStopped":      {"Run not sent: session ended before the cargo copy.", "Run non envoyé : session terminée avant la copie de soute."},
	"notSentUnknownItems": {"Run not sent: items unknown to EVE (%s). The copy may not have been a cargo hold.", "Run non envoyé : objets inconnus d'EVE (%s). La copie n'était peut-être pas une soute."},
	"notSentError":        {"Run not sent: %v", "Run non envoyé : %v"},

	// Upload
	"dryRunStatus": {"[dry-run] %v run not sent:\n%s", "[dry-run] run de %v non envoyé :\n%s"},
	"dryRunNotify": {"[dry-run] %v run detected.", "[dry-run] run de %v détecté."},
	"sentStatus":   {"Run sent: %s", "Run envoyé : %s"},
	"sentNotify":   {"Run sent to Abyss Tracker.", "Run envoyé sur Abyss Tracker."},
	"rejected":     {"Run rejected by Abyss Tracker: %s", "Run refusé par Abyss Tracker : %s"},
	"sendFailed":   {"Could not send, the run stays queued: %v", "Envoi impossible, le run reste en attente : %v"},
}

// tr returns the text for key in the given language, English when the language is unknown.
func tr(lang, key string, args ...any) string {
	m, ok := messages[key]
	if !ok {
		return key
	}
	s := m[0]
	if lang == "fr" {
		s = m[1]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// catalog returns every text in the given language, for the page template.
func catalog(lang string) map[string]string {
	c := make(map[string]string, len(messages))
	for key := range messages {
		c[key] = tr(lang, key)
	}
	return c
}
