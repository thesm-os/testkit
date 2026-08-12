// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

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
const Version = "0.5.1"

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

// The detector spellings this plugin branches on beyond its template
// dispatch: the keyed put draws from both pools and selects the keyed
// oracle, and the reader and writer are the canonical store pair.
const (
	shapeCompositeWriter = "compositewriter"
	shapeReader          = "reader"
	shapeWriter          = "writer"
	shapeAggregator      = "aggregator"
)

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

	// ConcReader and ConcWriter are the actions the concurrent leg drives
	// against the Porcupine keyed-store model, both nil where the leg does
	// not derive. They point into Actions, so the closures the two legs
	// spell agree about every method and type.
	ConcReader, ConcWriter *Action

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

// Concurrent reports whether the linearizability leg derives: a map-shaped
// pair the Porcupine keyed-store model speaks, unrefined by a claim that
// changes what a read means.
func (b *Bindings) Concurrent() bool { return b.ConcReader != nil && b.ConcWriter != nil }

// LinearizePkg surfaces the Porcupine wiring's import path to the templates.
func (*Bindings) LinearizePkg() string { return LinearizePkg }

// ModelPkg surfaces the runner's import path to the templates, which can
// reach a method and not a const.
func (*Bindings) ModelPkg() string { return ModelPkg }

// RefPkg returns the reference package's import path.
func (*Bindings) RefPkg() string { return RefPkg }

// TierName returns the path the tier reports under.
func (*Bindings) TierName() string { return TierName }

// NeedsFixture reports whether anything in the property reads the fixture —
// a pool, or a multi-argument writer's per-position pairs. An unused local
// is a compile error in a generated file.
func (b *Bindings) NeedsFixture() bool {
	if b.UsesKeys() || b.UsesValues() {
		return true
	}
	for _, a := range b.Actions {
		if len(a.Args) > 0 {
			return true
		}
	}
	return false
}

// UsesValues reports whether any action draws from the values pool.
func (b *Bindings) UsesValues() bool {
	for _, a := range b.Actions {
		if a.Pool == poolValues {
			return true
		}
	}
	return false
}

// UsesKeys reports whether anything draws from the keys pool. A composite
// writer draws from both, whatever its Pool says, and a pinned values pool
// draws a key for every value.
func (b *Bindings) UsesKeys() bool {
	if b.Values.Pin != "" {
		return true
	}
	for _, a := range b.Actions {
		if a.Pool == poolKeys || a.Shape == shapeCompositeWriter {
			return true
		}
	}
	return false
}

// Pool is one shared value source: a fixture field and its companion, and how
// far past them the draws reach.
type Pool struct {
	// Field and OtherField name the fixture fields the pool samples — dotted
	// where the key rides inside a fixture value rather than beside it.
	Field, OtherField string

	// Type is the drawn type, for the slice literal's element clause. Q is
	// the same type in the annotator's stamp spelling, which is what a law
	// role's stamps are compared against.
	Type sdk.Ref
	Q    string

	// Wide reports that the pool blends the fixture pair with arbitrary
	// [model.Make] draws; WhyNarrow is the header's reason where it cannot.
	Wide      bool
	WhyNarrow string

	// Pin is the value field overwritten with a keys-pool draw, so every
	// drawn value lands on a key the reads revisit — empty where the key is
	// an argument beside the value, or where a restricting claim holds the
	// pool to values the harness has proven accepted.
	Pin string
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

	// Oracle names which shipped store the adapter wraps; Dedupe and Pins are
	// its deduplicating and resolution-pinning refinements, each where a
	// mixin claims it. TwinWhy is the header's reason where the oracle is the
	// twin floor.
	Oracle  Oracle
	Dedupe  bool
	Pins    bool
	TwinWhy string

	// KeyField is the field of the value type the map oracle keys on, empty
	// for the keyed oracle, whose key is an argument.
	KeyField string
}

// Oracle names a shipped reference implementation the adapter can wrap.
type Oracle string

