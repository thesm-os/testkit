// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pool registers the //testkit:pool consumer. The directive
// promotes a Get method to the Pool composite-tier shape: every Get
// balances with a matching Put, the pool is leak-free across cycles,
// double-Put is rejected.
//
// Directive form:
//
//	//testkit:pool Put
//
// Validation: the named put method must exist on the same interface.
// Argument count: exactly one.
package pool

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved put-method name.
type Payload struct {
	// Put is the sibling that returns the pooled resource. Validated
	// to exist on the same interface.
	Put string
}

func init() {
	spec.RegisterConsumer(directive.Pool, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:pool directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Pool)
}

// Has reports whether the method carries //testkit:pool.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Pool) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("pool: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Pool, Payload{Put: want})
			return nil
		}
	}
	return fmt.Errorf("pool: put method %q not found on interface %s",
		want, data.Interface.Name)
}
