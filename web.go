package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

//go:embed page.html settings.html
var pages embed.FS

var pageTmpl = template.Must(template.New("").Funcs(template.FuncMap{"isk": isk}).ParseFS(pages, "*.html"))

var tierNames = []string{"Tranquil", "Calm", "Agitated", "Fierce", "Raging", "Chaotic", "Cataclysmic"}

// step is one of the three things a run needs, as the page draws it.
type step struct {
	Title, Note string // keys in messages
	State       string // done, now, todo or miss
}

// steps tells what the player is expected to do next: the headline, the copy hint that goes with it
// (empty when there is nothing to copy now) and the state of each step. All texts are keys in messages.
func steps(stage trackerState, base bool) (head, hint string, s [3]step) {
	s = [3]step{{"stepBase", "baseDone", "done"}, {"stepRun", "runWait", "todo"}, {"stepLoot", "lootTodo", "todo"}}
	switch {
	case stage == awaitingLoot:
		s[1], s[2] = step{"stepRun", "runDone", "done"}, step{"stepLoot", "waitCopy", "now"}
		return "headAwait", "copyAfter", s
	case stage == inAbyss:
		s[1] = step{"stepRun", "runNow", "now"}
		if !base {
			s[0] = step{"stepBase", "baseMissing", "miss"}
		}
		return "headInAbyss", "", s
	case base:
		s[1].State = "now"
		return "headReady", "", s
	}
	s[0] = step{"stepBase", "waitCopy", "now"}
	return "headNeedBase", "copyBefore", s
}

type pageData struct {
	L          map[string]string // interface texts in the current language
	Lang       string
	Langs      []string
	LangNames  map[string]string
	Version    string
	Update     release
	Status     string
	Active     bool
	DryRun     bool
	Character  string
	Head, Hint string // keys in messages; empty outside a session
	Steps      [3]step
	History    []histRun
	Lost       int     // runs of the history the ship did not survive
	Loot       float64 // total loot of the history, in ISK
	HasKey     bool
	ChatlogDir string
	Dir        string // where the program keeps its files
	Saved      bool   // the settings were just saved
	S          Session
	TierName   string
	Fits       []Fit
	FitsErr    string
	Weathers   []string
	Tiers      []string
	Hulls      []string
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.handlePage)
	mux.HandleFunc("GET /live", a.handleLive)
	mux.HandleFunc("GET /settings", a.handleSettingsPage)
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

// view gathers what the pages show. Lock held by the caller.
func (a *App) view() pageData {
	d := pageData{
		L: catalog(a.cfg.Lang), Lang: a.cfg.Lang, Langs: langs, LangNames: langNames,
		Version: version, Update: a.update, History: slices.Clone(a.history),
		Status: a.status, Active: a.tracker != nil, DryRun: a.dryRun,
		HasKey: a.cfg.APIKey != "", ChatlogDir: a.chatlogDir(), Dir: a.dir,
		S: a.cfg.Session, Fits: a.fits,
		Weathers: weatherValues, Tiers: tierNames, Hulls: hullValues,
	}
	if t := d.S.Tier; t >= 0 && t < len(tierNames) {
		d.TierName = tierNames[t]
	}
	if a.tracker != nil {
		d.Character = a.char.Name
		d.Head, d.Hint, d.Steps = steps(a.tracker.Progress())
	}
	for _, h := range d.History {
		if !h.Survived {
			d.Lost++
		}
		d.Loot += h.Loot
	}
	return d
}

func render(w http.ResponseWriter, name string, d pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.ExecuteTemplate(w, name, d); err != nil {
		log.Printf("%s: %v", name, err)
	}
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
	d := a.view()
	a.mu.Unlock()
	d.FitsErr = fitsErr
	render(w, "page.html", d)
}

// handleLive renders the session state and the history alone: the page asks for them again every few seconds.
func (a *App) handleLive(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	d := a.view()
	a.mu.Unlock()
	render(w, "live", d)
}

func (a *App) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	d := a.view()
	a.mu.Unlock()
	d.Saved = r.URL.Query().Has("saved")
	render(w, "settings.html", d)
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
	http.Redirect(w, r, "/settings?saved", http.StatusSeeOther)
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
