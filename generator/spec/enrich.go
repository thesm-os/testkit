// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"sync"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

// Consumer is the signature every directive consumer registers
// against. The consumer reads the directive and writes its payload
// into [Method.Attachments] via [Set]. Returning a non-nil error
// halts the enrichment pass with the canonical position-bearing
// wrapping applied by [Enrich].
//
// Consumers must be deterministic and side-effect-free outside of
// the Method they're handed — enrichment runs in a single pass and
// re-running it must produce identical results.
type Consumer func(method *Method, dir directive.Directive, data *Data, pkg *generator.Package) error

// consumerRegistry is the package-level dispatch table. Keys are
// directive names; values are the consumers that should fire when
// the directive appears on a method. Multiple consumers may
// register for the same directive (rare but supported — e.g. an
// audit consumer that fires alongside the canonical enricher).
var (
	consumerMu       sync.RWMutex
	consumerRegistry = make(map[string][]Consumer)
)

// RegisterConsumer registers c to fire when [Enrich] encounters a
// directive named directiveName on any method. Call this from the
// consumer package's init():
//
//	func init() {
//	    spec.RegisterConsumer(directive.Sample, sampleConsumer)
//	}
//
// Multiple consumers may register for the same directive; they
// dispatch in registration order.
func RegisterConsumer(directiveName string, c Consumer) {
	consumerMu.Lock()
	defer consumerMu.Unlock()
	consumerRegistry[directiveName] = append(consumerRegistry[directiveName], c)
}

// Consumers returns the registered consumers for directiveName, in
// registration order. The returned slice is a copy; mutating it
// does not affect the registry. Useful for tests asserting which
// directives have consumers wired.
func Consumers(directiveName string) []Consumer {
	consumerMu.RLock()
	defer consumerMu.RUnlock()
	out := make([]Consumer, len(consumerRegistry[directiveName]))
	copy(out, consumerRegistry[directiveName])
	return out
}

// Enrich walks every directive on every method and dispatches to
// the registered consumers. Each consumer mutates the method's
// [Attachments] map.
//
// Generators call this from their own Enrich function:
//
//	func Enrich(data *Data, pkg *generator.Package) error {
//	    if err := spec.Enrich(data.Data, pkg); err != nil {
//	        return err
//	    }
//	    // ... generator-specific enrichment that touches data's wrapper fields
//	    return nil
//	}
//
// Errors are wrapped with the method's position so the diagnostic
// points at the offending directive call site.
func Enrich(data *Data, pkg *generator.Package) error {
	consumerMu.RLock()
	snapshot := make(map[string][]Consumer, len(consumerRegistry))
	for k, v := range consumerRegistry {
		copied := make([]Consumer, len(v))
		copy(copied, v)
		snapshot[k] = copied
	}
	consumerMu.RUnlock()

	for i := range data.Methods {
		m := &data.Methods[i]
		for _, d := range m.Directives {
			for _, c := range snapshot[d.Name] {
				if err := c(m, d, data, pkg); err != nil {
					return generator.WrapErr(m.Pos, err,
						"directive %s on %s.%s", d.Name, data.Interface.Name, m.Name)
				}
			}
		}
	}
	return nil
}
