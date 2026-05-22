// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package domhint

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"pgregory.net/rapid"
)

// Hint carries the per-type generator plus its display name. The
// generator's analyze step looks up [Hint] by [reflect.Type] when
// it encounters an opaque parameter; the runner threads the typed
// generator through the action emitter.
type Hint[T any] struct {
	// TypeName is the consumer-visible name shown in directive
	// guidance and option constructors (e.g., `ids.RunID`).
	TypeName string

	// Generator is the rapid generator that produces values of T.
	Generator *rapid.Generator[T]
}

// Registry is a thread-safe map from [reflect.Type] to opaque
// [Hint] values. Construct via [NewRegistry] or use the
// package-level [DefaultRegistry] singleton.
type Registry struct {
	mu    sync.Mutex
	hints map[reflect.Type]any
	names map[reflect.Type]string
}

// NewRegistry returns an empty [Registry].
func NewRegistry() *Registry {
	return &Registry{
		hints: make(map[reflect.Type]any),
		names: make(map[reflect.Type]string),
	}
}

// Register stores gen under the [reflect.Type] of T plus an
// optional display name. The name defaults to the type's qualified
// string when empty.
//
// Panics on collision (always-err at registration). The panic
// names both the conflicting type and the prior display name so
// the consumer can diagnose without re-running.
func Register[T any](r *Registry, name string, gen *rapid.Generator[T]) {
	typ := reflect.TypeFor[T]()
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, dup := r.hints[typ]; dup {
		//nolint:forbidigo // collision is a programmer error caught at init.
		panic(fmt.Sprintf(
			"domhint: type %s already registered as %q (was %T)",
			typ, r.names[typ], existing,
		))
	}
	if name == "" {
		name = typ.String()
	}
	r.hints[typ] = Hint[T]{TypeName: name, Generator: gen}
	r.names[typ] = name
}

// Lookup returns the registered generator for T when present. Use
// when the consumer's call site knows T statically.
func Lookup[T any](r *Registry) (*rapid.Generator[T], bool) {
	if r == nil {
		return nil, false
	}
	typ := reflect.TypeFor[T]()
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hints[typ]
	if !ok {
		return nil, false
	}
	hint, ok := h.(Hint[T])
	if !ok {
		return nil, false
	}
	return hint.Generator, true
}

// LookupByType returns the registered hint as an opaque [any] when
// present, plus the registered display name. Used by the codegen
// path that reflects over interface methods and does not have T
// available statically.
func LookupByType(r *Registry, typ reflect.Type) (any, string, bool) {
	if r == nil {
		return nil, "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hints[typ]
	if !ok {
		return nil, "", false
	}
	return h, r.names[typ], true
}

// TypeNames returns the sorted list of registered type names. Used
// by diagnostics ("registered hints: a, b, c") and by header
// emitters that surface the hint set.
func TypeNames(r *Registry) []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.names))
	for _, n := range r.names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Len returns the number of registered hints.
func Len(r *Registry) int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.hints)
}

var (
	defaultOnce sync.Once
	defaultReg  *Registry
)

// DefaultRegistry returns the package-level [Registry] singleton.
// The `//testkit:domain-gen` directive's emitted init code lands
// here; consumer-supplied per-test registries take precedence.
func DefaultRegistry() *Registry {
	defaultOnce.Do(func() {
		defaultReg = NewRegistry()
	})
	return defaultReg
}
