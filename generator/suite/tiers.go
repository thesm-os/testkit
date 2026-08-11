// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"
	"strings"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"
)

// Tier names which half of testkit's evidence covers a classification.
//
// The assignment follows a rule rather than an opinion (docs/adr/0018): where
// [engine/model/law] carries the property, the classification is the model
// tier's; where none does, this tier owns it; and where the claim cannot be
// stated against one subject making a fixed call, neither covers it and the
// gate fails by name.
type Tier string

// The tiers a classification can be assigned to.
const (
	// TierSuite is this generator's: no law carries the property, and one
	// subject making a fixed sequence of calls can state it.
	TierSuite Tier = "suite"

	// TierModel is `engine/model`'s: a law already implements the property,
	// and re-implementing it as a template would be the drift this repository
	// has spent its history removing.
	TierModel Tier = "model"

	// TierNone is neither's, which the conformance gate reports. Each is a law
	// to write rather than a check to invent — a one-shot check with no way to
	// induce the condition passes against every implementation, including a
	// broken one.
	TierNone Tier = "none"
)

// Ownership is one classification's tier and the law that carries it.
type Ownership struct {
	// Tier is which half of the evidence covers this classification.
	Tier Tier

	// Law is the [engine/model/law] identifier for a model-tier
	// classification, empty otherwise.
	//
	// A string rather than a reference, because `engine` and `generator` are
	// separate modules and neither depends on the other (docs/adr/0005). What
	// that costs is a wrong identifier no compiler catches, which is why the
	// list below names the file it was read from.
	Law string
}

// lawWriteObservable is named because three write shapes share it: `writer`,
// `compositewriter` and `multiargwriter` differ only in arity, and the property
// a law states about them is the same one.
const lawWriteObservable = "AUTO-WRITE-OBSERVABLE"

