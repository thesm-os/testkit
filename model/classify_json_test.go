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
