// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"path/filepath"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/stub"
	"go.thesmos.sh/testkit/generator/suite"
)

// The framework conformance suites pin the static contract — a stable name,
// deterministic outputs, a well-formed multi-output shape, templates that
// parse, a directive schema nothing else declares.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, suite.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, suite.New(), []plugintest.GeneratorFixture{
			{
				Name:       "annotated interface",
				BuildStore: func(t *testing.T) *sdk.Store { t.Helper(); return mixed(t) },
			},
			{
				Name: "empty store",
				BuildStore: func(t *testing.T) *sdk.Store {
					t.Helper()
					return storefixture.New().Build()
				},
			},
		})
	})
}

// The signature is most of the volume, and it needs no directive. A design that
// counted only classifications emitted four checks for this interface; the
// signature alone owes ten, and the one directive on it adds the eleventh.
func TestSignatureChecks(t *testing.T) {
	t.Parallel()

	t.Run("gives every method a smoke check", func(t *testing.T) {
		t.Parallel()
		// The weakest check and the one that catches the most: a method that
		// panics on a derived value is one no other check in the file reaches.
		for _, name := range []string{"Store", "Validate", "Read"} {
			testkit.True(t, hasCheck(t, name, "smoke"),
				name+" must carry a smoke check")
		}
	})

	t.Run("gives a context-taking method the three context checks", func(t *testing.T) {
		t.Parallel()
		for _, want := range []string{
			"reports a cancelled context",
			"reports an expired deadline",
			"tolerates a nil context",
		} {
			testkit.True(t, hasCheck(t, "Store", want), "Store must carry "+want)
		}
	})

	t.Run("gives a method with no context none of them", func(t *testing.T) {
		t.Parallel()
		// Validate is `func(Payload) error`. Emitting a cancellation check for
		// it would not compile — there is no context to cancel — so the absence
		// is structural rather than an omission.
		for _, unwanted := range []string{
			"reports a cancelled context",
			"reports an expired deadline",
			"tolerates a nil context",
		} {
			testkit.False(t, hasCheck(t, "Validate", unwanted),
				"Validate takes no context, so it cannot carry "+unwanted)
		}
	})

	t.Run("checks the zero value only where there is one to check", func(t *testing.T) {
		t.Parallel()
		// Read returns (Payload, error); Store returns error alone. A
		// zero-value check over a lone error slot compares nothing.
		testkit.True(t, hasCheck(t, "Read", "an error carries the zero value"),
			"Read returns a value beside its error")
		testkit.False(t, hasCheck(t, "Store", "an error carries the zero value"),
			"Store returns no value to compare")
	})

	t.Run("counts ten across three methods", func(t *testing.T) {
		t.Parallel()
		// The number is the point of the design, not an incidental: a harness
		// that asserted one property per declared classification would emit one.
		testkit.Equal(t, contractOf(t).CheckCount(), 10,
			"the signature-derived family is most of the volume")
	})
}

// Every check has to be reachable by name, or a consumer whose subject
// legitimately violates one has no move but to stop running the suite.
func TestCheckPaths(t *testing.T) {
	t.Parallel()

	t.Run("names the drop key as the failure output reads", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasPath(t, "Store/reports a cancelled context"),
			"the drop key is <method>/<subtest>")
	})

	t.Run("gives each check a distinct exported assertion", func(t *testing.T) {
		t.Parallel()
		// Two checks sharing an identifier would collide in the generated file,
		// and the collision is invisible until the backend renders.
		seen := map[string]struct{}{}
		for _, m := range contractOf(t).Methods {
			for _, c := range m.Checks {
				_, dup := seen[c.Func]
				testkit.False(t, dup, "two checks both name "+c.Func)
				seen[c.Func] = struct{}{}
			}
		}
	})
}

// An interface whose method set cannot be completed is refused: a harness over
// part of a contract reports success for an implementation that fails the rest.
func TestIncompleteMethodSet(t *testing.T) {
	t.Parallel()

	t.Run("reports the embed it could not follow", func(t *testing.T) {
		t.Parallel()
		got := plugintest.Generate(t, suite.New(), foreignEmbed(t)).Diagnostics()
		testkit.Len(t, got, 1, "an unresolvable embed is reported once")
	})

	t.Run("generates nothing for it", func(t *testing.T) {
		t.Parallel()
		s := foreignEmbed(t)
		plugintest.Generate(t, suite.New(), s)
		testkit.Len(t, s.Emit().PendingOriginSlots(), 0,
			"a harness over half a contract is worse than none")
	})
}

