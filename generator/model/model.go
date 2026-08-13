// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"slices"
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
const Version = "0.39.0"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts an interface in.
const DirectiveName sdk.DirectiveName = "model"

// RefKey names a constructor in the source package that builds the reference,
// for an interface whose shape no shipped oracle models.
const RefKey = "ref"

// GenKey names a generator constructor in the routed output package —
// `func() *model.Generator[V]` over the values pool's type — for a value the
// wide pools cannot draw by reflection, or one whose domain the consumer
// knows better than any reflection walk.
const GenKey = "gen"

// WitnessKey names the concrete types a generic interface's property runs at,
// comma-separated in declaration order — `witness=string,int`. The same key
// the stub directive reads, because both answer the same question: a Go test
// cannot carry type parameters, so the source names the types or the tier
// declines. Required here where the stub derives: the pools, the reference
// and every law land at these types, and a silently guessed palette would
// change what the property asserts.
const WitnessKey = "witness"

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

// TracePkg is the trace vocabulary's import path — the event type the
// generated session classifier reads.
const TracePkg = "go.thesmos.sh/testkit/core/trace"

// sessionMixins are the five per-client guarantees, each carrying the
// version= param that names the ordering stamp on the value.
//
// mixinMonotonicReads is the read-ordering session mixin's spelling — the
// one the tests and the vocabulary list both name.
const mixinMonotonicReads = "monotonicreads"

//nolint:gochecknoglobals // a vocabulary list, read-only after init.
var sessionMixins = []string{
	mixinMonotonicReads, "monotonicwrites", "readyourwrites", "writesfollowreads",
	"causal",
}

// SessionSpec is the derived per-client classification: the one closure the
// session laws share, spelled once at file level so the sequential property
// and the concurrent leg reference the same derivation.
type SessionSpec struct {
	// ClassifyName is the generated file-level function's identifier.
	ClassifyName string

	// Reader is the keyed read the classifier interprets, with TakesCtx
	// mirrored for the header.
	Reader string

	// Value is the read's result type; KeyField and VersionField are its
	// identity and ordering members.
	Value                  sdk.Ref
	KeyField, VersionField string

	// Writer is the upserter-shaped write whose answered state carries the
	// stamp — empty where no write surfaces one, which classifies writes
	// out and binds the read-ordering law alone.
	Writer string

	// Key is the pool key type the laws instantiate at.
	Key sdk.Ref
}

// PublisherSpec is the derived subscription drain: the file-level sweep's
// identifier and the types its closure is spelled at.
type PublisherSpec struct {
	// DrainName is the generated sweep's identifier.
	DrainName string

	// Sub is the subscription handle's type — the receive channel — and Msg
	// the element it carries.
	Sub, Msg sdk.Ref
}

// SuppliedOption is one consumer-supplied door: the law field it fills, the
// config field the guarded registration reads, and the closure type spelled
// at the fixture's own instantiation.
type SuppliedOption struct {
	// Field is the law struct's field — the option is <Iface>Model<Field>.
	Field string

	// Config is the config struct's field — the field's name with its first
	// rune lowered, which the law literal and the guard both read.
	Config string

	// Shape selects the closure type's template arm; Iface, Key, Elem and
	// Out are the refs each arm spells.
	Shape                 string
	Iface, Key, Elem, Out sdk.Ref
}

// addSuppliedOption records a door once: two laws reading one field — the
// three isolation levels share History — get one option, and a second law
// asking the same name at a different shape is a refusal, not a shadow.
func (b *Bindings) addSuppliedOption(o *SuppliedOption) string {
	for _, have := range b.SuppliedOptions {
		if have.Config != o.Config {
			continue
		}
		if have.Shape != o.Shape {
			return "asks the " + o.Config + " option at a second shape"
		}
		return ""
	}
	b.SuppliedOptions = append(b.SuppliedOptions, o)
	return ""
}

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
	shapeAnsweringWriter = "answeringwriter"
	shapeAggregator      = "aggregator"
	shapeMultiReader     = "multireader"
	shapeBatchReader     = "batchreader"
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
				"Generates property-based state-machine tests for the annotated "+
					"interface: random sequences of its methods run against the "+
					"subject and a known-good in-memory reference side by side, "+
					"compared after every call. "+RefKey+" names a constructor in "+
					"the source package returning the interface, for a shape no "+
					"shipped reference models. "+GenKey+" names a generator "+
					"constructor in the routed output package, for a value type "+
					"the wide pools cannot draw by reflection. "+WitnessKey+
					" names the concrete types a generic interface's property "+
					"runs at, comma-separated in declaration order; required "+
					"exactly where the interface is generic.",
			).
			AllowedKeys(RefKey, GenKey, WitnessKey).
			On(sdk.NodeKindInterface).
			DenyNegation().
			Build(),
	}
}

// fieldKey is the identity convention's canonical member spelling.
const fieldKey = "Key"

// keyFieldConventions are the field names read as a value's identity, in
// preference order — the upsert inference's one convention.
//
//nolint:gochecknoglobals // a vocabulary table, read-only after init.
var keyFieldConventions = []string{"ID", fieldKey}

