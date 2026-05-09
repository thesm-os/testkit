// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package latency registers the //testkit:latency consumer. The
// bench generator reads the resolved payload to emit a
// [bench.LatencyWithin] gate per method — a per-call latency ceiling
// suitable for CI when paired with a fixed -benchtime.
//
// Directive form:
//
//	//testkit:latency 100us
//	//testkit:latency 5ms
package latency

import (
	"fmt"
	"time"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the parsed latency budget.
type Payload struct {
	// Max is the inclusive ceiling on per-call latency. Always
	// positive — the consumer rejects zero and negative durations.
	Max time.Duration

	// Raw is the original directive arg (e.g. "100us"). Templates
	// emit this as the rendered Go literal — round-trips Duration
	// values stably without locale-sensitive formatting.
	Raw string
}

func init() {
	spec.RegisterConsumer(directive.Latency, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:latency directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Latency)
}

// Has reports whether the method carries //testkit:latency.
func Has(m *spec.Method) bool {
	return spec.Has(m.Attachments, directive.Latency)
}

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("latency: %w", err)
	}
	d, err := time.ParseDuration(dir.Args[0])
	if err != nil {
		return fmt.Errorf("latency: %q: %w", dir.Args[0], err)
	}
	if d <= 0 {
		return fmt.Errorf("latency: %q must be positive", dir.Args[0])
	}
	spec.Set(&method.Attachments, directive.Latency, Payload{Max: d, Raw: dir.Args[0]})
	return nil
}
