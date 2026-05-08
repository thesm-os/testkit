// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"go.thesmos.sh/testkit/generator/directive"
)

// Loader caches loaded Go packages. Create one Loader per testkit
// invocation and reuse across generators — successive lookups of the
// same package hit the cache.
//
// The Loader is the single seam between the generator pipeline and
// go/types. Generators receive a [*Package] and never call
// packages.Load themselves.
type Loader struct {
	mu    sync.Mutex
	cache map[string]*Package // keyed by import path
}

// NewLoader creates an empty [Loader].
func NewLoader() *Loader {
	return &Loader{cache: make(map[string]*Package)}
}

// Load loads a Go package by pattern (import path, relative path, or
// "." for current). workDir resolves relative patterns — matching
// //go:generate behavior. Returns a cached result on repeated calls.
//
// Errors include go/packages diagnostic errors so type-check failures
// surface at the boundary instead of as nil panics later.
func (l *Loader) Load(pattern, workDir string) (*Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedName |
			packages.NeedModule |
			packages.NeedDeps |
			packages.NeedImports,
		Dir: workDir,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("load package %q: %w", pattern, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for pattern %q", pattern)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package %q has errors: %w", pattern, pkg.Errors[0])
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if cached, ok := l.cache[pkg.PkgPath]; ok {
		return cached, nil
	}
	p := &Package{
		Pkg:    pkg.Types,
		Syntax: pkg.Syntax,
		Fset:   pkg.Fset,
		Info:   pkg.TypesInfo,
		Module: pkg.Module,
	}
	l.cache[pkg.PkgPath] = p
	return p, nil
}

// Package is a loaded Go package. Methods on Package expose the generic
// Go type-system queries every generator needs (Interface, Struct, Var,
// Const, Methods, listings, doc helpers). Generator-specific queries —
// like sentinel's Err* scan or enum's iota walk — live in the
// generator's own package, taking a *Package as input.
type Package struct {
	Pkg    *types.Package
	Syntax []*ast.File
	Fset   *token.FileSet
	Info   *types.Info
	Module *packages.Module
}

// Path returns the package's import path.
func (p *Package) Path() string {
	if p.Pkg == nil {
		return ""
	}
	return p.Pkg.Path()
}

// Name returns the package's Go identifier.
func (p *Package) Name() string {
	if p.Pkg == nil {
		return ""
	}
	return p.Pkg.Name()
}

// PackageDirectives returns every //testkit: directive declared on
// the package-level doc comment of any of the package's files.
//
// Directives at this scope are package-wide (e.g. sentinel's
// //testkit:sentinel-no-overlap-with). Per-method directives live on
// [MethodInfo.Directives]; per-field on [FieldInfo.Directives];
// per-type on [InterfaceInfo.Directives] / [StructInfo.Directives].
func (p *Package) PackageDirectives() []directive.Directive {
	var out []directive.Directive
	for _, f := range p.Syntax {
		if f.Doc == nil {
			continue
		}
		out = append(out, parseDirectivesFromDoc(rawDocText(f.Doc), p.Fset.Position(f.Package))...)
	}
	return out
}

// Interface looks up a named interface. Returns a positioned [Error]
// when the name is not found or refers to a non-interface type.
func (p *Package) Interface(name string) (*InterfaceInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(noPos, "type %q not found in package %s", name, p.Pkg.Name())
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a type", name)
	}
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a named type", name)
	}
	iface, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not an interface (got %s)", name, named.Underlying())
	}
	return p.buildInterfaceInfo(name, named, iface), nil
}

// Struct looks up a named struct. Returns a positioned [Error] when
// the name is not found or refers to a non-struct type.
func (p *Package) Struct(name string) (*StructInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(noPos, "type %q not found in package %s", name, p.Pkg.Name())
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a type", name)
	}
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a named type", name)
	}
	strct, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a struct", name)
	}
	return p.buildStructInfo(name, named, strct), nil
}

