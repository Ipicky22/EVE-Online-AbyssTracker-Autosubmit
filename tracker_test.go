package main

import (
	"reflect"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC)

const (
	jita  = 30000142
	abyss = 32000042
)

// feed envoie une lecture toutes les 5 s et rend tous les résultats.
type feed struct {
	tr  Tracker
	now time.Time
	out []Outcome
}

func newFeed() *feed { return &feed{now: t0} }

func (f *feed) read(system int, docked bool, n int) *feed {
	for range n {
		f.out = append(f.out, f.tr.OnReading(Reading{At: f.now, SystemID: system, Docked: docked})...)
		f.now = f.now.Add(5 * time.Second)
	}
	return f
}

func (f *feed) cargo(c Cargo) *feed {
	f.out = append(f.out, f.tr.OnCargo(c, f.now)...)
	return f
}

func (f *feed) runs(kind OutcomeKind) []*Run {
	var runs []*Run
	for _, o := range f.out {
		if o.Kind == kind {
			runs = append(runs, o.Run)
		}
	}
	return runs
}

var (
	before = Cargo{"Tranquil Exotic Filament": 2, "Ammo": 100}
	after  = Cargo{"Tranquil Exotic Filament": 1, "Ammo": 80, "Loot": 5}
)

func TestNormalRun(t *testing.T) {
	f := newFeed().cargo(before).read(jita, false, 2).read(abyss, false, 120).read(jita, false, 2).cargo(after)
	runs := f.runs(Complete)
	if len(runs) != 1 || len(f.runs(Incomplete)) != 0 {
		t.Fatalf("complete=%d incomplete=%d, want 1/0", len(runs), len(f.runs(Incomplete)))
	}
	r := runs[0]
	if !r.Success || r.Start != t0.Add(10*time.Second) || r.Duration != 600*time.Second {
		t.Errorf("run = %+v", r)
	}
	if !reflect.DeepEqual(r.Gained, Cargo{"Loot": 5}) || !reflect.DeepEqual(r.Lost, Cargo{"Tranquil Exotic Filament": 1, "Ammo": 20}) {
		t.Errorf("gained=%v lost=%v", r.Gained, r.Lost)
	}
}

func TestChainedRunsReuseLastCopy(t *testing.T) {
	after2 := Cargo{"Ammo": 60, "Loot": 9}
	f := newFeed().cargo(before).
		read(abyss, false, 10).read(jita, false, 2).cargo(after).
		read(abyss, false, 10).read(jita, false, 2).cargo(after2)
	runs := f.runs(Complete)
	if len(runs) != 2 {
		t.Fatalf("complete=%d, want 2", len(runs))
	}
	if !reflect.DeepEqual(runs[1].Gained, Cargo{"Loot": 4}) {
		t.Errorf("2e run gained=%v", runs[1].Gained)
	}
}

func TestDeath(t *testing.T) {
	f := newFeed().cargo(before).read(abyss, false, 10).read(jita, true, 2)
	runs := f.runs(Complete)
	if len(runs) != 1 || runs[0].Success || len(runs[0].Gained) != 0 {
		t.Fatalf("runs=%+v, want 1 run perdu sans butin", runs)
	}
	// La soute de départ n'est plus valable : le run suivant ne doit pas partir.
	f.read(jita, false, 2).read(abyss, false, 10).read(jita, false, 2).cargo(after)
	if len(f.runs(Complete)) != 1 || len(f.runs(Incomplete)) != 1 {
		t.Errorf("complete=%d incomplete=%d, want 1/1", len(f.runs(Complete)), len(f.runs(Incomplete)))
	}
}

func TestForgottenCopy(t *testing.T) {
	f := newFeed().cargo(before).read(abyss, false, 10).read(jita, false, 2).read(abyss, false, 10).read(jita, false, 2).cargo(after)
	if len(f.runs(Complete)) != 0 || len(f.runs(Incomplete)) != 2 {
		t.Errorf("complete=%d incomplete=%d, want 0/2", len(f.runs(Complete)), len(f.runs(Incomplete)))
	}
}

func TestDockingInvalidatesBase(t *testing.T) {
	f := newFeed().cargo(before).read(jita, true, 1).read(jita, false, 1).read(abyss, false, 10).read(jita, false, 2).cargo(after)
	if len(f.runs(Complete)) != 0 || len(f.runs(Incomplete)) != 1 {
		t.Errorf("complete=%d incomplete=%d, want 0/1", len(f.runs(Complete)), len(f.runs(Incomplete)))
	}
}

func TestDockingBeforeCopyDropsRun(t *testing.T) {
	f := newFeed().cargo(before).read(abyss, false, 10).read(jita, false, 2).read(jita, true, 1).cargo(after)
	if len(f.runs(Complete)) != 0 || len(f.runs(Incomplete)) != 1 {
		t.Errorf("complete=%d incomplete=%d, want 0/1", len(f.runs(Complete)), len(f.runs(Incomplete)))
	}
}

func TestSingleStrayReadingIsIgnored(t *testing.T) {
	f := newFeed().cargo(before).read(jita, false, 2).read(abyss, false, 1).read(jita, false, 2).
		read(abyss, false, 10).read(jita, false, 1).read(abyss, false, 10).read(jita, false, 2).cargo(after)
	runs := f.runs(Complete)
	if len(runs) != 1 {
		t.Fatalf("complete=%d, want 1", len(runs))
	}
	if want := 21 * 5 * time.Second; runs[0].Duration != want {
		t.Errorf("duration=%v, want %v", runs[0].Duration, want)
	}
}

func TestUnchangedCopyAndCopyInsideAbyssAreIgnored(t *testing.T) {
	f := newFeed().cargo(before).read(abyss, false, 5).cargo(Cargo{"Ammo": 1}).read(abyss, false, 5).read(jita, false, 2).cargo(before)
	if len(f.runs(Complete)) != 0 {
		t.Fatalf("un run est parti avec une copie sans changement")
	}
	if f.cargo(after); len(f.runs(Complete)) != 1 {
		t.Errorf("complete=%d, want 1", len(f.runs(Complete)))
	}
}

func TestCopyJustBeforeExitIsDetected(t *testing.T) {
	// Copie faite dès la sortie, avant que la 2e lecture ne la confirme : pas besoin de recopier.
	f := newFeed().cargo(before).read(abyss, false, 10).read(jita, false, 1).cargo(after).read(jita, false, 1)
	if len(f.runs(Complete)) != 1 {
		t.Fatalf("complete=%d, want 1", len(f.runs(Complete)))
	}
	// Copie faite en plein run, longtemps avant la sortie : elle ne vaut pas copie d'après-run.
	f = newFeed().cargo(before).read(abyss, false, 2).cargo(after).read(abyss, false, 10).read(jita, false, 2)
	if len(f.runs(Complete)) != 0 {
		t.Fatalf("une copie de milieu de run a été prise pour la copie finale")
	}
}

func TestStopDropsAwaitingRun(t *testing.T) {
	f := newFeed().cargo(before).read(abyss, false, 10).read(jita, false, 2)
	f.out = append(f.out, f.tr.Stop()...)
	if len(f.runs(Incomplete)) != 1 {
		t.Errorf("incomplete=%d, want 1", len(f.runs(Incomplete)))
	}
}
