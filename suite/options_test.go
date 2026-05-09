// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/suite"
)

// fixture is a minimal impl-T placeholder for type-erased factory
// round-trips. The Options framework stores impl-typed callbacks as
// any; tests reflect via type assertion to confirm the registered
// concrete type round-trips intact.
type fixture struct{ tag string }

func TestOptionsZeroValue(t *testing.T) {
	t.Parallel()
	o := suite.ResolveOptions()
	testkit.True(t, o.InvalidFactory == nil, "InvalidFactory zero")
	testkit.True(t, o.PoisonedFactory == nil, "PoisonedFactory zero")
	testkit.True(t, o.PrePopulate == nil, "PrePopulate zero")
	testkit.Equal(t, o.ObservableVia, "", "ObservableVia zero")
	testkit.Equal(t, len(o.AggregatorBounds), 0, "AggregatorBounds zero")
	testkit.True(t, o.StreamSample == nil, "StreamSample zero")
	testkit.True(t, o.HookRecorder == nil, "HookRecorder zero")
	testkit.True(t, o.ScopeContext == nil, "ScopeContext zero")
	testkit.True(t, o.ScopeUnauthorized == nil, "ScopeUnauthorized zero")
	testkit.Equal(t, o.LeaseRelease, "", "LeaseRelease zero")
	testkit.True(t, o.StateEqual == nil, "StateEqual zero")
}

func TestOptionsWithInvalidFactory(t *testing.T) {
	t.Parallel()
	t.Run("stores typed factory", func(t *testing.T) {
		t.Parallel()
		want := &fixture{tag: "invalid"}
		factory := func() *fixture { return want }
		o := suite.ResolveOptions(suite.WithInvalidFactory(factory))
		got, ok := o.InvalidFactory.(func() *fixture)
		testkit.True(t, ok, "InvalidFactory type asserts to func() *fixture")
		testkit.Equal(t, got(), want, "round-tripped factory yields registered value")
	})
	t.Run("last-write wins", func(t *testing.T) {
		t.Parallel()
		first := func() *fixture { return &fixture{tag: "first"} }
		second := func() *fixture { return &fixture{tag: "second"} }
		o := suite.ResolveOptions(
			suite.WithInvalidFactory(first),
			suite.WithInvalidFactory(second),
		)
		got, ok := o.InvalidFactory.(func() *fixture)
		testkit.True(t, ok, "type assert")
		testkit.Equal(t, got().tag, "second", "later option overrides earlier")
	})
}

func TestOptionsWithPoisonedFactory(t *testing.T) {
	t.Parallel()
	want := &fixture{tag: "poison"}
	factory := func() *fixture { return want }
	o := suite.ResolveOptions(suite.WithPoisonedFactory(factory))
	got, ok := o.PoisonedFactory.(func() *fixture)
	testkit.True(t, ok, "PoisonedFactory type asserts")
	testkit.Equal(t, got(), want, "factory round-trips")
}

func TestOptionsWithPrePopulate(t *testing.T) {
	t.Parallel()
	t.Run("stores typed seed", func(t *testing.T) {
		t.Parallel()
		var seen string
		seed := func(f *fixture) { seen = f.tag }
		o := suite.ResolveOptions(suite.WithPrePopulate(seed))
		got, ok := o.PrePopulate.(func(*fixture))
		testkit.True(t, ok, "PrePopulate type asserts to func(*fixture)")
		got(&fixture{tag: "applied"})
		testkit.Equal(t, seen, "applied", "stored seed callback executes against impl")
	})
	t.Run("last-write wins", func(t *testing.T) {
		t.Parallel()
		var calls []string
		first := func(f *fixture) { calls = append(calls, "first:"+f.tag) }
		second := func(f *fixture) { calls = append(calls, "second:"+f.tag) }
		o := suite.ResolveOptions(
			suite.WithPrePopulate(first),
			suite.WithPrePopulate(second),
		)
		got, ok := o.PrePopulate.(func(*fixture))
		testkit.True(t, ok, "type assert")
		got(&fixture{tag: "x"})
		testkit.Equal(t, len(calls), 1, "only one seed invoked")
		testkit.True(t, strings.HasPrefix(calls[0], "second:"), "later option wins")
	})
}

func TestOptionsWithObservableVia(t *testing.T) {
	t.Parallel()
	t.Run("stores method name", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(suite.WithObservableVia("Read"))
		testkit.Equal(t, o.ObservableVia, "Read", "method name stored")
	})
	t.Run("last-write wins", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(
			suite.WithObservableVia("Old"),
			suite.WithObservableVia("New"),
		)
		testkit.Equal(t, o.ObservableVia, "New", "later option overrides")
	})
}

