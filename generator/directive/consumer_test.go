// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestConsumerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register stores (directive, generator) pairs", func(t *testing.T) {
		t.Parallel()
		r := directive.NewConsumerRegistry()
		testkit.NoError(t, r.Register(directive.Consumer{Directive: "errors", Generator: "stub"}), "first register")
		testkit.NoError(t, r.Register(directive.Consumer{Directive: "errors", Generator: "suite"}), "second register")
		testkit.Len(t, r.Consumers("errors"), 2, "two generators consume errors")
		testkit.Len(t, r.Consumers("missing"), 0, "no consumers for unknown directive")
	})

	t.Run("GeneratorsFor returns sorted generator names", func(t *testing.T) {
		t.Parallel()
		r := directive.NewConsumerRegistry()
		_ = r.Register(directive.Consumer{Directive: "errors", Generator: "suite"})
		_ = r.Register(directive.Consumer{Directive: "errors", Generator: "stub"})
		gens := r.GeneratorsFor("errors")
		testkit.Len(t, gens, 2, "two consumers")
		testkit.Equal(t, gens[0], "stub", "alphabetical order")
		testkit.Equal(t, gens[1], "suite", "alphabetical order")
	})

	t.Run("duplicate (generator, directive) errors", func(t *testing.T) {
		t.Parallel()
		r := directive.NewConsumerRegistry()
		_ = r.Register(directive.Consumer{Directive: "errors", Generator: "stub"})
		err := r.Register(directive.Consumer{Directive: "errors", Generator: "stub"})
		testkit.True(t, err != nil, "duplicate must error")
	})

	t.Run("Directives returns sorted directive names", func(t *testing.T) {
		t.Parallel()
		r := directive.NewConsumerRegistry()
		_ = r.Register(directive.Consumer{Directive: "errors", Generator: "stub"})
		_ = r.Register(directive.Consumer{Directive: "atomic", Generator: "stub"})
		dirs := r.Directives()
		testkit.Len(t, dirs, 2, "two directives have consumers")
		testkit.Equal(t, dirs[0], "atomic", "alphabetical order")
		testkit.Equal(t, dirs[1], "errors", "alphabetical order")
	})

	t.Run("MustRegister panics on duplicate", func(t *testing.T) {
		t.Parallel()
		r := directive.NewConsumerRegistry()
		r.MustRegister(directive.Consumer{Directive: "errors", Generator: "stub"})
		defer func() {
			testkit.True(t, recover() != nil, "duplicate must panic")
		}()
		r.MustRegister(directive.Consumer{Directive: "errors", Generator: "stub"})
	})

	t.Run("DefaultConsumers exposes the package-level registry", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, directive.DefaultConsumers() != nil, "DefaultConsumers non-nil")
	})

	t.Run("RegisterConsumer routes through default registry", func(t *testing.T) {
		t.Parallel()
		const dir = "test-register-consumer"
		const gen = "stub-test-fixture"
		// Idempotent: package-level registries persist across `go test
		// -count=N`, and RegisterConsumer panics on duplicate. Register
		// only on the first run; subsequent runs verify the entry is
		// still routed correctly.
		if len(directive.DefaultConsumers().GeneratorsFor(dir)) == 0 {
			directive.RegisterConsumer(dir, gen)
		}
		gens := directive.DefaultConsumers().GeneratorsFor(dir)
		testkit.Len(t, gens, 1, "registered exactly once")
		testkit.Equal(t, gens[0], gen, "generator name preserved")
	})
}
