// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifabsent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifmatch"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/outbox"
	"go.thesmos.sh/eidos/sdk"
)

// The emit kinds for the contract-derived checks.
//
// Three, where eidos registers many more, and the difference is not unwritten
// work. Most name a property [engine/model/law] implements, which makes them
// the model tier's under docs/adr/0018; the rest need a failing call, a second
// subject or a controlled clock, none of which a fixed sequence against one
// subject can produce.
//
// The counts and the per-contract reasons used to be written out here. They
// went stale twice — the contract axis grew upstream and this comment did not,
// and a reader comparing "twenty-four" against the registry found two more.
// `conformance/gate.UnevidencedClassifications` holds the reasons now, one row
// per classification, and a census fails the build when a row stops being
// true. A comment cannot do that, which is the whole argument for moving them.
const (
	KindIfAbsent sdk.Kind = "suite.check.ifabsent"
	KindIfMatch  sdk.Kind = "suite.check.ifmatch"
	KindOutbox   sdk.Kind = "suite.check.outbox"
)

// The contracts this generator acts on, with the role each check selects by.
//
// The names come from eidos's own packages. `if-match` exports its roles as
// constants too, so those are taken rather than spelled; the rest publish their
// vocabulary as a slice, and indexing one would bind this to a declaration
// order nothing promises. [TestContractVocabularyIsUpstream] holds every
// literal to the registry, so a rename upstream fails a test here rather than
// silently selecting nothing.
const (
	ContractIfAbsent     = ifabsent.Name
	ContractIfAbsentRole = "writer"

	ContractIfMatch      = ifmatch.Name
	ContractIfMatchRole  = ifmatch.RoleWriter
	ContractIfMatchMatch = ifmatch.RoleMatch

	ContractOutbox        = outbox.Name
	ContractOutboxRole    = "append"
	ContractOutboxPartner = "subscribe"
)

// contractDataOf reads the role and partner stamps of every contract this
// generator acts on.
//
// Pulled once into maps for the reason [mixinParamsOf] gives: a check is
// selected here and rendered later, by which time the source node is out of
// scope, and two derivations of one stamp are two chances to disagree about
// what the run classified.
//
// Enumerated rather than discovered. eidos exposes no "every stamp under this
// contract" accessor — the key constructors compose one key from a pair — so
// the list is the set of checks rather than an inventory of the registry.
func contractDataOf(bag *sdk.Bag) (roles, partners map[string]string) {
	set := func(dst *map[string]string, key, value string, found bool) {
		if !found {
			return
		}
		if *dst == nil {
			*dst = map[string]string{}
		}
		(*dst)[key] = value
	}

	for _, name := range shape.Contracts(bag) {
		role, found := shape.ContractRoleKey(name).Get(bag)
		set(&roles, name, role, found)
	}

	wantedPartners := [...]struct{ contract, role string }{
		{ContractOutbox, ContractOutboxPartner},
		{ContractIfMatch, ContractIfMatchMatch},
	}
	for _, w := range wantedPartners {
		v, found := shape.ContractPartnerKey(w.contract, w.role).Get(bag)
		set(&partners, w.contract+"."+w.role, v, found)
	}

	return roles, partners
}

// HasContractRole reports whether the annotator stamped this method as filling
// the named role of the named contract.
//
// Both halves, because a contract is a protocol rather than a property: an
// `outbox` check written against the subscriber would call the wrong method,
// and the role is the only thing that distinguishes the two members.
func (m Method) HasContractRole(contract, role string) bool {
	return m.contractRoles[contract] == role
}

// ContractPartner returns the local identifier a contract's role-keyed partner
// names, empty where the directive named none.
//
// Local, for the reason [Method.MixinPartner] gives: the resolver rewrites a
// partner into a qualified name so it is unambiguous across packages, and a
// generated call is on a subject the check already holds.
func (m Method) ContractPartner(contract, role string) string {
	return golang.LocalName(m.contractPartners[contract+"."+role])
}

// contractChecks selects the family a method owes for the contract roles
// attached to it.
//
// Each of the three is derivable from the role plus the signature, which is
// what excludes the rest. The exclusions are stated on [KindIfAbsent] rather
// than discovered here, because a reader asking "why is there no `tx` check"
// is asking about the catalogue and not about this function.
func contractChecks(
	c *sdk.Provenance, iface *sdk.Interface, f Fixture, m Method, methods []Method,
) ([]*Check, []decline) {
	base := checkFor(c, iface, m)

	var (
		out      []*Check
		declined []decline
	)
	if ck, ok := ifAbsentCheck(f, m, base); ok {
		out = append(out, ck)
	}
	ck, why := ifMatchCheck(f, m, methods, base)
	out, declined = keep(out, declined, ck, ContractIfMatch, why)
	ck, why = outboxCheck(f, m, methods, base)
	out, declined = keep(out, declined, ck, ContractOutbox, why)
	return out, declined
}

// ifAbsentCheck builds "a second write for one key is refused".
//
// The alternate value rather than the sample, because the harness seeds each
// fresh subject through whatever the annotator classified `writer` — which for
// this contract is the very method under check. Handed the seeded value the
// first write would already be the second, and a correct subject would fail on
// the line the check expects to succeed.
//
// ReturnsError because the whole claim is that the second call *reports* the
// conflict, and a method with nowhere to report one cannot make it.
func ifAbsentCheck(f Fixture, m Method, base checkBuilder) (*Check, bool) {
	if !m.HasContractRole(ContractIfAbsent, ContractIfAbsentRole) || !m.HasInput() || !m.ReturnsError() {
		return nil, false
	}
	ck := base(KindIfAbsent, ContractIfAbsent, "RefusesADuplicate", fixtureArgs(f, m, true))
	ck.NeedsDerivedInput = true
	ck.Sentinel = stampedSentinel(m, shape.ContractParamKey(ContractIfAbsent, "conflict"))
	return ck, true
}

