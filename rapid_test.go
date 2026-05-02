// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"strings"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
)

func TestDrawString(t *testing.T) {
	t.Parallel()

	t.Run("produces prefixed string", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(t *rapid.T) {
			s := testkit.DrawString(t, "user")
			if !strings.HasPrefix(s, "user-") {
				t.Fatalf("must start with prefix, got: %q", s)
			}
			if len(s) != len("user-")+6 {
				t.Fatalf("suffix must be 6 chars, got len %d: %q", len(s), s)
			}
		})
	})
}

func TestDrawBytes(t *testing.T) {
	t.Parallel()

	t.Run("produces bytes within max length", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(t *rapid.T) {
			b := testkit.DrawBytes(t, 32)
			if len(b) > 32 {
				t.Fatalf("must be <= 32 bytes, got %d", len(b))
			}
		})
	})
}

func TestDrawEnum(t *testing.T) {
	t.Parallel()

	t.Run("produces value in range", func(t *testing.T) {
		t.Parallel()
		type Status int
		rapid.Check(t, func(t *rapid.T) {
			v := testkit.DrawEnum[Status](t, 5)
			if v < 0 || v > 5 {
				t.Fatalf("must be in [0, 5], got %d", v)
			}
		})
	})
}

func TestDrawUint64(t *testing.T) {
	t.Parallel()

	t.Run("produces a value", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(t *rapid.T) {
			_ = testkit.DrawUint64(t) // just verify it doesn't panic
		})
	})
}

func TestDrawInt(t *testing.T) {
	t.Parallel()

	t.Run("produces value in range", func(t *testing.T) {
		t.Parallel()
		rapid.Check(t, func(t *rapid.T) {
			v := testkit.DrawInt(t, 10, 20)
			if v < 10 || v > 20 {
				t.Fatalf("must be in [10, 20], got %d", v)
			}
		})
	})
}
