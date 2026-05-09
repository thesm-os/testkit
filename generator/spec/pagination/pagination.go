// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pagination registers the //testkit:pagination consumer.
// The directive declares the method emits cursor-based paginated
// results — the contract subtest follows the cursor through the
// impl, collects items in a set, asserts no duplicates, and caps
// iteration at a runaway-safety limit.
//
// Directive form:
//
//	//testkit:pagination NextPage
//
// `NextPage` names the cursor field on the result type. Reflection
// at runtime extracts it from the page struct; the consumer just
// records the name verbatim.
package pagination

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the cursor field name.
type Payload struct {
	// CursorField is the verbatim arg — the name of the cursor field
	// on the result type (looked up via reflection at contract time).
	CursorField string
}

func init() {
	spec.RegisterConsumer(directive.Pagination, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:pagination directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Pagination)
}

// Has reports whether the method carries //testkit:pagination.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Pagination) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("pagination: %w", err)
	}
	spec.Set(&method.Attachments, directive.Pagination, Payload{CursorField: dir.Args[0]})
	return nil
}