// Var looks up a package-level variable. Returns a positioned [Error]
// when the name is not found or refers to a non-variable.
func (p *Package) Var(name string) (*VarInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(noPos, "var %q not found in package %s", name, p.Pkg.Name())
	}
	v, ok := obj.(*types.Var)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a variable", name)
	}
	return &VarInfo{
		Name: v.Name(),
		Type: v.Type(),
		Doc:  p.docFor(name),
		Pos:  p.position(v),
	}, nil
}

// ResolveVar resolves a possibly-qualified variable reference. Returns
// the variable info, the import path of its containing package
// (empty for the local package), and an error if not found.
//
// Accepts:
//
//	"ErrNotFound"            — local package
//	"errors.New"             — would fail (not a var)
//	"otherpkg.ErrSomething"  — must already be imported transitively
//
// Used by directive enrichers that resolve sentinel names.
func (p *Package) ResolveVar(name string) (*VarInfo, string, error) {
	if !strings.Contains(name, ".") {
		v, err := p.Var(name)
		return v, "", err
	}
	// Qualified — split into pkgName.varName and look up the import.
	parts := strings.SplitN(name, ".", 2)
	pkgName, varName := parts[0], parts[1]
	for _, imp := range p.Pkg.Imports() {
		if imp.Name() == pkgName {
			obj := imp.Scope().Lookup(varName)
			if obj == nil {
				return nil, imp.Path(), Errorf(noPos, "var %q not found in %s", varName, imp.Path())
			}
			v, ok := obj.(*types.Var)
			if !ok {
				return nil, imp.Path(), Errorf(noPos, "%q in %s is not a variable", varName, imp.Path())
			}
			return &VarInfo{
				Name: v.Name(),
				Type: v.Type(),
				Pos:  p.Fset.Position(v.Pos()),
			}, imp.Path(), nil
		}
	}
	return nil, "", Errorf(noPos, "package %q not imported", pkgName)
}

// Const looks up a package-level constant.
func (p *Package) Const(name string) (*ConstInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(noPos, "const %q not found in package %s", name, p.Pkg.Name())
	}
	c, ok := obj.(*types.Const)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a constant", name)
	}
	return &ConstInfo{
		Name:  c.Name(),
		Type:  c.Type(),
		Value: c.Val(),
		Doc:   p.docFor(name),
		Pos:   p.position(c),
	}, nil
}

