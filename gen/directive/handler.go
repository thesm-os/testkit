// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package directive provides the pluggable directive system for testkit
// generators. Each directive (errors, concurrent, allocs, etc.) is
// implemented as a [Handler] that processes //testkit: annotations and
// returns template extension blocks for generators to inject.
package directive

import "sort"

// Handler processes a //testkit: directive for generators. Each handler
// is registered once and called per method that carries the directive.
type Handler interface {
	// Name returns the directive name ("errors", "concurrent", etc.).
	Name() string

	// Process is called for each method carrying this directive.
	// It returns blocks to inject at template extension points.
	Process(ctx Context) (*Output, error)
}

// Context provides the information a handler needs to process a directive.
type Context struct {
	// Args are the parsed directive arguments.
	// e.g. for "//testkit:errors ErrNotFound ErrConflict" → ["ErrNotFound", "ErrConflict"]
	Args []string

	// Generator is the name of the generator consuming this directive
	// ("stub", "suite", "model"). Handlers can produce different output
	// per generator.
	Generator string

	// MethodName is the interface method this directive is attached to.
	MethodName string

	// InterfaceName is the interface this method belongs to.
	InterfaceName string
}

// Output is what a handler returns — blocks to inject at named
// extension points in generator templates.
type Output struct {
	Blocks []Block
}

// Block is a template extension injected at a named point.
type Block struct {
	// ExtensionPoint is where to inject: "stub-method-options",
	// "stub-method-dispatch", "suite-subtests", etc.
	ExtensionPoint string

	// Content is pre-rendered Go source to inject.
	Content string
}

// Registry holds directive handlers and provides lookup by name.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates an empty [Registry].
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds a handler. Panics if a handler with the same name
// is already registered.
func (r *Registry) Register(h Handler) {
	name := h.Name()
	if _, exists := r.handlers[name]; exists {
		panic("directive: duplicate handler: " + name) //nolint:forbidigo
	}
	r.handlers[name] = h
}

// Get returns the handler for the given directive name, or nil.
func (r *Registry) Get(name string) Handler {
	return r.handlers[name]
}

// Names returns all registered directive names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Process looks up the handler for the named directive and calls it.
// Returns nil output if no handler is registered (unknown directives
// are silently ignored — generators may warn about them separately).
func (r *Registry) Process(name string, ctx Context) (*Output, error) {
	h := r.handlers[name]
	if h == nil {
		return nil, nil //nolint:nilnil // nil output signals "no handler registered"
	}
	return h.Process(ctx)
}
