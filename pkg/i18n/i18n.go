// Package i18n provides lightweight notification string localization.
// Language resolution order: user preference → country default → "en".
// No external dependencies — translations are compiled into the binary.
package i18n

import "fmt"

// Fallback language used when a key or language is not found.
const DefaultLang = "en"

// Translate returns a localized string for key in lang.
// Extra args are passed to fmt.Sprintf if the translation contains format verbs.
// Falls back to English if lang is unsupported or key is missing.
func Translate(key, lang string, args ...interface{}) string {
	if lang == "" {
		lang = DefaultLang
	}

	langMap, ok := translations[key]
	if !ok {
		// Key entirely unknown — return the key itself so nothing is silently swallowed.
		return key
	}

	tmpl, ok := langMap[lang]
	if !ok {
		// Language not found — fall back to English.
		tmpl, ok = langMap[DefaultLang]
		if !ok {
			return key
		}
	}

	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}
