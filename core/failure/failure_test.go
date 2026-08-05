// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/failure"
	"go.thesmos.sh/testkit/core/trace"
)

func TestPositionFormatting(t *testing.T) {
	t.Parallel()

	t.Run("zero value renders <unknown>", func(t *testing.T) {
		t.Parallel()
		var p failure.Position
		testkit.True(t, p.IsZero(), "zero")
		testkit.Equal(t, p.String(), "<unknown>", "format")
	})

	t.Run("populated position renders file:line:col", func(t *testing.T) {
		t.Parallel()
		p := failure.Position{Filename: "store.go", Line: 42, Column: 8}
		testkit.False(t, p.IsZero(), "non-zero")
		testkit.Equal(t, p.String(), "store.go:42:8", "format")
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("populates required fields and Time", func(t *testing.T) {
		t.Parallel()
		before := time.Now()
		f := failure.New("model", failure.KindInvariant, errors.New("boom"))
		after := time.Now()

		testkit.Equal(t, f.Generator, "model", "generator")
		testkit.Equal(t, f.Kind, failure.KindInvariant, "kind")
		testkit.Equal(t, f.Err.Error(), "boom", "err")
		testkit.True(t, !f.Time.Before(before), "Time >= before")
		testkit.True(t, !f.Time.After(after), "Time <= after")
	})

	t.Run("optional fields default to zero", func(t *testing.T) {
		t.Parallel()
		f := failure.New("sim", failure.KindLiveness, nil)
		testkit.Equal(t, f.REQID, "", "REQID empty")
		testkit.Equal(t, f.Subject, "", "Subject empty")
		testkit.Equal(t, len(f.Artifacts), 0, "no artifacts")
		testkit.Equal(t, len(f.Details), 0, "no details")
	})
}

func TestFailureError(t *testing.T) {
	t.Parallel()

	t.Run("formats with all components", func(t *testing.T) {
		t.Parallel()
		f := &failure.Failure{
			Generator: "model",
			Kind:      failure.KindInvariant,
			REQID:     "REQ-STORE-001",
			Subject:   "basic.Store",
			Err:       errors.New("read-after-write violated"),
		}
		testkit.Equal(t, f.Error(),
			"[model/invariant] [REQ-STORE-001] basic.Store: read-after-write violated",
			"full format")
	})

	t.Run("collapses missing fields", func(t *testing.T) {
		t.Parallel()
		f := &failure.Failure{
			Generator: "chaos",
			Kind:      failure.KindChaosCrash,
			Err:       errors.New("panic"),
		}
		testkit.Equal(t, f.Error(),
			"[chaos/chaos-crash] panic",
			"REQID and Subject collapse")
	})

	t.Run("renders even without an underlying error", func(t *testing.T) {
		t.Parallel()
		f := &failure.Failure{
			Generator: "sim",
			Kind:      failure.KindLiveness,
			Subject:   "ledger.Subsystem",
		}
		testkit.Equal(t, f.Error(),
			"[sim/liveness] ledger.Subsystem:",
			"trailing colon when no err")
	})

	t.Run("kind alone formats with no generator prefix", func(t *testing.T) {
		t.Parallel()
		f := &failure.Failure{Kind: failure.KindStructural, Err: errors.New("bad")}
		testkit.Equal(t, f.Error(), "[structural] bad", "no slash when no generator")
	})
}

func TestFailureUnwrap(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("inner")
	f := failure.New("model", failure.KindInvariant, sentinel)
	testkit.True(t, errors.Is(f, sentinel), "errors.Is traverses Unwrap")
}

func TestFailureJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("preserves all top-level fields", func(t *testing.T) {
		t.Parallel()
		original := &failure.Failure{
			Kind:      failure.KindInvariant,
			REQID:     "REQ-001",
			Pos:       failure.Position{Filename: "store.go", Line: 42, Column: 8},
			Subject:   "basic.Store",
			Generator: "model",
			Seed:      0xCAFEBABE,
			Snapshot: &failure.Snapshot{
				PerComponent: map[string]any{"Ledger": map[string]any{"x": 1.0}},
			},
			Artifacts: []failure.Artifact{
				{Kind: failure.ArtifactFailfile, Path: "/tmp/f", Format: "binary"},
			},
			Details: map[string]any{"law_id": "AUTO-READ-AFTER-WRITE"},
			Err:     errors.New("boom"),
			Time:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		}

		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		var got failure.Failure
		err = json.Unmarshal(b, &got)
		testkit.NoError(t, err, "unmarshal")

		testkit.Equal(t, got.Kind, original.Kind, "kind")
		testkit.Equal(t, got.REQID, original.REQID, "req")
		testkit.Equal(t, got.Pos, original.Pos, "pos")
		testkit.Equal(t, got.Subject, original.Subject, "subject")
		testkit.Equal(t, got.Generator, original.Generator, "generator")
		testkit.Equal(t, got.Seed, original.Seed, "seed")
		testkit.Equal(t, got.Err.Error(), "boom", "err message preserved")
		testkit.Equal(t, got.Time.UTC(), original.Time.UTC(), "time")
		testkit.Equal(t, len(got.Artifacts), 1, "artifacts")
		testkit.Equal(t, got.Artifacts[0].Kind, failure.ArtifactFailfile, "artifact kind")
	})

	t.Run("nil error round-trips as no err field", func(t *testing.T) {
		t.Parallel()
		original := &failure.Failure{
			Kind:      failure.KindStructural,
			Generator: "sim",
			Time:      time.Now().UTC(),
		}
		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")
		testkit.Assert(t, string(b)).NotContains("\"err\":", "no err key emitted")

		var got failure.Failure
		err = json.Unmarshal(b, &got)
		testkit.NoError(t, err, "unmarshal")
		testkit.True(t, got.Err == nil, "err remains nil")
	})

	t.Run("unmarshals trace events", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(trace.Event{Method: "X"})

		original := &failure.Failure{
			Kind:      failure.KindInvariant,
			Generator: "model",
			Trace:     tr,
		}
		b, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		var got failure.Failure
		err = json.Unmarshal(b, &got)
		testkit.NoError(t, err, "unmarshal")
		testkit.True(t, got.Trace != nil, "trace present")
		testkit.Equal(t, got.Trace.Len(), 1, "one event recovered")
		testkit.Equal(t, got.Trace.Snapshot()[0].Method, "X", "method preserved")
	})

	t.Run("rejects unknown kind on unmarshal", func(t *testing.T) {
		t.Parallel()
		var f failure.Failure
		err := json.Unmarshal([]byte(`{"kind":"made-up","generator":"x"}`), &f)
		testkit.True(t, err != nil, "must error")
	})

	t.Run("rejects mistyped envelope field", func(t *testing.T) {
		t.Parallel()
		// Valid JSON but seed isn't an integer; exercises the
		// inner json.Unmarshal error path inside the custom
		// UnmarshalJSON.
		var f failure.Failure
		err := json.Unmarshal([]byte(`{"seed":"not-a-number"}`), &f)
		testkit.True(t, err != nil, "must error")
	})

	t.Run("propagates marshal errors from unmarshalable values", func(t *testing.T) {
		t.Parallel()
		f := &failure.Failure{
			Kind:    failure.KindStructural,
			Details: map[string]any{"ch": make(chan int)},
		}
		_, err := json.Marshal(f)
		testkit.True(t, err != nil, "must error on unmarshalable value")
	})
}
