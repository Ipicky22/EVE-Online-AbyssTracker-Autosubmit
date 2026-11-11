package main

import (
	"regexp"
	"slices"
	"testing"
)

var verb = regexp.MustCompile(`%[a-z]`)

// Both languages must exist for every key and take the same arguments in the same order.
func TestMessagesAreComplete(t *testing.T) {
	for key, m := range messages {
		if m[0] == "" || m[1] == "" {
			t.Errorf("%s: missing translation", key)
		}
		if en, fr := verb.FindAllString(m[0], -1), verb.FindAllString(m[1], -1); !slices.Equal(en, fr) {
			t.Errorf("%s: arguments differ, en=%v fr=%v", key, en, fr)
		}
	}
}

func TestTr(t *testing.T) {
	if got := tr("fr", "sessionStarted", "Dark", 0); got != "Session en cours : Dark T0. Copie ta soute avant le premier run." {
		t.Errorf("fr: %q", got)
	}
	if got := tr("de", "sessionEnded"); got != "Session ended." {
		t.Errorf("unknown language must fall back to English, got %q", got)
	}
}
