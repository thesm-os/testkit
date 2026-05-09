// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package partition registers the //testkit:partition consumer.
// The directive declares that the method's call recorder must
// shard by a named field — different partition values get
// independent recorders, mirroring real-world per-tenant /
// per-key isolation.
//
// Directive form:
//
//	//testkit:partition TenantID
//
// Validation: the field name must resolve to either:
//
//   - a top-level non-ctx parameter (rare — matches when the
//     parameter name equals the directive arg), or
//   - an exported field on a struct-typed non-ctx parameter.
//
// The resolved [Payload.FieldPath] is the dotted access expression
// templates use to read the partition value at call time
// ("Item.TenantID" for a struct field, "TenantID" for a direct
// parameter).
package partition

import (
	"fmt"
	"go/types"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved field path + type for templates.
type Payload struct {
	// FieldPath is the dotted access expression: "TenantID" when
	// the field IS a non-ctx parameter, or "Item.TenantID" when
	// the field lives on the parameter named "Item".
	FieldPath string

	// FieldName is the bare name (always equals the directive arg).
	FieldName string

	// FieldType is the rendered Go type of the resolved field,
	// qualified via the package's import tracker.
	FieldType string

	// ZeroKey is the rendered zero literal for [FieldType] — the
	// value [generator.ZeroValueOf] produces for the resolved
	// field's type. Auto-tests use it for `FaultForPartition(zero,
	// ...)` against a method invoked with [spec.Method.ZeroArgs]
	// (whose partition slot is also zero, so the predicate
	// matches).
	ZeroKey string

	// SampleKey is a non-zero sample literal for [FieldType] —
	// produced via [generator.SampleValueOf]. Auto-tests use it
	// for `FaultForOtherPartitions(sample, ...)`: a method invoked
	// with zero-args has a zero partition slot, which is != sample,
	// so the non-matching predicate fires.
	SampleKey string
}

func init() {
	spec.RegisterConsumer(directive.Partition, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:partition directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Partition)
}

// Has reports whether the method has a partition directive.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Partition) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("partition: %w", err)
	}
	field := dir.Args[0]

	// Walk non-ctx params. Direct match (param name == field) wins
	// over nested struct match — same precedence as the legacy
	// generator so existing fixtures keep their resolution.
	params := method.Signature.Params()
	n := params.Len()
	if method.Signature.Variadic() {
		n--
	}

	for i := range n {
		p := params.At(i)
		if generator.IsContextType(p.Type()) {
			continue
		}
		if p.Name() == field {
			spec.Set(&method.Attachments, directive.Partition, Payload{
				FieldPath: generator.Title(field),
				FieldName: field,
				FieldType: types.TypeString(p.Type(), data.Tracker.Qualifier()),
				ZeroKey:   generator.ZeroValueOf(p.Type(), data.Tracker),
				SampleKey: generator.SampleValueOf(p.Type(), field, data.Tracker),
			})
			return nil
		}
	}

	for i := range n {
		p := params.At(i)
		if generator.IsContextType(p.Type()) {
			continue
		}
		st, ok := p.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		paramName := p.Name()
		if paramName == "" {
			paramName = generator.ParamName(i)
		}
		for f := range st.Fields() {
			if f.Name() != field || !f.Exported() {
				continue
			}
			spec.Set(&method.Attachments, directive.Partition, Payload{
				FieldPath: generator.Title(paramName) + "." + field,
				FieldName: field,
				FieldType: types.TypeString(f.Type(), data.Tracker.Qualifier()),
				ZeroKey:   generator.ZeroValueOf(f.Type(), data.Tracker),
				SampleKey: generator.SampleValueOf(f.Type(), field, data.Tracker),
			})
			return nil
		}
	}

	return fmt.Errorf("partition: field %q not found in parameters of %s",
		field, method.Name)
}
