// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisher registers the //testkit:publisher consumer. The
// directive promotes a method to Publisher: every active subscriber
// receives every publication, with delivery semantics governed by
// the //testkit:delivery mixin.
//
// Directive form:
//
//	//testkit:publisher Subscribe
//
// Validation: the named subscribe method must exist on the same
// interface. Argument count: exactly one.
package publisher

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved subscribe-method name.
type Payload struct {
	// Subscribe is the sibling subscribe method. Validated to exist
	// on the same interface.
	Subscribe string
}

func init() {
	spec.RegisterConsumer(directive.Publisher, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:publisher directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Publisher)
}

// Has reports whether the method carries //testkit:publisher.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Publisher) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("publisher: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Publisher, Payload{Subscribe: want})
			return nil
		}
	}
	return fmt.Errorf("publisher: subscribe method %q not found on interface %s",
		want, data.Interface.Name)
}
