// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/shape"
)

// classifyAll loads src as package "p", classifies every method on the
// named interface, and returns a map of method name to detected shape
// name. Errors fail the test fatally.
//
// Used by routing/integration tests in registry_test.go and
// priorities_test.go that want the full detector cascade.
func classifyAll(t *testing.T, src, ifaceName string, dirs map[string][]directive.Directive) map[string]string {
	t.Helper()
	iface, _ := loadInterface(t, src, ifaceName)
	tracker := generator.NewImportTracker("p")
	out := make(map[string]string, iface.NumMethods())
	for m := range iface.Methods() {
		mi := generator.MethodInfo{
			Name:      m.Name(),
			Signature: m.Type().(*types.Signature),
		}
		info := shape.Classify(mi, tracker, dirs[m.Name()])
		out[m.Name()] = info.Shape.String()
	}
	return out
}

// classifyOne classifies a single method named "F" in the supplied
// source. Most routing tests have one method, so this saves the
// per-test extraction step.
func classifyOne(t *testing.T, src string, dirs ...directive.Directive) string {
	t.Helper()
	d := map[string][]directive.Directive{"F": dirs}
	return classifyAll(t, src, "I", d)["F"]
}

// classifyAllViaInterface mirrors [classifyAll] but routes through
// [shape.ClassifyInterface] so contract- and composite-tier detectors
// see the InterfaceContext. Returns method-name → shape-name.
func classifyAllViaInterface(
	t *testing.T,
	src, ifaceName string,
	dirs map[string][]directive.Directive,
) map[string]string {
	t.Helper()
	iface, _ := loadInterface(t, src, ifaceName)
	tracker := generator.NewImportTracker("p")
	methods := make([]generator.MethodInfo, 0, iface.NumMethods())
	for m := range iface.Methods() {
		methods = append(methods, generator.MethodInfo{
			Name:      m.Name(),
			Signature: m.Type().(*types.Signature),
		})
	}
	infos := shape.ClassifyInterface(methods, tracker, dirs)
	out := make(map[string]string, len(methods))
	for i, m := range methods {
		out[m.Name] = infos[i].Shape.String()
	}
	return out
}

// buildSig parses src as package "p", finds method "F" on interface
// "I", and returns the [shape.Signature] view ready to pass to a
// detector's Detect method directly.
//
// Per-detector tests use this helper to exercise exactly one
// detector's accept/reject logic without going through the [Registry]
// cascade — the detector under test stays decoupled from priority
// ordering and other detectors' behaviors.
func buildSig(t *testing.T, src string, dirs ...directive.Directive) shape.Signature {
	t.Helper()
	iface, _ := loadInterface(t, src, "I")
	for m := range iface.Methods() {
		if m.Name() == "F" {
			mi := generator.MethodInfo{
				Name:      m.Name(),
				Signature: m.Type().(*types.Signature),
			}
			return shape.ParseSignature(mi, generator.NewImportTracker("p"), dirs)
		}
	}
	t.Fatalf("method F not found")
	return shape.Signature{}
}

// detectorByName looks up a shipped [shape.Detector] by its public
// name. Per-detector tests use this so they can call Detect directly
// without exposing the package-private detector struct types.
//
// Names match [Detector.Name] return values: "Reader", "Writer",
// "Aggregator", and so on.
func detectorByName(t *testing.T, name string) shape.Detector {
	t.Helper()
	for _, d := range shape.DefaultRegistry().Detectors() {
		if d.Name() == name {
			return d
		}
	}
	t.Fatalf("detector %q not registered", name)
	return nil
}

// loadInterface parses src as package "p" and returns the named
// interface plus the type-check info object.
func loadInterface(t *testing.T, src, ifaceName string) (*types.Interface, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	testkit.NoError(t, err, "parse")
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	pkg, err := conf.Check("p", fset, []*ast.File{f}, info)
	testkit.NoError(t, err, "typecheck")
	obj := pkg.Scope().Lookup(ifaceName)
	testkit.True(t, obj != nil, "interface "+ifaceName+" defined")
	iface, ok := obj.Type().Underlying().(*types.Interface)
	testkit.True(t, ok, ifaceName+" is interface")
	return iface, info
}
