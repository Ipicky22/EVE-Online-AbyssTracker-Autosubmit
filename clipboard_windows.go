//go:build windows

package main

import (
	"context"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSequence = user32.NewProc("GetClipboardSequenceNumber")
	procOpen     = user32.NewProc("OpenClipboard")
	procClose    = user32.NewProc("CloseClipboard")
	procGetData  = user32.NewProc("GetClipboardData")
	procLock     = kernel32.NewProc("GlobalLock")
	procUnlock   = kernel32.NewProc("GlobalUnlock")
	procSize     = kernel32.NewProc("GlobalSize")
	procMove     = kernel32.NewProc("RtlMoveMemory")
)

const cfUnicodeText = 13

// watchClipboard appelle onText à chaque nouveau texte copié, tant que wanted() est vrai.
// Lecture passive du presse-papiers Windows : rien n'est envoyé au client EVE.
func watchClipboard(ctx context.Context, wanted func() bool, onText func(string)) {
	last, _, _ := procSequence.Call() // ce qui a été copié avant le lancement ne nous regarde pas
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		seq, _, _ := procSequence.Call()
		if seq == last {
			continue
		}
		if !wanted() {
			last = seq // hors session : on ne lit pas le contenu
			continue
		}
		text, opened := readClipboard()
		if !opened {
			continue // presse-papiers tenu par une autre application : nouvel essai au prochain tick
		}
		last = seq
		if text != "" {
			onText(text)
		}
	}
}

func readClipboard() (text string, opened bool) {
	// OpenClipboard et CloseClipboard doivent venir du même thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if r, _, _ := procOpen.Call(0); r == 0 {
		return "", false
	}
	defer procClose.Call()
	h, _, _ := procGetData.Call(cfUnicodeText)
	if h == 0 {
		return "", true // pas de texte (image, fichier...)
	}
	p, _, _ := procLock.Call(h)
	if p == 0 {
		return "", true
	}
	defer procUnlock.Call(h)
	size, _, _ := procSize.Call(h)
	if size < 2 {
		return "", true
	}
	// Le bloc appartient au presse-papiers : on le copie avant de le rendre, sans dépasser sa taille.
	u16 := make([]uint16, size/2)
	procMove.Call(uintptr(unsafe.Pointer(&u16[0])), p, uintptr(len(u16)*2))
	for i, c := range u16 {
		if c == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16)), true
}
