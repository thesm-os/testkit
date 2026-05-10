// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package watcher registers the //testkit:watcher consumer. The
// directive promotes a method to Watcher: the named sibling triggers
// the watcher's notification.
//
// Directive form:
//
//	//testkit:watcher Set
//
// Validation: the named trigger method must exist on the same
// interface. Argument count: exactly one.
package watcher

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved trigger-method name.
type Payload struct {
	// Trigger is the sibling method whose calls cause the watcher to
	// deliver. Validated to exist on the same interface.
	Trigger string
}

func init() {
	spec.RegisterConsumer(directive.Watcher, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:watcher directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Watcher)
}

// Has reports whether the method carries //testkit:watcher.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Watcher) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("watcher: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Watcher, Payload{Trigger: want})
			return nil
		}
	}
	return fmt.Errorf("watcher: trigger method %q not found on interface %s",
		want, data.Interface.Name)
}
