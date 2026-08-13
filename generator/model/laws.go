// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
	"go.thesmos.sh/testkit/generator/tiers"
)

// LawFieldKindPrefix composes each law field's emit kind —
// `model.lawfield.<Shape>` — which is the template that renders it.
//
// Dispatch is on the closure's shape, not the field's name: the catalogue
// spells Read on a keyed store and Read on a version cell with one word, and
// the two closures could not be less alike. The shape table below is the
// transcription of each law struct's field types, held to the shipped structs
// the same way the binding rows are — a wrong shape renders a closure that
// fails to compile in whichever corpus package arms it.
const LawFieldKindPrefix = "model.lawfield."

// LawBinding is one law, instantiated and filled, in the generated registry.
type LawBinding struct {
	sdk.BaseEmit

	// ID is the identifier the law reports under, for the header.
	ID string

	// Ctor is the law struct, qualified; Args are its type arguments after
	// the subject, resolved against the interface. Ptr addresses the literal,
	// for a stateful law whose Check lives on the pointer.
	Ctor *sdk.Expr
	Args []sdk.Ref
	Ptr  bool

	// Fields fill the struct, each through its shape's template. Clocked
	// marks a binding whose Advance reads the run's test clock, armed only
	// where the ModelClocked option supplies a subject on it. Session marks
	// a per-client law, re-registered on the concurrent leg where its
	// multi-client trace lives.
	Fields  []*LawField
	Clocked bool
	Session bool
}

// Kind returns the one template every binding renders through.
func (*LawBinding) Kind() sdk.Kind { return "model.law" }

// LawField is one filled field of a law struct.
type LawField struct {
	sdk.BaseEmit

	// KindName selects the field's template by its closure shape.
	KindName sdk.Kind

	// Name is the struct field, for the composite literal.
	Name string

	// Method is the role method a closure field calls, with TakesCtx saying
	// whether the call forwards the run's context.
	Method   string
	TakesCtx bool

	// Iface, Key and Value spell the closure's parameter and result types at
	// the pools' own types.
	Iface, Key, Value sdk.Ref

	// In and Out spell a closure's own types where they are not the pools':
	// the domain a roundtrip draws, the offset an append answers, the state
	// an observation returns.
	In, Out sdk.Ref

	// Pool names the shared local a generator field reuses, and KeyOfName the
	// shared key projection a handle field reuses — the same values the
	// actions and the derived reference already draw from, which is the
	// one-derivation rule inside the file.
	Pool, KeyOfName string

	// Const is a constant field's qualified value — a sentinel the
	// declaration stamped, rendered where a manifest names its stamp key.
	// Lit is the literal form, for a numeric stamp like a bound.
	Const *sdk.Expr
	Lit   string

	// KeyField names the fixture key a fixture-anchored closure reads or
	// writes — the fixed key an idempotent composite write repeats, the key
	// a keyed observation revisits.
	KeyField string

	// Pairs are the permitted transitions a workflow stamp declares, parsed
	// from its `from>to` list.
	Pairs [][2]string
}

// Kind returns the field's template key.
func (f *LawField) Kind() sdk.Kind { return f.KindName }

// ModelPkg surfaces the runner's import path to the field templates, whose
// closures take the runner's *T.
func (*LawField) ModelPkg() string { return ModelPkg }

// LawPool is one pool a law draws that the sequences do not: a wide input
// domain for a stateless claim, the adversarial strings a safety claim needs.
type LawPool struct {
	// Name is the local the property declares; Q is the element's stamp
	// spelling, so two laws asking for one name at two types are caught.
	Name, Q string

	// Elem is the drawn element type; Adversarial selects the hostile-string
	// generator, Offsets the bounded-duration one, instead of the
	// reflective default.
	Elem        sdk.Ref
	Adversarial bool
	Offsets     bool
}

// The law-declared pool names.
const (
	poolInputs   = "inputs"
	poolPayloads = "payloads"
	poolOffsets  = "offsets"
)

// builtinString and builtinInt are the builtins several derivations spell.
const (
	builtinString = "string"
	builtinInt    = "int"
)

// lawShape names the closure a law field renders — the template dispatch.
type lawShape string

// The closure vocabulary, one per distinct field type across the law structs.
const (
	shapeKeyedRead  lawShape = "Read"        // func(rt, T, K) (V, error)
	shapeValueOp    lawShape = "Write"       // func(rt, T, V) error
	shapeDrainSlice lawShape = "Drain"       // func(rt, T) ([]V, error), slice role
	shapeDrainSeq   lawShape = "DrainSeq"    // same field over an iterator role
	shapeScalar     lawShape = "Scalar"      // func(rt, T) (R, error)
	shapeScalarLen  lawShape = "ScalarLen"   // same, R = len of a returned slice
	shapeBoolCall   lawShape = "BoolCall"    // func(rt, T) bool
	shapeResultCall lawShape = "ResultCall"  // func(rt, T) R
	shapeInputCall  lawShape = "InputCall"   // func(rt, T, X) (R, error)
	shapeCtxOp      lawShape = "CtxOp"       // func(ctx, T) error
	shapeErrOp      lawShape = "ErrOp"       // func(rt, T) error
	shapeKeyedOp    lawShape = "KeyedOp"     // func(rt, T, K) error
	shapeKVOp       lawShape = "KVOp"        // func(rt, T, k, v string) error
	shapeSum        lawShape = "Sum"         // func(rt, T) int64
	shapeMerge      lawShape = "Merge"       // func(rt, dst, src T) error
	shapeSave       lawShape = "Save"        // func(rt, T, V) (K, error), K synthesized
	shapeAppendOff  lawShape = "Append"      // func(rt, T, V) (Off, error)
	shapeReplay     lawShape = "ReplaySlice" // func(rt, T, K) iter.Seq2[E, error]

	shapeOkOp        lawShape = "OkOp"        // func(rt, T) bool — err-op success
	shapeNextOp      lawShape = "NextOp"      // func(rt, T) (V, bool, error)
	shapeDoOp        lawShape = "DoOp"        // func(rt, T) — fire and forget
	shapePinnedWrite lawShape = "PinnedWrite" // func(rt, T, K, V) error — pin then put
	shapeCtxOpFixed  lawShape = "CtxOpFixed"  // func(ctx, T) error — fixed fixture arg
	shapeScheduleAt  lawShape = "ScheduleAt"  // func(rt, T, at time.Duration) error
	shapeCountObs    lawShape = "CountObs"    // func(rt, T) int — loud on error
	shapeSubscribe   lawShape = "Subscribe"   // func(rt, T) (Sub, error) — the handle kept, never compared
)

// lawRoleShapes transcribes each rowed law's role-field closure types from
// the engine structs — the second half of what the binding row transcribes.
// A rowed law's role field missing here is a build refusal by name, never a
// wrong guess.
//
// The field and role spellings this file repeats — each names one concept.
const (
	fRead    = "Read"
	fWrite   = "Write"
	fDrain   = "Drain"
	fCall    = "Call"
	fCount   = "Count"
	fProbe   = "Probe"
	fClose   = "Close"
	fromSelf = "self"

	// handleClassifier is the manifest spelling of the per-client
	// trace-classifier handle.
	handleClassifier = "trace-classifier"

	// The publisher rows' role spellings, and the drain option's manifest
	// name.
	fSubscribe = "Subscribe"
	fPublish   = "Publish"
	fRedeliver = "Redeliver"
	optDrain   = "drain"
	builtin64  = "int64"
)

