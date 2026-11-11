package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	listenAddr = "127.0.0.1:47813"
	pageURL    = "http://localhost:47813"
	tailEvery  = time.Second // how often the Local log is read again
)

// Session holds what the user picks once and that applies to every following run.
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
	APIKey     string
	ChatlogDir string // empty = EVE's default folder
	Lang       string // interface language, see langs
	Session    Session
}

// Character identifies the character, read from the Local log.
type Character struct {
	ID   int
	Name string
}

// job is a tracker outcome frozen with the session and character at that moment.
type job struct {
	Outcome
	Session   Session
	Character Character
}

type App struct {
	dir     string
	dryRun  bool
	jobs    chan job
	flushMu sync.Mutex // only one queue flush at a time

	mu      sync.Mutex
	cfg     Config
	char    Character       // character of the current Local log
	names   map[string]Item // cache: displayed name -> EVE type
	fits    []Fit
	tracker *Tracker // nil outside a session
	stop    context.CancelFunc
	status  string
}

func main() {
	dryRun := flag.Bool("dry-run", false, "write runs to the log instead of sending them")
	noOpen := flag.Bool("no-open", false, "do not open the page in the browser at startup")
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
		// Port taken: abyssup is already running, just show its page.
		if !*noOpen {
			openBrowser(pageURL)
		}
		return
	}

	// No console on Windows (-H=windowsgui): the log file is the only trace.
	if f, err := os.OpenFile(filepath.Join(a.dir, "abyssup.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	}
	a.load("config.json", &a.cfg)
	a.load("names.json", &a.names)
	if !slices.Contains(langs, a.cfg.Lang) {
		a.cfg.Lang = langs[0]
	}
	if a.cfg.Session.Weather == "" {
		a.cfg.Session = Session{Weather: "Exotic", Hull: hullValues[0]}
	}

	go a.work()
	go watchClipboard(context.Background(), a.active, a.onClipboard)
	if !*noOpen {
		openBrowser(pageURL)
	}
	log.Printf("abyssup ready on %s (dry-run=%v)", pageURL, a.dryRun)
	log.Fatal(http.Serve(ln, a.routes()))
}

func (a *App) load(name string, v any) bool {
	b, err := os.ReadFile(filepath.Join(a.dir, name))
	return err == nil && json.Unmarshal(b, v) == nil
}

func (a *App) save(name string, v any) {
	// 0600: config.json holds the Abyss Tracker token.
	// shortcut: stored in plain text in the user profile (like gh or aws); switch to DPAPI if the PC is shared.
	if err := os.WriteFile(filepath.Join(a.dir, name), marshal(v), 0o600); err != nil {
		log.Printf("writing %s: %v", name, err)
	}
}

func (a *App) active() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tracker != nil
}

// say translates a message into the interface language, shows it as the page status and logs it.
// It returns the text so the caller can also send it as a notification. Must be called without the lock.
func (a *App) say(key string, args ...any) string {
	a.mu.Lock()
	msg := tr(a.cfg.Lang, key, args...)
	a.status = time.Now().Format("15:04:05") + " — " + msg
	a.mu.Unlock()
	log.Print(msg)
	return msg
}

// startSession starts (or updates) the session. Called by the page.
// On refusal it returns the key and arguments of the message to show.
func (a *App) startSession(s Session) (errKey string, errArgs []any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	dir := a.chatlogDir()
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "errNoLogDir", []any{dir}
	}
	if a.cfg.APIKey == "" && !a.dryRun {
		return "errNoToken", nil
	}
	a.cfg.Session = s
	a.save("config.json", a.cfg)
	if a.tracker == nil {
		ctx, cancel := context.WithCancel(context.Background())
		a.tracker, a.stop = &Tracker{}, cancel
		go a.tail(ctx, dir)
		go a.flushPending()
	}
	return "", nil
}

func (a *App) stopSession() {
	a.feed(func(t *Tracker) []Outcome {
		a.stop()
		a.tracker = nil
		return t.Stop()
	})
}