func TestOptionsWithAggregatorBounds(t *testing.T) {
	t.Parallel()
	t.Run("appends single pair", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(suite.WithAggregatorBounds(1, 10))
		testkit.Equal(t, len(o.AggregatorBounds), 1, "one pair")
		testkit.Equal(t, o.AggregatorBounds[0].Lower.(int), 1, "lower")
		testkit.Equal(t, o.AggregatorBounds[0].Upper.(int), 10, "upper")
	})
	t.Run("appends multiple pairs", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(
			suite.WithAggregatorBounds(0, 5),
			suite.WithAggregatorBounds("a", "z"),
		)
		testkit.Equal(t, len(o.AggregatorBounds), 2, "two pairs")
		testkit.Equal(t, o.AggregatorBounds[0].Lower.(int), 0, "slot 0 lower")
		testkit.Equal(t, o.AggregatorBounds[0].Upper.(int), 5, "slot 0 upper")
		testkit.Equal(t, o.AggregatorBounds[1].Lower.(string), "a", "slot 1 lower")
		testkit.Equal(t, o.AggregatorBounds[1].Upper.(string), "z", "slot 1 upper")
	})
}

func TestOptionsWithAggregatorBoundsAt(t *testing.T) {
	t.Parallel()
	t.Run("sets in-range index", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(
			suite.WithAggregatorBounds(0, 0),
			suite.WithAggregatorBoundsAt(0, 7, 9),
		)
		testkit.Equal(t, len(o.AggregatorBounds), 1, "still one pair")
		testkit.Equal(t, o.AggregatorBounds[0].Lower.(int), 7, "overwritten lower")
		testkit.Equal(t, o.AggregatorBounds[0].Upper.(int), 9, "overwritten upper")
	})
	t.Run("extends past end", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(suite.WithAggregatorBoundsAt(2, 100, 200))
		testkit.Equal(t, len(o.AggregatorBounds), 3, "slice grown to index+1")
		testkit.True(t, o.AggregatorBounds[0].Lower == nil && o.AggregatorBounds[0].Upper == nil,
			"unset slots are zero-valued")
		testkit.True(t, o.AggregatorBounds[1].Lower == nil && o.AggregatorBounds[1].Upper == nil,
			"unset slots are zero-valued")
		testkit.Equal(t, o.AggregatorBounds[2].Lower.(int), 100, "set slot lower")
		testkit.Equal(t, o.AggregatorBounds[2].Upper.(int), 200, "set slot upper")
	})
	t.Run("preserves earlier entries when growing", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(
			suite.WithAggregatorBounds(1, 2),
			suite.WithAggregatorBoundsAt(3, 30, 40),
		)
		testkit.Equal(t, len(o.AggregatorBounds), 4, "grown to 4")
		testkit.Equal(t, o.AggregatorBounds[0].Lower.(int), 1, "earlier entry preserved")
		testkit.Equal(t, o.AggregatorBounds[0].Upper.(int), 2, "earlier entry preserved")
		testkit.Equal(t, o.AggregatorBounds[3].Lower.(int), 30, "new slot lower")
		testkit.Equal(t, o.AggregatorBounds[3].Upper.(int), 40, "new slot upper")
	})
}

func TestOptionsWithStreamSample(t *testing.T) {
	t.Parallel()
	t.Run("stores factory", func(t *testing.T) {
		t.Parallel()
		factory := func() io.Reader { return strings.NewReader("hello") }
		o := suite.ResolveOptions(suite.WithStreamSample(factory))
		testkit.True(t, o.StreamSample != nil, "factory wired")
		buf, err := io.ReadAll(o.StreamSample())
		testkit.NoError(t, err, "read sample")
		testkit.Equal(t, string(buf), "hello", "factory round-trips content")
	})
	t.Run("last-write wins", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(
			suite.WithStreamSample(func() io.Reader { return strings.NewReader("a") }),
			suite.WithStreamSample(func() io.Reader { return strings.NewReader("b") }),
		)
		buf, err := io.ReadAll(o.StreamSample())
		testkit.NoError(t, err, "read")
		testkit.Equal(t, string(buf), "b", "later sample wins")
	})
}

func TestOptionsWithScopeContext(t *testing.T) {
	t.Parallel()
	type scopeKey struct{}
	parent := t.Context()
	build := func(scope string) context.Context {
		return context.WithValue(parent, scopeKey{}, scope)
	}
	o := suite.ResolveOptions(suite.WithScopeContext(build))
	testkit.True(t, o.ScopeContext != nil, "ScopeContext wired")
	ctx := o.ScopeContext("admin")
	got, ok := ctx.Value(scopeKey{}).(string)
	testkit.True(t, ok, "scope value present")
	testkit.Equal(t, got, "admin", "scope round-trips")
}