//nolint:gochecknoglobals // a lookup table, read-only after init.
var lawRoleShapes = map[string]map[string]lawShape{
	lawid.Cacheable:             {fRead: shapeKeyedRead},
	lawid.DefaultOnError:        {fRead: shapeKeyedRead},
	lawid.DeleteReturnsNotFound: {fRead: shapeKeyedRead},
	lawid.PointInTime:           {fRead: shapeKeyedRead},
	lawid.ReadAfterWrite:        {fRead: shapeKeyedRead},
	lawid.Sticky:                {fRead: shapeKeyedRead},
	lawid.WriteObservable:       {fWrite: shapeValueOp, fRead: shapeKeyedRead},

	lawid.StreamCompletion:   {fDrain: shapeDrainSlice},
	lawid.StreamNoDuplicates: {fDrain: shapeDrainSlice},
	lawid.StreamOverMatch:    {fDrain: shapeDrainSlice},
	lawid.StreamPermutation:  {fDrain: shapeDrainSlice},
	lawid.StreamReentrant:    {"Collect": shapeDrainSlice},
	lawid.StreamStableOrder:  {fDrain: shapeDrainSlice},
	lawid.StreamReflectsMutations: {
		"Put":    shapeValueOp,
		"Delete": shapeValueOp,
		fDrain:   shapeDrainSlice,
	},

	lawid.AggregatorBounded:        {fRead: shapeScalar},
	lawid.CountEqualsReference:     {"Count": shapeScalar},
	lawid.LifecycleRespectsContext: {"Op": shapeCtxOp},
	lawid.MonotonicNonDecreasing:   {fRead: shapeScalar},
	lawid.PoisonIdempotentRead:     {fProbe: shapeErrOp},
	lawid.PoisonNilOnFresh:         {fProbe: shapeErrOp},
	lawid.PredicateConsistent:      {fCall: shapeBoolCall},
	lawid.PureDeterministic:        {fCall: shapeResultCall},
	lawid.TotalOver:                {fCall: shapeInputCall},

	lawid.Associative:      {"Apply": shapeValueOp},
	lawid.AtomicWrite:      {fWrite: shapeValueOp},
	lawid.CommutativeWrite: {fWrite: shapeValueOp},
	lawid.Conservative:     {fWrite: shapeValueOp, "Sum": shapeSum},
	lawid.CRDTMerge:        {fWrite: shapeValueOp, "Merge": shapeMerge},
	lawid.IdempotentWrite:  {fWrite: shapeValueOp},
	lawid.InjectionSafe:    {"Store": shapeKVOp, "Load": shapeKeyedRead},
	lawid.XSSSafe:          {"Render": shapeInputCall},

	lawid.AppenderMonotonicOffsets: {"Append": shapeAppendOff},
	lawid.CASAtomicOneWinner:       {"CAS": shapeValueOp, fRead: shapeScalar},
	lawid.LeakFree:                 {"Open": shapeErrOp, "Close": shapeErrOp},
	lawid.LeaseDoubleAcquireBlocks: {"Acquire": shapeKeyedOp, "Release": shapeKeyedOp},
	lawid.PersisterRetrievable:     {"Save": shapeSave, fRead: shapeKeyedRead},
	lawid.Roundtrip:                {"Forward": shapeInputCall, "Inverse": shapeInputCall},
	lawid.LossyRoundtrip:           {"Forward": shapeInputCall, "Inverse": shapeInputCall},
	lawid.UpdaterReplaces:          {"Update": shapeValueOp, fRead: shapeKeyedRead},
	lawid.UpserterIdempotent:       {"Upsert": shapeValueOp, fRead: shapeKeyedRead},
	lawid.ValidTransition:          {fWrite: shapeValueOp},

	lawid.AppendOnlyGrows:          {"Replay": shapeReplay},
	lawid.HashChainIntegrityVerify: {"Verify": shapeErrOp},
	lawid.ReplayDeterministic:      {"Replay": shapeReplay},

	lawid.TamperEvident:         {fWrite: shapeValueOp, "Tamper": shapeOkOp, "Verify": shapeErrOp},
	lawid.CursorCloseIdempotent: {fClose: shapeErrOp},
	lawid.CursorNextAfterClose:  {fClose: shapeErrOp, "Next": shapeNextOp},

	lawid.PublisherDelivers:    {fSubscribe: shapeSubscribe, fPublish: shapeValueOp},
	lawid.PublisherAtLeastOnce: {fSubscribe: shapeSubscribe, fPublish: shapeValueOp, fRedeliver: shapeValueOp},
	lawid.PublisherAtMostOnce:  {fSubscribe: shapeSubscribe, fPublish: shapeValueOp, fRedeliver: shapeValueOp},
	lawid.PublisherExactlyOnce: {fSubscribe: shapeSubscribe, fPublish: shapeValueOp, fRedeliver: shapeValueOp},
	lawid.IdempotentLifecycle:  {fCall: shapeErrOp},
	lawid.LifecycleAfterClose:  {fClose: shapeErrOp, "Op": shapeErrOp},
	lawid.PoisonConsistent:     {"Poison": shapeDoOp, fProbe: shapeErrOp},

	lawid.TTLExpiry:                  {"Put": shapePinnedWrite, fRead: shapeKeyedRead},
	lawid.DeadlineRespecting:         {"Op": shapeCtxOpFixed},
	lawid.ScheduledFiresAfterAdvance: {"Schedule": shapeScheduleAt, "FiredCount": shapeCountObs},
	lawid.Windowed:                   {"Incr": shapeKeyedOp, fCount: shapeKeyedRead},
}

// lawsOf selects and fills every law the interface's classifications earn.
//
// Selection is [tiers.Select] over each non-partner method's whole
// classification set. A selected rule that cannot be filled lands in
// [Bindings.Unbound] with what it is waiting on — rendered in the header,
// because a law that quietly failed to bind reads as a claim the run checks.
func lawsOf(b *Bindings, harness *suite.Contract, partners map[string]string, keyed *suite.Method) {
	// Selection composes per method, but a claim holds over the interface —
	// the sticky stamp rides the reader and negates the writer-earned
	// observability law — so the conflict scan runs against every method's
	// mixins, partners included: an excluded method's claim still holds.
	claims := map[string]bool{}
	for i := range harness.Methods {
		for _, name := range harness.Methods[i].Mixins {
			claims[name] = true
		}
	}
	// The derived adapter's inert arms: a law reaching one compares against
	// a body answering zeros — and the companion drives the adapter as a
	// full subject, where the lie becomes a red run on the reference itself.
	inert := map[string]string{}
	if b.Reference.Derived() {
		for _, am := range b.Adapter {
			if am.Op == "" {
				inert[am.Sig.Name] = am.Reason
			}
		}
	}
	// One outcome per (law, selecting method): a contract classification
	// rides every role method, and re-selecting the same rule from each
	// would register one law twice and print one refusal per carrier.
	seen := map[string]bool{}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		if _, partner := partners[m.Name]; partner && len(m.Mixins) == 0 {
			// A role-overridden partner — a validator, a tamper — carries no
			// claim of its own and selects nothing. A partner that hosts its
			// own mixin still voices it: the leakfree open half names itself,
			// and excluding it from the sequences must not silence its law.
			continue
		}
		selectable := classificationsOf(m)
		if _, partner := partners[m.Name]; partner {
			selectable = m.Mixins
		}
		for _, r := range tiers.Select(selectable, paramsOf(harness, m)) {
			if reason, negated := negatedBy(claims, r.Law); negated {
				if !seen[r.Law+"\x00"+reason] {
					seen[r.Law+"\x00"+reason] = true
					b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
				}
				continue
			}
			before := len(b.Unbound)
			binding, ok := lawOf(b, harness, r, m, keyed, inert)
			if !ok {
				// lawOf appended the refusal; keep it only if new.
				added := b.Unbound[before:]
				b.Unbound = b.Unbound[:before]
				for _, u := range added {
					if !seen[u.Method+"\x00"+u.Reason] {
						seen[u.Method+"\x00"+u.Reason] = true
						b.Unbound = append(b.Unbound, u)
					}
				}
				continue
			}
			key := r.Law + "\x00bound\x00" + bindingFingerprint(binding)
			if seen[key] {
				continue
			}
			seen[key] = true
			b.Laws = append(b.Laws, binding)
		}
	}
}