// ifMatchCheck builds "the write agrees with the predicate".
//
// Agreement rather than refusal, and deliberately. "A non-matching value is
// refused" needs a value the subject's predicate rejects, and nothing in the
// directive or the signature says which one that is — a derived value the
// predicate happens to admit would turn the check into a demand that a correct
// implementation fail. What both halves of the pair can always be asked is
// whether they agree, and a subject whose writer admits what its predicate
// rejects fails that.
//
// # Which spelling of the predicate this reads
//
// `match=Match` names a callable and is resolved like any other partner —
// qualified, reported when it names nothing in scope, back-stamped onto the
// predicate. `pred=` is the other spelling and carries an expression
// (`pred=Version==Expected`), which the resolver deliberately leaves verbatim
// because there is no callable in it to look up.
//
// This reads the role. A check has to *call* the predicate, and an expression
// is not something a generated call site can spell — so the param form is a
// declaration the model tier can act on and this one cannot.
func ifMatchCheck(f Fixture, m Method, methods []Method, base checkBuilder) (*Check, string) {
	if !m.HasContractRole(ContractIfMatch, ContractIfMatchRole) {
		return nil, ""
	}
	p := methodNamed(methods, m.ContractPartner(ContractIfMatch, ContractIfMatchMatch))
	if p == nil || !m.ReturnsError() || !predicateOver(m, *p) {
		return nil, ""
	}
	args, why := partnerArgs(f, m, *p)
	if why != "" {
		return nil, why
	}

	ck := base(KindIfMatch, ContractIfMatch, "AgreesWith"+p.Name, fixtureArgs(f, m, false))
	ck.Partner, ck.PartnerArgs = p, args
	return ck, ""
}

// predicateOver reports whether p answers a yes-or-no question about the very
// value m takes.
//
// The parameter lists have to match, not merely be compatible: a predicate over
// something else is a method the directive happened to name, and calling it
// with m's arguments would not compile. One bool and nothing else beside the
// error, because two value returns leave which one is the verdict a guess.
func predicateOver(m, p Method) bool {
	args, over := m.CallArgs(), p.CallArgs()
	if len(args) != len(over) {
		return false
	}
	for i := range args {
		if !args[i].Source.Equal(over[i].Source) {
			return false
		}
	}
	values := p.ValueReturns()
	return len(values) == 1 && golang.QName(values[0].Source) == "bool"
}

// outboxCheck builds "a record appended before anyone subscribed is still
// delivered".
//
// Which is the whole of what distinguishes an outbox from the `publisher`
// contract carrying the same two methods: a publisher may drop what nobody was
// listening for, and an outbox holds it until somebody is. So the check appends
// first and subscribes second, and a subject delivering only live traffic fails
// it.
//
// Two records rather than one. The harness seeds each fresh subject through the
// append role, so a subject that accepted the check's own record and dropped it
// would still deliver the seed's and pass. Demanding two arrive is what makes
// the seed's copy insufficient — and it holds for a consumer who replaced the
// seed with one that appends nothing, since the check writes both itself.
func outboxCheck(f Fixture, m Method, methods []Method, base checkBuilder) (*Check, string) {
	if !m.HasContractRole(ContractOutbox, ContractOutboxRole) {
		return nil, ""
	}
	p := methodNamed(methods, m.ContractPartner(ContractOutbox, ContractOutboxPartner))
	if p == nil || !deliversOver(m, *p) {
		return nil, ""
	}
	args, why := partnerArgs(f, m, *p)
	if why != "" {
		return nil, why
	}

	ck := base(KindOutbox, ContractOutbox, "ReachesASubscriber", fixtureArgs(f, m, false))
	ck.Partner, ck.PartnerArgs = p, args
	ck.NeedsClock = true
	return ck, ""
}

// deliversOver reports whether p hands back a stream of the very thing m
// appends.
//
// The element type is what makes the comparison at the end of the check
// possible at all: a stream of something else is a method the directive
// happened to name, and holding a received value up to the appended one would
// not compile. A send-only channel is excluded because the check receives from
// it.
//
// Both halves must report failure. The claim is that a record the subject
// *accepted* is delivered, so an append that cannot say it refused the record
// leaves the wait unable to tell "dropped" from "never taken".
func deliversOver(m, p Method) bool {
	args := m.CallArgs()
	if len(args) != 1 || !m.ReturnsError() || !p.ReturnsError() {
		return false
	}
	values := p.ValueReturns()
	if len(values) != 1 {
		return false
	}
	stream := values[0].Source
	if !golang.IsChannel(stream) || golang.ChanDir(stream) == golang.ChanSend {
		return false
	}
	elem := golang.ChanElem(stream)
	return elem != nil && elem.Equal(args[0].Source)
}

// methodNamed returns the method of that name in the resolved set, or nil.
//
// Against the resolved set so an inherited partner is found: a contract member
// may arrive through an embed, and a conformance check can only call what the
// subject declares.
func methodNamed(methods []Method, name string) *Method {
	if name == "" {
		return nil
	}
	for i := range methods {
		if methods[i].Name == name {
			return &methods[i]
		}
	}
	return nil
}
