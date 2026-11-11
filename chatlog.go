package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// The EVE client writes the Local channel log to disk, one line per system change:
//
//	[ 2026.11.11 12:56:51 ] Système EVE > Canal changé en Local : Inconnu
//	[ 2026.11.11 13:07:57 ] EVE System > Channel changed to Local : Jita
//
// It is read like any other file: nothing is asked of the client.

// LocalEvent is a system change read from the Local log.
type LocalEvent struct {
	At     time.Time // EVE time (UTC), to the second
	System string
	Abyss  bool
}

// clientLang is the wording one EVE client language uses in the Local log.
type clientLang struct {
	sender  string // pseudo-sender of system messages; written by the client, a player cannot fake it
	changed string // sentence announcing the new system, before the colon
	unknown string // name shown instead of a system name inside the Abyss
}

// shortcut: only the French line and "Inconnu" were checked against a real log. The other senders and
// sentences come from public EVE log parsers; the other "unknown" words are assumed.
// A wrong word only means runs are not detected in that language: confirm each with a real log.
var clientLangs = []clientLang{
	{"EVE System", "Channel changed to Local", "Unknown"},
	{"Système EVE", "Canal changé en Local", "Inconnu"},
	{"EVE-System", "Chatkanal geändert zu Lokal", "Unbekannt"},
	{"Система EVE", "Канал изменен на Локальный", "Неизвестно"},
	{"EVE系统", "频道更换为本地", "未知"},
}

var (
	localLine = regexp.MustCompile(`^\x{feff}?\[ (\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}) \] (.+?) > (.*)$`)
	listener  = regexp.MustCompile(`(?m)^\s*Listener:\s+(.+?)\s*$`)
	// Local_YYYYMMDD_HHMMSS_<character ID>.txt
	localFile = regexp.MustCompile(`^Local_\d{8}_\d{6}_(\d+)\.txt$`)
)

// squash drops every kind of space and lowercases, so that sentences compare the same whether the
// client writes a normal space, a non-breaking one, or none at all around the colon.
func squash(s string) string {
	return strings.ToLower(strings.Join(strings.FieldsFunc(s, unicode.IsSpace), ""))
}

func parseLocalLine(line string) (LocalEvent, bool) {
	m := localLine.FindStringSubmatch(line)
	if m == nil {
		return LocalEvent{}, false
	}
	at, err := time.ParseInLocation("2006.01.02 15:04:05", m[1], time.UTC)
	if err != nil {
		return LocalEvent{}, false
	}
	// The system name follows the last colon; the Chinese client writes a full-width one.
	msg := m[3]
	cut := strings.LastIndexAny(msg, ":：")
	if cut < 0 {
		return LocalEvent{}, false
	}
	_, size := utf8.DecodeRuneInString(msg[cut:])
	// Non-English clients mark names kept in English with a trailing star.
	name := strings.TrimSuffix(strings.TrimFunc(msg[cut+size:], unicode.IsSpace), "*")
	for _, l := range clientLangs {
		if m[2] == l.sender && squash(msg[:cut]) == squash(l.changed) && name != "" {
			e := LocalEvent{At: at, System: name}
			for _, u := range clientLangs {
				e.Abyss = e.Abyss || strings.EqualFold(name, u.unknown)
			}
			return e, true
		}
	}
	return LocalEvent{}, false
}

// defaultChatlogDir returns the folder where EVE stores its chat logs.
func defaultChatlogDir() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Documents", "EVE", "logs", "Chatlogs"),
		filepath.Join(home, "OneDrive", "Documents", "EVE", "logs", "Chatlogs"), // Documents redirected by OneDrive
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return candidates[0]
}

// errNoLocalLog: the folder holds no Local log yet, EVE has probably not been started.
var errNoLocalLog = errors.New("no Local log found")

// localTail follows the most recent Local log of a folder.
type localTail struct {
	dir    string
	path   string
	offset int64
	rest   []byte // end of file not yet terminated by a newline
	fresh  bool   // opened file whose header has not been returned yet
}

// read returns the system changes that appeared since the last call.
// fresh is true when a new file has just been opened (EVE login):
// the events are then history, and char identifies the character.
func (t *localTail) read() (events []LocalEvent, fresh bool, char Character, err error) {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return nil, false, char, err
	}
	var names []string
	for _, e := range entries {
		if localFile.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return nil, false, char, errNoLocalLog
	}
	// shortcut: the name starts with the date, so the last one alphabetically is the most recent.
	// One character at a time; with several clients open, the file will have to be chosen per character.
	sort.Strings(names)
	newest := filepath.Join(t.dir, names[len(names)-1])
	if newest != t.path {
		t.path, t.offset, t.rest, t.fresh = newest, 0, nil, true
	}

	f, err := os.Open(t.path)
	if err != nil {
		return nil, false, char, err
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	for {
		n, rerr := f.ReadAt(buf, t.offset)
		t.rest = append(t.rest, buf[:n]...)
		t.offset += int64(n)
		if rerr != nil {
			break // end of file reached
		}
	}

	// The file is UTF-16LE: only complete lines are decoded.
	end := bytes.LastIndex(t.rest, []byte{'\n', 0})
	for end >= 0 && end%2 != 0 {
		end = bytes.LastIndex(t.rest[:end], []byte{'\n', 0})
	}
	if end < 0 {
		return nil, false, char, nil
	}
	fresh, t.fresh = t.fresh, false
	text := decodeUTF16LE(t.rest[:end])
	t.rest = append([]byte(nil), t.rest[end+2:]...)

	if fresh {
		char.ID, _ = strconv.Atoi(localFile.FindStringSubmatch(filepath.Base(t.path))[1])
		if m := listener.FindStringSubmatch(text); m != nil {
			char.Name = m[1]
		}
	}
	for line := range strings.SplitSeq(text, "\n") {
		if e, ok := parseLocalLine(strings.TrimRight(line, "\r")); ok {
			events = append(events, e)
		}
	}
	return events, fresh, char, nil
}

func decodeUTF16LE(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	return string(utf16.Decode(u))
}
