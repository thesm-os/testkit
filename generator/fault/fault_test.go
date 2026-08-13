// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"path/filepath"
	"strings"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/fault"
	"go.thesmos.sh/testkit/generator/stub"
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
				BuildStore: func(t *testing.T) *sdk.Store { t.Helper(); return fixture(directives()...) },
			},
			{
				Name: "method with no fault directive",
				BuildStore: func(t *testing.T) *sdk.Store {
					t.Helper()
					return fixture()
				},
			},
		})
	})

	// This plugin is a generator as well as an annotator, and the generator
	// half was the one going unchecked — every sibling ran this suite and fault
	// did not. Four of its checks are live risks here rather than formalities:
	// output ordering runs through a map keyed by node, the emit base is derived
	// rather than built, the output tag is stamped rather than declared, and
	// Generate walks the source graph while writing diagnostics.
	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, fault.New(), []plugintest.GeneratorFixture{
			{
				Name:       "method with sentinels and keys",
				BuildStore: func(t *testing.T) *sdk.Store { t.Helper(); return fixture(directives()...) },
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

// A stale stamp is worse than a stale file: every generator reading it
// inherits the staleness, and none of them can tell.
func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("declares a non-empty version", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, fault.New().Version()).
			IsNotEmpty("an annotator without a version cannot invalidate its cache")
	})

	t.Run("reports the declared constant", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fault.New().Version(), fault.Version, "the method reports the constant")
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
		m := annotated(t, &sdk.Directive{Name: "mixin", Args: []string{"errors"}}, dir("ErrNotFound"))
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
		_, d := annotate(t, withKeys(fault.RetryKey, "soon"))
		testkit.True(t, d.HasErrors(), "an unparseable retry count must be reported")
	})

	t.Run("reports a retry count below one", func(t *testing.T) {
		t.Parallel()
		_, d := annotate(t, withKeys(fault.RetryKey, "0"))
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
		_, d := annotate(t, dir("ErrNotFound", "NotFound"))
		testkit.True(t, d.HasErrors(), "a helper-name collision must be reported")
	})

	t.Run("reports an unexported sentinel", func(t *testing.T) {
		t.Parallel()
		// A double is routed out of the package that declares the sentinel, so
		// an unexported name is one the generated file cannot spell. Emitting
		// it fails in the consumer's compiler against code they did not write.
		_, d := annotate(t, dir("errNotFound"))
		testkit.True(t, d.HasErrors(), "an unexported sentinel must be reported")
	})

	t.Run("stamps nothing for an unexported sentinel", func(t *testing.T) {
		t.Parallel()
		// Dropped rather than carried: it also defeats the collision guard,
		// because Helper strips an `Err` prefix the lowercase spelling does not
		// have — so `errNotFound` and `ErrNotFound` would generate two helpers
		// and read as no collision at all.
		m := annotated(t, dir("errNotFound"))
		testkit.Assert(t, fault.Sentinels(m.Meta())).IsEmpty("a refused sentinel leaves no stamp")
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

// The rendered filenames, composed by Layout from the source basename and the
// suffixes both plugins declare.
const (
	primaryFile   = "store" + stub.GoPrimarySuffix
	companionFile = "store" + stub.GoTestSuffix
)

// The guard is the whole reason this plugin reads the emit queue rather than
// the source directive. Layout materialises a slot contribution through a
// lookup-or-create, so a contributor that emits where its host did not writes
// the file on its own — methods hanging off types nothing declared, in a file
// nobody asked for.
func TestGenerateWithoutAHost(t *testing.T) {
	t.Parallel()

	t.Run("writes no file when the interface declares no double", func(t *testing.T) {
		t.Parallel()
		render(t, faulted(t, false)).AssertFileCount(0)
	})

	t.Run("skips an unhosted interface beside a hosted one", func(t *testing.T) {
		t.Parallel()
		// A run generally doubles some interfaces and not others, so the walk
		// has to pass over an unhosted one rather than stop at it.
		render(t, mixedHosting(t)).AssertFileCount(2)
	})

	t.Run("reports no diagnostic when it declines to contribute", func(t *testing.T) {
		t.Parallel()
		// Declining is the correct outcome, not a degraded one: a method may
		// carry fault configuration for the suite and model tiers without any
		// double being generated for its interface.
		p := render(t, faulted(t, false))
		testkit.Len(t, p.Diagnostics().Diagnostics(), 0,
			"an unhosted contribution is silent, not an error")
	})
}

// Landing in the host's file is what the matching output suffixes buy, and it
// is invisible until something checks the file count: a drifted suffix writes a
// second file rather than failing.
func TestGenerateIntoTheHostsFile(t *testing.T) {
	t.Parallel()

	t.Run("writes no file of its own", func(t *testing.T) {
		t.Parallel()
		// Two, not four: the double and its companion, each carrying both
		// plugins' contributions.
		render(t, faulted(t, true)).AssertFileCount(2)
	})

	t.Run("renders the helpers after the types they hang off", func(t *testing.T) {
		t.Parallel()
		// The `bottom` slot is what guarantees this. Ordering inside one slot
		// follows resolved plugin order, where this plugin's first appearance
		// is as an annotator — ahead of its own host.
		//
		// Asserted over the declarations rather than over byte offsets of a
		// marker comment: the comment is prose this plugin is free to reword,
		// and a claim about where a declaration sits should not break when it
		// is. It also stops silently passing on a run that emitted neither.
		golangtest.Rendered(t, render(t, faulted(t, true))).
			Suffixed(t, stub.GoPrimarySuffix).
			AssertOrder(t, "StoreGetStub", "FaultNotFound")
	})

	t.Run("puts the checks in the external test package", func(t *testing.T) {
		t.Parallel()
		render(t, faulted(t, true)).AssertFile(companionFile).
			Contains("package storetest_test")
	})
}

// What each directive key renders is the plugin's whole output, and every one
// of these is reachable only through a template — an emit-graph assertion would
// pass against a template that produced code which does not compile.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("renders a one-shot helper per sentinel", func(t *testing.T) {
		t.Parallel()
		f := render(t, faulted(t, true)).AssertFile(primaryFile)
		f.Contains("func (s *StoreGetStub) FaultNotFound() *StoreGetStub")
		f.Contains("func (s *StoreGetStub) FaultGone() *StoreGetStub")
	})

	t.Run("qualifies the sentinel against its source package", func(t *testing.T) {
		t.Parallel()
		// The double is routed into its own package in the ordinary case, where
		// the sentinel is not reachable unqualified.
		render(t, faulted(t, true)).AssertFile(primaryFile).
			Contains("s.Faults(storepkg.ErrNotFound, 1)")
	})

	t.Run("renders the retry schedule", func(t *testing.T) {
		t.Parallel()
		render(t, faulted(t, true)).AssertFile(primaryFile).
			Contains("func (s *StoreGetStub) RetrySchedule(err error) *StoreGetStub")
	})

	t.Run("types the partition key by the field it names", func(t *testing.T) {
		t.Parallel()
		// The key's type is whichever parameter the recorded-call field came
		// from, so a caller passes what the method takes.
		render(t, faulted(t, true)).AssertFile(primaryFile).
			Contains("func (s *StoreGetStub) FaultForPartition(key string, err error, n int) *StoreGetStub")
	})

	t.Run("renders both directions of the partition helper", func(t *testing.T) {
		t.Parallel()
		render(t, faulted(t, true)).AssertFile(primaryFile).
			Contains("func (s *StoreGetStub) FaultForOtherPartitions(key string, err error, n int) *StoreGetStub")
	})

	t.Run("checks each helper it generated", func(t *testing.T) {
		t.Parallel()
		f := render(t, faulted(t, true)).AssertFile(companionFile)
		f.Contains("func TestStoreStubGetFaults(t *testing.T)")
		f.Contains("s.OnGet.FaultNotFound()")
		f.Contains("s.OnGet.RetrySchedule(want)")
		f.Contains("s.OnGet.FaultForPartition(key, want, 1)")
	})

	t.Run("leaves a method configuring nothing alone", func(t *testing.T) {
		t.Parallel()
		// Close carries no fault directive, so it earns no helpers and no
		// check — and a method with none must not contribute blank lines to
		// somebody else's file.
		f := render(t, faulted(t, true)).AssertFile(companionFile)
		testkit.False(t, strings.Contains(f.String(), "TestStoreStubCloseFaults"),
			"a method with no fault configuration generates no checks")
	})
}