// The three store models Go interfaces declare — a value that carries its own
// key, a key passed beside the value, and an append-and-drain collection with
// no keys at all — plus the twin floor beneath them: a second instance from
// the subject's own factory. Two twins driven identically must agree, which
// catches nondeterminism, hidden shared state and races without an
// independent model, and misses what an independent model exists to catch —
// a subject that is systematically wrong agrees with itself. The header says
// which floor was reached and why, and `ref=` raises it.
const (
	OracleMap        Oracle = "map"
	OracleKeyed      Oracle = "keyed"
	OracleCollection Oracle = "collection"
	OracleTwin       Oracle = "twin"
)

// Supplied reports that the directive named the reference.
func (r Reference) Supplied() bool { return r.SuppliedCtor != nil }

// Twin reports the twin floor: the subject's own factory stands in.
func (r Reference) Twin() bool { return r.Oracle == OracleTwin }

// Derived reports that an adapter over a shipped oracle is generated — the
// case that owes a miss sentinel, an adapter and a companion proof.
func (r Reference) Derived() bool { return !r.Supplied() && !r.Twin() }

// StoreType is the wrapped oracle's type name, and "New" + StoreType its
// constructor — the naming convention `ref` keeps, relied on here so the
// template asks one question instead of branching twice.
func (r Reference) StoreType() string {
	switch r.Oracle {
	case OracleKeyed:
		return "KeyedStore"
	case OracleCollection:
		if r.Dedupe {
			return "SetCollection"
		}
		return "Collection"
	case OracleMap, OracleTwin:
		// The twin has no store; nothing renders the answer.
	}
	if r.Pins {
		return "StickyStore"
	}
	return "MapStore"
}

// Keyed reports the keyed-put oracle, whose constructor takes no projection.
func (r Reference) Keyed() bool { return r.Oracle == OracleKeyed }

// Collects reports the append-and-drain oracle: one type argument, no keys,
// no miss sentinel — a drain of nothing is an empty slice, not an error.
func (r Reference) Collects() bool { return r.Oracle == OracleCollection }

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
	// the shape carries none; Value2 is the second result of the shapes that
	// answer two.
	Key, Value, Value2 sdk.Ref

	// Args is the per-position pool of a multi-argument writer: each drawn
	// from its own fixture pair, because one shared pool cannot serve several
	// types.
	Args []ActionArg

	// Pool names the shared pool the constructor draws from — "keys",
	// "values", or empty for a shape that draws nothing.
	Pool string

	// NoError marks a method answering without an error where the
	// constructor compares one: the closure supplies nil itself.
	NoError bool

	// TakesCtx reports whether the method's first parameter is a context. The
	// constructor's closure always receives one; a method that does not take
	// it ignores it.
	TakesCtx bool
}

// ActionArg is one drawn argument of a multi-argument writer.
type ActionArg struct {
	// Field is the fixture field the position samples; Type its slice
	// literal's element clause.
	Field string
	Type  sdk.Ref
}

// ModelPkg surfaces the runner's import path to the action templates whose
// closures draw inline.
func (*Action) ModelPkg() string { return ModelPkg }

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
		// replaced, owed again by every armed package. The exceptions are the
		// references this plugin did not derive — a supplied constructor is
		// the consumer's to prove, and the twin floor is the subject's own
		// factory, which the contract run itself drives.
		queued := []sdk.EmitNode{b}
		if b.Reference.Derived() {
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

	// ConcurrentName is the concurrent leg's runner, empty where none
	// derives. The companion holds the leg to the derived reference: a
	// mutex-guarded store is linearizable, so a red run is the wiring's own.
	ConcurrentName string

	// Inert is every adapter method the companion calls once with derived
	// arguments — proving the body answers, whatever it is handed.
	Inert []InertProbe
}

// Kind returns [KindCompanion].
func (*Companion) Kind() sdk.Kind { return KindCompanion }

