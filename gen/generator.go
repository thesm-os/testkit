// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import "sort"

// Generator produces output files from Go type information. Each
// generator (stub, builder, sentinel, enum, suite, model, etc.)
// implements this interface and registers with the [Registry].
type Generator interface {
	// Name returns the subcommand name ("stub", "builder", etc.).
	Name() string

	// Generate produces output files for the given package and type
	// arguments. The args slice contains type names (e.g. ["Store"])
	// or is empty for generators that scan the whole package (sentinel).
	Generate(pkg *Package, args []string, cfg Config, opts Options) (*Result, error)
}

// Registry holds generators and provides lookup by name. The CLI
// uses this to dispatch subcommands to the correct generator.
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates an empty [Registry].
func NewRegistry() *Registry {
	return &Registry{generators: make(map[string]Generator)}
}

// Register adds a generator. Panics if a generator with the same
// name is already registered.
func (r *Registry) Register(g Generator) {
	name := g.Name()
	if _, exists := r.generators[name]; exists {
		panic("gen: duplicate generator: " + name) //nolint:forbidigo
	}
	r.generators[name] = g
}

// Get returns the generator for the given name, or nil if not found.
func (r *Registry) Get(name string) Generator {
	return r.generators[name]
}

// Names returns all registered generator names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