// feed applies an input to the current session's tracker, freezes session and character
// on each outcome, then hands over to the worker once the lock is released.
func (a *App) feed(f func(*Tracker) []Outcome) {
	a.mu.Lock()
	var jobs []job
	if a.tracker != nil {
		for _, o := range f(a.tracker) {
			jobs = append(jobs, job{Outcome: o, Session: a.cfg.Session,
				Character: a.char})
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
	// Any other text is dropped here: neither logged nor written to disk.
}

// chatlogDir returns EVE's chat log folder. Lock held by the caller.
func (a *App) chatlogDir() string {
	if a.cfg.ChatlogDir != "" {
		return a.cfg.ChatlogDir
	}
	return defaultChatlogDir()
}

// tail follows the Local log for as long as the session lasts.
func (a *App) tail(ctx context.Context, dir string) {
	t := &localTail{dir: dir}
	lastErr := ""
	for {
		events, fresh, char, err := t.read()
		if err != nil {
			if err.Error() != lastErr { // the same error comes back every second: report it only once
				lastErr = err.Error()
				if errors.Is(err, errNoLocalLog) {
					a.say("noLocalLog", dir)
				} else {
					a.say("logUnreadable", err)
				}
			}
		} else {
			lastErr = ""
			a.feed(func(tr *Tracker) []Outcome {
				if !fresh {
					var out []Outcome
					for _, e := range events {
						out = append(out, tr.OnLocal(e, time.Now())...)
					}
					return out
				}
				// New file: its lines are history, only the last location matters.
				a.char = char
				if len(events) > 0 {
					tr.SetLocation(events[len(events)-1])
				}
				return []Outcome{{Kind: Notify, Msg: "logFollowed", Args: []any{char.Name}}}
			})
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(tailEvery):
		}
	}
}

// work handles outcomes one by one, in order, outside the lock (network).
func (a *App) work() {
	for j := range a.jobs {
		switch j.Kind {
		case Notify:
			notify(a.say(j.Msg, j.Args...), "")
		case Incomplete:
			a.incomplete(j)
		case Complete:
			a.complete(j)
		}
	}
}

// incomplete tells the user why a run is not sent and keeps it on disk, with the message as shown.
func (a *App) incomplete(j job) {
	msg := a.say(j.Msg, j.Args...)
	notify(msg, "")
	j.Kind, j.Msg, j.Args = Incomplete, msg, nil
	a.save(filepath.Join("incomplete", j.Run.Start.Format("20060102-150405")+".json"), j)
}

// lang returns the interface language. Must be called without the lock.
func (a *App) lang() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Lang
}

func (a *App) complete(j job) {
	items, missing, err := a.resolve(j.Run)
	switch {
	case err != nil:
		j.Msg, j.Args = "notSentError", []any{err}
	case len(missing) > 0:
		j.Msg, j.Args = "notSentUnknownItems", []any{strings.Join(missing, ", ")}
	}
	if err != nil || len(missing) > 0 {
		a.incomplete(j)
		return
	}
	t := buildTelemetry(j.Run, j.Session, j.Character, items)
	if a.dryRun {
		a.say("dryRunStatus", j.Run.Duration, marshal(t)) // the status keeps the full payload
		notify(tr(a.lang(), "dryRunNotify", j.Run.Duration), "")
		return
	}
	a.save(filepath.Join("pending", t.RecordUUID+".json"), t)
	a.flushPending()
}

// resolve returns the EVE type of each gained or lost item, relying on the disk cache.
// missing lists the names EVE does not know.
func (a *App) resolve(run *Run) (items map[string]Item, missing []string, err error) {
	items = map[string]Item{}
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
		return items, nil, nil
	}
	var found map[string]Item
	// shortcut: 3 inline attempts rather than a retry queue.
	for range 3 {
		if found, err = resolveNames(unknown); err == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, nil, err
	}
	a.mu.Lock()
	for name, it := range found {
		a.names[name], items[name] = it, it
	}
	a.save("names.json", a.names)
	a.mu.Unlock()
	for _, name := range unknown {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return items, missing, nil
}

// flushPending sends the pending runs, oldest first.
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
			a.say("sentStatus", runURL+runID) // the status keeps the link
			notify(tr(a.lang(), "sentNotify"), runURL+runID)
		case errors.Is(err, errRejected):
			os.Rename(path, filepath.Join(a.dir, "rejected", e.Name()))
			notify(a.say("rejected", strings.TrimPrefix(err.Error(), errRejected.Error()+": ")), "")
		default:
			a.say("sendFailed", err)
			return // transient failure: keep the order and retry at the next run
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
		log.Printf("opening browser: %v", err)
	}
}
