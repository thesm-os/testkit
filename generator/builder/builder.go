// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"io/fs"
	"reflect"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/defaults"
	"go.thesmos.sh/testkit/generator/internal/emitq"
	"go.thesmos.sh/testkit/generator/internal/generic"
	"go.thesmos.sh/testkit/generator/internal/nodes"
	"go.thesmos.sh/testkit/generator/internal/witness"
)

// Name is the plugin's stable identifier.
const Name = "builder"

// Capability is the label the plugin advertises so a downstream consumer can
// declare a documentary dependency on builder generation.
const Capability = "builder"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// the plugin reads from source structs.
const DirectiveName sdk.DirectiveName = "builder"

// CompanionKey names the seeding function explicitly, for one that does not
// follow the convention or does not live beside the struct:
//
//	//testkit:builder defaults=example.com/seed.UserDefaults
//
// The full-path notation matters: a companion in another package would
// otherwise need an import written only for this directive, which does not
// compile.
const CompanionKey = "defaults"

// CompanionSuffix forms the name of the seeding function a constructor calls:
// `User` is seeded from `UserDefaults()`.
//
// Convention rather than declaration. The function is an ordinary declaration
// in the source package, so it is found by looking rather than by being told,
// and a package holding several types gets one companion each — which is why
// the name carries the type rather than being a bare `Defaults`.
const CompanionSuffix = "Defaults"

// SkipTag is the struct-tag key excluding a field from the builder:
//
//	Internal string `builder:"-"`
//
// For a field a test should never set but which cannot be unexported —
// something a neighbouring package reads directly. Any value other than `-` is
// rejected, so a typo is reported rather than silently keeping the setter.
const SkipTag = "builder"

// SkipValue is the only value [SkipTag] accepts.
const SkipValue = "-"

// SlotName is the [emit.File] slot the builders land in. `top` renders between
// the package clause and the first core decl, which is where a
// template-rendered block of whole declarations belongs.
const SlotName = "top"

// KindBuilder and KindBuilderTests are the plugin-defined emit kinds. The
// backend resolves a template by the kind's string value, so each constant
// doubles as the name the matching template defines.
const (
	KindBuilder      sdk.Kind = "builder.type"
	KindBuilderTests sdk.Kind = "builder.test"
)

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits — the projection or the templates alike.
//
// A constant rather than a digest of the templates: the version renders into
// every generated file's `Plugins:` header, so a content-derived one would
// churn the header of every output in every consuming repository on any
// template edit.
const Version = "1.0.0"

// Suffix is the trailer appended to the source type's name to form the
// builder's identifier.
const Suffix = "Builder"

// langGo is the backend language identifier the per-language adapters key on.
const langGo = golang.Language

// Plugin is the builder generator.
type Plugin struct{}

// New returns a fresh plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Version returns [Version].
func (*Plugin) Version() string { return Version }

// Priority places the plugin in the foundation bucket: a builder is a base a
// later generator may decorate, so it exists before composition runs.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorFoundation }

// Provides advertises [Capability].
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — the plugin reads source structs and depends on no
// other plugin's contribution.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `//testkit:builder` schema.
//
// The directive takes no positional argument: a builder exists exactly where
// one is declared, so deleting the line is the suppression and a negated form
// would have nothing to act on.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates a fluent builder for the annotated struct, plus a " +
					"companion test file exercising it. Takes no positional " +
					"argument. A `<Type>Defaults()` function in the same package " +
					"seeds the constructor, and per-field //testkit:default " +
					"directives override it. The negated form is rejected — a builder exists only where " +
					"declared, so removing the directive is the suppression.",
			).
			AllowedKeys(CompanionKey).
			On(node.KindStruct).
			DenyNegation().
			Build(),
	}
}

// Outputs dispatches to the per-language adapter.
func (*Plugin) Outputs(lang string) []sdk.Output {
	if lang == langGo {
		return GoOutputs()
	}
	return nil
}

// Templates dispatches to the per-language adapter's template tree.
func (*Plugin) Templates(lang string) (fs.FS, bool) {
	if lang == langGo {
		return GoTemplates()
	}
	return nil, false
}

// TemplateFuncs dispatches to the per-language adapter's funcmap.
func (*Plugin) TemplateFuncs(lang string) template.FuncMap {
	if lang == langGo {
		return GoFuncMap()
	}
	return nil
}

// TemplateOverrides returns nil — the plugin replaces no canonical funcmap
// entry.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }

