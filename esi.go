package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

// Only public ESI calls are made: no character login is needed.
const (
	esiBase   = "https://esi.evetech.net/latest"
	userAgent = "abyssup/0.1" // generic name: no personal data is sent to ESI
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Item is an EVE item type with its English name, the only one Abyss Tracker accepts.
type Item struct {
	TypeID int
	Name   string
}

// itemLangs are the client languages whose item names can be resolved, tried in this order.
var itemLangs = []string{"en", "fr", "de", "ru", "zh"}

// resolveNames converts names shown by the client into EVE types.
// The client language is not known: try English first, then each other language for what is left.
// Names that cannot be found are absent from the result.
func resolveNames(names []string) (map[string]Item, error) {
	found := map[string]Item{}
	missing := names
	for _, lang := range itemLangs {
		if len(missing) == 0 {
			break
		}
		ids, err := universeIDs(missing, lang)
		if err != nil {
			return nil, err
		}
		var still []string
		for _, n := range missing {
			if id, ok := ids[strings.ToLower(n)]; ok {
				found[n] = Item{TypeID: id, Name: n}
			} else {
				still = append(still, n)
			}
		}
		missing = still
	}
	if len(found) == 0 {
		return found, nil
	}
	// /universe/names always returns the English name, whatever the client language.
	ids := make([]int, 0, len(found))
	for _, it := range found {
		if !slices.Contains(ids, it.TypeID) { // ESI rejects duplicates (same item copied under both its names)
			ids = append(ids, it.TypeID)
		}
	}
	body, _ := json.Marshal(ids)
	req, _ := http.NewRequest("POST", esiBase+"/universe/names/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	var res []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := doJSON(req, &res); err != nil {
		return nil, fmt.Errorf("ESI names: %w", err)
	}
	english := map[int]string{}
	for _, r := range res {
		english[r.ID] = r.Name
	}
	for n, it := range found {
		it.Name = english[it.TypeID]
		found[n] = it
	}
	return found, nil
}

// universeIDs returns lowercase name -> typeID for the names known in the given language.
func universeIDs(names []string, lang string) (map[string]int, error) {
	// shortcut: ESI accepts 500 names per call; a run's cargo has a few dozen. Chunk if that changes.
	body, _ := json.Marshal(names)
	req, _ := http.NewRequest("POST", esiBase+"/universe/ids/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", lang)
	var res struct {
		InventoryTypes []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"inventory_types"`
	}
	if err := doJSON(req, &res); err != nil {
		return nil, fmt.Errorf("ESI ids: %w", err)
	}
	ids := map[string]int{}
	for _, t := range res.InventoryTypes {
		ids[strings.ToLower(t.Name)] = t.ID
	}
	return ids, nil
}

// httpError carries the HTTP status to tell a rejection (4xx) from a transient failure.
type httpError struct {
	Status int
	Body   string
}

func (e *httpError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body) }

func doJSON(req *http.Request, out any) error {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &httpError{Status: resp.StatusCode, Body: strings.TrimSpace(string(body[:min(len(body), 300)]))}
	}
	return json.Unmarshal(body, out)
}
