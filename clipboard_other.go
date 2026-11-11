//go:build !windows

package main

import "context"

// Hors Windows, abyssup ne sert qu'à développer et tester : pas de presse-papiers ni de notification.

func watchClipboard(ctx context.Context, wanted func() bool, onText func(string)) {}

func notify(msg, link string) {}
