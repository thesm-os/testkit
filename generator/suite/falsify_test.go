// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readernoerror"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/stub"
	"go.thesmos.sh/testkit/generator/suite"
)

// The companion output drives every generated check against a stand-in built to
// violate it, so a check that cannot fail is a build failure rather than a
// quiet line of coverage.
//
// What is asserted here is the projection. That the guards compile and reject
// is the corpus's job, where 84 of these files run against real signatures —
// nothing at this level can tell a guard that rejects from one that only looks
// as though it would.
func TestFalsification(t *testing.T) {
	t.Parallel()

	t.Run("proves every check with a derivable violator", func(t *testing.T) {
		t.Parallel()
		f := falsificationIn(t, falsifiableStore(t))
		testkit.Equal(t, guardNames(f), []string{
			"TestAssertStorePutSmokeCanFail",
			"TestAssertStorePutCancelsCanFail",
			"TestAssertStorePutHonoursDeadlineCanFail",
			"TestAssertStorePutToleratesNilContextCanFail",
		}, "one guard per check, in harness order")
	})

	t.Run("suffixes the guard so the family is selectable", func(t *testing.T) {
		t.Parallel()
		// `go test -run CanFail` is what makes the proof runnable as a stage of
		// its own, and a prefix would not select it.
		for _, name := range guardNames(falsificationIn(t, falsifiableStore(t))) {
			testkit.Assert(t, name).HasSuffix("CanFail", "the family shares a suffix")
		}
	})

	t.Run("keys each guard on its own template", func(t *testing.T) {
		t.Parallel()
		// The backend resolves a template by the emit kind's string value, and
		// a kind it cannot find renders nothing and fails nowhere — so the file
		// would simply come out short.
		f := falsificationIn(t, falsifiableStore(t))
		testkit.Equal(t, f.Cases[0].Kind(), suite.KindViolateSmoke,
			"the first check's guard renders through the smoke violator")
	})

	t.Run("configures both methods a spanning violation needs", func(t *testing.T) {
		t.Parallel()
		// A store that ignores partitions files through one method and answers
		// through the other, so overriding the write alone leaves the read
		// still isolating — and the guard would prove nothing.
		f := falsificationIn(t, spanningStore(t))
		var partition *suite.Violation
		for _, c := range f.Cases {
			if c.KindName == suite.KindViolatePartition {
				partition = c
			}
		}
		testkit.True(t, partition != nil, "the partition check has a violator")
		testkit.Equal(t, partition.Option, "WithStorePut", "the write is overridden")
		testkit.Equal(t, partition.PartnerOption, "WithStoreRead", "and so is the read")
	})

	t.Run("names what a partner's stream carries", func(t *testing.T) {
		t.Parallel()
		// The violator makes a channel, and the element is in the ref's type
		// arguments — which a template has no way to reach into.
		f := falsificationIn(t, streamingStore(t))
		var outbox *suite.Violation
		for _, c := range f.Cases {
			if c.KindName == suite.KindViolateOutbox {
				outbox = c
			}
		}
		testkit.True(t, outbox != nil, "the outbox check has a violator")
		testkit.True(t, outbox.StreamElem != nil, "which knows what to make a channel of")
	})

	t.Run("routes to the test output", func(t *testing.T) {
		t.Parallel()
		// The suffix earns Layout's external-test-package shift, which is what
		// makes a guard reach the harness the way a consumer does.
		testkit.Equal(t, falsificationIn(t, falsifiableStore(t)).OutputTag(), suite.GoTestOutputTag,
			"the companion is routed apart from the harness")
	})

	t.Run("names the checks it has no proof for yet", func(t *testing.T) {
		t.Parallel()
		// Stated rather than omitted: a file silent about a check is
		// indistinguishable from one where the generator failed to handle it.
		// And the reason names what is missing rather than claiming the check
		// is unbreakable — a stand-in answers from a closure, so there is none
		// that could not be built.
		f := falsificationIn(t, unprovenStore(t))
		testkit.Len(t, f.Unproven, 1, "the miss check is named")
		testkit.Assert(t, f.Unproven[0].Func).Contains("ReportsAMiss", "by the check it is about")
		testkit.Assert(t, f.Unproven[0].Reason).Contains("no literal can be written",
			"with the piece a violator would still need")
	})

	t.Run("fetches the fixture only where a guard reads it", func(t *testing.T) {
		t.Parallel()
		// Go refuses a declared-and-unused local, and a method taking nothing
		// after its context is handed nothing.
		f := falsificationIn(t, parameterlessStore(t))
		for _, c := range f.Cases {
			testkit.False(t, c.NeedsFixture(), "a parameterless method reads no derived value")
		}
	})
}