// ownership assigns every classification eidos registers to a tier.
//
// One entry per classification, which is the machine-readable mapping RFC-0002
// asks for: without it the assignment becomes the per-classification opinion
// ADR-0017 refused, and a generated file could not say which tier covers what.
// [TestOwnershipIsComplete] holds it to the live registries, so a classification
// added or renamed upstream fails a test here rather than rendering a header
// that silently omits it.
//
// The law identifiers are `engine/model/law`'s, read from the ID methods in
// that package. Nothing binds the two at compile time — the modules are
// separate — so a law renamed there leaves a stale string here, which is the
// one gap this table cannot close on its own.
//
//nolint:gochecknoglobals // a lookup table, read-only after init
var ownership = map[string]Ownership{
	// Detectors. The suite tier owns the shape checks no law reaches; the rest
	// add a model binding.
	"reader":          {Tier: TierSuite},
	"readernoerror":   {Tier: TierSuite},
	"readerwithbool":  {Tier: TierSuite},
	"lookup":          {Tier: TierSuite},
	"pointerreader":   {Tier: TierSuite},
	"multireader":     {Tier: TierSuite},
	"batchreader":     {Tier: TierSuite},
	"mutator":         {Tier: TierSuite},
	"streamconsumer":  {Tier: TierSuite},
	"voidlifecycle":   {Tier: TierSuite},
	"writer":          {Tier: TierModel, Law: lawWriteObservable},
	"compositewriter": {Tier: TierModel, Law: lawWriteObservable},
	"multiargwriter":  {Tier: TierModel, Law: lawWriteObservable},
	"aggregator":      {Tier: TierModel, Law: "AUTO-COUNT-EQUALS-REFERENCE"},
	"multiaggregator": {Tier: TierModel, Law: "AUTO-COUNT-EQUALS-REFERENCE"},
	"streamreader":    {Tier: TierModel, Law: "AUTO-STREAM-COMPLETION"},
	"lifecycle":       {Tier: TierModel, Law: "AUTO-LIFECYCLE-RESPECTS-CONTEXT"},
	"predicate":       {Tier: TierModel, Law: "AUTO-PREDICATE-CONSISTENT"},
	"poisonaccessor":  {Tier: TierModel, Law: "AUTO-POISON-NIL-ON-FRESH"},

	// Mixins. `pure` is both a detector and a mixin and takes one entry, since
	// the property is the same either way.
	"pure":                    {Tier: TierModel, Law: "AUTO-PURE-DETERMINISTIC"},
	"deprecated":              {Tier: TierSuite},
	"errors":                  {Tier: TierSuite},
	"hooks":                   {Tier: TierSuite},
	"integrationonly":         {Tier: TierSuite},
	"nilsafe":                 {Tier: TierSuite},
	"orderafter":              {Tier: TierSuite},
	"partition":               {Tier: TierSuite},
	"retrysucceeds":           {Tier: TierSuite},
	"sample":                  {Tier: TierSuite},
	"scope":                   {Tier: TierSuite},
	"sideeffect":              {Tier: TierSuite},
	"timeout":                 {Tier: TierSuite},
	"validates":               {Tier: TierSuite},
	"wrappedvia":              {Tier: TierSuite},
	"concurrent":              {Tier: TierSuite},
	"concurrentreaders":       {Tier: TierSuite},
	"atomic":                  {Tier: TierModel, Law: "AUTO-ATOMIC-WRITE"},
	"bounded":                 {Tier: TierModel, Law: "AUTO-AGGREGATOR-BOUNDED"},
	"cacheable":               {Tier: TierModel, Law: "AUTO-CACHEABLE"},
	"crdtmerge":               {Tier: TierModel, Law: "AUTO-CRDT-MERGE"},
	"deleteremoves":           {Tier: TierModel, Law: "AUTO-DELETE-RETURNS-NOT-FOUND"},
	"eventually":              {Tier: TierModel, Law: "AUTO-EVENTUAL-CONVERGENCE"},
	"idempotent":              {Tier: TierModel, Law: "AUTO-IDEMPOTENT-WRITE"},
	"lifecycleafterclose":     {Tier: TierModel, Law: "AUTO-LIFECYCLE-AFTER-CLOSE"},
	"monotonic":               {Tier: TierModel, Law: "AUTO-MONOTONIC-NON-DECREASING"},
	"readafterwrite":          {Tier: TierModel, Law: "AUTO-READ-AFTER-WRITE"},
	"streamreflectsmutations": {Tier: TierModel, Law: "AUTO-STREAM-REFLECTS-MUTATIONS"},

	// Contracts. `batch-writer` is the model tier's rather than this one's,
	// which corrects RFC-0002's table: `mode=atomic` is the claim that an error
	// leaves observable state unchanged, and AUTO-ATOMIC-WRITE already
	// implements exactly that — so ADR-0018's rule assigns it there.
	"if-absent":       {Tier: TierSuite},
	"if-match":        {Tier: TierSuite},
	"outbox":          {Tier: TierSuite},
	"appender":        {Tier: TierModel, Law: "AUTO-APPEND-ONLY-GROWS"},
	"batch-writer":    {Tier: TierModel, Law: "AUTO-ATOMIC-WRITE"},
	"cache":           {Tier: TierModel, Law: "AUTO-CACHEABLE"},
	"cas":             {Tier: TierModel, Law: "AUTO-CAS-ATOMIC-ONE-WINNER"},
	"cursor":          {Tier: TierModel, Law: "AUTO-CURSOR-NEXT-AFTER-CLOSE"},
	"lease":           {Tier: TierModel, Law: "AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS"},
	"pagination":      {Tier: TierModel, Law: "AUTO-PAGINATOR-NO-DUPLICATES"},
	"persister":       {Tier: TierModel, Law: "AUTO-PERSISTER-RETRIEVABLE"},
	"pool":            {Tier: TierModel, Law: "AUTO-POOL-BALANCED"},
	"publisher":       {Tier: TierModel, Law: "AUTO-PUBLISHER-DELIVERS"},
	"saga":            {Tier: TierModel, Law: "AUTO-SAGA-FULL-COMPENSATION"},
	"singleflight":    {Tier: TierModel, Law: "AUTO-SINGLEFLIGHT-COALESCES"},
	"transaction":     {Tier: TierModel, Law: "AUTO-TRANSACTION-ROLLBACK"},
	"updater":         {Tier: TierModel, Law: "AUTO-UPDATER-REPLACES"},
	"upserter":        {Tier: TierModel, Law: "AUTO-UPSERTER-IDEMPOTENT"},
	"watcher":         {Tier: TierModel, Law: "AUTO-WATCHER-RETURNS-ON-CHANGE"},
	"workflow":        {Tier: TierModel, Law: "AUTO-VALID-TRANSITION"},
	"circuit-breaker": {Tier: TierNone},
	"leader-election": {Tier: TierNone},
	"rate-limit":      {Tier: TierNone},
	"tx":              {Tier: TierNone},
}

