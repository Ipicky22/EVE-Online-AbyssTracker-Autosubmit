package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"
)

func TestParseLocalLine(t *testing.T) {
	at := time.Date(2026, 11, 11, 12, 56, 51, 0, time.UTC)
	cases := []struct {
		line string
		want LocalEvent
		ok   bool
	}{
		// Real lines from a French client: leading BOM, non-breaking space before the colon, star after the name.
		{"\ufeff[ 2026.11.11 12:56:51 ] Système EVE > Canal changé en Local\u00a0: Alillere*", LocalEvent{At: at, System: "Alillere"}, true},
		{"\ufeff[ 2026.11.11 12:56:51 ] Système EVE > Canal changé en Local\u00a0: Inconnu", LocalEvent{At: at, System: "Inconnu", Abyss: true}, true},
		// English client.
		{"\ufeff[ 2026.11.11 12:56:51 ] EVE System > Channel changed to Local : Jita", LocalEvent{At: at, System: "Jita"}, true},
		{"\ufeff[ 2026.11.11 12:56:51 ] EVE System > Channel changed to Local : Unknown", LocalEvent{At: at, System: "Unknown", Abyss: true}, true},
		// A player typing the same sentence is not the system.
		{"\ufeff[ 2026.11.11 12:56:51 ] Some Pilot > Channel changed to Local : Unknown", LocalEvent{}, false},
		{"          Channel Name:    Local", LocalEvent{}, false},
		{"", LocalEvent{}, false},
	}
	for _, c := range cases {
		got, ok := parseLocalLine(c.line)
		if ok != c.ok || got != c.want {
			t.Errorf("%q: got %+v ok=%v, want %+v ok=%v", c.line, got, ok, c.want, c.ok)
		}
	}
}

// appendUTF16 writes text the way the EVE client does: UTF-16LE, CRLF line endings.
func appendUTF16(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var b []byte
	for _, u := range utf16.Encode([]rune(text)) {
		b = append(b, byte(u), byte(u>>8))
	}
	if _, err := f.Write(b); err != nil {
		t.Fatal(err)
	}
}

const header = "\ufeff\r\n\r\n        ---------------------------------------------------------------\r\n\r\n" +
	"          Channel ID:      local\r\n          Channel Name:    Local\r\n" +
	"          Listener:        Test Pilot\r\n          Session started: 2026.11.11 11:28:29\r\n" +
	"        ---------------------------------------------------------------\r\n\r\n"

func TestLocalTail(t *testing.T) {
	dir := t.TempDir()
	tail := &localTail{dir: dir}
	if _, _, _, err := tail.read(); err == nil {
		t.Fatal("a folder without a Local log must return an error")
	}

	old := filepath.Join(dir, "Local_20261110_090000_90000001.txt")
	appendUTF16(t, old, header+"\ufeff[ 2026.11.10 09:00:05 ] EVE System > Channel changed to Local : Jita\r\n")
	path := filepath.Join(dir, "Local_20261111_112829_90000001.txt")
	appendUTF16(t, path, header+"\ufeff[ 2026.11.11 11:28:32 ] Système EVE > Canal changé en Local\u00a0: Alillere*\r\n")
	// Another channel in the same folder must not be mistaken for Local.
	appendUTF16(t, filepath.Join(dir, "Corp_20261111_120000_90000001.txt"), header)

	events, fresh, char, err := tail.read()
	if err != nil || !fresh || len(events) != 1 || events[0].System != "Alillere" {
		t.Fatalf("first read: events=%+v fresh=%v err=%v", events, fresh, err)
	}
	if char != (Character{ID: 90000001, Name: "Test Pilot"}) {
		t.Errorf("character = %+v", char)
	}

	// Nothing new, then a line written in two parts: it only counts once complete.
	if events, fresh, _, _ = tail.read(); len(events) != 0 || fresh {
		t.Fatalf("read with nothing new: events=%+v fresh=%v", events, fresh)
	}
	appendUTF16(t, path, "\ufeff[ 2026.11.11 12:56:51 ] Système EVE > Canal changé en Lo")
	if events, _, _, _ = tail.read(); len(events) != 0 {
		t.Fatalf("incomplete line was read: %+v", events)
	}
	appendUTF16(t, path, "cal\u00a0: Inconnu\r\n\ufeff[ 2026.11.11 13:07:57 ] Système EVE > Canal changé en Local\u00a0: Alillere*\r\n")
	events, _, _, _ = tail.read()
	if len(events) != 2 || !events[0].Abyss || events[1].Abyss || events[1].At.Sub(events[0].At) != 666*time.Second {
		t.Fatalf("run read: %+v", events)
	}

	// New EVE login: a more recent file takes over.
	appendUTF16(t, filepath.Join(dir, "Local_20261111_200000_90000002.txt"), header+"\ufeff[ 2026.11.11 20:00:03 ] EVE System > Channel changed to Local : Amarr\r\n")
	events, fresh, char, _ = tail.read()
	if !fresh || len(events) != 1 || events[0].System != "Amarr" || char.ID != 90000002 {
		t.Fatalf("file switch: events=%+v fresh=%v char=%+v", events, fresh, char)
	}
}
