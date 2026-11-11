package main

import "fmt"

// Interface languages. The first one is the default and the fallback.
var langs = []string{"en", "fr", "de", "ru", "zh"}

var langNames = map[string]string{"en": "English", "fr": "Français", "de": "Deutsch", "ru": "Русский", "zh": "中文"}

// messages holds every text shown to the user, per language. Each language lives in its own file.
// Technical errors (network, HTTP) stay in English and are passed in as arguments.
var messages = map[string]map[string]string{
	"en": messagesEN, "fr": messagesFR, "de": messagesDE, "ru": messagesRU, "zh": messagesZH,
}

// tr returns the text for key in the given language, English when the language or the text is missing.
func tr(lang, key string, args ...any) string {
	s := messages[lang][key]
	if s == "" {
		s = messagesEN[key]
	}
	if s == "" {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// catalog returns every text in the given language, for the page template.
func catalog(lang string) map[string]string {
	c := make(map[string]string, len(messagesEN))
	for key := range messagesEN {
		c[key] = tr(lang, key)
	}
	return c
}