// A partition naming no parameter would render a helper with no type in key's
// position — code that does not compile, blamed on the generator rather than on
// the directive that asked for it.
func TestGenerateRejectsAnUnknownPartition(t *testing.T) {
	t.Parallel()

	t.Run("reports the directive", func(t *testing.T) {
		t.Parallel()
		got := render(t, badPartition(t)).Diagnostics().Diagnostics()
		testkit.Len(t, got, 1, "an unresolvable partition is reported once")
	})

	t.Run("names the field it could not resolve", func(t *testing.T) {
		t.Parallel()
		got := render(t, badPartition(t)).Diagnostics().Diagnostics()
		testkit.Contains(t, got[0].Message, "Nonexistent",
			"the diagnostic names the field the directive asked for")
	})

	t.Run("emits no partition helper for it", func(t *testing.T) {
		t.Parallel()
		// Dropped rather than guessed at: the run fails on the diagnostic, and
		// a half-rendered helper would fail again in the consumer's compiler.
		body := render(t, badPartition(t)).AssertFile(primaryFile).String()
		testkit.False(t, strings.Contains(body, "FaultForPartition"),
			"an unresolvable partition renders nothing")
	})
}

// The companion always lands in an external test package, so it reaches
// neither the double nor its constructor unqualified — and neither package is
// known until Layout has routed the double.
func TestSetOutputPackages(t *testing.T) {
	t.Parallel()

	t.Run("repoints the double at where it was routed", func(t *testing.T) {
		t.Parallel()
		tests := &fault.Tests{TypeName: "StoreStub"}
		tests.SetOutputPackages(map[string]string{"": "example.com/storepkg/storetest"})
		testkit.Equal(t, tests.StubRef.Pkg, "example.com/storepkg/storetest",
			"the reference follows the double's routing")
	})

	t.Run("repoints the constructor with it", func(t *testing.T) {
		t.Parallel()
		tests := &fault.Tests{TypeName: "StoreStub"}
		tests.SetOutputPackages(map[string]string{"": "example.com/storepkg/storetest"})
		testkit.Equal(t, tests.CtorRef.Name, "NewStoreStub",
			"the constructor lives beside the double and routes with it")
	})

	t.Run("keeps the provisional reference when routing derived no path", func(t *testing.T) {
		t.Parallel()
		// Centralised layout resolves without a derivable import path. A wrong
		// package is a compile error naming the symbol; a bare name silently
		// binds to whatever else is in scope.
		tests := &fault.Tests{TypeName: "StoreStub", StubRef: sdk.NewExternal("example.com/storepkg", "StoreStub")}
		tests.SetOutputPackages(map[string]string{"": ""})
		testkit.Equal(t, tests.StubRef.Pkg, "example.com/storepkg",
			"an underivable path leaves the provisional reference in place")
	})
}

