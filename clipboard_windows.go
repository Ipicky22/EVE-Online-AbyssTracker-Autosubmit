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

// watchClipboard calls onText for each newly copied text, as long as wanted() is true.
// Passive read of the Windows clipboard: nothing is sent to the EVE client.
func watchClipboard(ctx context.Context, wanted func() bool, onText func(string)) {
	last, _, _ := procSequence.Call() // whatever was copied before startup is none of our business
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
			last = seq // outside a session: the content is not read
			continue
		}
		text, opened := readClipboard()
		if !opened {
			continue // clipboard held by another application: retry on the next tick
		}
		last = seq
		if text != "" {
			onText(text)
		}
	}
}

func readClipboard() (text string, opened bool) {
	// OpenClipboard and CloseClipboard must be called from the same thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if r, _, _ := procOpen.Call(0); r == 0 {
		return "", false
	}
	defer procClose.Call()
	h, _, _ := procGetData.Call(cfUnicodeText)
	if h == 0 {
		return "", true // no text (image, file...)
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
	// The block belongs to the clipboard: copy it before releasing it, without reading past its size.
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