// NeedsFixture reports whether any inert probe takes an argument — an unused
// fixture local is a compile error in a generated file.
func (c *Companion) NeedsFixture() bool {
	for _, p := range c.Inert {
		if len(p.ArgFields) > 0 {
			return true
		}
	}
	return false
}

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
	if b.Concurrent() {
		comp.ConcurrentName = b.IfaceName + "ModelConcurrent"
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

	// Classification first, actions second: the values pool is one local, and
	// which writer feeds it decides which writers may draw from it — a second
	// writer taking a different type would draw values no signature accepts.
	partners := partnerMethods(iface)
	var keyed, composite, collector, keyFallback, valueFallback *suite.Method
	var writers []*suite.Method
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if _, partner := partners[m.Name]; partner {
			continue
		}
		switch pseudoShape(m) {
		case shapeReader:
			keyed = m
		case shapeWriter:
			writers = append(writers, m)
		case shapeCompositeWriter:
			composite = m
		case tiers.ShapeCollector:
			collector = m
		case "readernoerror", "readerwithbool", "pointerreader", "multireader",
			"lookup", "batchreader":
			// Key-drawing shapes no oracle reads through: they cannot select
			// a store, but where nothing else supplies the keys pool, their
			// first argument's fixture pair is it.
			if keyFallback == nil {
				keyFallback = m
			}
		case "mutator":
			if valueFallback == nil {
				valueFallback = m
			}
		}
	}
	valued := feederOf(keyed, collector, writers)

	valueQ := ""
	if valued != nil {
		valueQ, _ = shape.MetaValueType.Get(valued.Source.Meta())
	}
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
		if a.Shape == shapeWriter && a.Pool == poolValues && m != valued {
			if q, _ := shape.MetaValueType.Get(m.Source.Meta()); q != valueQ {
				b.Skipped = append(b.Skipped, Skip{
					Method: m.Name,
					Reason: "takes " + q + " where the values pool draws " + valueQ,
				})
				continue
			}
		}
		b.Actions = append(b.Actions, a)
	}

	if len(b.Actions) == 0 {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: no method of %q maps to an action, so the sequences would drive "+
				"nothing; the header lists what each method is waiting on",
			Name, iface.Name)
		return nil, false
	}

	if !referenceOf(ctx, iface, harness, b, keyed, valued, composite, collector, partners) {
		return nil, false
	}
	// The sequences drive only what the oracle models: an action on a method
	// the derived adapter holds inert compares the subject against a body
	// answering zeros, and fails every correct implementation at the first
	// observable answer.
	if b.Reference.Derived() {
		inert := map[string]string{}
		for _, am := range b.Adapter {
			if am.Op == "" {
				inert[am.Sig.Name] = am.Reason
			}
		}
		kept := b.Actions[:0]
		for _, a := range b.Actions {
			if reason, is := inert[a.Method]; is {
				b.Skipped = append(b.Skipped, Skip{
					Method: a.Method,
					Reason: "the derived reference holds it inert — " + reason,
				})
				continue
			}
			kept = append(kept, a)
		}
		b.Actions = kept
	}
	// The oracle derivation sees only the canonical reader and writer; the
	// pools serve every drawing action, so their sources widen to the
	// fallbacks where the canonical shapes are absent.
	keySrc, valueSrc := keyed, valued
	if keySrc == nil {
		keySrc = keyFallback
	}
	if valueSrc == nil && composite == nil {
		valueSrc = valueFallback
	}
	poolsOf(ctx, b, harness, keySrc, valueSrc, composite)
	lawsOf(b, harness, partners, keyed)
	concurrentOf(b, keyed, valued)
	return b, true
}

// concurrentOf wires the linearizability leg where the map derivation holds
// unrefined: the Porcupine keyed-store model speaks reader and writer over
// one key at a time, and a claim that changes what a read means — the sticky
// pin — is a different model, not a different wiring. The leg reuses the
// sequential actions, so both legs draw from the same pools and spell the
// same closures; concurrency that never collides checks nothing, which is
// the mistake the shared pools exist to rule out.
func concurrentOf(b *Bindings, keyed, valued *suite.Method) {
	if b.Reference.Oracle != OracleMap || !b.Reference.Derived() || b.Reference.Pins || keyed == nil || valued == nil {
		return
	}
	for _, a := range b.Actions {
		switch a.Method {
		case keyed.Name:
			b.ConcReader = a
		case valued.Name:
			b.ConcWriter = a
		}
	}
	if b.ConcReader == nil || b.ConcWriter == nil {
		// Half a pair drives nothing Porcupine can order; the leg derives
		// whole or not at all.
		b.ConcReader, b.ConcWriter = nil, nil
	}
}

