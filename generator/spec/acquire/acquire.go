// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package acquire registers the //testkit:acquire consumer. The
// directive promotes a method to AcquireLease: double-acquire is
// rejected, the named release method returns the lease.
//
// Directive form:
//
//	//testkit:acquire Release
//
// Validation: the named release method must exist on the same
// interface. Argument count: exactly one.
package acquire

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved release-method name.
type Payload struct {
	// Release is the sibling that returns the lease. Validated to
	// exist on the same interface.
	Release string
}

func init() {
	spec.RegisterConsumer(directive.Acquire, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:acquire directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Acquire)
}

// Has reports whether the method carries //testkit:acquire.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Acquire) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("acquire: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Acquire, Payload{Release: want})
			return nil
		}
	}
	return fmt.Errorf("acquire: release method %q not found on interface %s",
		want, data.Interface.Name)
}
