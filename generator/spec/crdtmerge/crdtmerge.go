// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package crdtmerge registers the //testkit:crdt-merge consumer.
// The directive sits on a merge method and names the paired merge
// counterpart; the contract subtest applies operations in opposite
// orders on two impls and asserts both converge to equal state
// (commutative + associative + idempotent merge).
//
// Directive form (on a merge method):
//
//	//testkit:crdt-merge Other=Merge
//	Merge(ctx context.Context, other State) error
//
// Validation: the named other-method must exist on the same
// interface (typically the same name as the carrier method when
// both impls expose merge with a symmetric signature).
package crdtmerge

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved counterpart method name.
type Payload struct {
	// Other is the name of the paired merge method on the same
	// interface.
	Other string
}

func init() {
	spec.RegisterConsumer(directive.CRDTMerge, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:crdt-merge directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.CRDTMerge)
}

// Has reports whether the method carries //testkit:crdt-merge.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.CRDTMerge) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("crdt-merge: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.CRDTMerge, Payload{Other: want})
			return nil
		}
	}
	return fmt.Errorf("crdt-merge: counterpart method %q not found on interface %s",
		want, data.Interface.Name)
}