// actionOf builds one method's action, or says why there is none.
func actionOf(b *Bindings, m *suite.Method) (*Action, string) {
	name := pseudoShape(m)
	if name == "" {
		return nil, "the annotator classified no shape for it"
	}
	if name == tiers.ShapeCollector {
		// The aggregator constructors compare a comparable result, and a
		// slice is not one; the stream action drains it instead.
		elem, err := collectorElem(b, m)
		if err != "" {
			return nil, err
		}
		return &Action{
			BaseEmit: b.BaseEmit,
			KindName: sdk.Kind(ActionKindPrefix + tiers.ShapeCollector),
			Method:   m.Name,
			Shape:    tiers.ShapeCollector,
			Ctor:     sdk.NewExternal(actionPkg, "Stream"),
			Iface:    b.IfaceRef,
			TakesCtx: m.TakesContext(),
			Value:    elem,
		}, ""
	}
	ctor, mapped := tiers.ActionFor(name)
	if !mapped {
		return nil, "no action drives the " + name + " shape"
	}
	for _, r := range m.Returns {
		// A live handle — a channel, a function — compares by identity, and
		// two sides' handles never share one; the comparison would fail
		// every correct subject on its first answer.
		if r.Source != nil && (golang.IsChannel(r.Source) || r.Source.IsFunc()) {
			return nil, "answers a live handle only identity could compare"
		}
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
	case shapeReader, "readernoerror", "pointerreader", "readerwithbool":
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		a.Value = m.Returns[0].Type
	case "multireader", "lookup":
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		a.Value = m.Returns[0].Type
		a.Value2 = m.Returns[1].Type
	case "batchreader":
		a.Pool = poolKeys
		a.Key = m.CallArgs()[0].Type
		elem, err := collectorElem(b, m)
		if err != "" {
			return nil, err
		}
		a.Value = elem
	case shapeWriter:
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
	case "mutator":
		a.Pool = poolValues
		a.Value = m.CallArgs()[0].Type
	case shapeCompositeWriter:
		// Draws from both pools: the key beside the value is the shape.
		a.Pool = poolValues
		a.Key = m.CallArgs()[0].Type
		a.Value = m.CallArgs()[1].Type
	case "multiargwriter":
		// One shared pool cannot serve several types; each position draws
		// from its own fixture pair instead.
		for i, arg := range m.CallArgs() {
			a.Args = append(a.Args, ActionArg{Field: m.ArgFields[i], Type: arg.Type})
		}
	case shapeAggregator, "pure", "predicate":
		if m.HasResults() {
			a.Value = m.Returns[0].Type
		}
		a.NoError = name == shapeAggregator && len(m.Returns) == 1
		if name != shapeAggregator {
			// A pure call's closure has no draw handle, so its inputs are
			// the fixture's own — one honest point per argument; the wide
			// exploration of a pure function is the fuzz tier's business.
			for i, arg := range m.CallArgs() {
				a.Args = append(a.Args, ActionArg{Field: m.ArgFields[i], Type: arg.Type})
			}
		}
	case "multiaggregator":
		a.Value = m.Returns[0].Type
		a.Value2 = m.Returns[1].Type
	case "streamreader":
		// The stream drains inside the closure, so the element type is the
		// stamp's — nothing else states what the iterator yields.
		q, stamped := shape.MetaValueType.Get(m.Source.Meta())
		if !stamped || q == "" {
			return nil, "streams elements no stamp names"
		}
		ref, err := golang.RefForQualified(q, b.IfaceName)
		if err != nil {
			return nil, "streams " + q + ", which no closure can spell: " + err.Error()
		}
		a.Value = ref
	case "streamconsumer":
		return nil, "consumes a caller-built stream no derivation can construct"
	case "lifecycle", "voidlifecycle", "poisonaccessor":
		// The call is the whole action.
	}
	return a, ""
}

