// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

// Loader caches loaded packages. Create one per testkit invocation
// and reuse across generators to avoid redundant go/packages calls.
type Loader struct {
	mu    sync.Mutex
	cache map[string]*Package // keyed by import path
}

// NewLoader returns a new [Loader] with an empty cache.
func NewLoader() *Loader {
	return &Loader{cache: make(map[string]*Package)}
}

// Load loads a Go package by pattern (import path, relative path, or
// "." for current directory). workDir is the directory to resolve
// relative patterns from — matching //go:generate behavior.
// Returns a cached result on repeated calls for the same package.
func (l *Loader) Load(pattern, workDir string) (*Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedName |
			packages.NeedModule |
			packages.NeedDeps,
		Dir: workDir,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("load package %s: %w", pattern, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for pattern %s", pattern)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package %s has errors: %w", pattern, pkg.Errors[0])
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

// Package is a loaded Go package with query methods for type
// information. Generators call the methods they need — the struct
// is stable and does not change when new generators are added.
type Package struct {
	Pkg    *types.Package
	Syntax []*ast.File
	Fset   *token.FileSet
	Info   *types.Info
	Module *packages.Module
}

// Interface looks up a named interface in the package. Returns a
// positioned [Error] if the name does not exist or is not an interface.
func (p *Package) Interface(name string) (*InterfaceInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(token.Position{}, "type %q not found in package %s", name, p.Pkg.Name())
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, Errorf(p.position(obj), "type %q is not a named type", name)
	}
	iface, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, Errorf(p.position(obj), "type %q is not an interface", name)
	}
	return p.buildInterfaceInfo(name, named, iface), nil
}

// Struct looks up a named struct in the package. Returns a positioned
// [Error] if the name does not exist or is not a struct.
func (p *Package) Struct(name string) (*StructInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(token.Position{}, "type %q not found in package %s", name, p.Pkg.Name())
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, Errorf(p.position(obj), "type %q is not a named type", name)
	}
	strct, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, Errorf(p.position(obj), "type %q is not a struct", name)
	}
	return p.buildStructInfo(name, named, strct), nil
}

// ResolveVar looks up a variable by name. The name can be a bare identifier
// (resolved in the source package) or a qualified name like "otherpkg.ErrXxx"
// (resolved in the named import). Returns the VarInfo and the import path of
// the package containing the variable (empty string for the source package).
func (p *Package) ResolveVar(name string) (*VarInfo, string, error) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 1 {
		// Bare identifier — resolve in source package.
		v, err := p.Var(name)
		return v, "", err
	}
	pkgName, varName := parts[0], parts[1]
	for _, imp := range p.Pkg.Imports() {
		if imp.Name() == pkgName {
			obj := imp.Scope().Lookup(varName)
			if obj == nil {
				return nil, "", Errorf(token.Position{}, "var %q not found in package %s", varName, pkgName)
			}
			v, ok := obj.(*types.Var)
			if !ok {
				return nil, "", Errorf(token.Position{}, "%q in package %s is not a variable", varName, pkgName)
			}
			return &VarInfo{
				Name: v.Name(),
				Type: v.Type(),
				Pos:  token.Position{},
			}, imp.Path(), nil
		}
	}
	return nil, "", Errorf(token.Position{}, "package %q not found in imports of %s", pkgName, p.Pkg.Name())
}

// Var looks up a named package-level variable. Returns a positioned
// [Error] if not found or not a variable.
func (p *Package) Var(name string) (*VarInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(token.Position{}, "var %q not found in package %s", name, p.Pkg.Name())
	}
	v, ok := obj.(*types.Var)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a variable", name)
	}
	return &VarInfo{
		Name: v.Name(),
		Type: v.Type(),
		Doc:  p.docFor(v.Name()),
		Pos:  p.position(obj),
	}, nil
}

// Const looks up a named constant. Returns a positioned [Error] if
// not found or not a constant.
func (p *Package) Const(name string) (*ConstInfo, error) {
	obj := p.Pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, Errorf(token.Position{}, "const %q not found in package %s", name, p.Pkg.Name())
	}
	c, ok := obj.(*types.Const)
	if !ok {
		return nil, Errorf(p.position(obj), "%q is not a constant", name)
	}
	return &ConstInfo{
		Name:  c.Name(),
		Type:  c.Type(),
		Value: c.Val(),
		Doc:   p.docFor(c.Name()),
		Pos:   p.position(obj),
	}, nil
}

