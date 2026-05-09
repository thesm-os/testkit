// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package eventually registers the //testkit:eventually consumer.
// The directive declares the method's result converges to a stable
// value within the named timeout; the contract subtest polls
// until two consecutive calls return equal values, failing if the
// deadline expires before convergence.
//
// Directive form:
//
//	//testkit:eventually 500ms
//
// The duration string is stored verbatim and emitted into
// `time.Now().Add({{ duration }})` in the generated contract.
package eventually

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the rendered convergence-deadline duration.
type Payload struct {
	// Duration is the verbatim arg ("500ms", "1s") emitted directly
	// into the generated `time.After(...)` / `time.Now().Add(...)`
	// calls.
	Duration string
}

func init() {
	spec.RegisterConsumer(directive.Eventually, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:eventually directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Eventually)
}

// Has reports whether the method carries //testkit:eventually.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Eventually) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("eventually: %w", err)
	}
	spec.Set(&method.Attachments, directive.Eventually, Payload{Duration: dir.Args[0]})
	return nil
}
