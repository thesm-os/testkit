// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Equal compares got and want using [cmp.Diff] and calls tb.Fatalf with a
// structural diff if they differ. The msg argument describes the contract
// being checked and appears as the first line of the failure output.
//
//	testkit.Equal(t, store.Get(ctx, id), item, "Get must return the stored item")
func Equal[T any](tb testing.TB, got, want T, msg string) {
	tb.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
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
	if cmp.Equal(got, want) {
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

// AssertSequence calls tb.Fatalf if any adjacent pair (items[i-1], items[i])
// does not satisfy pred. Empty and singleton slices pass trivially. The
// failure message cites the index and both values of the first violation.
//
//	testkit.AssertSequence(t, timestamps, func(a, b time.Time) bool {
//	    return a.Before(b)
//	}, "events must be in chronological order")
func AssertSequence[T any](tb testing.TB, items []T, pred func(earlier, later T) bool, msg string) {
	tb.Helper()
	for i := 1; i < len(items); i++ {
		if !pred(items[i-1], items[i]) {
			tb.Fatalf("%s: sequence violated at index %d: %v → %v",
				msg, i, items[i-1], items[i])
		}
	}
}
