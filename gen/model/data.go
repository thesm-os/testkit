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

	// PureMethods lists all Pure-shaped methods (for law emission).
	PureMethods []*suite.SpecMethodData

	// PredicateMethods lists all Predicate-shaped methods (for law emission).
	PredicateMethods []*suite.SpecMethodData

	// StreamMethods lists all StreamReader-shaped methods (for law emission).
	StreamMethods []*suite.SpecMethodData

	// AutoLaws lists the auto-derived law names that will be emitted.
	AutoLaws []string

	// SkippedMethods lists methods with Unknown shape that aren't
	// covered by auto-derived actions.
	SkippedMethods []string

	// HasChain is true when at least //testkit:appends is detected.
	HasChain bool

	// ChainAppendMethod is the method with //testkit:appends directive.
	ChainAppendMethod *suite.SpecMethodData

	// ChainReplayMethod is the method with //testkit:replays directive.
	ChainReplayMethod *suite.SpecMethodData

	// ChainVerifyMethod is the method with //testkit:verifies directive.
	// Nil when absent; framework uses PoisonAccessor Err() instead.
	ChainVerifyMethod *suite.SpecMethodData

	// ChainPartitionField is the Entry struct field for partition key extraction.
	// Set from //testkit:partition-by=Field on the Replay method.
	ChainPartitionField string

	// ChainEntryIDField is the Entry struct field for unique entry ID.
	// Set from //testkit:entry-id=Field on the Replay method.
	ChainEntryIDField string

	// ChainDependsOnField is the Entry struct field for dependency list.
	// Set from //testkit:depends-on=Field on the Replay method.
	ChainDependsOnField string

	// ChainHashFunc is the qualified hash function override.
	// Set from //testkit:hash=PkgPath.FuncName on the interface.
	ChainHashFunc string

	// IsTimeAware is true when //testkit:time-aware directive is present.
	// When true, the generator emits clock factory option, dual TestClock
	// setup, and AdvanceClock action.
	IsTimeAware bool

	// CanLinearize is true when the interface has Reader + (Writer or Deleter)
	// shapes — the only combination linearize.KV currently models.
	//
	// Non-CRUD interfaces (Mutator, ReaderWithBool, Lookup, etc.) do NOT
	// get XxxModelConcurrent emitted. Concurrent stress for those shapes
	// requires manual WithConcurrent wiring (StressActions only, no Porcupine).
	// See model.md "Non-CRUD concurrent stress emission" deferral.
	CanLinearize bool

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
			return "nil"
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

// IsChainPartitioned reports whether the chain is per-partition.
func (d *Data) IsChainPartitioned() bool { return d.ChainPartitionField != "" }

// HasCausalOrdering reports whether causal ordering directives are present.
func (d *Data) HasCausalOrdering() bool {
	return d.ChainEntryIDField != "" && d.ChainDependsOnField != ""
}

// HasPure reports whether any Pure-shaped methods were detected.
func (d *Data) HasPure() bool { return len(d.PureMethods) > 0 }