// bindingFingerprint spells what makes two bindings the same law twice: the
// methods its fields close over. Two writers earning one law separately are
// two bindings; one contract riding two roles is one.
func bindingFingerprint(lb *LawBinding) string {
	var out strings.Builder
	for _, f := range lb.Fields {
		out.WriteString(f.Name + "=" + f.Method + ";")
	}
	return out.String()
}

// negatedBy resolves the first conflict row a held claim triggers, in the
// table's own order so the generated header is deterministic.
func negatedBy(claims map[string]bool, law string) (string, bool) {
	for _, n := range tiers.LawNegations() {
		if n.Law == law && claims[n.Mixin] {
			return n.Reason, true
		}
	}
	return "", false
}

// lawOf fills one rule, false where [Bindings.Unbound] records why not.
func lawOf(
	b *Bindings,
	harness *suite.Contract,
	r tiers.Rule,
	m, keyed *suite.Method,
	inert map[string]string,
) (*LawBinding, bool) {
	spec, specified := tiers.BindingFor(r.Law)
	if !specified {
		b.Unbound = append(b.Unbound, Skip{
			Method: r.Law,
			Reason: "the catalogue carries no instantiation spec for it",
		})
		return nil, false
	}

	pkg := LawPkg
	if spec.Timeaware {
		pkg = TimeawarePkg
	}
	lb := &LawBinding{
		BaseEmit: b.BaseEmit,
		ID:       r.Law,
		Ctor:     sdk.NewExternal(pkg, spec.Type),
		Ptr:      spec.Ptr,
		// The subject leads every law's argument list; the spec spells only
		// what follows it.
		Args: []sdk.Ref{b.IfaceRef},
	}
	for _, a := range spec.Args {
		ref, reason := resolveArg(b, harness, r, a, m, keyed)
		if reason != "" {
			b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
			return nil, false
		}
		lb.Args = append(lb.Args, ref)
	}

	for _, f := range r.Fields {
		field, reason := lawFieldOf(b, harness, r, f, m, keyed)
		if reason != "" {
			b.Unbound = append(b.Unbound, Skip{Method: r.Law, Reason: reason})
			return nil, false
		}
		if field != nil {
			lb.Fields = append(lb.Fields, field)
		}
	}
	for _, field := range lb.Fields {
		if field.Kind() == sdk.Kind(LawFieldKindPrefix+"Advance") {
			// A clock-bound law arms only where the ModelClocked option
			// supplies a subject on the run's clock; the template guards it.
			lb.Clocked = true
			b.UsesClock = true
		}
		if field.Kind() == sdk.Kind(LawFieldKindPrefix+"Classify") {
			lb.Session = true
		}
		if reason, held := inert[field.Method]; field.Method != "" && held {
			b.Unbound = append(b.Unbound, Skip{
				Method: r.Law,
				Reason: field.Name + " reaches " + field.Method +
					", which the derived reference answers inertly — " + reason,
			})
			return nil, false
		}
	}
	return lb, true
}

// resolveArg lifts one binding-row argument into a renderable type.
func resolveArg(
	b *Bindings, harness *suite.Contract, r tiers.Rule, a tiers.BindArg, m, keyed *suite.Method,
) (sdk.Ref, string) {
	switch a {
	case tiers.BindKey:
		if b.Keys.Type == nil {
			return nil, "instantiates at a key type no method here draws"
		}
		return b.Keys.Type, ""
	case tiers.BindValue:
		if b.Values.Type == nil {
			return nil, "instantiates at a value type no method here draws"
		}
		return b.Values.Type, ""
	case tiers.BindObservation:
		obs, reason := observationOf(b, harness, keyed)
		if reason != "" {
			return nil, reason
		}
		return obs.Out, ""
	case tiers.BindPartition:
		// The single anonymous partition, until a partition projection is
		// declared and stamped.
		return sdk.Builtin(builtinString), ""
	}

	form, fieldName, qualified := a.Qualifier()
	if !qualified {
		return nil, "instantiates through " + string(a) + ", which nothing resolves"
	}
	role, reason := ruleFieldRole(b, harness, r, fieldName, m, keyed)
	if reason != "" {
		return nil, reason
	}
	switch form {
	case "result":
		if lawRoleShapes[r.Law][fieldName] == shapeNextOp {
			// NextOp's closure carries the multi-valued return whole, so
			// the law instantiates at the first non-error result — the
			// element a cursor's Next yields beside its ok flag.
			return firstResultType(role)
		}
		ref, _, why := resultType(role)
		return ref, why
	case "input":
		if len(role.CallArgs()) == 0 {
			return nil, "instantiates at " + role.Name + "'s input, and it takes none"
		}
		return role.CallArgs()[0].Type, ""
	case "elem":
		return drainedElem(b, role)
	case "scalar":
		ref, _, why := scalarType(role)
		return ref, why
	}
	return nil, "instantiates through " + string(a) + ", which nothing resolves"
}

// ruleFieldRole resolves a binding argument's field reference to the method
// that fills it — the same resolution the field itself gets.
func ruleFieldRole(
	b *Bindings, harness *suite.Contract, r tiers.Rule, fieldName string, m, keyed *suite.Method,
) (*suite.Method, string) {
	for _, f := range r.Fields {
		if f.Name != fieldName {
			continue
		}
		if f.Kind != tiers.KindRole {
			return nil, "instantiates through " + fieldName + ", which is not a role field"
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, fieldName + " " + reason
		}
		return role, ""
	}
	return nil, "instantiates through " + fieldName + ", which the manifest does not name"
}

// firstResultType is a method's first non-error result — the instantiation
// point for a law whose closure shape returns the method's results whole,
// where [resultType]'s single-valued strictness would refuse the method.
func firstResultType(m *suite.Method) (sdk.Ref, string) {
	for i := range m.Returns {
		ret := &m.Returns[i]
		if ret.Source != nil && golang.IsError(ret.Source) {
			continue
		}
		return ret.Type, ""
	}
	return nil, "observes through " + m.Name + ", which answers nothing to observe"
}

// resultType is a method's single non-error result.
func resultType(m *suite.Method) (sdk.Ref, *golang.Return, string) {
	results := make([]*golang.Return, 0, len(m.Returns))
	for i := range m.Returns {
		ret := &m.Returns[i]
		if ret.Source != nil && golang.IsError(ret.Source) {
			continue
		}
		results = append(results, ret)
	}
	if len(results) == 0 {
		return nil, nil, "observes through " + m.Name + ", which answers nothing to observe"
	}
	if len(results) > 1 {
		return nil, nil, "observes through " + m.Name +
			", which answers several results no single-valued closure returns"
	}
	return results[0].Type, results[0], ""
}

