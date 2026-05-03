// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directiveparse

import (
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directives"
)

// Re-export directive name constants for backward compatibility.
// The canonical definitions live in gen/directives.
const (
	DirPure              = directives.Pure
	DirCacheable         = directives.Cacheable
	DirIdempotent        = directives.Idempotent
	DirMonotonic         = directives.Monotonic
	DirSideEffect        = directives.SideEffect
	DirConcurrent        = directives.Concurrent
	DirConcurrentReaders = directives.ConcurrentReaders
	DirRetryable         = directives.Retryable
	DirRetrySucceedsOn   = directives.RetrySucceedsOn
	DirDeleter           = directives.Deleter
)

// CompositionKind classifies a composition issue.
type CompositionKind int

const (
	// Conflict means two directives are contradictory (error).
	Conflict CompositionKind = iota
	// Redundant means one directive implies another (warning).
	Redundant
	// MissingRequired means a directive requires another that is absent (error).
	MissingRequired
)

// CompositionIssue describes a problem with directive combinations.
type CompositionIssue struct {
	DirectiveA string
	DirectiveB string
	Kind       CompositionKind
	Message    string
}

// ValidateComposition checks composition rules for a set of
// directives on a single method. Returns issues found — conflicts
// and missing-required are errors, redundancies are warnings.
func ValidateComposition(directives []gen.Directive) []CompositionIssue {
	names := make(map[string]bool, len(directives))
	for _, d := range directives {
		names[d.Name] = true
	}

	var issues []CompositionIssue

	// Conflicts — contradictory pairs.
	conflicts := [][3]string{
		{DirPure, DirSideEffect, "pure methods cannot have side effects"},
		{DirPure, DirMonotonic, "monotonic mutations are non-pure"},
		{DirCacheable, DirSideEffect, "cacheable methods cannot have side effects"},
		{DirCacheable, DirMonotonic, "cacheable methods cannot be monotonic"},
		{DirConcurrent, DirConcurrentReaders, "choose one concurrency model"},
	}
	for _, c := range conflicts {
		if names[c[0]] && names[c[1]] {
			issues = append(issues, CompositionIssue{
				DirectiveA: c[0],
				DirectiveB: c[1],
				Kind:       Conflict,
				Message:    c[2],
			})
		}
	}

	// Required pairs — one directive requires another.
	if names[DirRetrySucceedsOn] && !names[DirRetryable] {
		issues = append(issues, CompositionIssue{
			DirectiveA: DirRetrySucceedsOn,
			DirectiveB: DirRetryable,
			Kind:       MissingRequired,
			Message:    "retry-succeeds-on requires retryable",
		})
	}

	// Redundancies — one directive implies another.
	redundancies := [][3]string{
		{DirPure, DirIdempotent, "pure implies idempotent"},
		{DirCacheable, DirPure, "cacheable implies pure"},
		{DirCacheable, DirIdempotent, "cacheable implies idempotent"},
	}
	for _, r := range redundancies {
		if names[r[0]] && names[r[1]] {
			issues = append(issues, CompositionIssue{
				DirectiveA: r[0],
				DirectiveB: r[1],
				Kind:       Redundant,
				Message:    r[2],
			})
		}
	}

	return issues
}