// Shape classifies a field by what setter it owes, which depends on the
// field's type rather than on its name.
type Shape string

// The field shapes. Scalar is the zero value, so a projection that never
// classified reads as "one plain setter" rather than as an unhandled case.
const (
	// Scalar owes one replacing setter. Defined types land here and keep
	// their own type: a `Weekday int` setter taking `int` loses what the
	// declaration was for.
	Scalar Shape = ""

	// Slice owes a variadic replacing setter and an appending one.
	Slice Shape = "slice"

	// Bytes owes a `[]byte` setter and a string-accepting one, so a caller
	// with a string need not convert at every call site.
	Bytes Shape = "bytes"

	// Map owes a replacing setter, a single-entry setter, and a merging one.
	Map Shape = "map"

	// Set owes the same three, with the value parameter gone. A map to the
	// empty struct carries its whole meaning in its keys, so a setter asking
	// for the value asks the caller for the one thing they cannot vary.
	//
	// The value has to be an anonymous `struct{}`. A named one — `type unit
	// struct{}`, `map[string]unit` — arrives as a reference into a package this
	// generator never read, so its emptiness is not knowable here and the field
	// takes the ordinary map shape.
	Set Shape = "set"

	// Chan owes a plain setter, and a check comparing identity: a freshly made
	// channel is distinct from any the constructor could have seeded, so one
	// value proves what a comparable type needs two for.
	Chan Shape = "chan"

	// Error owes a plain setter, and a check matching by identity. An error is
	// an interface, so no value of it can be written down — but it is a builtin
	// interface with a universal constructor, which is what separates it from
	// [io.Reader] and the rest.
	Error Shape = "error"

	// Func owes a plain setter, and a check that a function arrived at all. A
	// func is not comparable, so there is nothing else to assert — but a setter
	// assigning nothing leaves nil, which is what the check catches.
	Func Shape = "func"

	// Pointer owes a setter taking the pointee by value and addressing it. A
	// pointer field distinguishes unset from zero, and the caller who wants to
	// say "set" should not have to produce an address to say it. Clearing the
	// field, or pointing two values at one address, goes through Mutate.
	Pointer Shape = "pointer"
)

// Field is one rendered struct field.
type Field struct {
	// Name is the field identifier, which is also what the setter is named
	// after — `Username` gives `WithUsername`.
	Name string

	// Type is the field's declared type. Named types arrive named: an alias is
	// its underlying type by the time the frontend records it, which is what
	// makes `Bytes = []byte` take the byte-slice setter rather than one of its
	// own.
	Type emit.Ref

	Shape Shape

	// Elem is a slice's element type or a map's value type, nil otherwise.
	Elem emit.Ref

	// Key is a map's key type, nil otherwise.
	Key emit.Ref

	// Default is the field's declared default as Go source, empty when it
	// declared none. It renders straight into the constructor's literal.
	Default string

	// DefaultRef qualifies a default naming a symbol in another package, nil
	// when the default is a plain literal. A rendered file has to register the
	// import, which only a reference carries — text cannot.
	DefaultRef *emit.Expr

	// Sample and Alternate are two distinct values of whatever the field's
	// setter takes, empty when its type admits none. See [samplesFor] for why
	// the checks need a pair rather than one value, and [resolver] for how far
	// "admits none" reaches.
	Sample    Sample
	Alternate Sample

	// Returns are a func field's return types, for the literal a check hands
	// its setter. Empty for every other shape.
	Returns []emit.Ref
}

// Builder is the emit value rendered into the primary output.
type Builder struct {
	sdk.BaseEmit

	// TypeName is the builder's identifier — `<Type>Builder`.
	TypeName string

	// SourceName is the struct's own identifier, which names the constructors:
	// `NewUser`, `NewUserFrom`.
	SourceName string

	// ValueRef qualifies the struct the builder constructs. A builder routed
	// into its own package cannot reach it unqualified, and where the two share
	// a package the backend renders it bare.
	ValueRef *emit.Expr

	// TypeParams is the source struct's type-parameter list in declaration
	// form, empty for a plain struct.
	TypeParams []*emit.TypeParam

	// TypeArgs is the same list in use position — `[K, V]`, or empty.
	TypeArgs string

	Fields []Field

	// Companion qualifies the seeding function the constructor calls, nil when
	// the package declares none. It lives in the source package, so a builder
	// routed elsewhere cannot reach it unqualified.
	Companion *emit.Expr
}

