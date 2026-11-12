package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sent is a run as the pending/ file holds it: 600 s, T6 Exotic.
const sent = `{"runStartTime": 1700000000, "runDurationSeconds": 600, "weather": "Exotic", "tier": 6, "success": true}`

func fakeAbyssTracker(t *testing.T, status int, body string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/abyss-tracker/runs/run-1" || r.Header.Get("X-API-KEY") != "" {
			t.Errorf("unexpected request: %s, key %q", r.URL.Path, r.Header.Get("X-API-KEY"))
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	oldBase, oldRetry := ewbBase, historyRetry
	ewbBase, historyRetry = srv.URL, 0
	t.Cleanup(func() { srv.Close(); ewbBase, historyRetry = oldBase, oldRetry })
}

func renderLive(a *App) string {
	rec := httptest.NewRecorder()
	a.handleLive(rec, httptest.NewRequest("GET", "/live", nil))
	return rec.Body.String()
}

func TestRememberShowsAbyssTrackerValues(t *testing.T) {
	// Answer in the server's own shape, as read from a real public run.
	fakeAbyssTracker(t, 200, `{"Run":{"Id":"run-1","Survived":true,"TimeSpendSeconds":1065,"LootValue":121056680.1},"Error":false}`)
	a := &App{cfg: Config{Lang: "fr"}}
	a.remember("run-1", []byte(sent))

	if len(a.history) != 1 || !a.history[0].Valued || a.history[0].Pending {
		t.Fatalf("history: %+v", a.history)
	}
	html := renderLive(a)
	start := time.Unix(1700000000, 0).Format("15:04") // shown in the PC's time zone
	for _, want := range []string{"Exotic T6", start, "17 min 45 s", `<span class="isk">121,1 M</span>`,
		"<strong>121,1 M ISK</strong>", `href="https://abysstracker.com/run/run-1"`} {
		if !strings.Contains(html, want) {
			t.Errorf("history lacks %q:\n%s", want, html)
		}
	}
}

func TestRememberKeepsTheRunWhenAbyssTrackerDoesNotAnswer(t *testing.T) {
	for name, c := range map[string]struct {
		status int
		body   string
	}{
		"server error":  {500, "boom"},
		"run not found": {200, `{"Run":null,"Error":true,"Message":"No run found"}`},
	} {
		fakeAbyssTracker(t, c.status, c.body)
		a := &App{cfg: Config{Lang: "en"}}
		a.remember("run-1", []byte(sent))

		if len(a.history) != 1 || a.history[0].Valued || a.history[0].Pending {
			t.Fatalf("%s: history: %+v", name, a.history)
		}
		// The duration is the one that was sent; no loot value is made up, and nothing claims to be still coming.
		html := renderLive(a)
		if !strings.Contains(html, "10 min 00 s") || !strings.Contains(html, "<td>—</td>") ||
			strings.Contains(html, `class="isk"`) || strings.Contains(html, "Valuing") {
			t.Errorf("%s:\n%s", name, html)
		}
	}
}

func TestHistoryStates(t *testing.T) {
	a := &App{cfg: Config{Lang: "en"}}
	if html := renderLive(a); !strings.Contains(html, "No run sent yet.") {
		t.Errorf("empty history:\n%s", html)
	}
	a.history = []histRun{
		{ID: "new", Survived: true, Pending: true},
		{ID: "dead", Valued: true},
		{ID: "old", Survived: true, Valued: true, Loot: 36.2e6},
	}
	html := renderLive(a)
	for _, want := range []string{"Valuing…", `<span class="lost">Lost</span>`, "Runs <strong>3</strong>",
		"Lost <strong>1</strong>", "<strong>36.2 M ISK</strong>"} {
		if !strings.Contains(html, want) {
			t.Errorf("history lacks %q:\n%s", want, html)
		}
	}
	if strings.Index(html, "/run/new") > strings.Index(html, "/run/old") {
		t.Error("newest run must come first")
	}
}

func TestRememberPutsTheNewestFirst(t *testing.T) {
	fakeAbyssTracker(t, 500, "")
	a := &App{cfg: Config{Lang: "en"}, history: []histRun{{ID: "older"}}}
	a.remember("run-1", []byte(sent))
	if len(a.history) != 2 || a.history[0].ID != "run-1" || a.history[1].ID != "older" {
		t.Errorf("order: %+v", a.history)
	}
}

func TestISK(t *testing.T) {
	if got := isk("fr", 41.84e6); got != "41,8 M" {
		t.Errorf("fr: %q", got)
	}
	if got := isk("en", 41.84e6); got != "41.8 M" {
		t.Errorf("en: %q", got)
	}
}
