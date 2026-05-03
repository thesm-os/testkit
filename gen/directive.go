// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/ast"
	"go/token"
	"strings"
)

const (
	testkitDirectivePrefix  = "//testkit:"
	generateDirectivePrefix = "//go:generate testkit "
)

// Directives returns all //testkit: annotations on the doc comment
// of a top-level type declaration.
func (p *Package) Directives(objectName string) []Directive {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != objectName {
					continue
				}
				doc := ts.Doc
				if doc == nil {
					doc = gd.Doc
				}
				return parseDirectivesFromCommentGroup(doc)
			}
		}
	}
	return nil
}

// MethodDirectives returns all //testkit: annotations on a method
// of an interface or concrete type.
func (p *Package) MethodDirectives(typeName, methodName string) []Directive {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name.Name != typeName {
						continue
					}
					iface, ok := ts.Type.(*ast.InterfaceType)
					if !ok {
						continue
					}
					for _, field := range iface.Methods.List {
						if len(field.Names) > 0 && field.Names[0].Name == methodName {
							return parseDirectivesFromCommentGroup(field.Doc)
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || d.Name.Name != methodName {
					continue
				}
				if recvTypeName(d.Recv) == typeName {
					return parseDirectivesFromCommentGroup(d.Doc)
				}
			}
		}
	}
	return nil
}

// FieldDirectives returns all //testkit: annotations on a struct field.
func (p *Package) FieldDirectives(typeName, fieldName string) []Directive {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					continue
				}
				strct, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range strct.Fields.List {
					for _, name := range field.Names {
						if name.Name == fieldName {
							// Merge doc comment and inline comment directives.
							result := parseDirectivesFromCommentGroup(field.Doc)
							result = append(result, parseDirectivesFromCommentGroup(field.Comment)...)
							return result
						}
					}
				}
			}
		}
	}
	return nil
}

// VarDirectives returns all //testkit: annotations on a package-level
// variable declaration.
func (p *Package) VarDirectives(varName string) []Directive {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					if name.Name == varName {
						doc := vs.Doc
						if doc == nil {
							doc = gd.Doc
						}
						return parseDirectivesFromCommentGroup(doc)
					}
				}
			}
		}
	}
	return nil
}

// ConstDirectives returns all //testkit: annotations on a package-level
// constant declaration.
func (p *Package) ConstDirectives(varName string) []Directive {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					if name.Name == varName {
						doc := vs.Doc
						if doc == nil {
							doc = gd.Doc
						}
						return parseDirectivesFromCommentGroup(doc)
					}
				}
			}
		}
	}
	return nil
}

// EffectiveMethodDirectives returns the merged directives for a method,
// combining interface-level directives (inherited by all methods) with
// method-level directives. Interface-level directives appear first.
// A method-level directive with the same name as an interface-level
// one does NOT replace it — both are kept. Use an explicit
// "//testkit:<name>" with no args on the method to clear an inherited
// parameterised directive if needed.
func (p *Package) EffectiveMethodDirectives(typeName, methodName string) []Directive {
	inherited := p.Directives(typeName)
	method := p.MethodDirectives(typeName, methodName)
	if len(inherited) == 0 {
		return method
	}
	if len(method) == 0 {
		return inherited
	}
	merged := make([]Directive, 0, len(inherited)+len(method))
	merged = append(merged, inherited...)
	merged = append(merged, method...)
	return merged
}

// GenerateDirectives returns all //go:generate testkit directives
// found in the package's source files.
func (p *Package) GenerateDirectives() []GenerateDirective {
	var result []GenerateDirective
	for _, f := range p.Syntax {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, generateDirectivePrefix) {
					continue
				}
				d := parseGenerateDirective(c.Text)
				d.File = p.Fset.Position(c.Pos()).Filename
				d.Line = p.Fset.Position(c.Pos()).Line
				result = append(result, d)
			}
		}
	}
	return result
}

// assertDirectiveName is the batch directive that expands each
// argument into a separate no-arg directive.
//
//	//testkit:assert idempotent concurrent ctx
//
// expands to three directives: idempotent, concurrent, ctx.
const assertDirectiveName = "assert"

// parseDirectivesFromCommentGroup extracts //testkit: directives from
// a comment group. The "assert" batch directive is expanded inline.
func parseDirectivesFromCommentGroup(doc *ast.CommentGroup) []Directive {
	if doc == nil {
		return nil
	}
	var result []Directive
	for _, c := range doc.List {
		text := strings.TrimSpace(c.Text)
		if !strings.HasPrefix(text, testkitDirectivePrefix) {
			continue
		}
		body := strings.TrimPrefix(text, testkitDirectivePrefix)
		parts := strings.Fields(body)
		if len(parts) == 0 {
			continue
		}
		// Expand //testkit:assert a b c → directives a, b, c.
		if parts[0] == assertDirectiveName {
			for _, name := range parts[1:] {
				result = append(result, Directive{Name: name})
			}
			continue
		}
		result = append(result, Directive{
			Name: parts[0],
			Args: parts[1:],
		})
	}
	return result
}

// parseGenerateDirective parses a //go:generate testkit line into a
// GenerateDirective. Extracts subcommand, -o flag, and type arguments.
func parseGenerateDirective(text string) GenerateDirective {
	body := strings.TrimPrefix(text, generateDirectivePrefix)
	parts := strings.Fields(body)
	if len(parts) == 0 {
		return GenerateDirective{}
	}

	d := GenerateDirective{Generator: parts[0]}
	parts = parts[1:]

	// Parse flags and collect remaining args as type names.
	for i := 0; i < len(parts); i++ {
		if parts[i] == "-o" && i+1 < len(parts) {
			d.Output = parts[i+1]
			i++ // skip value
			continue
		}
		// Skip other flags (start with -)
		if strings.HasPrefix(parts[i], "-") {
			continue
		}
		d.Types = append(d.Types, parts[i])
	}
	return d
}