// Bindings is the value queued once per interface carrying the directive.
type Bindings struct {
	sdk.BaseEmit
	suite.Subject

	// Witnesses are the concrete types a generic interface's property runs
	// at, in declaration order — empty in the ordinary non-generic case. The
	// template renders them wherever the file names one of the harness's own
	// generic types, and [Bindings.IfaceRef] arrives already instantiated.
	Witnesses []sdk.Ref

	// witnessQ maps each type parameter's name to its witness's stamp
	// spelling. The annotator spells a parameter's key or value type by its
	// bare name, so every stamp read routes through [Bindings.substQ].
	witnessQ map[string]string

	// Session is the derived per-client classification, nil where no session
	// mixin stamps a version. One derivation, referenced by the sequential
	// registry and the concurrent leg alike. sessionKeyField is the value's
	// identity member, derived beside the twin decision where the reader is
	// in hand.
	Session         *SessionSpec
	sessionKeyField string

	// Publisher is the derived subscription sweep, nil where no publisher
	// law binds through a channel-answering subscribe. One derivation: the
	// file-level sweep, the option that outranks it, and the property local
	// every drain field reads.
	Publisher *PublisherSpec

	// SuppliedOptions are the typed doors this file generates: one option
	// and one config field per supplied law field, spelled at the fixture's
	// own types. A law reading one registers only when it is set.
	SuppliedOptions []*SuppliedOption

	// OptionName is `<Iface>Model` — the option a consumer passes to the
	// contract entry to run this tier. PropertyName is `<Iface>ModelProperty`,
	// the composition point it and any bespoke harness share. OptionTypeName
	// and ConfigName carry the tier's own option surface.
	OptionName, PropertyName, OptionTypeName, ConfigName string

	// SatLaws and SatMutants are the saturation surface: per bound law, the
	// methods its closures reach; per reached method, the mutant wrappers
	// the generated prover wears. Binding a law is necessary; this is what
	// makes it sufficient — a law no mutant of its own methods can redden
	// is bound but unsaturatable, and the prover says so by name.
	SatLaws    []SatLaw
	SatMutants []SatMutant

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

	// LawPools are the pools the laws declare beyond the shared pair: a wide
	// input domain for a stateless claim, the adversarial strings a safety
	// claim needs. LawsUseFixture marks a law closure anchored on a fixture
	// key, which obliges the property to construct the fixture.
	LawPools       []LawPool
	LawsUseFixture bool

	// UsesClock marks a clock-bound law in the set, which obliges the
	// property to offer the ModelClocked option and guard those laws on it.
	UsesClock bool

	// Coalesced marks a bound coalescing law, which obliges the property to
	// declare the locked call counter its compute probe increments.
	Coalesced bool

	// RecordsHistory marks a bound no-drops law: the property declares the
	// append log at HistoryElem, the append action records into it, and the
	// runner clears it each iteration.
	RecordsHistory bool
	HistoryElem    sdk.Ref

	// ConcEntry is the append leg's entry type — the appender method's own
	// argument, because the writer action's value stamp is the answered
	// offset there, not the entry the log holds.
	ConcEntry sdk.Ref

	// ConcFamily picks the concurrent leg's model: empty for none, "kv" for
	// the keyed-store pair, "lease" for the acquire/release table.
	ConcFamily string

	// ConcReader and ConcWriter are the actions the kv leg drives against
	// the Porcupine keyed-store model; ConcAcquire and ConcRelease the
	// lease leg's two. All point into Actions, so the closures the legs
	// spell agree with the sequential ones about every method and type.
	ConcReader, ConcWriter   *Action
	ConcAcquire, ConcRelease *Action

	// PkgName is where Layout routed the file — see [Bindings.SetOutputPackages].
	PkgName string

	// contractKeySrc is the contract-oracle role method whose argument is the
	// store's key domain, and contractKeyedRoles the role methods that draw
	// it — a lease's acquire and release take the key, and a reader-less
	// keyed contract would otherwise never declare the pool its laws draw.
	contractKeySrc     *suite.Method
	contractKeyedRoles map[string]bool

	// concAcquireName and concReleaseName are the lease leg's role methods,
	// recorded by the derivation for the action lookup the leg wires.
	concAcquireName, concReleaseName string
}

// The lease contract's role spellings, as the directives stamp them, and
// the concurrent-leg families the template branches on.
const (
	roleLeaseAcquire = "acquire"
	roleLeaseRelease = "release"

	concFamilyKV      = "kv"
	concFamilyLease   = "lease"
	concFamilySession = "session"
	concFamilyCAS     = "cas"
	concFamilyAppend  = "append"

	// shapeCASWriter is the re-pointed shape the contract-role pass spells
	// for the cas write, matched here when the cell leg derives.
	shapeCASWriter = "cas.writer"
)

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

// Concurrent reports whether a linearizability leg derives: the map-shaped
// pair the keyed-store model speaks, or a contract family whose own model
// ships — each unrefined by a claim that changes what its operations mean.
func (b *Bindings) Concurrent() bool { return b.ConcFamily != "" }

// LeaseHeld is the held sentinel the lease leg's model matches — the same
// identity the oracle constructor renders.
func (b *Bindings) LeaseHeld() CtorErr {
	for _, e := range b.Reference.CtorErrs {
		if e.Sym != nil || e.Name != "" {
			return e
		}
	}
	return CtorErr{}
}

// CasMismatch is the stale-version sentinel the cas leg's model matches —
// the spec's first error row, the same identity the sequential oracle's
// constructor consumes.
func (b *Bindings) CasMismatch() CtorErr {
	if len(b.Reference.CtorErrs) == 0 {
		return CtorErr{}
	}
	return b.Reference.CtorErrs[0]
}

// LinearizePkg surfaces the Porcupine wiring's import path to the templates.
func (*Bindings) LinearizePkg() string { return LinearizePkg }

// HistoryPkg surfaces the append log's import path to the template that
// declares the recording local.
func (*Bindings) HistoryPkg() string { return HistoryPkg }

// RootPkg surfaces the runtime module's import path — the prover's
// FailableTB lives there.
func (*Bindings) RootPkg() string { return RootPkg }

// SatNeedsFixture reports whether any wearable defect flaps the fixture
// pair, which obliges the prover to construct the fixture.
func (b *Bindings) SatNeedsFixture() bool {
	for _, m := range b.SatMutants {
		if m.Kind == kindFlap {
			return true
		}
	}
	return false
}

// ModelPkg surfaces the runner's import path to the templates, which can
// reach a method and not a const.
func (*Bindings) ModelPkg() string { return ModelPkg }

// RefPkg returns the reference package's import path.
func (*Bindings) RefPkg() string { return RefPkg }

// ClockPkg surfaces the test clock's import path to the templates.
func (*Bindings) ClockPkg() string { return ClockPkg }

// TracePath surfaces the trace vocabulary's import path to the templates.
func (*Bindings) TracePath() string { return TracePkg }

// LawPath surfaces the law package's import path to the templates.
func (*Bindings) LawPath() string { return LawPkg }

// TierName returns the tier's base path.
func (*Bindings) TierName() string { return TierName }

// TierPath is the path this interface's run reports under: "model" where an
// independent oracle stands opposite the subject, "model/twin" where the
// subject's own factory is the floor — in the test output, because a weaker
// claim a reader has to open a generated file to learn about is a claim
// most readers hold wrong.
func (b *Bindings) TierPath() string {
	if b.Reference.Twin() {
		return TierName + "/twin"
	}
	return TierName
}