// Layout decides where the harness landed, and every symbol a guard names lives
// there — so the package is corrected after routing rather than guessed during
// Generate.
func TestFalsificationTakesItsPackageFromLayout(t *testing.T) {
	t.Parallel()

	t.Run("repoints every guard", func(t *testing.T) {
		t.Parallel()
		f := falsificationIn(t, falsifiableStore(t))
		f.SetOutputPackages(map[string]string{"": "example.com/miss/misstest"})

		testkit.Equal(t, f.HarnessPkg, "example.com/miss/misstest", "the file takes the harness's package")
		for _, c := range f.Cases {
			testkit.Equal(t, c.HarnessPkg, "example.com/miss/misstest", "and so does every guard")
		}
	})

	t.Run("leaves the provisional package where routing failed", func(t *testing.T) {
		t.Parallel()
		// A run that recorded routing errors reaches dispatch with some tags
		// missing, and the primary's entry can be present-but-empty. A wrong
		// package is a compile error naming the symbol; an empty one would
		// compose a bare name that binds to whatever else is in scope.
		for _, byTag := range []map[string]string{{}, {"test": "x"}, {"": ""}} {
			f := falsificationIn(t, falsifiableStore(t))
			before := f.HarnessPkg
			f.SetOutputPackages(byTag)
			testkit.Equal(t, f.HarnessPkg, before, "a partial map changes nothing")
		}
	})
}

// A guard configures the generated stand-in and is a Test function, so an
// interface with neither gets no companion — and the harness says which.
func TestUnfalsifiableInterface(t *testing.T) {
	t.Parallel()

	t.Run("declines an interface with no double", func(t *testing.T) {
		t.Parallel()
		c := contractIn(t, undoubledStore(t))
		testkit.Assert(t, c.Unfalsifiable).Contains("no //testkit:stub",
			"the harness names why nothing proves its checks, in the source's own terms")
		testkit.False(t, hasFalsification(t, undoubledStore(t)), "and no companion is queued")
	})

	t.Run("declines a generic interface", func(t *testing.T) {
		t.Parallel()
		// A guard is a Test function, which cannot carry type arguments, and
		// nothing in the source names a concrete instantiation.
		c := contractIn(t, genericStore(t))
		testkit.Assert(t, c.Unfalsifiable).Contains("the interface is generic",
			"the harness names why nothing proves its checks")
		testkit.False(t, hasFalsification(t, genericStore(t)), "and no companion is queued")
	})
}

// falsificationIn drives the plugin over a store and returns the queued
// companion, failing when none was.
func falsificationIn(t *testing.T, s *sdk.Store) *suite.Falsification {
	t.Helper()
	plugintest.Generate(t, suite.New(), s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if f, ok := p.Item.(*suite.Falsification); ok {
			return f
		}
	}
	t.Fatal("the run queued no companion output")
	return nil
}

// hasFalsification reports whether the run queued a companion at all.
func hasFalsification(t *testing.T, s *sdk.Store) bool {
	t.Helper()
	plugintest.Generate(t, suite.New(), s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if _, ok := p.Item.(*suite.Falsification); ok {
			return true
		}
	}
	return false
}

// guardNames returns each case's Test function, in order.
func guardNames(f *suite.Falsification) []string {
	out := make([]string, 0, len(f.Cases))
	for _, c := range f.Cases {
		out = append(out, c.TestName)
	}
	return out
}

// falsifiableStore is one writer with a double, which is the ordinary case.
func falsifiableStore(t *testing.T) *sdk.Store {
	t.Helper()
	return doubledStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.Named("string"))
			m.Return(storefixture.Named("error"))
		})
	})
}