// pseudoShape is the detector's spelling, refined by the one fact the
// annotator does not state: an aggregator returning a slice is a collector,
// which drains rather than compares.
func pseudoShape(m *suite.Method) string {
	name := shape.Get(m.Source.Meta())
	if name == shapeAggregator && returnsSlice(m) {
		return tiers.ShapeCollector
	}
	return name
}

// collectorElem lifts the collector's element type into a renderable
// reference.
func collectorElem(b *Bindings, m *suite.Method) (sdk.Ref, string) {
	elem := shape.GoSliceElem(m.Returns[0].Source)
	ref, err := golang.RefForQualified(shape.QName(elem), b.IfaceName)
	if err != nil {
		return nil, "collects " + shape.QName(elem) + ", which no reference can spell: " + err.Error()
	}
	return ref, ""
}

// referenceOf fills the strongest reference the shape derives.
//
// Three oracles cover the store models Go interfaces actually declare. A
// composite writer — Put(ctx, k, v) — selects the keyed store, and needs no
// key projection: the key is an argument. A plain writer — Store(ctx, v) —
// selects the map, keyed on the one field of the value that holds the key.
// The composite wins where both appear, because a keyed oracle can host a
// value write only inertly while the reverse loses the delete. A shape no
// store models falls to the twin floor rather than refusing: a weaker
// differential the header names honestly beats an unarmed tier, and `ref=`
// raises the floor by hand.
func referenceOf(
	ctx *sdk.GeneratorContext,
	iface *sdk.Interface,
	harness *suite.Contract,
	b *Bindings,
	keyed, valued, composite, collector *suite.Method,
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

	twin := func(why string) bool {
		b.Reference = Reference{Oracle: OracleTwin, TwinWhy: why}
		return true
	}

	// A claim that defeats store modeling outranks every derivation: the
	// twins lag together, where an immediate oracle reads the claim's own
	// slack as divergence.
	for i := range harness.Methods {
		for _, mixin := range harness.Methods[i].Mixins {
			if reason, defeated := tiers.DefeatsOracles(mixin); defeated {
				return twin(reason)
			}
		}
	}

	if keyed == nil && collector != nil && valued != nil {
		// A value writer beside a collector, nothing keyed to read. The one
		// agreement to check is that the writer adds what the collector
		// returns.
		wroteV, _ := shape.MetaValueType.Get(valued.Source.Meta())
		elem := shape.QName(shape.GoSliceElem(collector.Returns[0].Source))
		if wroteV == "" || wroteV != elem {
			return twin("the drain returns " + elem + " where the writer adds " + wroteV)
		}
		// A derivable key field means upsert semantics — a second add under
		// a held key replaces — and the map is the store that models it. A
		// value with no key is a log entry, deduplicated only where a claim
		// licenses the collapse. The claims scan interface-wide, the way the
		// negation table does: a refinement rides whichever method carries
		// the stamp, and holds over the whole store. The corpus taught both
		// forks — every keyed-map subject diverged from a log at the first
		// repeated add, and the first history subject held two identical
		// events the inferred upsert map collapsed to one.
		history, dedupe := false, false
		for i := range harness.Methods {
			m := &harness.Methods[i]
			for _, c := range append(append([]string{}, m.Mixins...), m.Contracts...) {
				history = history || tiers.DrainsHistory(c)
				dedupe = dedupe || tiers.CollectionDedupes(c)
			}
		}
		if !history {
			if field, keyRef := upsertKeyField(ctx, b, wroteV); field != "" {
				names.Oracle = OracleMap
				names.KeyField = field
				b.Keys.Type = keyRef
				b.Reference = names
				b.Adapter = adapterOf(harness, partners, OracleMap, wroteV)
				return true
			}
		}
		names.Oracle = OracleCollection
		names.Dedupe = dedupe
		b.Reference = names
		b.Adapter = adapterOf(harness, partners, OracleCollection, wroteV)
		return true
	}

	if keyed != nil && composite != nil {
		keyQ, _ := shape.MetaKeyType.Get(keyed.Source.Meta())
		readV, _ := shape.MetaValueType.Get(keyed.Source.Meta())
		putK, _ := shape.MetaKeyType.Get(composite.Source.Meta())
		putV, _ := shape.MetaValueType.Get(composite.Source.Meta())
		if keyQ != putK || readV == "" || readV != putV {
			return twin("the reader speaks (" + keyQ + " → " + readV +
				") where the keyed writer takes (" + putK + ", " + putV + ")")
		}
		names.Oracle = OracleKeyed
		b.Reference = names
		b.Adapter = adapterOf(harness, partners, OracleKeyed, putV)
		return true
	}

	if keyed == nil || valued == nil {
		return twin("no reader/writer pair derives a store")
	}

	keyQ, _ := shape.MetaKeyType.Get(keyed.Source.Meta())
	readV, _ := shape.MetaValueType.Get(keyed.Source.Meta())
	wroteV, _ := shape.MetaValueType.Get(valued.Source.Meta())
	if readV == "" || readV != wroteV {
		return twin("the reader answers " + readV + " where the writer takes " + wroteV)
	}

	field, why := keyFieldOf(ctx, readV, keyQ)
	if field == "" {
		return twin("the key projection is underivable — " + why)
	}

	names.Oracle = OracleMap
	names.KeyField = field
	for _, mixin := range keyed.Mixins {
		// The reader's claim refines the oracle the way the drain's dedupe
		// claim refines the collection: sticky resolution is a different
		// store, not a different pool.
		if tiers.MapStorePins(mixin) {
			names.Pins = true
		}
	}
	b.Reference = names
	b.Adapter = adapterOf(harness, partners, OracleMap, readV)
	return true
}