// NeedsFixture reports whether anything in the property reads the fixture —
// a pool, a fixture-anchored law closure, or a multi-argument writer's
// per-position pairs. An unused local is a compile error in a generated file.
func (b *Bindings) NeedsFixture() bool {
	if b.UsesKeys() || b.UsesValues() || b.LawsUseFixture {
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

// SatLaw is one bound law's saturation obligation: its identifier, the
// methods its closures reach, and the arming it waits on — supplied doors
// or the clocked factory — without which the prover skips it visibly.
type SatLaw struct {
	ID      string
	Methods []string
	Guards  []string
	Clocked bool

	// Unwearable marks a law whose closures reach no method — doors and
	// traces only — which the prover skips by name rather than dooms.
	Unwearable bool

	// AcceptSemantic widens the kill criterion to the runner's semantic
	// divergence — for the one law that is the differential restated, whose
	// violation the actions catch before the law can speak.
	AcceptSemantic bool
}

// SatMutant is one wearable defect: a method answered wrongly in one of
// the prover's kinds — inert (zeros), flap (the fixture pair alternated),
// or wane (a descending count). Params and Returns carry the signature the
// override renders.
type SatMutant struct {
	Method  string
	Kind    string
	Params  []sdk.Ref
	Returns []sdk.Ref

	// TakesCtx marks the leading context parameter; Out is the flapped,
	// waned or faded result's type where the kind answers one; Last indexes
	// the trailing error return the sputtering kind mints into.
	TakesCtx bool
	Out      sdk.Ref
	Last     int

	// Over is the literal a boundary-crossing wear answers, and ViaLen says
	// the crossing is a length rather than a value — a slice of that many
	// elements instead of the number itself.
	//
	// Derived from the law's own stamped bound, which is the only place the
	// line is written down. A wear invented from the shape cannot know it:
	// every generic defect this prover wears answers *inside* an aggregate's
	// declared range, which is why a bound law survived all of them and
	// read as unsaturatable.
	Over   string
	ViaLen bool

	// Seq is the arity of a streamed result — 1 for an `iter.Seq`, 2 for an
	// `iter.Seq2`, zero for anything else. A wear answering the zero value
	// for one of those hands back a nil function, and ranging over it panics
	// before the law it was worn for is ever consulted.
	Seq int
}

// SeqHelper names the runtime helper that answers an empty sequence of the
// wear's own shape, or the empty string where the result is not a stream.
//
// A method rather than arithmetic in the template: which helper applies is a
// fact about the signature, and the template's job is to spell the call.
func (m SatMutant) SeqHelper() string {
	switch m.Seq {
	case 1:
		return "EmptySeq"
	case 2:
		return "EmptySeq2"
	}
	return ""
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

	// GenFunc names the consumer's generator constructor where the gen=
	// directive key supplied one; the wide arm draws through it instead of
	// reflection.
	GenFunc string

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

	// MissSym is the declaration's own miss sentinel, routed into the
	// oracle's constructor where a mixin stamps one — the guard a
	// sentinel-checking law reads then matches the identity the fixture
	// declared, instead of a minted private error it can never equal.
	// Nil falls back to the minted MissName var.
	MissSym *sdk.Expr

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

	// The contract oracle's own surface: ContractStore is the ref type,
	// ContractName the claim that selected it, ContractArg its one type
	// argument, and CtorErrs the constructor's error arguments in order —
	// a named entry is a minted sentinel, an unnamed one renders nil.
	ContractStore string
	ContractName  string
	ContractArg   sdk.Ref
	CtorErrs      []CtorErr

	// CtorFns are ref-package functions instantiated at ContractArg and
	// called with nothing, rendered before the error slots — the oracle's
	// own semantics choices, like the chain's default hash. VersionField is
	// the value field the version= stamp names; when set, the constructor's
	// first argument is the generated projection of it.
	CtorFns      []string
	VersionField string

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
	OracleContract   Oracle = "contract"
	OracleTwin       Oracle = "twin"
)

// CtorErr is one error argument of a contract oracle's constructor. Name is
// the generated sentinel's identifier, empty where the slot renders nil.
type CtorErr struct {
	// Name is the minted sentinel's identifier, empty where the slot is nil
	// or the declaration stamped one. Sym is the stamped sentinel — the
	// declaration's own error, which gives the oracle and the bound law one
	// identity to agree on.
	Name, Msg string
	Sym       *sdk.Expr
}

// Supplied reports that the directive named the reference.
func (r Reference) Supplied() bool { return r.SuppliedCtor != nil }

// IsContract reports the contract oracle: role-stamped delegation over a
// shipped store whose semantics are the claim's own.
func (r Reference) IsContract() bool { return r.Oracle == OracleContract }

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
	case OracleContract:
		return r.ContractStore
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

	// Records marks the append the no-drops law watches: the closure logs
	// every successful call into the property's append history.
	Records bool

	// Sentinel is the declaration's stamped miss identity, armed on the
	// error-answering reader shapes: where both sides err, the pair must
	// also agree on whether the error is this one. Nil where nothing is
	// stamped, and the comparison stays presence-only.
	Sentinel *sdk.Expr

	// TxCommit and TxRollback spell the two-phase composite's terminal
	// methods, with their ctx flags beside them: the template threads one
	// begin's handle into its own drawn terminal, which is the driving a
	// standalone commit could never do — its handles came from a pool no
	// begin filled. Value carries the handle type.
	TxCommit, TxRollback       string
	TxCommitCtx, TxRollbackCtx bool
}

// ActionPkg is the engine constructors' import path, for the option a
// template appends beside the closure.
func (*Action) ActionPkg() string { return actionPkg }

// ActionArg is one drawn argument of a multi-argument writer or a
// parameterised pure call.
type ActionArg struct {
	// Field is the fixture field the position samples; Type its slice
	// literal's element clause. Wide blends the pair with arbitrary draws —
	// licensed for pure inputs unconditionally, because a pure call stores
	// nothing a claim could refuse.
	Field string
	Type  sdk.Ref
	Wide  bool
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
	// Collect marks an op that streams through an iterator while the method
	// answers a slice, so the body drains rather than returning the call.
	Op      string
	Collect bool

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
		witnesses, usable := modelWitnesses(ctx, iface)
		if !usable {
			continue
		}

		b, ok := bindingsOf(ctx, c, iface, harness, witnesses)
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
			queued = append(queued, companionOf(c, iface, b))
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

	// Saturated marks a harness that emitted the saturation prover, which
	// the companion then holds to the derived reference — the proof ships
	// with the derivation.
	Saturated bool

	// ConcurrentName is the concurrent leg's runner, empty where none
	// derives. The companion holds the leg to the derived reference: a
	// mutex-guarded store is linearizable, so a red run is the wiring's own.
	ConcurrentName string

	// Mutants is the kill matrix: one row per driven method, each a
	// reference whose one method answers zeros and forwards nothing. The
	// property must fail every row — a mutant that survives means that
	// method's participation in the run checks nothing, which is a hole in
	// this derivation rather than in any consumer's subject.
	Mutants []Mutant

	// LowerIface prefixes the mutant type names.
	LowerIface string
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

// Mutant is one row of the companion's kill matrix.
type Mutant struct {
	// Method is the one method the mutant makes inert; Sig spells the
	// override.
	Method string
	Sig    *golang.Sig
}

// RootPkg surfaces the runtime module's import path to the template, which
// reaches the failure surrogate through it.
func (*Companion) RootPkg() string { return RootPkg }

// companionOf derives the companion from the bindings it proves.
func companionOf(c *sdk.Provenance, iface *sdk.Interface, b *Bindings) *Companion {
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
	comp.Saturated = len(b.SatLaws) > 0
	comp.LowerIface = strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:]
	// One kill-matrix row per driven method: the coherence rule already
	// guarantees each has a live adapter op, so its inertness is observable —
	// by the comparison that reads it, or by the read that follows it.
	sigs := map[string]*golang.Sig{}
	for _, am := range b.Adapter {
		sigs[am.Sig.Name] = am.Sig
	}
	for _, a := range b.Actions {
		comp.Mutants = append(comp.Mutants, Mutant{Method: a.Method, Sig: sigs[a.Method]})
	}
	return comp
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
// witnessedHarness rewrites a generic interface's projection at its
// witnesses: every parameter and return naming a type parameter lands at the
// concrete type, and the subject reference arrives instantiated. The
// projection is the suite's emit value, shared with every plugin that reads
// it, so the rewrite clones — methods, signatures and the subject alike —
// and the non-generic path returns the harness untouched.
func witnessedHarness(
	harness *suite.Contract, iface *sdk.Interface, witnesses []sdk.Ref,
) (*suite.Contract, map[string]string) {
	by := golang.WitnessBindings(iface.TypeParams, witnesses)
	if by == nil {
		return harness, nil
	}
	clone := *harness
	clone.IfaceRef = sdk.External(iface.Package, iface.Name, witnesses...)
	methods := make([]suite.Method, len(harness.Methods))
	for i := range harness.Methods {
		m := harness.Methods[i]
		sig := *m.Sig
		params := make([]golang.Param, len(sig.Params))
		for j, p := range sig.Params {
			p.Type = golang.SubstituteTypeParams(p.Type, by)
			params[j] = p
		}
		returns := make([]golang.Return, len(sig.Returns))
		for j, r := range sig.Returns {
			r.Type = golang.SubstituteTypeParams(r.Type, by)
			returns[j] = r
		}
		sig.Params, sig.Returns = params, returns
		m.Sig = &sig
		methods[i] = m
	}
	clone.Methods = methods

	q := make(map[string]string, len(by))
	for name, ref := range by {
		q[name] = witnessSpelling(ref)
	}
	return &clone, q
}

// witnessSpelling is a witness in the annotator's stamp vocabulary: bare for
// a builtin, package-qualified for anything else — the same form
// [golang.RefForQualified] lifts back into a reference.
func witnessSpelling(r sdk.Ref) string {
	if ext, qualified := r.(*sdk.ExternalRef); qualified {
		return ext.Package + "." + ext.Name
	}
	if b, builtin := r.(*sdk.BuiltinRef); builtin {
		return b.Name
	}
	return ""
}

// substQ rewrites a classification stamp's spelling at the witnesses: a
// stamp naming a type parameter answers the concrete type the property runs
// at, and every other spelling passes through untouched.
func (b *Bindings) substQ(q string) string {
	if w, bound := b.witnessQ[q]; bound {
		return w
	}
	return q
}

// modelWitnesses resolves the concrete types a generic interface's property
// runs at, or reports the interface unusable after diagnosing why.
//
// Required rather than derived: the stub's companion only has to compile, so
// a derived palette serves it, but the property's pools, reference and laws
// all assert THROUGH these types — a silent guess would change the claim.
// Nothing here checks a witness satisfies its constraint; a wrong one is a
// compile error naming the type in the generated file, which is the best
// available outcome for a fact the generator cannot know.
func modelWitnesses(ctx *sdk.GeneratorContext, iface *sdk.Interface) ([]sdk.Ref, bool) {
	if len(iface.TypeParams) == 0 {
		return nil, true
	}
	dir := iface.Directive(DirectiveName)
	raw, given := "", false
	if dir != nil {
		raw, given = dir.KV[WitnessKey]
	}
	if !given {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: interface %q is generic, and the property, the reference and "+
				"the pools all land at concrete types; name them with %s= — one "+
				"per type parameter, in declaration order",
			Name, iface.Name, WitnessKey)
		return nil, false
	}
	names := strings.Split(raw, ",")
	if len(names) != len(iface.TypeParams) {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %s=%q on %s names %d types for %d type parameters; supply one per parameter",
			Name, WitnessKey, raw, iface.Name, len(names), len(iface.TypeParams))
		return nil, false
	}
	out := make([]sdk.Ref, 0, len(names))
	for _, n := range names {
		out = append(out, golang.RefFor(strings.TrimSpace(n), iface.Package))
	}
	return out, true
}

