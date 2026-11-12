package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSteps(t *testing.T) {
	for _, c := range []struct {
		name       string
		stage      trackerState
		base       bool
		head, hint string
		states     string
	}{
		{"session just started", idle, false, "headNeedBase", "copyBefore", "now todo todo"},
		{"cargo copied", idle, true, "headReady", "", "done now todo"},
		{"in the Abyss", inAbyss, true, "headInAbyss", "", "done now todo"},
		{"in the Abyss without a copy", inAbyss, false, "headInAbyss", "", "miss now todo"},
		{"back from the Abyss", awaitingLoot, true, "headAwait", "copyAfter", "done done now"},
	} {
		head, hint, s := steps(c.stage, c.base)
		states := s[0].State + " " + s[1].State + " " + s[2].State
		if head != c.head || hint != c.hint || states != c.states {
			t.Errorf("%s: got %s, %q, %s", c.name, head, hint, states)
		}
		// Every text the page will look up must exist.
		for _, key := range []string{head, s[0].Title, s[0].Note, s[1].Title, s[1].Note, s[2].Title, s[2].Note} {
			if messagesEN[key] == "" {
				t.Errorf("%s: no text for %q", c.name, key)
			}
		}
	}
}

// Each page must render to its end in every language and every session state:
// a template error stops the output halfway and is only logged.
func TestPagesRender(t *testing.T) {
	get := func(a *App, path string) string {
		rec := httptest.NewRecorder()
		a.routes().ServeHTTP(rec, httptest.NewRequest("GET", "http://localhost:47813"+path, nil))
		return rec.Body.String()
	}
	for _, lang := range langs {
		for name, tr := range map[string]*Tracker{"no session": nil, "waiting for a copy": {}, "in the Abyss": {state: inAbyss},
			"back from the Abyss": {state: awaitingLoot, base: Cargo{}}} {
			a := &App{cfg: Config{Lang: lang, Session: Session{Weather: "Dark", Tier: 4, Hull: "Cruiser"}}, tracker: tr,
				char: Character{Name: "Pilot"}, status: "12:00:00 — status", history: []histRun{{ID: "x", Survived: true, Valued: true}},
				update: release{Tag: "v9.9.9", URL: "https://github.com/x"}}
			page := get(a, "/")
			if !strings.HasSuffix(strings.TrimSpace(page), "</html>") || !strings.Contains(page, `value="Dark" checked`) ||
				!strings.Contains(page, "— Raging") || !strings.Contains(page, "v9.9.9") {
				t.Errorf("%s, %s: session page incomplete:\n%s", lang, name, page)
			}
			if tr != nil && !strings.Contains(page, "Pilot") {
				t.Errorf("%s, %s: character missing", lang, name)
			}
			if live := get(a, "/live"); !strings.Contains(live, "12:00:00 — status") || !strings.Contains(live, "/run/x") {
				t.Errorf("%s, %s: live block incomplete:\n%s", lang, name, live)
			}
		}
		a := &App{cfg: Config{Lang: lang}, dir: "/data/dir"}
		if page := get(a, "/settings?saved"); !strings.HasSuffix(strings.TrimSpace(page), "</html>") ||
			!strings.Contains(page, "/data/dir") || !strings.Contains(page, `value="`+lang+`" checked`) ||
			!strings.Contains(page, `class="saved"`) || !strings.Contains(page, `class="banner"`) {
			t.Errorf("%s: settings page incomplete:\n%s", lang, page)
		}
	}
}
