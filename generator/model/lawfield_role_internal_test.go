// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/suite"
)

func TestRoleFieldShapes(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("a scalar count binds, and its variants pick their spellings", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.CountEqualsReference, "Count",
			projected("Count", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{res(namedRef("int")), errRet}))
		testkit.True(t, reason == "" && field != nil, "a nullary (int, error) counts: "+reason)

		field, reason = bindField(b, lawid.AggregatorBounded, "Read",
			projected("List", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(sliceRef(namedRef(qStr))), errRet}))
		testkit.True(t, reason == "", "a slice observation is its length: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.ScalarLen",
			"through the length adaptation")

		field, reason = bindField(b, lawid.CountEqualsReference, "Count",
			projected("Count", nil, []golang.Return{res(namedRef("int"))}))
		testkit.True(t, reason == "", "a bare scalar threads its own nil: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.ScalarNoErr",
			"through the no-error spelling")
	})

	t.Run("a scalar refuses inputs, handles, and unordered bounds", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.CountEqualsReference, "Count",
			projected("Count", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("takes inputs", "an input needs a supplier no observation has")

		_, reason = bindField(b, lawid.CountEqualsReference, "Count",
			projected("Watch", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{res(chanRef()), errRet}))
		testkit.Assert(t, reason).Contains("identity", "a channel compares by identity")

		_, reason = bindField(b, lawid.AggregatorBounded, "Read",
			projected("Peek", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(pkgRef("example.com/x", "Payload")), errRet}))
		testkit.Assert(t, reason).Contains("no bound orders", "a bound needs an ordered scalar")
	})

	t.Run("the bare calls hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.PredicateConsistent, "Call",
			projected("IsEmpty", nil, []golang.Return{res(namedRef("bool"))}))
		testkit.True(t, reason == "", "a bare predicate binds: "+reason)

		_, reason = bindField(b, lawid.PredicateConsistent, "Call",
			projected("IsEmpty", nil, []golang.Return{res(namedRef("bool")), errRet}))
		testkit.Assert(t, reason).Contains("not a bare predicate", "an error return is not the shape")

		_, reason = bindField(b, lawid.PureDeterministic, "Call",
			projected("Describe", nil, []golang.Return{res(namedRef(qStr))}))
		testkit.True(t, reason == "", "a bare pure call binds: "+reason)

		_, reason = bindField(b, lawid.PureDeterministic, "Call",
			projected("Describe", nil, []golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("not a bare pure call", "an error return is not the shape")
	})

	t.Run("the transformations thread one input", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.TotalOver, "Call",
			projected("Classify", []golang.Param{arg("ctx", ctxRef()), arg("in", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.True(t, reason == "" && field.In != nil, "a one-input transformation binds: "+reason)

		_, reason = bindField(b, lawid.TotalOver, "Call",
			projected("Join", []golang.Param{arg("a", namedRef(qStr)), arg("b", namedRef(qStr))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("several inputs", "two inputs compose no single-value closure")

		_, reason = bindField(b, lawid.TotalOver, "Call",
			projected("Sink", []golang.Param{arg("in", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nothing to observe", "a void transformation observes nothing")
	})

	t.Run("the error operations hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Keys: Pool{Q: qStr}}
		_, reason := bindField(b, lawid.LifecycleRespectsContext, "Op",
			projected("Close", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a context-taking close binds: "+reason)

		_, reason = bindField(b, lawid.LifecycleRespectsContext, "Op",
			projected("Stop", nil, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("context-respecting",
			"the context law needs a context to cancel")

		_, reason = bindField(b, lawid.PoisonNilOnFresh, "Probe",
			projected("Err", nil, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a nullary error probe binds: "+reason)

		_, reason = bindField(b, lawid.PoisonNilOnFresh, "Probe",
			projected("Check", []golang.Param{arg("k", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nullary error operation", "a probe takes nothing")

		_, reason = bindField(b, lawid.LeaseDoubleAcquireBlocks, "Acquire",
			projected("Acquire", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "", "a keyed error operation binds: "+reason)

		_, reason = bindField(b, lawid.LeaseDoubleAcquireBlocks, "Acquire",
			projected("Acquire", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("keyed error operation", "an acquire takes its key")

		_, reason = bindField(b, lawid.LeaseDoubleAcquireBlocks, "Acquire",
			stamp(projected("Acquire", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef("int"))},
				[]golang.Return{errRet}), "", "int", ""))
		testkit.Assert(t, reason).Contains("beside a pool of", "the pool and the key must agree")
	})

	t.Run("the paired string write holds its shape", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.InjectionSafe, "Store",
			projected("Store", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a string-keyed string write binds: "+reason)

		_, reason = bindField(b, lawid.InjectionSafe, "Store",
			projected("Store", []golang.Param{
				arg("ctx", ctxRef()), arg("k", namedRef("int")), arg("v", namedRef(qStr)),
			}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("string-keyed", "the injection probe speaks strings")
	})

	t.Run("the sum and the merge hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.Conservative, "Sum",
			projected("Total", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.True(t, reason == "", "an integer total binds: "+reason)

		_, reason = bindField(b, lawid.Conservative, "Sum",
			projected("Name", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("no sum totals", "a string is not a conserved quantity")

		_, reason = bindField(b, lawid.Conservative, "Sum",
			projected("Total", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("takes inputs", "a sum observes without input")

		_, reason = bindField(b, lawid.CRDTMerge, "Merge",
			projected("Merge", []golang.Param{arg("ctx", ctxRef()), arg("peer", namedRef("Replica"))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "", "a one-peer merge binds: "+reason)

		_, reason = bindField(b, lawid.CRDTMerge, "Merge",
			projected("Merge", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("does not merge one peer", "a merge folds exactly one")
	})

	t.Run("the persisting write synthesizes its identity or refuses", func(t *testing.T) {
		t.Parallel()
		save := projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})

		b := &Bindings{
			Subject:   suite.Subject{IfaceName: "Mixed"},
			Reference: Reference{KeyField: fieldKey},
		}
		field, reason := bindField(b, lawid.PersisterRetrievable, "Save", save)
		testkit.True(t, reason == "" && field.KeyOfName == b.KeyOfName(),
			"the saved identity is the shared projection: "+reason)

		b = &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason = bindField(b, lawid.PersisterRetrievable, "Save", save)
		testkit.Assert(t, reason).Contains("key projection", "no projection, no identity")

		_, reason = bindField(b, lawid.PersisterRetrievable, "Save",
			projected("Put", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("does not save one value", "a save takes its value")
	})

	t.Run("the offset-answering append holds its shape", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		errRet := res(namedRef("error"))
		_, reason := bindField(b, lawid.AppenderMonotonicOffsets, "Append",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
				[]golang.Return{res(namedRef("int64")), errRet}))
		testkit.True(t, reason == "", "an offset-answering append binds: "+reason)

		_, reason = bindField(b, lawid.AppenderMonotonicOffsets, "Append",
			projected("Run", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("answers none", "an offset is an integer")

		_, reason = bindField(b, lawid.AppenderMonotonicOffsets, "Append",
			projected("Run", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int64")), errRet}))
		testkit.Assert(t, reason).Contains("does not append one value", "an append takes its value")
	})

	t.Run("the replay adapts a slice and refuses the rest", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		errRet := res(namedRef("error"))
		field, reason := bindField(b, lawid.AppendOnlyGrows, "Replay",
			projected("Replay", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet}))
		testkit.True(t, reason == "" && field.Out != nil, "a slice replay adapts: "+reason)

		_, reason = bindField(b, lawid.AppendOnlyGrows, "Replay",
			projected("Replay", []golang.Param{arg("ctx", ctxRef()), arg("part", namedRef(qStr))},
				[]golang.Return{res(sliceRef(namedRef("Entry"))), errRet}))
		testkit.Assert(t, reason).Contains("single-partition replay", "a partitioned replay needs the projection")

		_, reason = bindField(b, lawid.AppendOnlyGrows, "Replay",
			stamp(projected("Replay", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("Seq")), errRet}), "streamreader", "", "example.com/x.Entry"))
		testkit.Assert(t, reason).Contains("iterator", "an iterator replay is not yet composed")
	})

	t.Run("the drains pick the slice or the collect loop", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		errRet := res(namedRef("error"))
		field, reason := bindField(b, lawid.StreamNoDuplicates, "Drain",
			projected("List", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(sliceRef(namedRef(qStr))), errRet}))
		testkit.True(t, reason == "", "a slice drain binds: "+reason)
		testkit.Equal(t, string(field.Kind()), "model.lawfield.Drain", "through the slice spelling")

		_, reason = bindField(b, lawid.StreamNoDuplicates, "Drain",
			projected("List", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("Seq"))}))
		testkit.Assert(t, reason).Contains("no stamp names", "an unstamped iterator names no element")
	})

	t.Run("a rowed law's unmapped field refuses by name", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.Cacheable, "Bogus",
			projected("Get", nil, []golang.Return{res(namedRef(qStr))}))
		testkit.Assert(t, reason).Contains("transcribes no closure shape",
			"a field the table does not map is a refusal, never a guess")
	})
}

func TestRoleMethodFamilies(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Values: Pool{Q: qStr}}

	t.Run("the writer family is the values pool's feeder", func(t *testing.T) {
		t.Parallel()
		merge := stamp(projected("Merge", []golang.Param{arg("ctx", ctxRef()), arg("p", namedRef("Replica"))},
			[]golang.Return{errRet}), "writer", "", "example.com/x.Replica")
		add := stamp(projected("Add", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
			[]golang.Return{errRet}), "writer", "", qStr)

		role, reason := roleMethod(b, harnessOf(merge, add), famWriter, nil, nil)
		testkit.True(t, reason == "", "the matching writer resolves: "+reason)
		testkit.Equal(t, role.Name, "Add", "by the pool's own type, not declaration order")

		_, reason = roleMethod(b, harnessOf(merge), famWriter, nil, nil)
		testkit.Assert(t, reason).Contains("feeding the pool", "a peer-merger is not the feeder")

		loose := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		role, reason = roleMethod(loose, harnessOf(merge), famWriter, nil, nil)
		testkit.True(t, reason == "" && role.Name == "Merge",
			"with no pool to feed, the first writer stands: "+reason)
	})

	t.Run("the aggregator and cell families resolve or refuse", func(t *testing.T) {
		t.Parallel()
		agg := stamp(projected("Total", []golang.Param{arg("ctx", ctxRef())},
			[]golang.Return{res(namedRef("int")), errRet}), "aggregator", "", "int")
		self := projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})

		role, reason := roleMethod(b, harnessOf(agg), "family.aggregator", nil, nil)
		testkit.True(t, reason == "" && role.Name == "Total", "the aggregate resolves: "+reason)

		role, reason = roleMethod(b, harnessOf(self, agg), "family.cell", self, nil)
		testkit.True(t, reason == "" && role.Name == "Total",
			"the nullary read is the cell, the selecting writer skipped: "+reason)

		_, reason = roleMethod(b, harnessOf(self), "family.cell", self, nil)
		testkit.Assert(t, reason).Contains("no nullary read", "no cell, no cell family")
	})

	t.Run("a contract's own role and its partners resolve off the stamps", func(t *testing.T) {
		t.Parallel()
		host := projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef("Value"))},
			[]golang.Return{errRet})
		shape.ContractRoleKey("updater").Set(host.Source.EnsureMeta(), "writer", "test")
		shape.ContractPartnerKey("updater", "reader").Set(host.Source.EnsureMeta(), "Get", "test")

		role, reason := roleMethod(b, nil, "updater.writer", host, nil)
		testkit.True(t, reason == "" && role == host, "the host fills its own role: "+reason)

		reader := projected("Get", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(namedRef("Value")), errRet})
		role, reason = roleMethod(b, harnessOf(host, reader), "updater.reader", host, nil)
		testkit.True(t, reason == "" && role.Name == "Get", "the partner stamp names the sibling: "+reason)

		_, reason = roleMethod(b, harnessOf(host), "updater.reader", host, nil)
		testkit.Assert(t, reason).Contains("not a method of", "a partner naming an absent method refuses")
	})
}

// TestClockShapedRoleFields pins the B1 shape vocabulary: the closures the
// isolation-and-clock laws render, each held to its transcription and each
// refusal to a reason a header prints.
func TestClockShapedRoleFields(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))

	t.Run("the corruption operations hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.TamperEvident, "Tamper",
			projected("Tamper", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.True(t, reason == "", "a nullary tamper binds: "+reason)

		_, reason = bindField(b, lawid.TamperEvident, "Tamper",
			projected("Tamper", []golang.Param{arg("k", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("nullary error operation", "a tamper takes nothing")

		_, reason = bindField(b, lawid.PoisonConsistent, "Poison",
			projected("Poison", nil, nil))
		testkit.True(t, reason == "", "a fire-and-forget corruption binds: "+reason)

		_, reason = bindField(b, lawid.PoisonConsistent, "Poison",
			projected("Poison", []golang.Param{arg("dose", namedRef("int"))}, nil))
		testkit.Assert(t, reason).Contains("no nullary corruption", "a poison takes nothing")
	})

	t.Run("the cursor's next answers the triple", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		field, reason := bindField(b, lawid.CursorNextAfterClose, "Next",
			projected("Next", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), res(namedRef("bool")), errRet}))
		testkit.True(t, reason == "" && field.Out != nil, "a (value, more, error) next binds: "+reason)

		_, reason = bindField(b, lawid.CursorNextAfterClose, "Next",
			projected("Next", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("triple", "a two-return next is not the shape")
	})

	t.Run("the pinned write pins through the pool", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}, Values: Pool{Pin: fieldKey}}
		field, reason := bindField(b, lawid.TTLExpiry, "Put",
			projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "" && field.KeyField == fieldKey, "a one-value put binds on the pin: "+reason)

		_, reason = bindField(b, lawid.TTLExpiry, "Put",
			projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr)), arg("v", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("one value", "a composite put is not the shape")

		unpinned := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason = bindField(unpinned, lawid.TTLExpiry, "Put",
			projected("Put", []golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("pins nothing", "an unpinned pool cannot age a known key")
	})

	t.Run("the deadline op anchors on the projection's fixture field", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		anchored := projected("Op", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{errRet})
		anchored.ArgFields = []string{"Target"}
		field, reason := bindField(b, lawid.DeadlineRespecting, "Op", anchored)
		testkit.True(t, reason == "" && field.KeyField == "Target", "an anchored context op binds: "+reason)
		testkit.True(t, b.LawsUseFixture, "and the binding records the fixture use")

		_, reason = bindField(b, lawid.DeadlineRespecting, "Op",
			projected("Op", []golang.Param{arg("k", namedRef(qStr))}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("context operation", "the deadline needs a context to expire")

		_, reason = bindField(b, lawid.DeadlineRespecting, "Op",
			projected("Op", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("fixture field", "an unanchored input has no fixed argument")
	})

	t.Run("the scheduler's pair hold their shapes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: suite.Subject{IfaceName: "Mixed"}}
		_, reason := bindField(b, lawid.ScheduledFiresAfterAdvance, "Schedule",
			projected("At", []golang.Param{arg("ctx", ctxRef()), arg("after", pkgRef("time", "Duration"))},
				[]golang.Return{errRet}))
		testkit.True(t, reason == "", "a one-offset schedule binds: "+reason)

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "Schedule",
			projected("At", []golang.Param{arg("ctx", ctxRef())}, []golang.Return{errRet}))
		testkit.Assert(t, reason).Contains("one offset", "a schedule takes its instant")

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "FiredCount",
			projected("Fired", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.True(t, reason == "", "a counting observation binds: "+reason)

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "FiredCount",
			projected("Fired", []golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
				[]golang.Return{res(namedRef("int")), errRet}))
		testkit.Assert(t, reason).Contains("no nullary observation", "a count takes nothing")

		_, reason = bindField(b, lawid.ScheduledFiresAfterAdvance, "FiredCount",
			projected("Fired", []golang.Param{arg("ctx", ctxRef())},
				[]golang.Return{res(namedRef(qStr)), errRet}))
		testkit.Assert(t, reason).Contains("not a count", "a string is no firing tally")
	})
}