// contractOf drives the plugin over the fixture and returns the queued harness.
func contractOf(t *testing.T) *suite.Contract {
	t.Helper()
	s := mixed(t)
	plugintest.Generate(t, suite.New(), s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if c, ok := p.Item.(*suite.Contract); ok {
			return c
		}
	}
	t.Fatal("the run queued no contract")
	return nil
}

// hasCheck reports whether the method carries a check reporting under subtest.
func hasCheck(t *testing.T, method, subtest string) bool {
	t.Helper()
	for _, m := range contractOf(t).Methods {
		if m.Name != method {
			continue
		}
		for _, c := range m.Checks {
			if c.Subtest == subtest {
				return true
			}
		}
	}
	return false
}

// hasPath reports whether any check answers to the drop key.
func hasPath(t *testing.T, path string) bool {
	t.Helper()
	for _, m := range contractOf(t).Methods {
		for _, c := range m.Checks {
			if c.Path == path {
				return true
			}
		}
	}
	return false
}

// mixed is the corpus fixture in store form: a writer carrying a mixin, the
// validator it names, and a reader.
//
// The same three methods conformance/corpus/iface/mixin/validates declares, so
// what this asserts about the projection is what the corpus compiles.
func mixed(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("validates", "example.com/validates").
		Struct("Payload", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("validates/iface.go", 1, 1))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("validates/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Validate", func(m *storefixture.MethodBuilder) {
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.PkgNamed("example.com/validates", "Payload"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// foreignEmbed is an annotated interface embedding one the run never loaded.
func foreignEmbed(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("validates", "example.com/validates").
		Interface("Partial", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("validates/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Embed(storefixture.PkgNamed("io", "Closer"))
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// Rendering is where a generator actually fails. Every assertion driven off the
// emit graph passes against a template that produces code which does not
// compile — an undeclared local, a call whose arity disagrees with the source.
// Those surface only once the backend runs, so the templates are driven
// end-to-end here rather than trusted.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("the toolchain accepts what it emits", func(t *testing.T) {
		t.Parallel()
		gen := golangtest.Render(t, backendgolang.New(), mixedPackage(t), suite.New()).
			WithSource(golangtest.GoFile("validates/iface.go", mixedSource())).
			WithRequire(suite.Module, filepath.Join("..", ".."))
		gen.AssertCompiles(t)
		gen.AssertVets(t)
	})
}

// mixedPackage is [mixed] as a package node, for the render path.
func mixedPackage(t *testing.T) *sdk.Package {
	t.Helper()
	for _, p := range mixed(t).Nodes().Packages().Items() {
		return p
	}
	t.Fatal("fixture has no package")
	return nil
}

// mixedSource is the hand-written half the generated harness is compiled
// against, so the two are bound by the compiler rather than by review.
func mixedSource() string {
	return `package validates

import "context"

type Payload struct{ Key, Body string }

type Mixed interface {
	Store(ctx context.Context, v Payload) error
	Validate(v Payload) error
	Read(ctx context.Context, key string) (Payload, error)
}
`
}

// contractIn drives the plugin over a store and returns the queued harness.
func contractIn(t *testing.T, s *sdk.Store) *suite.Contract {
	t.Helper()
	plugintest.Generate(t, suite.New(), s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if c, ok := p.Item.(*suite.Contract); ok {
			return c
		}
	}
	t.Fatal("the run queued no contract")
	return nil
}

// hasCheckIn reports whether the store's harness carries the named check.
func hasCheckIn(t *testing.T, s *sdk.Store, method, subtest string) bool {
	t.Helper()
	for _, m := range contractIn(t, s).Methods {
		if m.Name != method {
			continue
		}
		for _, c := range m.Checks {
			if c.Subtest == subtest {
				return true
			}
		}
	}
	return false
}

// callbackFixture takes a parameter no literal can be written for.
func callbackFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("watch", "example.com/watch").
		Interface("Watcher", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("watch/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Watch", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("fn", storefixture.Func(nil, nil))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// sliceFixture takes a slice, which eidos derives no sample for.
func sliceFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("col/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Slice(storefixture.Named("byte")))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// collidingFixture names one parameter identically across two composite types.
func collidingFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("col/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Slice(storefixture.Named("byte")))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Slice(storefixture.Named("string")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// An interface is opted in by the directive, and a package generally holds more
// than the ones a harness was asked for.
func TestUndirectedInterface(t *testing.T) {
	t.Parallel()

	t.Run("generates nothing for it", func(t *testing.T) {
		t.Parallel()
		s := undirected(t)
		plugintest.Generate(t, suite.New(), s)
		testkit.Len(t, s.Emit().PendingOriginSlots(), 0,
			"a harness is generated where one is declared")
	})
}

// A directive on an interface declaring nothing asks for a harness that would
// assert nothing at all.
func TestEmptyInterface(t *testing.T) {
	t.Parallel()

	t.Run("reports it", func(t *testing.T) {
		t.Parallel()
		got := plugintest.Generate(t, suite.New(), emptyIface(t)).Diagnostics()
		testkit.Len(t, got, 1, "an interface with no method is reported once")
		testkit.Contains(t, got[0].Message, "declares no method", "and named for what is wrong")
	})
}

// Swallowing a failed append reads downstream as an interface nobody annotated
// rather than as a fault, and the harness is this generator's whole output.
func TestGenerateReportsAFailedAppend(t *testing.T) {
	t.Parallel()

	s := mixed(t)
	// Freezing from outside the pipeline stands in for the real cause: an
	// append arriving after Layout has closed the graph.
	s.Emit().Freeze()

	err := suite.New().Generate(&sdk.GeneratorContext{
		Store: s, Reader: sdk.NewStoreReader(s), Diag: sdk.NewSink(),
	})

	testkit.Error(t, err, "a failed append must surface")
	testkit.Contains(t, err.Error(), "Mixed", "the error must name the declaration")
}

// undirected declares an interface carrying no directive.
func undirected(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Interface("Internal", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("cfg/iface.go", 1, 1))
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// emptyIface carries the directive and declares nothing for it to cover.
func emptyIface(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Interface("Empty", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("cfg/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
		}).
		Build()
}

// An embed that contributed nothing is reported by why, because the two causes
// call for different action: an unloaded package a wider run would reach, and a
// source defect no run repairs.
func TestEmbedIssues(t *testing.T) {
	t.Parallel()

	t.Run("reports a non-interface embed as an error", func(t *testing.T) {
		t.Parallel()
		got := plugintest.Generate(t, suite.New(), structEmbed(t)).Diagnostics()
		testkit.Len(t, got, 1, "an embed that is not an interface is reported")
		testkit.Contains(t, got[0].Message, "nothing is generated",
			"and the harness is withheld rather than covering half a contract")
	})
}

// A harness runs itself through the double where one was queued for the same
// interface, which is read off the double rather than off the directive.
func TestDoubleFromTheQueue(t *testing.T) {
	t.Parallel()

	t.Run("names the identifiers the double emitted", func(t *testing.T) {
		t.Parallel()
		// Composed by the stub generator and read here, so the harness cannot
		// name a constructor the double never wrote.
		got := contractIn(t, withDouble(t)).Double
		testkit.True(t, got != nil, "a queued double is found")
		testkit.Equal(t, got.CtorName, "NewMixedStub", "the constructor comes off the queue")
		testkit.Equal(t, got.DelegateToName, "MixedStubDelegateTo", "and so does the wrapper")
	})

	t.Run("is absent where nothing queued one", func(t *testing.T) {
		t.Parallel()
		// A directive says what was asked for; the queue says what was
		// produced. An interface whose method set the double generator could
		// not complete leaves nothing to run through.
		testkit.True(t, contractIn(t, mixed(t)).Double == nil,
			"no double, no second run")
	})
}

// structEmbed embeds something that is not an interface, which no wider run
// repairs.
func structEmbed(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Meta", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/iface.go", 1, 1))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		Interface("Broken", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("cfg/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Embed(storefixture.PkgNamed("example.com/cfg", "Meta"))
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// withDouble drives the double generator first, so its emit value is on the
// queue when the harness looks for one — which is the ordering the buckets give
// a real run.
func withDouble(t *testing.T) *sdk.Store {
	t.Helper()
	s := mixed(t)
	for _, iface := range s.Nodes().Interfaces().Items() {
		iface.DirectiveList = append(iface.DirectiveList, storefixture.Directive("stub"))
	}
	plugintest.Generate(t, stub.New(), s)
	return s
}