// Swallowing a failed append reads downstream as a method nobody configured
// rather than as a fault, and the helpers are this plugin's whole output.
func TestGenerateReportsAFailedAppend(t *testing.T) {
	t.Parallel()

	s := storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("fault", storefixture.Arg("ErrNotFound")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()

	// The host has to queue first, or there is nothing to contribute against.
	// Freezing afterwards provokes the failure from outside the pipeline,
	// standing in for the real cause — an append arriving after Layout has
	// closed the graph.
	// Assembled through the SDK rather than driven by [plugintest.Annotate] /
	// [plugintest.Generate]: those build a sink per call and fail the test on an
	// error, and this test shares one sink across three phases and asserts on
	// the error the last one returns.
	sink := sdk.NewSink()
	f := fault.New()
	// The stamps are this plugin's own annotator pass; without it the
	// directive is present and the metadata it drives is not.
	if err := f.Annotate(&sdk.AnnotatorContext{Store: s, Reader: sdk.NewStoreReader(s), Diag: sink}); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	ctx := &sdk.GeneratorContext{Store: s, Reader: sdk.NewStoreReader(s), Diag: sink}
	if err := stub.New().Generate(ctx); err != nil {
		t.Fatalf("host Generate: %v", err)
	}
	s.Emit().Freeze()

	err := f.Generate(ctx)

	testkit.Error(t, err, "a failed append must surface")
	testkit.Contains(t, err.Error(), fault.Name, "the error must name the plugin")
}

// dir builds one fault directive carrying positional sentinels.
func dir(sentinels ...string) *sdk.Directive {
	return &sdk.Directive{Name: sdk.DirectiveName(fault.DirectiveName), Args: sentinels}
}

// withKeys builds one fault directive carrying a single key.
func withKeys(key, value string) *sdk.Directive {
	return &sdk.Directive{
		Name: sdk.DirectiveName(fault.DirectiveName),
		KV:   map[string]string{key: value},
	}
}

// directives is the canonical mixed fixture: sentinels plus both keys.
func directives() []*sdk.Directive {
	return []*sdk.Directive{
		dir("ErrNotFound", "ErrGone"),
		withKeys(fault.RetryKey, "3"),
		withKeys(fault.PartitionKey, "RunID"),
	}
}

// fixture returns a store holding one interface whose Get method carries the
// supplied directives.
func fixture(dirs ...*sdk.Directive) *sdk.Store {
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
func annotated(t *testing.T, dirs ...*sdk.Directive) *sdk.Method {
	t.Helper()
	m, _ := annotate(t, dirs...)
	return m
}

// annotate is [annotated] with the sink as well, for the cases asserting on
// what the run reported.
//
// [plugintest.Annotate] rather than a hand-assembled plugin.AnnotatorContext:
// it recovers a panic into a named test failure instead of taking the binary
// down with it, and keeps this file off a direct plugin import.
func annotate(t *testing.T, dirs ...*sdk.Directive) (*sdk.Method, *sdk.Sink) {
	t.Helper()

	s := fixture(dirs...)
	sink := plugintest.Annotate(t, fault.New(), s)

	ifaces := s.Nodes().Interfaces().Items()
	if len(ifaces) != 1 || len(ifaces[0].Methods) != 1 {
		t.Fatalf("fixture shape changed: %d interfaces", len(ifaces))
	}
	return ifaces[0].Methods[0], sink
}

// render drives both plugins and the Go backend over pkg through a synthetic
// pipeline, so routing, slot materialisation, and rendering all participate.
//
// The fault plugin is registered under both the roles it implements — the same
// instance, which is what the directive owner reading its own stamps requires.
func render(t *testing.T, pkg *sdk.Package) *pipelinetest.Pipeline {
	t.Helper()
	// This plugin stamps the metadata it later reads, so it has to reach the
	// annotator list as well as the generator list. [golangtest.Driver] puts it
	// there: it registers each generator under every role the plugin implements,
	// which is what the CLI does. Adding the annotator half again here would
	// register one instance twice within a single role, which the pipeline
	// rejects — the exemption for a dual-role plugin covers one instance across
	// roles, not twice inside one.
	return golangtest.Driver(t, backendgolang.New(), pkg, fault.New(), stub.New()).
		Build().
		Run("./...")
}

// mixedHosting returns a package holding one interface that requests a double
// and one that carries fault configuration without requesting one.
func mixedHosting(t *testing.T) *sdk.Package {
	t.Helper()
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Unhosted", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("storepkg/store.go", 1, 1))
			i.Method("Drop", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("fault", storefixture.Arg("ErrGone")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("storepkg/store.go", 1, 1))
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("fault", storefixture.Arg("ErrNotFound")))
				m.Return(storefixture.Named("error"))
			})
		}).
		PackageNode()
}

