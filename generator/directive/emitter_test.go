// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestEmitterRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register stores mixin emission pairs", func(t *testing.T) {
		t.Parallel()
		r := directive.NewEmitterRegistry()
		testkit.NoError(t, r.Register(directive.Emission{Directive: "atomic", Generator: "suite"}), "suite atomic")
		testkit.NoError(t, r.Register(directive.Emission{Directive: "atomic", Generator: "bench"}), "bench atomic")
		testkit.Len(t, r.Emitters("atomic"), 2, "two generators emit for atomic")
		testkit.Len(t, r.Emitters("missing"), 0, "no emitters for unknown directive")
	})

	t.Run("Emitters returns sorted by generator name", func(t *testing.T) {
		t.Parallel()
		r := directive.NewEmitterRegistry()
		_ = r.Register(directive.Emission{Directive: "atomic", Generator: "suite"})
		_ = r.Register(directive.Emission{Directive: "atomic", Generator: "bench"})
		em := r.Emitters("atomic")
		testkit.Equal(t, em[0].Generator, "bench", "alphabetical order")
		testkit.Equal(t, em[1].Generator, "suite", "alphabetical order")
	})

	t.Run("duplicate (generator, directive) errors", func(t *testing.T) {
		t.Parallel()
		r := directive.NewEmitterRegistry()
		_ = r.Register(directive.Emission{Directive: "atomic", Generator: "suite"})
		err := r.Register(directive.Emission{Directive: "atomic", Generator: "suite"})
		testkit.True(t, err != nil, "duplicate must error")
	})

	t.Run("MustRegister panics on duplicate", func(t *testing.T) {
		t.Parallel()
		r := directive.NewEmitterRegistry()
		r.MustRegister(directive.Emission{Directive: "atomic", Generator: "suite"})
		defer func() {
			testkit.True(t, recover() != nil, "duplicate must panic")
		}()
		r.MustRegister(directive.Emission{Directive: "atomic", Generator: "suite"})
	})

	t.Run("Directives returns sorted directive names", func(t *testing.T) {
		t.Parallel()
		r := directive.NewEmitterRegistry()
		_ = r.Register(directive.Emission{Directive: "bounded", Generator: "suite"})
		_ = r.Register(directive.Emission{Directive: "atomic", Generator: "suite"})
		dirs := r.Directives()
		testkit.Len(t, dirs, 2, "two directives")
		testkit.Equal(t, dirs[0], "atomic", "alphabetical")
		testkit.Equal(t, dirs[1], "bounded", "alphabetical")
	})

	t.Run("DefaultEmitters exposes the package-level registry", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, directive.DefaultEmitters() != nil, "non-nil")
	})

	t.Run("RegisterEmitter routes through default registry", func(t *testing.T) {
		t.Parallel()
		const dir = "test-register-emitter"
		const gen = "suite-test-fixture"
		// Idempotent: package-level registries persist across `go test
		// -count=N`, and RegisterEmitter panics on duplicate. Register
		// only on the first run; subsequent runs verify the entry is
		// still routed correctly.
		if len(directive.DefaultEmitters().Emitters(dir)) == 0 {
			directive.RegisterEmitter(dir, gen)
		}
		em := directive.DefaultEmitters().Emitters(dir)
		testkit.Len(t, em, 1, "registered exactly once")
		testkit.Equal(t, em[0].Generator, gen, "generator name preserved")
	})
}
