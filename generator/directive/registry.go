// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"fmt"
	"go/token"
	"sort"
	"strings"
)

// Phase enumerates the rollout phases from the directive design spec.
// Higher phase numbers ship later. Used for documentation and roadmap
// planning; not load-bearing at runtime.
type Phase int

// Recognized phases.
const (
	PhaseUnspecified Phase = 0
	Phase1           Phase = 1 // High value, low complexity
	Phase2           Phase = 2 // Performance (bench-scoped)
	Phase3           Phase = 3 // Medium complexity
	Phase4           Phase = 4 // High complexity
	Phase5           Phase = 5 // Niche
	Phase6           Phase = 6 // Broader scopes (field/type/var/const)
)

// Registry holds [Descriptor]s indexed by name. The registry is the
// single source of truth for known-directive validation, arg-spec
// validation, composition checks, and doc-gen.
//
// Construct via [NewRegistry] or use [DefaultRegistry] for the
// package-level singleton populated with the canonical descriptor set.
type Registry struct {
	descriptors map[string]Descriptor
}

// NewRegistry returns an empty [Registry].
func NewRegistry() *Registry {
	return &Registry{descriptors: make(map[string]Descriptor)}
}

// Register adds a descriptor. Returns an error if a descriptor with
// the same name is already registered. Init code that knows the
// registration is unique should call [MustRegister].
func (r *Registry) Register(d Descriptor) error {
	if _, exists := r.descriptors[d.Name]; exists {
		return fmt.Errorf("duplicate directive descriptor: %q", d.Name)
	}
	r.descriptors[d.Name] = d
	return nil
}

// MustRegister adds a descriptor and panics on duplicate.
func (r *Registry) MustRegister(d Descriptor) {
	if err := r.Register(d); err != nil {
		panic(err.Error()) //nolint:forbidigo // init-time programmer error
	}
}

// Get returns the descriptor for a name, or zero+false if absent.
func (r *Registry) Get(name string) (Descriptor, bool) {
	d, ok := r.descriptors[name]
	return d, ok
}

// IsKnown reports whether a directive name is registered.
func (r *Registry) IsKnown(name string) bool {
	_, ok := r.descriptors[name]
	return ok
}

// Names returns all registered directive names sorted alphabetically.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.descriptors))
	for name := range r.descriptors {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Descriptors returns all descriptors sorted by name.
func (r *Registry) Descriptors() []Descriptor {
	out := make([]Descriptor, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// DescriptorsIn returns all descriptors in the named [Category],
// sorted by name. Used by doc-gen to render per-category sections.
func (r *Registry) DescriptorsIn(c Category) []Descriptor {
	out := make([]Descriptor, 0)
	for _, d := range r.descriptors {
		if d.Category == c {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Validate checks that every directive in dirs is known and that its
// arguments match the descriptor's [ArgSpec] schema.
//
// methodName and pos provide diagnostic context — they appear in
// error messages but are not otherwise consulted. Pipelines that hold
// [generator.MethodInfo] iterate methods themselves and call Validate
// per method.
//
// Unknown directives produce errors; "experimental:<name>" directives
// produce warnings via warn (when non-nil) and are otherwise allowed.
func (r *Registry) Validate(dirs []Directive, methodName string, pos token.Position, warn func(string)) []error {
	var errs []error
	for _, d := range dirs {
		if strings.HasPrefix(d.Name, "experimental:") {
			if warn != nil {
				warn("experimental directive " + d.Name + " on " + methodName)
			}
			continue
		}
		desc, ok := r.descriptors[d.Name]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: unknown directive %q on method %s", pos, d.Name, methodName))
			continue
		}
		for _, e := range desc.ValidateArgs(d.Args, d.Off) {
			errs = append(errs, fmt.Errorf("%s: on method %s: %w", pos, methodName, e))
		}
	}
	return errs
}

// closeImplications computes the transitive closure of every
// descriptor's Implies relation and merges the implied descriptors'
// Conflicts and Requires into the current descriptor.
//
// After this runs, `desc.Conflicts` is the *effective* conflict set
// (its own + every transitively-implied descriptor's). Same for
// Requires. Implies itself is also flattened to the full closure.
//
// Cycles in Implies are tolerated (visited tracking).
func (r *Registry) closeImplications() {
	for name, desc := range r.descriptors {
		closure := r.implicationClosure(name)
		var conflicts []string
		var requires []string
		var implies []string
		seen := map[string]bool{name: true}

		conflicts = append(conflicts, desc.Conflicts...)
		requires = append(requires, desc.Requires...)

		for _, impliedName := range closure {
			if seen[impliedName] {
				continue
			}
			seen[impliedName] = true
			implies = append(implies, impliedName)
			implied, ok := r.descriptors[impliedName]
			if !ok {
				continue
			}
			conflicts = appendUnique(conflicts, implied.Conflicts...)
			requires = appendUnique(requires, implied.Requires...)
		}
		desc.Conflicts = conflicts
		desc.Requires = requires
		desc.Implies = implies
		r.descriptors[name] = desc
	}
}

// implicationClosure returns every descriptor reachable from name via
// Implies edges, in BFS order. Excludes name itself.
func (r *Registry) implicationClosure(name string) []string {
	visited := map[string]bool{name: true}
	var queue []string
	if d, ok := r.descriptors[name]; ok {
		queue = append(queue, d.Implies...)
	}
	var out []string
	for i := 0; i < len(queue); i++ {
		next := queue[i]
		if visited[next] {
			continue
		}
		visited[next] = true
		out = append(out, next)
		if d, ok := r.descriptors[next]; ok {
			queue = append(queue, d.Implies...)
		}
	}
	return out
}

// appendUnique appends names to s skipping any value already present.
func appendUnique(s []string, names ...string) []string {
	seen := make(map[string]bool, len(s))
	for _, v := range s {
		seen[v] = true
	}
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		s = append(s, n)
	}
	return s
}
