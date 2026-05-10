// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package upserter registers the //testkit:upserter consumer. The
// directive promotes a Writer or CompositeWriter to Upserter: an
// idempotent insert-or-update where repeated calls with the same
// input produce the same observable state.
//
// Directive form:
//
//	//testkit:upserter Get
//
// Validation: the named Reader method must exist on the same
// interface. Argument count: exactly one.
package upserter

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved Reader-method name.
type Payload struct {
	// Reader is the Reader sibling that observes the upsert. Validated
	// to exist on the same interface.
	Reader string
}

func init() {
	spec.RegisterConsumer(directive.Upserter, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:upserter directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Upserter)
}

// Has reports whether the method carries //testkit:upserter.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Upserter) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("upserter: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Upserter, Payload{Reader: want})
			return nil
		}
	}
	return fmt.Errorf("upserter: reader method %q not found on interface %s",
		want, data.Interface.Name)
}
