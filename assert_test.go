// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("identical ints pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Equal(f, 42, 42, "must be equal")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("identical structs pass", func(t *testing.T) {
		t.Parallel()
		type point struct{ X, Y int }
		f := testkit.NewFailableTB()
		testkit.Equal(f, point{1, 2}, point{1, 2}, "must be equal")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("different values fail", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Equal(f, 1, 2, "values must match")
		if !f.Failed() {
			t.Fatal("should fail for different values")
		}
	})

	t.Run("failure message includes context and diff", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Equal(f, "got", "want", "response body")
		msg := f.Msg()
		if !strings.Contains(msg, "response body") {
			t.Fatalf("message should include context, got: %s", msg)
		}
		if !strings.Contains(msg, "-") {
			t.Fatalf("message should include diff markers, got: %s", msg)
		}
	})

	t.Run("nil slices pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		var a, b []string
		testkit.Equal(f, a, b, "nil slices must be equal")
		if f.Failed() {
			t.Fatalf("should pass for nil slices, got: %s", f.Msg())
		}
	})

	t.Run("calls tb.Helper", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Equal(f, 1, 1, "pass")
		if f.HelperCalls() == 0 {
			t.Fatal("should call tb.Helper()")
		}
	})
}

func TestNotEqual(t *testing.T) {
	t.Parallel()

	t.Run("different values pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.NotEqual(f, 1, 2, "must differ")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("identical values fail", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.NotEqual(f, "same", "same", "must differ")
		if !f.Failed() {
			t.Fatal("should fail for identical values")
		}
	})

	t.Run("failure message includes context", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.NotEqual(f, 42, 42, "user ID")
		msg := f.Msg()
		if !strings.Contains(msg, "user ID") {
			t.Fatalf("message should include context, got: %s", msg)
		}
	})
}

var errSentinel = errors.New("sentinel")

func TestErrorIs(t *testing.T) {
	t.Parallel()

	t.Run("matching sentinel passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.ErrorIs(f, errSentinel, errSentinel, "must match")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("wrapped sentinel passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		wrapped := errors.Join(errSentinel, errors.New("extra"))
		testkit.ErrorIs(f, wrapped, errSentinel, "must unwrap")
		if f.Failed() {
			t.Fatalf("should pass on wrapped, got: %s", f.Msg())
		}
	})

	t.Run("non-matching error fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.ErrorIs(f, errors.New("other"), errSentinel, "must match")
		if !f.Failed() {
			t.Fatal("should fail for non-matching error")
		}
	})
}

type customError struct{ Code int }

func (*customError) Error() string { return "custom" }

func TestErrorAs(t *testing.T) {
	t.Parallel()

	t.Run("matching type passes and returns value", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		got := testkit.ErrorAs[*customError](f, &customError{Code: 42}, "must unwrap")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
		if got.Code != 42 {
			t.Fatalf("should return unwrapped value, got code: %d", got.Code)
		}
	})

	t.Run("non-matching type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		_ = testkit.ErrorAs[*customError](f, errors.New("plain"), "must unwrap")
		if !f.Failed() {
			t.Fatal("should fail for non-matching type")
		}
	})
}

func TestNoError(t *testing.T) {
	t.Parallel()

	t.Run("nil error passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.NoError(f, nil, "must succeed")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-nil error fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.NoError(f, errors.New("boom"), "must succeed")
		if !f.Failed() {
			t.Fatal("should fail for non-nil error")
		}
		if !strings.Contains(f.Msg(), "boom") {
			t.Fatalf("message should include error, got: %s", f.Msg())
		}
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("non-nil error passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Error(f, errors.New("boom"), "must have error")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("nil error fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Error(f, nil, "must have error")
		if !f.Failed() {
			t.Fatal("should fail for nil error")
		}
	})
}

func TestTrue(t *testing.T) {
	t.Parallel()

	t.Run("true passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.True(f, true, "must be true")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("false fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.True(f, false, "must be true")
		if !f.Failed() {
			t.Fatal("should fail for false")
		}
	})
}

func TestFalse(t *testing.T) {
	t.Parallel()

	t.Run("false passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.False(f, false, "must be false")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("true fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.False(f, true, "must be false")
		if !f.Failed() {
			t.Fatal("should fail for true")
		}
	})
}

func TestLen(t *testing.T) {
	t.Parallel()

	t.Run("slice with matching length passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Len(f, []int{1, 2, 3}, 3, "slice length")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("map with matching length passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Len(f, map[string]int{"a": 1}, 1, "map length")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("string with matching length passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Len(f, "abc", 3, "string length")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("wrong length fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Len(f, []int{1}, 5, "slice length")
		if !f.Failed() {
			t.Fatal("should fail for wrong length")
		}
	})

	t.Run("unsupported type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Len(f, 42, 1, "int has no length")
		if !f.Failed() {
			t.Fatal("should fail for unsupported type")
		}
	})
}

func TestPanics(t *testing.T) {
	t.Parallel()

	t.Run("panicking function passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		v := testkit.Panics(f, func() { panic("boom") }, "must panic")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
		if v != "boom" {
			t.Fatalf("should return recovered value, got: %v", v)
		}
	})

	t.Run("non-panicking function fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Panics(f, func() {}, "must panic")
		if !f.Failed() {
			t.Fatal("should fail when function does not panic")
		}
	})
}

func TestAssertSequence(t *testing.T) {
	t.Parallel()

	t.Run("ascending ints pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertSequence(f, []int{1, 2, 3}, func(a, b int) bool { return a < b }, "ascending")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-ascending ints fail", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertSequence(f, []int{1, 3, 2}, func(a, b int) bool { return a < b }, "ascending")
		if !f.Failed() {
			t.Fatal("should fail for non-ascending")
		}
		if !strings.Contains(f.Msg(), "index 2") {
			t.Fatalf("should cite failing index, got: %s", f.Msg())
		}
	})

	t.Run("empty slice passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertSequence(f, []int{}, func(a, b int) bool { return a < b }, "empty")
		if f.Failed() {
			t.Fatalf("empty slice should pass, got: %s", f.Msg())
		}
	})

	t.Run("singleton passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.AssertSequence(f, []int{42}, func(a, b int) bool { return a < b }, "single")
		if f.Failed() {
			t.Fatalf("singleton should pass, got: %s", f.Msg())
		}
	})
}
