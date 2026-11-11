package main

import (
	"reflect"
	"testing"
)

func TestParseCargo(t *testing.T) {
	cases := []struct {
		name string
		text string
		want Cargo
	}{
		{"anglais", "Tranquil Exotic Filament\t2\tAbyssal Filaments\r\nScourge Light Missile\t1,234\tLight Missile\r\n",
			Cargo{"Tranquil Exotic Filament": 2, "Scourge Light Missile": 1234}},
		{"français, espace insécable", "Filament exotique placide\t1\nMissile léger Scourge\t1 234\n",
			Cargo{"Filament exotique placide": 1, "Missile léger Scourge": 1234}},
		{"quantité vide = 1, piles additionnées", "Small Shield Booster I\t\tShield Booster\nTritanium\t5\nTritanium\t7",
			Cargo{"Small Shield Booster I": 1, "Tritanium": 12}},
	}
	for _, c := range cases {
		got, ok := parseCargo(c.text)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v ok=%v, want %v", c.name, got, ok, c.want)
		}
	}
}

func TestParseCargoRejectsOtherText(t *testing.T) {
	for _, text := range []string{
		"",
		"mon mot de passe",
		"https://abysstracker.com/run/123",
		"Tritanium\tbeaucoup",
		"colonne\tunique\nligne sans tabulation",
	} {
		if c, ok := parseCargo(text); ok {
			t.Errorf("%q accepté comme soute: %v", text, c)
		}
	}
}

func TestDiffCargo(t *testing.T) {
	before := Cargo{"Tranquil Exotic Filament": 3, "Scourge Light Missile": 1000, "Nanite Repair Paste": 10}
	after := Cargo{"Tranquil Exotic Filament": 2, "Scourge Light Missile": 800, "Nanite Repair Paste": 10,
		"Triglavian Survey Database": 12, "Crystalline Isogen-10": 40}
	gained, lost := diffCargo(before, after)
	if want := (Cargo{"Triglavian Survey Database": 12, "Crystalline Isogen-10": 40}); !reflect.DeepEqual(gained, want) {
		t.Errorf("gained = %v, want %v", gained, want)
	}
	if want := (Cargo{"Tranquil Exotic Filament": 1, "Scourge Light Missile": 200}); !reflect.DeepEqual(lost, want) {
		t.Errorf("lost = %v, want %v", lost, want)
	}
}