// scalarType is a method's scalar observation: its single non-error result,
// or the length of the slice it returns.
func scalarType(m *suite.Method) (ref sdk.Ref, viaLen bool, reason string) {
	if returnsSlice(m) {
		return sdk.Builtin(builtinInt), true, ""
	}
	r, _, why := resultType(m)
	return r, false, why
}

// drainedElem is the element type of the stream a method drains — a slice's
// element, or the stamped yield of an iterator.
func drainedElem(b *Bindings, m *suite.Method) (sdk.Ref, string) {
	if returnsSlice(m) {
		return collectorElem(b, m)
	}
	q, stamped := b.valueQOf(m)
	if !stamped || q == "" {
		return nil, "drains " + m.Name + ", which streams elements no stamp names"
	}
	ref, err := golang.RefForQualified(q, b.IfaceName)
	if err != nil {
		return nil, "drains " + q + ", which no closure can spell: " + err.Error()
	}
	return ref, ""
}

// observation is the composed whole-state read the before/after laws share.
type observation struct {
	Method   *suite.Method
	Out      sdk.Ref
	Keyed    bool
	TakesCtx bool
}

// observationOf derives the strongest whole-state observation the interface
// offers: the drained collection, the aggregate, or a read of the fixture
// key — in that order, because each earlier one sees strictly more.
func observationOf(
	b *Bindings,
	harness *suite.Contract,
	keyed *suite.Method,
) (*observation, string) {
	var agg, keyedReader *suite.Method
	for i := range harness.Methods {
		m := &harness.Methods[i]
		switch pseudoShape(m) {
		case tiers.ShapeCollector:
			elem, why := collectorElem(b, m)
			if why != "" {
				continue
			}
			return &observation{Method: m, Out: sdk.SliceOf(elem), TakesCtx: m.TakesContext()}, ""
		case shapeAggregator:
			if agg == nil && len(m.CallArgs()) == 0 {
				if _, _, why := resultType(m); why == "" {
					agg = m
				}
			}
		case shapeReader:
			if keyedReader == nil {
				keyedReader = m
			}
		}
	}
	if agg != nil {
		out, _, _ := resultType(agg)
		return &observation{Method: agg, Out: out, TakesCtx: agg.TakesContext()}, ""
	}
	if keyedReader != nil && b.UsesKeys() && b.Keys.Field != "" {
		out, _, why := resultType(keyedReader)
		if why == "" {
			return &observation{
				Method: keyedReader, Out: out, Keyed: true, TakesCtx: keyedReader.TakesContext(),
			}, ""
		}
	}
	if keyed != nil && b.UsesKeys() && b.Keys.Field != "" {
		out, _, why := resultType(keyed)
		if why == "" {
			return &observation{
				Method:   keyed,
				Out:      out,
				Keyed:    true,
				TakesCtx: keyed.TakesContext(),
			}, ""
		}
	}
	return nil, "observes state through no method here — no drain, no aggregate, no keyed read"
}

// lawFieldOf fills one manifest entry: a field, nil for one the law defaults,
// or the reason nothing can fill it.
func lawFieldOf(
	b *Bindings, harness *suite.Contract, r tiers.Rule, f tiers.Field, m, keyed *suite.Method,
) (*LawField, string) {
	field := &LawField{
		BaseEmit: b.BaseEmit,
		Name:     f.Name,
		Iface:    b.IfaceRef,
		Key:      b.Keys.Type,
		Value:    b.Values.Type,
	}

	switch f.Kind {
	case tiers.KindDefault:
		// The law's Check defaults it; a generated value would be a second
		// opinion about a number the law already owns.
		return nil, ""
	case tiers.KindTrace:
		// The runner binds the trace on any law implementing TraceBinder;
		// a generated value would race the binding it already gets.
		return nil, ""
	case tiers.KindSupplied:
		if f.From == optDrain {
			return drainFieldOf(b, harness, f, field, m, keyed)
		}
		if f.Optional {
			// The manifest says zero is sound: the law reads the field's
			// absence as the claim's unrefined form, so the binding omits it
			// and the option that would fill it stays a consumer's choice.
			return nil, ""
		}
		return nil, f.Name + " waits on the " + f.From + " option, which no generated value can stand in for"
	case tiers.KindRole:
		got, reason := roleFieldOf(b, harness, r, f, field, m, keyed)
		if reason != "" && f.Optional {
			// The manifest says absence is the claim's unrefined form — a
			// redeliver nothing declares skips the redelivery arm, never
			// the law.
			return nil, ""
		}
		return got, reason
	case tiers.KindConstant:
		return constFieldOf(harness, r, f, field, m)
	case tiers.KindGenerator:
		return generatorFieldOf(b, harness, r, f, field, m, keyed)
	case tiers.KindHandle:
		return handleFieldOf(b, harness, r, f, field, m, keyed)
	}
	return nil, f.Name + " has the unknown kind " + string(f.Kind)
}