// upsertKeyField finds the identity field of an unread value type — the
// conventional ID or Key spelling — and lifts its type, for the writer-plus-
// drain interfaces whose subjects upsert by it. No reader states the key
// type, so the convention is the whole signal; a value keyed otherwise falls
// to the collection oracle, and a log subject with an incidental Key field
// falls to ref= — the header names the store either way.
func upsertKeyField(ctx *sdk.GeneratorContext, b *Bindings, valueQ string) (string, sdk.Ref) {
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name != valueQ {
			continue
		}
		for _, preferred := range []string{"ID", "Key"} {
			for _, f := range cand.Fields {
				if f.Name != preferred {
					continue
				}
				ref, err := golang.RefForQualified(shape.QName(f.Type), b.IfaceName)
				if err != nil {
					return "", nil
				}
				return f.Name, ref
			}
		}
	}
	return "", nil
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
// matching operation or holds an inert body. valueQ is the value spelling the
// oracle models — a writer of any other type stays inert, because forwarding
// it would hand the store a value its element clause refuses to compile.
func adapterOf(harness *suite.Contract, partners map[string]string, oracle Oracle, valueQ string) []AdapterMethod {
	out := make([]AdapterMethod, 0, len(harness.Methods))
	for i := range harness.Methods {
		m := &harness.Methods[i]
		am := AdapterMethod{Sig: m.Sig}
		op, fromMixin := oracleOp(oracle, m)
		wroteQ, _ := shape.MetaValueType.Get(m.Source.Meta())
		switch role, partner := partners[m.Name]; {
		case partner:
			am.Reason = role
		case !m.TakesContext():
			am.Reason = "it takes no context to forward to the oracle"
		case op == "":
			am.Reason = "the oracle does not model its shape"
		case !fromMixin && pseudoShape(m) == shapeWriter && wroteQ != valueQ:
			am.Reason = "takes " + wroteQ + " where the oracle holds " + valueQ
		default:
			am.Op = op
		}
		out = append(out, am)
	}
	return out
}

// oracleOp resolves one method's delegation for the chosen oracle: a mixin
// assignment first — the stamp says what a method is for, outranking what it
// looks like — then the oracle's shape table. The second result reports the
// mixin route, whose argument is a key no value check applies to.
func oracleOp(oracle Oracle, m *suite.Method) (string, bool) {
	if oracle == OracleKeyed {
		for _, name := range m.Mixins {
			if op, assigned := tiers.KeyedStoreMixinOp(name); assigned {
				return op, true
			}
		}
	}
	var op string
	switch oracle {
	case OracleKeyed:
		op, _ = tiers.KeyedStoreOp(pseudoShape(m))
	case OracleCollection:
		op, _ = tiers.CollectionOp(pseudoShape(m))
	case OracleMap:
		op, _ = tiers.MapStoreOp(pseudoShape(m))
	case OracleTwin:
	}
	return op, false
}

