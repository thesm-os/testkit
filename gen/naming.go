// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"strings"
	"unicode"
)

// commonInitialisms is the set of Go-conventional initialisms that
// should be fully uppercased in exported names. Matches the set used
// by golint/staticcheck.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"UUID":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSS":   true,
}

// Title uppercases the first letter of s, with awareness of Go
// initialisms. "id" → "ID", "url" → "URL", "name" → "Name".
func Title(s string) string {
	if s == "" {
		return s
	}
	upper := strings.ToUpper(s)
	if commonInitialisms[upper] {
		return upper
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// CamelCase converts a string to CamelCase, splitting on underscores,
// hyphens, and case boundaries. "hello_world" → "HelloWorld".
func CamelCase(s string) string {
	parts := SplitWords(s)
	for i := range parts {
		parts[i] = Title(strings.ToLower(parts[i]))
	}
	return strings.Join(parts, "")
}

// LowerCamelCase converts a string to lowerCamelCase.
// "hello_world" → "helloWorld".
func LowerCamelCase(s string) string {
	c := CamelCase(s)
	if c == "" {
		return c
	}
	r := []rune(c)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// SnakeCase converts a string to snake_case.
// "HelloWorld" → "hello_world".
func SnakeCase(s string) string {
	parts := SplitWords(s)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}

// QualifyType prefixes typeName with qualifier if non-empty.
// QualifyType("store", "Item") → "store.Item".
// QualifyType("", "Item") → "Item".
func QualifyType(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return qualifier + "." + typeName
}

// FormatDocComment prefixes each line of doc with "// " so it can be
// pasted directly into generated Go source. Returns empty string for
// empty doc.
func FormatDocComment(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = "//"
		} else {
			lines[i] = "// " + line
		}
	}
	return strings.Join(lines, "\n")
}

// SplitWords splits a string into words on underscores, hyphens, and
// CamelCase boundaries.
func SplitWords(s string) []string {
	var words []string
	var current []rune
	for i, r := range s {
		switch {
		case r == '_' || r == '-':
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		case unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(rune(s[i-1])):
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// ParamName generates a parameter name for index i: "p0", "p1", etc.
func ParamName(i int) string {
	return "p" + string(rune('0'+i))
}
