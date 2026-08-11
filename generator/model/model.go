// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// Name is the plugin's stable identifier.
const Name = "model"

// Capability is the label the plugin advertises.
const Capability = "model"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits, the projection or the templates alike.
const Version = "0.2.0"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts an interface in.
const DirectiveName sdk.DirectiveName = "model"

// RefKey names a constructor in the source package that builds the reference,
// for an interface whose shape no shipped oracle models.
const RefKey = "ref"

// TierName is the path the model run reports under inside the contract entry,
// and the one `<Iface>Without` drops it by.
const TierName = "model"

// SlotName is the [sdk.EmitFile] slot the bindings land in.
const SlotName = "top"

// KindBindings is the emit kind of the one value queued per interface, and
// therefore the template that renders the file.
const KindBindings sdk.Kind = "model.bindings"

// ActionKindPrefix composes each action's emit kind — `model.action.<shape>` —
// which is the template that renders its constructor call.
const ActionKindPrefix = "model.action."

// The two shared pool locals the generated property declares. Every draw in
// the file goes through one of them, which is what keeps a law's values
// colliding with the sequences it runs beside.
const (
	poolKeys   = "keys"
	poolValues = "values"
)

// shapeCompositeWriter is the one detector spelling this plugin branches on
// beyond its template dispatch: the keyed put draws from both pools and
// selects the keyed oracle.
const shapeCompositeWriter = "compositewriter"

