package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// histRun is one run sent during the session, as the page lists it. Kept in memory only.
type histRun struct {
	ID       string
	Start    time.Time
	Weather  string
	Tier     int
	Survived bool
	Duration time.Duration
	Loot     float64 // ISK
	Valued   bool    // Abyss Tracker answered: Survived, Duration and Loot are its values, not ours
	Pending  bool    // Abyss Tracker is being asked
}

func (h histRun) URL() string  { return runURL + h.ID }
func (h histRun) Time() string { return h.Start.Format("15:04") }

func (h histRun) Length() string {
	s := int(h.Duration.Seconds())
	return fmt.Sprintf("%d min %02d s", s/60, s%60)
}

// isk writes an ISK amount in millions, with the decimal mark of the interface language.
func isk(lang string, v float64) string {
	s := fmt.Sprintf("%.1f M", v/1e6)
	if lang == "fr" || lang == "de" || lang == "ru" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

type runDetail struct {
	Survived         bool
	TimeSpendSeconds int
	LootValue        float64
}

// fetchRun reads a run back from Abyss Tracker. The call is public: no token is sent.
func fetchRun(id string) (runDetail, error) {
	req, _ := http.NewRequest("GET", ewbBase+"/abyss-tracker/runs/"+url.PathEscape(id), nil)
	var res struct {
		Error   bool
		Message string
		Run     *runDetail
	}
	if err := doJSON(req, &res); err != nil {
		return runDetail{}, fmt.Errorf("run detail: %w", err)
	}
	if res.Error || res.Run == nil {
		return runDetail{}, fmt.Errorf("run detail: %s", res.Message)
	}
	return *res.Run, nil
}

var historyRetry = 3 * time.Second

// remember adds a run that was just sent to the session history: at once with what was sent, then with
// what Abyss Tracker made of it.
func (a *App) remember(runID string, body []byte) {
	var t telemetry
	json.Unmarshal(body, &t)
	h := histRun{ID: runID, Start: time.Unix(t.RunStartTime, 0), Weather: t.Weather, Tier: t.Tier,
		Survived: t.Success, Duration: time.Duration(t.RunDurationSeconds) * time.Second, Pending: true}
	a.mu.Lock()
	a.history = slices.Insert(a.history, 0, h)
	a.mu.Unlock()

	h.Pending = false
	var err error
	// shortcut: 3 inline attempts rather than a retry queue; after that the row keeps what was sent, without a loot value.
	for range 3 {
		var d runDetail
		if d, err = fetchRun(runID); err == nil {
			h.Survived, h.Duration = d.Survived, time.Duration(d.TimeSpendSeconds)*time.Second
			h.Loot, h.Valued = d.LootValue, true
			break
		}
		time.Sleep(historyRetry)
	}
	if err != nil {
		log.Printf("history: %v", err)
	}
	a.mu.Lock()
	for i := range a.history { // no longer there when a new session started meanwhile
		if a.history[i].ID == runID {
			a.history[i] = h
		}
	}
	a.mu.Unlock()
}