// roleFieldOf fills a closure field per its law's transcribed shape.
func roleFieldOf(
	b *Bindings, harness *suite.Contract, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *suite.Method,
) (*LawField, string) {
	role, reason := roleMethod(b, harness, f.From, m, keyed)
	if reason != "" {
		return nil, f.Name + " " + reason
	}
	shapes, known := lawRoleShapes[r.Law]
	sh, mapped := shapes[f.Name]
	if !known || !mapped {
		return nil, f.Name + " closes over " + role.Name +
			", and the generator transcribes no closure shape for it"
	}
	field.Method = role.Name
	field.TakesCtx = role.TakesContext()
	field.KindName = sdk.Kind(LawFieldKindPrefix + string(sh))

	switch sh {
	case shapeDrainSeq, shapeScalarLen:
		// Override spellings, never table entries: the slice arms below pick
		// them when the role streams or the observation is a length.
		return nil, f.Name + " names an override shape no table row spells"
	case shapeKeyedRead:
		spec, _ := tiers.BindingFor(r.Law)
		if why := keyedReadMismatch(b, f.Name, role, slices.Contains(spec.Args, tiers.BindValue)); why != "" {
			return nil, why
		}
		// The closure is typed by the role itself — the pools agree where the
		// mismatch check above demands it, and a reader whose value no pool
		// draws (a cache, a persister's load) still compiles.
		field.Key = role.CallArgs()[0].Type
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Value = out
		return field, ""

	case shapeValueOp:
		return valueOpField(b, f, field, role)

	case shapeDrainSlice:
		elem, why := drainedElem(b, role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Value = elem
		if !returnsSlice(role) {
			field.KindName = sdk.Kind(LawFieldKindPrefix + string(shapeDrainSeq))
		}
		return field, ""

	case shapeScalar:
		ref, viaLen, why := scalarType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary observation supplies"
		}
		field.Out = ref
		switch {
		case viaLen:
			field.KindName = sdk.Kind(LawFieldKindPrefix + string(shapeScalarLen))
		case !role.ReturnsError():
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ScalarNoErr")
		}
		if r.Law == lawid.AggregatorBounded && !viaLen && !orderedScalar(role) {
			return nil, f.Name + " observes " + role.Name +
				"'s result, which no bound orders"
		}
		if r.Law == lawid.CountEqualsReference && identityCompared(role) {
			return nil, f.Name + " observes " + role.Name +
				"'s result, a live handle only identity could compare"
		}
		return field, ""

	case shapeBoolCall:
		if len(role.CallArgs()) > 0 || len(role.Returns) != 1 ||
			role.Returns[0].Source == nil || !golang.IsBuiltinNamed(role.Returns[0].Source, "bool") {

			return nil, f.Name + " closes over " + role.Name + ", which is not a bare predicate"
		}
		return field, ""

	case shapeResultCall:
		if len(role.CallArgs()) > 0 || role.ReturnsError() {
			return nil, f.Name + " closes over " + role.Name + ", which is not a bare pure call"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		return field, ""

	case shapeInputCall:
		if len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes several inputs no single-value closure composes"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.In = role.CallArgs()[0].Type
		field.Out = out
		return field, ""

	case shapeCtxOp:
		if !role.TakesContext() || len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a context-respecting error operation"
		}
		return field, ""

	case shapeErrOp:
		if len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a nullary error operation"
		}
		return field, ""

	case shapeKeyedOp:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which is not a keyed error operation"
		}
		if b.Keys.Q != "" {
			if q, _ := b.keyQOf(role); q != "" && q != b.Keys.Q {
				return nil, f.Name + " closes over " + role.Name +
					", which keys on " + q + " beside a pool of " + b.Keys.Q
			}
		}
		return field, ""

	case shapeKVOp:
		args := role.CallArgs()
		if len(args) != 2 || !errOnly(role) || !stringParam(args[0]) || !stringParam(args[1]) {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a string-keyed string write"
		}
		return field, ""

	case shapeSum:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary observation supplies"
		}
		_, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if ret.Source == nil || !integerResult(ret) {
			return nil, f.Name + " observes " + role.Name + "'s result, which no sum totals"
		}
		return field, ""

	case shapeMerge:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not merge one peer"
		}
		return field, ""

	case shapeSave:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not save one value"
		}
		if b.Reference.KeyField == "" {
			return nil, f.Name + " synthesizes the saved identity from the key projection, " +
				"which was not derivable here"
		}
		field.KeyOfName = b.KeyOfName()
		field.In = role.CallArgs()[0].Type
		field.Out = b.Keys.Type
		return field, ""

	case shapeAppendOff:
		if len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name + ", which does not append one value"
		}
		out, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if !integerResult(ret) {
			return nil, f.Name + " expects an offset, and " + role.Name + " answers none"
		}
		field.In = role.CallArgs()[0].Type
		field.Out = out
		return field, ""

	case shapeSubscribe:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name + ", which takes inputs no subscription draw supplies"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		return field, ""

	case shapeOkOp:
		if len(role.CallArgs()) > 0 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which is not a nullary error operation"
		}
		return field, ""

	case shapeNextOp:
		if len(role.CallArgs()) > 0 || len(role.Returns) != 3 || !role.ReturnsError() {
			return nil, f.Name + " closes over " + role.Name +
				", which does not answer the (value, more, error) triple"
		}
		field.Out = role.Returns[0].Type
		return field, ""

	case shapeDoOp:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary corruption supplies"
		}
		return field, ""

	case shapePinnedWrite:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not put one value"
		}
		if b.Values.Pin == "" {
			return nil, f.Name + " pins the drawn key into the value, and this pool pins nothing"
		}
		field.In = role.CallArgs()[0].Type
		field.KeyField = b.Values.Pin
		return field, ""

	case shapeCtxOpFixed:
		if !role.TakesContext() || !errOnly(role) || len(role.CallArgs()) != 1 {
			return nil, f.Name + " closes over " + role.Name +
				", which is not a one-input context operation"
		}
		if len(role.ArgFields) == 0 {
			return nil, f.Name + " anchors on a fixture field the projection does not carry"
		}
		field.KeyField = role.ArgFields[0]
		b.LawsUseFixture = true
		return field, ""

	case shapeScheduleAt:
		if len(role.CallArgs()) != 1 || !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which does not take one offset"
		}
		return field, ""

	case shapeCountObs:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs no nullary observation supplies"
		}
		_, ret, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if !integerResult(ret) {
			return nil, f.Name + " counts " + role.Name + "'s result, which is not a count"
		}
		return field, ""

	case shapeReplay:
		if len(role.CallArgs()) > 0 {
			return nil, f.Name + " closes over " + role.Name +
				", which takes inputs the single-partition replay does not thread"
		}
		elem, why := drainedElem(b, role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if !returnsSlice(role) {
			return nil, f.Name + " drains " + role.Name +
				", which streams through an iterator this adapter does not compose"
		}
		field.Out = elem
		return field, ""
	}
	return nil, f.Name + " has the unrendered shape " + string(sh)
}

// valueOpField fills a single-value mutation closure: the role's one value
// input, or a composite write anchored on the fixture key.
func valueOpField(
	b *Bindings,
	f tiers.Field,
	field *LawField,
	role *suite.Method,
) (*LawField, string) {
	args := role.CallArgs()
	switch {
	case len(args) == 1:
		field.In = args[0].Type
		if !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which answers more than an error"
		}
		return field, ""
	case len(args) == 2 && b.UsesKeys() && b.Keys.Field != "":
		// A composite write anchored on the fixture key: the law repeats one
		// (key, value) pair, which is its claim restricted to a key every
		// other draw revisits.
		q, _ := b.keyQOf(role)
		if q != "" && b.Keys.Q != "" && q != b.Keys.Q {
			return nil, f.Name + " closes over " + role.Name +
				", which keys on " + q + " beside a pool of " + b.Keys.Q
		}
		if !errOnly(role) {
			return nil, f.Name + " closes over " + role.Name + ", which answers more than an error"
		}
		field.In = args[1].Type
		field.KeyField = b.Keys.Field
		field.KindName = sdk.Kind(LawFieldKindPrefix + "WriteFixedKey")
		b.LawsUseFixture = true
		return field, ""
	default:
		return nil, f.Name + " closes over " + role.Name +
			", which takes several inputs no single-value closure composes"
	}
}

// constFieldOf fills a stamped constant: a qualified sentinel, a numeric
// literal, or the workflow's transition list.
func constFieldOf(
	harness *suite.Contract, r tiers.Rule, f tiers.Field, field *LawField, m *suite.Method,
) (*LawField, string) {
	value, ok := stampValue(harness, m, f.From)
	if !ok {
		if f.Optional {
			return nil, ""
		}
		return nil, f.Name + " reads the " + f.From + " stamp, which this declaration does not carry"
	}

	if r.Law == lawid.ValidTransition && f.Name == "Allowed" {
		pairs, why := transitionPairs(value)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Pairs = pairs
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Transitions")
		return field, ""
	}

	if r.Law == lawid.PublisherAtLeastOnce || r.Law == lawid.PublisherAtMostOnce ||
		r.Law == lawid.PublisherExactlyOnce {
		// The mode spelling is the engine's own enum, not a symbol the
		// source declares — the directive says which claim, the law package
		// says what it is called.
		mode, spelled := deliveryModes[value]
		if !spelled {
			return nil, f.Name + "'s stamp names " + value + ", which is not a delivery mode"
		}
		field.Const = sdk.NewExternal(LawPkg, mode)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Sentinel")
		return field, ""
	}

	if pkg, name, qualified := splitQualified(value); qualified {
		field.Const = sdk.NewExternal(pkg, name)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Sentinel")
		return field, ""
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		field.Lit = value
		field.KindName = sdk.Kind(LawFieldKindPrefix + "ConstLit")
		return field, ""
	}
	if d, err := time.ParseDuration(value); err == nil {
		// The duration as untyped nanoseconds: assignable to time.Duration,
		// and free of any import the literal spelling would drag in.
		field.Lit = strconv.FormatInt(d.Nanoseconds(), 10)
		field.KindName = sdk.Kind(LawFieldKindPrefix + "ConstLit")
		return field, ""
	}
	return nil, f.Name + "'s stamp names " + value +
		", which is neither a qualified symbol nor a number"
}