func bindingsOf(
	ctx *sdk.GeneratorContext,
	c *sdk.Provenance,
	iface *sdk.Interface,
	harness *suite.Contract,
	witnesses []sdk.Ref,
) (*Bindings, bool) {
	harness, witnessQ := witnessedHarness(harness, iface, witnesses)
	b := &Bindings{
		BaseEmit:       sdk.EmitBase(c, iface),
		Subject:        harness.Subject,
		OptionName:     harness.IfaceName + "Model",
		PropertyName:   harness.IfaceName + "ModelProperty",
		OptionTypeName: harness.IfaceName + "ModelOption",
		ConfigName:     strings.ToLower(harness.IfaceName[:1]) + harness.IfaceName[1:] + "ModelConfig",
		EntryName:      harness.EntryName,
		FixtureCtor:    harness.Fixture.CtorName,
		Witnesses:      witnesses,
		witnessQ:       witnessQ,
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
		case shapeWriter, shapeAnsweringWriter:
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
	valued := feederOf(b, keyed, collector, writers)

	valueQ := ""
	if valued != nil {
		valueQ, _ = b.valueQOf(valued)
	}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if role, partner := partners[m.Name]; partner {
			b.Skipped = append(b.Skipped, Skip{Method: m.Name, Reason: role})
			continue
		}
		a, skip := actionOf(ctx, b, m)
		if skip != "" {
			b.Skipped = append(b.Skipped, Skip{Method: m.Name, Reason: skip})
			continue
		}
		if a.Shape == shapeWriter && a.Pool == poolValues && m != valued {
			if q, _ := b.valueQOf(m); q != valueQ {
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
	// A keyed contract's roles draw keys — an acquire's argument is the
	// lease's key — and the actions were composed before the derivation
	// could say so.
	for _, a := range b.Actions {
		if b.contractKeyedRoles[a.Method] && a.Shape == shapeWriter {
			a.Pool = poolKeys
		}
	}
	// The declaration's miss identity, armed on every error-answering
	// reader: the actions were composed before the reference resolved it,
	// and the identity agreed on here is the same one the derived oracle's
	// constructor consumes — a subject answering a private error where the
	// declaration stamped a sentinel stops reading as agreement.
	if sym := b.Reference.MissSym; sym != nil {
		for _, a := range b.Actions {
			switch a.Shape {
			case shapeReader, shapeMultiReader, shapeBatchReader:
				a.Sentinel = sym
			}
		}
	}
	contractActionsOf(b, harness)
	// The oracle derivation sees only the canonical reader and writer; the
	// pools serve every drawing action, so their sources widen to the
	// fallbacks where the canonical shapes are absent.
	keySrc, valueSrc := keyed, valued
	if keySrc == nil {
		keySrc = keyFallback
	}
	if keySrc == nil {
		keySrc = b.contractKeySrc
	}
	if valueSrc == nil && composite == nil {
		valueSrc = valueFallback
	}
	genFunc, _ := directiveValue(iface, GenKey)
	if strings.Contains(genFunc, ".") {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %s=%q on %q carries a qualifier; name a generator constructor "+
				"in the routed output package",
			Name, GenKey, genFunc, iface.Name)
		return nil, false
	}
	poolsOf(ctx, b, harness, keySrc, valueSrc, composite, genFunc)
	lawsOf(b, harness, partners, keyed)
	saturationOf(b, harness)
	concurrentOf(b, harness, keyed, valued)
	return b, true
}

// concurrentOf wires the linearizability leg where the map derivation holds
// unrefined: the Porcupine keyed-store model speaks reader and writer over
// one key at a time, and a claim that changes what a read means — the sticky
// pin — is a different model, not a different wiring. The leg reuses the
// sequential actions, so both legs draw from the same pools and spell the
// same closures; concurrency that never collides checks nothing, which is
// the mistake the shared pools exist to rule out.
//
// A keyless fold is not here on purpose: its state is one accumulation, so
// no partition derives, and a commutative or associative fold is
// order-insensitive by its own claim — linearizability over an operation
// whose order is unobservable checks close to nothing, and the claims that
// do bite are already bound as sequential laws.
func concurrentOf(b *Bindings, harness *suite.Contract, keyed, valued *suite.Method) {
	// The lease leg: acquire and release over the shared keys pool, checked
	// against the lease-table model — the same op vocabulary the model
	// switches on, and the same lenient release the oracle speaks.
	if b.concAcquireName != "" && b.UsesKeys() {
		for _, a := range b.Actions {
			switch a.Method {
			case b.concAcquireName:
				b.ConcAcquire = a
			case b.concReleaseName:
				b.ConcRelease = a
			}
		}
		if b.ConcAcquire != nil && b.ConcRelease != nil {
			b.ConcFamily = concFamilyLease
			return
		}
		b.ConcAcquire, b.ConcRelease = nil, nil
	}
	// The session leg: the same reader/writer interleaving, checked by the
	// per-client laws over the multi-client trace rather than by Porcupine —
	// a store-assigned version defeats the KV model's value equality, so the
	// model stays stepless and the laws carry the run.
	if b.Session != nil && keyed != nil && valued != nil {
		for _, a := range b.Actions {
			switch a.Method {
			case keyed.Name:
				b.ConcReader = a
			case valued.Name:
				b.ConcWriter = a
			}
		}
		if b.ConcReader != nil && b.ConcWriter != nil {
			b.ConcFamily = concFamilySession
			return
		}
		b.ConcReader, b.ConcWriter = nil, nil
	}
	// The cas leg: the version-guarded write against the cell model, in the
	// live oracle's own dialect — stamp is seen+1, an empty cell matches
	// only the zero version. Only the shipped VersionedCell derives it,
	// because the model matches the stamped mismatch identity the same
	// constructor consumes.
	if b.Reference.Oracle == OracleContract && b.Reference.ContractStore == "VersionedCell" &&
		b.Reference.VersionField != "" {

		var w, r *Action
		for _, a := range b.Actions {
			switch a.Shape {
			case shapeCASWriter:
				w = a
			case shapeAggregator:
				r = a
			}
		}
		if w != nil && r != nil && w.Pool != "" {
			b.ConcWriter, b.ConcReader = w, r
			b.ConcFamily = concFamilyCAS
		}
		return
	}
	// The append leg: offset-answering appends into the one shared history.
	// The monotonic-offsets law states the claim per client; this leg states
	// it across them, which is where a torn append hides.
	if a, entry := appendActionOf(b, harness); a != nil {
		b.ConcWriter = a
		b.ConcEntry = entry
		b.ConcFamily = concFamilyAppend
		return
	}
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
		return
	}
	b.ConcFamily = concFamilyKV
}

// appendActionOf answers the driven offset-answering append of an appender
// contract, nil where the interface carries none. The offset type is held
// to int64 because the shared-history model counts in it; a log offsetting
// otherwise keeps its sequential law and no leg.
func appendActionOf(b *Bindings, harness *suite.Contract) (*Action, sdk.Ref) {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if !slices.Contains(m.Contracts, "appender") {
			continue
		}
		if len(m.CallArgs()) != 1 || len(m.Returns) == 0 ||
			shape.QName(m.Returns[0].Source) != "int64" {

			continue
		}
		if a := b.actionFor(m.Name); a != nil && a.Pool != "" {
			return a, m.CallArgs()[0].Type
		}
	}
	return nil, nil
}

