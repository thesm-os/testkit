// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bounded registers the //testkit:bounded consumer. The
// directive declares the method's first non-error result is in a
// closed range [min, max]; the contract subtest calls the method
// once and asserts the result is within range.
//
// Directive form:
//
//	//testkit:bounded 0..100
//
// `min..max` is parsed into separate Min/Max strings emitted
// directly into the generated assertion. Both bounds are stored
// verbatim — no numeric parsing here; the consumer trusts the
// directive registry's ArgRange validator.
package bounded

import (
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the parsed [Min, Max] literals.
type Payload struct {
	// Min is the verbatim lower bound from the `min..max` arg.
	Min string

	// Max is the verbatim upper bound.
	Max string
}

func init() {
	spec.RegisterConsumer(directive.Bounded, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:bounded directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Bounded)
}

// Has reports whether the method carries //testkit:bounded.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Bounded) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("bounded: %w", err)
	}
	lower, upper, ok := strings.Cut(dir.Args[0], "..")
	if !ok {
		return fmt.Errorf("bounded: arg %q must be in `min..max` form", dir.Args[0])
	}
	lower, upper = strings.TrimSpace(lower), strings.TrimSpace(upper)
	if lower == "" || upper == "" {
		return fmt.Errorf("bounded: arg %q must have both bounds (min..max)", dir.Args[0])
	}
	spec.Set(&method.Attachments, directive.Bounded, Payload{Min: lower, Max: upper})
	return nil
}
