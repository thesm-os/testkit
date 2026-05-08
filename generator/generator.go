// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import "sort"

// Generator produces output files from Go type information. Each
// generator (stub, builder, sentinel, enum, suite, model, bench, …)
// implements this interface and registers with a [Registry].
//
// Generators are typically thin wrappers around a configured [Pipeline]
// — see the per-generator packages under generator/<name>/.
type Generator interface {
	// Name returns the subcommand name ("stub", "builder", etc.).
	// Names must be unique across the registry.
	Name() string

	// Generate produces output files for the given package and type
	// arguments. The args slice contains type names (e.g. ["Store"])
	// or is empty for generators that scan the whole package
	// (sentinel, enum without args).
	Generate(pkg *Package, args []string, cfg Config, opts Options) (*Result, error)
}

// Registry holds generators and provides lookup by name. The CLI uses
// this to dispatch subcommands to the correct generator.
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates an empty [Registry].
func NewRegistry() *Registry {
	return &Registry{generators: make(map[string]Generator)}
}

// Register adds a generator. Returns an error if a generator with the
// same name is already registered. Unlike the legacy gen.Registry, this
// does not panic — registration errors are returned so the CLI or tests
// can decide how to handle them.
func (r *Registry) Register(g Generator) error {
	name := g.Name()
	if _, exists := r.generators[name]; exists {
		return Errorf(noPos, "duplicate generator: %q", name)
	}
	r.generators[name] = g
	return nil
}

// MustRegister adds a generator and panics on duplicate. Use only in
// init() where a duplicate is a programmer error.
func (r *Registry) MustRegister(g Generator) {
	if err := r.Register(g); err != nil {
		panic(err.Error()) //nolint:forbidigo // init-time programmer error
	}
}

// Get returns the generator for the given name, or nil if not found.
func (r *Registry) Get(name string) Generator {
	return r.generators[name]
}

// Names returns all registered generator names in sorted order.
// Sort is deterministic for stable CLI help output.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Len returns the number of registered generators.
func (r *Registry) Len() int { return len(r.generators) }
