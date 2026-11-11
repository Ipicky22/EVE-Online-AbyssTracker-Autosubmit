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

// Seuls des appels ESI publics sont faits : aucune connexion du personnage n'est nécessaire.
const (
	esiBase   = "https://esi.evetech.net/latest"
	userAgent = "abyssup/0.1" // nom générique : aucune donnée personnelle envoyée à ESI
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Item est un type d'objet EVE avec son nom anglais, seul accepté par Abyss Tracker.
type Item struct {
	TypeID int
	Name   string
}

// resolveNames convertit des noms affichés par le client en types EVE.
// Le client peut être en anglais ou en français : on essaie l'anglais, puis le français pour le reste.
// Les noms introuvables sont absents du résultat.
func resolveNames(names []string) (map[string]Item, error) {
	found := map[string]Item{}
	missing := names
	for _, lang := range []string{"en", "fr"} {
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
	// /universe/names rend toujours le nom anglais, quelle que soit la langue du client.
	ids := make([]int, 0, len(found))
	for _, it := range found {
		if !slices.Contains(ids, it.TypeID) { // ESI refuse les doublons (même objet copié sous ses deux noms)
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

// universeIDs rend nom en minuscules -> typeID pour les noms connus dans la langue donnée.
func universeIDs(names []string, lang string) (map[string]int, error) {
	// shortcut: ESI accepte 500 noms par appel ; une soute de run en compte quelques dizaines. Découper si ça change.
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

// httpError porte le code HTTP pour distinguer un refus (4xx) d'une panne passagère.
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
