// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
)

// Name is the generator's CLI subcommand name. Exported so other
// packages reference suite by typed constant rather than the raw
// "suite" string.
const Name = "suite"

// Data is the top-level template input for one suite generation.
// The renderer emits one file containing the contract drivers
// (`Assert<Iface>Contract` + `Assert<Iface>ContractAcrossImpls`)
// and per-method subtests against any factory-produced impl.
//
// Data wraps [spec.Data] so suite reads the shared analysis output —
// methods, shape classifications, directive payloads — through one
// pointer. Suite-specific naming + per-method projection live on
// the wrapper.
type Data struct {
	// Spec is the shared analysis result.
	Spec *spec.Data

	// PackageName / Imports / GenQualifier / ImplImportPath are the
	// standard output-file fields. The contract driver lives in the
	// external _test package alongside the impl's tests.
	PackageName    string
	Imports        []generator.Import
	ImplImportPath string
	GenQualifier   string

	// IfaceName is the bare interface name ("Store", "Cache") —
	// drives the public driver names below.
	IfaceName string

	// DriverName is the public single-impl driver — `Assert<Iface>Contract`.
	DriverName string

	// AcrossImplsName is the public multi-impl driver —
	// `Assert<Iface>ContractAcrossImpls`. The consumer supplies one
	// factory per implementation; the driver runs the full contract
	// once per impl with `t.Run(name, ...)`.
	AcrossImplsName string

	// FactoryTypeName is the helper alias for the factory function
	// signature — `<Iface>Factory`.
	FactoryTypeName string

	// NamedFactoryTypeName is the helper alias for the multi-impl
	// (name, factory) tuple — `<Iface>NamedFactory`.
	NamedFactoryTypeName string

	// HasErrorMethod is true when at least one non-skipped method
	// returns an error. Drives the driver's `errTest` declaration —
	// suite's per-sentinel and fault-helper-validation subtests
	// reference a shared canonical test error.
	HasErrorMethod bool

	// Methods is one [MethodView] per interface method, populated
	// by the Pipeline's PostEnrich step.
	Methods []MethodView
}

// HasContent reports whether the interface has at least one method
// worth verifying. Reads [Spec] because the Pipeline invokes
// HasContent right after Analyze, before [Project] populates
// [Methods].
func (d *Data) HasContent() bool { return d.Spec != nil && len(d.Spec.Methods) > 0 }

// QualifiedTypeForTest delegates to [spec.Data.QualifiedTypeForTest]
// — the wrapper exists so templates can call .QualifiedTypeForTest
// off the suite view without reaching through .Spec.
func (d *Data) QualifiedTypeForTest() string {
	return d.Spec.QualifiedTypeForTest()
}

// FirstNonSkipMethod returns the first non-skipped method — used by
// driver-level subtests that only need a representative method
// (e.g. "factory yields a non-nil value").
func (d *Data) FirstNonSkipMethod() *MethodView {
	for i := range d.Methods {
		if !d.Methods[i].IsIntegrationOnly() {
			return &d.Methods[i]
		}
	}
	return nil
}

// MethodView wraps a [*spec.Method] with the suite-specific naming
// conventions. Templates call directly into the embedded spec.Method
// for signature-driven helpers (ParamList, ResultDecls, ZeroArgs,
// IsIntegrationOnly, …) and onto MethodView's own methods for
// suite-specific concerns (subtest naming, sample-driven invocation).
type MethodView struct {
	// Method is the underlying analyzed method — embedded for
	// promotion so templates can call its receiver methods directly
	// (e.g. {{.HasContext}}, {{.IsIntegrationOnly}}, {{.ZeroArgs .Tracker}}).
	*spec.Method

	// ifaceName is captured at projection time so per-method
	// renderers stay self-contained when emitting subtests qualified
	// by interface name.
	ifaceName string

	// typeArgs captured at projection time — the test-time concrete
	// instantiation suffix for generic interfaces. Empty for
	// non-generic. Mirrors stub's MethodView.typeArgs pattern.
	typeArgs string
}

// IfaceName returns the source interface's bare name. Suite emits
// subtest names like "Store/Get/smoke" where the iface name is
// implicit in the driver but useful for shape-specific assertion
// references (e.g. `suite.AssertReturnsForKey[basic.Store, ...]`).
func (m *MethodView) IfaceName() string { return m.ifaceName }