// Seeded reports whether any field declares a default, which is what decides
// whether the constructor builds a literal or an empty builder.
func (b *Builder) Seeded() bool {
	for i := range b.Fields {
		if b.Fields[i].Default != "" {
			return true
		}
	}
	return false
}

// Copies reports whether the field owns storage a clone must not share.
func (f Field) Copies() bool {
	return f.Shape == Slice || f.Shape == Bytes || f.Shape == Map || f.Shape == Set
}

// Kind returns [KindBuilder].
func (*Builder) Kind() sdk.Kind { return KindBuilder }

// Tests is the emit value rendered into the tagged test output.
//
// The companion lands in the external test package of wherever the builder was
// routed, so it reaches neither the builder nor the struct unqualified. The
// struct's package is known during Generate; the builder's is not decided until
// Layout, which is why [Tests] implements [emit.OutputPackageSetter].
type Tests struct {
	sdk.BaseEmit

	// TypeName is the builder's identifier, which names the generated check.
	TypeName string

	// SourceName is the struct's own identifier, which names the constructors
	// the check calls.
	SourceName string

	// CtorRef qualifies the builder's constructor. Set during Generate against
	// the source package as a provisional value, then corrected once routing
	// resolves — a wrong package is a compile error naming the symbol, while a
	// bare name silently binds to whatever else is in scope.
	CtorRef *emit.Expr

	// FromRef qualifies the seeding constructor, which lives beside it.
	FromRef *emit.Expr

	// ValueRef qualifies the struct the builder constructs.
	ValueRef *emit.Expr

	TypeArgs   string
	TypeParams []*emit.TypeParam

	// Witnesses are the concrete types the entry points instantiate at, empty
	// for a plain struct and for one whose constraints admit none — the latter
	// gets a note in place of its checks.
	Witnesses []emit.Ref

	Fields []Field

	// Seeded mirrors [Builder.Seeded], so the constructor's check asserts what
	// the constructor actually does rather than what it usually does.
	Seeded bool

	// Companion mirrors [Builder.Companion]. With one and no field defaults
	// the check compares the constructed value against the companion's own
	// return, which is exact — anything weaker would pass against a
	// constructor that called something else.
	Companion *emit.Expr
}

// Generic reports that the struct is parameterised and no witness could be
// found, which is the one case where no check can be written: a Go test
// function cannot take type parameters, so there is nothing to instantiate at.
func (t *Tests) Generic() bool {
	return len(t.TypeParams) > 0 && len(t.Witnesses) == 0
}

// Copies reports whether any field owns storage a clone must not share, which
// is what decides whether the independence check is emitted at all.
func (t *Tests) Copies() bool {
	for i := range t.Fields {
		if t.Fields[i].Copies() {
			return true
		}
	}
	return false
}

// Kind returns [KindBuilderTests].
func (*Tests) Kind() sdk.Kind { return KindBuilderTests }

// SetOutputPackages repoints the references at wherever Layout routed the
// builder.
func (t *Tests) SetOutputPackages(byTag map[string]string) {
	path, ok := emitq.PrimaryPackage(byTag)
	if !ok {
		return
	}
	t.CtorRef = sdk.NewExternal(path, "New"+t.SourceName)
	t.FromRef = sdk.NewExternal(path, "New"+t.SourceName+"From")
}

// Generate walks every source struct carrying `//testkit:builder` and queues
// one [Builder] against the primary output.
//
// A struct with no exported fields is skipped with a positioned diagnostic: a
// builder with no setters configures nothing, and emitting the shell would hide
// a declaration that cannot do what it says.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name, sdk.EmitTarget{})
	rv := newResolver(ctx.Reader)
	for _, s := range ctx.Reader.Structs().Slice() {
		if !s.HasPositiveDirective(DirectiveName) {
			continue
		}
		fields := fieldsOf(ctx, rv, s)
		if len(fields) == 0 {
			ctx.Diag.Errorf(s.Pos(),
				"%s: struct %q carries //testkit:%s but has no exported fields to set",
				Name, s.QName(), DirectiveName)
			continue
		}

		value := &Builder{
			BaseEmit:   emitq.Base(c, s),
			TypeName:   s.Name + Suffix,
			SourceName: s.Name,
			ValueRef:   sdk.NewExternal(s.Package, s.Name),
			TypeParams: generic.Params(s.TypeParams),
			TypeArgs:   generic.Args(s.TypeParams),
			Fields:     fields,
			Companion:  companionOf(ctx, s),
		}
		w := witness.For(s.TypeParams)

		// Queued in one call rather than two: the pair differs only in its emit
		// kind and output tag, and a second append is where the two would drift.
		if err := emitq.Append(ctx, c, SlotName, s, value, &Tests{
			BaseEmit:   emitq.Tagged(value.BaseEmit, GoTestOutputTag),
			TypeName:   value.TypeName,
			SourceName: s.Name,
			CtorRef:    sdk.NewExternal(s.Package, "New"+s.Name),
			FromRef:    sdk.NewExternal(s.Package, "New"+s.Name+"From"),
			ValueRef:   sdk.NewExternal(s.Package, s.Name),
			TypeArgs:   witness.Args(s.TypeParams),
			TypeParams: value.TypeParams,
			Fields:     substituted(fields, s.TypeParams, w),
			Seeded:     value.Seeded(),
			Companion:  value.Companion,
			Witnesses:  w,
		}); err != nil {
			return err
		}
	}
	return nil
}

