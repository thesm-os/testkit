// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cas registers the //testkit:cas consumer. The directive
// promotes a Writer to CompareAndSwap: concurrent writers race; the
// one whose version matches the current value wins, the others see
// a version-mismatch error.
//
// Directive form:
//
//	//testkit:cas Version
//
// Validation: argument count is exactly one. The named field is not
// resolved against the input type at this layer — that resolution
// requires typed access to the parameter and lives in the model
// generator's analyze step. The consumer attaches the raw field
// name; downstream code carries the resolution.
package cas

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the version-bearing field name. Field-on-type
// resolution is deferred to the consumer of the payload.
type Payload struct {
	// VersionField is the name of the version/revision/etag field on
	// the input value. Used by codegen to extract the version for the
	// optimistic-concurrency comparison.
	VersionField string
}

func init() {
	spec.RegisterConsumer(directive.CAS, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:cas directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.CAS)
}

// Has reports whether the method carries //testkit:cas.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.CAS) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("cas: %w", err)
	}
	spec.Set(&method.Attachments, directive.CAS, Payload{VersionField: dir.Args[0]})
	return nil
}
