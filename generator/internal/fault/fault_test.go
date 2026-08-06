// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/fault"
)

// The framework contracts every plugin owes, plus the annotator contracts: no
// panic on an empty store, no change to the source graph, and idempotent
// stamping — running twice must leave the same metadata as running once.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, fault.New())
	})

	t.Run("satisfies the annotator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunAnnotatorSuite(t, fault.New(), []plugintest.AnnotatorFixture{
			{
				Name:       "method with sentinels and keys",
				BuildStore: func(t *testing.T) *store.Store { t.Helper(); return fixture(directives()...) },
			},
			{
				Name: "method with no fault directive",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return fixture()
				},
			},
		})
	})
}

func TestAnnotate(t *testing.T) {
	t.Parallel()

	t.Run("stamps sentinels in the order written", func(t *testing.T) {
		t.Parallel()
		m := annotated(t, dir("ErrNotFound", "ErrGone"))
		testkit.Equal(t, fault.Sentinels(m.Meta())[0], "ErrNotFound", "first sentinel")
		testkit.Equal(t, fault.Sentinels(m.Meta())[1], "ErrGone", "second sentinel")
	})

	t.Run("unions sentinels across repeated directives", func(t *testing.T) {
		t.Parallel()
		// One line per concern is how a method with several sentinels stays
		// readable, so repetition has to accumulate rather than overwrite.
		m := annotated(t, dir("ErrNotFound"), dir("ErrGone"))
		testkit.Len(t, fault.Sentinels(m.Meta()), 2, "repeated directives accumulate")
	})

	t.Run("stamps nothing when no directive is present", func(t *testing.T) {
		t.Parallel()
		m := annotated(t)
		testkit.Assert(t, fault.Sentinels(m.Meta())).IsEmpty("an unannotated method configures nothing")
	})

	t.Run("ignores directives it does not own", func(t *testing.T) {
		t.Parallel()
		// Methods carry directives from several plugins — a shape mixin sits
		// beside a fault line on the same method — so the annotator has to
		// walk past the ones that are not its own rather than misread them.
		m := annotated(t, &directive.Directive{Name: "mixin", Args: []string{"errors"}}, dir("ErrNotFound"))
		testkit.Len(t, fault.Sentinels(m.Meta()), 1, "a foreign directive is skipped, not parsed")
	})

	t.Run("stamps the retry attempt", func(t *testing.T) {
		t.Parallel()
		m := annotated(t, withKeys(fault.RetryKey, "3"))
		testkit.Equal(t, fault.Retry(m.Meta()), 3, "retry attempt")
	})

	t.Run("stamps the partition field", func(t *testing.T) {
		t.Parallel()
		m := annotated(t, withKeys(fault.PartitionKey, "RunID"))
		testkit.Equal(t, fault.Partition(m.Meta()), "RunID", "partition field")
	})

	t.Run("takes the last value when a key repeats", func(t *testing.T) {
		t.Parallel()
		m := annotated(t, withKeys(fault.RetryKey, "3"), withKeys(fault.RetryKey, "5"))
		testkit.Equal(t, fault.Retry(m.Meta()), 5, "the later line wins")
	})

	t.Run("reports a retry count that does not parse", func(t *testing.T) {
		t.Parallel()
		// Guessing zero would read as "no retry configured" — the one answer
		// indistinguishable from the directive being absent.
		d := diag.New()
		annotateWith(t, d, withKeys(fault.RetryKey, "soon"))
		testkit.True(t, d.HasErrors(), "an unparseable retry count must be reported")
	})

	t.Run("reports a retry count below one", func(t *testing.T) {
		t.Parallel()
		d := diag.New()
		annotateWith(t, d, withKeys(fault.RetryKey, "0"))
		testkit.True(t, d.HasErrors(), "a retry count must name a real attempt")
	})

	t.Run("stamps no retry when the count is rejected", func(t *testing.T) {
		t.Parallel()
		m := annotated(t, withKeys(fault.RetryKey, "soon"))
		testkit.Equal(t, fault.Retry(m.Meta()), 0, "a rejected count configures nothing")
	})

	t.Run("reports two sentinels that generate the same helper", func(t *testing.T) {
		t.Parallel()
		// `ErrNotFound` and `NotFound` both want FaultNotFound, and the
		// generated file would not compile — with the compiler blaming
		// generated code rather than the directive that caused it.
		d := diag.New()
		annotateWith(t, d, dir("ErrNotFound", "NotFound"))
		testkit.True(t, d.HasErrors(), "a helper-name collision must be reported")
	})

	t.Run("keeps only the first of two colliding sentinels", func(t *testing.T) {
		t.Parallel()
		m := annotated(t, dir("ErrNotFound", "NotFound"))
		testkit.Len(t, fault.Sentinels(m.Meta()), 1, "the collision is dropped, not duplicated")
	})
}

func TestHelper(t *testing.T) {
	t.Parallel()

	t.Run("strips the Err prefix", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fault.Helper("ErrNotFound"), "FaultNotFound", "the helper names the action")
	})

	t.Run("leaves a name without the prefix whole", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fault.Helper("Timeout"), "FaultTimeout", "no prefix to strip")
	})
}

// The accessors are read against nodes that may never have been annotated —
// a generator asks every method, not only the configured ones.
func TestAccessorsTolerateAnAbsentBag(t *testing.T) {
	t.Parallel()

	t.Run("sentinels", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, fault.Sentinels(nil)).IsEmpty("a nil bag configures nothing")
	})

	t.Run("retry", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fault.Retry(nil), 0, "a nil bag configures nothing")
	})

	t.Run("partition", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fault.Partition(nil), "", "a nil bag configures nothing")
	})
}

// dir builds one fault directive carrying positional sentinels.
func dir(sentinels ...string) *directive.Directive {
	return &directive.Directive{Name: directive.Name(fault.DirectiveName), Args: sentinels}
}

// withKeys builds one fault directive carrying a single key.
func withKeys(key, value string) *directive.Directive {
	return &directive.Directive{
		Name: directive.Name(fault.DirectiveName),
		KV:   map[string]string{key: value},
	}
}

// directives is the canonical mixed fixture: sentinels plus both keys.
func directives() []*directive.Directive {
	return []*directive.Directive{
		dir("ErrNotFound", "ErrGone"),
		withKeys(fault.RetryKey, "3"),
		withKeys(fault.PartitionKey, "RunID"),
	}
}

// fixture returns a store holding one interface whose Get method carries the
// supplied directives.
func fixture(dirs ...*directive.Directive) *store.Store {
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
				for _, d := range dirs {
					m.Directive(d)
				}
			})
		}).
		Build()
}

// annotated runs the annotator over a fixture carrying dirs and returns the
// annotated method.
func annotated(t *testing.T, dirs ...*directive.Directive) *node.Method {
	t.Helper()
	return annotateWith(t, diag.New(), dirs...)
}

// annotateWith is annotated with a caller-supplied sink, for the cases that
// assert on what was reported.
func annotateWith(t *testing.T, d *diag.Sink, dirs ...*directive.Directive) *node.Method {
	t.Helper()

	s := fixture(dirs...)
	if err := fault.New().Annotate(&plugin.AnnotatorContext{
		Store: s, Reader: store.NewReader(s), Diag: d,
	}); err != nil {
		t.Fatalf("Annotate: %v", err)
	}

	ifaces := s.Nodes().Interfaces().Items()
	if len(ifaces) != 1 || len(ifaces[0].Methods) != 1 {
		t.Fatalf("fixture shape changed: %d interfaces", len(ifaces))
	}
	return ifaces[0].Methods[0]
}
