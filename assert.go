// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// defaultCmpOpts are applied to all cmp.Diff/cmp.Equal calls in
// testkit assertions. Exporter(true) allows comparing unexported
// fields — prevents panics on types like time.Time or any struct
// with private fields. EquateEmpty treats nil and empty slices/maps
// as equal (a common source of false negatives in test assertions).
var defaultCmpOpts = cmp.Options{
	cmp.Exporter(func(reflect.Type) bool { return true }),
	cmpopts.EquateEmpty(),
}

// Equal compares got and want using [cmp.Diff] and calls tb.Fatalf with a
// structural diff if they differ. The msg argument describes the contract
// being checked and appears as the first line of the failure output.
//
//	testkit.Equal(t, store.Get(ctx, id), item, "Get must return the stored item")
func Equal[T any](tb testing.TB, got, want T, msg string) {
	tb.Helper()
	if diff := cmp.Diff(want, got, defaultCmpOpts...); diff != "" {
		tb.Fatalf("%s: (-want +got)\n%s", msg, diff)
	}
}

// NotEqual calls tb.Fatalf if got and want are structurally equal according
// to [cmp.Equal]. Use this when a value must have changed or must differ from
// a known bad value.
//
//	testkit.NotEqual(t, token, previousToken, "Refresh must issue a new token")
func NotEqual[T any](tb testing.TB, got, want T, msg string) {
	tb.Helper()
	if cmp.Equal(got, want, defaultCmpOpts...) {
		tb.Fatalf("%s: values are equal, want different\n got: %+v", msg, got)
	}
}

// ErrorIs calls tb.Fatalf if [errors.Is](err, target) returns false. Use this
// to verify that a function returns a specific sentinel error, even when the
// error is wrapped.
//
//	testkit.ErrorIs(t, err, store.ErrNotFound, "Get on missing key must return ErrNotFound")
func ErrorIs(tb testing.TB, err, target error, msg string) {
	tb.Helper()
	if !errors.Is(err, target) {
		tb.Fatalf("%s: got error %v, want %v", msg, err, target)
	}
}

// ErrorIsNot calls tb.Fatalf if [errors.Is](err, target) returns true. Use
// this to verify that two errors are distinct — they must not satisfy Is.
//
//	testkit.ErrorIsNot(t, errA, errB, "ErrNotFound must not match ErrConflict")
func ErrorIsNot(tb testing.TB, err, target error, msg string) {
	tb.Helper()
	if errors.Is(err, target) {
		tb.Fatalf("%s: errors.Is(%v, %v) = true, want false", msg, err, target)
	}
}

// ErrorAs calls tb.Fatalf if [errors.As] cannot unwrap err into a value of
// type T. On success it returns the unwrapped value, allowing further
// inspection without a second type assertion.
//
//	apiErr := testkit.ErrorAs[*apierror.Error](t, err, "must return API error")
//	testkit.Equal(t, apiErr.Code, 404, "API error code must be 404")
func ErrorAs[T any](tb testing.TB, err error, msg string) T {
	tb.Helper()
	var target T
	if !errors.As(err, &target) {
		tb.Fatalf("%s: got error %v, want type %T", msg, err, target)
	}
	return target
}

// NoError calls tb.Fatalf if err is not nil. Use this for operations that
// must succeed — the error message includes the unexpected error value.
//
//	testkit.NoError(t, store.Put(ctx, item), "Put must succeed")
func NoError(tb testing.TB, err error, msg string) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

// Error calls tb.Fatalf if err is nil. Use this when an operation must fail
// but you do not care which specific error is returned — pair with [ErrorIs]
// or [ErrorAs] when the sentinel matters.
//
//	testkit.Error(t, store.Put(ctx, duplicate), "Put duplicate must fail")
func Error(tb testing.TB, err error, msg string) {
	tb.Helper()
	if err == nil {
		tb.Fatalf("%s: expected error, got nil", msg)
	}
}

// True calls tb.Fatalf if cond is false.
//
//	testkit.True(t, user.IsActive(), "user must be active after confirmation")
func True(tb testing.TB, cond bool, msg string) {
	tb.Helper()
	if !cond {
		tb.Fatalf("%s: expected true, got false", msg)
	}
}

// False calls tb.Fatalf if cond is true.
//
//	testkit.False(t, user.IsDeleted(), "user must not be deleted")
func False(tb testing.TB, cond bool, msg string) {
	tb.Helper()
	if cond {
		tb.Fatalf("%s: expected false, got true", msg)
	}
}