func TestOptionsWithScopeUnauthorized(t *testing.T) {
	t.Parallel()
	t.Run("stores sentinel", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("forbidden")
		o := suite.ResolveOptions(suite.WithScopeUnauthorized(sentinel))
		testkit.ErrorIs(t, o.ScopeUnauthorized, sentinel, "sentinel preserved")
	})
	t.Run("last-write wins", func(t *testing.T) {
		t.Parallel()
		first := errors.New("first")
		second := errors.New("second")
		o := suite.ResolveOptions(
			suite.WithScopeUnauthorized(first),
			suite.WithScopeUnauthorized(second),
		)
		testkit.ErrorIs(t, o.ScopeUnauthorized, second, "later sentinel wins")
		testkit.True(t, !errors.Is(o.ScopeUnauthorized, first), "first sentinel discarded")
	})
}

func TestOptionsWithLeaseRelease(t *testing.T) {
	t.Parallel()
	t.Run("stores method name", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(suite.WithLeaseRelease("Release"))
		testkit.Equal(t, o.LeaseRelease, "Release", "method name stored")
	})
	t.Run("last-write wins", func(t *testing.T) {
		t.Parallel()
		o := suite.ResolveOptions(
			suite.WithLeaseRelease("OldRelease"),
			suite.WithLeaseRelease("NewRelease"),
		)
		testkit.Equal(t, o.LeaseRelease, "NewRelease", "later option overrides")
	})
}

func TestOptionsWithStateEqual(t *testing.T) {
	t.Parallel()
	eq := func(a, b *fixture) bool { return a.tag == b.tag }
	o := suite.ResolveOptions(suite.WithStateEqual(eq))
	got, ok := o.StateEqual.(func(a, b *fixture) bool)
	testkit.True(t, ok, "StateEqual type asserts")
	testkit.True(t, got(&fixture{tag: "x"}, &fixture{tag: "x"}), "equal tags")
	testkit.False(t, got(&fixture{tag: "x"}, &fixture{tag: "y"}), "differing tags")
}

func TestOptionsAccumulateMultiple(t *testing.T) {
	t.Parallel()
	// All scalar/factory options together — confirms the variadic
	// fold preserves every channel without cross-talk.
	invalid := func() *fixture { return &fixture{tag: "bad"} }
	poisoned := func() *fixture { return &fixture{tag: "poison"} }
	seed := func(*fixture) {}
	stream := func() io.Reader { return strings.NewReader("s") }
	parent := t.Context()
	scopeBuild := func(string) context.Context { return parent }
	scopeErr := errors.New("scope")
	stateEq := func(a, b *fixture) bool { return a == b }

	o := suite.ResolveOptions(
		suite.WithInvalidFactory(invalid),
		suite.WithPoisonedFactory(poisoned),
		suite.WithPrePopulate(seed),
		suite.WithObservableVia("Get"),
		suite.WithAggregatorBounds(0, 100),
		suite.WithAggregatorBoundsAt(2, 200, 300),
		suite.WithStreamSample(stream),
		suite.WithScopeContext(scopeBuild),
		suite.WithScopeUnauthorized(scopeErr),
		suite.WithLeaseRelease("Unlock"),
		suite.WithStateEqual(stateEq),
	)

	testkit.True(t, o.InvalidFactory != nil, "InvalidFactory accumulated")
	testkit.True(t, o.PoisonedFactory != nil, "PoisonedFactory accumulated")
	testkit.True(t, o.PrePopulate != nil, "PrePopulate accumulated")
	testkit.Equal(t, o.ObservableVia, "Get", "ObservableVia accumulated")
	testkit.Equal(t, len(o.AggregatorBounds), 3, "AggregatorBounds spans append+at")
	testkit.Equal(t, o.AggregatorBounds[0].Upper.(int), 100, "appended slot intact")
	testkit.Equal(t, o.AggregatorBounds[2].Lower.(int), 200, "indexed slot intact")
	testkit.True(t, o.StreamSample != nil, "StreamSample accumulated")
	testkit.True(t, o.ScopeContext != nil, "ScopeContext accumulated")
	testkit.ErrorIs(t, o.ScopeUnauthorized, scopeErr, "ScopeUnauthorized accumulated")
	testkit.Equal(t, o.LeaseRelease, "Unlock", "LeaseRelease accumulated")
	testkit.True(t, o.StateEqual != nil, "StateEqual accumulated")
}