// poolsOf fills the shared pools from the fixture fields the harness already
// derived, then decides how far past them the value draws reach.
//
// The composite writer supplies the value pool where one exists: a plain
// writer beside it is usually a delete or a touch, whose one argument is a
// key, and a values pool drawn from that would feed keys to every value slot.
//
// The keys stay the fixture pair: collision density is what makes a read
// revisit a write and an overwrite land on held state, and a wide key pool
// would pass every comparison over a history that never collides. The values
// widen wherever the claims license it, and — where the value carries its own
// key — pin that field to the keys pool, so the same key is rewritten under
// different bodies: the overwrite two fixed fixture values never draw.
func poolsOf(
	ctx *sdk.GeneratorContext,
	b *Bindings,
	harness *suite.Contract,
	keyed, valued, composite *suite.Method,
) {
	switch {
	case keyed != nil:
		arg := keyed.CallArgs()[0]
		keyQ, _ := shape.MetaKeyType.Get(keyed.Source.Meta())
		b.Keys = Pool{
			Field:      keyed.ArgFields[0],
			OtherField: keyed.ArgFields[0] + suite.OtherSuffix,
			Type:       arg.Type,
			Q:          keyQ,
		}
	case composite != nil:
		// A keyed writer with no reader still draws keys; its own first
		// argument's fixture pair is the pool.
		arg := composite.CallArgs()[0]
		keyQ, _ := shape.MetaKeyType.Get(composite.Source.Meta())
		b.Keys = Pool{
			Field:      composite.ArgFields[0],
			OtherField: composite.ArgFields[0] + suite.OtherSuffix,
			Type:       arg.Type,
			Q:          keyQ,
		}
	}
	switch {
	case composite != nil:
		arg := composite.CallArgs()[1]
		valueQ, _ := shape.MetaValueType.Get(composite.Source.Meta())
		b.Values = Pool{
			Field:      composite.ArgFields[1],
			OtherField: composite.ArgFields[1] + suite.OtherSuffix,
			Type:       arg.Type,
			Q:          valueQ,
		}
	case valued != nil:
		arg := valued.CallArgs()[0]
		valueQ, _ := shape.MetaValueType.Get(valued.Source.Meta())
		b.Values = Pool{
			Field:      valued.ArgFields[0],
			OtherField: valued.ArgFields[0] + suite.OtherSuffix,
			Type:       arg.Type,
			Q:          valueQ,
		}
	default:
		return
	}

	if restricted := widenValues(ctx, b, harness); restricted {
		// A restricting claim holds the pool to values the harness has
		// proven accepted — recombining a proven body with another key is
		// already a value nothing proved.
		return
	}
	pinValues(ctx, b, keyed, valued, composite)
}

// pinValues pins the value's key field to the keys pool where the value
// carries its own key, deriving the pool itself where no reader supplies one.
//
// A supplied or twin reference skipped the key-field derivation, so the pin
// retries it best-effort: found, the wide pool lands on pooled keys the way a
// derived one does. Not found, a supplied reference narrows — it models a
// store, and wide values keyed afresh would never collide with a key the
// reads revisit — where the twin stays wide: its comparisons are twin against
// twin, and two twins agree about a miss as readily as a hit.
func pinValues(ctx *sdk.GeneratorContext, b *Bindings, keyed, valued, composite *suite.Method) {
	pin := b.Reference.KeyField
	if pin == "" && !b.Reference.Derived() && keyed != nil && valued != nil && composite == nil {
		keyQ, _ := shape.MetaKeyType.Get(keyed.Source.Meta())
		readV, _ := shape.MetaValueType.Get(keyed.Source.Meta())
		wroteV, _ := shape.MetaValueType.Get(valued.Source.Meta())
		if readV != "" && readV == wroteV {
			pin, _ = keyFieldOf(ctx, readV, keyQ)
		}
		if pin == "" && b.Reference.Supplied() {
			b.Values.Wide = false
			b.Values.WhyNarrow = "no key field is derivable to pin a wide draw onto the pooled keys"
			return
		}
	}
	b.Values.Pin = pin
	if pin != "" && keyed == nil {
		// The drain path has no reader to supply keys; the fixture values'
		// own key fields are the colliding set.
		b.Keys.Field = b.Values.Field + "." + pin
		b.Keys.OtherField = b.Values.OtherField + "." + pin
	}
}

