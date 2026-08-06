// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package samples picks the values a generated check writes for a Go builtin.
//
// A check that sets a field and compares it against the zero value it already
// held passes against code that did nothing, so every generator emitting one
// needs a value distinguishable from what was there before. The rule is the
// same wherever it is asked, and a second derivation would be free to disagree
// about it — which is why this is shared rather than copied.
package samples

import "strings"

// For returns two distinct values of the named Go builtin as source text, or
// two empty strings when the type admits none.
//
// Two values rather than one, because a check comparing against a single value
// passes whenever the subject already held it, and what it held is not always
// knowable — a constructor's seed may come from a function this generator
// cannot read. Whatever it was equals at most one of a pair.
//
// The string form carries the field's own name so a value appearing in a
// failure message says where it came from.
//
// Returned as source text rather than as a reference because every value is a
// builtin literal: it names no package, so nothing here can produce something
// the rendered file would have to import.
//
// Only builtins are answered. A defined type is recorded by name, so `Weekday
// int` arrives as `Weekday` with no way to learn it is an integer, and a
// literal written for it would not compile. Resolving those is the caller's
// job — see the builder's resolver, which reads the loaded graph.
//
// The arms spell Go's own type names, which several unrelated tables in this
// module also spell — witness's palette, the double's no-import set. They answer
// different questions about one domain rather than sharing an answer, so
// hoisting the names to constants would put an identifier between each table and
// the thing it is about, and is suppressed instead.
//
//nolint:goconst // see above.
func For(typeName, fieldName string) (sample, alternate string) {
	switch typeName {
	case "string":
		lower := strings.ToLower(fieldName)
		return `"test-` + lower + `"`, `"other-` + lower + `"`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune":
		return "42", "7"
	case "float32", "float64":
		return "3.14", "2.72"
	case "complex64", "complex128":
		return "1 + 2i", "3 + 4i"
	case "bool":
		// The only type whose pair exhausts its values, which is what makes a
		// bool check the strictest of them: code that assigned nothing fails
		// against one arm no matter what was there before.
		return "true", "false"
	}
	return "", ""
}