// Plugin is the model-tier generator: it turns an interface's classifications
// into a property-based state-machine run inside the generated contract entry.
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// [sdk.GeneratorCrossCutting], one bucket after the harness's
// [sdk.GeneratorComposition] — the cross-bucket ordering is what guarantees
// the projection is queued before this reads it. The Requires is documentary:
// eidos's sorter ignores a dependency naming an earlier bucket.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplatesFS, GoOutputs()...).
		Version(Version).
		Priority(sdk.GeneratorCrossCutting).
		Provides(Capability).
		Requires(suite.Capability).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:model` schema.
//
// Negation is denied because the tier exists exactly where one is declared,
// so deleting the line is the suppression (docs/adr/0016) — and deleting it
// removes the emission, the file and the `engine` module requirement together.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates property-based state-machine tests for the annotated " +
					"interface: random sequences of its methods run against the " +
					"subject and a known-good in-memory reference side by side, " +
					"compared after every call. " + RefKey + " names a constructor in " +
					"the source package returning the interface, for a shape no " +
					"shipped reference models.",
			).
			AllowedKeys(RefKey).
			On(sdk.NodeKindInterface).
			DenyNegation().
			Build(),
	}
}

// Bindings is the value queued once per interface carrying the directive.
type Bindings struct {
	sdk.BaseEmit
	suite.Subject

	// OptionName is `<Iface>Model` — the option a consumer passes to the
	// contract entry to run this tier. PropertyName is `<Iface>ModelProperty`,
	// the composition point it and any bespoke harness share. OptionTypeName
	// and ConfigName carry the tier's own option surface.
	OptionName, PropertyName, OptionTypeName, ConfigName string

	// EntryName is the harness entry the option is passed to, for the header.
	EntryName string

	// FixtureCtor is the harness's derived-input constructor. The pools draw
	// from its fields rather than from generators of their own, because the
	// harness's checks, the seeds and these sequences must keep hitting the
	// same keys — a pool of fresh random keys never revisits a written one,
	// and every comparison passes over a history with no collisions in it.
	FixtureCtor string

	// Keys and Values are the two shared pools. Values.Field is empty for an
	// interface with no writer-shaped method, and the template then declares
	// no pool — an unused local is a compile error in a generated file.
	Keys, Values Pool

	// Reference is the known-good implementation every action compares
	// against.
	Reference Reference

	// Actions is one per driven method, in declaration order.
	Actions []*Action

	// Adapter is every interface method, delegated to the oracle or inert,
	// in declaration order. Empty when the reference was supplied.
	Adapter []AdapterMethod

	// Skipped names the methods no action drives, each with why. Rendered
	// into the header: a method absent from the sequences without a stated
	// reason is indistinguishable from a generator that forgot it.
	Skipped []Skip

	// Laws is every law the classifications earned and this build could fill,
	// registered on the run through the same registry a consumer's own laws
	// join. Unbound is the rest — selected, and waiting on something the
	// header names, because a law that quietly failed to bind reads as a
	// claim the run checks.
	Laws    []*LawBinding
	Unbound []Skip

	// PkgName is where Layout routed the file — see [Bindings.SetOutputPackages].
	PkgName string
}

// Kind returns [KindBindings].
func (*Bindings) Kind() sdk.Kind { return KindBindings }

// SetOutputPackages records where Layout routed the file, which is not known
// during Generate: `out=`/`pkg=` directives move it after this plugin ran.
//
// The one consumer is the miss sentinel's message, whose text must open with
// the package that declares it — the convention every hand-written error in
// this repository is held to, and a generated file is not excused from.
func (b *Bindings) SetOutputPackages(byTag map[string]string) {
	if path := byTag[""]; path != "" {
		b.PkgName = path[strings.LastIndexByte(path, '/')+1:]
	}
}

// MissPrefix is what the sentinel's message opens with: the routed package
// where Layout resolved one, the interface's own spelling where a run never
// reached Layout — still a plausible message rather than an empty prefix.
func (b *Bindings) MissPrefix() string {
	if b.PkgName != "" {
		return b.PkgName
	}
	return strings.ToLower(b.IfaceName)
}

// KeyOfName is the shared key projection's identifier — one derivation used
// by the reference constructor and every law field that keys a value, so the
// two cannot disagree about which field is the identity.
func (b *Bindings) KeyOfName() string {
	return strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "ModelKeyOf"
}

// ModelPkg surfaces the runner's import path to the templates, which can
// reach a method and not a const.
func (*Bindings) ModelPkg() string { return ModelPkg }

// RefPkg returns the reference package's import path.
func (*Bindings) RefPkg() string { return RefPkg }

// TierName returns the path the tier reports under.
func (*Bindings) TierName() string { return TierName }

// UsesValues reports whether any action draws from the values pool.
func (b *Bindings) UsesValues() bool {
	for _, a := range b.Actions {
		if a.Pool == poolValues {
			return true
		}
	}
	return false
}

// UsesKeys reports whether any action draws from the keys pool. A composite
// writer draws from both, whatever its Pool says.
func (b *Bindings) UsesKeys() bool {
	for _, a := range b.Actions {
		if a.Pool == poolKeys || a.Shape == shapeCompositeWriter {
			return true
		}
	}
	return false
}

// Pool is one shared value source: a fixture field and its companion.
type Pool struct {
	// Field and OtherField name the fixture fields the pool samples.
	Field, OtherField string

	// Type is the drawn type, for the slice literal's element clause.
	Type sdk.Ref
}

// Reference is how the run builds its oracle.
type Reference struct {
	// SuppliedCtor is the constructor the directive named, nil where the
	// oracle is derived. When set, nothing else here is: the arguments belong
	// to the consumer's own constructor.
	SuppliedCtor *sdk.Expr

	// TypeName, CtorName and MissName are the derived adapter's identifiers:
	// the struct over the oracle, its constructor, and the sentinel the oracle
	// reports for a key nothing wrote.
	TypeName, CtorName, MissName string

	// Oracle names which shipped store the adapter wraps.
	Oracle Oracle

	// KeyField is the field of the value type the map oracle keys on, empty
	// for the keyed oracle, whose key is an argument.
	KeyField string
}

// Oracle names a shipped reference implementation the adapter can wrap.
type Oracle string

// The two store models Go interfaces declare: a value that carries its own
// key, and a key passed beside the value.
const (
	OracleMap   Oracle = "map"
	OracleKeyed Oracle = "keyed"
)

// Supplied reports that the directive named the reference.
func (r Reference) Supplied() bool { return r.SuppliedCtor != nil }

// StoreType is the wrapped oracle's type name, and "New" + StoreType its
// constructor — the naming convention `ref` keeps, relied on here so the
// template asks one question instead of branching twice.
func (r Reference) StoreType() string {
	if r.Oracle == OracleKeyed {
		return "KeyedStore"
	}
	return "MapStore"
}

// Keyed reports the keyed-put oracle, whose constructor takes no projection.
func (r Reference) Keyed() bool { return r.Oracle == OracleKeyed }

// Action is one method, driven and compared.
type Action struct {
	sdk.BaseEmit

	// KindName is `model.action.<shape>` — the template that renders the
	// constructor call, selected the way the harness selects its check
	// templates.
	KindName sdk.Kind

	// Method is the source identifier, and the name a failing step reports.
	Method string

	// Shape is the detector that chose the constructor, for the header.
	Shape string

	// Ctor is the `engine/model/action` constructor, as a qualified
	// expression.
	Ctor *sdk.Expr

	// Iface is the subject type, for the closure's parameter.
	Iface sdk.Ref

	// Key and Value are the closure's drawn-input and result types, nil where
	// the shape carries none.
	Key, Value sdk.Ref

	// Pool names the shared pool the constructor draws from — "keys",
	// "values", or empty for a shape that draws nothing.
	Pool string

	// TakesCtx reports whether the method's first parameter is a context. The
	// constructor's closure always receives one; a method that does not take
	// it ignores it.
	TakesCtx bool
}

// Kind returns the shape-specific template key.
func (a *Action) Kind() sdk.Kind { return a.KindName }

// AdapterMethod is one interface method on the derived reference.
type AdapterMethod struct {
	// Sig is the source signature the body is composed from.
	Sig *golang.Sig

	// Op is the oracle method the body forwards to, empty for an inert body.
	Op string

	// Reason says why an inert body is inert, for the comment above it.
	Reason string
}

// Skip is a method with no action, and the reason.
type Skip struct{ Method, Reason string }

// Generate queues one set of bindings per interface carrying the directive.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
	harnesses := sdk.PendingByOrigin[*suite.Contract](ctx.Store.Emit())

	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		harness, hosted := harnesses[sdk.Node(iface)]
		if !hosted {
			// Every generated identifier comes from the projection, so there
			// is nothing to bind onto. Asked and impossible.
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but no harness exists for it; "+
					"add //testkit:%s",
				Name, iface.Name, DirectiveName, suite.DirectiveName)
			continue
		}
		if len(iface.TypeParams) > 0 {
			// The property, the reference and the pools all land at concrete
			// types, and nothing in the source names any.
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q is generic, and the model tier has no types to "+
					"instantiate at; remove //testkit:%s",
				Name, iface.Name, DirectiveName)
			continue
		}

		b, ok := bindingsOf(ctx, c, iface, harness)
		if !ok {
			continue
		}
		// The companion rides with the bindings rather than being optional:
		// an emission nothing proves is the hand-written probe this output
		// replaced, owed again by every armed package. A supplied reference
		// is the one exception — it is not the emission, and holding a
		// consumer's constructor to the tier is their suite's business.
		queued := []sdk.EmitNode{b}
		if !b.Reference.Supplied() {
			queued = append(queued, companionOf(c, iface, b, harness))
		}
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface, queued...); err != nil {
			return fmt.Errorf("%s: queue interface %q: %w", Name, iface.Name, err)
		}
	}
	return nil
}

// KindCompanion is the companion's emit kind and template.
const KindCompanion sdk.Kind = "model.companion"

// Companion is the generated proof of the emission, routed to the test
// output: the derived reference passes its own tier, and every inert body
// answers zero values rather than panicking.
//
// It certifies what the generator produced, not any consumer's subject —
// which is why it is generated beside the bindings instead of asked of the
// package that arms them.
type Companion struct {
	sdk.BaseEmit
	suite.Subject

	// PropertyName, RefCtorName and ReferenceOptionName reach the bindings
	// across the package boundary; FixtureCtor supplies the inert calls'
	// arguments. HarnessPkg qualifies all four — see [Companion.SetOutputPackages].
	PropertyName, RefCtorName, ReferenceOptionName, FixtureCtor string
	HarnessPkg                                                  string

	// Inert is every adapter method the companion calls once with derived
	// arguments — proving the body answers, whatever it is handed.
	Inert []InertProbe
}

// Kind returns [KindCompanion].
func (*Companion) Kind() sdk.Kind { return KindCompanion }

// ModelPkg surfaces the runner's import path to the template.
func (*Companion) ModelPkg() string { return ModelPkg }

// SetOutputPackages records where Layout routed the bindings. The companion
// lands in the external test package and reaches every generated identifier
// through that qualifier; the provisional value is the source package, whose
// failure mode is a compile error naming the symbol rather than a bare name
// silently binding to whatever is in scope.
//
// Layout may pass a partial map — a run that recorded routing errors reaches
// dispatch with tags missing — so absence keeps the provisional value.
func (c *Companion) SetOutputPackages(byTag map[string]string) {
	if path := byTag[""]; path != "" {
		c.HarnessPkg = path
	}
}

// InertProbe is one inert method call the companion makes.
type InertProbe struct {
	// Method is the call; TakesCtx forwards the test's context.
	Method   string
	TakesCtx bool

	// ArgFields name the fixture fields the call is handed.
	ArgFields []string

	// Assign discards the call's results — "_, _ = " for two, empty for a
	// method returning nothing. Spelled here rather than composed in the
	// template, which has no loop over a count.
	Assign string
}

// companionOf derives the companion from the bindings it proves.
func companionOf(c *sdk.Provenance, iface *sdk.Interface, b *Bindings, harness *suite.Contract) *Companion {
	comp := &Companion{
		BaseEmit:            sdk.EmitBaseTagged(sdk.EmitBase(c, iface), GoTestOutputTag),
		Subject:             b.Subject,
		PropertyName:        b.PropertyName,
		RefCtorName:         b.Reference.CtorName,
		ReferenceOptionName: b.IfaceName + "ModelReference",
		FixtureCtor:         b.FixtureCtor,
		HarnessPkg:          iface.Package,
	}
	for _, am := range b.Adapter {
		if am.Op != "" {
			continue
		}
		m := methodOf(harness, am.Sig.Name)
		comp.Inert = append(comp.Inert, InertProbe{
			Method:    am.Sig.Name,
			TakesCtx:  m.TakesContext(),
			ArgFields: m.ArgFields,
			Assign:    discardOf(len(am.Sig.Returns)),
		})
	}
	return comp
}

// discardOf spells the assignment that drops n results.
func discardOf(n int) string {
	if n == 0 {
		return ""
	}
	return strings.Repeat("_, ", n-1) + "_ = "
}

// methodOf finds one projection method by name; the adapter was built from
// the same list, so a miss is unreachable.
func methodOf(harness *suite.Contract, name string) *suite.Method {
	for i := range harness.Methods {
		if harness.Methods[i].Name == name {
			return &harness.Methods[i]
		}
	}
	return nil
}

// bindingsOf derives one interface's bindings, false where it reported why it
// could not.
func bindingsOf(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	iface *sdk.Interface,
	harness *suite.Contract,
) (*Bindings, bool) {
	b := &Bindings{
		BaseEmit:       sdk.EmitBase(c, iface),
		Subject:        harness.Subject,
		OptionName:     harness.IfaceName + "Model",
		PropertyName:   harness.IfaceName + "ModelProperty",
		OptionTypeName: harness.IfaceName + "ModelOption",
		ConfigName:     strings.ToLower(harness.IfaceName[:1]) + harness.IfaceName[1:] + "ModelConfig",
		EntryName:      harness.EntryName,
		FixtureCtor:    harness.Fixture.CtorName,
	}

	partners := partnerMethods(iface)
	var keyed, valued, composite *suite.Method
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if role, partner := partners[m.Name]; partner {
			b.Skipped = append(b.Skipped, Skip{Method: m.Name, Reason: role})
			continue
		}
		a, skip := actionOf(b, m)
		if skip != "" {
			b.Skipped = append(b.Skipped, Skip{Method: m.Name, Reason: skip})
			continue
		}
		b.Actions = append(b.Actions, a)
		switch a.Shape {
		case "reader":
			keyed = m
		case "writer":
			valued = m
		case shapeCompositeWriter:
			composite = m
		}
	}

	if len(b.Actions) == 0 {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: no method of %q maps to an action, so the sequences would drive "+
				"nothing; the header lists what each method is waiting on",
			Name, iface.Name)
		return nil, false
	}

	if !referenceOf(ctx, iface, harness, b, keyed, valued, composite, partners) {
		return nil, false
	}
	poolsOf(b, keyed, valued, composite)
	lawsOf(b, harness, partners, keyed)
	return b, true
}

// actionOf builds one method's action, or says why there is none.
func actionOf(b *Bindings, m *suite.Method) (*Action, string) {
	name := shape.Get(m.Source.Meta())
	if name == "" {
		return nil, "the annotator classified no shape for it"
	}
	ctor, mapped := tiers.ActionFor(name)
	if !mapped {
		return nil, "no action drives the " + name + " shape"
	}

	a := &Action{
		BaseEmit: b.BaseEmit,
		KindName: sdk.Kind(ActionKindPrefix + name),
		Method:   m.Name,
		Shape:    name,
		Ctor:     sdk.NewExternal(actionPkg, ctor),
		Iface:    b.IfaceRef,
		TakesCtx: m.TakesContext(),
	}
	switch name {
	case "reader":
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		a.Value = m.Returns[0].Type
	case "writer":
		a.Pool = poolValues
		a.Value = m.CallArgs()[0].Type
		// A writer whose mixin assigns it an oracle operation is one whose
		// argument is a key — a delete, not a put — and drawing values would
		// feed it strings no writer ever stored.
		for _, mixin := range m.Mixins {
			if _, assigned := tiers.KeyedStoreMixinOp(mixin); assigned {
				a.Pool = poolKeys
			}
		}
	case shapeCompositeWriter:
		// Draws from both pools: the key beside the value is the shape.
		a.Pool = poolValues
		a.Key = m.CallArgs()[0].Type
		a.Value = m.CallArgs()[1].Type
	case "aggregator", "pure", "predicate":
		if m.HasResults() {
			a.Value = m.Returns[0].Type
		}
	case "lifecycle":
		// The call is the whole action.
	}
	return a, ""
}

// referenceOf fills the reference, or reports what would let one be derived.
//
// Two oracles cover the two store models Go interfaces actually declare. A
// composite writer — Put(ctx, k, v) — selects the keyed store, and needs no
// key projection: the key is an argument. A plain writer — Store(ctx, v) —
// selects the map, keyed on the one field of the value that holds the key.
// The composite wins where both appear, because a keyed oracle can host a
// value write only inertly while the reverse loses the delete.
func referenceOf(
	ctx *sdk.GeneratorContext,
	iface *sdk.Interface,
	harness *suite.Contract,
	b *Bindings,
	keyed, valued, composite *suite.Method,
	partners map[string]string,
) bool {
	if named, given := directiveValue(iface, RefKey); given {
		if strings.Contains(named, ".") {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: %s=%q on %q carries a qualifier; name a constructor in the "+
					"interface's own package",
				Name, RefKey, named, iface.Name)
			return false
		}
		b.Reference = Reference{SuppliedCtor: sdk.NewExternal(iface.Package, named)}
		return true
	}

	lower := strings.ToLower(harness.IfaceName[:1]) + harness.IfaceName[1:]
	names := Reference{
		TypeName: lower + "ModelReference",
		// Exported: the generated companion lands in the external test
		// package and proves the oracle from there, the way a consumer
		// would — and a consumer comparing against it gets the same door.
		CtorName: "New" + harness.IfaceName + "ModelReference",
		MissName: lower + "ModelMiss",
	}

	if keyed != nil && composite != nil {
		keyQ, _ := shape.MetaKeyType.Get(keyed.Source.Meta())
		readV, _ := shape.MetaValueType.Get(keyed.Source.Meta())
		putK, _ := shape.MetaKeyType.Get(composite.Source.Meta())
		putV, _ := shape.MetaValueType.Get(composite.Source.Meta())
		if keyQ != putK || readV == "" || readV != putV {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: %q reads (%s → %s) where its keyed writer takes (%s, %s), "+
					"which one store cannot model; name a constructor with "+
					"//testkit:%s %s=<constructor>",
				Name, iface.Name, keyQ, readV, putK, putV, DirectiveName, RefKey)
			return false
		}
		names.Oracle = OracleKeyed
		b.Reference = names
		b.Adapter = adapterOf(harness, partners, OracleKeyed)
		return true
	}

	if keyed == nil || valued == nil {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %q has no reader/writer pair for the derived reference to model; "+
				"name a constructor with //testkit:%s %s=<constructor>",
			Name, iface.Name, DirectiveName, RefKey)
		return false
	}

	keyQ, _ := shape.MetaKeyType.Get(keyed.Source.Meta())
	readV, _ := shape.MetaValueType.Get(keyed.Source.Meta())
	wroteV, _ := shape.MetaValueType.Get(valued.Source.Meta())
	if readV == "" || readV != wroteV {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %q reads %s where its writer takes %s, which one map cannot model; "+
				"name a constructor with //testkit:%s %s=<constructor>",
			Name, iface.Name, readV, wroteV, DirectiveName, RefKey)
		return false
	}

	field, why := keyFieldOf(ctx, readV, keyQ)
	if field == "" {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %q needs the field of %s holding the key and %s; name a "+
				"constructor with //testkit:%s %s=<constructor>",
			Name, iface.Name, readV, why, DirectiveName, RefKey)
		return false
	}

	names.Oracle = OracleMap
	names.KeyField = field
	b.Reference = names
	b.Adapter = adapterOf(harness, partners, OracleMap)
	return true
}

// keyFieldOf finds the one field of the value struct that can hold the key,
// returning the failure's spelling when it cannot.
//
// One candidate answers directly. Several prefer the conventional spellings —
// ID, then Key — because a value type keyed on one of two same-typed fields is
// keyed on the one its author named for the job; a value keyed otherwise is
// what [RefKey] exists for.
func keyFieldOf(ctx *sdk.GeneratorContext, valueQ, keyQ string) (string, string) {
	var s *sdk.Struct
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name == valueQ {
			s = cand
			break
		}
	}
	if s == nil {
		return "", "no struct declaration was found for it"
	}

	var candidates []string
	for _, f := range s.Fields {
		if shape.QName(f.Type) == keyQ {
			candidates = append(candidates, f.Name)
		}
	}
	switch len(candidates) {
	case 0:
		return "", "no field of it has the key's type"
	case 1:
		return candidates[0], ""
	}
	for _, preferred := range []string{"ID", "Key"} {
		for _, cand := range candidates {
			if cand == preferred {
				return cand, ""
			}
		}
	}
	return "", "several fields share the key's type and none is named ID or Key"
}

// adapterOf builds the delegation table: every method forwards to the oracle's
// matching operation or holds an inert body.
func adapterOf(harness *suite.Contract, partners map[string]string, oracle Oracle) []AdapterMethod {
	out := make([]AdapterMethod, 0, len(harness.Methods))
	for i := range harness.Methods {
		m := &harness.Methods[i]
		am := AdapterMethod{Sig: m.Sig}
		switch role, partner := partners[m.Name]; {
		case partner:
			am.Reason = role
		case !m.TakesContext():
			am.Reason = "it takes no context to forward to the oracle"
		default:
			if op, modelled := oracleOp(oracle, m); modelled {
				am.Op = op
			} else {
				am.Reason = "the oracle does not model its shape"
			}
		}
		out = append(out, am)
	}
	return out
}

// oracleOp resolves one method's delegation for the chosen oracle: a mixin
// assignment first — the stamp says what a method is for, outranking what it
// looks like — then the oracle's shape table.
func oracleOp(oracle Oracle, m *suite.Method) (string, bool) {
	if oracle == OracleKeyed {
		for _, name := range m.Mixins {
			if op, assigned := tiers.KeyedStoreMixinOp(name); assigned {
				return op, true
			}
		}
		return tiers.KeyedStoreOp(shape.Get(m.Source.Meta()))
	}
	return tiers.MapStoreOp(shape.Get(m.Source.Meta()))
}

// poolsOf fills the shared pools from the fixture fields the harness already
// derived — the same values its checks and seed use, so the sequences keep
// hitting the keys the rest of the file talks about.
//
// The composite writer supplies the value pool where one exists: a plain
// writer beside it is usually a delete or a touch, whose one argument is a
// key, and a values pool drawn from that would feed keys to every value slot.
func poolsOf(b *Bindings, keyed, valued, composite *suite.Method) {
	if keyed != nil {
		arg := keyed.CallArgs()[0]
		b.Keys = Pool{
			Field:      keyed.ArgFields[0],
			OtherField: keyed.ArgFields[0] + suite.OtherSuffix,
			Type:       arg.Type,
		}
	}
	switch {
	case composite != nil:
		arg := composite.CallArgs()[1]
		b.Values = Pool{
			Field:      composite.ArgFields[1],
			OtherField: composite.ArgFields[1] + suite.OtherSuffix,
			Type:       arg.Type,
		}
	case valued != nil:
		arg := valued.CallArgs()[0]
		b.Values = Pool{
			Field:      valued.ArgFields[0],
			OtherField: valued.ArgFields[0] + suite.OtherSuffix,
			Type:       arg.Type,
		}
	}
}

// partnerMethods maps each method excluded by a mixin's sibling reference to
// the `<mixin>.<param>: <reason>` that claims it.
//
// Only the references whose role overrides their shape exclude: a validator
// is writer-shaped, and a sequence that drives it as a writer corrupts the
// reference with stores the subject never made. Most references name an
// ordinary method — a put that is a writer — and those stay in the sequences;
// [tiers.PartnerDriven] is the classification, held total by the census. The
// registry is consulted for which parameters are sibling references rather
// than values — that is the annotator's vocabulary, not a spelling this
// plugin owns.
func partnerMethods(iface *sdk.Interface) map[string]string {
	out := map[string]string{}
	for _, m := range iface.Methods {
		for _, name := range shape.Mixins(m.Meta()) {
			for _, p := range siblingParams(name) {
				v, ok := shape.MixinParamKey(name, p).Get(m.Meta())
				if !ok || v == "" {
					continue
				}
				if driven, reason := tiers.PartnerDriven(name, p); !driven {
					out[golang.LocalName(v)] = "the " + name + "." + p + " partner — " + reason
				}
			}
		}
	}
	return out
}

// siblingParams returns the named mixin's sibling-reference parameters.
func siblingParams(name string) []string {
	for _, m := range mixins.All() {
		if m.Name == name {
			return m.SiblingParams
		}
	}
	return nil
}

// directiveValue reads one key off the interface's own directive.
func directiveValue(iface *sdk.Interface, key string) (string, bool) {
	for _, dir := range iface.Directives() {
		if string(dir.Name) != string(DirectiveName) {
			continue
		}
		if v, ok := dir.KV[key]; ok && v != "" {
			return v, true
		}
	}
	return "", false
}
