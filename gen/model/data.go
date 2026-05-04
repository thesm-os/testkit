// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"go/types"
	"strings"
	"text/template"

	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directives"
	"go.thesmos.sh/testkit/gen/suite"
)

// Data extends SpecData with model-specific analysis.
type Data struct {
	*suite.SpecData

	// HasCRUD is true when the interface has at least Reader + Writer shapes.
	HasCRUD bool

	// CanSynthesizeRef is true when refmap.MapStore can satisfy the full
	// interface — i.e. every non-skipped method is Reader, Writer, Deleter,
	// Aggregator, or StreamReader shaped (the shapes MapStore implements).
	CanSynthesizeRef bool

	// RefBlockers lists method names whose shapes prevent refmap synthesis.
	// Empty when CanSynthesizeRef is true.
	RefBlockers []string

	// KeyField is the struct field name used for key extraction.
	// Set from //testkit:keyfield directive or heuristic ("ID" field).
	KeyField string

	// ReaderMethod, WriterMethod, etc. are the first detected method
	// of each shape, used for wiring actions and laws.
	ReaderMethod  *suite.SpecMethodData
	WriterMethod  *suite.SpecMethodData
	DeleterMethod *suite.SpecMethodData
	CountMethod   *suite.SpecMethodData // Aggregator returning int
	StreamMethod  *suite.SpecMethodData

	// AutoLaws lists the auto-derived law names that will be emitted.
	AutoLaws []string

	// SkippedMethods lists methods with Unknown shape that aren't
	// covered by auto-derived actions.
	SkippedMethods []string

	// IsGeneric is true when the interface has uninstantiated type parameters.
	// When true, the generator emits parameterized functions with type
	// constraints threaded through.
	IsGeneric bool

	// TypeParamDecl is the Go type parameter declaration for generic
	// interfaces, e.g., "[K comparable, V any]". Empty for non-generic.
	TypeParamDecl string

	// TypeParamNames is the instantiation-site type list, e.g., "[K, V]".
	// Empty for non-generic.
	TypeParamNames string
}

// IsCRUD reports whether Tier 0 reference synthesis is possible.
func (d *Data) IsCRUD() bool { return d.HasCRUD }

// templateFuncs returns template functions that need access to Data.
func (d *Data) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"writerGen": func(m *suite.SpecMethodData) string {
			return d.WriterGenName(m)
		},
		"firstSentinel": func(m *suite.SpecMethodData) string {
			if len(m.Sentinels) > 0 {
				return m.Sentinels[0].Qualified
			}
			return ""
		},
		"keySamples": func() string {
			return d.KeySamples()
		},
	}
}

// WriterGenName returns the generator variable name for a Writer-shaped
// method. If the method's V type matches the Reader's K type, returns
// the key generator; otherwise returns the value generator.
func (d *Data) WriterGenName(m *suite.SpecMethodData) string {
	if d.ReaderMethod != nil && m.Shape.ValType == d.ReaderMethod.Shape.KeyType {
		return "keyGen"
	}
	return "valGen"
}

// KeySamples returns sample literal values for the key generator,
// appropriate for the key type. Defaults to string literals; emits
// int literals for integer-underlying key types.
func (d *Data) KeySamples() string {
	if d.ReaderMethod == nil {
		return `"a", "b", "c", "d", "e"`
	}
	// Check if the key type's underlying type is integer-based.
	sig := d.ReaderMethod.Signature
	for p := range sig.Params().Variables() {
		if gen.IsContextType(p.Type()) {
			continue
		}
		t := p.Type()
		if named, ok := t.(*types.Named); ok {
			t = named.Underlying()
		}
		if basic, ok := t.(*types.Basic); ok {
			switch basic.Kind() { //nolint:exhaustive // only int kinds need special handling
			case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
				types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
				return "1, 2, 3, 4, 5"
			}
		}
		break
	}
	return `"a", "b", "c", "d", "e"`
}

// HasDeleter reports whether a Deleter-shaped method was detected.
func (d *Data) HasDeleter() bool { return d.DeleterMethod != nil }

// HasCount reports whether an Aggregator-shaped method returning int was detected.
func (d *Data) HasCount() bool { return d.CountMethod != nil }

// HasStream reports whether a StreamReader-shaped method was detected.
func (d *Data) HasStream() bool { return d.StreamMethod != nil }

