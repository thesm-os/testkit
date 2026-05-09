// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timeout registers the //testkit:timeout consumer. The
// directive declares the method must complete within the named
// duration; the contract subtest spawns the call in a goroutine
// and selects on either completion or [time.After] firing.
//
// Directive form:
//
//	//testkit:timeout 100ms
//
// The duration string is stored verbatim in [Payload.Duration] and
// emitted into the generated assertion via `time.After({{ duration
// }})`. The directive registry validates ArgDuration form at parse
// time; this consumer trusts the parsed string.
package timeout

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the rendered duration string.
type Payload struct {
	// Duration is the verbatim arg ("100ms", "1s", "500us") — emitted
	// directly into the generated `time.After(...)` call.
	Duration string
}

func init() {
	spec.RegisterConsumer(directive.Timeout, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:timeout directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Timeout)
}

// Has reports whether the method carries //testkit:timeout.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Timeout) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}
	spec.Set(&method.Attachments, directive.Timeout, Payload{Duration: dir.Args[0]})
	return nil
}
