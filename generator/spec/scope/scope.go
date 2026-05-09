// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package scope registers the //testkit:scope consumer. The
// directive declares the method requires authorization for the
// named scope; the contract subtest exercises both an
// unauthorized-context call (asserts the configured sentinel) and
// an authorized-context call (asserts success).
//
// Directive form:
//
//	//testkit:scope billing.read
//
// The scope name is recorded verbatim. The contract subtest pulls
// `WithScopeContext` and `WithScopeUnauthorized` from the resolved
// suite Options at runtime.
package scope

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the required scope name.
type Payload struct {
	// Name is the verbatim scope arg.
	Name string
}

func init() {
	spec.RegisterConsumer(directive.Scope, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:scope directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Scope)
}

// Has reports whether the method carries //testkit:scope.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Scope) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("scope: %w", err)
	}
	spec.Set(&method.Attachments, directive.Scope, Payload{Name: dir.Args[0]})
	return nil
}
