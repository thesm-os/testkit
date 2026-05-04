// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

// Action is a named command that runs against both the SUT and
// reference. The runner dispatches actions randomly via rapid's
// state-machine orchestration.
type Action[T any] struct {
	// Name is the action name (e.g., "Get", "Put", "Delete").
	Name string

	// Run executes the action against both SUT and reference.
	Run func(rt *rapid.T, sut, ref T)
}

// Config holds the configuration for a model-based test run.
type Config[T any] struct {
	// SUTFactory creates a fresh SUT per test run. Required.
	SUTFactory func() T

	// RefFactory creates a fresh reference model per test run. Required.
	RefFactory func() T

	// Actions are the commands the runner randomly selects from.
	Actions []Action[T]

	// Laws are the invariants checked after every action.
	Laws *Registry[T]

	// Cleanup is called on SUT and ref after each iteration.
	// Optional. Use for impls that hold resources (connections,
	// goroutines, file handles).
	Cleanup func(T)
}

// Run executes a model-based test. For each rapid iteration, it
// creates fresh SUT and reference instances, runs a random sequence
// of actions, and checks all registered laws after every action.
//
// Use -rapid.checks=N to control iteration count (default 100).
func Run[T any](t rapid.TB, cfg Config[T]) {
	t.Helper()
	opts := []Option[T]{
		WithActions[T](cfg.Actions...),
	}
	if cfg.RefFactory != nil {
		opts = append(opts, WithReference(cfg.RefFactory))
	}
	if cfg.Laws != nil {
		opts = append(opts, WithLaws(cfg.Laws))
	}
	if cfg.Cleanup != nil {
		opts = append(opts, WithCleanup(cfg.Cleanup))
	}
	rapid.Check(t, Property(cfg.SUTFactory, opts...))
}

// Property builds the rapid property function from a factory and options
// without running it. Use this to obtain the property for [rapid.MakeFuzz]:
//
//	prop := model.Property(factory, model.WithReference(ref), ...)
//	f.Fuzz(rapid.MakeFuzz(prop))
func Property[T any](sutFactory func() T, opts ...Option[T]) func(*rapid.T) {
	cfg := Config[T]{SUTFactory: sutFactory}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Laws == nil {
		cfg.Laws = NewRegistry[T]()
	}

	return func(rt *rapid.T) {
		if cfg.SUTFactory == nil {
			rt.Fatal("model.Property: SUTFactory is required")
		}
		if cfg.RefFactory == nil {
			rt.Fatal("model.Property: RefFactory is required")
		}
		if len(cfg.Actions) == 0 {
			rt.Fatal("model.Property: at least one Action is required")
		}

		sut := cfg.SUTFactory()
		ref := cfg.RefFactory()
		step := 0

		if cfg.Cleanup != nil {
			defer cfg.Cleanup(sut)
			defer cfg.Cleanup(ref)
		}

		actionMap := make(map[string]func(*rapid.T), len(cfg.Actions)+1)
		for _, a := range cfg.Actions {
			a := a
			actionMap[a.Name] = func(rt *rapid.T) {
				a.Run(rt, sut, ref)
				step++
			}
		}

		// Empty-string key: rapid's check action, after every command.
		actionMap[""] = func(rt *rapid.T) {
			for _, l := range cfg.Laws.laws {
				cfg.Laws.ran[l.ID()]++
				if err := l.Check(rt, sut, ref); err != nil {
					f := &Failure{
						Kind:    FailureInvariant,
						LawID:   l.ID(),
						REQID:   l.REQID(),
						Step:    step,
						Message: err.Error(),
						Err:     err,
					}
					rt.Fatalf("%v", f)
				}
			}
		}

		rt.Repeat(actionMap)
	}
}

// Option configures a model-based test run.
type Option[T any] func(*Config[T])

// WithReference sets the reference model factory.
func WithReference[T any](factory func() T) Option[T] {
	return func(c *Config[T]) { c.RefFactory = factory }
}

// WithActions sets the action list.
func WithActions[T any](actions ...Action[T]) Option[T] {
	return func(c *Config[T]) { c.Actions = actions }
}

// WithLaws sets the entire law registry. Used by the generator to pass
// the pre-built auto-law registry.
func WithLaws[T any](r *Registry[T]) Option[T] {
	return func(c *Config[T]) { c.Laws = r }
}

// WithLaw adds a law to the registry.
func WithLaw[T any](l law.Law[T]) Option[T] {
	return func(c *Config[T]) {
		if c.Laws == nil {
			c.Laws = NewRegistry[T]()
		}
		c.Laws.Add(l)
	}
}

// WithLawREQ adds a law with a REQ tag for traceability.
func WithLawREQ[T any](reqID string, l law.Law[T]) Option[T] {
	return func(c *Config[T]) {
		if c.Laws == nil {
			c.Laws = NewRegistry[T]()
		}
		c.Laws.Add(&taggedLaw[T]{Law: l, reqID: reqID})
	}
}

// WithCleanup sets a cleanup function called on SUT and ref after
// each iteration.
func WithCleanup[T any](fn func(T)) Option[T] {
	return func(c *Config[T]) { c.Cleanup = fn }
}

// SkipLaw removes an auto-derived law by ID.
func SkipLaw[T any](id string) Option[T] {
	return func(c *Config[T]) {
		if c.Laws != nil {
			c.Laws.SkipByID(id)
		}
	}
}

// Assert is the convenience entry point.
func Assert[T any](t rapid.TB, sutFactory func() T, opts ...Option[T]) {
	t.Helper()
	rapid.Check(t, Property(sutFactory, opts...))
}

// taggedLaw wraps a law with a REQ ID override.
type taggedLaw[T any] struct {
	law.Law[T]
	reqID string
}

func (t *taggedLaw[T]) REQID() string { return t.reqID }
