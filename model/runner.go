// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"time"

	"github.com/anishathalye/porcupine"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

// OpInput is recorded as porcupine.Operation.Input for concurrent
// linearizability checking. Used by both the concurrent runner and
// the linearize package's model builders.
type OpInput struct {
	Name         string // action name ("Get", "Put", "Delete")
	PartitionKey string // for Porcupine partitioning; "" for unpartitioned
	Args         any    // shape-specific: K for Reader/Deleter, V for Writer
}

// OpOutput is recorded as porcupine.Operation.Output.
type OpOutput struct {
	Result any // shape-specific typed result
}

// Action is a named command that runs against both the SUT and
// reference. The runner dispatches actions randomly via rapid's
// state-machine orchestration.
type Action[T any] struct {
	// Name is the action name (e.g., "Get", "Put", "Delete").
	Name string

	// Run executes the action against both SUT and reference.
	Run func(rt *rapid.T, sut, ref T)
}

// ConcurrentAction is an action that records structured I/O for
// Porcupine linearizability checking. Unlike [Action], it operates
// on the SUT only (no reference) and captures typed input/output.
type ConcurrentAction[T any] struct {
	// Name is the action name (e.g., "Get", "Put").
	Name string

	// Gen draws a random input using rapid and returns it.
	Gen func(rt *rapid.T) any

	// Apply runs the operation against an impl and returns the result.
	Apply func(ctx context.Context, impl T, input any) any

	// PartitionKey extracts the string partition key from the input.
	// Return "" for unpartitioned operations.
	PartitionKey func(input any) string
}

// ConcurrentConfig configures concurrent linearizability testing.
type ConcurrentConfig[T any] struct {
	// Workers is the number of concurrent goroutines. Default 4.
	Workers int
	// OpsPerWorker is the number of operations each worker performs. Default 50.
	OpsPerWorker int
	// Timeout for the Porcupine check. Default 10s. Zero means unlimited.
	Timeout time.Duration
	// Model is the Porcupine linearizability model. Use [linearize.KV]
	// for CRUD interfaces or [linearize.NewModelBuilder] for custom specs.
	Model porcupine.Model
	// Actions are linearizability-checked via Porcupine.
	Actions []ConcurrentAction[T]
	// StressActions run concurrently alongside linearizability workers
	// but are NOT recorded to Porcupine. Purpose: race detection under -race.
	StressActions []Action[T]
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

	// HistoryResetters are called at the start of each rapid iteration
	// to reset per-iteration chain history traces. Wired by
	// [WithHistoryReset].
	HistoryResetters []func()

	// Concurrent enables concurrent linearizability testing.
	// When set, the runner spawns workers and validates via Porcupine
	// instead of running the sequential property.
	Concurrent *ConcurrentConfig[T]
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

		// Reset per-iteration history traces (chain laws).
		for _, reset := range cfg.HistoryResetters {
			reset()
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
			actionMap[a.Name] = func(rt *rapid.T) {
				a.Run(rt, sut, ref)
				step++
			}
		}

		// Empty-string key: rapid's check action, after every command.
		actionMap[""] = func(rt *rapid.T) {
			for _, l := range cfg.Laws.laws {
				cfg.Laws.ran[l.ID()]++
				var err error
				if sl, ok := l.(law.StatefulLaw[T]); ok {
					err = sl.CheckWithStep(rt, sut, ref, step)
				} else {
					err = l.Check(rt, sut, ref)
				}
				if err != nil {
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

// WithConcurrent enables concurrent linearizability testing.
func WithConcurrent[T any](cfg ConcurrentConfig[T]) Option[T] {
	return func(c *Config[T]) { c.Concurrent = &cfg }
}

// WithHistoryReset registers a reset function called at the start of
// each rapid iteration. Used by chain action helpers to clear the
// per-iteration append history.
func WithHistoryReset[T any](reset func()) Option[T] {
	return func(c *Config[T]) {
		c.HistoryResetters = append(c.HistoryResetters, reset)
	}
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
	cfg := Config[T]{SUTFactory: sutFactory}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Concurrent != nil {
		runConcurrent(t, cfg)
		return
	}
	rapid.Check(t, Property(sutFactory, opts...))
}

// taggedLaw wraps a law with a REQ ID override.
type taggedLaw[T any] struct {
	law.Law[T]
	reqID string
}

func (t *taggedLaw[T]) REQID() string { return t.reqID }