// faulted returns a store whose Get carries every fault key at once, with the
// double either requested or not.
//
// One method carrying sentinels, a retry, and a partition together is
// deliberate: the three are independent settings on one directive, and a
// projection that dropped one when another was present would still pass every
// single-key fixture.
func faulted(t *testing.T, doubled bool) *sdk.Package {
	t.Helper()
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			// Layout composes the output filename from the source basename, so
			// the fixture needs a position for the rendered names to be
			// anything other than a bare suffix.
			i.Pos(sdk.At("storepkg/store.go", 1, 1))
			// Routed into its own package, as every corpus fixture is: it is
			// the only arrangement under which a sentinel reference has to
			// carry a qualifier, and the unrouted case would silently elide it.
			i.Directive(storefixture.Directive("out",
				storefixture.Arg("storetest/"),
				storefixture.KV("pkg", "storetest"),
			))
			if doubled {
				i.Directive(storefixture.Directive("stub"))
			}
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("fault",
					storefixture.Arg("ErrNotFound"),
					storefixture.Arg("ErrGone"),
					storefixture.KV(fault.RetryKey, "3"),
					storefixture.KV(fault.PartitionKey, "Tenant"),
				))
				m.Param("tenant", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Close", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
			})
		}).
		PackageNode()
}

// badPartition returns a store whose Get names a partition field no parameter
// supplies.
func badPartition(t *testing.T) *sdk.Package {
	t.Helper()
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("storepkg/store.go", 1, 1))
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("fault",
					storefixture.KV(fault.PartitionKey, "Nonexistent"),
				))
				m.Param("tenant", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		PackageNode()
}

