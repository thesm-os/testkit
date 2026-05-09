// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lease registers the //testkit:lease consumer. The
// directive declares the method acquires a resource that must be
// released through the named release method. The contract subtest
// verifies acquire-release-acquire works (resource freed) and
// acquire-acquire fails (resource leased; cannot be re-acquired).
//
// Directive form:
//
//	//testkit:lease Release
//
// Validation: the named release method must exist on the same
// interface.
package lease

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the paired release-method name.
type Payload struct {
	// Release is the name of the release method on the same
	// interface. Validated to exist by the consumer.
	Release string
}

func init() {
	spec.RegisterConsumer(directive.Lease, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:lease directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Lease)
}

// Has reports whether the method carries //testkit:lease.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Lease) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("lease: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.Lease, Payload{Release: want})
			return nil
		}
	}
	return fmt.Errorf("lease: release method %q not found on interface %s",
		want, data.Interface.Name)
}
