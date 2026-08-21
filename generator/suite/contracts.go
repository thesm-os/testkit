// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/chain"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/cursor"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifabsent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifmatch"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/lease"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/outbox"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/pool"
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
	ContractIfAbsent         = ifabsent.Name
	ContractIfAbsentRole     = "writer"
	ContractIfAbsentConflict = ifabsent.ParamConflict

	ContractIfMatch      = ifmatch.Name
	ContractIfMatchRole  = ifmatch.RoleWriter
	ContractIfMatchMatch = ifmatch.RoleMatch

	ContractOutbox        = outbox.Name
	ContractOutboxRole    = "append"
	ContractOutboxPartner = "subscribe"

	// The cursor contract's open arm: the producing method's smoke
	// closes the handle it answers, and the laws deriver words the
	// cursor laws from the next and close partners. The sentinel is
	// read through the contract-param keys when the bindings land.
	ContractCursor      = cursor.Name
	ContractCursorOpen  = cursor.RoleOpen
	ContractCursorClose = cursor.ParamClose
	ContractCursorNext  = cursor.ParamNext

	// ContractChain marks the append-and-replay protocol, whose bundle
	// claim speaks "chain law" — the one wording the bundle varies by;
	// ContractLease the keyed-exclusion protocol whose oracle tiers
	// ships. Both are read by membership alone, so no role consts.
	ContractChain = chain.Name
	ContractLease = lease.Name

	// The pool contract's put arm: the returning method's smoke
	// borrows from the get sibling first, because its input is
	// pool-produced and not the fixture's to derive. eidos publishes
	// the pool vocabulary as a slice, so the role spellings are
	// literals held upstream by TestContractVocabularyIsUpstream.
	ContractPool    = pool.Name
	ContractPoolGet = "get"
	ContractPoolPut = "put"
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
func contractDataOf(bag *sdk.Bag) (roles, partners, params map[string]string) {
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
		{ContractCursor, ContractCursorClose},
		{ContractCursor, ContractCursorNext},
	}
	for _, w := range wantedPartners {
		v, found := shape.ContractPartnerKey(w.contract, w.role).Get(bag)
		set(&partners, w.contract+"."+w.role, v, found)
	}

	// The params a rule reads. Opaque to the resolver — a param names a
	// value rather than a callable, so there is nothing to resolve it
	// against — which is why they arrive verbatim. Listed rather than
	// swept: a contract's full param schema is its own business, and
	// only the ones a derivation here spends are worth projecting.
	wantedParams := [...]struct{ contract, param string }{
		{ContractIfAbsent, ContractIfAbsentConflict},
	}
	for _, w := range wantedParams {
		v, found := shape.ContractParamKey(w.contract, w.param).Get(bag)
		set(&params, w.contract+"."+w.param, v, found)
	}

	return roles, partners, params
}
