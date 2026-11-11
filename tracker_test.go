package main

import (
	"reflect"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 11, 12, 50, 0, 0, time.UTC)

const (
	home  = "Alillere"
	other = "Dodixie"
)

// feed replays a sequence of system changes and cargo copies.
type feed struct {
	tr  Tracker
	now time.Time
	out []Outcome
}

func newFeed() *feed {
	f := &feed{now: t0}
	f.tr.SetLocation(LocalEvent{System: home})
	return f
}

// wait moves the clock forward.
func (f *feed) wait(d time.Duration) *feed { f.now = f.now.Add(d); return f }

func (f *feed) to(system string) *feed {
	f.out = append(f.out, f.tr.OnLocal(LocalEvent{At: f.now, System: system}, f.now)...)
	return f
}

func (f *feed) abyss() *feed {
	f.out = append(f.out, f.tr.OnLocal(LocalEvent{At: f.now, System: "Inconnu", Abyss: true}, f.now)...)
	return f
}

func (f *feed) cargo(c Cargo) *feed {
	f.out = append(f.out, f.tr.OnCargo(c, f.now)...)
	return f
}

// run plays a 10-minute run that exits into the given system.
func (f *feed) run(exit string) *feed {
	return f.wait(time.Minute).abyss().wait(10 * time.Minute).to(exit).wait(30 * time.Second)
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

func (f *feed) expect(t *testing.T, complete, incomplete int) {
	t.Helper()
	if c, i := len(f.runs(Complete)), len(f.runs(Incomplete)); c != complete || i != incomplete {
		t.Fatalf("complete=%d incomplete=%d, want %d/%d", c, i, complete, incomplete)
	}
}

var (
	before = Cargo{"Tranquil Exotic Filament": 2, "Ammo": 100}
	after  = Cargo{"Tranquil Exotic Filament": 1, "Ammo": 80, "Loot": 5}
)

func TestNormalRun(t *testing.T) {
	f := newFeed().cargo(before).run(home).cargo(after)
	f.expect(t, 1, 0)
	r := f.runs(Complete)[0]
	if !r.Success || r.Start != t0.Add(time.Minute) || r.Duration != 10*time.Minute {
		t.Errorf("run = %+v", r)
	}
	if !reflect.DeepEqual(r.Gained, Cargo{"Loot": 5}) || !reflect.DeepEqual(r.Lost, Cargo{"Tranquil Exotic Filament": 1, "Ammo": 20}) {
		t.Errorf("gained=%v lost=%v", r.Gained, r.Lost)
	}
}

func TestChainedRunsReuseLastCopy(t *testing.T) {
	f := newFeed().cargo(before).run(home).cargo(after).run(home).cargo(Cargo{"Ammo": 60, "Loot": 9})
	f.expect(t, 2, 0)
	if got := f.runs(Complete)[1].Gained; !reflect.DeepEqual(got, Cargo{"Loot": 4}) {
		t.Errorf("2e run gained=%v", got)
	}
}

func TestDeathIsExitInAnotherSystem(t *testing.T) {
	f := newFeed().cargo(before).run(other)
	f.expect(t, 1, 0)
	if r := f.runs(Complete)[0]; r.Success || len(r.Gained) != 0 {
		t.Fatalf("run = %+v, want un run perdu sans butin", r)
	}
	// The starting cargo is no longer valid: the next run must not be sent.
	f.run(other).cargo(after)
	f.expect(t, 1, 1)
}

func TestForgottenCopy(t *testing.T) {
	f := newFeed().cargo(before).run(home).run(home).cargo(after)
	f.expect(t, 0, 2)
}

func TestTravelInvalidatesBase(t *testing.T) {
	// Round trip to another system: the log does not say whether the cargo was emptied in a station.
	f := newFeed().cargo(before).to(other).to(home).run(home).cargo(after)
	f.expect(t, 0, 1)
}

func TestTravelBeforeCopyDropsRun(t *testing.T) {
	f := newFeed().cargo(before).run(home).to(other).cargo(after)
	f.expect(t, 0, 1)
}

func TestCopyJustBeforeExitIsRead(t *testing.T) {
	// Copy made right at the exit, before the log line is read.
	f := newFeed().cargo(before).wait(time.Minute).abyss().wait(10 * time.Minute)
	exit := f.now
	f.wait(2 * time.Second).cargo(after)
	f.out = append(f.out, f.tr.OnLocal(LocalEvent{At: exit, System: home}, f.now)...)
	f.expect(t, 1, 0)

	// Copy made mid-run, long before the exit: it does not count as the after-run copy.
	f = newFeed().cargo(before).wait(time.Minute).abyss().wait(time.Minute).cargo(after).wait(9 * time.Minute).to(home)
	f.expect(t, 0, 0)
}

func TestUnchangedCopyIsIgnored(t *testing.T) {
	f := newFeed().cargo(before).run(home).cargo(before)
	f.expect(t, 0, 0)
	f.cargo(after)
	f.expect(t, 1, 0)
}

func TestSessionStartedMidRun(t *testing.T) {
	f := &feed{now: t0}
	f.tr.SetLocation(LocalEvent{System: "Inconnu", Abyss: true})
	f.wait(5 * time.Minute).to(home)
	f.expect(t, 0, 0)
	// The next run goes through normally.
	f.cargo(before).run(home).cargo(after)
	f.expect(t, 1, 0)
}

func TestStopDropsAwaitingRun(t *testing.T) {
	f := newFeed().cargo(before).run(home)
	f.out = append(f.out, f.tr.Stop()...)
	f.expect(t, 0, 1)
}
