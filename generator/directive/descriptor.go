// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Descriptor describes a known directive. It is a pure metadata
// record — descriptors do not process directives. Generators register
// behavior via [RegisterConsumer] (enrichment) and [RegisterEmitter]
// (mixin emission); detectors in the shape package consume
// [SignatureHint] descriptors directly.
//
// Most callers should use [New] with [Option] funcs rather than
// constructing this struct literally; the builder validates
// invariants (Multi only on last arg, etc.) at registration time.
type Descriptor struct {
	// Name is the directive name (no //testkit: prefix).
	Name string

	// Description is a one-line summary for documentation.
	Description string

	// Category groups the directive with its peers. See [Category].
	Category Category

	// Phase is the rollout phase from the directive design spec.
	Phase Phase

	// Args declares the argument slots (positional, in order). The
	// validator walks Args against the actual args on each directive
	// occurrence.
	Args []ArgSpec

	// Conflicts names directives that must NOT appear alongside this
	// one on the same method. Validation surfaces a [Conflict] issue
	// for every conflicting pair present.
	Conflicts []string

	// Requires names directives that MUST appear alongside this one.
	// Validation surfaces a [MissingRequired] issue for every absent
	// requirement.
	Requires []string

	// Implies names directives that this one transitively claims.
	// Pipeline behavior (today): redundancy warning when the implied
	// directive is also explicitly present. Future: auto-apply the
	// implied directive's emitter without requiring its declaration
	// in source — gated behind explicit consumer opt-in.
	Implies []string

	// ComposesWith is informational: directives that pair naturally
	// with this one. Surfaces in doc-gen ("commonly used with…") and
	// is not enforced.
	ComposesWith []string

	// Experimental marks the directive as opt-in via the
	// "experimental:<name>" prefix. Validation warns instead of
	// erroring for experimental directives that have no consumer.
	Experimental bool

	// Consumers maps a generator name (`stub`, `suite`, `bench`,
	// `model`) to a one-line description of what that generator
	// emits in response to the directive. Populated via [Consumed]
	// — generators surface this map in file headers so reviewers
	// see, per directive, which generators it shapes and how.
	Consumers map[string]string
}

// Option mutates a [Descriptor] during construction. Used as the
// trailing varargs to [New].
type Option func(*Descriptor)

// New constructs a [Descriptor] using option functions and validates
// internal consistency. Returns the value; call [Registry.Register]
// or [Registry.MustRegister] separately.
//
// Panics on invalid configuration (Multi on non-last arg, duplicate
// conflict/require entries, ...) — these are programmer errors caught
// at init time. The panic message lists every issue found, not just
// the first, so a single fix-and-rerun cycle resolves all of them.
func New(name string, opts ...Option) Descriptor {
	d := Descriptor{Name: name}
	for _, opt := range opts {
		opt(&d)
	}
	if errs := validateDescriptor(&d); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = "  - " + e.Error()
		}
		//nolint:forbidigo // init-time programmer error
		panic(fmt.Sprintf("directive.New(%q):\n%s", name, strings.Join(msgs, "\n")))
	}
	return d
}

// Describe sets the descriptor's [Descriptor.Description].
func Describe(s string) Option { return func(d *Descriptor) { d.Description = s } }

// InCategory sets the descriptor's [Descriptor.Category].
func InCategory(c Category) Option { return func(d *Descriptor) { d.Category = c } }

// InPhase sets the descriptor's [Descriptor.Phase].
func InPhase(p Phase) Option { return func(d *Descriptor) { d.Phase = p } }

// Arg appends one [ArgSpec] to the descriptor's positional args.
// Use [Required], [Multi], and [OneOf] to refine the slot:
//
//	directive.Arg("ErrName", directive.ArgIdent, directive.Required, directive.Multi)
func Arg(name string, kind ArgKind, opts ...ArgOption) Option {
	spec := ArgSpec{Name: name, Kind: kind}
	for _, opt := range opts {
		opt(&spec)
	}
	return func(d *Descriptor) {
		d.Args = append(d.Args, spec)
	}
}

// ConflictsWith adds entries to [Descriptor.Conflicts]. May be called
// multiple times; later calls append.
func ConflictsWith(names ...string) Option {
	return func(d *Descriptor) {
		d.Conflicts = append(d.Conflicts, names...)
	}
}

// Requires adds entries to [Descriptor.Requires]. The package-level
// function does not collide with [ArgSpec.Required] (an [ArgOption])
// because they have distinct types and signatures.
func Requires(names ...string) Option {
	return func(d *Descriptor) {
		d.Requires = append(d.Requires, names...)
	}
}

