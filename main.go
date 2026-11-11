package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	listenAddr = "127.0.0.1:47813"
	pageURL    = "http://localhost:47813"
	pollEvery  = 5 * time.Second // égal au cache ESI : lire plus souvent ne donnerait rien de neuf
)

// Session regroupe ce que l'utilisateur choisit une fois et qui vaut pour tous les runs suivants.
type Session struct {
	Weather    string
	Tier       int
	Hull       string
	FitID      string
	FitName    string
	ShipName   string
	ShipTypeID int
}

type Config struct {
	APIKey      string
	ESIClientID string
	Session     Session
}

// Character identifie le personnage sans transporter ses jetons : un job finit dans des fichiers lisibles.
type Character struct {
	ID   int
	Name string
}

// job est un résultat du tracker figé avec la session et le personnage du moment.
type job struct {
	Outcome
	Session   Session
	Character Character
}

type App struct {
	dir     string
	dryRun  bool
	jobs    chan job
	flushMu sync.Mutex // un seul envoi de la file à la fois

	mu      sync.Mutex
	cfg     Config
	tok     *Token
	names   map[string]Item // cache nom affiché -> type EVE
	fits    []Fit
	tracker *Tracker // nil hors session
	stop    context.CancelFunc
	status  string
	// connexion SSO en cours
	oauthState, oauthVerifier string
}

func main() {
	dryRun := flag.Bool("dry-run", false, "affiche les runs dans le journal au lieu de les envoyer")
	noOpen := flag.Bool("no-open", false, "n'ouvre pas la page dans le navigateur au lancement")
	flag.Parse()

	base, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	a := &App{dir: filepath.Join(base, "abyssup"), dryRun: *dryRun, jobs: make(chan job, 16), names: map[string]Item{}}
	for _, d := range []string{"pending", "sent", "incomplete", "rejected"} {
		if err := os.MkdirAll(filepath.Join(a.dir, d), 0o700); err != nil {
			log.Fatal(err)
		}
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		// Port pris : abyssup tourne déjà, on montre simplement sa page.
		if !*noOpen {
			openBrowser(pageURL)
		}
		return
	}

	// Sans console sous Windows (-H=windowsgui) : le journal est la seule trace.
	if f, err := os.OpenFile(filepath.Join(a.dir, "abyssup.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	}
	a.load("config.json", &a.cfg)
	a.load("names.json", &a.names)
	var tok Token
	if a.load("token.json", &tok) {
		a.tok = &tok
	}
	if a.cfg.Session.Weather == "" {
		a.cfg.Session = Session{Weather: "Exotic", Hull: hullValues[0]}
	}

	go a.work()
	go watchClipboard(context.Background(), a.active, a.onClipboard)
	if !*noOpen {
		openBrowser(pageURL)
	}
	log.Printf("abyssup prêt sur %s (dry-run=%v)", pageURL, a.dryRun)
	log.Fatal(http.Serve(ln, a.routes()))
}

func (a *App) load(name string, v any) bool {
	b, err := os.ReadFile(filepath.Join(a.dir, name))
	return err == nil && json.Unmarshal(b, v) == nil
}

func (a *App) save(name string, v any) {
	// 0600 : config.json et token.json contiennent des jetons.
	// shortcut: stockés en clair dans le profil utilisateur (comme gh ou aws) ; passer à DPAPI si le PC est partagé.
	if err := os.WriteFile(filepath.Join(a.dir, name), marshal(v), 0o600); err != nil {
		log.Printf("écriture %s: %v", name, err)
	}
}

func (a *App) active() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tracker != nil
}

func (a *App) setStatus(msg string) {
	a.mu.Lock()
	a.status = time.Now().Format("15:04:05") + " — " + msg
	a.mu.Unlock()
	log.Print(msg)
}

// startSession lance (ou met à jour) la session. Appelé par la page.
func (a *App) startSession(s Session) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tok == nil {
		return errors.New("connecte d'abord ton personnage EVE (Réglages)")
	}
	if a.cfg.APIKey == "" && !a.dryRun {
		return errors.New("renseigne d'abord ton jeton Abyss Tracker (Réglages)")
	}
	a.cfg.Session = s
	a.save("config.json", a.cfg)
	if a.tracker == nil {
		ctx, cancel := context.WithCancel(context.Background())
		a.tracker, a.stop = &Tracker{}, cancel
		go a.poll(ctx)
		go a.flushPending()
	}
	return nil
}

func (a *App) stopSession() {
	a.feed(func(t *Tracker) []Outcome {
		a.stop()
		a.tracker = nil
		return t.Stop()
	})
}

// feed applique une entrée au tracker de la session en cours, fige session et personnage
// sur chaque résultat, puis passe la main au worker une fois le verrou rendu.
func (a *App) feed(f func(*Tracker) []Outcome) {
	a.mu.Lock()
	var jobs []job
	if a.tracker != nil {
		for _, o := range f(a.tracker) {
			jobs = append(jobs, job{Outcome: o, Session: a.cfg.Session,
				Character: Character{ID: a.tok.CharacterID, Name: a.tok.CharacterName}})
		}
	}
	a.mu.Unlock()
	for _, j := range jobs {
		a.jobs <- j
	}
}