// OwnershipNames returns every classification the table assigns, sorted.
//
// The other half of the completeness claim: without it a stale entry for a
// classification eidos dropped goes unnoticed until somebody reads the table
// against the registry by hand.
func OwnershipNames() []string {
	out := make([]string, 0, len(ownership))
	for name := range ownership {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// OwnershipOf returns a classification's tier, and whether the table knows it.
func OwnershipOf(name string) (Ownership, bool) {
	o, known := ownership[name]
	return o, known
}

// Coverage is one classification an interface carries, and what covers it.
//
// Rendered into the harness header a reader meets before the checks, so an
// absent check is a stated boundary rather than a suspected defect. Without it
// a `deleteremoves` harness reads as thirteen checks with its declared mixin
// invisible, and a reader cannot tell "deferred to the model tier" from
// "silently ignored" — which is the tier-boundary form of what ADR-0017 exists
// to prevent.
type Coverage struct {
	// Axis is `detector`, `mixin` or `contract`, and Name the classification.
	Axis, Name string

	// Methods names the methods carrying it, in method-set order.
	Methods []string

	// Ownership is the tier that covers it and the law that carries it.
	Ownership Ownership

	// Checked reports whether this generator emitted a check for it here.
	//
	// Not the same as being the suite tier's: `deprecated` is this tier's and
	// generates nothing, because it is a fact about a method rather than a
	// claim about its behaviour.
	Checked bool
}

// Elsewhere reports whether the evidence for this classification comes from
// somewhere other than a check a consumer could write.
//
// The distinction the header turns on. A model-tier classification needs a
// reference implementation to compare against, so telling a consumer to write
// the check themselves is bad advice; one this generator has simply not written
// is a gap, and the extension point is exactly where it belongs. Conflating the
// two produced a header that told consumers to hand-write `deleteremoves`,
// which nobody can state against a single subject.
func (c Coverage) Elsewhere() bool { return c.Ownership.Tier == TierModel }

// checkedBy names the emit kinds that assert each classification.
//
// A mixin or a contract reports under its own name, so the subtest answers
// directly. A detector does not: `reader`'s check reports as "an error carries
// the zero value", which is what it asserts rather than what stamped it — so
// the pairing has to be written down, or a harness that checks a shape lists it
// as unchecked.
//
//nolint:gochecknoglobals // a lookup table, read-only after init
var checkedBy = map[string][]sdk.Kind{
	"reader": {KindZeroOnError},
	// Both members of the miss family, because which one a shape earns is
	// decided by the signature rather than by the stamp: a `readerwithbool`
	// whose bool the source dropped reports its miss in the value, and keying
	// on the flag form alone would list a checked shape as unchecked.
	"readernoerror":  {KindMissZero, KindMissFlag},
	"pointerreader":  {KindMissZero, KindMissFlag},
	"readerwithbool": {KindMissZero, KindMissFlag},
	"lookup":         {KindMissZero, KindMissFlag},
	"batchreader":    {KindBatchSize},
}

// DetectorCheck reports whether a kind is the check a shape stamp earns, rather
// than one the signature or a directive does.
//
// The three families name their subtests differently: a mixin's is the
// classification, and a detector's is what it asserts. So only the second needs
// the mapping, and only the second can be missing from it.
func DetectorCheck(kind sdk.Kind) bool {
	switch kind {
	case KindMissZero, KindMissFlag, KindBatchSize, KindZeroOnError:
		return true
	default:
		return false
	}
}

// ShapesCheckedBy returns the classifications a kind is the check for.
func ShapesCheckedBy(kind sdk.Kind) []string {
	var out []string
	for name, kinds := range checkedBy {
		if slices.Contains(kinds, kind) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// checksFor reports whether the method carries a check about this
// classification.
func checksFor(m Method, name string) bool {
	for _, ck := range m.Checks {
		if ck.Subtest == name {
			return true
		}
		if slices.Contains(checkedBy[name], ck.KindName) {
			return true
		}
	}
	return false
}

// coverageOf collects every classification the interface's methods carry.
//
// Union across the method set rather than per method, because the header is
// about the interface: a reader asking "what does this file not check" wants
// one list, and which methods carry each is what the entry names.
func coverageOf(methods []Method) []Coverage {
	byName := map[string]*Coverage{}
	order := []string{}

	add := func(axis, name, method string, checked bool) {
		if name == "" {
			return
		}
		c, seen := byName[name]
		if !seen {
			o, known := OwnershipOf(name)
			if !known {
				// An unregistered classification, which the completeness test
				// makes impossible from a real run. Recorded as covered by
				// nobody rather than dropped: a name the table does not know is
				// exactly what a reader needs to see.
				o = Ownership{Tier: TierNone}
			}
			c = &Coverage{Axis: axis, Name: name, Ownership: o}
			byName[name] = c
			order = append(order, name)
		}
		if !slices.Contains(c.Methods, method) {
			c.Methods = append(c.Methods, method)
		}
		c.Checked = c.Checked || checked
	}

	for _, m := range methods {
		add("detector", m.Shape(), m.Name, checksFor(m, m.Shape()))
		for _, name := range m.Mixins {
			add("mixin", name, m.Name, checksFor(m, name))
		}
		for _, name := range m.Contracts {
			add("contract", name, m.Name, checksFor(m, name))
		}
	}

	out := make([]Coverage, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return out
}

// Shape returns the detector the annotator stamped on this method, empty when
// it stamped none.
func (m Method) Shape() string {
	if m.Source == nil {
		return ""
	}
	return shape.Get(m.Source.Meta())
}

// MethodList spells the methods carrying a classification, for the header.
func (c Coverage) MethodList() string { return strings.Join(c.Methods, ", ") }
