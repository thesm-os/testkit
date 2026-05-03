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

// ModelData extends SpecData with model-specific analysis.
type ModelData struct {
	*suite.SpecData

	// HasCRUD is true when the interface has at least Reader + Writer shapes.
	HasCRUD bool

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
}

// IsCRUD reports whether Tier 0 reference synthesis is possible.
func (d *ModelData) IsCRUD() bool { return d.HasCRUD }

// templateFuncs returns template functions that need access to ModelData.
func (d *ModelData) templateFuncs() template.FuncMap {
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
	}
}

// WriterGenName returns the generator variable name for a Writer-shaped
// method. If the method's V type matches the Reader's K type, returns
// the key generator; otherwise returns the value generator.
func (d *ModelData) WriterGenName(m *suite.SpecMethodData) string {
	if d.ReaderMethod != nil && m.Shape.ValType == d.ReaderMethod.Shape.KeyType {
		return "keyGen"
	}
	return "valGen"
}

// HasDeleter reports whether a Deleter-shaped method was detected.
func (d *ModelData) HasDeleter() bool { return d.DeleterMethod != nil }

// HasCount reports whether an Aggregator-shaped method returning int was detected.
func (d *ModelData) HasCount() bool { return d.CountMethod != nil }

// HasStream reports whether a StreamReader-shaped method was detected.
func (d *ModelData) HasStream() bool { return d.StreamMethod != nil }

func buildModelData(spec *suite.SpecData, pkg *gen.Package) *ModelData {
	md := &ModelData{SpecData: spec}

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
		case gen.ShapeUnknown:
			md.SkippedMethods = append(md.SkippedMethods,
				m.Name+"("+m.Shape.Shape.String()+")")
		}
	}

	md.HasCRUD = md.ReaderMethod != nil && md.WriterMethod != nil

	// Find keyfield directive or heuristic.
	if md.HasCRUD {
		md.KeyField = findKeyField(spec, pkg)
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
func findKeyField(spec *suite.SpecData, _ *gen.Package) string {
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
// with a field named "ID" (case-insensitive).
func findIDField(m *suite.SpecMethodData) string {
	params := m.Signature.Params()
	for i := range params.Len() {
		p := params.At(i)
		if gen.IsContextType(p.Type()) {
			continue
		}
		// This is V. Dereference named types to get the struct.
		underlying := p.Type().Underlying()
		st, ok := underlying.(*types.Struct)
		if !ok {
			return "" // not a struct — can't extract key field
		}
		for j := range st.NumFields() {
			f := st.Field(j)
			if strings.EqualFold(f.Name(), "ID") {
				return f.Name()
			}
		}
		return "" // struct but no ID field
	}
	return ""
}

// isStructValType reports whether the Writer method's V parameter is a struct type.
func isStructValType(m *suite.SpecMethodData) bool {
	params := m.Signature.Params()
	for i := range params.Len() {
		p := params.At(i)
		if gen.IsContextType(p.Type()) {
			continue
		}
		_, ok := p.Type().Underlying().(*types.Struct)
		return ok
	}
	return false
}
