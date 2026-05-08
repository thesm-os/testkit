// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"go/constant"
	"go/token"
	"path/filepath"
	"strings"

	"go.thesmos.sh/testkit/generator"
)

// Analyze produces a [Data] from the loaded package and the
// user-supplied list of enum type names. Returns an error when any
// requested type has no associated constants — silent skips mask
// real bugs (typos, refactors that drop the const block) and the
// CI cost of a hard error is small.
func Analyze(pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options) (*Data, error) {
	if len(args) == 0 {
		return nil, generator.Errorf(token.Position{}, "enum: no types specified")
	}

	ctx, err := generator.BuildOutputCtx(pkg, cfg, opts)
	if err != nil {
		return nil, err
	}

	enums := make([]TypeData, 0, len(args))
	for _, typeName := range args {
		td, err := analyzeType(pkg, typeName, ctx.Qualifier)
		if err != nil {
			return nil, err
		}
		enums = append(enums, td)
	}

	return &Data{
		PackageName: ctx.PackageName,
		ImportPath:  ctx.ImportPath,
		Qualifier:   ctx.Qualifier,
		GoldenFile:  stripTestSuffix(filepath.Base(opts.Output)) + "_wire.json",
		Enums:       enums,
		// Enum consumes no //testkit: directives today. When wire-break
		// or skip-empty land, list them here so the header echoes only
		// what enum actually reads — not every package directive.
		Directives: nil,
	}, nil
}

// analyzeType produces the full [TypeData] for a single named type
// — its constants and method-presence flags. Returns an error when
// the type has no associated constants so "typo in `go:generate`"
// surfaces immediately, and when the underlying kind isn't integer
// so the wire-compat assertion never tries to emit non-compiling
// `int(<Const>)` casts.
func analyzeType(pkg *generator.Package, typeName, qualifier string) (TypeData, error) {
	consts := generator.ScanConstsOfType(pkg, typeName)
	if len(consts) == 0 {
		return TypeData{}, generator.Errorf(token.Position{},
			"enum: type %q has no constants in package %s", typeName, pkg.Name())
	}
	// Wire-compat asserts an integer mapping; the template emits
	// `int(<Const>)` casts. String-typed enums and other non-integer
	// underlying kinds would emit non-compiling code, so reject them
	// up front with a clear diagnostic instead of producing
	// nonsense that fails downstream.
	if k := consts[0].Value.Kind(); k != constant.Int {
		return TypeData{}, generator.Errorf(token.Position{},
			"enum: type %q has %s-kind constants; only integer enums are supported",
			typeName, k)
	}

	parseFunc := "Parse" + typeName
	td := TypeData{
		TypeName:         typeName,
		Qualifier:        qualifier,
		HasString:        generator.HasMethod(pkg, typeName, "String", generator.StringerSig),
		HasParse:         generator.HasFunc(pkg, parseFunc, generator.ParseSig(typeName)),
		ParseFunc:        parseFunc,
		HasMarshalText:   hasTextMarshalRoundTrip(pkg, typeName),
		HasMarshalJSON:   hasJSONMarshalRoundTrip(pkg, typeName),
		HasMarshalBinary: hasBinaryMarshalRoundTrip(pkg, typeName),
	}

	for _, c := range consts {
		var intVal int64
		if c.Value.Kind() == constant.Int {
			v, _ := constant.Int64Val(c.Value)
			intVal = v
			if v > td.MaxValue {
				td.MaxValue = v
			}
		}
		expected := c.Comment
		if expected == "" {
			expected = stripPrefix(c.Name, typeName)
		}
		td.Values = append(td.Values, Value{
			Name:        c.Name,
			ExpectedStr: expected,
			IntValue:    intVal,
		})
		if intVal == 0 && td.ZeroValueName == "" {
			td.ZeroValueName = c.Name
		}
	}
	return td, nil
}

// hasTextMarshalRoundTrip reports whether the type implements both
// halves of [encoding.TextMarshaler] / [encoding.TextUnmarshaler].
// A one-sided implementation can't round-trip, so the marshal
// subtests are gated on both.
func hasTextMarshalRoundTrip(pkg *generator.Package, typeName string) bool {
	return generator.HasMethod(pkg, typeName, "MarshalText", generator.MarshalTextSig) &&
		generator.HasMethod(pkg, typeName, "UnmarshalText", generator.UnmarshalTextSig)
}

// hasJSONMarshalRoundTrip is the JSON equivalent of
// [hasTextMarshalRoundTrip].
func hasJSONMarshalRoundTrip(pkg *generator.Package, typeName string) bool {
	return generator.HasMethod(pkg, typeName, "MarshalJSON", generator.MarshalJSONSig) &&
		generator.HasMethod(pkg, typeName, "UnmarshalJSON", generator.UnmarshalJSONSig)
}

// hasBinaryMarshalRoundTrip is the binary equivalent of
// [hasTextMarshalRoundTrip] — requires both halves of the
// [encoding.BinaryMarshaler] / [encoding.BinaryUnmarshaler] pair.
func hasBinaryMarshalRoundTrip(pkg *generator.Package, typeName string) bool {
	return generator.HasMethod(pkg, typeName, "MarshalBinary", generator.MarshalBinarySig) &&
		generator.HasMethod(pkg, typeName, "UnmarshalBinary", generator.UnmarshalBinarySig)
}

// stripPrefix removes the type-name prefix from a const identifier,
// trimming any trailing underscore so SNAKE_CASE-ish names render
// cleanly:
//
//	"StatusPending"   with type "Status" → "Pending"
//	"Status_Pending"  with type "Status" → "Pending"
//	"OtherName"       with type "Status" → "OtherName" (no prefix to strip)
func stripPrefix(constName, typeName string) string {
	rest, ok := strings.CutPrefix(constName, typeName)
	if !ok {
		return constName
	}
	return strings.TrimPrefix(rest, "_")
}

// stripTestSuffix returns the basename with `_test.go` removed (or
// `.go` as a fallback). The result is the prefix used for per-type
// wire-golden filenames so they group with the emitted test file in
// directory listings.
func stripTestSuffix(name string) string {
	if base, ok := strings.CutSuffix(name, "_test.go"); ok {
		return base
	}
	return strings.TrimSuffix(name, ".go")
}
