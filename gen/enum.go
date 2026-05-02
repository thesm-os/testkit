// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/constant"
	"strings"
)

const (
	enumGenerator     = "enum"
	enumTmplFile      = "enum.go.tmpl"
	enumDefaultSuffix = "_enum"
)

// GenerateEnum produces exhaustiveness, stringer, boundary, and
// distinctness tests for each named enum type. Returns a [Result]
// with a single test file.
func GenerateEnum(pkg *Package, typeNames []string, cfg Config, opts Options) (*Result, error) {
	errs := ValidateTypes(pkg, typeNames, KindAny)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	outputPath := opts.Output
	if outputPath == "" {
		outputPath = strings.ToLower(typeNames[0]) + enumDefaultSuffix + cfg.GeneratedSuffix + testFileSuffix
	}

	pkgName := DerivePackageName(outputPath, pkg.Pkg.Name(), cfg)

	isExternal := pkgName != pkg.Pkg.Name()
	qualifier := ""
	importPath := ""
	if isExternal {
		qualifier = pkg.Pkg.Name()
		importPath = pkg.Pkg.Path()
	}

	var enums []enumData
	for _, typeName := range typeNames {
		consts := pkg.ConstsOfType(typeName)
		if len(consts) == 0 {
			return nil, Errorf(emptyPos, "no constants found for type %s in package %s", typeName, pkg.Pkg.Name())
		}

		var values []enumValueData
		var maxVal int64
		for _, c := range consts {
			val, _ := constant.Int64Val(c.Value)
			if val > maxVal {
				maxVal = val
			}
			expectedStr := c.Name
			if c.Comment != "" {
				expectedStr = c.Comment
			}
			values = append(values, enumValueData{
				Name:        c.Name,
				ExpectedStr: expectedStr,
			})
		}

		enums = append(enums, enumData{
			TypeName:      typeName,
			QualifiedType: QualifyType(qualifier, typeName),
			Values:        values,
			MaxValue:      maxVal,
		})
	}

	data := enumTemplateData{
		PackageName:  pkgName,
		PkgQualifier: qualifier,
		ImportPath:   importPath,
		Enums:        enums,
	}

	header := Header{
		Subcommand: enumGenerator,
		Args:       enumGenerator + " " + strings.Join(typeNames, " "),
	}

	content, err := Render(templateFile(enumTmplFile), data, header)
	if err != nil {
		return nil, WrapErr(emptyPos, err, "render enum tests")
	}

	return &Result{
		Files: []OutputFile{
			{Path: outputPath, Content: content},
		},
	}, nil
}

// --- data types ---

type enumValueData struct {
	Name        string // "StatusPending"
	ExpectedStr string // "Pending" (from inline comment) or "StatusPending" (fallback)
}

type enumData struct {
	TypeName      string // "Status"
	QualifiedType string // "store.Status" or "Status"
	Values        []enumValueData
	MaxValue      int64 // highest constant value — used for boundary test
}

type enumTemplateData struct {
	PackageName  string
	PkgQualifier string
	ImportPath   string
	Enums        []enumData
}
