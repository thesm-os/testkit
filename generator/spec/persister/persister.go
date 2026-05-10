// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package persister registers the //testkit:persister consumer. The
// directive promotes a Writer-with-result to Persister: the value the
// writer returns is the lookup key for the named sibling Reader.
//
// Directive form:
//
//	//testkit:persister Get
//
// Validation: the named Reader method must exist on the same
// interface. Argument count: exactly one.
package persister

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved Reader-method name.
type Payload struct {
	// Reader is the Reader sibling whose key type matches the
	// writer's returned ID. Validated to exist on the same interface.
	Reader string
}

func init() {
	spec.RegisterConsumer(directive.Persister, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:persister directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Persister)
}

// Has reports whether the method carries //testkit:persister.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Persister) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("persister: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Persister, Payload{Reader: want})
			return nil
		}
	}
	return fmt.Errorf("persister: reader method %q not found on interface %s",
		want, data.Interface.Name)
}