func (a *App) onClipboard(text string) {
	if cargo, ok := parseCargo(text); ok {
		now := time.Now()
		a.feed(func(t *Tracker) []Outcome { return t.OnCargo(cargo, now) })
	}
	// Tout autre texte est oublié ici : ni journal, ni disque.
}

// poll lit la position tant que la session dure.
func (a *App) poll(ctx context.Context) {
	delay := pollEvery
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		r, err := a.location()
		if err != nil {
			delay = min(delay*2, time.Minute) // ESI bannit les clients qui insistent sur des erreurs
			a.setStatus(fmt.Sprintf("Lecture de position impossible, nouvel essai dans %v : %v", delay, err))
			continue
		}
		delay = pollEvery
		a.feed(func(t *Tracker) []Outcome { return t.OnReading(r) })
	}
}

func (a *App) location() (Reading, error) {
	a.mu.Lock()
	tok, clientID := a.tok, a.cfg.ESIClientID
	a.mu.Unlock()
	if !tok.valid() {
		fresh, err := refreshToken(clientID, tok.Refresh)
		if err != nil {
			return Reading{}, err
		}
		a.mu.Lock()
		a.tok = fresh
		a.save("token.json", fresh)
		a.mu.Unlock()
		tok = fresh
	}
	return fetchLocation(tok)
}

// work traite les résultats un par un, dans l'ordre, hors du verrou (réseau).
func (a *App) work() {
	for j := range a.jobs {
		switch j.Kind {
		case Notify:
			a.setStatus(j.Msg)
			notify(j.Msg, "")
		case Incomplete:
			name := j.Run.Start.Format("20060102-150405") + ".json"
			a.save(filepath.Join("incomplete", name), j)
			a.setStatus(j.Msg)
			notify(j.Msg, "")
		case Complete:
			a.complete(j)
		}
	}
}

func (a *App) complete(j job) {
	items, err := a.resolve(j.Run)
	if err != nil {
		j.Msg = "Run non envoyé : " + err.Error()
		j.Kind = Incomplete
		a.save(filepath.Join("incomplete", j.Run.Start.Format("20060102-150405")+".json"), j)
		a.setStatus(j.Msg)
		notify(j.Msg, "")
		return
	}
	t := buildTelemetry(j.Run, j.Session, j.Character, items)
	if a.dryRun {
		a.setStatus(fmt.Sprintf("[dry-run] run de %v non envoyé :\n%s", j.Run.Duration, marshal(t)))
		notify(fmt.Sprintf("[dry-run] run de %v détecté.", j.Run.Duration), "")
		return
	}
	a.save(filepath.Join("pending", t.RecordUUID+".json"), t)
	a.flushPending()
}

// resolve rend le type EVE de chaque objet gagné ou perdu, en s'appuyant sur le cache disque.
func (a *App) resolve(run *Run) (map[string]Item, error) {
	items := map[string]Item{}
	var unknown []string
	a.mu.Lock()
	for _, c := range []Cargo{run.Gained, run.Lost} {
		for name := range c {
			if it, ok := a.names[name]; ok {
				items[name] = it
			} else {
				unknown = append(unknown, name)
			}
		}
	}
	a.mu.Unlock()
	if len(unknown) == 0 {
		return items, nil
	}
	var found map[string]Item
	var err error
	// shortcut: 3 essais en ligne plutôt qu'une file de reprise ; ESI répondait quelques secondes plus tôt.
	for range 3 {
		if found, err = resolveNames(unknown); err == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	for name, it := range found {
		a.names[name], items[name] = it, it
	}
	a.save("names.json", a.names)
	a.mu.Unlock()
	var missing []string
	for _, name := range unknown {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("objets inconnus d'EVE (%s) : la copie n'était peut-être pas une soute", strings.Join(missing, ", "))
	}
	return items, nil
}

// flushPending envoie les runs en attente, du plus ancien au plus récent.
func (a *App) flushPending() {
	a.flushMu.Lock()
	defer a.flushMu.Unlock()
	a.mu.Lock()
	apiKey := a.cfg.APIKey
	a.mu.Unlock()
	if a.dryRun || apiKey == "" {
		return
	}
	entries, _ := os.ReadDir(filepath.Join(a.dir, "pending"))
	sort.Slice(entries, func(i, j int) bool {
		ii, _ := entries[i].Info()
		jj, _ := entries[j].Info()
		return ii != nil && jj != nil && ii.ModTime().Before(jj.ModTime())
	})
	for _, e := range entries {
		path := filepath.Join(a.dir, "pending", e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		runID, err := sendTelemetry(apiKey, body)
		switch {
		case err == nil:
			os.Rename(path, filepath.Join(a.dir, "sent", e.Name()))
			a.setStatus("Run envoyé : " + runURL + runID)
			notify("Run envoyé sur Abyss Tracker.", runURL+runID)
		case errors.Is(err, errRejected):
			os.Rename(path, filepath.Join(a.dir, "rejected", e.Name()))
			a.setStatus(err.Error())
			notify(err.Error(), "")
		default:
			a.setStatus(fmt.Sprintf("Envoi impossible, le run reste en attente : %v", err))
			return // panne passagère : on garde l'ordre et on réessaiera au prochain run
		}
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("ouverture du navigateur: %v", err)
	}
}
