// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"fmt"
	"sort"
	"sync"
)

// Consumer pairs a directive name with a single generator name. The
// package-level [defaultConsumers] holds all registered pairs in the
// binary.
//
// The consumer record does NOT hold the enricher function value —
// each generator owns its own enricher map locally and dispatches via
// its own pipeline. The registry exists for documentation and spec
// headers ("which generators consume directive X").
type Consumer struct {
	Directive string
	Generator string
}

// ConsumerRegistry maps directives to the generators that consume
// them. Used for:
//
//   - documentation: spec headers list "directives present, consumers"
//   - validation: when a directive has no consumer, the registry can
//     warn (or fail, depending on policy) instead of silently ignoring
//     it.
//
// The registry is goroutine-safe; init-time registration writes,
// generator runtime queries reads.
type ConsumerRegistry struct {
	mu       sync.RWMutex
	byDir    map[string][]Consumer // directive name → consumers
	byGenDir map[string]Consumer   // "<gen>:<dir>" → consumer (uniqueness)
}

// NewConsumerRegistry returns an empty [ConsumerRegistry].
func NewConsumerRegistry() *ConsumerRegistry {
	return &ConsumerRegistry{
		byDir:    make(map[string][]Consumer),
		byGenDir: make(map[string]Consumer),
	}
}

// Register adds a consumer. Returns an error if (generator, directive)
// is already registered.
func (r *ConsumerRegistry) Register(c Consumer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := c.Generator + ":" + c.Directive
	if _, exists := r.byGenDir[key]; exists {
		return fmt.Errorf("duplicate consumer: generator %q already consumes directive %q",
			c.Generator, c.Directive)
	}
	r.byGenDir[key] = c
	r.byDir[c.Directive] = append(r.byDir[c.Directive], c)
	return nil
}

// MustRegister registers a consumer and panics on duplicate.
func (r *ConsumerRegistry) MustRegister(c Consumer) {
	if err := r.Register(c); err != nil {
		panic(err.Error()) //nolint:forbidigo // init-time programmer error
	}
}

// Consumers returns the consumers for a directive name, sorted by
// generator name. Empty slice when the directive has no registered
// consumers.
func (r *ConsumerRegistry) Consumers(directive string) []Consumer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cs := r.byDir[directive]
	out := make([]Consumer, len(cs))
	copy(out, cs)
	sort.Slice(out, func(i, j int) bool { return out[i].Generator < out[j].Generator })
	return out
}

// GeneratorsFor returns the names of generators that consume the given
// directive, sorted alphabetically. Used by spec/header to render the
// truthful "Directives:" line.
func (r *ConsumerRegistry) GeneratorsFor(directive string) []string {
	cs := r.Consumers(directive)
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Generator
	}
	return out
}

// Directives returns all directive names that have at least one
// consumer registered, sorted.
func (r *ConsumerRegistry) Directives() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byDir))
	for k := range r.byDir {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// defaultConsumers is the package-level registry that init-time
// generator code populates. Tests that need isolation construct their
// own registry.
var defaultConsumers = NewConsumerRegistry()

// RegisterConsumer registers a (directive, generator) pair in the
// default registry. Generators call this from their package init()
// functions to declare which directives they consume:
//
//	func init() {
//	    directive.RegisterConsumer(directive.Errors, "stub")
//	    directive.RegisterConsumer(directive.IntegrationOnly, "stub")
//	}
//
// The actual enricher functions live in the generator's package and
// are dispatched by the generator's pipeline — the registry only
// records the relationship for doc-gen and audit.
func RegisterConsumer(dir, gen string) {
	defaultConsumers.MustRegister(Consumer{
		Directive: dir,
		Generator: gen,
	})
}

// DefaultConsumers returns the package-level [ConsumerRegistry].
func DefaultConsumers() *ConsumerRegistry { return defaultConsumers }