// historyDrained reports whether any classification marks the drained
// slice as an event log — the refinement that outranks every keyed
// election, because a map oracle collapses the repeats a log faithfully
// holds.
func historyDrained(harness *suite.Contract) bool {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		drains := slices.ContainsFunc(m.Mixins, tiers.DrainsHistory) ||
			slices.ContainsFunc(m.Contracts, tiers.DrainsHistory)
		if drains {
			return true
		}
	}
	return false
}

// missSentinelOf reports the declaration's own miss sentinel: the first
// sentinel= or notfound= a mixin stamps anywhere in the method set,
// qualified by the resolver. Routed into the derived oracle's constructor,
// it is what lets a sentinel-checking law's guard match the identity the
// fixture declared — against a minted private error the guard never passes,
// and the law it feeds is dead without anyone saying so.
func missSentinelOf(harness *suite.Contract) *sdk.Expr {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		for _, mx := range m.Mixins {
			for _, key := range []string{"sentinel", "notfound"} {
				v, stamped := shape.MixinParamKey(mx, key).Get(m.Source.Meta())
				if !stamped || v == "" {
					continue
				}
				if pkg, name, qualified := splitQualified(v); qualified {
					return sdk.NewExternal(pkg, name)
				}
			}
		}
	}
	return nil
}

// sessionVersionOf reports the first session mixin carrying a version=
// param anywhere in the method set: the carrying method, the member it
// names, and whether one was found.
func sessionVersionOf(harness *suite.Contract) (carrier *suite.Method, member string, stamped bool) {
	for i := range harness.Methods {
		m := &harness.Methods[i]
		for _, mx := range m.Mixins {
			if !slices.Contains(sessionMixins, mx) {
				continue
			}
			if v, given := shape.MixinParamKey(mx, "version").Get(m.Source.Meta()); given && v != "" {
				return m, v, true
			}
		}
	}
	return nil, "", false
}

// versionFieldDiag holds version= to the value struct's own fields. Every
// projection of the ordering stamp is a field selector — the session
// classifier reads it, and the cas cell assigns it (v.Rev = cur.Rev + 1),
// which no method form can satisfy — so a method or a missing member is
// refused here by name. Without the refusal the stamp passes every layer
// unvalidated and the failure surfaces as a build error in the consumer's
// package, attributed to generated code rather than to the directive that
// caused it. A value type whose struct declaration is out of reach passes
// through: the compile keeps that case honest, and refusing what cannot be
// seen would break a witnessed value spelled by its parameter name.
func versionFieldDiag(ctx *sdk.GeneratorContext, iface *sdk.Interface, valueQ, member string) bool {
	var s *sdk.Struct
	for cand := range ctx.Reader.Structs().All() {
		if cand.Package+"."+cand.Name == valueQ {
			s = cand
			break
		}
	}
	if s == nil {
		return true
	}
	for _, f := range s.Fields {
		if f.Name == member {
			return true
		}
	}
	for _, m := range s.Methods {
		if m != nil && m.Name == member {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: version=%q on %q names a method of %s; the ordering stamp is "+
					"read and assigned as a field, and no method can stand there",
				Name, member, iface.Name, valueQ)
			return false
		}
	}
	ctx.Diag.Errorf(iface.Pos(),
		"%s: version=%q on %q names no member of %s",
		Name, member, iface.Name, valueQ)
	return false
}

