// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package allocs registers the //testkit:allocs consumer. The bench
// generator reads the resolved payload to emit an [bench.AllocsWithin]
// gate per method — a CI-deterministic ceiling on per-call allocation
// count.
//
// Directive form:
//
//	//testkit:allocs 0
//	//testkit:allocs 3
package allocs

import (
	"fmt"
	"strconv"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the parsed allocation budget.
type Payload struct {
	// Max is the inclusive ceiling on allocations per call. Zero is
	// allowed (asserts the method is alloc-free); negative values
	// are rejected by the consumer.
	Max int
}

func init() {
	spec.RegisterConsumer(directive.Allocs, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:allocs directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Allocs)
}

// Has reports whether the method carries //testkit:allocs.
func Has(m *spec.Method) bool {
	return spec.Has(m.Attachments, directive.Allocs)
}

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("allocs: %w", err)
	}
	n, err := strconv.Atoi(dir.Args[0])
	if err != nil || n < 0 {
		return fmt.Errorf("allocs: %q is not a non-negative integer", dir.Args[0])
	}
	spec.Set(&method.Attachments, directive.Allocs, Payload{Max: n})
	return nil
}