// substituted rewrites each field's type with the witnesses the checks
// instantiate at, so a companion can be an ordinary non-generic test function.
//
// A Go test function cannot take type parameters, so a check naming `T` in a
// field position would not compile. Rewriting the projection is enough here
// because a builder's checks name types and nothing else — unlike a double's,
// which also name the subject's own methods.
//
// Returns fields unchanged when there is nothing to substitute, so the
// non-generic path allocates nothing.
func substituted(fields []Field, params []*node.TypeParam, witnesses []emit.Ref) []Field {
	if len(params) == 0 || len(witnesses) != len(params) {
		return fields
	}
	by := make(map[string]emit.Ref, len(params))
	for i, p := range params {
		by[p.Name] = witnesses[i]
	}
	out := make([]Field, len(fields))
	for i, f := range fields {
		f.Type = replace(f.Type, by)
		f.Elem = replace(f.Elem, by)
		f.Key = replace(f.Key, by)
		// Upgraded rather than overwritten: substitution only ever resolves a
		// type parameter into something more concrete, so it can add a pair
		// where none was derivable and must never clear one that was.
		if sample, alternate := samplesOfRef(sampleSource(f), f.Name); sample.OK() {
			f.Sample, f.Alternate = sample, alternate
		}
		out[i] = f
	}
	return out
}

// replace swaps a reference naming a type parameter for its witness,
// recursing so `[]T` and `map[K]V` are rewritten as well as a bare `T`.
func replace(r emit.Ref, by map[string]emit.Ref) emit.Ref {
	switch typed := r.(type) {
	case nil:
		return nil
	case *emit.BuiltinRef:
		if w, ok := by[typed.Name]; ok {
			return w
		}
	case *emit.CompositeRef:
		clone := *typed
		clone.Elem = replace(typed.Elem, by)
		clone.MapKey = replace(typed.MapKey, by)
		clone.MapValue = replace(typed.MapValue, by)
		return &clone
	}
	return r
}

// companionOf finds the seeding function for s, or nil when none applies.
//
// A `defaults=` key names one explicitly, in either notation [defaults.Resolve]
// accepts, which is what lets a companion live in another package — including
// one imported only for this directive. Absent the key, the convention applies:
// a `<Type>Defaults()` beside the struct.
//
// The signature is checked rather than only the name: a `UserDefaults` taking
// arguments, or returning something else, is a different function that happens
// to collide, and calling it would emit a constructor that does not compile.
func companionOf(ctx *sdk.GeneratorContext, s *node.Struct) *emit.Expr {
	for _, dir := range s.Directives() {
		if string(dir.Name) != string(DirectiveName) {
			continue
		}
		if raw := dir.KV[CompanionKey]; raw != "" {
			pkg, symbol, err := defaults.Resolve(ctx.Reader, nil, raw)
			if err != nil {
				ctx.Diag.Errorf(s.Pos(), "%s: %s on %s: %v", Name, CompanionKey, s.Name, err)
				return nil
			}
			if pkg == "" {
				pkg = s.Package
			}
			return sdk.NewExternal(pkg, symbol)
		}
	}
	name := s.Name + CompanionSuffix
	for _, fn := range ctx.Reader.Functions().Slice() {
		if fn.Name != name || fn.Package != s.Package {
			continue
		}
		if len(fn.Params) != 0 || len(fn.Returns) != 1 {
			continue
		}
		if r := fn.Returns[0].Type; r == nil || r.Name != s.Name {
			continue
		}
		return sdk.NewExternal(s.Package, name)
	}
	return nil
}

