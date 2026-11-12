//go:build !windows

package main

import "context"

// Outside Windows, abysstracker-autosubmit is only used for development and tests: no clipboard, no notification.

func watchClipboard(ctx context.Context, wanted func() bool, onText func(string)) {}

func notify(msg, link string) {}
