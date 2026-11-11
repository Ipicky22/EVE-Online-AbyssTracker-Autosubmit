package main

import "time"

// Plage des systèmes abyssaux (AD001 à AD200) dans ESI.
const (
	abyssMin = 32000001
	abyssMax = 32000200
)

// Reading est une lecture de position ESI.
type Reading struct {
	At       time.Time
	SystemID int
	Docked   bool
}

func (r Reading) inAbyss() bool { return r.SystemID >= abyssMin && r.SystemID <= abyssMax }

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

// Tracker transforme lectures de position et copies de soute en runs.
// Aucune entrée/sortie ici : tout passe par les valeurs de retour.
type Tracker struct {
	state     trackerState
	base      Cargo    // soute avant le prochain run ; nil = inconnue
	candidate *Reading // 1re lecture d'un changement, à confirmer par la suivante
	run       Run
	late      Cargo // dernière copie faite alors que le run semblait encore en cours
	lateAt    time.Time
}

// lateGrace couvre le retard de détection de la sortie : cache ESI (5 s) + intervalle de lecture (5 s) + marge.
const lateGrace = 15 * time.Second

func (t *Tracker) OnReading(r Reading) []Outcome {
	switch t.state {
	case idle, awaitingLoot:
		if r.inAbyss() {
			if t.candidate == nil {
				t.candidate = &r
				return nil
			}
			return t.enter(*t.candidate)
		}
		t.candidate = nil
		if r.Docked {
			return t.docked()
		}
	case inAbyss:
		if !r.inAbyss() {
			if t.candidate == nil {
				t.candidate = &r
				return nil
			}
			return t.exit(*t.candidate)
		}
		t.candidate = nil
	}
	return nil
}

func (t *Tracker) enter(first Reading) []Outcome {
	out := t.dropAwaiting("nouveau run lancé sans copie de soute")
	t.candidate = nil
	t.state = inAbyss
	t.run, t.late = Run{Start: first.At}, nil
	if t.base == nil {
		out = append(out, Outcome{Kind: Notify, Msg: "Run détecté sans copie de soute au départ : il ne sera pas envoyé."})
	}
	return out
}

func (t *Tracker) exit(first Reading) []Outcome {
	t.candidate = nil
	t.run.Duration = first.At.Sub(t.run.Start)
	// Mort dans l'Abysse : la capsule est détruite aussi, le personnage se réveille amarré.
	if first.Docked {
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
	// La sortie est connue 5 à 15 s après coup : une copie faite juste avant la détection est celle d'après-run.
	if late := t.late; late != nil && first.At.Sub(t.lateAt) <= lateGrace {
		if out := t.OnCargo(late, t.lateAt); out != nil {
			return out
		}
	}
	return []Outcome{{Kind: Notify, Msg: "Run terminé. Copie ta soute (Ctrl+A, Ctrl+C)."}}
}

func (t *Tracker) docked() []Outcome {
	out := t.dropAwaiting("amarrage avant la copie de soute")
	if t.base != nil {
		t.base = nil
		out = append(out, Outcome{Kind: Notify, Msg: "Amarré : recopie ta soute avant le prochain run."})
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
