package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	ssoAuthorize = "https://login.eveonline.com/v2/oauth/authorize"
	ssoToken     = "https://login.eveonline.com/v2/oauth/token"
	esiBase      = "https://esi.evetech.net/latest"
	esiScope     = "esi-location.read_location.v1"
	userAgent    = "abyssup/0.1" // nom générique : aucune donnée personnelle envoyée à ESI
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Token est la session SSO du personnage.
type Token struct {
	Access        string
	Refresh       string
	Expiry        time.Time
	CharacterID   int
	CharacterName string
}

func (t *Token) valid() bool { return t != nil && time.Until(t.Expiry) > 30*time.Second }

func randomString() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// authorizeURL prépare la connexion SSO en PKCE : pas de secret client à stocker.
func authorizeURL(clientID, redirect, state, verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"response_type":         {"code"},
		"redirect_uri":          {redirect},
		"client_id":             {clientID},
		"scope":                 {esiScope},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	return ssoAuthorize + "?" + q.Encode()
}

func exchangeCode(clientID, code, verifier string) (*Token, error) {
	return requestToken(url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "code_verifier": {verifier},
	})
}

func refreshToken(clientID, refresh string) (*Token, error) {
	return requestToken(url.Values{
		"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh},
	})
}

func requestToken(form url.Values) (*Token, error) {
	req, _ := http.NewRequest("POST", ssoToken, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := doJSON(req, &res); err != nil {
		return nil, fmt.Errorf("SSO EVE: %w", err)
	}
	tok := &Token{Access: res.AccessToken, Refresh: res.RefreshToken}
	if tok.Refresh == "" {
		tok.Refresh = form.Get("refresh_token")
	}
	// Le jeton d'accès est un JWT reçu directement du SSO en TLS : on lit ses champs sans revérifier la signature.
	parts := strings.Split(tok.Access, ".")
	if len(parts) != 3 {
		return nil, errors.New("SSO EVE: jeton d'accès illisible")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("SSO EVE: %w", err)
	}
	var claims struct {
		Sub  string `json:"sub"` // "CHARACTER:EVE:<id>"
		Name string `json:"name"`
		Exp  int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("SSO EVE: %w", err)
	}
	id, err := strconv.Atoi(claims.Sub[strings.LastIndex(claims.Sub, ":")+1:])
	if err != nil {
		return nil, fmt.Errorf("SSO EVE: personnage illisible dans %q", claims.Sub)
	}
	tok.CharacterID, tok.CharacterName, tok.Expiry = id, claims.Name, time.Unix(claims.Exp, 0)
	return tok, nil
}

// fetchLocation lit la position du personnage (cache ESI : 5 s).
func fetchLocation(tok *Token) (Reading, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/characters/%d/location/", esiBase, tok.CharacterID), nil)
	req.Header.Set("Authorization", "Bearer "+tok.Access)
	var res struct {
		SolarSystemID int   `json:"solar_system_id"`
		StationID     int64 `json:"station_id"`
		StructureID   int64 `json:"structure_id"`
	}
	if err := doJSON(req, &res); err != nil {
		return Reading{}, fmt.Errorf("ESI location: %w", err)
	}
	return Reading{At: time.Now(), SystemID: res.SolarSystemID, Docked: res.StationID != 0 || res.StructureID != 0}, nil
}

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
