package main

import (
	"strconv"
	"strings"
	"unicode"
)

// Cargo maps the item name shown by the client (English or French) to a quantity.
type Cargo map[string]int

// parseCargo recognises an EVE cargo hold copy (Ctrl+A, Ctrl+C): one stack per line,
// tab-separated columns, name then quantity.
// Any other text is rejected: the clipboard also holds whatever the user copies elsewhere.
func parseCargo(text string) (Cargo, bool) {
	cargo := Cargo{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		// The client marks translated names with a trailing "*": ESI does not know that mark.
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(cols[0]), "*"))
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

// parseQuantity accepts "1,234" (English client) and "1 234" with a non-breaking space (French client).
// An empty quantity means 1: the client shows nothing for a non-stackable item.
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

// diffCargo returns what appeared and what disappeared between two cargo copies.
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
