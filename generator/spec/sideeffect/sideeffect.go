// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sideeffect registers the //testkit:sideeffect consumer.
// The directive declares that calling the annotated method changes
// state observable through the named paired method (typically a
// reader); the contract subtest reads via that method before and
// after, asserts they differ.
//
// Directive form:
//
//	//testkit:sideeffect Get
//
// Validation: the named method must exist on the same interface.
package sideeffect

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved observation-method name.
type Payload struct {
	// Method is the name of the paired observation method on the same
	// interface. Validated to exist by the consumer.
	Method string
}

func init() {
	spec.RegisterConsumer(directive.SideEffect, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:sideeffect directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.SideEffect)
}

// Has reports whether the method carries //testkit:sideeffect.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.SideEffect) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("sideeffect: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.SideEffect, Payload{Method: want})
			return nil
		}
	}
	return fmt.Errorf("sideeffect: method %q not found on interface %s",
		want, data.Interface.Name)
}