// spanningStore carries a partition mixin, whose violation is visible only
// across the write and the read together.
func spanningStore(t *testing.T) *sdk.Store {
	t.Helper()
	s := doubledStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("part", storefixture.Named("string"))
			m.Param("key", storefixture.Named("string"))
			m.Param("value", storefixture.Named("string"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Read", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("part", storefixture.Named("string"))
			m.Param("key", storefixture.Named("string"))
			m.Return(storefixture.Named("string"))
			m.Return(storefixture.Named("error"))
		})
	})
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Put" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{suite.MixinPartition}, "test")
			shape.MixinParamKey(suite.MixinPartition, suite.MixinPartitionRead).
				Set(bag, "Read", "test")
			shape.MixinParamKey(suite.MixinPartition, suite.MixinPartitionAxis).
				Set(bag, "part", "test")
		}
	}
	return s
}

// streamingStore carries the outbox contract, whose violator hands back a
// channel nothing arrives on.
func streamingStore(t *testing.T) *sdk.Store {
	t.Helper()
	s := doubledStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Append", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.Named("string"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Subscribe", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.RecvChan(storefixture.Named("string")))
			m.Return(storefixture.Named("error"))
		})
	})
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Append" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaContracts.Set(bag, []string{suite.ContractOutbox}, "test")
			shape.ContractRoleKey(suite.ContractOutbox).Set(bag, suite.ContractOutboxRole, "test")
			shape.ContractPartnerKey(suite.ContractOutbox, suite.ContractOutboxPartner).
				Set(bag, "example.com/miss.Store.Subscribe", "test")
		}
	}
	return s
}

// parameterlessStore takes nothing after its context, so no guard reads the
// fixture.
func parameterlessStore(t *testing.T) *sdk.Store {
	t.Helper()
	return doubledStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Close", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.Named("error"))
		})
	})
}

// unprovenStore carries a miss check whose return admits no literal, so the
// stand-in has nothing believable to answer with.
func unprovenStore(t *testing.T) *sdk.Store {
	t.Helper()
	s := doubledStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Load", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("key", storefixture.Named("string"))
			m.Return(storefixture.Func(nil, nil))
		})
	})
	stamp(s, "Load", readernoerror.Name)
	return s
}

// undoubledStore declares no stub, so there is nothing to make behave badly.
func undoubledStore(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("miss", "example.com/miss").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// genericStore carries type parameters, which a Test function cannot.
func genericStore(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("stub"))
			i.TypeParam("V", nil)
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.TypeParamRef("V"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	plugintest.Generate(t, stub.New(), s)
	return s
}

// doubledStore is an opted-in interface whose double has already been queued.
//
// The stub generator is run first, because the harness reads the double off the
// emit queue rather than off the directive: a directive says what was asked
// for, and the queue says what was produced.
func doubledStore(t *testing.T, methods func(*storefixture.InterfaceBuilder)) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("stub"))
			methods(i)
		}).
		Build()
	plugintest.Generate(t, stub.New(), s)
	return s
}

// Every check this generator emits has a stand-in written to break it.
//
// The claim the whole companion output rests on, and the one nothing else
// makes: a kind added without a violator produces a harness whose new check
// nobody ever falsified, and the file it would have been named in is silent
// because the kind is not in the table either.
func TestEveryCheckKindHasAViolator(t *testing.T) {
	t.Parallel()

	for _, kind := range suite.CheckKinds() {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			violation, known := suite.ViolatorFor(kind)
			testkit.True(t, known, string(kind)+" has a stand-in that breaks it")
			testkit.Assert(t, string(violation)).HasPrefix("suite.violate.",
				"and it renders through a violator template")
		})
	}
}

// Every violator names a template the plugin ships.
//
// A kind the backend cannot resolve renders nothing and fails nowhere, so the
// guard would simply be absent from the file — which is the shape of failure
// the companion exists to remove, arriving in the companion itself.
func TestEveryViolatorShipsATemplate(t *testing.T) {
	t.Parallel()

	tree, ok := suite.New().Templates(golang.Language)
	testkit.True(t, ok, "the plugin reports a Go template tree")

	for _, kind := range suite.CheckKinds() {
		violation, _ := suite.ViolatorFor(kind)
		name := "violate/" + string(violation) + ".tmpl"
		_, err := fs.Stat(tree, name)
		testkit.NoError(t, err, name+" ships")
	}
}