// Implies adds entries to [Descriptor.Implies]. The registry computes
// a transitive closure at build time so this descriptor's effective
// Conflicts and Requires include those of every implied descriptor —
// declaring `Implies(Pure)` is enough; you do not also need to repeat
// `Pure`'s conflicts.
func Implies(names ...string) Option {
	return func(d *Descriptor) {
		d.Implies = append(d.Implies, names...)
	}
}

// ComposesWith adds entries to [Descriptor.ComposesWith].
// Informational only; surfaces in doc-gen.
func ComposesWith(names ...string) Option {
	return func(d *Descriptor) {
		d.ComposesWith = append(d.ComposesWith, names...)
	}
}

// Experimental marks the descriptor as opt-in via the
// "experimental:<name>" prefix.
func Experimental() Option {
	return func(d *Descriptor) { d.Experimental = true }
}

// Consumed records that the named generator (`stub`, `suite`,
// `bench`, `model`) consumes the directive, with a one-line action
// describing what gets emitted. Multiple [Consumed] calls register
// multiple consumers — many directives shape more than one
// generator (e.g. `errors` produces stub fault helpers AND suite
// per-sentinel subtests).
func Consumed(consumer, action string) Option {
	return func(d *Descriptor) {
		if d.Consumers == nil {
			d.Consumers = make(map[string]string)
		}
		d.Consumers[consumer] = action
	}
}

// validateDescriptor returns every invariant violation it finds. The
// builder aggregates and reports them all in one panic so the user
// can fix the entire descriptor in a single edit cycle.
func validateDescriptor(d *Descriptor) []error {
	var errs []error
	if d.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if d.Category == CategoryUnspecified {
		errs = append(errs, errors.New("category is required (use directive.InCategory(...))"))
	}
	for i, a := range d.Args {
		if a.Multi && i != len(d.Args)-1 {
			errs = append(errs, fmt.Errorf("arg %q: Multi only allowed on last arg", a.Name))
		}
		if a.Kind == ArgEnum && len(a.Enum) == 0 {
			errs = append(errs, fmt.Errorf("arg %q: ArgEnum requires non-empty Enum (use OneOf(...))", a.Name))
		}
	}
	if dup := firstDuplicate(d.Conflicts); dup != "" {
		errs = append(errs, fmt.Errorf("duplicate entry %q in Conflicts", dup))
	}
	if dup := firstDuplicate(d.Requires); dup != "" {
		errs = append(errs, fmt.Errorf("duplicate entry %q in Requires", dup))
	}
	if dup := firstDuplicate(d.Implies); dup != "" {
		errs = append(errs, fmt.Errorf("duplicate entry %q in Implies", dup))
	}
	for _, n := range d.Conflicts {
		if n == d.Name {
			errs = append(errs, fmt.Errorf("self-conflict: %q lists itself in Conflicts", d.Name))
		}
	}
	for _, n := range d.Implies {
		if n == d.Name {
			errs = append(errs, fmt.Errorf("self-implication: %q lists itself in Implies", d.Name))
		}
	}
	return errs
}

// firstDuplicate returns the first repeated entry in s, or "" if
// every entry is unique. Stable order preserved by walking left to
// right.
func firstDuplicate(s []string) string {
	seen := make(map[string]bool, len(s))
	for _, v := range s {
		if seen[v] {
			return v
		}
		seen[v] = true
	}
	return ""
}

// ValidateArgs checks one directive occurrence against the descriptor's
// argument schema. Returns one error per violation. The pipeline calls
// this at directive-validation time alongside known-name checks.
//
// Off=true directives skip arg validation — the off-toggle form is
// always argless by parser convention.
func (d Descriptor) ValidateArgs(args []string, off bool) []error {
	if off {
		return nil
	}
	var errs []error
	requiredCount := 0
	for _, a := range d.Args {
		if a.Required {
			requiredCount++
		}
	}
	if len(args) < requiredCount {
		errs = append(errs, fmt.Errorf("%s: needs %d argument(s), got %d",
			d.Name, requiredCount, len(args)))
		return errs
	}

	idx := 0
	for _, spec := range d.Args {
		if spec.Multi {
			if !spec.Required && idx >= len(args) {
				return errs
			}
			for ; idx < len(args); idx++ {
				if err := validateArg(spec, args[idx]); err != nil {
					errs = append(errs, fmt.Errorf("%s: %w", d.Name, err))
				}
			}
			return errs
		}
		if idx >= len(args) {
			return errs
		}
		if err := validateArg(spec, args[idx]); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Name, err))
		}
		idx++
	}

	if idx < len(args) {
		// Surplus args past the last spec. Report as a single error.
		extras := append([]string(nil), args[idx:]...)
		sort.Strings(extras) // deterministic message
		errs = append(errs, fmt.Errorf("%s: unexpected extra args %v", d.Name, extras))
	}
	return errs
}
