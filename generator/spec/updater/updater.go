// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package updater registers the //testkit:updater consumer. The
// directive promotes a Writer or CompositeWriter to Updater: the
// writer replaces an existing entry by key, and the named sibling
// Reader looks the entry up.
//
// Directive form:
//
//	//testkit:updater Get
//
// Validation: the named Reader method must exist on the same
// interface. Argument count: exactly one.
package updater

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved Reader-method name.
type Payload struct {
	// Reader is the Reader sibling whose key matches the writer's
	// key argument. Validated to exist on the same interface.
	Reader string
}

func init() {
	spec.RegisterConsumer(directive.Updater, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:updater directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Updater)
}

// Has reports whether the method carries //testkit:updater.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Updater) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("updater: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Updater, Payload{Reader: want})
			return nil
		}
	}
	return fmt.Errorf("updater: reader method %q not found on interface %s",
		want, data.Interface.Name)
}
