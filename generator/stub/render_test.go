// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"strings"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/stub"
)

// Rendering is where a generator actually fails. Every assertion driven off
// the emit graph passes against a template that renders code which does not
// compile — an unused local, a redeclared name, a field the template still
// references. Those surface only once the backend runs, so the templates are
// driven end-to-end here rather than trusted.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("renders both outputs", func(t *testing.T) {
		t.Parallel()
		renderFixture(t).AssertFileCount(2)
	})

	t.Run("renders without a diagnostic", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, renderFixture(t).Diagnostics().Diagnostics(), 0,
			"a clean fixture must render without diagnostics")
	})

	t.Run("emits the double's configuration surface", func(t *testing.T) {
		t.Parallel()
		// Asserted on tokens rather than whole lines: the backend runs the
		// formatter, so alignment is padding this test has no business
		// pinning. The golden below covers the exact bytes.
		f := renderFixture(t).AssertFile(primaryFile)
		for _, want := range []string{
			"type StoreGetStub struct",
			"*stub.MethodStub[StoreGetCall]",
			"func (s *StoreGetStub) Returns(item string, err error) *StoreGetStub",
			"type StoreGetReturn struct",
			"type StoreStub struct",
			"OnGet   *StoreGetStub",
			"func NewStoreStub(tb testing.TB, opts ...StoreStubOption) *StoreStub",
			"func (s *StoreStub) ResetCalls()",
			"func StoreStubStrict() StoreStubOption",
			"func StoreStubDelegateTo(",
			"func StoreStubWithClock(",
			"func StoreStubBenchMode() StoreStubOption",
			"func WithStoreGet(",
		} {
			f.Contains(want)
		}
	})

	t.Run("resolves a call through the shared dispatcher", func(t *testing.T) {
		t.Parallel()
		// Which arm answers, and that every arm records, is stub.Answer's
		// contract and is tested there against real calls. What the template
		// owes is the binding: the arms wired to the right fields, so a
		// double cannot dispatch differently from every other double.
		f := renderFixture(t).AssertFile(primaryFile)
		for _, want := range []string{
			"r := stub.Answer(s.OnGet.MethodStub, &call, stub.Arms[StoreGetCall, StoreGetReturn]{",
			"Invoke:   s.OnGet.invoke(ctx, id),",
			"Fallback: s.OnGet.fallback,",
			"Fault:    func(err error) StoreGetReturn { return StoreGetReturn{Err: err} },",
		} {
			f.Contains(want)
		}
	})

	t.Run("gives a method that cannot fail no fault arm", func(t *testing.T) {
		t.Parallel()
		// An injected fault has nowhere to go in a signature with no error,
		// and a nil Fault is how Answer knows to skip that arm rather than
		// inventing an error slot.
		body := renderFixture(t).AssertFile(primaryFile).String()
		closeArms := body[strings.Index(body, "stub.Arms[StoreCloseCall"):]
		testkit.False(t, strings.Contains(closeArms[:strings.Index(closeArms, "})")], "Fault:"),
			"a method with no error return declares no fault arm")
	})

	t.Run("puts the companion in the external test package", func(t *testing.T) {
		t.Parallel()
		renderFixture(t).AssertFile(companionFile).Contains("package storepkg_test")
	})
}

// The goldens are the readable record of what this generator produces. A diff
// on them is the review surface for any template change — the token
// assertions above say a construct is present, and only the golden says what
// the whole file reads like.
//
// Regenerate with `go test ./generator/stub/ -update-golden`.
func TestRenderMatchesGolden(t *testing.T) {
	t.Parallel()

	t.Run("the double", func(t *testing.T) {
		t.Parallel()
		matchesGolden(t, primaryFile)
	})

	t.Run("the companion", func(t *testing.T) {
		t.Parallel()
		matchesGolden(t, companionFile)
	})
}

// matchesGolden compares one rendered file against its golden, with the
// header's Command line normalised.
//
// The backend stamps os.Args into that line when nothing sets it, which under
// `go test` is the test binary's own flags — including a temp path that
// differs on every run. Normalising it keeps the golden about the generated
// code rather than about how the suite was invoked.
func matchesGolden(t *testing.T, name string) {
	t.Helper()
	body := renderFixture(t).AssertFile(name).String()
	pipelinetest.MatchesGoldenBytes(t, []byte(normaliseCommand(body)), "testdata/golden/"+name)
}

// normaliseCommand rewrites the header's Command line to a fixed value.
func normaliseCommand(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "// Command:") {
			lines[i] = "// Command:   testkit run ./storepkg"
			break
		}
	}
	return strings.Join(lines, "\n")
}

// The rendered filenames, composed by Layout from the source basename and the
// adapter's suffixes.
const (
	primaryFile   = "store" + stub.GoPrimarySuffix
	companionFile = "store" + stub.GoTestSuffix
)

// renderFixture drives the plugin and the Go backend over the shared fixture
// through a synthetic pipeline, so routing and rendering both participate.
func renderFixture(t *testing.T) *pipelinetest.Pipeline {
	t.Helper()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(fixturePackage())).
		WithGenerator(stub.New()).
		WithBackend(backendgolang.New()).
		Build().
		Run()
}

// fixturePackage is storeFixture's package node, for the synthetic frontend.
func fixturePackage() *node.Package {
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			// Layout composes the output filename from the source basename,
			// so the fixture needs a position for the rendered names to be
			// anything other than a bare suffix.
			i.Pos(position.At("storepkg/store.go", 1, 1))
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("id", storefixture.Named("string"))
				m.NamedReturn("item", storefixture.Named("string"))
				m.NamedReturn("err", storefixture.Named("error"))
			})
			i.Method("List", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Slice(storefixture.Named("string")))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Close", nil)
		}).
		PackageNode()
}
