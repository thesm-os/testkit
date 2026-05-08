// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestParseSignature(t *testing.T) {
	t.Parallel()

	t.Run("populates HasCtx and NonCtxParams from the leading ctx", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) error }
`)
		testkit.True(t, sig.HasCtx, "HasCtx")
		testkit.Len(t, sig.NonCtxParams, 1, "1 non-ctx param")
	})

	t.Run("HasCtx is false when no ctx parameter is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) error }
`)
		testkit.False(t, sig.HasCtx, "no ctx")
		testkit.Len(t, sig.NonCtxParams, 1, "non-ctx param counted")
	})

	t.Run("Variadic separates the trailing variadic param", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ks ...string) ([]int, error) }
`)
		testkit.True(t, sig.HasCtx, "HasCtx")
		testkit.Len(t, sig.NonCtxParams, 0, "variadic excluded from NonCtxParams")
		testkit.True(t, sig.Variadic != nil, "Variadic populated")
	})

	t.Run("populates HasError, ErrIdx, NonErrResults", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, string, error) }
`)
		testkit.True(t, sig.HasError, "HasError")
		testkit.Equal(t, sig.ErrIdx, 2, "ErrIdx points to last result")
		testkit.Len(t, sig.NonErrResults, 2, "2 non-error results")
	})

	t.Run("ErrIdx is -1 when no error result is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() int }
`)
		testkit.False(t, sig.HasError, "HasError false")
		testkit.Equal(t, sig.ErrIdx, -1, "ErrIdx defaults to -1")
	})

	t.Run("Iter is populated for iter.Seq returns", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "iter"
type I interface { F() iter.Seq[int] }
`)
		testkit.True(t, sig.Iter.IsSeq, "iter.Seq detected")
	})

	t.Run("Iter is populated for iter.Seq2 returns", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "iter"
type I interface { F() iter.Seq2[int, error] }
`)
		testkit.True(t, sig.Iter.IsSeq2, "iter.Seq2 detected")
		testkit.True(t, sig.Iter.Seq2Error, "Seq2Error flagged")
	})

	t.Run("AllResults preserves the full result tuple including error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, string, error) }
`)
		testkit.Len(t, sig.AllResults, 3, "all 3 results")
	})
}

func TestSignature_HasDirective(t *testing.T) {
	t.Parallel()

	t.Run("returns true when a matching directive is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() }
`, directive.Directive{Name: directive.Deleter},
			directive.Directive{Name: directive.Idempotent})
		testkit.True(t, sig.HasDirective(directive.Deleter), "Deleter present")
		testkit.True(t, sig.HasDirective(directive.Idempotent), "Idempotent present")
	})

	t.Run("returns false when the directive name is absent", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() }
`, directive.Directive{Name: directive.Deleter})
		testkit.False(t, sig.HasDirective(directive.Mutator), "Mutator absent")
	})

	t.Run("returns false on an empty directive set", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() }
`)
		testkit.False(t, sig.HasDirective(directive.Deleter), "no directives at all")
	})
}

// TestDirectiveIndifference asserts that detectors which do not
// consult directives (every detector except Deleter and Mutator)
// classify identically whether the method carries unrelated
// directives or not. Guards against a future bug where, say, the
// Reader detector starts inspecting directives and changes behavior
// based on them.
func TestDirectiveIndifference(t *testing.T) {
	t.Parallel()

	const src = `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, error) }
`
	// Reader-shape signature. Throw a bunch of non-shape-affecting
	// directives at it and confirm classification is unchanged.
	noDirs := classifyOne(t, src)
	withDirs := classifyOne(t, src,
		directive.Directive{Name: directive.Atomic},
		directive.Directive{Name: directive.Idempotent},
		directive.Directive{Name: directive.Cacheable},
		directive.Directive{Name: directive.Errors, Args: []string{"ErrNotFound"}},
		directive.Directive{Name: directive.Bounded, Args: []string{"0..1"}},
		directive.Directive{Name: directive.Retryable},
	)

	testkit.Equal(t, noDirs, "Reader", "baseline classification")
	testkit.Equal(t, withDirs, "Reader", "non-shape-affecting directives must not change classification")
}
