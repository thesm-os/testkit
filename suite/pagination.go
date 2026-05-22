// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"go.thesmos.sh/testkit"
)

// PaginationConfig configures one [AssertPaginates] run.
//
// `CursorField` is the reflection name of the cursor field on the
// page struct returned by the method. `EmptyCursor` is the
// zero-equivalent the impl returns when iteration is complete
// (typically the empty string for string cursors, the zero of
// whatever type is in use). `Iterations` caps the loop as a
// runaway safety net.
type PaginationConfig struct {
	CursorField string
	EmptyCursor any
	Iterations  int
}

// AssertPaginates verifies cursor-based iteration: the call closure
// receives a cursor and returns a page; the helper extracts the
// page's `Items` slice and named cursor field via reflection,
// follows the cursor until empty, and asserts no item appears
// twice.
//
// Page must be a struct (or pointer to a struct) with:
//   - an exported `Items` field of slice kind, and
//   - an exported field with the configured CursorField name.
//
// When the page shape doesn't match, the helper fails the test with
// a clear message naming the missing field.
func AssertPaginates[T any, C comparable, Page any](
	cfg PaginationConfig,
	call func(ctx context.Context, impl T, cursor C) (Page, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("pagination terminates without duplication", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			seen := make(map[any]bool)
			var cursor C
			limit := cfg.Iterations
			if limit <= 0 {
				limit = 1000
			}
			for range limit {
				page, err := call(t.Context(), impl, cursor)
				testkit.NoError(t, err, "pagination call must not error")
				items, next, perr := paginationFields(page, cfg.CursorField)
				if perr != nil {
					t.Fatalf("pagination: %v", perr)
				}
				for j := 0; j < items.Len(); j++ {
					it := items.Index(j).Interface()
					if seen[it] {
						t.Fatalf("pagination: item %v yielded twice", it)
					}
					seen[it] = true
				}
				nextC, ok := next.(C)
				if !ok {
					t.Fatalf("pagination: cursor field %q is %T, want %T",
						cfg.CursorField, next, cursor)
				}
				if reflect.DeepEqual(nextC, cfg.EmptyCursor) {
					return
				}
				cursor = nextC
			}
			t.Fatalf("pagination: did not terminate within %d iterations", limit)
		})
	}
}

// paginationFields extracts the `Items` slice and the named cursor
// field from a page struct. Returns a clear error when either
// field is missing or shaped wrong.
func paginationFields(page any, cursorField string) (reflect.Value, any, error) {
	v := reflect.ValueOf(page)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, nil, fmt.Errorf("page type is %s, want struct", v.Kind())
	}
	items := v.FieldByName("Items")
	if !items.IsValid() {
		return reflect.Value{}, nil, errors.New("suite: page struct missing Items field")
	}
	if items.Kind() != reflect.Slice {
		return reflect.Value{}, nil, fmt.Errorf("items field is %s, want slice", items.Kind())
	}
	cursor := v.FieldByName(cursorField)
	if !cursor.IsValid() {
		return reflect.Value{}, nil, fmt.Errorf("page struct missing %q field", cursorField)
	}
	return items, cursor.Interface(), nil
}