// Interfaces returns all named interfaces in the package, sorted by name.
func (p *Package) Interfaces() []*InterfaceInfo {
	var out []*InterfaceInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		iface, ok := named.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		out = append(out, p.buildInterfaceInfo(name, named, iface))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Structs returns all named structs in the package, sorted by name.
func (p *Package) Structs() []*StructInfo {
	var out []*StructInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		strct, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		out = append(out, p.buildStructInfo(name, named, strct))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// MethodsOn returns the method set of a named concrete type. Useful
// for suite generation that consumes directives on concrete-method
// doc comments rather than interface methods.
func (p *Package) MethodsOn(typeName string) []*MethodInfo {
	obj := p.Pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil
	}
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return nil
	}
	var out []*MethodInfo
	for m := range named.Methods() {
		sig, ok := m.Type().(*types.Signature)
		if !ok {
			continue
		}
		out = append(out, &MethodInfo{
			Name:       m.Name(),
			Signature:  sig,
			Doc:        p.methodDoc(typeName, m.Name()),
			Directives: p.parseMethodDirectives(typeName, m.Name()),
			Pos:        p.position(m),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// buildInterfaceInfo composes an [InterfaceInfo] from raw go/types
// data, flattening embedded interfaces and sorting methods by name.
func (p *Package) buildInterfaceInfo(name string, named *types.Named, iface *types.Interface) *InterfaceInfo {
	info := &InterfaceInfo{
		Name:       name,
		Type:       iface,
		Doc:        p.docFor(name),
		Directives: p.parseTypeDirectives(name),
		Pos:        p.position(named.Obj()),
	}

	// Type parameters.
	if tparams := named.TypeParams(); tparams != nil {
		for tp := range tparams.TypeParams() {
			info.TypeParams = append(info.TypeParams, TypeParamInfo{
				Name:       tp.Obj().Name(),
				Constraint: tp.Constraint(),
			})
		}
	}

	// Methods (Methods iterates promoted methods from embedded interfaces too).
	for m := range iface.Methods() {
		sig, ok := m.Type().(*types.Signature)
		if !ok {
			continue
		}
		info.Methods = append(info.Methods, MethodInfo{
			Name:       m.Name(),
			Signature:  sig,
			Doc:        p.interfaceMethodDoc(name, m.Name()),
			Directives: p.parseInterfaceMethodDirectives(name, m.Name()),
			Pos:        p.position(m),
		})
	}
	sort.Slice(info.Methods, func(i, j int) bool { return info.Methods[i].Name < info.Methods[j].Name })
	return info
}

// buildStructInfo composes a [StructInfo] preserving field declaration
// order.
func (p *Package) buildStructInfo(name string, named *types.Named, strct *types.Struct) *StructInfo {
	info := &StructInfo{
		Name:       name,
		Type:       strct,
		Doc:        p.docFor(name),
		Directives: p.parseTypeDirectives(name),
		Pos:        p.position(named.Obj()),
	}
	if tparams := named.TypeParams(); tparams != nil {
		for tp := range tparams.TypeParams() {
			info.TypeParams = append(info.TypeParams, TypeParamInfo{
				Name:       tp.Obj().Name(),
				Constraint: tp.Constraint(),
			})
		}
	}
	for i := range strct.NumFields() {
		f := strct.Field(i)
		info.Fields = append(info.Fields, FieldInfo{
			Name:          f.Name(),
			Type:          f.Type(),
			Exported:      f.Exported(),
			Tag:           strct.Tag(i),
			Directives:    p.parseFieldDirectives(name, f.Name()),
			InlineComment: p.inlineCommentFor(name, f.Name()),
		})
	}
	return info
}

// position returns the source position for a types.Object.
func (p *Package) position(obj types.Object) token.Position {
	return p.Fset.Position(obj.Pos())
}

// docFor returns the leading doc comment of a top-level declaration,
// or empty string when no comment is attached. Stripped of "// "
// prefixes; multi-line comments retain their newlines.
func (p *Package) docFor(name string) string {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Doc == nil {
					continue
				}
				for _, spec := range d.Specs {
					if matchesSpec(spec, name) {
						return rawDocText(d.Doc)
					}
				}
			case *ast.FuncDecl:
				if d.Name.Name == name && d.Doc != nil {
					return rawDocText(d.Doc)
				}
			}
		}
	}
	return ""
}

// inlineCommentFor returns the trailing line comment after a struct
// field declaration, or empty.
func (p *Package) inlineCommentFor(typeName, fieldName string) string {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					for _, n := range field.Names {
						if n.Name == fieldName && field.Comment != nil {
							return strings.TrimRight(field.Comment.Text(), "\n")
						}
					}
				}
			}
		}
	}
	return ""
}

// rawDocText returns the raw text of a [*ast.CommentGroup] preserving
// every line's leading "//" — including directive-style lines that
// [ast.CommentGroup.Text] silently filters out.
//
// We need the raw form because [ast.CommentGroup.Text] applies
// `isDirective` filtering (lines like "//go:build", "//testkit:errors")
// — exactly the lines [parseDirectivesFromDoc] needs to read.
func rawDocText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	parts := make([]string, len(g.List))
	for i, c := range g.List {
		parts[i] = c.Text
	}
	return strings.Join(parts, "\n")
}

// methodDoc returns the doc comment of a concrete-type method.
func (p *Package) methodDoc(typeName, methodName string) string {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Name.Name != methodName {
				continue
			}
			if recvName(fd.Recv) == typeName && fd.Doc != nil {
				return rawDocText(fd.Doc)
			}
		}
	}
	return ""
}

