// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"io"
)

// Options carries impl-supplied inputs that downstream baseline
// assertions and contract directives need. Fields are typed where the
// shape is uniform across impls (durations, names, sentinels) and
// erased to any where the impl-T varies per interface — per-shape
// Asserters reflect on those fields at call sites.
//
// Construct an Options via [ResolveOptions] applied to one or more
// [Option] values returned by the With* constructors. An Options with
// zero values means "no impl-supplied input"; per-shape baselines that
// require a field skip the relevant assertion (or use a documented
// default) when the field is unset.
type Options struct {
	// InvalidFactory is a factory that produces an impl which must
	// fail RejectInvalid baselines (Lifecycle, VoidLifecycle, Mutator,
	// PoisonAccessor). The concrete type varies per interface, so the
	// field is any; per-shape Asserters reflect to call it.
	InvalidFactory any

	// PoisonedFactory is a factory that produces a poisoned impl, used
	// by PoisonAccessor's RejectInvalid baseline.
	PoisonedFactory any

	// PrePopulate is a func(T) seed callback applied to every freshly
	// factory-produced impl before contract subtests run. Stored as
	// any because T varies; the driver template wraps the user's
	// factory with a reflective trampoline.
	PrePopulate any

	// ObservableVia is the name of the reader method through which a
	// Mutator's or VoidLifecycle's effect can be observed. Used by
	// paired-method observation in RejectInvalid baselines.
	ObservableVia string

	// AggregatorBounds is a slice indexed by result-slot position.
	// Single-result Aggregator reads index 0; MultiAggregator reads
	// indices 0 and 1. Sliced rather than per-slot fields to avoid
	// Lower2/Upper2/etc. proliferation.
	AggregatorBounds []BoundPair

	// StreamSample is a factory producing a fresh stream value for
	// StreamConsumer baselines. Defaults are applied by the runtime
	// when nil.
	StreamSample func() io.Reader

	// HookRecorder is a typed registry the impl writes to via context.
	// Task 11r refines this field to the concrete *HookRecorder type
	// declared in suite/hookrecorder.go; for now it is any to avoid
	// a forward dep.
	HookRecorder any

	// ScopeContext produces a context.Context carrying the granted
	// scope; consumed by the scope-directive contract.
	ScopeContext func(scope string) context.Context

	// ScopeUnauthorized is the sentinel returned by impls for
	// unauthorized scope calls.
	ScopeUnauthorized error

	// LeaseRelease is the name of the release method paired with the
	// lease-directive method.
	LeaseRelease string

	// StateEqual is a func(a, b T) bool for atomic-directive's pre/post
	// state-equality check. Falls back to reflect.DeepEqual when nil.
	// Stored as any because T varies per impl.
	StateEqual any
}

// BoundPair holds a lower/upper bound for a single Aggregator result
// slot. Lower and Upper are any so the same slot can carry int, time,
// float, etc. as the impl requires; consumers reflect on the type.
type BoundPair struct {
	Lower any
	Upper any
}

// Option is a functional option that mutates an [Options] in place.
// Multiple options are folded by [ResolveOptions]; later options
// overwrite earlier ones for scalar fields (last-write semantics).
// Slice-valued fields like AggregatorBounds use append/index semantics
// documented on the relevant constructors.
type Option func(*Options)

// ResolveOptions folds a variadic sequence of [Option] into a single
// [Options] value. Options are applied left-to-right; later options
// overwrite earlier ones for scalar fields. The zero value is returned
// when no options are given.
func ResolveOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithInvalidFactory installs a factory that produces an impl which
// must fail the RejectInvalid baseline. The generic parameter T is
// erased into Options.InvalidFactory; per-shape Asserters reflect to
// call it.
func WithInvalidFactory[T any](factory func() T) Option {
	return func(o *Options) {
		o.InvalidFactory = factory
	}
}

// WithPoisonedFactory installs a factory that produces a poisoned
// impl, used by the PoisonAccessor RejectInvalid baseline.
func WithPoisonedFactory[T any](factory func() T) Option {
	return func(o *Options) {
		o.PoisonedFactory = factory
	}
}

// WithPrePopulate installs a func(T) seed callback applied to every
// freshly factory-produced impl before contract subtests run. T is
// erased; the driver template reflects on the value.
func WithPrePopulate[T any](seed func(T)) Option {
	return func(o *Options) {
		o.PrePopulate = seed
	}
}

// WithObservableVia names the reader method through which a Mutator's
// or VoidLifecycle's effect can be observed during paired-method
// RejectInvalid baselines.
func WithObservableVia(method string) Option {
	return func(o *Options) {
		o.ObservableVia = method
	}
}

// WithAggregatorBounds appends a [BoundPair] to
// [Options.AggregatorBounds]. Use this for the next unassigned slot —
// for explicit indexing, see [WithAggregatorBoundsAt].
func WithAggregatorBounds[R any](lower, upper R) Option {
	return func(o *Options) {
		o.AggregatorBounds = append(o.AggregatorBounds, BoundPair{
			Lower: lower,
			Upper: upper,
		})
	}
}

// WithAggregatorBoundsAt sets the i-th [BoundPair] in
// [Options.AggregatorBounds], extending the slice with zero-valued
// pairs if i is past the current length. Use this when the impl
// returns multiple result slots (e.g. MultiAggregator) and only one
// slot needs a non-trivial bound.
func WithAggregatorBoundsAt[R any](i int, lower, upper R) Option {
	return func(o *Options) {
		if i >= len(o.AggregatorBounds) {
			grown := make([]BoundPair, i+1)
			copy(grown, o.AggregatorBounds)
			o.AggregatorBounds = grown
		}
		o.AggregatorBounds[i] = BoundPair{Lower: lower, Upper: upper}
	}
}

// WithStreamSample installs a factory producing a fresh stream value
// for StreamConsumer baselines. Pass nil to fall back to runtime
// defaults.
func WithStreamSample(factory func() io.Reader) Option {
	return func(o *Options) {
		o.StreamSample = factory
	}
}

// WithScopeContext installs a builder that wraps a parent context with
// the granted scope; consumed by the scope-directive contract.
func WithScopeContext(fn func(string) context.Context) Option {
	return func(o *Options) {
		o.ScopeContext = fn
	}
}

// WithScopeUnauthorized installs the sentinel error returned by impls
// for unauthorized scope calls.
func WithScopeUnauthorized(err error) Option {
	return func(o *Options) {
		o.ScopeUnauthorized = err
	}
}

// WithLeaseRelease names the release method paired with the
// lease-directive method.
func WithLeaseRelease(method string) Option {
	return func(o *Options) {
		o.LeaseRelease = method
	}
}

// WithStateEqual installs a func(a, b T) bool for the atomic-directive
// pre/post state-equality check; the runtime falls back to
// reflect.DeepEqual when this option is not supplied. T is erased.
func WithStateEqual[T any](eq func(a, b T) bool) Option {
	return func(o *Options) {
		o.StateEqual = eq
	}
}
