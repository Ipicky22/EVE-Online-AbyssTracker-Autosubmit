package main

import (
	"reflect"
	"testing"
)

func TestFilamentName(t *testing.T) {
	if got := filamentName(Session{Tier: 0, Weather: "Dark"}); got != "Tranquil Dark Filament" {
		t.Errorf("got %q", got)
	}
	for name, want := range map[string]bool{
		"Tranquil Dark Filament":     true,
		"Cataclysmic Gamma Filament": true,
		"Dark Filament":              false,
		"Tranquil Dark":              false,
		"Tranquil Shiny Filament":    false,
		"Triglavian Survey Database": false,
	} {
		if isFilament(name) != want {
			t.Errorf("isFilament(%q) = %v, want %v", name, !want, want)
		}
	}
}

func TestFixFilament(t *testing.T) {
	fil := Item{TypeID: 1, Name: "Tranquil Dark Filament"}
	f := func(count int) lootItem { return lootItem{Name: fil.Name, Count: count, TypeID: 1} }
	other := lootItem{Name: "Calm Dark Filament", Count: 3, TypeID: 2}
	db := lootItem{Name: "Triglavian Survey Database", Count: 9, TypeID: 3}
	ammo := lootItem{Name: "Scourge Light Missile", Count: 200, TypeID: 4}

	cases := []struct {
		name                 string
		gained, lost         []lootItem
		used                 int
		wantGained, wantLost []lootItem
	}{
		{"nothing looted: unchanged", []lootItem{db}, []lootItem{ammo, f(3)}, 3, []lootItem{db}, []lootItem{ammo, f(3)}},
		{"cruiser, nothing looted: unchanged", []lootItem{db}, []lootItem{f(1)}, 1, []lootItem{db}, []lootItem{f(1)}},
		{"one looted: two lost becomes three lost, one gained", []lootItem{db}, []lootItem{ammo, f(2)}, 3, []lootItem{f(1), db}, []lootItem{ammo, f(3)}},
		{"as many looted as used: filament absent from the diff", []lootItem{db}, []lootItem{ammo}, 3, []lootItem{f(3), db}, []lootItem{ammo, f(3)}},
		{"more looted than used: net gain of two", []lootItem{f(2), db}, []lootItem{ammo}, 3, []lootItem{f(5), db}, []lootItem{ammo, f(3)}},
		{"another filament was consumed: settings do not match, unchanged", []lootItem{db}, []lootItem{other}, 3, []lootItem{db}, []lootItem{other}},
		{"more lost than a run uses: unchanged", []lootItem{db}, []lootItem{f(5)}, 3, []lootItem{db}, []lootItem{f(5)}},
		{"unknown hull: unchanged", []lootItem{db}, []lootItem{f(2)}, 0, []lootItem{db}, []lootItem{f(2)}},
	}
	for _, c := range cases {
		g, l := fixFilament(c.gained, c.lost, fil, c.used)
		if !reflect.DeepEqual(g, c.wantGained) || !reflect.DeepEqual(l, c.wantLost) {
			t.Errorf("%s:\n gained %v, want %v\n lost   %v, want %v", c.name, g, c.wantGained, l, c.wantLost)
		}
	}
}

func TestBuildTelemetryFixesFilament(t *testing.T) {
	items := map[string]Item{
		"Filament sombre placide": {TypeID: 1, Name: "Tranquil Dark Filament"}, // name as a French client shows it
		"Tranquil Dark Filament":  {TypeID: 1, Name: "Tranquil Dark Filament"},
		"Loot":                    {TypeID: 3, Name: "Loot"},
	}
	s := Session{Tier: 0, Weather: "Dark", Hull: "Frigate"}
	run := &Run{Success: true, Gained: Cargo{"Loot": 5}, Lost: Cargo{"Filament sombre placide": 2}}
	got := buildTelemetry(run, s, Character{}, items)
	wantLost := []lootItem{{Name: "Tranquil Dark Filament", Count: 3, TypeID: 1}}
	wantGained := []lootItem{{Name: "Loot", Count: 5, TypeID: 3}, {Name: "Tranquil Dark Filament", Count: 1, TypeID: 1}}
	if !reflect.DeepEqual(got.LootLost, wantLost) || !reflect.DeepEqual(got.LootGained, wantGained) {
		t.Errorf("gained %v, lost %v", got.LootGained, got.LootLost)
	}

	// A lost run carries no loot: nothing to restore.
	dead := buildTelemetry(&Run{Gained: Cargo{}, Lost: Cargo{}}, s, Character{}, items)
	if len(dead.LootGained) != 0 || len(dead.LootLost) != 0 {
		t.Errorf("lost run: gained %v, lost %v", dead.LootGained, dead.LootLost)
	}
}