// actionOf builds one method's action, or says why there is none.
func actionOf(ctx *sdk.GeneratorContext, b *Bindings, m *suite.Method) (*Action, string) {
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
		if r.Source == nil || golang.IsError(r.Source) {
			// The error return is an interface too, and it is the one every
			// action already knows how to compare.
			continue
		}
		// A live handle — a channel, a function, an interface — compares by
		// identity, and two sides' handles never share one; the comparison
		// would fail every correct subject on its first answer.
		if golang.IsChannel(r.Source) || r.Source.IsFunc() || golang.IsInterface(r.Source) {
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
		if len(m.Returns) != 3 {
			// The action drives a (value, value, error) triple and nothing
			// wider — a page-shaped read answers more, and its law walks it.
			return nil, "answers more than the (value, value, error) triple its action drives"
		}
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
	case shapeAnsweringWriter:
		a.Pool = poolValues
		a.Value = m.CallArgs()[0].Type
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
		if name != shapeAggregator && len(m.CallArgs()) > 0 {
			// A parameterised pure call drives through the drawn-args
			// variant: each position draws from its own fixture pair,
			// blended with arbitrary values wherever the type can be seen
			// to the bottom — sound unconditionally, because a pure call
			// stores nothing a claim could refuse.
			for i, arg := range m.CallArgs() {
				a.Args = append(a.Args, ActionArg{
					Field: m.ArgFields[i],
					Type:  arg.Type,
					Wide:  unmakeable(ctx, shape.QName(arg.Source), map[string]bool{}) == "",
				})
			}
			ctor := "PureVar"
			if name == "predicate" {
				ctor = "PredicateVar"
			}
			a.KindName = sdk.Kind(ActionKindPrefix + name + "var")
			a.Ctor = sdk.NewExternal(actionPkg, ctor)
		}
	case "multiaggregator":
		a.Value = m.Returns[0].Type
		a.Value2 = m.Returns[1].Type
	case "streamreader":
		// The stream drains inside the closure, so the element type is the
		// stamp's — nothing else states what the iterator yields.
		q, stamped := b.valueQOf(m)
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
		MissSym:  missSentinelOf(harness),
	}

	twin := func(why string) bool {
		b.Reference = Reference{Oracle: OracleTwin, TwinWhy: why}
		return true
	}

	// A version-stamped fixture is a claim no store oracle survives: the
	// subject assigns the ordering member on write, a value-storing oracle
	// holds the input's zero, and the first read-back diverges on a correct
	// store. Twins stamp together. The key member is derived here, where
	// the reader is in hand — the classifier needs the same identity the
	// upsert inference reads, and the twin carries no adapter to ask. This
	// arm runs before the defeat scan because a session mixin can sit in
	// both tables, and only this arm derives what the classifier needs.
	if vm, member, stamped := sessionVersionOf(harness); stamped {
		if q, _ := b.valueQOf(vm); q != "" && !versionFieldDiag(ctx, iface, q, member) {
			return false
		}
		if keyed != nil {
			if q, _ := b.valueQOf(keyed); q != "" {
				b.sessionKeyField, _ = upsertKeyField(ctx, b, q)
			}
		}
		return twin("the subject assigns the version member on write, which no value-storing oracle stamps")
	}

	// A claim that defeats store modeling outranks every remaining
	// derivation: the twins lag together, where an immediate oracle reads
	// the claim's own slack as divergence.
	for i := range harness.Methods {
		for _, mixin := range harness.Methods[i].Mixins {
			if reason, defeated := tiers.DefeatsOracles(mixin); defeated {
				return twin(reason)
			}
		}
	}

	// A contract claim outranks the shapes: its roles say what each method
	// is FOR, and the shipped store carries the claim's own semantics —
	// which is more than any shape-derived map can promise.
	handled, lenified, refused := contractOf(ctx, iface, b, harness, partners, names)
	if refused {
		return false
	}
	if handled {
		return true
	}
	if lenified != "" {
		return twin(lenified)
	}

	if (keyed == nil || historyDrained(harness)) && collector != nil && valued != nil {
		// A value writer beside a collector. Ordinarily nothing keyed reads
		// beside them — and where a history claim stands, a keyed read does
		// not change the election: the claim says the drain is an event log,
		// a map oracle collapses the log's repeats, and the corpus caught
		// exactly that when the isolation fixture grew its read. The one
		// agreement to check is that the writer adds what the collector
		// returns; the keyed read stays inert on the log oracle, and the
		// header says so.
		wroteV, _ := b.valueQOf(valued)
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
		history, dedupe := historyDrained(harness), false
		for i := range harness.Methods {
			m := &harness.Methods[i]
			for _, c := range append(append([]string{}, m.Mixins...), m.Contracts...) {
				dedupe = dedupe || tiers.CollectionDedupes(c)
			}
		}
		if !history {
			if field, keyRef := upsertKeyField(ctx, b, wroteV); field != "" {
				names.Oracle = OracleMap
				names.KeyField = field
				b.Keys.Type = keyRef
				b.Reference = names
				b.Adapter = adapterOf(b, harness, partners, OracleMap, wroteV)
				return true
			}
		}
		names.Oracle = OracleCollection
		names.Dedupe = dedupe
		b.Reference = names
		b.Adapter = adapterOf(b, harness, partners, OracleCollection, wroteV)
		return true
	}

	if keyed != nil && composite != nil {
		keyQ, _ := b.keyQOf(keyed)
		readV, _ := b.valueQOf(keyed)
		putK, _ := b.keyQOf(composite)
		putV, _ := b.valueQOf(composite)
		if keyQ != putK || readV == "" || readV != putV {
			return twin("the reader speaks (" + keyQ + " → " + readV +
				") where the keyed writer takes (" + putK + ", " + putV + ")")
		}
		names.Oracle = OracleKeyed
		b.Reference = names
		b.Adapter = adapterOf(b, harness, partners, OracleKeyed, putV)
		return true
	}

	if keyed == nil || valued == nil {
		return twin("no reader/writer pair derives a store")
	}

	keyQ, _ := b.keyQOf(keyed)
	readV, _ := b.valueQOf(keyed)
	wroteV, _ := b.valueQOf(valued)
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
	b.Adapter = adapterOf(b, harness, partners, OracleMap, readV)
	return true
}

