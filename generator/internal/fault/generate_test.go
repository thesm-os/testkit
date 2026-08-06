// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"strings"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/fault"
	"go.thesmos.sh/testkit/generator/stub"
)

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
		body := render(t, faulted(t, true)).AssertFile(primaryFile).String()
		helpers := strings.Index(body, "// --- Fault injection ---")
		double := strings.Index(body, "type StoreGetStub struct")
		testkit.True(t, double >= 0, "the double must render")
		testkit.True(t, helpers > double, "the fault surface must follow the double")
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
	sink := diag.New()
	f := fault.New()
	// The stamps are this plugin's own annotator pass; without it the
	// directive is present and the metadata it drives is not.
	if err := f.Annotate(&plugin.AnnotatorContext{Store: s, Reader: store.NewReader(s), Diag: sink}); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	ctx := &plugin.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: sink}
	if err := stub.New().Generate(ctx); err != nil {
		t.Fatalf("host Generate: %v", err)
	}
	s.Emit().Freeze()

	err := f.Generate(ctx)

	testkit.Error(t, err, "a failed append must surface")
	testkit.Contains(t, err.Error(), fault.Name, "the error must name the plugin")
}

// render drives both plugins and the Go backend over pkg through a synthetic
// pipeline, so routing, slot materialisation, and rendering all participate.
//
// The fault plugin is registered under both the roles it implements — the same
// instance, which is what the directive owner reading its own stamps requires.
func render(t *testing.T, pkg *node.Package) *pipelinetest.Pipeline {
	t.Helper()
	f := fault.New()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(pkg)).
		WithAnnotator(f).
		WithGenerator(stub.New()).
		WithGenerator(f).
		WithBackend(backendgolang.New()).
		Build().
		Run()
}

// mixedHosting returns a package holding one interface that requests a double
// and one that carries fault configuration without requesting one.
func mixedHosting(t *testing.T) *node.Package {
	t.Helper()
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Unhosted", func(i *storefixture.InterfaceBuilder) {
			i.Pos(position.At("storepkg/store.go", 1, 1))
			i.Method("Drop", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("fault", storefixture.Arg("ErrGone")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(position.At("storepkg/store.go", 1, 1))
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
func faulted(t *testing.T, doubled bool) *node.Package {
	t.Helper()
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			// Layout composes the output filename from the source basename, so
			// the fixture needs a position for the rendered names to be
			// anything other than a bare suffix.
			i.Pos(position.At("storepkg/store.go", 1, 1))
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
func badPartition(t *testing.T) *node.Package {
	t.Helper()
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(position.At("storepkg/store.go", 1, 1))
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
