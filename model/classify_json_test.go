// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/failure"
	"go.thesmos.sh/testkit/trace"
)

func TestModelKindToFailureKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		modelK FailureKind
		want   failure.Kind
	}{
		{FailureStructural, failure.KindStructural},
		{FailureSemantic, failure.KindSemantic},
		{FailureInvariant, failure.KindInvariant},
		{FailureLiveness, failure.KindLiveness},
		{FailureUnclassified, failure.KindUnclassified},
	}
	for _, c := range cases {
		t.Run(c.modelK.String(), func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, modelKindToFailureKind(c.modelK), c.want, "kind mapping")
		})
	}
}

func TestToUnifiedFailure(t *testing.T) {
	t.Parallel()

	t.Run("populates generator, kind, req, and details", func(t *testing.T) {
		t.Parallel()
		f := &Failure{
			Kind:          FailureInvariant,
			LawID:         "AUTO-READ-AFTER-WRITE",
			REQID:         "REQ-001",
			StepRan:       StepID{WorkerID: 2, Index: 7},
			StepReported:  StepID{WorkerID: 2, Index: 9},
			Err:           errors.New("boom"),
			ArtifactPaths: []string{"/tmp/witness.txt"},
		}
		uf := ToUnifiedFailure(f)
		testkit.Equal(t, uf.Generator, "model", "generator tag")
		testkit.Equal(t, uf.Kind, failure.KindInvariant, "kind mapped")
		testkit.Equal(t, uf.REQID, "REQ-001", "req preserved")
		testkit.Equal(t, uf.Err.Error(), "boom", "error preserved")
		testkit.Equal(t, uf.Details["law_id"], "AUTO-READ-AFTER-WRITE", "lawID surfaced")
		testkit.Equal(t, uf.Details["step_ran"], "worker=2 index=7", "stepRan formatted")
		testkit.Equal(t, uf.Details["step_reported"], "worker=2 index=9", "stepReported formatted")
	})
}

func TestWriteClassifiedFailure(t *testing.T) {
	t.Parallel()

	f := &Failure{
		Kind:  FailureInvariant,
		LawID: "AUTO-CAS-ATOMIC-ONE-WINNER",
		REQID: "REQ-STORE-001",
		Err:   errors.New("two writers won"),
	}

	t.Run("writes <dir>/failure-<seed>.json with valid JSON", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path, err := WriteClassifiedFailure(dir, "0xDEAD", f)
		testkit.NoError(t, err, "write")
		testkit.True(t, strings.HasSuffix(path, "failure-0xDEAD.json"), "filename")

		body, err := os.ReadFile(path)
		testkit.NoError(t, err, "read")
		var uf failure.Failure
		testkit.NoError(t, json.Unmarshal(body, &uf), "valid JSON")
		testkit.Equal(t, uf.Kind, failure.KindInvariant, "kind round-trips")
		testkit.Equal(t, uf.REQID, "REQ-STORE-001", "REQ round-trips")
	})

	t.Run("empty seed defaults to 'classified'", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path, err := WriteClassifiedFailure(dir, "", f)
		testkit.NoError(t, err, "write")
		testkit.True(t, strings.HasSuffix(path, "failure-classified.json"), "default seed")
	})

	t.Run("empty dir errors", func(t *testing.T) {
		t.Parallel()
		_, err := WriteClassifiedFailure("", "seed", f)
		testkit.True(t, err != nil, "empty dir rejected")
	})

	t.Run("nested dir is created", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "a", "b", "c")
		path, err := WriteClassifiedFailure(dir, "x", f)
		testkit.NoError(t, err, "mkdir + write")
		_, statErr := os.Stat(path)
		testkit.NoError(t, statErr, "file exists at nested path")
	})
}

func TestDistinctClients(t *testing.T) {
	t.Parallel()

	t.Run("returns sorted unique non-sequential client ids", func(t *testing.T) {
		t.Parallel()
		got := distinctClients([]trace.Event{
			{ClientID: 2}, {ClientID: 0}, {ClientID: 2}, {ClientID: 1}, {ClientID: 0},
		})
		testkit.Equal(t, len(got), 3, "three distinct clients")
		testkit.Equal(t, got[0], 0, "sorted ascending")
		testkit.Equal(t, got[1], 1, "")
		testkit.Equal(t, got[2], 2, "")
	})

	t.Run("excludes sequential ClientID -1", func(t *testing.T) {
		t.Parallel()
		got := distinctClients([]trace.Event{{ClientID: -1}, {ClientID: -1}})
		testkit.Equal(t, len(got), 0, "no multi-client view for sequential trace")
	})

	t.Run("mixed sequential and concurrent skips the sequential", func(t *testing.T) {
		t.Parallel()
		got := distinctClients([]trace.Event{{ClientID: -1}, {ClientID: 0}, {ClientID: 1}})
		testkit.Equal(t, len(got), 2, "two concurrent clients")
	})
}

func TestFilterTraceByClient(t *testing.T) {
	t.Parallel()

	events := []trace.Event{
		{ClientID: 0, Method: "A"},
		{ClientID: 1, Method: "B"},
		{ClientID: 0, Method: "C"},
		{ClientID: 2, Method: "D"},
	}
	got := filterTraceByClient(events, 0)
	testkit.Equal(t, len(got), 2, "two events for client 0")
	testkit.Equal(t, got[0].Method, "A", "first preserved")
	testkit.Equal(t, got[1].Method, "C", "second preserved")
}

func TestEmitPerClientJSON(t *testing.T) {
	t.Parallel()

	t.Run("no-op for sequential trace", func(t *testing.T) {
		t.Parallel()
		ft := testkit.NewFailableTB().WithName("seq")
		f := &Failure{Kind: FailureSemantic, Trace: []trace.Event{{ClientID: -1}}}
		got := emitPerClientJSON(ft, t.TempDir(), f)
		testkit.Equal(t, len(got), 0, "sequential trace produces no per-client files")
	})

	t.Run("no-op for single-client trace", func(t *testing.T) {
		t.Parallel()
		ft := testkit.NewFailableTB().WithName("one")
		f := &Failure{Kind: FailureSemantic, Trace: []trace.Event{{ClientID: 0}, {ClientID: 0}}}
		got := emitPerClientJSON(ft, t.TempDir(), f)
		testkit.Equal(t, len(got), 0, "one client is not multi-client")
	})

	t.Run("writes one JSON per client for multi-client trace", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		ft := testkit.NewFailableTB().WithName("multi")
		f := &Failure{
			Kind: FailureSemantic,
			Err:  errors.New("non-linearizable"),
			Trace: []trace.Event{
				{ClientID: 0, Method: "Get"},
				{ClientID: 1, Method: "Put"},
				{ClientID: 0, Method: "Delete"},
				{ClientID: 1, Method: "Get"},
			},
		}
		got := emitPerClientJSON(ft, dir, f)
		testkit.Equal(t, len(got), 2, "two per-client files")
		testkit.True(t, strings.HasSuffix(got[0], "failure-multi-client0.json"), "client0 filename")
		testkit.True(t, strings.HasSuffix(got[1], "failure-multi-client1.json"), "client1 filename")

		// Per-client JSON should carry only that client's events.
		body, readErr := os.ReadFile(got[0])
		testkit.NoError(t, readErr, "read client0 JSON")
		var uf failure.Failure
		testkit.NoError(t, json.Unmarshal(body, &uf), "valid JSON")
		testkit.Equal(t, uf.Details["trace_events"].(float64), 2.0, "two events for client0")
	})
}