// contractOf derives the contract oracle where an interface's stamps resolve
// a shipped store's whole role vocabulary: the carrier's role= names its own
// part, the partner keys name the siblings, and every role must land on a
// method or the family stays underived — half a lease checks nothing a twin
// does not. The store's one type argument is spoken by the type-arg role's
// own signature, and the constructor's error arguments are minted sentinels
// or the lenient nil, per the family's row. refused reports a directive
// invalid by name — a version= that no field projection can satisfy — and
// aborts the whole binding rather than falling through to a weaker oracle.
func contractOf(
	ctx *sdk.GeneratorContext,
	iface *sdk.Interface,
	b *Bindings,
	harness *suite.Contract,
	partners map[string]string,
	names Reference,
) (handled bool, lenified string, refused bool) {
	for i := range harness.Methods {
		carrier := &harness.Methods[i]
		for _, contract := range carrier.Contracts {
			spec, shipped := tiers.ContractStore(contract)
			if !shipped {
				continue
			}
			roles := contractRoleMethods(harness, carrier, contract)
			complete := true
			for _, role := range tiers.ContractRoles(contract) {
				complete = complete && roles[role] != nil
			}
			src := roles[spec.TypeArgRole]
			if !complete || src == nil {
				continue
			}
			var arg sdk.Ref
			var argQ string
			switch {
			case spec.TypeArgResult && len(src.Returns) > 0:
				arg = src.Returns[0].Type
				argQ = shape.QName(src.Returns[0].Source)
			case !spec.TypeArgResult && len(src.CallArgs()) > 0:
				arg = src.CallArgs()[0].Type
				argQ = shape.QName(src.CallArgs()[0].Source)
			}
			if arg == nil {
				continue
			}

			if spec.VersionParam != "" {
				field, stamped := stampValue(harness, carrier,
					shape.ContractParamKey(contract, spec.VersionParam).Name())
				if !stamped {
					// The cell cannot guard what nothing names; the twins
					// stand in and the header's floor says so.
					continue
				}
				if !versionFieldDiag(ctx, iface, b.substQ(argQ), field) {
					return false, "", true
				}
				names.VersionField = field
			}
			names.CtorFns = spec.CtorFns
			names.Oracle = OracleContract
			names.ContractStore = spec.Store
			names.ContractName = contract
			names.ContractArg = arg
			if !spec.TypeArgResult {
				// The store's type argument is a role argument, so the roles
				// draw keys: record the source for the pool derivation and
				// the methods whose actions draw from it.
				b.contractKeySrc = src
				b.contractKeyedRoles = map[string]bool{}
				for _, rm := range roles {
					b.contractKeyedRoles[rm.Name] = true
				}
			}

			lower := strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:]
			minted := false
			for _, e := range spec.Errs {
				ce := CtorErr{Msg: e.Msg}
				if e.Suffix != "" && !roleClaims(roles[e.Role], e.NilUnder) {
					// The declaration's own sentinel where one is stamped —
					// the bound law compares identities, and an oracle
					// disagreeing under a different spelling of the same
					// state would fail every correct subject.
					if sym, stamped := stampedSentinel(harness, carrier, contract, e.Param); stamped {
						ce.Sym = sym
					} else {
						ce.Name = lower + "Model" + e.Suffix
					}
					minted = true
				}
				names.CtorErrs = append(names.CtorErrs, ce)
			}
			if len(spec.Errs) > 0 && !minted {
				// Every sentinel lenified away is an oracle that can never
				// disagree — the kill matrix proved a fully-nil tracker
				// cannot see its own methods go inert. The twins say so
				// instead of a store pretending to check.
				return false, "the claims lenify every sentinel the " + contract +
					" oracle could disagree with", false
			}
			b.Reference = names
			b.Adapter = contractAdapterOf(harness, partners, contract, roles)
			// The concurrent leg's roles, recorded only for an oracle that
			// held: a lenified family fell to the twins above, and a leg
			// wired against a sentinel nothing minted renders nothing valid.
			if spec.ConcModel == "LeaseTable" && roles[roleLeaseAcquire] != nil &&
				roles[roleLeaseRelease] != nil {

				b.concAcquireName = roles[roleLeaseAcquire].Name
				b.concReleaseName = roles[roleLeaseRelease].Name
			}
			return true, "", false
		}
	}
	return false, "", false
}

// stampedSentinel resolves a contract error's declared sentinel, false where
// the parameter is unnamed or unstamped.
func stampedSentinel(
	harness *suite.Contract, carrier *suite.Method, contract, param string,
) (*sdk.Expr, bool) {
	if param == "" {
		return nil, false
	}
	v, ok := stampValue(harness, carrier, shape.ContractParamKey(contract, param).Name())
	if !ok {
		return nil, false
	}
	pkg, name, qualified := splitQualified(v)
	if !qualified {
		return nil, false
	}
	return sdk.NewExternal(pkg, name), true
}

// contractActionsOf re-points contract-role actions to the constructors that
// drive the role as itself. The actions were composed before the contract
// resolved its roles — the keyed-pool pass one loop up is the precedent —
// and the single-method rows are deliberately renames: the writer closure is
// already the constructor's shape, so only the name and the header change.
// The tx composite is the exception with teeth: the begin's action becomes
// the whole begin-terminal cycle and the terminal siblings' standalone
// actions are dropped, because a commit drawn from a value pool operates on
// a handle no begin minted — agreement over bogus handles was the entire
// content of that driving. A recording append keeps its recording closure:
// the rename touches the constructor, never the history log the writer
// template emits around it.
func contractActionsOf(b *Bindings, harness *suite.Contract) {
	for i := range harness.Methods {
		carrier := &harness.Methods[i]
		for _, contract := range carrier.Contracts {
			roles := contractRoleMethods(harness, carrier, contract)
			for role, rm := range roles {
				ctor, mapped := tiers.ContractActionFor(contract, role)
				if !mapped || rm == nil {
					continue
				}
				a := b.actionFor(rm.Name)
				if a == nil {
					continue
				}
				consumed := tiers.ContractActionConsumes(contract, role)
				if len(consumed) == 0 {
					a.Ctor = sdk.NewExternal(actionPkg, ctor)
					a.Shape = contract + "." + role
					continue
				}
				commit, rollback := roles[consumed[0]], roles[consumed[1]]
				if commit == nil || rollback == nil ||
					len(rm.Returns) == 0 || golang.IsError(rm.Returns[0].Source) {

					continue // half a trio, or a begin answering no handle
				}
				cAct, rAct := b.actionFor(commit.Name), b.actionFor(rollback.Name)
				if cAct == nil || rAct == nil {
					continue
				}
				a.Ctor = sdk.NewExternal(actionPkg, ctor)
				a.KindName = sdk.Kind(ActionKindPrefix + "twophase")
				a.Shape = contract + "." + role
				a.Pool = ""
				a.Value = rm.Returns[0].Type
				a.TxCommit, a.TxCommitCtx = commit.Name, commit.TakesContext()
				a.TxRollback, a.TxRollbackCtx = rollback.Name, rollback.TakesContext()
				reason := "driven through the " + rm.Name +
					" composite — a standalone terminal would operate on handles no begin minted"
				b.dropAction(commit.Name, reason)
				b.dropAction(rollback.Name, reason)
			}
		}
	}
}

// actionFor answers the driven action on the named method, nil where the
// method drives nothing.
func (b *Bindings) actionFor(method string) *Action {
	for _, a := range b.Actions {
		if a.Method == method {
			return a
		}
	}
	return nil
}

// dropAction removes the named method's action and records why in the
// header's not-driven block.
func (b *Bindings) dropAction(method, reason string) {
	kept := b.Actions[:0]
	for _, a := range b.Actions {
		if a.Method == method {
			b.Skipped = append(b.Skipped, Skip{Method: method, Reason: reason})
			continue
		}
		kept = append(kept, a)
	}
	b.Actions = kept
}

