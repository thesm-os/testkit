// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/engine/model/law"
)

// clientClassify interprets test events: method "read" carries the
// key in Inputs[0] and the read value's version in Output; method
// "write" carries the key in Inputs[0] and the write version in
// Inputs[1]. Versions are the store-assigned ordering oracle.
func clientClassify(ev trace.Event) (law.ClientOp[string], bool) {
	switch ev.Method {
	case "read":
		return law.ClientOp[string]{Write: false, Key: ev.Inputs[0].(string), Version: ev.Output.(int64)}, true
	case "write":
		return law.ClientOp[string]{Write: true, Key: ev.Inputs[0].(string), Version: ev.Inputs[1].(int64)}, true
	}
	return law.ClientOp[string]{}, false
}

func rd(client int, key string, version int64) trace.Event {
	return trace.Event{ClientID: client, Method: "read", Inputs: []any{key}, Output: version}
}

func wr(client int, key string, version int64) trace.Event {
	return trace.Event{ClientID: client, Method: "write", Inputs: []any{key, version}}
}

func TestMonotonicReads(t *testing.T) {
	t.Parallel()

	t.Run("non-decreasing read versions per client+key pass", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 1))
		tr.Record(rd(1, "a", 2))
		tr.Record(rd(1, "b", 5))
		tr.Record(rd(2, "a", 9)) // different client, independent
		l := &law.MonotonicReads[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("a read that goes backwards for a client+key fires", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 2))
		tr.Record(rd(1, "a", 1)) // stale read — version went backwards
		l := &law.MonotonicReads[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected monotonic-reads violation")
			}
		})
	})
}

func TestReadYourWrites(t *testing.T) {
	t.Parallel()

	t.Run("read after own write sees that write or newer", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "a", 2))
		tr.Record(rd(1, "a", 2)) // own write
		tr.Record(rd(1, "a", 5)) // newer (someone else wrote) — fine
		l := &law.ReadYourWrites[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("read older than the client's own write fires", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "a", 3))
		tr.Record(rd(1, "a", 1)) // stale: older than own write
		l := &law.ReadYourWrites[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected read-your-writes violation")
			}
		})
	})
}

func TestMonotonicWrites(t *testing.T) {
	t.Parallel()

	t.Run("strictly increasing write versions per client+key pass", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "a", 1))
		tr.Record(wr(1, "a", 2))
		tr.Record(wr(1, "b", 1)) // different key, independent
		tr.Record(wr(2, "a", 1)) // different client, independent
		l := &law.MonotonicWrites[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("a write stamped out of issue order fires", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "a", 5))
		tr.Record(wr(1, "a", 3)) // later write got an older version
		l := &law.MonotonicWrites[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected monotonic-writes violation")
			}
		})
	})
}

func TestWritesFollowReads(t *testing.T) {
	t.Parallel()

	t.Run("a write newer than everything read passes", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 3))
		tr.Record(wr(1, "b", 4)) // write follows the read it depends on
		tr.Record(rd(1, "a", 4))
		tr.Record(wr(1, "b", 7))
		l := &law.WritesFollowReads[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	t.Run("a write older than something the client read fires", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 5))
		tr.Record(wr(1, "a", 2)) // write is older than a value already read
		l := &law.WritesFollowReads[struct{}, string]{Classify: clientClassify}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected writes-follow-reads violation")
			}
		})
	})
}

// unclassified is an event no classifier recognises. Every per-client law must
// skip it rather than treat it as a read or a write — a trace carries more
// than the operations any one guarantee cares about.
func unclassified(client int) trace.Event {
	return trace.Event{ClientID: client, Method: "ping"}
}

