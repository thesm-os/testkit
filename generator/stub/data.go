// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub generates a recording test double for a Go interface.
// Each Generate call emits two files: the impl (the [Stub] type and
// dispatch wired to the runtime [stub.MethodStub]), and the
// auto-test that verifies the stub's plumbing (dispatch, recording,
// fault helpers, DelegateTo, ResetCalls).
//
// Stub is signature-driven for the recording structure and
// directive-aware for behavioral additions (errors → fault helpers,
// retry-succeeds-on-attempt → fault sequencer, partition → keyed
// recorder, order-after → runtime ordering, wrapped-via → wrap
// target). The recording structure itself is the same for every
// shape — Reader, Writer, Mutator, Pure, Predicate, PoisonAccessor,
// Aggregator, Lifecycle, StreamReader, etc. all render identically.
//
// Every shared piece — signature flags, param/result rendering,
// call-struct fields, iter detection, directive payload accessors,
// directive render helpers — lives in spec/ and spec/<directive>.
// This package contains ONLY stub-specific concerns: per-method
// type names, the Returns()/Yields() helper rendering, and the
// dispatch body's stub-specific shape.
package stub

import (
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/deprecated"
	"go.thesmos.sh/testkit/generator/spec/errors"
	"go.thesmos.sh/testkit/generator/spec/integrationonly"
	"go.thesmos.sh/testkit/generator/spec/orderafter"
	"go.thesmos.sh/testkit/generator/spec/partition"
	"go.thesmos.sh/testkit/generator/spec/retrysucceeds"
	"go.thesmos.sh/testkit/generator/spec/wrappedvia"
)

// Name is the generator's CLI subcommand name. Exported so other
// packages reference stub by typed constant rather than the raw
// "stub" string.
const Name = "stub"

// Data is the top-level template input for one stub generation.
// Two files are rendered from the same Data — the impl
// (`stub.go.tmpl`) and the auto-test (`stub_test.go.tmpl`); the
// test view's [Renderer.Transform] hook reshapes PackageName /
// Imports / GenQualifier for the external _test package.
type Data struct {
	// Spec is the shared analysis result. The Pipeline's directive
	// validator runs against [Spec.Methods] before consumers fire;
	// PostEnrich projects [Spec.Methods] into [Methods] after
	// enrichment populates each method's Attachments.
	Spec *spec.Data

	// PackageName / Imports / ImplImportPath / GenQualifier are
	// the standard output-file fields every interface-conformance
	// generator carries.
	PackageName    string
	Imports        []generator.Import
	ImplImportPath string
	GenQualifier   string

	// StubName is the generated stub type name ("StoreStub").
	StubName string

	// TestTypeArgs renders the concrete-instantiation suffix the
	// auto-test uses for generic stubs ("[int]", "[string, int]",
	// constraint-driven). Empty for non-generic interfaces — the
	// test then refers to the stub by its bare name.
	TestTypeArgs string

	// HasOrderConstraint is true when any method carries
	// //testkit:order-after — drives emission of the
	// stub.OrderTracker field and the ResetCalls() reset path.
	HasOrderConstraint bool

	// HasErrorMethod is true when any method returns an error —
	// drives emission of the shared `errTest` sentinel in the
	// auto-test.
	HasErrorMethod bool

	// Methods is one [Method] per interface method, populated by
	// the Pipeline's PostEnrich step.
	Methods []MethodView
}

// HasContent reports whether the interface has at least one method
// worth stubbing. Reads [Spec] because the Pipeline invokes
// HasContent right after Analyze, before [Project] populates
// [Methods].
func (d *Data) HasContent() bool { return d.Spec != nil && len(d.Spec.Methods) > 0 }

// FirstContextMethod returns the first non-skipped method that
// takes a [context.Context] — used by latency auto-tests where one
// representative method exercises the runtime stub's Latency
// contract.
func (d *Data) FirstContextMethod() *MethodView {
	for i := range d.Methods {
		if !d.Methods[i].Skip() && d.Methods[i].HasContext() {
			return &d.Methods[i]
		}
	}
	return nil
}

// FirstErrorMethod returns the first non-skipped method that
// returns an error — used by FaultsFor auto-tests.
func (d *Data) FirstErrorMethod() *MethodView {
	for i := range d.Methods {
		if !d.Methods[i].Skip() && d.Methods[i].ReturnsError() {
			return &d.Methods[i]
		}
	}
	return nil
}

// QualifiedTypeForTest returns the source-package-qualified
// interface reference suffixed with the test-time concrete
// type-args. For non-generic interfaces it equals
// [spec.Data.QualifiedType]. For generics it strips the type-
// parameter-name suffix and appends [TestTypeArgs] — used by the
// compile-time guard so a concrete instantiation of the stub
// catches signature drift at build time.
//
//	non-generic:        "basic.Store"
//	generic Cache[V]:   "generics.Cache[int]"      (V → int)
//	KeyMap[K, V]:       "generics.KeyMap[string, int]"
func (d *Data) QualifiedTypeForTest() string {
	if !d.Spec.IsGeneric {
		return d.Spec.QualifiedType
	}
	base := strings.TrimSuffix(d.Spec.QualifiedType, d.Spec.TypeParamArgs)
	return base + d.TestTypeArgs
}