// roleClaims reports whether the role's method carries the named mixin — the
// stamp that flips a constructor sentinel to the oracle's lenient nil.
func roleClaims(m *suite.Method, mixin string) bool {
	return m != nil && mixin != "" && slices.Contains(m.Mixins, mixin)
}

// contractRoleMethods resolves the named contract's roles to methods: every
// method filling a role by its own stamp, plus each partner key the carrier
// names. Both walks, because a protocol splits its directives — the chain
// stamps append on one method and verify on another, and reading only the
// carrier's would leave a role the interface plainly fills unresolved.
func contractRoleMethods(harness *suite.Contract, carrier *suite.Method, contract string) map[string]*suite.Method {
	out := map[string]*suite.Method{}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if role, ok := shape.ContractRoleKey(contract).Get(m.Source.Meta()); ok && role != "" {
			out[role] = m
		}
	}
	for _, role := range tiers.ContractRoles(contract) {
		v, ok := shape.ContractPartnerKey(contract, role).Get(carrier.Source.Meta())
		if !ok || v == "" {
			continue
		}
		if m := methodOf(harness, golang.LocalName(v)); m != nil {
			out[role] = m
		}
	}
	return out
}

// contractAdapterOf builds the role-stamped delegation table: a method
// filling a role forwards to the role's op, a non-role method whose shape
// the spec's ShapeOps claim forwards likewise — the cell's read is no role
// the cas contract declares — and everything else is inert.
func contractAdapterOf(
	harness *suite.Contract,
	partners map[string]string,
	contract string,
	roles map[string]*suite.Method,
) []AdapterMethod {
	spec, _ := tiers.ContractStore(contract)
	opOf := map[string]string{}
	drains := map[string]bool{}
	for role, m := range roles {
		if op, ok := tiers.ContractRoleOp(contract, role); ok {
			opOf[m.Name] = op
			drains[m.Name] = tiers.ContractRoleDrains(contract, role)
		}
	}
	out := make([]AdapterMethod, 0, len(harness.Methods))
	for i := range harness.Methods {
		m := &harness.Methods[i]
		am := AdapterMethod{Sig: m.Sig}
		op := opOf[m.Name]
		if op == "" {
			op = spec.ShapeOps[pseudoShape(m)]
		}
		switch role, partner := partners[m.Name]; {
		case partner:
			am.Reason = role
		case !m.TakesContext():
			am.Reason = "it takes no context to forward to the oracle"
		case op == "":
			am.Reason = "the " + contract + " oracle models only its roles"
		default:
			am.Op = op
			am.Collect = drains[m.Name]
		}
		out = append(out, am)
	}
	return out
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
		for _, preferred := range keyFieldConventions {
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
	for _, preferred := range keyFieldConventions {
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
func adapterOf(
	b *Bindings, harness *suite.Contract, partners map[string]string, oracle Oracle, valueQ string,
) []AdapterMethod {
	out := make([]AdapterMethod, 0, len(harness.Methods))
	for i := range harness.Methods {
		m := &harness.Methods[i]
		am := AdapterMethod{Sig: m.Sig}
		op, fromMixin := oracleOp(oracle, m)
		wroteQ, _ := b.valueQOf(m)
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
	case OracleContract, OracleTwin:
		// The contract adapter resolves by role, not by shape, and the twin
		// has no adapter at all; neither reaches this table.
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
	genFunc string,
) {
	switch {
	case keyed != nil:
		arg := keyed.CallArgs()[0]
		keyQ, _ := b.keyQOf(keyed)
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
		keyQ, _ := b.keyQOf(composite)
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
		valueQ, _ := b.valueQOf(composite)
		b.Values = Pool{
			Field:      composite.ArgFields[1],
			OtherField: composite.ArgFields[1] + suite.OtherSuffix,
			Type:       arg.Type,
			Q:          valueQ,
		}
	case valued != nil:
		arg := valued.CallArgs()[0]
		valueQ, _ := b.valueQOf(valued)
		b.Values = Pool{
			Field:      valued.ArgFields[0],
			OtherField: valued.ArgFields[0] + suite.OtherSuffix,
			Type:       arg.Type,
			Q:          valueQ,
		}
	default:
		return
	}

	if restricted := widenValues(ctx, b, harness, genFunc); restricted {
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
		keyQ, _ := b.keyQOf(keyed)
		readV, _ := b.valueQOf(keyed)
		wroteV, _ := b.valueQOf(valued)
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
func widenValues(ctx *sdk.GeneratorContext, b *Bindings, harness *suite.Contract, genFunc string) bool {
	// A supplied generator outranks every verdict below: the consumer
	// authored the domain, which is more than a reflection walk or a claim
	// scan can ever know.
	if genFunc != "" {
		b.Values.GenFunc = genFunc
		b.Values.Wide = true
		return false
	}
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
		if q, ok := b.valueQOf(m); ok && q != "" {
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
	"bool": true, builtinString: true, "byte": true, "rune": true,
	builtinInt: true, "int8": true, "int16": true, "int32": true, builtin64: true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "float32": true, "float64": true,
}

// feederOf picks the writer whose values fill the pool: the one agreeing
// with the reader — or, with no reader, with the collector's element — else
// the first in declaration order, so the choice never depends on where a
// method sits in the source.
func feederOf(b *Bindings, keyed, collector *suite.Method, writers []*suite.Method) *suite.Method {
	if len(writers) == 0 {
		return nil
	}
	want := ""
	switch {
	case keyed != nil:
		want, _ = b.valueQOf(keyed)
	case collector != nil:
		want = shape.QName(shape.GoSliceElem(collector.Returns[0].Source))
	}
	if want != "" {
		for _, w := range writers {
			if q, _ := b.valueQOf(w); q == want {
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

// siblingParams returns the named mixin's sibling-reference parameters —
// the callable-kinded keys, whose values name methods of this interface.
// Member-kinded keys stay out: they name methods of a role's answered
// handle, and a sibling scan claiming them would mark an interface method
// that merely shares the name.
func siblingParams(name string) []string {
	for _, m := range mixins.All() {
		if m.Name == name {
			var out []string
			for _, p := range m.Params {
				if p.Kind == shape.KindCallable {
					out = append(out, p.Key)
				}
			}
			return out
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

// keyQOf reads a method's key-type stamp through the witness substitution:
// the annotator spells a type parameter by its bare name, and the property
// runs at the concrete type the source pinned. The stamped flag mirrors the
// meta read for the sites that distinguish absent from empty.
//
//nolint:unparam // mirrors valueQOf; callers today read only the spelling
func (b *Bindings) keyQOf(m *suite.Method) (string, bool) {
	q, stamped := shape.MetaKeyType.Get(m.Source.Meta())
	return b.substQ(q), stamped
}

// valueQOf is [Bindings.keyQOf] for the value-type stamp.
func (b *Bindings) valueQOf(m *suite.Method) (string, bool) {
	q, stamped := shape.MetaValueType.Get(m.Source.Meta())
	return b.substQ(q), stamped
}
