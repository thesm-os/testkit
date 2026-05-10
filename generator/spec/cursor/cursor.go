// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cursor registers the //testkit:cursor consumer. The
// directive promotes a Next method to the Cursor composite-tier
// shape: Next yields each element exactly once until exhaustion;
// Close is idempotent; Next-after-Close returns the not-found
// sentinel.
//
// Directive form:
//
//	//testkit:cursor Close
//
// Validation: the named close method must exist on the same
// interface. Argument count: exactly one.
package cursor

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved close-method name.
type Payload struct {
	// Close is the sibling finisher. Validated to exist on the same
	// interface.
	Close string
}

func init() {
	spec.RegisterConsumer(directive.Cursor, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:cursor directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Cursor)
}

// Has reports whether the method carries //testkit:cursor.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Cursor) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("cursor: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Cursor, Payload{Close: want})
			return nil
		}
	}
	return fmt.Errorf("cursor: close method %q not found on interface %s",
		want, data.Interface.Name)
}
