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
	L          map[string]string // interface texts in the current language
	Lang       string
	Langs      []string
	LangNames  map[string]string
	Status     string
	Active     bool
	DryRun     bool
	Character  string
	HasKey     bool
	ChatlogDir string
	S          Session
	Fits       []Fit
	FitsErr    string
	Weathers   []string
	Tiers      []string
	Hulls      []string
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.handlePage)
	mux.HandleFunc("POST /settings", a.handleSettings)
	mux.HandleFunc("POST /session/start", a.handleStart)
	mux.HandleFunc("POST /session/stop", a.handleStop)
	// Another site open in the browser must neither post here (CSRF) nor read the page (DNS rebinding).
	protected := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "localhost:47813" && r.Host != listenAddr {
			http.Error(w, "forbidden host", http.StatusForbidden)
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
		L: catalog(a.cfg.Lang), Lang: a.cfg.Lang, Langs: langs, LangNames: langNames,
		Status: a.status, Active: a.tracker != nil, DryRun: a.dryRun,
		HasKey: a.cfg.APIKey != "", ChatlogDir: a.chatlogDir(),
		S: a.cfg.Session, Fits: a.fits, FitsErr: fitsErr,
		Weathers: weatherValues, Tiers: tierNames, Hulls: hullValues,
	}
	if a.tracker != nil {
		d.Character = a.char.Name
	}
	a.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, d); err != nil {
		log.Printf("page: %v", err)
	}
}

func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	// The token is never shown again: an empty field keeps the saved value.
	if key := strings.TrimSpace(r.FormValue("api_key")); key != "" {
		a.cfg.APIKey = key
	}
	// The default folder is not frozen in the config: it is detected again at each start.
	if dir := strings.TrimSpace(r.FormValue("chatlog_dir")); dir == defaultChatlogDir() {
		a.cfg.ChatlogDir = ""
	} else {
		a.cfg.ChatlogDir = dir
	}
	if lang := r.FormValue("lang"); slices.Contains(langs, lang) {
		a.cfg.Lang = lang
	}
	a.save("config.json", a.cfg)
	a.mu.Unlock()
	a.say("settingsSaved")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleStart(w http.ResponseWriter, r *http.Request) {
	tier, _ := strconv.Atoi(r.FormValue("tier"))
	s := Session{Weather: r.FormValue("weather"), Tier: tier, Hull: r.FormValue("hull"), AllCans: r.FormValue("all_cans") != ""}
	if !slices.Contains(weatherValues, s.Weather) || tier < 0 || tier >= len(tierNames) || !slices.Contains(hullValues, s.Hull) {
		http.Error(w, "invalid session settings", http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	for _, f := range a.fits {
		if f.ID == r.FormValue("fit") {
			s.FitID, s.FitName, s.ShipName, s.ShipTypeID = f.ID, f.Name, f.Ship.Name, f.Ship.TypeID
		}
	}
	a.mu.Unlock()
	if errKey, errArgs := a.startSession(s); errKey != "" {
		a.say(errKey, errArgs...)
	} else {
		a.say("sessionStarted", s.Weather, s.Tier)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleStop(w http.ResponseWriter, r *http.Request) {
	a.stopSession()
	a.say("sessionEnded")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
