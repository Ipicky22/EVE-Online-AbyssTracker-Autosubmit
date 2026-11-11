package main

import (
	"strconv"
	"strings"
	"unicode"
)

// Cargo associe le nom affiché par le client (anglais ou français) à une quantité.
type Cargo map[string]int

// parseCargo reconnaît une copie de soute EVE (Ctrl+A, Ctrl+C) : une pile par ligne,
// colonnes séparées par des tabulations, nom puis quantité.
// Tout autre texte est refusé : le presse-papiers contient aussi ce que l'utilisateur copie ailleurs.
func parseCargo(text string) (Cargo, bool) {
	cargo := Cargo{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		name := strings.TrimSpace(cols[0])
		if len(cols) < 2 || name == "" {
			return nil, false
		}
		qty, ok := parseQuantity(cols[1])
		if !ok {
			return nil, false
		}
		cargo[name] += qty
	}
	return cargo, len(cargo) > 0
}

// parseQuantity accepte "1,234" (client anglais) et "1 234" avec espace insécable (client français).
// Une quantité vide vaut 1 : le client n'affiche rien pour un objet non empilable.
func parseQuantity(s string) (int, bool) {
	var digits strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case unicode.IsSpace(r), r == ',', r == '.', r == ' ', r == ' ':
		default:
			return 0, false
		}
	}
	if digits.Len() == 0 {
		return 1, strings.TrimSpace(s) == ""
	}
	n, err := strconv.Atoi(digits.String())
	return n, err == nil
}

// diffCargo rend ce qui est apparu et ce qui a disparu entre deux copies de soute.
func diffCargo(before, after Cargo) (gained, lost Cargo) {
	gained, lost = Cargo{}, Cargo{}
	for name, qty := range after {
		if d := qty - before[name]; d > 0 {
			gained[name] = d
		}
	}
	for name, qty := range before {
		if d := qty - after[name]; d > 0 {
			lost[name] = d
		}
	}
	return gained, lost
}