// HasPredicate reports whether any Predicate-shaped methods were detected.
func (d *Data) HasPredicate() bool { return len(d.PredicateMethods) > 0 }

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
			md.StreamMethods = append(md.StreamMethods, m)
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
		case gen.ShapePure:
			md.PureMethods = append(md.PureMethods, m)
		case gen.ShapePredicate:
			md.PredicateMethods = append(md.PredicateMethods, m)
		case gen.ShapeLifecycle:
			// Handled by template; no first-method tracking needed.
		case gen.ShapeUnknown:
			md.SkippedMethods = append(md.SkippedMethods,
				m.Name+"("+m.Shape.Shape.String()+")")
		}
	}

	md.HasCRUD = md.ReaderMethod != nil && md.WriterMethod != nil
	// CanLinearize requires Reader (not ReaderWithBool/Lookup) + Writer/Deleter.
	// Extended shapes (Mutator, ReaderWithBool, Lookup) are stress-only
	// under concurrent mode — no Porcupine linearizability check.
	hasReader := false
	hasWriterOrDeleter := false
	for _, m := range spec.Methods {
		if m.Skip {
			continue
		}
		switch m.Shape.Shape { //nolint:exhaustive // only Reader/Writer/Deleter matter for linearizability
		case gen.ShapeReader:
			hasReader = true
		case gen.ShapeWriter, gen.ShapeDeleter:
			hasWriterOrDeleter = true
		}
	}
	md.CanLinearize = hasReader && hasWriterOrDeleter

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
	if md.HasPure() {
		md.AutoLaws = append(md.AutoLaws, "AUTO-PURE-DETERMINISTIC")
	}
	if md.HasPredicate() {
		md.AutoLaws = append(md.AutoLaws, "AUTO-PREDICATE-CONSISTENT")
	}
	if md.HasStream() {
		md.AutoLaws = append(md.AutoLaws, "AUTO-STREAM-REENTRANT")
	}

	// Detect chain directives.
	for _, m := range spec.Methods {
		for _, d := range m.Directives {
			switch d.Name {
			case directives.Appends:
				md.HasChain = true
				md.ChainAppendMethod = m
			case directives.Replays:
				md.ChainReplayMethod = m
			case directives.Verifies:
				md.ChainVerifyMethod = m
			case directives.PartitionBy:
				if len(d.Args) > 0 {
					md.ChainPartitionField = d.Args[0]
				}
			case directives.EntryIDField:
				if len(d.Args) > 0 {
					md.ChainEntryIDField = d.Args[0]
				}
			case directives.DependsOnField:
				if len(d.Args) > 0 {
					md.ChainDependsOnField = d.Args[0]
				}
			case directives.HashFunc:
				if len(d.Args) > 0 {
					md.ChainHashFunc = d.Args[0]
				}
			case directives.TimeAware:
				md.IsTimeAware = true
			}
		}
	}
	if md.HasChain {
		md.AutoLaws = append(md.AutoLaws, "AUTO-APPEND-ONLY-GROWS", "AUTO-HASH-CHAIN-INTEGRITY")
		if md.ChainReplayMethod != nil {
			md.AutoLaws = append(md.AutoLaws,
				"AUTO-APPEND-ONLY-NO-DROPS", "AUTO-REPLAY-DETERMINISTIC")
		}
		if md.HasCausalOrdering() {
			md.AutoLaws = append(md.AutoLaws, "AUTO-REPLAY-CAUSAL-ORDERING")
		}
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

// validateChainShape checks chain-specific constraints at codegen time.
// Returns a positioned error with directive-specific guidance.
func validateChainShape(d *Data, spec *suite.SpecData) error {
	if !d.HasChain {
		return nil
	}

	// Rule 1: Mutator + chain without poison surface.
	if d.ChainAppendMethod != nil && d.ChainAppendMethod.IsMutator() {
		hasPoisonOrVerify := d.ChainVerifyMethod != nil
		for _, m := range spec.Methods {
			if m.IsPoisonAccessor() {
				hasPoisonOrVerify = true
				break
			}
		}
		if !hasPoisonOrVerify {
			return gen.Errorf(d.ChainAppendMethod.Pos,
				"//testkit:appends on Mutator-shaped method %s requires either "+
					"//testkit:verifies on a Verify method or an Err() error method "+
					"for hash chain integrity checking",
				d.ChainAppendMethod.Name)
		}
	}

	// Rule 2: depends-on without entry-id (or vice versa).
	if (d.ChainEntryIDField == "") != (d.ChainDependsOnField == "") {
		pos := d.ChainAppendMethod.Pos
		if d.ChainReplayMethod != nil {
			pos = d.ChainReplayMethod.Pos
		}
		return gen.Errorf(pos,
			"//testkit:entry-id and //testkit:depends-on must both be present "+
				"or both absent; found entry-id=%q depends-on=%q",
			d.ChainEntryIDField, d.ChainDependsOnField)
	}

	// Rule 3: partition-by without replays directive.
	if d.ChainPartitionField != "" && d.ChainReplayMethod == nil {
		return gen.Errorf(d.ChainAppendMethod.Pos,
			"//testkit:partition-by=%s requires a method with //testkit:replays",
			d.ChainPartitionField)
	}

	// Rule 4: causal ordering without replays.
	if d.HasCausalOrdering() && d.ChainReplayMethod == nil {
		return gen.Errorf(d.ChainAppendMethod.Pos,
			"//testkit:entry-id and //testkit:depends-on require a method with //testkit:replays")
	}

	return nil
}

// validateTimeAware checks time-aware constraints at codegen time.
func validateTimeAware(d *Data, spec *suite.SpecData) error {
	if !d.IsTimeAware {
		return nil
	}

	// Rule: time-aware without a Reader-shaped method. TTL expiry is
	// only detectable via Read returning the sentinel error. Without
	// a Reader, clock advancement has no observable effect through the
	// generated property.
	if d.ReaderMethod == nil {
		// Find the position of the time-aware directive for the error.
		for _, m := range spec.Methods {
			for _, dir := range m.Directives {
				if dir.Name == directives.TimeAware {
					return gen.Errorf(m.Pos,
						"//testkit:time-aware requires at least one Reader-shaped method "+
							"so clock advancement has observable effect (TTL expiry, "+
							"deadline-driven reads)")
				}
			}
		}
	}
	return nil
}
