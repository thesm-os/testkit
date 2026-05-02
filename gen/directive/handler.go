// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package directive provides the known-directive registry and
// composition validation for testkit generators. Actual directive
// processing is done by enricher functions in each generator package
// (gen/stub/, gen/suite/, etc.) — this package handles validation
// and documentation concerns.
package directive

import (
	"sort"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// Descriptor describes a known directive for validation and
// documentation. Unlike the previous Handler interface, descriptors
// do not process directives — enricher functions in each generator
// package handle that.
type Descriptor struct {
	// Name is the directive name ("errors", "concurrent", etc.).
	Name string

	// Description is a one-line summary for documentation.
	Description string

	// Args describes expected arguments ("ErrName [ErrName...]" or
	// "" for no-arg directives).
	Args string

	// Generators lists which generators consume this directive.
	Generators []string

	// Phase is the implementation phase (1-6) from the spec.
	Phase int
}

// Registry holds directive descriptors and provides validation.
type Registry struct {
	descriptors map[string]Descriptor
}

// NewRegistry creates an empty [Registry].
func NewRegistry() *Registry {
	return &Registry{descriptors: make(map[string]Descriptor)}
}

// Register adds a directive descriptor. Panics if a descriptor with
// the same name is already registered.
func (r *Registry) Register(d Descriptor) {
	if _, exists := r.descriptors[d.Name]; exists {
		panic("directive: duplicate descriptor: " + d.Name) //nolint:forbidigo
	}
	r.descriptors[d.Name] = d
}

// Get returns the descriptor for the given name and true, or a zero
// Descriptor and false if not found.
func (r *Registry) Get(name string) (Descriptor, bool) {
	d, ok := r.descriptors[name]
	return d, ok
}

// IsKnown reports whether the directive name is registered.
func (r *Registry) IsKnown(name string) bool {
	_, ok := r.descriptors[name]
	return ok
}

// Names returns all registered directive names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.descriptors))
	for name := range r.descriptors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Validate checks that all directives on the given methods are known.
// Unknown directives are errors (strict-by-default). Directives with
// the "experimental:" prefix produce warnings via the warn callback
// instead of errors.
func (r *Registry) Validate(methods []gen.MethodInfo, warn func(string)) []error {
	var errs []error
	for _, m := range methods {
		for _, d := range m.Directives {
			if strings.HasPrefix(d.Name, "experimental:") {
				if warn != nil {
					warn("experimental directive " + d.Name + " on " + m.Name)
				}
				continue
			}
			if !r.IsKnown(d.Name) {
				errs = append(errs, gen.Errorf(m.Pos,
					"unknown directive %q on method %s", d.Name, m.Name,
				))
			}
		}
	}
	return errs
}