// Interfaces returns all exported interfaces in the package, sorted
// by name.
func (p *Package) Interfaces() []*InterfaceInfo {
	var result []*InterfaceInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		if _, isTypeName := obj.(*types.TypeName); !isTypeName {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		iface, ok := named.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		result = append(result, p.buildInterfaceInfo(name, named, iface))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Structs returns all exported structs in the package, sorted by name.
func (p *Package) Structs() []*StructInfo {
	var result []*StructInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		if _, isTypeName := obj.(*types.TypeName); !isTypeName {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		strct, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		result = append(result, p.buildStructInfo(name, named, strct))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ErrorVars returns all exported package-level variables whose name
// starts with "Err", sorted by name. If sourceFile is non-empty, only
// variables declared in that file are returned (for file-scoped generation
// via $GOFILE).
func (p *Package) ErrorVars(sourceFile ...string) []*VarInfo {
	var result []*VarInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		v, ok := obj.(*types.Var)
		if !ok {
			continue
		}
		if !strings.HasPrefix(v.Name(), "Err") {
			continue
		}
		// File-scoped filter.
		if len(sourceFile) > 0 && sourceFile[0] != "" {
			pos := p.position(obj)
			if filepath.Base(pos.Filename) != sourceFile[0] {
				continue
			}
		}
		result = append(result, &VarInfo{
			Name: v.Name(),
			Type: v.Type(),
			Doc:  p.docFor(v.Name()),
			Pos:  p.position(obj),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ConstsOfType returns all exported constants whose type matches the
// named type, sorted by name.
func (p *Package) ConstsOfType(typeName string) []*ConstInfo {
	var result []*ConstInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		c, ok := obj.(*types.Const)
		if !ok {
			continue
		}
		named, ok := c.Type().(*types.Named)
		if !ok {
			continue
		}
		if named.Obj().Name() != typeName {
			continue
		}
		result = append(result, &ConstInfo{
			Name:    c.Name(),
			Type:    c.Type(),
			Value:   c.Val(),
			Doc:     p.docFor(c.Name()),
			Comment: p.inlineCommentFor(c.Name()),
			Pos:     p.position(obj),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// MethodsOn returns the method set of a named concrete type, sorted
// by name. Returns nil if the type has no methods.
func (p *Package) MethodsOn(typeName string) []*MethodInfo {
	obj := p.Pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}
	// Use pointer receiver method set to include both value and pointer methods.
	mset := types.NewMethodSet(types.NewPointer(named))
	var result []*MethodInfo
	for sel := range mset.Methods() {
		fn, ok := sel.Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		sig := fn.Type().(*types.Signature)
		result = append(result, &MethodInfo{
			Name:      fn.Name(),
			Signature: sig,
			Doc:       p.methodDoc(typeName, fn.Name()),
			Pos:       p.position(fn),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ErrorTypes returns all exported struct types that implement the error
// interface (via pointer receiver), sorted by name. These are custom
// error types like NotFoundError with an Error() string method.
func (p *Package) ErrorTypes() []*StructInfo {
	errorIface := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	var result []*StructInfo
	scope := p.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); !ok {
			continue
		}
		// Check if *T implements error.
		ptrType := types.NewPointer(named)
		if !types.Implements(ptrType, errorIface) {
			continue
		}
		result = append(result, p.buildStructInfo(name, named, named.Underlying().(*types.Struct)))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// ErrorTypeHasIs reports whether the named error type has a custom
// Is(error) bool method for matching semantics.
func (p *Package) ErrorTypeHasIs(typeName string) bool {
	obj := p.Pkg.Scope().Lookup(typeName)
	if obj == nil {
		return false
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}
	mset := types.NewMethodSet(types.NewPointer(named))
	for sel := range mset.Methods() {
		fn, ok := sel.Obj().(*types.Func)
		if !ok || fn.Name() != "Is" {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
			continue
		}
		paramIsError := IsErrorType(sig.Params().At(0).Type()) ||
			sig.Params().At(0).Type().String() == "error"
		if paramIsError {
			return true
		}
	}
	return false
}

// ErrorTypeHasUnwrap reports whether the named error type has an
// Unwrap() error method for error chain traversal.
func (p *Package) ErrorTypeHasUnwrap(typeName string) bool {
	obj := p.Pkg.Scope().Lookup(typeName)
	if obj == nil {
		return false
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}
	mset := types.NewMethodSet(types.NewPointer(named))
	for sel := range mset.Methods() {
		fn, ok := sel.Obj().(*types.Func)
		if !ok || fn.Name() != "Unwrap" {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() == 0 && sig.Results().Len() == 1 {
			return true
		}
	}
	return false
}

// --- internal helpers ---

func (p *Package) buildInterfaceInfo(name string, named *types.Named, iface *types.Interface) *InterfaceInfo {
	methods := make([]MethodInfo, 0, iface.NumMethods())
	for fn := range iface.Methods() {
		sig := fn.Type().(*types.Signature)
		methods = append(methods, MethodInfo{
			Name:      fn.Name(),
			Signature: sig,
			Doc:       p.methodDoc(name, fn.Name()),
			Pos:       p.position(fn),
		})
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })

	return &InterfaceInfo{
		Name:       name,
		Type:       iface,
		Methods:    methods,
		TypeParams: extractTypeParams(named),
		Doc:        p.docFor(name),
		Pos:        p.position(named.Obj()),
	}
}

func (p *Package) buildStructInfo(name string, named *types.Named, strct *types.Struct) *StructInfo {
	fields := make([]FieldInfo, 0, strct.NumFields())
	for i := range strct.NumFields() {
		f := strct.Field(i)
		fields = append(fields, FieldInfo{
			Name:     f.Name(),
			Type:     f.Type(),
			Exported: f.Exported(),
			Tag:      strct.Tag(i),
		})
	}

	return &StructInfo{
		Name:       name,
		Type:       strct,
		Fields:     fields,
		TypeParams: extractTypeParams(named),
		Doc:        p.docFor(name),
		Pos:        p.position(named.Obj()),
	}
}

func extractTypeParams(named *types.Named) []TypeParamInfo {
	tparams := named.TypeParams()
	if tparams == nil || tparams.Len() == 0 {
		return nil
	}
	result := make([]TypeParamInfo, tparams.Len())
	for i := range tparams.Len() {
		tp := tparams.At(i)
		result[i] = TypeParamInfo{
			Name:       tp.Obj().Name(),
			Constraint: tp.Constraint(),
		}
	}
	return result
}

// position returns the source position of a types.Object.
func (p *Package) position(obj types.Object) token.Position {
	return p.Fset.Position(obj.Pos())
}

// docFor returns the doc comment for a top-level declaration by name.
func (p *Package) docFor(name string) string {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == name {
						doc := s.Doc
						if doc == nil {
							doc = gd.Doc
						}
						if doc != nil {
							return strings.TrimSpace(doc.Text())
						}
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n.Name == name {
							doc := s.Doc
							if doc == nil {
								doc = gd.Doc
							}
							if doc != nil {
								return strings.TrimSpace(doc.Text())
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// inlineCommentFor returns the inline comment (after the declaration)
// for a named constant or variable. Returns empty if no inline comment.
func (p *Package) inlineCommentFor(name string) string {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, n := range vs.Names {
					if n.Name == name && vs.Comment != nil {
						return strings.TrimSpace(vs.Comment.Text())
					}
				}
			}
		}
	}
	return ""
}

// methodDoc returns the doc comment for a method on an interface or
// concrete type.
func (p *Package) methodDoc(typeName, methodName string) string {
	for _, f := range p.Syntax {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				// Interface methods are inside TypeSpec → InterfaceType.
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
							if field.Doc != nil {
								return strings.TrimSpace(field.Doc.Text())
							}
						}
					}
				}
			case *ast.FuncDecl:
				// Concrete type methods.
				if d.Recv == nil || d.Name.Name != methodName {
					continue
				}
				recvType := recvTypeName(d.Recv)
				if recvType == typeName && d.Doc != nil {
					return strings.TrimSpace(d.Doc.Text())
				}
			}
		}
	}
	return ""
}

// recvTypeName extracts the type name from a method receiver field list.
func recvTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	// Unwrap pointer: *T → T
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	// Generic receiver: T[K, V] → T
	if idx, ok := t.(*ast.IndexExpr); ok {
		if ident, ok := idx.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	if idx, ok := t.(*ast.IndexListExpr); ok {
		if ident, ok := idx.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}