// Len calls tb.Fatalf if the length of obj does not equal want. It supports
// array, slice, map, channel, and string values. Passing an unsupported type
// is itself a fatal error.
//
//	testkit.Len(t, items, 3, "List must return exactly 3 items")
func Len(tb testing.TB, obj any, want int, msg string) {
	tb.Helper()
	v := reflect.ValueOf(obj)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.String:
		got := v.Len()
		if got != want {
			tb.Fatalf("%s: expected length %d, got %d", msg, want, got)
		}
	default:
		tb.Fatalf("%s: Len not supported for %T", msg, obj)
	}
}

// Contains calls tb.Fatalf if haystack does not contain needle. For strings
// and []byte it checks for a substring. For slices and arrays it checks for
// an element using [reflect.DeepEqual]. For maps it checks for a key.
//
//	testkit.Contains(t, resp.Body, "success", "response must contain success")
func Contains(tb testing.TB, haystack, needle any, msg string) {
	tb.Helper()
	found, supported := contains(haystack, needle)
	if !supported {
		tb.Fatalf("%s: Contains not supported for %T", msg, haystack)
	}
	if !found {
		tb.Fatalf("%s: %+v does not contain %+v", msg, haystack, needle)
	}
}

// NotContains calls tb.Fatalf if haystack contains needle. See [Contains]
// for the containment rules.
//
//	testkit.NotContains(t, resp.Body, "error", "response must not contain error")
func NotContains(tb testing.TB, haystack, needle any, msg string) {
	tb.Helper()
	found, supported := contains(haystack, needle)
	if !supported {
		tb.Fatalf("%s: NotContains not supported for %T", msg, haystack)
	}
	if found {
		tb.Fatalf("%s: %+v should not contain %+v", msg, haystack, needle)
	}
}

// Panics calls tb.Fatalf if fn does not panic. On success it returns the
// recovered value for further inspection.
//
//	v := testkit.Panics(t, func() { store.MustGet(ctx, "") }, "MustGet on empty key must panic")
//	testkit.Equal(t, v, "empty key", "panic message must describe the problem")
func Panics(tb testing.TB, fn func(), msg string) (recovered any) {
	tb.Helper()
	defer func() {
		recovered = recover()
	}()
	fn()
	tb.Fatalf("%s: expected panic, got none", msg)
	return nil
}

// Sequence calls tb.Fatalf if any adjacent pair (items[i-1], items[i])
// does not satisfy pred. Empty and singleton slices pass trivially. The
// failure message cites the index and both values of the first violation.
//
//	testkit.Sequence(t, timestamps, func(a, b time.Time) bool {
//	    return a.Before(b)
//	}, "events must be in chronological order")
func Sequence[T any](tb testing.TB, items []T, pred func(earlier, later T) bool, msg string) {
	tb.Helper()
	for i := 1; i < len(items); i++ {
		if !pred(items[i-1], items[i]) {
			tb.Fatalf("%s: sequence violated at index %d: %v → %v",
				msg, i, items[i-1], items[i])
		}
	}
}

// HasPrefix calls tb.Fatalf if s does not start with prefix. Useful
// for asserting structural properties of generated or formatted output
// (sentinel error messages, log lines, render headers).
//
//	testkit.HasPrefix(t, err.Error(), "store: ", "errors must carry the package prefix")
func HasPrefix(tb testing.TB, s, prefix, msg string) {
	tb.Helper()
	if !strings.HasPrefix(s, prefix) {
		tb.Fatalf("%s: %q does not start with %q", msg, s, prefix)
	}
}

// HasSuffix calls tb.Fatalf if s does not end with suffix.
//
//	testkit.HasSuffix(t, path, ".gen.go", "generated files must end in .gen.go")
func HasSuffix(tb testing.TB, s, suffix, msg string) {
	tb.Helper()
	if !strings.HasSuffix(s, suffix) {
		tb.Fatalf("%s: %q does not end with %q", msg, s, suffix)
	}
}

// ContainsInOrder calls tb.Fatalf if every needle is not present in
// haystack in the given order. Each needle must appear after the
// previous one's match end. Useful when [Contains] is too lax —
// asserting that fields appear in a specific order in formatted
// output catches Error-format regressions that swap or reorder
// fields.
//
//	testkit.ContainsInOrder(t, err.Error(),
//	    []string{"basic:", "validation:", "test-field", "test-message"},
//	    "Error() must include fields in source order")
func ContainsInOrder(tb testing.TB, haystack string, needles []string, msg string) {
	tb.Helper()
	cursor := 0
	for i, n := range needles {
		idx := strings.Index(haystack[cursor:], n)
		if idx < 0 {
			tb.Fatalf("%s: needle[%d] %q not found after position %d in %q",
				msg, i, n, cursor, haystack)
			return
		}
		cursor += idx + len(n)
	}
}