// The helpers and checks this plugin contributes land in the stub generator's
// file, so nothing that renders stub alone ever compiles them.
//
// What that leaves exposed is everything this plugin derives rather than copies:
// the partition helper's key parameter takes whichever type the named
// recorded-call field came from, and the sentinel references are qualified
// against the source package because the double is routed out of it. Both are
// invisible to a substring assertion and both are compile errors in a
// consumer's repository if they are wrong.
//
// Compiles and vets, not [golangtest.Generated.AssertTestsPass]: the sentinels
// the projected package declares are typed and uninitialised, so running the
// suite would assert against nil errors. Giving them values means `errors.New`,
// and an import referenced only by an initialiser is dropped from the
// projection, which the frontend now marks.
func TestToolchainAcceptsTheContribution(t *testing.T) {
	t.Parallel()

	b := toolchainFixture()
	p := fault.New()
	gen := golangtest.Rendered(t,
		golangtest.Driver(t, backendgolang.New(), b.PackageNode(), p, stub.New()).
			Build().
			Run("./..."),
	).
		WithSource(golangtest.GoFile(b.GoSource())).
		WithRequire(stub.Module, filepath.Join("..", ".."))

	gen.AssertCompiles(t)
	gen.AssertVets(t)
}

// toolchainFixture is [faulted] as a builder, plus the sentinels the generated
// helpers name.
//
// The sentinels are typed and uninitialised — `var ErrNotFound error` — which
// is the one spelling the projection accepts for a package-level variable
// without pulling in an import it would then drop.
func toolchainFixture() *storefixture.Builder {
	b := storefixture.New().Package("storepkg", "example.com/storepkg")
	for _, name := range []string{"ErrNotFound", "ErrGone"} {
		b.Variable(name, func(v *storefixture.VariableBuilder) {
			v.Pos(sdk.At("storepkg/store.go", 1, 1))
			v.Type(storefixture.Named("error"))
		})
	}
	b.Interface("Store", func(i *storefixture.InterfaceBuilder) {
		i.Pos(sdk.At("storepkg/store.go", 1, 1))
		i.Directive(storefixture.Directive("stub"))
		i.Method("Get", func(m *storefixture.MethodBuilder) {
			m.Directive(storefixture.Directive("fault",
				storefixture.Arg("ErrNotFound"),
				storefixture.Arg("ErrGone"),
				storefixture.KV(fault.RetryKey, "3"),
				storefixture.KV(fault.PartitionKey, "Tenant"),
			))
			m.Param("tenant", storefixture.Named("string"))
			m.Return(storefixture.Named("string"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Close", func(m *storefixture.MethodBuilder) {
			m.Return(storefixture.Named("error"))
		})
	})
	return b
}
