// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"
	"strconv"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

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

	// Proves names the wear kinds whose defect class this law's identifier
	// claims to catch, sorted. Empty where the wardrobe produces none of
	// them.
	//
	// The criterion this prover was missing. Its question was whether *some*
	// defect made the law fail, which every law answers yes to — a Put that
	// does nothing breaks every claim about what a Put leaves behind, and
	// `AUTO-CAS-ATOMIC-ONE-WINNER` was green on exactly that, a law about two
	// writers contending proved by a subject with no writer at all.
	//
	// So the loop skips a wear this law's name never mentions, and a law with
	// none skips outright with the reason. That second answer is the one the
	// old criterion could not give: "the wardrobe cannot produce the defect
	// this law is named for" is a gap in the wardrobe, and it is worth telling
	// apart from a law that survived defects it should have caught.
	Proves []string

	// Unreached says why a wear of this law's own class never reaches it,
	// empty where one does.
	//
	// The third verdict. [SatLaw.Proves] empty means the wardrobe has no
	// defect of this law's class at all; this means it has one and the
	// dressing cannot get to the check — a trace law reading recorded
	// operations, a bound on redelivery a within-drain duplicate cannot
	// reach. Both are gaps in the wardrobe and they want different wears, so
	// the prover says which.
	Unreached string
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

	// Member names the field a wear rewrites inside the value it answers,
	// rather than replacing the value whole.
	//
	// A session guarantee is stated against an ordering stamp the subject
	// assigns, and the trace classifier reads the key off the same value. A
	// wear that answers a zero therefore changes the key too, and the law
	// files the two reads under different keys and compares neither — which
	// is how a version-ordering claim survived every defect that replaced
	// its subject's answer outright.
	Member string

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

// SeqDefect names the runtime helper a stream-shaped defect wears — the
// faded drain or the doubled one, at the wear's own arity.
func (m SatMutant) SeqDefect() string {
	base := "Faded"
	switch m.Kind {
	case "dupseq":
		base = "Doubled"
	case "flood":
		base = "Flooded"
	}
	if m.Seq == 2 {
		return base + "Seq2"
	}
	return base + "Seq"
}

// saturationOf derives the saturation surface from what lawsOf bound: per
// law, the methods its closures reach; per reached method, the defects the
// prover can wear. Clocked laws stay out — the clocked factory builds its
// own subjects, so a worn defect never reaches the run — and a witnessed
// interface emits no prover at all, because its wrappers would need the
// witness instantiation the surface does not thread.
func saturationOf(b *Bindings, harness *suite.Contract) {
	if len(b.Witnesses) > 0 {
		return
	}
	worn := map[string]bool{}
	for _, lb := range b.Laws {
		sl := SatLaw{ID: lb.ID, Guards: lb.Supplied, Clocked: lb.Clocked}
		mset := map[string]bool{}
		for _, f := range lb.Fields {
			if f.Method == "" || mset[f.Method] {
				continue
			}
			mset[f.Method] = true
			sl.Methods = append(sl.Methods, f.Method)
		}
		if lb.Session && b.Session != nil {
			// A trace law closes over no method directly; what a defect can
			// wear is the reader and writer whose calls the trace records.
			for _, name := range []string{b.Session.Reader, b.Session.Writer} {
				if name != "" && !mset[name] {
					mset[name] = true
					sl.Methods = append(sl.Methods, name)
				}
			}
		}
		slices.Sort(sl.Methods)
		sl.Unwearable = len(sl.Methods) == 0
		sl.AcceptSemantic = lb.ID == lawid.CountEqualsReference
		for _, kind := range Wardrobe() {
			if Proves(kind, lb.ID) {
				sl.Proves = append(sl.Proves, kind)
			}
		}
		sl.Unreached = Unreached(lb.ID)
		b.SatLaws = append(b.SatLaws, sl)
		for _, name := range sl.Methods {
			if !worn[name] {
				worn[name] = true
				if m := methodOf(harness, name); m != nil {
					b.SatMutants = append(b.SatMutants, satMutantsOf(b, m)...)
				}
			}
			// The boundary wear is the law's, not the method's: two laws over
			// one method cross different lines, and the shared wardrobe keyed
			// by method alone would keep only the first.
			if over, ok := overshootOf(b, harness, lb, name); ok && !worn[name+"\x00"+kindOvershoot] {
				worn[name+"\x00"+kindOvershoot] = true
				b.SatMutants = append(b.SatMutants, over)
			}
		}
	}
}

