package main

import "time"

// Run is a detected run, before the session settings are added.
type Run struct {
	Start    time.Time
	Duration time.Duration
	Success  bool
	Gained   Cargo
	Lost     Cargo
}

type OutcomeKind int

const (
	Notify     OutcomeKind = iota // message only
	Complete                      // run to send
	Incomplete                    // run without reliable loot: kept locally, never sent
)

type Outcome struct {
	Kind OutcomeKind
	Run  *Run
	Msg  string // key in messages, translated when shown
	Args []any
}

type trackerState int

const (
	idle trackerState = iota
	inAbyss
	awaitingLoot
)

// Tracker turns system changes and cargo copies into runs.
// No I/O here: everything goes through return values.
type Tracker struct {
	state  trackerState
	base   Cargo  // cargo before the next run; nil = unknown
	system string // last known system outside the Abyss
	entry  string // system the current run was started from
	ignore bool   // run already in progress when the session started: neither start nor cargo known
	run    Run
	late   Cargo // last copy made while the run still looked in progress
	lateAt time.Time
}

// lateGrace covers the delay between the real exit and its line being read from the log.
// It is measured on the PC clock (time of the copy, time the line was read):
// the EVE time written in the log may be offset from the PC clock.
const lateGrace = 15 * time.Second

// SetLocation gives the location read from the log history, before any new event.
func (t *Tracker) SetLocation(e LocalEvent) {
	if e.Abyss {
		t.state, t.ignore = inAbyss, true
		return
	}
	t.system = e.System
}

// OnLocal handles a system change; seen is the PC time at which the line was read.
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
	out := t.dropAwaiting("notSentNewRun")
	t.state, t.entry = inAbyss, t.system
	t.run, t.late = Run{Start: at}, nil
	if t.base == nil {
		out = append(out, Outcome{Kind: Notify, Msg: "noBaseAtStart"})
	}
	return out
}

func (t *Tracker) exit(e LocalEvent, seen time.Time) []Outcome {
	entry := t.entry
	t.system = e.System
	if t.ignore {
		t.state, t.ignore = idle, false
		return []Outcome{{Kind: Notify, Msg: "midRunIgnored"}}
	}
	t.run.Duration = e.At.Sub(t.run.Start)
	// Death in the Abyss: the capsule is destroyed too, the character wakes up in their clone.
	// shortcut: undetectable when the clone is in the system the filament was used in.
	if entry != "" && e.System != entry {
		t.state, t.base = idle, nil
		run := t.run
		run.Gained, run.Lost = Cargo{}, Cargo{}
		return []Outcome{{Kind: Complete, Run: &run, Msg: "runLost"}}
	}
	if t.base == nil {
		t.state = idle
		run := t.run
		return []Outcome{{Kind: Incomplete, Run: &run, Msg: "notSentNoBase"}}
	}
	t.state = awaitingLoot
	t.run.Success = true
	// A copy made just before the exit is read from the log is the after-run copy.
	if late := t.late; late != nil && seen.Sub(t.lateAt) <= lateGrace {
		if out := t.OnCargo(late, t.lateAt); out != nil {
			return out
		}
	}
	return []Outcome{{Kind: Notify, Msg: "runDoneCopy"}}
}

// moved: the character changed system outside a run. The log does not say whether they docked on the way,
// so the known cargo is no longer reliable.
func (t *Tracker) moved() []Outcome {
	out := t.dropAwaiting("notSentMoved")
	if t.base != nil {
		t.base = nil
		out = append(out, Outcome{Kind: Notify, Msg: "movedRecopy"})
	}
	return out
}

// dropAwaiting gives up the run waiting for its loot: the cargo may have changed, the diff would be unreliable.
func (t *Tracker) dropAwaiting(msg string) []Outcome {
	if t.state != awaitingLoot {
		return nil
	}
	run := t.run
	t.state, t.base = idle, nil
	return []Outcome{{Kind: Incomplete, Run: &run, Msg: msg}}
}

func (t *Tracker) OnCargo(c Cargo, at time.Time) []Outcome {
	switch t.state {
	case idle:
		t.base = c
		return []Outcome{{Kind: Notify, Msg: "baseSaved"}}
	case inAbyss:
		t.late, t.lateAt = c, at
	case awaitingLoot:
		gained, lost := diffCargo(t.base, c)
		if len(gained) == 0 && len(lost) == 0 {
			return nil // same copy as before the run: the consumed filament would necessarily be missing
		}
		run := t.run
		run.Gained, run.Lost = gained, lost
		t.state, t.base = idle, c
		return []Outcome{{Kind: Complete, Run: &run}}
	}
	return nil
}

// Progress tells the page where the current run stands, and whether a cargo copy made before it is known.
// A run already in progress when the session started has none that counts.
func (t *Tracker) Progress() (stage trackerState, base bool) {
	return t.state, t.base != nil && !t.ignore
}

// Stop ends the session: a run still waiting for its loot will never be completed.
func (t *Tracker) Stop() []Outcome {
	return t.dropAwaiting("notSentStopped")
}
