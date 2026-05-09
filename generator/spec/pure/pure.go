// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pure registers the //testkit:pure marker. The directive
// declares the method has no side effects and depends only on its
// inputs — same input on independent impls yields the same output.
// Templates check presence to emit the pure-contract subtest
// (independent impls, equal results).
package pure

import (
	"errors"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Presence is the empty payload type attached when pure fires.
type Presence struct{}

func init() {
	spec.RegisterConsumer(directive.Pure, consume)
}

// Has reports whether the method carries //testkit:pure.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Pure) }

// consume validates that the method has at least one non-error
// result. The pure contract compares results across independent
// impls; void methods can't be compared.
func consume(method *spec.Method, _ directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if !method.HasNonErrorResults() {
		return errors.New("pure: method must return a non-error result to compare across impls")
	}
	spec.Set(&method.Attachments, directive.Pure, Presence{})
	return nil
}