// widenValues decides how far past the fixture pair the values pool reaches,
// reporting whether a restricting claim made the decision.
//
// Wide is the default the contract earns: a writer with no restricting claim
// accepts any value of its type, so the pool blends the fixture pair with
// arbitrary draws and a subject refusing one has broken its own claim. A
// [tiers.ValueRestriction] inverts the license, and a type whose graph this
// build cannot see to the bottom would arm a [model.Make] panic instead of a
// deeper run; both keep the pair, and the header says which.
func widenValues(ctx *sdk.GeneratorContext, b *Bindings, harness *suite.Contract) bool {
	drawing := map[string]bool{}
	for _, a := range b.Actions {
		if a.Pool == poolValues {
			drawing[a.Method] = true
		}
	}
	var valueQ string
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if !drawing[m.Name] {
			continue
		}
		for _, mixin := range m.Mixins {
			if reason, restricted := tiers.ValueRestriction(mixin); restricted {
				b.Values.WhyNarrow = "the " + mixin + " claim on " + m.Name + " " + reason
				return true
			}
		}
		if q, ok := shape.MetaValueType.Get(m.Source.Meta()); ok && q != "" {
			valueQ = q
		}
	}
	if valueQ == "" {
		// A drawing method with no value stamp — a mutator's key, say —
		// still has the pool's own spelling to be checked against.
		valueQ = b.Values.Q
	}
	if why := unmakeable(ctx, valueQ, map[string]bool{}); why != "" {
		b.Values.WhyNarrow = why
		return false
	}
	b.Values.Wide = true
	return false
}

// unmakeable reports why [model.Make] cannot be trusted to draw the named
// type, empty where it can.
//
// rapid resolves the type graph at run time and panics on a kind it cannot
// draw, so the walk here is the mirror of that resolution over what this
// build can see: a scalar draws, an exported struct field recurses, an
// unexported one is skipped the way Make skips it, and anything else — a
// declaration out of reach, an interface, a spelling the frontend does not
// resolve to a struct — keeps the pool narrow instead of arming a panic.
func unmakeable(ctx *sdk.GeneratorContext, typeQ string, seen map[string]bool) string {
	if scalarKinds[typeQ] || seen[typeQ] {
		return ""
	}
	seen[typeQ] = true
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name != typeQ {
			continue
		}
		for _, f := range cand.Fields {
			if r, _ := utf8.DecodeRuneInString(f.Name); !unicode.IsUpper(r) {
				// Unexported: Make leaves it zero, which draws fine.
				continue
			}
			if why := unmakeable(ctx, shape.QName(f.Type), seen); why != "" {
				return why
			}
		}
		return ""
	}
	return typeQ + " reaches a type this build cannot prove a wide draw serves"
}

// scalarKinds is the predeclared vocabulary a wide draw serves unconditionally.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var scalarKinds = map[string]bool{
	"bool": true, "string": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "float32": true, "float64": true,
}

// feederOf picks the writer whose values fill the pool: the one agreeing
// with the reader — or, with no reader, with the collector's element — else
// the first in declaration order, so the choice never depends on where a
// method sits in the source.
func feederOf(keyed, collector *suite.Method, writers []*suite.Method) *suite.Method {
	if len(writers) == 0 {
		return nil
	}
	want := ""
	switch {
	case keyed != nil:
		want, _ = shape.MetaValueType.Get(keyed.Source.Meta())
	case collector != nil:
		want = shape.QName(shape.GoSliceElem(collector.Returns[0].Source))
	}
	if want != "" {
		for _, w := range writers {
			if q, _ := shape.MetaValueType.Get(w.Source.Meta()); q == want {
				return w
			}
		}
	}
	return writers[0]
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