// FirstNonSkipMethod returns the first non-skipped method — used by
// auto-tests whose subtests only need a single method to exercise a
// stub-level option (BenchMode, Times, ...).
func (d *Data) FirstNonSkipMethod() *MethodView {
	for i := range d.Methods {
		if !d.Methods[i].Skip() {
			return &d.Methods[i]
		}
	}
	return nil
}

// FirstNonSkipMethodWithSampleableResults returns the first method
// whose results can be sampled and asserted: HasAssertableNonErrorResults
// AND not an iter.Seq/iter.Seq2 returner (the function-typed return
// can't round-trip through Returns(...) without a deeper helper).
// Used by DelegateTo's value-passthrough subtest where a real
// observable round-trip beats a count-only assertion.
func (d *Data) FirstNonSkipMethodWithSampleableResults() *MethodView {
	tracker := d.Spec.Tracker
	for i := range d.Methods {
		m := &d.Methods[i]
		if m.Skip() || !m.HasAssertableNonErrorResults() {
			continue
		}
		iter := m.IterReturn(tracker)
		if iter.IsSeq || iter.IsSeq2 {
			continue
		}
		return m
	}
	return nil
}

// MethodView wraps a [*spec.MethodView] with the stub-specific naming
// conventions and render helpers. Templates call directly into the
// embedded spec.MethodView for signature-driven helpers (ParamList,
// ResultDecls, CallStructFields, IterReturn, …) and onto MethodView's
// own methods for stub-specific names (StubType, ReturnType, …).
type MethodView struct {
	// Spec is the underlying [*spec.Method] — embedded for
	// promotion so templates can call its receiver methods
	// directly (e.g. {{.ParamList .Tracker}}).
	*spec.Method

	// stubName + typeArgs are captured at projection time so the
	// per-method name renderers stay self-contained.
	stubName string
	typeArgs string
}

// Skip reports whether the method carries //testkit:integration-only.
// The stub emits a no-op body for skipped methods (so the stub still
// satisfies the interface), but omits the per-method [stub.MethodStub]
// embedding and recording slice.
func (m *MethodView) Skip() bool {
	return integrationonly.Has(m.Method)
}

// StubType is the per-method stub struct name ("<Stub><Method>") —
// e.g. "StoreStubGet". Embeds [*stub.MethodStub] at runtime.
func (m *MethodView) StubType() string { return m.stubName + m.Name }

// QualStubType is [StubType] suffixed with the interface's type
// arguments — used inside generic stub templates so the per-method
// type expression matches the parameterized stub type.
func (m *MethodView) QualStubType() string { return m.StubType() + m.typeArgs }

// CallType is the per-method Call struct name ("<Stub><Method>Call").
// Prefixed with the stub name so multiple stubs emitted into one
// package don't collide on shared method names — Holder.Get and
// KeyMap.Get both produce a "Get" method but their Call structs
// must be distinct.
func (m *MethodView) CallType() string { return m.stubName + m.Name + "Call" }

// QualCallType is [CallType] suffixed with the interface's type
// arguments.
func (m *MethodView) QualCallType() string { return m.CallType() + m.typeArgs }

// ReturnType is the per-method return-record struct name —
// "<lowerStub><Method>Return". Lowercased so the type stays
// unexported (the fallback record is implementation detail).
// Empty when the method has no results.
func (m *MethodView) ReturnType() string {
	if !m.HasResults() {
		return ""
	}
	return generator.LowerFirst(m.stubName) + m.Name + "Return"
}

// QualReturnType is [ReturnType] suffixed with the interface's
// type arguments.
func (m *MethodView) QualReturnType() string {
	if !m.HasResults() {
		return ""
	}
	return m.ReturnType() + m.typeArgs
}

// --- typed directive accessors ---
//
// These project the registered consumer payloads through stub's
// preferred names. Templates reference them by method receiver so
// the directive name strings stay encapsulated inside the
// spec/<directive> packages — no magic strings here.

// Errors returns the resolved //testkit:errors sentinels for this
// method. Empty when the directive isn't set.
func (m *MethodView) Errors() []errors.Sentinel {
	if p, ok := errors.Get(m.Method); ok {
		return p.Sentinels
	}
	return nil
}

// Deprecated returns the //testkit:deprecated replacement-method
// name, or "" when not deprecated.
func (m *MethodView) Deprecated() string {
	if p, ok := deprecated.Get(m.Method); ok {
		return p.Replacement
	}
	return ""
}

// RetryN returns the //testkit:retry-succeeds-on-attempt count, or
// 0 when not configured.
func (m *MethodView) RetryN() int {
	if p, ok := retrysucceeds.Get(m.Method); ok {
		return p.N
	}
	return 0
}

// OrderAfter returns the //testkit:order-after prerequisite method
// name, or "" when not configured.
func (m *MethodView) OrderAfter() string {
	if p, ok := orderafter.Get(m.Method); ok {
		return p.Method
	}
	return ""
}

// Partition returns the resolved //testkit:partition payload, or
// nil when not configured. Templates use `{{with $m.Partition}}` to
// gate partition-specific rendering.
func (m *MethodView) Partition() *partition.Payload {
	if p, ok := partition.Get(m.Method); ok {
		return &p
	}
	return nil
}

// WrappedVia returns the resolved //testkit:wrapped-via payload,
// or nil when not configured.
func (m *MethodView) WrappedVia() *wrappedvia.Payload {
	if p, ok := wrappedvia.Get(m.Method); ok {
		return &p
	}
	return nil
}