// overshootOf spells the defect a bound law's own declaration defines: an
// answer one past the line the stamp drew.
//
// The manifest marks which of a law's fields come from a stamp, and a numeric
// one is a boundary — `bounded limit=5` fills Max with 5, so 6 is the
// smallest answer that leaves the range. Nothing about the method's shape
// carries that number, which is why the generic wardrobe cannot violate a
// bound: zeros, alternations and waning counts all answer *inside* it.
//
// Only the upper bound. A lower one is optional in the manifest and defaults
// to the floor of the counting shapes it attaches to, so crossing it means
// answering a negative count — which the shapes cannot express.
func overshootOf(b *Bindings, harness *suite.Contract, lb *LawBinding, method string) (SatMutant, bool) {
	var bound, read *LawField
	for _, f := range lb.Fields {
		switch {
		case f.Name == fieldMax && f.Lit != "":
			bound = f
		case f.Method == method:
			read = f
		}
	}
	if bound == nil || read == nil {
		return SatMutant{}, false
	}
	n, err := strconv.Atoi(bound.Lit)
	if err != nil {
		// A fractional bound crossed by one is a different arithmetic, and
		// no counting shape answers a fraction.
		return SatMutant{}, false
	}
	m := methodOf(harness, method)
	if m == nil {
		return SatMutant{}, false
	}
	over := SatMutant{
		Method:   method,
		Kind:     kindOvershoot,
		TakesCtx: m.TakesContext(),
		Over:     strconv.Itoa(n + 1),
		ViaLen:   read.Kind() == sdk.Kind(LawFieldKindPrefix+string(shapeScalarLen)),
	}
	for _, p := range m.Params {
		over.Params = append(over.Params, p.Type)
	}
	for i := range m.Returns {
		over.Returns = append(over.Returns, m.Returns[i].Type)
	}
	over.Last = len(over.Returns) - 1
	if over.ViaLen {
		elem, why := drainedElem(b, m)
		if why != "" {
			return SatMutant{}, false
		}
		over.Out = elem
		return over, true
	}
	ref, _, why := resultType(m)
	if why != "" {
		return SatMutant{}, false
	}
	over.Out = ref
	return over, true
}

