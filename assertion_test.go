// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestAssertion_Equals(t *testing.T) {
	t.Parallel()

	t.Run("matching values pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).Equals(42, "must match")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("different values fail with diff", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "got").Equals("want", "body")
		if !f.Failed() {
			t.Fatal("should fail")
		}
		if !strings.Contains(f.Msg(), "body") {
			t.Fatalf("should include context, got: %s", f.Msg())
		}
	})
}

func TestAssertion_NotEquals(t *testing.T) {
	t.Parallel()

	t.Run("different values pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 1).NotEquals(2, "must differ")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("identical values fail", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 1).NotEquals(1, "must differ")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})
}

func TestAssertion_IsNil(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		var p *int
		testkit.Assert(f, p).IsNil("must be nil")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-nil pointer fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		v := 42
		testkit.Assert(f, &v).IsNil("must be nil")
		if !f.Failed() {
			t.Fatal("should fail for non-nil")
		}
	})

	t.Run("non-nillable type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).IsNil("int cannot be nil")
		if !f.Failed() {
			t.Fatal("should fail for non-nillable type")
		}
	})
}

func TestAssertion_IsNotNil(t *testing.T) {
	t.Parallel()

	t.Run("non-nil passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		v := 42
		testkit.Assert(f, &v).IsNotNil("must exist")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("nil pointer fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		var p *int
		testkit.Assert(f, p).IsNotNil("must exist")
		if !f.Failed() {
			t.Fatal("should fail for nil")
		}
	})
}

func TestAssertion_HasLen(t *testing.T) {
	t.Parallel()

	t.Run("correct length passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{1, 2}).HasLen(2, "must have 2")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("wrong length fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{1}).HasLen(5, "must have 5")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("unsupported type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).HasLen(1, "int has no length")
		if !f.Failed() {
			t.Fatal("should fail for unsupported type")
		}
	})
}

func TestAssertion_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty slice passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{}).IsEmpty("must be empty")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-empty fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{1}).IsEmpty("must be empty")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("unsupported type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).IsEmpty("int has no length")
		if !f.Failed() {
			t.Fatal("should fail for unsupported type")
		}
	})
}

func TestAssertion_IsNotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("non-empty passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{1}).IsNotEmpty("must have items")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("empty fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{}).IsNotEmpty("must have items")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("unsupported type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).IsNotEmpty("int has no length")
		if !f.Failed() {
			t.Fatal("should fail for unsupported type")
		}
	})
}

func TestAssertion_Contains(t *testing.T) {
	t.Parallel()

	t.Run("string contains substring", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello world").Contains("world", "must contain")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("string missing substring fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello").Contains("world", "must contain")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("slice contains element", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []int{1, 2, 3}).Contains(2, "must contain")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("map contains key", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, map[string]int{"a": 1}).Contains("a", "must contain key")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("byte slice contains substring", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []byte("hello world")).Contains([]byte("world"), "must contain")
		if f.Failed() {
			t.Fatalf("should pass for []byte, got: %s", f.Msg())
		}
	})

	t.Run("string with non-string needle fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello").Contains(42, "string + int needle")
		if !f.Failed() {
			t.Fatal("should fail when needle is not string-like")
		}
	})

	t.Run("map with wrong key type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, map[string]int{"a": 1}).Contains(42, "wrong key type")
		if !f.Failed() {
			t.Fatal("should fail when needle type does not match key type")
		}
	})

	t.Run("unsupported type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).Contains(1, "int has no contains")
		if !f.Failed() {
			t.Fatal("should fail for unsupported type")
		}
	})
}

func TestAssertion_NotContains(t *testing.T) {
	t.Parallel()

	t.Run("missing element passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello").NotContains("xyz", "must not contain")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("present element fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello").NotContains("ell", "must not contain")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("unsupported type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).NotContains(1, "int has no contains")
		if !f.Failed() {
			t.Fatal("should fail for unsupported type")
		}
	})
}

func TestAssertion_IsError(t *testing.T) {
	t.Parallel()

	t.Run("matching error passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		sentinel := errors.New("sentinel")
		testkit.Assert(f, sentinel).IsError(sentinel, "must match")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-matching error fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, errors.New("other")).IsError(errors.New("sentinel"), "must match")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("non-error type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "not an error").IsError(errors.New("x"), "must be error")
		if !f.Failed() {
			t.Fatal("should fail for non-error type")
		}
	})
}

