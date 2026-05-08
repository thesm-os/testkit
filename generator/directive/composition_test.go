// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"go/token"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

// makeDirs builds a slice of [directive.Directive] from names, attaching
// the args those names need to satisfy ArgSpec validation. Tests reach
// for this when they care about composition, not arg syntax.
func makeDirs(names ...string) []directive.Directive {
	out := make([]directive.Directive, len(names))
	for i, n := range names {
		args := []string{}
		switch n {
		case "retry-succeeds-on-attempt":
			args = []string{"3"}
		case "sideeffect", "monotonic", "order-after":
			args = []string{"X"}
		case "timeout":
			args = []string{"1s"}
		case "bounded":
			args = []string{"0..1"}
		}
		out[i] = directive.Directive{Name: n, Args: args}
	}
	return out
}

func TestComposition(t *testing.T) {
	t.Parallel()

	t.Run("clean composition produces no issues", func(t *testing.T) {
		t.Parallel()
		issues := directive.Issues(makeDirs("atomic", "idempotent"))
		testkit.Len(t, issues, 0, "no composition issues")
	})

	t.Run("explicit conflict pairs are flagged", func(t *testing.T) {
		t.Parallel()
		issues := directive.Issues(makeDirs("pure", "sideeffect"))
		assertHasIssue(t, issues, directive.Conflict, "pure vs sideeffect")
	})

	t.Run("transitive Implies pulls in conflicts", func(t *testing.T) {
		t.Parallel()
		// cacheable Implies pure — pure conflicts monotonic — so cacheable
		// transitively conflicts monotonic, *without* repeating the entry
		// in cacheable's own Conflicts list.
		issues := directive.Issues(makeDirs("cacheable", "monotonic"))
		assertHasIssue(t, issues, directive.Conflict, "cacheable vs monotonic via transitive closure")
	})

	t.Run("concurrent + concurrent-readers conflict", func(t *testing.T) {
		t.Parallel()
		issues := directive.Issues(makeDirs("concurrent", "concurrent-readers"))
		assertHasIssue(t, issues, directive.Conflict, "concurrency models are mutually exclusive")
	})

	t.Run("missing required companion is flagged", func(t *testing.T) {
		t.Parallel()
		issues := directive.Issues(makeDirs("retry-succeeds-on-attempt"))
		assertHasIssue(t, issues, directive.MissingRequired, "retry-succeeds requires retryable")
	})

	t.Run("required companion satisfied passes", func(t *testing.T) {
		t.Parallel()
		issues := directive.Issues(makeDirs("retry-succeeds-on-attempt", "retryable"))
		for _, iss := range issues {
			testkit.True(t, iss.Kind != directive.MissingRequired, "no missing-required when companion present")
		}
	})

	t.Run("ValidateComposition returns nil when no hard issues", func(t *testing.T) {
		t.Parallel()
		err := directive.ValidateComposition(makeDirs("atomic", "idempotent"), token.Position{})
		testkit.NoError(t, err, "clean composition")
	})

	t.Run("ValidateComposition surfaces conflicts as positioned errors", func(t *testing.T) {
		t.Parallel()
		err := directive.ValidateComposition(makeDirs("pure", "sideeffect"), token.Position{Filename: "x.go", Line: 5})
		testkit.True(t, err != nil, "conflict surfaces")
	})

	t.Run("ValidateComposition surfaces missing-required as error", func(t *testing.T) {
		t.Parallel()
		err := directive.ValidateComposition(makeDirs("retry-succeeds-on-attempt"), token.Position{})
		testkit.True(t, err != nil, "missing-required surfaces")
	})
}

// assertHasIssue is a tiny test helper that fails when none of the given
// issues match the requested Kind. Inlined here so each subtest reads
// "find this kind of issue" without a deep helper chain.
func assertHasIssue(t *testing.T, issues []directive.Issue, kind directive.IssueKind, msg string) {
	t.Helper()
	for _, iss := range issues {
		if iss.Kind == kind {
			return
		}
	}
	t.Errorf("expected %v issue (%s); got %+v", kind, msg, issues)
}