// satMutantsOf spells the defects one method can wear: inert always, and —
// for a single-result reader beside its error — the fixture pair flapped
// where the result is the pool's own type, or a waning count where it is an
// integer. Wider vocabularies earn their kinds when a surviving law names
// the need.
func satMutantsOf(b *Bindings, m *subject.Method) []SatMutant {
	base := SatMutant{Method: m.Name, TakesCtx: m.TakesContext(), Seq: seqArity(m)}
	for _, p := range m.Params {
		base.Params = append(base.Params, p.Type)
	}
	for i := range m.Returns {
		base.Returns = append(base.Returns, m.Returns[i].Type)
	}
	base.Last = len(base.Returns) - 1
	inert := base
	inert.Kind = kindInert
	out := []SatMutant{inert}
	if len(m.Returns) > 0 {
		// The flickering defect: every second call answers zeros where the
		// subject would have answered.
		//
		// The wear the whole stability family needs, and the one the
		// wardrobe had no shape for. Cacheable, deterministic, consistent,
		// non-decreasing — every such claim is about two calls agreeing, and
		// `inert` satisfies all of them: zeros forever is perfectly stable.
		// What breaks a stability claim is an answer that *changes*, and the
		// smallest change available at any return type is the subject's own
		// answer alternating with its zero.
		flicker := base
		flicker.Kind = kindFlicker
		out = append(out, flicker)
	}
	if m.ReturnsError() {
		// The sputtering defect: alternating minted refusals, for the laws
		// about what an error must coincide with.
		sputter := base
		sputter.Kind = kindSputter
		out = append(out, sputter)
	}
	if errOnly(m) {
		// The sticking defect: the operation works once and refuses forever
		// after — the close that is not idempotent, the release that cannot
		// be repeated.
		//
		// sputter cannot express it. An idempotence law calls twice and
		// discards the first answer, so an alternating refusal is absorbed by
		// the very call the law is not reading, on every iteration, forever.
		// What the claim forbids is the *second* call failing, and only a
		// defect that arrives after the first can say so.
		stick := base
		stick.Kind = kindStick
		out = append(out, stick)

		// The latching defect: the first call takes effect and every later
		// one is quietly dropped.
		//
		// The order-sensitive fold, which is the one defect a commutativity
		// claim can see. That law builds both its instances from the same
		// worn factory and applies the same values in opposite orders, so
		// every wear reaches both sides identically and the two observations
		// agree however wrong they are. A fold that keeps only what it saw
		// first keeps a different thing on each side, and the claim fails as
		// itself.
		latch := base
		latch.Kind = kindLatch
		out = append(out, latch)
	}
	for i, p := range m.Params {
		// Nullary: a computation the method runs, not a callback it feeds.
		// Calling one that takes arguments would need values invented here,
		// and a defect that has to invent its own inputs is a different
		// subject under test.
		if p.Source == nil || !p.Source.IsFunc() || len(p.Source.FuncParams) > 0 {
			continue
		}
		// The greedy defect: the caller's callback run one extra time.
		//
		// A method taking a computation promises something about how often it
		// runs it — deduplicated, memoised, once. Nothing the wardrobe wore
		// could touch that: every other defect changes what the method
		// *answers*, and a coalescing claim counts what it *called*. The
		// counter the law reads lives inside the closure the law passed in,
		// so the only way to move it is to invoke that closure.
		greedy := base
		greedy.Kind = kindGreedy
		greedy.Member = strconv.Itoa(i)
		out = append(out, greedy)
		break
	}
	if b.Session != nil && b.Session.VersionField != "" && m.Name == b.Session.Reader {
		// The regressing defect: the subject's own answer with its ordering
		// stamp walked backwards, and every other field left alone.
		//
		// The session laws key their state on the value's own key member, so
		// a wear that answers a zero files each read under a different key
		// and the law compares nothing — which is why an ordering claim
		// survived inert and flicker alike. Rewriting one member is the only
		// defect that keeps the reads comparable and makes the order wrong.
		regress := base
		regress.Kind = kindRegress
		regress.Out = b.Session.Value
		regress.Member = b.Session.VersionField
		out = append(out, regress)
	}
	if base.Seq > 0 {
		// The stream defects, which the slice form has had all along: a drain
		// reversed and short one, and a drain that repeats. `fade` was dressed
		// on the shape that returns a slice and never on the shape that
		// streams, so a claim about order or duplicates had nothing worn on it
		// that could be false.
		faded := base
		faded.Kind = kindFadeSeq
		dup := base
		dup.Kind = kindDupSeq
		// The stream that will not end. Bounded at the completion law's own
		// limit, and it stops when the consumer stops: a wear that ignored
		// the yield's verdict would run until the machine gave out, which is
		// how this one was learned.
		flood := base
		flood.Kind = kindFlood
		out = append(out, faded, dup, flood)
	}
	if len(m.CallArgs()) == 1 && len(m.Returns) == 4 && m.ReturnsError() && returnsSlice(m) {
		// The echoing defect: the first page answered forever, with more
		// always promised — the walk that never advances.
		echo := base
		echo.Kind = kindEcho
		out = append(out, echo)
	}
	if len(m.Returns) != 2 || !m.ReturnsError() {
		return out
	}
	if returnsSlice(m) {
		elem, why := drainedElem(b, m)
		if why != "" {
			return out
		}
		// The fading defect: every second answer reversed and short one —
		// the replay that lies about the log it already showed.
		fade := base
		fade.Kind = kindFade
		fade.Out = elem
		out = append(out, fade)

		// The three single-drain defects, which alternation cannot express.
		//
		// A claim read off one drain — sorted, duplicate-free, terminating —
		// is answered by whichever call the law happens to make, and the
		// prover isolates one law at a time, so that is exactly one call per
		// step. An every-second defect then lands entirely on the parity the
		// law never sees, and the claim survives a wear built to break it.
		// These fire on every call for that reason.
		jumble := base
		jumble.Kind = kindJumble
		jumble.Out = elem
		dup := base
		dup.Kind = kindDupDrain
		dup.Out = elem
		flood := base
		flood.Kind = kindFlood
		flood.Out = elem
		out = append(out, jumble, dup, flood)
		return out
	}
	ref, ret, why := resultType(m)
	if why != "" {
		return out
	}
	// The spilling defect: a refusal carrying the answer the subject found —
	// "whatever the failed lookup left behind", which is the exact phrase the
	// default-on-error claim exists to forbid.
	//
	// Every refusal the wardrobe could mint until now arrived with the zero
	// value beside it, and the value a failed read is supposed to answer is
	// usually the zero. So the defect and the claim agreed, and a law about
	// what an error must coincide with had nothing that could disagree.
	spill := base
	spill.Kind = kindSpill
	spill.Out = ref
	out = append(out, spill)

	if args := m.CallArgs(); len(args) == 1 &&
		shape.QName(args[0].Source) == shape.QName(ret.Source) {
		// The passthrough defect: the input answered verbatim, where the
		// method's whole job was to change it.
		//
		// Every wear above answers something the subject did not — a zero, a
		// stale repeat, a wrong count. A claim about what a transformation
		// *removes* survives all of them, because a zero has had everything
		// removed and passes trivially. Escaping, sanitising, normalising:
		// the defect is always that the dangerous thing came through, and
		// the only way to spell that is to answer the argument.
		pass := base
		pass.Kind = kindPassthrough
		pass.Out = ref
		out = append(out, pass)
	}
	switch {
	case b.Values.Q != "" && shape.QName(ret.Source) == b.Values.Q &&
		b.Values.Field != "" && b.Values.OtherField != "":
		flap := base
		flap.Kind = kindFlap
		flap.Out = ref
		out = append(out, flap)
	case integerResult(ret):
		wane := base
		wane.Kind = kindWane
		wane.Out = ref
		out = append(out, wane)
		// And the rising twin: a count that only grows, for the laws about
		// what must return to rest.
		wax := base
		wax.Kind = kindWax
		wax.Out = ref
		out = append(out, wax)
	}
	return out
}

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
