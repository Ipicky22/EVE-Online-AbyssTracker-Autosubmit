package main

import "time"

// Run est un run détecté, avant ajout des paramètres de session.
type Run struct {
	Start    time.Time
	Duration time.Duration
	Success  bool
	Gained   Cargo
	Lost     Cargo
}

type OutcomeKind int

const (
	Notify     OutcomeKind = iota // message seul
	Complete                      // run à envoyer
	Incomplete                    // run sans butin fiable : gardé en local, jamais envoyé
)

type Outcome struct {
	Kind OutcomeKind
	Run  *Run
	Msg  string
}

type trackerState int

const (
	idle trackerState = iota
	inAbyss
	awaitingLoot
)

// Tracker transforme changements de système et copies de soute en runs.
// Aucune entrée/sortie ici : tout passe par les valeurs de retour.
type Tracker struct {
	state  trackerState
	base   Cargo  // soute avant le prochain run ; nil = inconnue
	system string // dernier système connu hors Abysse
	entry  string // système d'où le run en cours a été lancé
	ignore bool   // run déjà en cours au démarrage de la session : ni début ni soute connus
	run    Run
	late   Cargo // dernière copie faite alors que le run semblait encore en cours
	lateAt time.Time
}

// lateGrace couvre le délai entre la sortie réelle et sa lecture dans le journal.
// Il se mesure sur l'horloge du PC (heure de la copie, heure de lecture de la ligne) :
// l'heure EVE écrite dans le journal peut être décalée de celle du PC.
const lateGrace = 15 * time.Second

// SetLocation donne la position lue dans l'historique du journal, avant tout nouvel événement.
func (t *Tracker) SetLocation(e LocalEvent) {
	if e.Abyss {
		t.state, t.ignore = inAbyss, true
		return
	}
	t.system = e.System
}

// OnLocal traite un changement de système ; seen est l'heure du PC à laquelle la ligne a été lue.
func (t *Tracker) OnLocal(e LocalEvent, seen time.Time) []Outcome {
	switch {
	case e.Abyss:
		if t.state == inAbyss {
			return nil
		}
		return t.enter(e.At)
	case t.state == inAbyss:
		return t.exit(e, seen)
	case e.System != t.system:
		t.system = e.System
		return t.moved()
	}
	return nil
}

func (t *Tracker) enter(at time.Time) []Outcome {
	out := t.dropAwaiting("nouveau run lancé sans copie de soute")
	t.state, t.entry = inAbyss, t.system
	t.run, t.late = Run{Start: at}, nil
	if t.base == nil {
		out = append(out, Outcome{Kind: Notify, Msg: "Run détecté sans copie de soute au départ : il ne sera pas envoyé."})
	}
	return out
}

func (t *Tracker) exit(e LocalEvent, seen time.Time) []Outcome {
	entry := t.entry
	t.system = e.System
	if t.ignore {
		t.state, t.ignore = idle, false
		return []Outcome{{Kind: Notify, Msg: "Run déjà en cours au démarrage de la session : ignoré. Copie ta soute pour le prochain."}}
	}
	t.run.Duration = e.At.Sub(t.run.Start)
	// Mort dans l'Abysse : la capsule est détruite aussi, le personnage se réveille dans son clone.
	// shortcut: indétectable si le clone est dans le système d'où le filament a été lancé.
	if entry != "" && e.System != entry {
		t.state, t.base = idle, nil
		run := t.run
		run.Gained, run.Lost = Cargo{}, Cargo{}
		return []Outcome{{Kind: Complete, Run: &run, Msg: "Run perdu (mort dans l'Abysse)."}}
	}
	if t.base == nil {
		t.state = idle
		run := t.run
		return []Outcome{{Kind: Incomplete, Run: &run, Msg: "Run non envoyé : pas de copie de soute au départ. Copie ta soute maintenant pour le prochain."}}
	}
	t.state = awaitingLoot
	t.run.Success = true
	// Une copie faite juste avant que la sortie ne soit lue dans le journal est celle d'après-run.
	if late := t.late; late != nil && seen.Sub(t.lateAt) <= lateGrace {
		if out := t.OnCargo(late, t.lateAt); out != nil {
			return out
		}
	}
	return []Outcome{{Kind: Notify, Msg: "Run terminé. Copie ta soute (Ctrl+A, Ctrl+C)."}}
}

// moved : le personnage a changé de système hors run. Le journal ne dit pas s'il s'est amarré en route,
// donc la soute connue n'est plus fiable.
func (t *Tracker) moved() []Outcome {
	out := t.dropAwaiting("changement de système avant la copie de soute")
	if t.base != nil {
		t.base = nil
		out = append(out, Outcome{Kind: Notify, Msg: "Changement de système : recopie ta soute avant le prochain run."})
	}
	return out
}

// dropAwaiting abandonne le run en attente de butin : la soute a pu changer, le diff ne serait plus fiable.
func (t *Tracker) dropAwaiting(reason string) []Outcome {
	if t.state != awaitingLoot {
		return nil
	}
	run := t.run
	t.state, t.base = idle, nil
	return []Outcome{{Kind: Incomplete, Run: &run, Msg: "Run non envoyé : " + reason + "."}}
}

func (t *Tracker) OnCargo(c Cargo, at time.Time) []Outcome {
	switch t.state {
	case idle:
		t.base = c
		return []Outcome{{Kind: Notify, Msg: "Soute de départ enregistrée."}}
	case inAbyss:
		t.late, t.lateAt = c, at
	case awaitingLoot:
		gained, lost := diffCargo(t.base, c)
		if len(gained) == 0 && len(lost) == 0 {
			return nil // même copie qu'avant le run : le filament consommé manquerait forcément
		}
		run := t.run
		run.Gained, run.Lost = gained, lost
		t.state, t.base = idle, c
		return []Outcome{{Kind: Complete, Run: &run}}
	}
	return nil
}

// Stop clôt la session : un run encore en attente de butin ne sera jamais complété.
func (t *Tracker) Stop() []Outcome {
	return t.dropAwaiting("session terminée avant la copie de soute")
}
