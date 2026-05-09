// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package validates registers the //testkit:validates consumer.
// The directive declares the method validates input on the named
// field and returns a validation error for zero/invalid values.
// The contract subtest calls with [m.ZeroArgs] and asserts a
// non-nil error return.
//
// Directive form:
//
//	//testkit:validates ID
//
// The field name is recorded verbatim; runtime resolution against
// the method's parameter or struct-field shape is the contract's
// responsibility, not the consumer's.
package validates

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the field name being validated.
type Payload struct {
	// Field is the bare name of the validated field (parameter name
	// or struct-field name, depending on the impl's signature).
	Field string
}

func init() {
	spec.RegisterConsumer(directive.Validates, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:validates directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Validates)
}

// Has reports whether the method carries //testkit:validates.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Validates) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("validates: %w", err)
	}
	if !method.ReturnsError() {
		return fmt.Errorf("validates: method %q must return an error to surface validation failures",
			method.Name)
	}
	spec.Set(&method.Attachments, directive.Validates, Payload{Field: dir.Args[0]})
	return nil
}
