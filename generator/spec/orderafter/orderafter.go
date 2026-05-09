// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package orderafter registers the //testkit:order-after consumer.
// The directive declares that the annotated method must be called
// after the named prerequisite method; the stub enforces the
// constraint at runtime, surfacing call-order bugs in the consumer.
//
// Directive form:
//
//	//testkit:order-after Open
//
// Validation: the named method must exist on the same interface.
// Cross-interface ordering is a separate concern (cookbook-driven)
// and not modeled here.
package orderafter

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the prerequisite method name.
type Payload struct {
	// Method is the name of the prerequisite method on the same
	// interface. Validated to exist by the consumer.
	Method string
}

func init() {
	spec.RegisterConsumer(directive.OrderAfter, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:order-after directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.OrderAfter)
}

// Has reports whether the method has an order-after constraint.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.OrderAfter) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("order-after: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.OrderAfter, Payload{Method: want})
			return nil
		}
	}
	return fmt.Errorf("order-after: method %q not found on interface %s",
		want, data.Interface.Name)
}