func TestPerClientLawsSkipUnclassifiedEvents(t *testing.T) {
	t.Parallel()

	build := func() *trace.Trace {
		tr := trace.New()
		tr.Record(unclassified(1))
		tr.Record(rd(1, "a", 1))
		tr.Record(unclassified(1))
		tr.Record(wr(1, "a", 2))
		tr.Record(unclassified(2))
		return tr
	}

	t.Run("MonotonicReads", func(t *testing.T) {
		t.Parallel()
		l := &law.MonotonicReads[any, string]{Classify: clientClassify, Trace: build()}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("unrecognised events must be skipped: %v", err)
		}
	})

	t.Run("ReadYourWrites", func(t *testing.T) {
		t.Parallel()
		l := &law.ReadYourWrites[any, string]{Classify: clientClassify, Trace: build()}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("unrecognised events must be skipped: %v", err)
		}
	})

	t.Run("MonotonicWrites", func(t *testing.T) {
		t.Parallel()
		l := &law.MonotonicWrites[any, string]{Classify: clientClassify, Trace: build()}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("unrecognised events must be skipped: %v", err)
		}
	})

	t.Run("WritesFollowReads", func(t *testing.T) {
		t.Parallel()
		l := &law.WritesFollowReads[any, string]{Classify: clientClassify, Trace: build()}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("unrecognised events must be skipped: %v", err)
		}
	})
}

// Each per-client law watches one half of the trace and must ignore the other:
// MonotonicReads says nothing about writes, MonotonicWrites nothing about
// reads. A law that inspected both would fire on traces it has no opinion on.
func TestPerClientLawsIgnoreTheOtherOperation(t *testing.T) {
	t.Parallel()

	t.Run("MonotonicReads ignores descending writes", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "a", 9))
		tr.Record(wr(1, "a", 2)) // would violate MonotonicWrites
		l := &law.MonotonicReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("MonotonicReads has no opinion on writes: %v", err)
		}
	})

	t.Run("MonotonicWrites ignores descending reads", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 9))
		tr.Record(rd(1, "a", 2)) // would violate MonotonicReads
		l := &law.MonotonicWrites[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("MonotonicWrites has no opinion on reads: %v", err)
		}
	})

	// Both laws are scoped per (client, key): the same key seen by a different
	// client, or a different key on the same client, carries no constraint.
	t.Run("violations do not cross client or key boundaries", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 9))
		tr.Record(rd(2, "a", 2)) // different client
		tr.Record(rd(1, "b", 1)) // different key
		l := &law.MonotonicReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("per-(client,key) scoping must isolate these: %v", err)
		}
	})
}

// WritesFollowReads scopes its watermark to (client, key), like the three
// laws beside it. A version is a fact about one key, so comparing one key's
// version against another's asserts an ordering that per-key versioning — the
// dominant design — does not have.
func TestWritesFollowReadsIsPerKey(t *testing.T) {
	t.Parallel()

	t.Run("a write below a read on another key passes", func(t *testing.T) {
		t.Parallel()
		// The case that made the law unsound. A store versioning each key
		// independently answers 9 for a and stamps 3 on b, and there is
		// nothing wrong with either number: they are counters of different
		// things. The key-agnostic form failed this correct history.
		tr := trace.New()
		tr.Record(rd(1, "a", 9))
		tr.Record(wr(1, "b", 3))
		l := &law.WritesFollowReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("versions of different keys are incomparable: %v", err)
		}
	})

	t.Run("a write below a read of the same key is flagged", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 9))
		tr.Record(wr(1, "a", 3))
		l := &law.WritesFollowReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err == nil {
			t.Fatal("a write behind what the client read of that key must be flagged")
		}
	})

	t.Run("one client's reads do not constrain another's writes", func(t *testing.T) {
		t.Parallel()
		// The guarantee is per-session. Client 2 never read anything, so
		// nothing it writes can fail to follow a read.
		tr := trace.New()
		tr.Record(rd(1, "a", 9))
		tr.Record(wr(2, "a", 3))
		l := &law.WritesFollowReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("a client that has read nothing cannot violate this law: %v", err)
		}
	})

	t.Run("a write before any read is unconstrained", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "a", 1))
		l := &law.WritesFollowReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err != nil {
			t.Fatalf("a client that has read nothing cannot violate this law: %v", err)
		}
	})

	t.Run("the read watermark only ever rises", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(rd(1, "a", 9))
		tr.Record(rd(1, "a", 4)) // lower read must not lower the watermark
		tr.Record(wr(1, "a", 5))
		l := &law.WritesFollowReads[any, string]{Classify: clientClassify, Trace: tr}
		if err := l.Check(nil, nil, nil); err == nil {
			t.Fatal("a later lower read must not reset the watermark")
		}
	})
}