// fieldsOf lifts every field a builder can set.
//
// Unexported fields are skipped: a builder in another package cannot name them,
// and one in the same package would offer a setter the type's own invariants
// were written to prevent.
func fieldsOf(ctx *sdk.GeneratorContext, rv *resolver, s *node.Struct) []Field {
	out := make([]Field, 0, len(s.Fields)+len(s.Embeds))
	// An embedded type is a field named after itself, and the frontend records
	// it apart from the declared ones. It takes a single setter for the whole
	// value rather than promoting the fields inside it: a struct literal sets
	// an embedded value as a unit, and promoting would offer two ways to write
	// the same thing that disagree about whether the embedded value is set.
	for _, e := range s.Embeds {
		name, pointer := nodes.EmbedName(e.Type)
		if name == "" || !golang.IsExported(name) {
			continue
		}
		field := Field{Name: name, Type: golang.FromNode(e.Type)}
		source := e.Type
		if pointer {
			source = e.Type.Elem
			// Embedded by pointer takes the same setter a pointer field does:
			// the promoted fields are reachable only once the pointer is
			// non-nil, so a setter demanding an address makes every caller
			// allocate before it can set anything.
			field.Shape = Pointer
			field.Elem = golang.FromNode(e.Type.Elem)
		}
		field.Sample, field.Alternate = rv.samples(source, name, make(map[string]bool))
		out = append(out, field)
	}
	for _, f := range s.Fields {
		if !golang.IsExported(f.Name) || skipped(ctx, s, f) {
			continue
		}
		field := Field{
			Name:    f.Name,
			Type:    golang.FromNode(f.Type),
			Default: defaults.Of(f.Meta()),
		}
		if pkg := defaults.Package(f.Meta()); pkg != "" {
			field.DefaultRef = sdk.NewExternal(pkg, field.Default)
		}
		classify(rv, &field, f.Type)
		out = append(out, field)
	}
	return out
}

// skipped reports whether the field opted out of a setter.
func skipped(ctx *sdk.GeneratorContext, s *node.Struct, f *node.Field) bool {
	if f.Tag == "" {
		return false
	}
	raw, ok := reflect.StructTag(strings.Trim(f.Tag, "`")).Lookup(SkipTag)
	if !ok {
		return false
	}
	if raw != SkipValue {
		ctx.Diag.Errorf(f.Pos(),
			"%s: %s.%s carries %s:%q; the only value that excludes a field is %q",
			Name, s.Name, f.Name, SkipTag, raw, SkipValue)
		return false
	}
	return true
}

// refsOf lifts a list of source types into their emit form.
func refsOf(types []*node.TypeRef) []emit.Ref {
	if len(types) == 0 {
		return nil
	}
	out := make([]emit.Ref, len(types))
	for i, t := range types {
		out[i] = golang.FromNode(t)
	}
	return out
}

// classify records the shape the field's setter follows, and the values its
// check sets it to.
func classify(rv *resolver, field *Field, t *node.TypeRef) {
	if t == nil {
		return
	}
	seen := make(map[string]bool)
	switch {
	case t.TypeKind == node.TypeRefFunc:
		field.Shape = Func
		field.Returns = refsOf(t.FuncReturns)
	case nodes.IsBidirectionalChan(t):
		field.Shape = Chan
	case nodes.IsError(t):
		field.Shape = Error
	case t.TypeKind == node.TypeRefSlice && nodes.IsByte(t.Elem):
		field.Shape = Bytes
	case t.TypeKind == node.TypeRefSlice:
		field.Shape = Slice
		field.Elem = golang.FromNode(t.Elem)
	case t.TypeKind == node.TypeRefMap && nodes.IsEmptyStruct(t.MapValue):
		// Ahead of the map arm: a set is a map, and the narrower reading wins.
		// Elem stays nil — there is no value type worth carrying when every
		// value is the same one.
		field.Shape = Set
		field.Key = golang.FromNode(t.MapKey)
		field.Sample, field.Alternate = rv.samples(t.MapKey, field.Name, seen)
	case t.TypeKind == node.TypeRefMap:
		field.Shape = Map
		field.Key = golang.FromNode(t.MapKey)
		field.Elem = golang.FromNode(t.MapValue)
	case t.TypeKind == node.TypeRefPointer:
		field.Shape = Pointer
		field.Elem = golang.FromNode(t.Elem)
		field.Sample, field.Alternate = rv.samples(t.Elem, field.Name, seen)
	default:
		field.Sample, field.Alternate = rv.samples(t, field.Name, seen)
	}
}
