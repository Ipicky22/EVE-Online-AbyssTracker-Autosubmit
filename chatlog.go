package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Le client EVE écrit sur disque le journal du canal Local, une ligne à chaque changement de système :
//
//	[ 2026.11.11 12:56:51 ] Système EVE > Canal changé en Local : Inconnu
//	[ 2026.11.11 13:07:57 ] EVE System > Channel changed to Local : Jita
//
// On le lit comme n'importe quel fichier : rien n'est demandé au client.

// LocalEvent est un changement de système lu dans le journal Local.
type LocalEvent struct {
	At     time.Time // heure EVE (UTC), à la seconde
	System string
	Abyss  bool
}

var (
	// Le client français met une espace insécable avant les deux-points et une étoile après le nom.
	localLine = regexp.MustCompile(`^\x{feff}?\[ (\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}) \] (?:EVE System|Système EVE) > .*Local[\s\x{a0}]*:[\s\x{a0}]*(.+?)\*?\s*$`)
	listener  = regexp.MustCompile(`(?m)^\s*Listener:\s+(.+?)\s*$`)
	// Local_AAAAMMJJ_HHMMSS_<identifiant du personnage>.txt
	localFile = regexp.MustCompile(`^Local_\d{8}_\d{6}_(\d+)\.txt$`)
)

// L'Abysse n'a pas de nom de système dans le Local.
// shortcut: « Inconnu » vérifié sur un journal français ; « Unknown » supposé pour le client anglais,
// à confirmer avec un vrai journal anglais.
var abyssNames = []string{"Inconnu", "Unknown"}

func parseLocalLine(line string) (LocalEvent, bool) {
	m := localLine.FindStringSubmatch(line)
	if m == nil {
		return LocalEvent{}, false
	}
	at, err := time.ParseInLocation("2006.01.02 15:04:05", m[1], time.UTC)
	if err != nil {
		return LocalEvent{}, false
	}
	e := LocalEvent{At: at, System: m[2]}
	for _, n := range abyssNames {
		e.Abyss = e.Abyss || strings.EqualFold(e.System, n)
	}
	return e, true
}

// defaultChatlogDir rend le dossier où EVE range ses journaux de discussion.
func defaultChatlogDir() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Documents", "EVE", "logs", "Chatlogs"),
		filepath.Join(home, "OneDrive", "Documents", "EVE", "logs", "Chatlogs"), // Documents redirigé par OneDrive
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return candidates[0]
}

// localTail suit le journal Local le plus récent d'un dossier.
type localTail struct {
	dir    string
	path   string
	offset int64
	rest   []byte // fin de fichier pas encore terminée par un retour à la ligne
	fresh  bool   // fichier ouvert dont l'en-tête n'a pas encore été rendu
}

// read rend les changements de système apparus depuis le dernier appel.
// fresh vaut vrai quand un nouveau fichier vient d'être ouvert (connexion à EVE) :
// les événements sont alors de l'historique, et char identifie le personnage.
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
		return nil, false, char, fmt.Errorf("aucun journal Local dans %s : lance EVE", t.dir)
	}
	// shortcut: le nom commence par la date, donc le dernier par ordre alphabétique est le plus récent.
	// Un seul personnage à la fois ; avec plusieurs clients ouverts il faudra choisir par personnage.
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
			break // fin de fichier atteinte
		}
	}

	// Le fichier est en UTF-16LE : on ne décode que les lignes complètes.
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