func TestAssertion_IsNotError(t *testing.T) {
	t.Parallel()

	t.Run("different errors pass", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, errors.New("a")).IsNotError(errors.New("b"), "must not match")
		if f.Failed() {
			t.Fatalf("should pass for different errors, got: %s", f.Msg())
		}
	})

	t.Run("same sentinel fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		sentinel := errors.New("sentinel")
		testkit.Assert(f, sentinel).IsNotError(sentinel, "must not match")
		if !f.Failed() {
			t.Fatal("should fail for same sentinel")
		}
	})

	t.Run("non-error type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "not an error").IsNotError(errors.New("x"), "must be error")
		if !f.Failed() {
			t.Fatal("should fail for non-error type")
		}
	})
}

func TestAssertion_Matches(t *testing.T) {
	t.Parallel()

	t.Run("matching pattern passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "abc-123").Matches(`^[a-z]+-\d+$`, "must match pattern")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-matching pattern fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "abc").Matches(`^\d+$`, "must be digits")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("byte slice matches pattern", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, []byte("hello")).Matches(`^hello$`, "must match")
		if f.Failed() {
			t.Fatalf("should pass for []byte, got: %s", f.Msg())
		}
	})

	t.Run("invalid pattern fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "abc").Matches(`[invalid`, "bad regex")
		if !f.Failed() {
			t.Fatal("should fail for invalid pattern")
		}
	})

	t.Run("non-string type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).Matches(`\d+`, "int is not string")
		if !f.Failed() {
			t.Fatal("should fail for non-string type")
		}
	})
}

func TestAssertion_Approximately(t *testing.T) {
	t.Parallel()

	t.Run("within tolerance passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 3.14).Approximately(3.14159, 0.01, "must be close to pi")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("outside tolerance fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 3.0).Approximately(3.14159, 0.01, "must be close to pi")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("float32 works", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, float32(3.14)).Approximately(3.14, 0.01, "must be close")
		if f.Failed() {
			t.Fatalf("should pass for float32, got: %s", f.Msg())
		}
	})

	t.Run("non-numeric type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "not a number").Approximately(1.0, 0.1, "must be numeric")
		if !f.Failed() {
			t.Fatal("should fail for non-numeric type")
		}
	})
}

func TestAssertion_Within(t *testing.T) {
	t.Parallel()

	t.Run("value in range passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 5).Within(1, 10, "must be in range")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("value out of range fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 15).Within(1, 10, "must be in range")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("uint works", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, uint(5)).Within(1, 10, "must be in range")
		if f.Failed() {
			t.Fatalf("should pass for uint, got: %s", f.Msg())
		}
	})

	t.Run("non-numeric type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "not a number").Within(1, 10, "must be numeric")
		if !f.Failed() {
			t.Fatal("should fail for non-numeric type")
		}
	})
}

func TestAssertion_Panics(t *testing.T) {
	t.Parallel()

	t.Run("panicking function passes", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, func() { panic("boom") }).Panics("must panic")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("non-panicking function fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, func() {}).Panics("must panic")
		if !f.Failed() {
			t.Fatal("should fail")
		}
	})

	t.Run("non-func type fails", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, 42).Panics("must be func")
		if !f.Failed() {
			t.Fatal("should fail for non-func type")
		}
	})
}

func TestAssertion_chaining(t *testing.T) {
	t.Parallel()

	t.Run("multiple matchers AND together", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello world").
			Contains("hello", "must contain hello").
			Contains("world", "must contain world").
			HasLen(11, "must be 11 chars")
		if f.Failed() {
			t.Fatalf("should pass, got: %s", f.Msg())
		}
	})

	t.Run("first failing matcher stops chain", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.Assert(f, "hello").
			HasLen(99, "wrong length").
			Contains("xyz", "second should not matter")
		if !f.Failed() {
			t.Fatal("should fail on first matcher")
		}
		if !strings.Contains(f.Msg(), "wrong length") {
			t.Fatalf("should report first failure, got: %s", f.Msg())
		}
	})
}
