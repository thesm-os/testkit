// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package monotonic registers the //testkit:monotonic marker.
// Monotonic means repeated reads return non-decreasing values —
// applies to Aggregator-shape methods whose result type satisfies
// [cmp.Ordered]. Templates emit a non-decreasing subtest; the
// directive validator (Task 18) rejects monotonic on non-ordered
// result types.
package monotonic

import (
	"errors"
	"fmt"
	"go/types"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Presence is the empty payload type attached when monotonic fires.
type Presence struct{}

func init() {
	spec.RegisterConsumer(directive.Monotonic, consume)
}

// Has reports whether the method carries //testkit:monotonic.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Monotonic) }

// consume validates that the method's first non-error result type
// satisfies the monotonic-comparison contract: it must be an
// integer, float, or string (the [cmp.Ordered] constraint set).
// Methods with no non-error result, or with a non-ordered first
// result, are rejected — the contract isn't expressible.
func consume(method *spec.Method, _ directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if !method.HasNonErrorResults() {
		return errors.New("monotonic: method must return a non-error result to compare across calls")
	}
	results := method.Signature.Results()
	first := results.At(0).Type()
	if !isOrdered(first) {
		return fmt.Errorf("monotonic: first result type %s does not satisfy cmp.Ordered (need integer, float, or string)", first)
	}
	spec.Set(&method.Attachments, directive.Monotonic, Presence{})
	return nil
}

// isOrdered reports whether t's underlying type satisfies the
// [cmp.Ordered] constraint — Go's built-in ordered set: integer
// kinds, float kinds, and string. Pointers, interfaces, slices,
// maps, structs, and named types whose underlying isn't an ordered
// basic type are rejected.
func isOrdered(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch b.Kind() {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
		types.Float32, types.Float64, types.String:
		return true
	default:
		return false
	}
}
