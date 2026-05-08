// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"strings"
	"unicode"
)

// commonInitialisms is the set of identifiers Go style preserves in caps
// when title-casing. Keep in sync with golang.org/x/lint/golint.
var commonInitialisms = map[string]bool{
	"ACL": true, "API": true, "ASCII": true, "CPU": true, "CSS": true, "CSV": true,
	"DNS": true, "EOF": true, "GUID": true, "HTML": true, "HTTP": true, "HTTPS": true,
	"ID": true, "IP": true, "JSON": true, "LHS": true, "QPS": true, "RAM": true,
	"RHS": true, "RPC": true, "SLA": true, "SMTP": true, "SQL": true, "SSH": true,
	"TCP": true, "TLS": true, "TTL": true, "UDP": true, "UI": true, "UID": true,
	"UUID": true, "URI": true, "URL": true, "UTF8": true, "VM": true, "XML": true,
	"XMPP": true, "XSRF": true, "XSS": true, "YAML": true,
}

// Title returns s with the first rune title-cased, preserving Go-style
// initialisms (ID, URL, HTTP, ...).
//
//	Title("id")    → "ID"
//	Title("user")  → "User"
//	Title("uRL")   → "URL"
func Title(s string) string {
	if s == "" {
		return s
	}
	upper := strings.ToUpper(s)
	if commonInitialisms[upper] {
		return upper
	}
	r, n := firstRune(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// CamelCase converts snake_case or kebab-case to UpperCamelCase.
//
//	CamelCase("user_id")    → "UserID"
//	CamelCase("http-method") → "HTTPMethod"
func CamelCase(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	upperNext := true
	for _, r := range s {
		switch {
		case r == '_' || r == '-' || r == ' ':
			upperNext = true
		case upperNext:
			b.WriteRune(unicode.ToUpper(r))
			upperNext = false
		default:
			b.WriteRune(r)
		}
	}
	// Promote any title-cased segments that match a known initialism.
	// We always invoke the promoter — the cost is a single linear walk
	// and the previous "containsInitialism" heuristic was buggy
	// (Title("http") = "HTTP" so it never matched a Title-cased form).
	return promoteInitialisms(b.String())
}

// LowerCamelCase converts to lowerCamelCase. The first rune is lowered;
// subsequent words follow camelCase rules with initialism promotion.
//
//	LowerCamelCase("user_id")     → "userID"
//	LowerCamelCase("http_method") → "httpMethod"
func LowerCamelCase(s string) string {
	c := CamelCase(s)
	if c == "" {
		return c
	}
	r, n := firstRune(c)
	return string(unicode.ToLower(r)) + c[n:]
}

// SnakeCase converts CamelCase or kebab-case to snake_case.
//
//	SnakeCase("UserID")     → "user_id"
//	SnakeCase("httpMethod") → "http_method"
func SnakeCase(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i, r := range s {
		switch {
		case r == '-' || r == ' ':
			b.WriteByte('_')
		case unicode.IsUpper(r):
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// firstRune decomposes s into its first rune and the byte length consumed.
// Returns the zero rune and 0 for empty strings.
func firstRune(s string) (rune, int) {
	for i, r := range s {
		_ = i
		return r, len(string(r))
	}
	return 0, 0
}

// promoteInitialisms walks s and replaces title-cased initialisms with
// their all-caps form: "Url" → "URL", "Http" → "HTTP".
func promoteInitialisms(s string) string {
	// Walk left-to-right finding word boundaries (uppercase rune).
	type word struct{ start, end int }
	var words []word
	for i, r := range s {
		if unicode.IsUpper(r) || i == 0 {
			if len(words) > 0 {
				words[len(words)-1].end = i
			}
			words = append(words, word{start: i})
		}
	}
	if len(words) == 0 {
		return s
	}
	words[len(words)-1].end = len(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, w := range words {
		piece := s[w.start:w.end]
		upper := strings.ToUpper(piece)
		if commonInitialisms[upper] {
			b.WriteString(upper)
		} else {
			b.WriteString(piece)
		}
	}
	return b.String()
}