// interfaceMethodDoc returns the doc comment of an interface method
// declaration, including //testkit: directives that immediately precede
// the method (separated only by whitespace, not blank lines).
func (p *Package) interfaceMethodDoc(typeName, methodName string) string {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					continue
				}
				it, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				for _, field := range it.Methods.List {
					for _, n := range field.Names {
						if n.Name == methodName && field.Doc != nil {
							return rawDocText(field.Doc)
						}
					}
				}
			}
		}
	}
	return ""
}

// parseTypeDirectives extracts //testkit: directives from a top-level
// type's doc comment.
func (p *Package) parseTypeDirectives(name string) []directive.Directive {
	doc := p.docFor(name)
	if doc == "" {
		return nil
	}
	return parseDirectivesFromDoc(doc, p.declPosition(name))
}

// parseInterfaceMethodDirectives extracts //testkit: directives from
// an interface method's doc comment.
func (p *Package) parseInterfaceMethodDirectives(typeName, methodName string) []directive.Directive {
	doc := p.interfaceMethodDoc(typeName, methodName)
	if doc == "" {
		return nil
	}
	return parseDirectivesFromDoc(doc, p.interfaceMethodPosition(typeName, methodName))
}

// parseMethodDirectives extracts //testkit: directives from a concrete-
// type method's doc comment.
func (p *Package) parseMethodDirectives(typeName, methodName string) []directive.Directive {
	doc := p.methodDoc(typeName, methodName)
	if doc == "" {
		return nil
	}
	return parseDirectivesFromDoc(doc, p.methodPosition(typeName, methodName))
}

// parseFieldDirectives extracts //testkit: directives from a struct
// field's doc comment AND inline comment (rare, but allowed for
// directives like //testkit:default that fit on one line).
func (p *Package) parseFieldDirectives(typeName, fieldName string) []directive.Directive {
	var out []directive.Directive
	if doc := p.fieldDoc(typeName, fieldName); doc != "" {
		out = append(out, parseDirectivesFromDoc(doc, p.fieldPosition(typeName, fieldName))...)
	}
	if inline := p.inlineCommentFor(typeName, fieldName); inline != "" {
		out = append(out, parseDirectivesFromDoc(inline, p.fieldPosition(typeName, fieldName))...)
	}
	return out
}

// fieldDoc returns the doc comment for a struct field.
func (p *Package) fieldDoc(typeName, fieldName string) string {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					for _, n := range field.Names {
						if n.Name == fieldName && field.Doc != nil {
							return rawDocText(field.Doc)
						}
					}
				}
			}
		}
	}
	return ""
}

// declPosition returns the position of a top-level declaration.
func (p *Package) declPosition(name string) token.Position {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return token.Position{}
	}
	return p.position(obj)
}

// methodPosition returns the position of a concrete-type method.
func (p *Package) methodPosition(typeName, methodName string) token.Position {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Name.Name != methodName {
				continue
			}
			if recvName(fd.Recv) == typeName {
				return p.Fset.Position(fd.Pos())
			}
		}
	}
	return token.Position{}
}

// interfaceMethodPosition returns the position of an interface method.
func (p *Package) interfaceMethodPosition(typeName, methodName string) token.Position {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					continue
				}
				it, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				for _, field := range it.Methods.List {
					for _, n := range field.Names {
						if n.Name == methodName {
							return p.Fset.Position(field.Pos())
						}
					}
				}
			}
		}
	}
	return token.Position{}
}

// fieldPosition returns the position of a struct field.
func (p *Package) fieldPosition(typeName, fieldName string) token.Position {
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					for _, n := range field.Names {
						if n.Name == fieldName {
							return p.Fset.Position(field.Pos())
						}
					}
				}
			}
		}
	}
	return token.Position{}
}

// matchesSpec reports whether a top-level spec declares the named
// identifier.
func matchesSpec(spec ast.Spec, name string) bool {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		return s.Name.Name == name
	case *ast.ValueSpec:
		for _, n := range s.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

// recvName extracts the receiver type name from a method's receiver
// list, stripping any leading pointer indirection.
func recvName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}
