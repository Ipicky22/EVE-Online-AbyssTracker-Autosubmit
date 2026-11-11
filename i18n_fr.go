package main

// French interface texts.
var messagesFR = map[string]string{
	// Page
	"dryRunBanner":  "dry-run : rien n'est envoyé",
	"sessionOn":     "Session en cours",
	"sessionOff":    "Pas de session",
	"weather":       "Météo",
	"tier":          "Tier",
	"hull":          "Coque",
	"fit":           "Fit",
	"allCans":       "Je pille tous les conteneurs",
	"allCansHelp":   "Abyss Tracker classe les runs envoyés en « all cans » si la case est cochée, en « Bioadaptive and some cans » sinon.",
	"fitNeedsToken": "renseigne ton jeton Abyss Tracker pour lister tes fits",
	"none":          "Aucun",
	"startSession":  "Démarrer la session",
	"updateSession": "Mettre à jour la session",
	"endSession":    "Terminer la session",
	"settings":      "Réglages",
	"tokenLabel":    "Jeton Abyss Tracker",
	"tokenSaved":    "enregistré ; laisse vide pour le garder",
	"tokenHowTo":    "à créer sur abysstracker.com → My account → Personal access tokens",
	"logDirLabel":   "Dossier des journaux de discussion d'EVE",
	"logDirHelp":    "là où EVE écrit ses fichiers Local_….txt",
	"language":      "Langue",
	"save":          "Enregistrer",

	// Session
	"settingsSaved":  "Réglages enregistrés.",
	"sessionStarted": "Session en cours : %s T%d. Copie ta soute avant le premier run.",
	"sessionEnded":   "Session terminée.",
	"errNoLogDir":    "Dossier des journaux EVE introuvable : %s (Réglages)",
	"errNoToken":     "Renseigne d'abord ton jeton Abyss Tracker (Réglages)",

	// Local log
	"logFollowed":   "Journal Local suivi : %s.",
	"noLocalLog":    "Aucun journal Local dans %s : lance EVE.",
	"logUnreadable": "Journal Local illisible : %v",

	// Runs
	"baseSaved":           "Soute de départ enregistrée.",
	"noBaseAtStart":       "Run détecté sans copie de soute au départ : il ne sera pas envoyé.",
	"midRunIgnored":       "Run déjà en cours au démarrage de la session : ignoré. Copie ta soute pour le prochain.",
	"runDoneCopy":         "Run terminé. Copie ta soute (Ctrl+A, Ctrl+C).",
	"runLost":             "Run perdu (mort dans l'Abysse).",
	"movedRecopy":         "Changement de système : recopie ta soute avant le prochain run.",
	"notSentNoBase":       "Run non envoyé : pas de copie de soute au départ. Copie ta soute maintenant pour le prochain.",
	"notSentNewRun":       "Run non envoyé : nouveau run lancé sans copie de soute.",
	"notSentMoved":        "Run non envoyé : changement de système avant la copie de soute.",
	"notSentStopped":      "Run non envoyé : session terminée avant la copie de soute.",
	"notSentUnknownItems": "Run non envoyé : objets inconnus d'EVE (%s). La copie n'était peut-être pas une soute.",
	"notSentError":        "Run non envoyé : %v",

	// Upload
	"dryRunStatus": "[dry-run] run de %v non envoyé :\n%s",
	"dryRunNotify": "[dry-run] run de %v détecté.",
	"sentStatus":   "Run envoyé : %s",
	"sentNotify":   "Run envoyé sur Abyss Tracker.",
	"rejected":     "Run refusé par Abyss Tracker : %s",
	"sendFailed":   "Envoi impossible, le run reste en attente : %v",
}
