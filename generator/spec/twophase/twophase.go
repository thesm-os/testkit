// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package twophase registers the //testkit:two-phase consumer. The
// directive promotes the entry method (typically Begin) to the
// TwoPhase composite-tier shape: mutex of Commit-or-Rollback (one
// wins), and Rollback after Commit is rejected.
//
// Directive form:
//
//	//testkit:two-phase Commit Rollback
//
// Validation: both named methods must exist on the same interface.
// Argument count: exactly two.
package twophase

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved commit/rollback method names.
type Payload struct {
	// Commit is the sibling that finalizes the transaction.
	Commit string

	// Rollback is the sibling that aborts the transaction.
	Rollback string
}

func init() {
	spec.RegisterConsumer(directive.TwoPhase, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:two-phase directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.TwoPhase)
}

// Has reports whether the method carries //testkit:two-phase.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.TwoPhase) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 2); err != nil {
		return fmt.Errorf("two-phase: %w", err)
	}
	commit, rollback := dir.Args[0], dir.Args[1]
	if !methodExists(data, commit) {
		return fmt.Errorf("two-phase: commit method %q not found on interface %s",
			commit, data.Interface.Name)
	}
	if !methodExists(data, rollback) {
		return fmt.Errorf("two-phase: rollback method %q not found on interface %s",
			rollback, data.Interface.Name)
	}
	spec.Set(&method.Attachments, directive.TwoPhase, Payload{Commit: commit, Rollback: rollback})
	return nil
}

func methodExists(data *spec.Data, name string) bool {
	for _, m := range data.Interface.Methods {
		if m.Name == name {
			return true
		}
	}
	return false
}
