// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

// Get retrieves a typed value from an attachment map (e.g.
// [Method.Enrichment] or [Method.Mixins]). Returns the zero value
// and false when the key is missing or the stored value has a
// different type.
//
// Use this for read access from generators / templates:
//
//	if errs, ok := spec.Get[errorsenrich.Payload](method.Enrichment, directive.Errors); ok {
//	    // render fault helpers
//	}
//
// The two-value return makes "directive present and well-typed" a
// single clean check; templates can branch on `ok` without a separate
// presence check.
func Get[T any](m map[string]any, key string) (T, bool) {
	var zero T
	if m == nil {
		return zero, false
	}
	v, ok := m[key]
	if !ok {
		return zero, false
	}
	t, ok := v.(T)
	if !ok {
		return zero, false
	}
	return t, true
}

// Set stores a typed value in an attachment map, allocating the map
// when nil. The receiver is a pointer to a map field so consumers
// can write [Method.Enrichment] / [Method.Mixins] without checking
// for nil first:
//
//	spec.Set(&method.Enrichment, directive.Errors, payload)
//
// Returns the (possibly newly-allocated) map for chaining and so
// callers can inspect the post-write state.
func Set[T any](m *map[string]any, key string, val T) map[string]any {
	if *m == nil {
		*m = make(map[string]any)
	}
	(*m)[key] = val
	return *m
}

// Has reports whether key is present in the attachment map. Faster
// than [Get] when the caller doesn't need the value.
func Has(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}
