// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Assertion provides a fluent chain for multi-property assertions on a single
// subject. Every matcher calls tb.Fatalf on failure and returns the receiver
// for chaining (AND logic). Because [FailableTB] captures the first fatal
// and ignores subsequent calls, only the first failing matcher in a chain
// produces output.
//
//	testkit.Assert(t, user).
//	    IsNotNil("must exist").
//	    HasLen(3, "must have 3 fields populated")
type Assertion[T any] struct {
	tb  testing.TB
	got T
}

// Assert starts a fluent assertion chain on got. Call matchers on the returned
// [Assertion] to verify properties of got. Each matcher returns the receiver,
// so calls can be chained.
//
//	testkit.Assert(t, resp.StatusCode).
//	    Equals(200, "status must be OK")
func Assert[T any](tb testing.TB, got T) *Assertion[T] {
	tb.Helper()
	return &Assertion[T]{tb: tb, got: got}
}

// Equals compares got and want using [cmp.Diff] and calls tb.Fatalf with a
// structural diff if they differ.
func (a *Assertion[T]) Equals(want T, msg string) *Assertion[T] {
	a.tb.Helper()
	if diff := cmp.Diff(want, a.got, defaultCmpOpts...); diff != "" {
		a.tb.Fatalf("%s: (-want +got)\n%s", msg, diff)
	}
	return a
}

// NotEquals calls tb.Fatalf if got and want are structurally equal according
// to [cmp.Equal].
func (a *Assertion[T]) NotEquals(want T, msg string) *Assertion[T] {
	a.tb.Helper()
	if cmp.Equal(a.got, want, defaultCmpOpts...) {
		a.tb.Fatalf("%s: values are equal, want different\n got: %+v", msg, a.got)
	}
	return a
}

// IsNil calls tb.Fatalf if got is not nil. Handles typed nils correctly —
// a (*T)(nil) interface value is considered nil.
func (a *Assertion[T]) IsNil(msg string) *Assertion[T] {
	a.tb.Helper()
	if !isNilValue(a.got) {
		a.tb.Fatalf("%s: expected nil, got %+v", msg, a.got)
	}
	return a
}

// IsNotNil calls tb.Fatalf if got is nil. Handles typed nils correctly.
func (a *Assertion[T]) IsNotNil(msg string) *Assertion[T] {
	a.tb.Helper()
	if isNilValue(a.got) {
		a.tb.Fatalf("%s: expected non-nil", msg)
	}
	return a
}

// HasLen calls tb.Fatalf if the length of got does not equal want. Supports
// array, slice, map, channel, and string values.
func (a *Assertion[T]) HasLen(want int, msg string) *Assertion[T] {
	a.tb.Helper()
	v := reflect.ValueOf(a.got)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.String:
		if v.Len() != want {
			a.tb.Fatalf("%s: expected length %d, got %d", msg, want, v.Len())
		}
	default:
		a.tb.Fatalf("%s: HasLen not supported for %T", msg, a.got)
	}
	return a
}

// IsEmpty calls tb.Fatalf if got is not empty. Supports array, slice, map,
// channel, and string values.
func (a *Assertion[T]) IsEmpty(msg string) *Assertion[T] {
	a.tb.Helper()
	v := reflect.ValueOf(a.got)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.String:
		if v.Len() != 0 {
			a.tb.Fatalf("%s: expected empty, got length %d", msg, v.Len())
		}
	default:
		a.tb.Fatalf("%s: IsEmpty not supported for %T", msg, a.got)
	}
	return a
}

// IsNotEmpty calls tb.Fatalf if got is empty. Supports array, slice, map,
// channel, and string values.
func (a *Assertion[T]) IsNotEmpty(msg string) *Assertion[T] {
	a.tb.Helper()
	v := reflect.ValueOf(a.got)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.String:
		if v.Len() == 0 {
			a.tb.Fatalf("%s: expected non-empty", msg)
		}
	default:
		a.tb.Fatalf("%s: IsNotEmpty not supported for %T", msg, a.got)
	}
	return a
}

// Contains calls tb.Fatalf if got does not contain needle. For strings and
// []byte, it checks for a substring. For slices and arrays, it checks for an
// element using [reflect.DeepEqual]. For maps, it checks for a key.
func (a *Assertion[T]) Contains(needle any, msg string) *Assertion[T] {
	a.tb.Helper()
	found, supported := contains(a.got, needle)
	if !supported {
		a.tb.Fatalf("%s: Contains not supported for %T", msg, a.got)
	}
	if !found {
		a.tb.Fatalf("%s: %+v does not contain %+v", msg, a.got, needle)
	}
	return a
}

// NotContains calls tb.Fatalf if got contains needle. See [Assertion.Contains]
// for the containment rules.
func (a *Assertion[T]) NotContains(needle any, msg string) *Assertion[T] {
	a.tb.Helper()
	found, supported := contains(a.got, needle)
	if !supported {
		a.tb.Fatalf("%s: NotContains not supported for %T", msg, a.got)
	}
	if found {
		a.tb.Fatalf("%s: %+v should not contain %+v", msg, a.got, needle)
	}
	return a
}