func buildData(spec *suite.SpecData, typeParams []gen.TypeParamInfo) *Data {
	md := &Data{SpecData: spec}

	// Set up generic type parameter strings.
	if len(typeParams) > 0 {
		md.IsGeneric = true
		md.TypeParamDecl = formatTypeParamDecl(typeParams)
		md.TypeParamNames = formatTypeParamNames(typeParams)
	}

	// Detect shapes.
	for _, m := range spec.Methods {
		if m.Skip {
			continue
		}
		switch m.Shape.Shape {
		case gen.ShapeReader:
			if md.ReaderMethod == nil {
				md.ReaderMethod = m
			}
		case gen.ShapeWriter:
			if md.WriterMethod == nil {
				md.WriterMethod = m
			} else if isStructValType(m) && !isStructValType(md.WriterMethod) {
				// Prefer the Writer with struct-typed V over primitive-typed V.
				// Delete(ctx, string) is Writer-shaped when no //testkit:deleter
				// directive is present; Put(ctx, Item) is the real Writer.
				md.WriterMethod = m
			}
		case gen.ShapeDeleter:
			if md.DeleterMethod == nil {
				md.DeleterMethod = m
			}
		case gen.ShapeAggregator:
			if md.CountMethod == nil {
				md.CountMethod = m
			}
		case gen.ShapeStreamReader:
			if md.StreamMethod == nil {
				md.StreamMethod = m
			}
		case gen.ShapeReaderWithBool:
			// ReaderWithBool uses same key pool as Reader.
			if md.ReaderMethod == nil {
				md.ReaderMethod = m
			}
		case gen.ShapeLookup:
			// Lookup uses same key pool as Reader.
			if md.ReaderMethod == nil {
				md.ReaderMethod = m
			}
		case gen.ShapeMutator:
			// Mutator is a state-changing command; treat like Writer
			// for CRUD detection (it modifies state).
			if md.WriterMethod == nil {
				md.WriterMethod = m
			}
		case gen.ShapePoisonAccessor:
			// Handled by template; emits poison check in law loop.
		case gen.ShapeLifecycle, gen.ShapePure, gen.ShapePredicate:
			// Handled by template; no first-method tracking needed.
		case gen.ShapeUnknown:
			md.SkippedMethods = append(md.SkippedMethods,
				m.Name+"("+m.Shape.Shape.String()+")")
		}
	}

	md.HasCRUD = md.ReaderMethod != nil && md.WriterMethod != nil

	// CanSynthesizeRef: true only when every non-skipped method is a
	// shape that refmap.MapStore implements.
	md.CanSynthesizeRef = md.HasCRUD
	for _, m := range spec.Methods {
		if m.Skip {
			continue
		}
		switch m.Shape.Shape {
		case gen.ShapeReader, gen.ShapeWriter, gen.ShapeDeleter,
			gen.ShapeAggregator, gen.ShapeStreamReader:
			// MapStore covers these.
		default:
			md.RefBlockers = append(md.RefBlockers, m.Name)
			md.CanSynthesizeRef = false
		}
	}

	// Find keyfield directive or heuristic.
	// For generic interfaces, keyfield resolution is deferred to the
	// consumer (V is unknown at codegen time).
	if md.HasCRUD && !md.IsGeneric {
		md.KeyField = findKeyField(spec)
	}

	// Determine auto-laws.
	if md.HasCRUD && md.KeyField != "" {
		md.AutoLaws = append(md.AutoLaws, "AUTO-READ-AFTER-WRITE")
		if md.HasDeleter() {
			md.AutoLaws = append(md.AutoLaws, "AUTO-DELETE-RETURNS-NOT-FOUND")
		}
	}
	if md.HasCount() {
		md.AutoLaws = append(md.AutoLaws, "AUTO-COUNT-EQUALS-REFERENCE")
	}

	return md
}

// findKeyField looks for //testkit:keyfield directive on any method,
// then falls back to heuristic: struct field named "ID" on the preferred
// Writer method's V type.
func findKeyField(spec *suite.SpecData) string {
	// Check directives on each method for keyfield.
	for _, m := range spec.Methods {
		for _, d := range m.Directives {
			if d.Name == directives.KeyField && len(d.Args) > 0 {
				return d.Args[0]
			}
		}
	}

	// Heuristic: find a Writer-shaped method with struct-typed V and
	// look for an "ID" field.
	for _, m := range spec.Methods {
		if m.Shape.Shape != gen.ShapeWriter {
			continue
		}
		if field := findIDField(m); field != "" {
			return field
		}
	}

	return ""
}

// findIDField checks if the Writer method's V parameter type is a struct
// (or pointer-to-struct) with a field named "ID" (case-insensitive).
func findIDField(m *suite.SpecMethodData) string {
	for p := range m.Signature.Params().Variables() {
		if gen.IsContextType(p.Type()) {
			continue
		}
		// This is V. Dereference pointer and named types to get the struct.
		t := p.Type()
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		st, ok := t.Underlying().(*types.Struct)
		if !ok {
			return "" // not a struct — can't extract key field
		}
		for f := range st.Fields() {
			if strings.EqualFold(f.Name(), "ID") {
				return f.Name()
			}
		}
		return "" // struct but no ID field
	}
	return ""
}

// isStructValType reports whether the Writer method's V parameter is a
// struct or pointer-to-struct type.
func isStructValType(m *suite.SpecMethodData) bool {
	for p := range m.Signature.Params().Variables() {
		if gen.IsContextType(p.Type()) {
			continue
		}
		t := p.Type()
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		_, ok := t.Underlying().(*types.Struct)
		return ok
	}
	return false
}

// formatTypeParamDecl formats type parameters as a Go declaration:
// "[K comparable, V any]".
func formatTypeParamDecl(params []gen.TypeParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	var parts []string
	for _, p := range params {
		parts = append(parts, p.Name+" "+types.TypeString(p.Constraint, nil))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// formatTypeParamNames formats type parameters as an instantiation list:
// "[K, V]".
func formatTypeParamNames(params []gen.TypeParamInfo) string {
	if len(params) == 0 {
		return ""
	}
	var names []string
	for _, p := range params {
		names = append(names, p.Name)
	}
	return "[" + strings.Join(names, ", ") + "]"
}
