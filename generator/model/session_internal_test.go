// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/core/tiers"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// TestSessionClassifierDerivation pins the per-client classifier's arms:
// every refusal names what is missing, the write-ordering laws hold out for
// a write the trace can see, and the derivation is one file-level function
// however many laws read it.
func TestSessionClassifierDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	sessionRule := func(law string) tiers.Rule {
		return tiers.Rule{Law: law, Needs: []string{mixinMonotonicReads}, Fields: []tiers.Field{
			{Name: "Classify", Kind: tiers.KindHandle, From: handleClassifier},
		}}
	}
	classify := func(b *Bindings, lawID string, m, keyed *subject.Method) (*LawField, string) {
		r := sessionRule(lawID)
		return lawFieldOf(b, nil, r, r.Fields[0], m, keyed)
	}
	stampedReader := func() *subject.Method {
		reader := projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
		shape.MixinParamKey(mixinMonotonicReads, "version").Set(reader.Source.EnsureMeta(), "Rev", "test")
		return reader
	}

	t.Run("no keyed reader, no ordering to read", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		_, reason := classify(b, lawid.MonotonicReads, unstamped(), nil)
		testkit.Assert(t, reason).Contains("no keyed reader", "the guarantee is about reads")
	})

	t.Run("no version= member, no ordering stamp", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		reader := projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
		_, reason := classify(b, lawid.MonotonicReads, unstamped(), reader)
		testkit.Assert(t, reason).Contains("version=", "the mixin names the member or nothing orders")
	})

	t.Run("a write-ordering law holds out for a visible write", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{Subject: subject.Subject{IfaceName: "Mixed"}}
		r := tiers.Rule{Law: lawid.MonotonicWrites, Needs: []string{"monotonicwrites"}, Fields: []tiers.Field{
			{Name: "Classify", Kind: tiers.KindHandle, From: handleClassifier},
		}}
		reader := stampedReader()
		shape.MixinParamKey("monotonicwrites", "version").Set(reader.Source.EnsureMeta(), "Rev", "test")
		_, reason := lawFieldOf(b, &subject.Projection{}, r, r.Fields[0], reader, reader)
		testkit.Assert(t, reason).Contains("answering", "the shape that would surface the stamp is named")
	})

	t.Run("the read-ordering law binds and the derivation memoizes", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject:         subject.Subject{IfaceName: "Mixed"},
			Keys:            Pool{Type: sdk.Builtin(qStr)},
			sessionKeyField: fieldKey,
		}
		reader := stampedReader()
		field, reason := classify(b, lawid.MonotonicReads, reader, reader)
		testkit.True(t, reason == "" && field != nil, "a stamped keyed reader classifies: "+reason)
		testkit.Equal(t, field.KeyOfName, "mixedSessionClassify", "through the one file-level function")
		testkit.True(t, b.Session != nil && b.Session.VersionField == "Rev",
			"and the derivation lands on the bindings")

		again, reason := classify(b, lawid.MonotonicReads, reader, reader)
		testkit.True(t, reason == "" && again.KeyOfName == field.KeyOfName,
			"a second law reads the same derivation: "+reason)
	})

	t.Run("a key pool nothing draws refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject:         subject.Subject{IfaceName: "Mixed"},
			sessionKeyField: fieldKey,
		}
		_, reason := classify(b, lawid.MonotonicReads, stampedReader(), stampedReader())
		testkit.Assert(t, reason).Contains("key type", "the laws instantiate at the pool's key")
	})

	t.Run("a reader with several results observes no single version", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject:         subject.Subject{IfaceName: "Mixed"},
			Keys:            Pool{Type: sdk.Builtin(qStr)},
			sessionKeyField: fieldKey,
		}
		multi := projected("Get",
			[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), res(namedRef("bool")), res(namedRef("error"))})
		shape.MixinParamKey(mixinMonotonicReads, "version").Set(multi.Source.EnsureMeta(), "Rev", "test")
		_, reason := classify(b, lawid.MonotonicReads, multi, multi)
		testkit.Assert(t, reason).Contains("several results", "the classifier reads one value")
	})

	t.Run("an answering writer joins the classification", func(t *testing.T) {
		t.Parallel()
		errRet := res(namedRef("error"))
		b := &Bindings{
			Subject:         subject.Subject{IfaceName: "Mixed"},
			Keys:            Pool{Type: sdk.Builtin(qStr)},
			sessionKeyField: fieldKey,
		}
		up := projected("Persist",
			[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/s", "Value"))},
			[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
		h := &subject.Projection{Methods: []subject.Method{*up}}
		r := tiers.Rule{Law: lawid.MonotonicReads, Needs: []string{mixinMonotonicReads}, Fields: []tiers.Field{
			{Name: "Classify", Kind: tiers.KindHandle, From: handleClassifier},
		}}
		field, reason := lawFieldOf(b, h, r, r.Fields[0], stampedReader(), stampedReader())
		testkit.True(t, reason == "" && field != nil, "the spec derives beside the writer: "+reason)
		testkit.Equal(t, b.Session.Writer, "Persist", "and the write arm classifies through it")
	})

	t.Run("a value with no conventional key member refuses", func(t *testing.T) {
		t.Parallel()
		b := &Bindings{
			Subject: subject.Subject{IfaceName: "Mixed"},
			Keys:    Pool{Type: sdk.Builtin(qStr)},
		}
		_, reason := classify(b, lawid.MonotonicReads, stampedReader(), stampedReader())
		testkit.Assert(t, reason).Contains("no convention names", "per-client state needs the value's identity")
	})
}

// TestSessionVersionScan pins the stamp scan the twin decision reads: the
// first session mixin carrying version= answers, anything else scans past.
func TestSessionVersionScan(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	reader := projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	reader.Mixins = []string{"monotonicreads"}
	shape.MixinParamKey(mixinMonotonicReads, "version").Set(reader.Source.EnsureMeta(), "Rev", "test")

	carrier, member, stamped := sessionVersionOf(&subject.Projection{Methods: []subject.Method{*reader}})
	testkit.True(t, stamped && member == "Rev", "a stamped session mixin names its member")
	testkit.True(t, carrier != nil && carrier.Name == "Get", "and the carrying method rides along")

	bare := projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	bare.Mixins = []string{"monotonicreads"}
	_, _, stamped = sessionVersionOf(&subject.Projection{Methods: []subject.Method{*bare}})
	testkit.False(t, stamped, "a session mixin without version= stamps no ordering")

	other := projected("Put",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", namedRef(qStr))},
		[]golang.Return{errRet})
	other.Mixins = []string{"idempotent"}
	_, _, stamped = sessionVersionOf(&subject.Projection{Methods: []subject.Method{*other}})
	testkit.False(t, stamped, "a non-session mixin is not in the scan")
}

// TestSessionConcurrentDerivation pins the session leg: it derives whole —
// reader and writer both driven — or not at all, and lands the stepless
// family the trace laws carry.
func TestSessionConcurrentDerivation(t *testing.T) {
	t.Parallel()

	errRet := res(namedRef("error"))
	keyed := projected("Get",
		[]golang.Param{arg("ctx", ctxRef()), arg("k", namedRef(qStr))},
		[]golang.Return{res(pkgRef("example.com/s", "Value")), errRet})
	valued := projected("Persist",
		[]golang.Param{arg("ctx", ctxRef()), arg("v", pkgRef("example.com/s", "Value"))},
		[]golang.Return{errRet})

	b := &Bindings{
		Subject: subject.Subject{IfaceName: "Mixed"},
		Session: &SessionSpec{ClassifyName: "mixedSessionClassify"},
		Actions: []*Action{{Method: "Get"}, {Method: "Persist"}},
	}
	concurrentOf(b, &subject.Projection{}, keyed, valued)
	testkit.Equal(t, b.ConcFamily, "session", "both halves in hand derive the stepless leg")

	half := &Bindings{
		Subject: subject.Subject{IfaceName: "Mixed"},
		Session: &SessionSpec{ClassifyName: "mixedSessionClassify"},
		Actions: []*Action{{Method: "Get"}},
	}
	concurrentOf(half, &subject.Projection{}, keyed, valued)
	testkit.Equal(t, half.ConcFamily, "", "half a pair interleaves nothing worth checking")
	testkit.True(t, half.ConcReader == nil && half.ConcWriter == nil,
		"and the halves are reset rather than left dangling")
}