// IsError calls tb.Fatalf if got (which must implement error) does not satisfy
// [errors.Is](got, target). If got is not an error, it fatals immediately.
func (a *Assertion[T]) IsError(target error, msg string) *Assertion[T] {
	a.tb.Helper()
	err, ok := any(a.got).(error)
	if !ok {
		a.tb.Fatalf("%s: got is not an error: %+v", msg, a.got)
		return a
	}
	ErrorIs(a.tb, err, target, msg)
	return a
}

// IsNotError calls tb.Fatalf if got (which must implement error) satisfies
// [errors.Is](got, target). Use to assert two errors are distinct.
func (a *Assertion[T]) IsNotError(target error, msg string) *Assertion[T] {
	a.tb.Helper()
	err, ok := any(a.got).(error)
	if !ok {
		a.tb.Fatalf("%s: got is not an error: %+v", msg, a.got)
		return a
	}
	ErrorIsNot(a.tb, err, target, msg)
	return a
}

// Matches calls tb.Fatalf if got (string or []byte) does not match the
// regular expression pattern. An invalid pattern is itself a fatal error.
//
//	testkit.Assert(t, id).Matches(`^[a-f0-9]{32}$`, "must be a hex UUID")
func (a *Assertion[T]) Matches(pattern, msg string) *Assertion[T] {
	a.tb.Helper()
	s, ok := stringOf(a.got)
	if !ok {
		a.tb.Fatalf("%s: Matches requires string or []byte, got %T", msg, a.got)
		return a
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		a.tb.Fatalf("%s: invalid pattern %q: %v", msg, pattern, err)
		return a
	}
	if !re.MatchString(s) {
		a.tb.Fatalf("%s: %q does not match pattern %q", msg, s, pattern)
	}
	return a
}

// IsApproximately calls tb.Fatalf if got (any numeric type) is not within
// tolerance of want. Comparison uses absolute difference:
// |got - want| <= tolerance.
//
//	testkit.Assert(t, elapsed.Seconds()).IsApproximately(1.0, 0.05, "must complete in ~1s")
func (a *Assertion[T]) IsApproximately(want, tolerance float64, msg string) *Assertion[T] {
	a.tb.Helper()
	f, ok := floatOf(a.got)
	if !ok {
		a.tb.Fatalf("%s: Approximately requires numeric type, got %T", msg, a.got)
		return a
	}
	if math.Abs(f-want) > tolerance {
		a.tb.Fatalf("%s: %v not within %v of %v", msg, f, tolerance, want)
	}
	return a
}

// IsWithin calls tb.Fatalf if got (any numeric type) is not in the closed
// range [lo, hi].
//
//	testkit.Assert(t, port).IsWithin(1024, 65535, "must be an unprivileged port")
func (a *Assertion[T]) IsWithin(lo, hi float64, msg string) *Assertion[T] {
	a.tb.Helper()
	f, ok := floatOf(a.got)
	if !ok {
		a.tb.Fatalf("%s: Within requires numeric type, got %T", msg, a.got)
		return a
	}
	if f < lo || f > hi {
		a.tb.Fatalf("%s: %v not in range [%v, %v]", msg, f, lo, hi)
	}
	return a
}

// Panics calls tb.Fatalf if got (which must be a func()) does not panic. If
// got is not a func(), it fatals immediately.
func (a *Assertion[T]) Panics(msg string) *Assertion[T] {
	a.tb.Helper()
	fn, ok := any(a.got).(func())
	if !ok {
		a.tb.Fatalf("%s: Panics requires func(), got %T", msg, a.got)
		return a
	}
	panicked := true
	func() {
		defer func() { _ = recover() }()
		fn()
		panicked = false
	}()
	if !panicked {
		a.tb.Fatalf("%s: expected panic, got none", msg)
	}
	return a
}

// --- helpers ---

// isNilValue reports whether v is nil, including typed nil interface values
// such as (*T)(nil) stored in an any.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

// contains reports whether haystack contains needle. It returns (found,
// supported) where supported is false if the haystack type is not one of
// string, []byte, slice, array, or map.
func contains(haystack, needle any) (bool, bool) {
	if hs, ok := stringOf(haystack); ok {
		ns, ok := stringOf(needle)
		if !ok {
			return false, false
		}
		return strings.Contains(hs, ns), true
	}

	v := reflect.ValueOf(haystack)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if cmp.Equal(v.Index(i).Interface(), needle, defaultCmpOpts...) {
				return true, true
			}
		}
		return false, true
	case reflect.Map:
		key := reflect.ValueOf(needle)
		if key.IsValid() && key.Type().AssignableTo(v.Type().Key()) {
			return v.MapIndex(key).IsValid(), true
		}
		return false, true
	default:
		return false, false
	}
}

// stringOf extracts a string from a string or []byte value.
func stringOf(v any) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	case []byte:
		return string(s), true
	default:
		return "", false
	}
}

// floatOf converts any Go numeric type to float64 for approximate comparison.
func floatOf(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}
