package main

import (
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

//go:embed page.html
var pageHTML string

var pageTmpl = template.Must(template.New("page").Parse(pageHTML))

var tierNames = []string{"Tranquil", "Calm", "Agitated", "Fierce", "Raging", "Chaotic", "Cataclysmic"}

type pageData struct {
	Status    string
	Active    bool
	DryRun    bool
	Character string
	HasKey    bool
	ClientID  string
	Redirect  string
	S         Session
	Fits      []Fit
	FitsErr   string
	Weathers  []string
	Tiers     []string
	Hulls     []string
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.handlePage)
	mux.HandleFunc("POST /settings", a.handleSettings)
	mux.HandleFunc("GET /login", a.handleLogin)
	mux.HandleFunc("GET /callback", a.handleCallback)
	mux.HandleFunc("POST /session/start", a.handleStart)
	mux.HandleFunc("POST /session/stop", a.handleStop)
	// Un autre site ouvert dans le navigateur ne doit ni poster ici (CSRF) ni lire la page (DNS rebinding).
	protected := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "localhost:47813" && r.Host != listenAddr {
			http.Error(w, "hôte refusé", http.StatusForbidden)
			return
		}
		protected.ServeHTTP(w, r)
	})
}

func (a *App) handlePage(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	apiKey := a.cfg.APIKey
	a.mu.Unlock()
	var fits []Fit
	var fitsErr string
	if apiKey != "" {
		var err error
		if fits, err = fetchFits(apiKey); err != nil {
			fitsErr = err.Error()
		}
	}

	a.mu.Lock()
	if fitsErr == "" {
		a.fits = fits
	}
	d := pageData{
		Status: a.status, Active: a.tracker != nil, DryRun: a.dryRun,
		HasKey: a.cfg.APIKey != "", ClientID: a.cfg.ESIClientID, Redirect: pageURL + "/callback",
		S: a.cfg.Session, Fits: a.fits, FitsErr: fitsErr,
		Weathers: weatherValues, Tiers: tierNames, Hulls: hullValues,
	}
	if a.tok != nil {
		d.Character = a.tok.CharacterName
	}
	a.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, d); err != nil {
		log.Printf("page: %v", err)
	}
}

func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	// Le jeton n'est jamais réaffiché : un champ laissé vide garde la valeur enregistrée.
	if key := strings.TrimSpace(r.FormValue("api_key")); key != "" {
		a.cfg.APIKey = key
	}
	a.cfg.ESIClientID = strings.TrimSpace(r.FormValue("esi_client_id"))
	a.save("config.json", a.cfg)
	a.mu.Unlock()
	a.setStatus("Réglages enregistrés.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	clientID := a.cfg.ESIClientID
	a.oauthState, a.oauthVerifier = randomString(), randomString()
	state, verifier := a.oauthState, a.oauthVerifier
	a.mu.Unlock()
	if clientID == "" {
		a.setStatus("Renseigne d'abord l'identifiant de ton application EVE (Réglages).")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, authorizeURL(clientID, pageURL+"/callback", state, verifier), http.StatusSeeOther)
}

func (a *App) handleCallback(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	state, verifier, clientID := a.oauthState, a.oauthVerifier, a.cfg.ESIClientID
	a.oauthState = "" // usage unique
	a.mu.Unlock()
	if state == "" || r.FormValue("state") != state {
		http.Error(w, "retour SSO inattendu", http.StatusBadRequest)
		return
	}
	tok, err := exchangeCode(clientID, r.FormValue("code"), verifier)
	if err != nil {
		a.setStatus("Connexion EVE échouée : " + err.Error())
	} else {
		a.mu.Lock()
		a.tok = tok
		a.save("token.json", tok)
		a.mu.Unlock()
		a.setStatus("Personnage connecté : " + tok.CharacterName)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleStart(w http.ResponseWriter, r *http.Request) {
	tier, _ := strconv.Atoi(r.FormValue("tier"))
	s := Session{Weather: r.FormValue("weather"), Tier: tier, Hull: r.FormValue("hull")}
	if !slices.Contains(weatherValues, s.Weather) || tier < 0 || tier >= len(tierNames) || !slices.Contains(hullValues, s.Hull) {
		http.Error(w, "paramètres de session invalides", http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	for _, f := range a.fits {
		if f.ID == r.FormValue("fit") {
			s.FitID, s.FitName, s.ShipName, s.ShipTypeID = f.ID, f.Name, f.Ship.Name, f.Ship.TypeID
		}
	}
	a.mu.Unlock()
	if err := a.startSession(s); err != nil {
		a.setStatus(err.Error())
	} else {
		a.setStatus("Session en cours : " + s.Weather + " T" + strconv.Itoa(s.Tier) + ". Copie ta soute avant le premier run.")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleStop(w http.ResponseWriter, r *http.Request) {
	a.stopSession()
	a.setStatus("Session terminée.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
