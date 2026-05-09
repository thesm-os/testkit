// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cacheable registers the //testkit:cacheable marker.
// Cacheable implies pure (handled by the directive registry's
// Implies chain) plus repeated identical responses to repeated
// identical queries. Templates emit a cache-hit subtest verifying
// three sequential calls return equal results.
package cacheable

import (
	"errors"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Presence is the empty payload type attached when cacheable fires.
type Presence struct{}

func init() {
	spec.RegisterConsumer(directive.Cacheable, consume)
}

// Has reports whether the method carries //testkit:cacheable.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Cacheable) }

// consume validates the method has a non-error result. The
// cacheable contract compares three sequential results; void
// methods have nothing to compare.
func consume(method *spec.Method, _ directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if !method.HasNonErrorResults() {
		return errors.New("cacheable: method must return a non-error result to compare across calls")
	}
	spec.Set(&method.Attachments, directive.Cacheable, Presence{})
	return nil
}
