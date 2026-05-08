// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"fmt"
	"sort"
	"sync"
)

// Emission pairs a directive name with the generator name that emits
// assertion-suite output for it. The package-level [defaultEmitters]
// holds all registrations across all generators in the binary.
//
// Like [Consumer], Emission does NOT hold the emitter function
// value — each generator owns its own emitter map locally and
// dispatches via its own pipeline. The registry exists for
// documentation and audit.
type Emission struct {
	Directive string
	Generator string
}

// EmitterRegistry maps mixin directives to the generators that emit
// assertion-suite output for them. Parallels [ConsumerRegistry] but
// for the mixin pass rather than the enrichment pass.
//
// Multiple generators may register for the same directive — `bounded`
// typically has both a suite emitter (range-check assertion) and a
// bench emitter (range guard). Each (directive, generator) pair must
// be unique.
type EmitterRegistry struct {
	mu       sync.RWMutex
	byDir    map[string][]Emission
	byGenDir map[string]Emission
}

// NewEmitterRegistry returns an empty [EmitterRegistry].
func NewEmitterRegistry() *EmitterRegistry {
	return &EmitterRegistry{
		byDir:    make(map[string][]Emission),
		byGenDir: make(map[string]Emission),
	}
}

// Register adds an emitter registration. Returns an error if
// (generator, directive) is already registered.
func (r *EmitterRegistry) Register(e Emission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := e.Generator + ":" + e.Directive
	if _, exists := r.byGenDir[key]; exists {
		return fmt.Errorf("duplicate emitter: generator %q already emits for directive %q",
			e.Generator, e.Directive)
	}
	r.byGenDir[key] = e
	r.byDir[e.Directive] = append(r.byDir[e.Directive], e)
	return nil
}

// MustRegister registers an emitter and panics on duplicate.
func (r *EmitterRegistry) MustRegister(e Emission) {
	if err := r.Register(e); err != nil {
		panic(err.Error()) //nolint:forbidigo // init-time programmer error
	}
}

// Emitters returns all emitter registrations for a directive, sorted
// by generator name. Empty slice when the directive has no emitters.
func (r *EmitterRegistry) Emitters(directive string) []Emission {
	r.mu.RLock()
	defer r.mu.RUnlock()
	emissions := r.byDir[directive]
	out := make([]Emission, len(emissions))
	copy(out, emissions)
	sort.Slice(out, func(i, j int) bool { return out[i].Generator < out[j].Generator })
	return out
}

// Directives returns every directive name that has at least one
// registered emitter, sorted.
func (r *EmitterRegistry) Directives() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byDir))
	for k := range r.byDir {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// defaultEmitters is the package-level mixin registry. Generators
// register at init time.
var defaultEmitters = NewEmitterRegistry()

// RegisterEmitter registers a (directive, generator) pair in the
// default mixin registry. Generators call this from package init:
//
//	func init() {
//	    directive.RegisterEmitter(directive.Atomic, "suite")
//	    directive.RegisterEmitter(directive.Atomic, "bench")
//	}
//
// The actual emitter functions live in the generator's package and
// are dispatched by the generator's pipeline — the registry only
// records the relationship for doc-gen and audit.
func RegisterEmitter(dir, gen string) {
	defaultEmitters.MustRegister(Emission{
		Directive: dir,
		Generator: gen,
	})
}

// DefaultEmitters returns the package-level [EmitterRegistry].
func DefaultEmitters() *EmitterRegistry { return defaultEmitters }