// deliveryModes maps the directive's mode spellings to the engine enum.
//
//nolint:gochecknoglobals // a vocabulary table, read-only after init.
var deliveryModes = map[string]string{
	"at-least-once": "DeliveryAtLeastOnce",
	"at-most-once":  "DeliveryAtMostOnce",
	"exactly-once":  "DeliveryExactlyOnce",
}

// generatorFieldOf fills a pool field: the run's shared pools, or a
// law-declared one for a domain the sequences never draw.
func generatorFieldOf(
	b *Bindings, harness *suite.Contract, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *suite.Method,
) (*LawField, string) {
	switch f.From {
	case poolKeys:
		if !b.UsesKeys() {
			return nil, f.Name + " draws from the " + f.From + " pool, which no action here declares"
		}
		field.Pool = f.From
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Keys")
		return field, ""
	case poolValues:
		if !b.UsesValues() {
			return nil, f.Name + " draws from the " + f.From + " pool, which no action here declares"
		}
		field.Pool = f.From
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Values")
		return field, ""
	case poolInputs:
		elem, q, why := lawInputElem(b, harness, r, m, keyed)
		if why != "" {
			return nil, f.Name + " " + why
		}
		if reason := b.addLawPool(LawPool{Name: poolInputs, Q: q, Elem: elem}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolInputs
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""
	case poolPayloads:
		if reason := b.addLawPool(LawPool{
			Name: poolPayloads, Q: builtinString, Elem: sdk.Builtin(builtinString), Adversarial: true,
		}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolPayloads
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""
	case "messages":
		// The publish role is the writer feeding the values pool, so the
		// messages a law publishes are the values the sequences publish —
		// one pool, colliding by construction.
		if !b.UsesValues() {
			return nil, f.Name + " draws from the values pool, which no action here declares"
		}
		field.Pool = poolValues
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Values")
		return field, ""

	case poolOffsets:
		// Bounded durations rather than arbitrary ones: an offset past the
		// advance horizon never fires inside the law's own window, and a
		// negative one is a schedule for the past nothing promises.
		elem, err := golang.RefForQualified("time.Duration", b.IfaceName)
		if err != nil {
			return nil, f.Name + " spells time.Duration, which no ref composes: " + err.Error()
		}
		if reason := b.addLawPool(LawPool{
			Name: poolOffsets, Q: "time.Duration", Elem: elem, Offsets: true,
		}); reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Pool = poolOffsets
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Pool")
		return field, ""
	}
	return nil, f.Name + " draws from the " + f.From + " pool, which this build does not compose"
}

// lawInputElem is the element type of a law's wide input pool: the first
// role field's own input, which is the domain the stateless claim ranges
// over.
func lawInputElem(
	b *Bindings, harness *suite.Contract, r tiers.Rule, m, keyed *suite.Method,
) (sdk.Ref, string, string) {
	for _, f := range r.Fields {
		if f.Kind != tiers.KindRole {
			continue
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" || len(role.CallArgs()) == 0 {
			continue
		}
		arg := role.CallArgs()[0]
		return arg.Type, shape.QName(arg.Source), ""
	}
	return nil, "", "draws a domain no role here states"
}

// addLawPool registers a law-declared pool, refusing a second element type
// under one name.
func (b *Bindings) addLawPool(p LawPool) string {
	for _, held := range b.LawPools {
		if held.Name == p.Name {
			if held.Q == p.Q {
				return ""
			}
			return "draws " + p.Q + " from the " + p.Name + " pool, which already draws " + held.Q
		}
	}
	b.LawPools = append(b.LawPools, p)
	return ""
}

// handleFieldOf fills a handle the generated file constructs and shares.
func handleFieldOf(
	b *Bindings, harness *suite.Contract, r tiers.Rule, f tiers.Field,
	field *LawField, m, keyed *suite.Method,
) (*LawField, string) {
	switch f.From {
	case "key-projection":
		if b.Reference.KeyField == "" {
			return nil, f.Name + " needs the key projection, which was not derivable here"
		}
		field.KeyOfName = b.KeyOfName()
		field.KindName = sdk.Kind(LawFieldKindPrefix + "KeyOf")
		return field, ""

	case "identity-hash":
		// Identity over the drained element: the hash argument is the value
		// itself, so the closure needs only the element's type.
		if elem, why := hashElem(b, harness, r, m, keyed); why == "" {
			field.Value = elem
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Hash")
		return field, ""

	case "subject-factory":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Factory")
		return field, ""

	case handleClassifier:
		spec, why := sessionSpecOf(b, harness, r, m, keyed)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Classify")
		field.KeyOfName = spec.ClassifyName
		return field, ""

	case "natural-order":
		role, reason := roleMethod(b, harness, fromSelf, m, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		if !orderedScalar(role) {
			return nil, f.Name + " orders " + role.Name + "'s result, which the language does not"
		}
		out, _, why := resultType(role)
		if why != "" {
			return nil, f.Name + " " + why
		}
		field.Out = out
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Less")
		return field, ""

	case "observation":
		obs, reason := observationOf(b, harness, keyed)
		if reason != "" {
			return nil, f.Name + " " + reason
		}
		field.Method = obs.Method.Name
		field.TakesCtx = obs.TakesCtx
		field.Out = obs.Out
		if obs.Keyed {
			field.KeyField = b.Keys.Field
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ObserveKeyed")
			b.LawsUseFixture = true
		} else {
			field.KindName = sdk.Kind(LawFieldKindPrefix + "ObserveCall")
		}
		return field, ""

	case "partitions":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Partitions")
		return field, ""

	case "clock":
		field.KindName = sdk.Kind(LawFieldKindPrefix + "Advance")
		return field, ""
	case "history":
		return nil, f.Name + " waits on an append-recording history hook the runner does not offer"
	}
	return nil, f.Name + " needs the " + f.From + " handle, which this build does not construct"
}

// hashElem resolves the identity hash's element: the drained element of the
// same rule's Drain field where one exists, the values pool otherwise.
func hashElem(
	b *Bindings, harness *suite.Contract, r tiers.Rule, m, keyed *suite.Method,
) (sdk.Ref, string) {
	for _, f := range r.Fields {
		if f.Kind != tiers.KindRole || (f.Name != fDrain && f.Name != "Collect") {
			continue
		}
		role, reason := roleMethod(b, harness, f.From, m, keyed)
		if reason != "" {
			return nil, reason
		}
		return drainedElem(b, role)
	}
	if b.Values.Type != nil {
		return b.Values.Type, ""
	}
	return nil, "hashes a value type no method here draws"
}

// keyedReadMismatch holds a keyed-read role to the shape its template spells:
// `(ctx, K) (V, error)` at the pools' own types, so a role of another shape —
// or of the right shape over other types — renders a closure that fails to
// compile in whichever package arms it.
func keyedReadMismatch(b *Bindings, fieldName string, role *suite.Method, strictValue bool) string {
	keyQ, _ := b.keyQOf(role)
	valueQ, _ := b.valueQOf(role)
	if pseudoShape(role) != shapeReader {
		return fieldName + " closes over " + role.Name + ", whose shape is " +
			pseudoShape(role) + " rather than a keyed reader"
	}
	// The value half is held to the pool only where the law's own row draws
	// it: a windowed count reads int beside string pools, lawfully, because
	// nothing compares its answer to a drawn value.
	if (b.Keys.Q != "" && keyQ != b.Keys.Q) ||
		(strictValue && b.Values.Q != "" && valueQ != b.Values.Q) {

		return fieldName + " closes over " + role.Name + ", which reads (" + keyQ +
			" → " + valueQ + ") beside pools of (" + b.Keys.Q + ", " + b.Values.Q + ")"
	}
	return ""
}

// errOnly reports whether the method returns exactly one error and nothing
// else.
func errOnly(m *suite.Method) bool {
	return len(m.Returns) == 1 && m.Returns[0].Source != nil &&
		golang.IsError(m.Returns[0].Source)
}

// stringParam reports whether the parameter is a bare string.
func stringParam(p golang.Param) bool {
	return p.Source != nil && golang.IsBuiltinNamed(p.Source, builtinString)
}

// integerResult reports whether the return slot is a builtin integer — the
// shape an offset or a conserved sum totals.
func integerResult(ret *golang.Return) bool {
	if ret.Source == nil {
		return false
	}
	for _, name := range builtinInts {
		if golang.IsBuiltinNamed(ret.Source, name) {
			return true
		}
	}
	return false
}

// builtinInts are the signed integer builtins a sum or an offset totals;
// builtinOrdered is everything `<` orders.
//
//nolint:gochecknoglobals // vocabulary tables, read-only after init.
var (
	builtinInts    = []string{builtinInt, "int8", "int16", "int32", builtin64}
	builtinOrdered = append(append([]string{}, builtinInts...),
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", builtinString)
)

// identityCompared reports whether the method's first result is a live
// handle — a channel, a function, a pointer — that `!=` compares by identity,
// which two independently built sides never share.
func identityCompared(m *suite.Method) bool {
	if len(m.Returns) == 0 || m.Returns[0].Source == nil {
		return false
	}
	src := m.Returns[0].Source
	return golang.IsChannel(src) || src.IsFunc() || shape.GoPointerElem(src) != nil
}

// orderedScalar reports whether the method's single result is a type `<`
// orders — the builtin integers, floats and string.
func orderedScalar(m *suite.Method) bool {
	_, ret, why := resultType(m)
	if why != "" || ret.Source == nil {
		return false
	}
	for _, name := range builtinOrdered {
		if golang.IsBuiltinNamed(ret.Source, name) {
			return true
		}
	}
	return false
}

// transitionPairs parses a workflow's `from>to[,from>to…]` stamp.
func transitionPairs(value string) ([][2]string, string) {
	var out [][2]string
	for part := range strings.SplitSeq(value, ",") {
		from, to, ok := strings.Cut(strings.TrimSpace(part), ">")
		if !ok || from == "" || to == "" {
			return nil, "reads " + value + ", which is not a from>to transition list"
		}
		out = append(out, [2]string{from, to})
	}
	return out, ""
}

// sessionSpecOf derives the per-client classification the session laws
// share — memoized on the bindings, because the classifier is one file-level
// function however many laws read through it.
//
// The write-ordering laws need the version the store assigned to each write,
// and a writer answering only an error hides it: the trace records what was
// sent, never what was stamped. Those laws bind only beside an
// upserter-shaped write — (ctx, V) (V, error), the stored state answered —
// and refuse otherwise with the shape that would carry them.
func sessionSpecOf(
	b *Bindings, harness *suite.Contract, r tiers.Rule, m, keyed *suite.Method,
) (*SessionSpec, string) {
	if keyed == nil {
		return nil, "orders reads no keyed reader here answers"
	}
	mixin := ""
	if len(r.Needs) > 0 {
		mixin = r.Needs[0]
	}
	version, given := shape.MixinParamKey(mixin, "version").Get(m.Source.Meta())
	if !given || version == "" {
		return nil, "names no version= member, and a session guarantee is defined against the value's ordering stamp"
	}
	if r.Law != lawid.MonotonicReads && answeringWriterOf(harness) == nil {
		return nil, "orders writes the trace cannot see — the writer answers only an error, " +
			"and the version the store assigned dies with the call; an answering " +
			"write (ctx, V) (V, error) surfaces it"
	}
	if b.Session != nil {
		return b.Session, ""
	}
	if b.sessionKeyField == "" {
		return nil, "keys per client on a value member no convention names"
	}
	value, _, why := resultType(keyed)
	if why != "" {
		return nil, why
	}
	if b.Keys.Type == nil {
		return nil, "instantiates at a key type no method here draws"
	}
	spec := &SessionSpec{
		ClassifyName: strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "SessionClassify",
		Reader:       keyed.Name,
		Value:        value,
		KeyField:     b.sessionKeyField,
		VersionField: version,
		Key:          b.Keys.Type,
	}
	if up := answeringWriterOf(harness); up != nil {
		spec.Writer = up.Name
	}
	b.Session = spec
	return spec, ""
}

// drainFieldOf derives the subscription sweep where the subscribe role
// answers a channel, or refuses with the option that would serve instead.
//
// The derived form is the synchronous floor: everything a subscriber is
// owed is in its channel by the time Publish returns, and the sweep reads
// what is there without blocking. An asynchronous publisher supplies its
// own drain through the generated option, which outranks this derivation —
// the property prefers the config's closure and falls back to the sweep.
func drainFieldOf(
	b *Bindings, harness *suite.Contract, f tiers.Field,
	field *LawField, m, keyed *suite.Method,
) (*LawField, string) {
	role, reason := roleMethod(b, harness, "publisher.subscribe", m, keyed)
	if reason != "" {
		return nil, f.Name + " " + reason
	}
	out, ret, why := resultType(role)
	if why != "" {
		return nil, f.Name + " " + why
	}
	isChan := false
	if ret.Source != nil {
		isChan, _ = golang.MetaIsChannel.Get(ret.Source.Meta())
	}
	if !isChan {
		return nil, f.Name + " waits on the " + f.From + " option — the subscription " +
			"answers no channel this sweep can read"
	}
	if b.Publisher == nil {
		msgQ, stamped := golang.MetaChanElem.Get(ret.Source.Meta())
		if !stamped || msgQ == "" {
			return nil, f.Name + " drains a channel whose element no stamp names"
		}
		msg, err := golang.RefForQualified(b.substQ(msgQ), b.IfaceName)
		if err != nil {
			return nil, f.Name + " drains " + msgQ + ", which no closure can spell: " + err.Error()
		}
		b.Publisher = &PublisherSpec{
			DrainName: strings.ToLower(b.IfaceName[:1]) + b.IfaceName[1:] + "DrainSubscription",
			Sub:       out,
			Msg:       msg,
		}
	}
	field.KindName = sdk.Kind(LawFieldKindPrefix + "DrainSub")
	field.KeyOfName = "drainSub"
	return field, ""
}

// answeringWriterOf finds a write that answers the stored state — one value
// in, the same type out beside the error — or nil. Structural rather than
// stamped, so a hand-built projection in a test answers the same way the
// annotated corpus does.
func answeringWriterOf(harness *suite.Contract) *suite.Method {
	if harness == nil {
		return nil
	}
	for i := range harness.Methods {
		m := &harness.Methods[i]
		args := m.CallArgs()
		if len(args) != 1 || len(m.Returns) != 2 || !m.ReturnsError() {
			continue
		}
		in, inOK := args[0].Source, args[0].Source != nil
		out, outOK := m.Returns[0].Source, m.Returns[0].Source != nil
		if inOK && outOK && shape.QName(in) != "" && shape.QName(in) == shape.QName(out) {
			return m
		}
	}
	return nil
}

// roleMethod resolves a manifest role to the method whose call fills it:
// the selecting method itself, a shape family, or a partner the selecting
// method's own stamp names.
func roleMethod(
	b *Bindings,
	harness *suite.Contract,
	from string,
	m, keyed *suite.Method,
) (*suite.Method, string) {
	switch from {
	case fromSelf:
		return m, ""
	case "family.reader":
		if keyed == nil {
			return nil, "names the reader family, and the interface has no keyed reader"
		}
		return keyed, ""
	case "family.writer":
		if harness != nil {
			var fallback *suite.Method
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if pseudoShape(candidate) != shapeWriter {
					continue
				}
				// The family's writer is the values pool's own feeder: a
				// peer-merging method is writer-shaped too, and a law fed by
				// one writes values no pool ever draws.
				if q, _ := b.valueQOf(candidate); q == b.Values.Q {
					return candidate, ""
				}
				if fallback == nil {
					fallback = candidate
				}
			}
			if b.Values.Q == "" && fallback != nil {
				return fallback, ""
			}
		}
		return nil, "names the writer family, and the interface has no value writer feeding the pool"
	case "family.aggregator":
		if harness != nil {
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if pseudoShape(candidate) == shapeAggregator && len(candidate.CallArgs()) == 0 {
					return candidate, ""
				}
			}
		}
		return nil, "names the aggregator family, and the interface has no aggregate"
	case "family.cell":
		if harness != nil {
			for i := range harness.Methods {
				candidate := &harness.Methods[i]
				if candidate.Name == m.Name || len(candidate.CallArgs()) > 0 {
					continue
				}
				if _, _, why := resultType(candidate); why == "" {
					return candidate, ""
				}
			}
		}
		return nil, "names the cell family, and the interface has no nullary read"
	}
	if owner, param, ok := strings.Cut(from, "."); ok && !strings.HasPrefix(from, "family.") {
		// A mixin's sibling parameter first — `deleteremoves.read` names the
		// method its stamp points at.
		v, stamped := shape.MixinParamKey(owner, param).Get(m.Source.Meta())
		if stamped && v != "" {
			role := methodOf(harness, golang.LocalName(v))
			if role == nil {
				return nil, "names " + from + " = " + v + ", which is not a method of " + b.IfaceName
			}
			return role, ""
		}
		// Then a contract role: the selecting method fills it itself, or its
		// directive's partner key names the sibling that does.
		if held, ownRole := shape.ContractRoleKey(owner).Get(m.Source.Meta()); ownRole && held == param {
			return m, ""
		}
		partner, named := shape.ContractPartnerKey(owner, param).Get(m.Source.Meta())
		if named && partner != "" {
			role := methodOf(harness, golang.LocalName(partner))
			if role == nil {
				return nil, "names " + from + " = " + partner + ", which is not a method of " + b.IfaceName
			}
			return role, ""
		}
		return nil, "names " + from + ", which the selecting method does not stamp"
	}
	return nil, "names " + from + ", which nothing resolves"
}

// returnsSlice reports whether the method's first result is a slice.
func returnsSlice(m *suite.Method) bool {
	return len(m.Returns) > 0 && m.Returns[0].Source != nil &&
		shape.GoSliceElem(m.Returns[0].Source) != nil
}

// stampValue reads one classification parameter, by the raw key the manifest
// spells — off the selecting method first, and for a contract parameter off
// every carrier of the same contract, because the stamp lives on the
// directive host and any role method may be the one selecting the rule.
func stampValue(harness *suite.Contract, m *suite.Method, key string) (string, bool) {
	if v, ok := sdk.EnsureKey(key, sdk.StringParser).Get(m.Source.Meta()); ok && v != "" {
		return v, true
	}
	contract, isContract := contractOfParamKey(key)
	if !isContract || harness == nil {
		return "", false
	}
	for i := range harness.Methods {
		carrier := &harness.Methods[i]
		if !slices.Contains(carrier.Contracts, contract) {
			continue
		}
		if v, ok := sdk.EnsureKey(key, sdk.StringParser).Get(carrier.Source.Meta()); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// contractOfParamKey extracts the contract a param stamp key belongs to,
// false for a mixin's.
func contractOfParamKey(key string) (string, bool) {
	rest, found := strings.CutPrefix(key, "shape.contract.")
	if !found {
		return "", false
	}
	name, _, ok := strings.Cut(rest, ".param.")
	return name, ok && name != ""
}

// splitQualified splits a resolver-qualified name into its package path and
// trailing identifier.
func splitQualified(v string) (pkg, name string, ok bool) {
	i := strings.LastIndexByte(v, '.')
	if i <= 0 || i == len(v)-1 {
		return "", "", false
	}
	return v[:i], v[i+1:], true
}

// classificationsOf is the method's whole set, in one namespace: its detector
// shape, its mixins, and the contracts it fills a role in.
func classificationsOf(m *suite.Method) []string {
	out := []string{}
	if s := shape.Get(m.Source.Meta()); s != "" {
		out = append(out, s)
	}
	out = append(out, m.Mixins...)
	return append(out, m.Contracts...)
}

// paramsOf collects the classification parameters the When clauses condition
// on, keyed the way [tiers.Condition.Param] spells them — the mixin params off
// the method, the contract params off every carrier of the same contract,
// because a contract's parameter speaks for the protocol and lives on the
// directive host: the codec's fidelity is stamped on the forward role, and a
// rule selected from the inverse conditions on it all the same.
func paramsOf(harness *suite.Contract, m *suite.Method) map[string]string {
	out := map[string]string{}
	for _, name := range m.Mixins {
		for _, p := range mixinParamNames(name) {
			if v, ok := shape.MixinParamKey(name, p).Get(m.Source.Meta()); ok {
				out[shape.MixinParamKey(name, p).Name()] = v
			}
		}
	}
	for _, name := range m.Contracts {
		for _, p := range contractParamNames(name) {
			for i := range harness.Methods {
				carrier := &harness.Methods[i]
				if !slices.Contains(carrier.Contracts, name) {
					continue
				}
				v, held := shape.ContractParamKey(name, p).Get(carrier.Source.Meta())
				if held && v != "" {
					out[shape.ContractParamKey(name, p).Name()] = v
				}
			}
		}
	}
	return out
}

// mixinParamNames returns the named mixin's declared parameters — the
// registry's fact, like the sibling scan in model.go.
func mixinParamNames(name string) []string {
	for _, mx := range mixins.All() {
		if mx.Name == name {
			return mx.Params
		}
	}
	return nil
}

// contractParamNames returns the named contract's declared parameters.
func contractParamNames(name string) []string {
	for _, c := range contracts.All() {
		if c.Name == name {
			return c.Params
		}
	}
	return nil
}
