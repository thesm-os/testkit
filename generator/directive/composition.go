// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"fmt"
	"go/token"
)

// IssueKind classifies a composition violation.
type IssueKind int

// Issue kinds.
const (
	// Conflict: two directives that contradict each other (pure +
	// sideeffect, concurrent + concurrent-readers, ...). Hard error.
	Conflict IssueKind = iota + 1

	// MissingRequired: one directive requires another to be present
	// (retry-succeeds-on-attempt requires retryable). Hard error.
	MissingRequired

	// Redundant: one directive implies another (cacheable implies
	// pure). Warning, not error — codegen continues.
	Redundant
)

// Issue describes a single composition violation.
type Issue struct {
	Kind       IssueKind
	DirectiveA string
	DirectiveB string
	Message    string
}

// ValidateComposition checks the directives on a single method against
// the composition rules carried by each [Descriptor]'s metadata
// (Conflicts, Requires, Implies). Conflict and MissingRequired are
// returned as errors via the pipeline's CompositionValidator hook;
// Redundant issues live alongside but the caller decides whether to
// log or surface them (typically logged as warnings).
//
// Returns nil when no hard issues, the first Conflict/MissingRequired
// as a positioned error otherwise. Use [Issues] for the full slice
// including redundancies.
func (r *Registry) ValidateComposition(directives []Directive, pos token.Position) error {
	for _, issue := range r.Issues(directives) {
		if issue.Kind == Conflict || issue.Kind == MissingRequired {
			return fmt.Errorf("%s: %s", pos, issue.Message)
		}
	}
	return nil
}

// Issues returns every composition violation among the given
// directives, including redundancies. Order is deterministic: walks
// directives in source order, then emits the descriptor's
// Conflicts/Requires/Implies edges in declaration order.
//
// Each Conflict pair is reported once, regardless of which side
// declared it (either directive's descriptor may carry the entry).
func (r *Registry) Issues(directives []Directive) []Issue {
	present := make(map[string]bool, len(directives))
	for _, d := range directives {
		present[d.Name] = true
	}

	var out []Issue
	seenConflict := make(map[string]bool)
	for _, d := range directives {
		desc, ok := r.descriptors[d.Name]
		if !ok {
			continue
		}
		for _, conflict := range desc.Conflicts {
			if !present[conflict] {
				continue
			}
			key := pairKey(d.Name, conflict)
			if seenConflict[key] {
				continue
			}
			seenConflict[key] = true
			out = append(out, Issue{
				Kind:       Conflict,
				DirectiveA: d.Name,
				DirectiveB: conflict,
				Message:    d.Name + " conflicts with " + conflict,
			})
		}
		for _, required := range desc.Requires {
			if present[required] {
				continue
			}
			out = append(out, Issue{
				Kind:       MissingRequired,
				DirectiveA: d.Name,
				DirectiveB: required,
				Message:    d.Name + " requires " + required,
			})
		}
		for _, implied := range desc.Implies {
			if !present[implied] {
				continue
			}
			out = append(out, Issue{
				Kind:       Redundant,
				DirectiveA: d.Name,
				DirectiveB: implied,
				Message:    d.Name + " implies " + implied + " — the second is redundant",
			})
		}
	}
	return out
}

// pairKey returns a canonical key for a pair of directive names so
// (a, b) and (b, a) collide in the seen map.
func pairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

// ValidateComposition is the package-level convenience that consults
// [DefaultRegistry]. Pipelines that use the default registry can call
// this directly; tests that want isolation use the method form on a
// custom [*Registry].
func ValidateComposition(directives []Directive, pos token.Position) error {
	return DefaultRegistry().ValidateComposition(directives, pos)
}

// Issues is the package-level convenience for [DefaultRegistry.Issues].
func Issues(directives []Directive) []Issue {
	return DefaultRegistry().Issues(directives)
}
