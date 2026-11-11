package main

import (
	"regexp"
	"slices"
	"testing"
)

var verb = regexp.MustCompile(`%[a-z]`)

// Every language must have every text, with the same arguments in the same order as English.
func TestMessagesAreComplete(t *testing.T) {
	if len(messages) != len(langs) {
		t.Fatalf("%d message tables for %d languages", len(messages), len(langs))
	}
	for _, lang := range langs {
		if langNames[lang] == "" {
			t.Errorf("%s: no display name", lang)
		}
		table := messages[lang]
		if len(table) != len(messagesEN) {
			t.Errorf("%s: %d texts, English has %d", lang, len(table), len(messagesEN))
		}
		for key, en := range messagesEN {
			if table[key] == "" {
				t.Errorf("%s: %s is missing", lang, key)
			}
			if a, b := verb.FindAllString(en, -1), verb.FindAllString(table[key], -1); !slices.Equal(a, b) {
				t.Errorf("%s: %s takes %v, English takes %v", lang, key, b, a)
			}
		}
	}
}

func TestTr(t *testing.T) {
	if got := tr("fr", "sessionStarted", "Dark", 0); got != "Session en cours : Dark T0. Copie ta soute avant le premier run." {
		t.Errorf("fr: %q", got)
	}
	if got := tr("xx", "sessionEnded"); got != "Session ended." {
		t.Errorf("unknown language must fall back to English, got %q", got)
	}
}
