// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shrinker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/shrinker"
)

func TestWitnessString(t *testing.T) {
	t.Parallel()

	t.Run("single-step witness formats as after S1: reason", func(t *testing.T) {
		t.Parallel()
		w := shrinker.Witness{Sequence: []string{"Put(k, v1)"}, Reason: "Get returned nothing"}
		testkit.Equal(t, w.String(), "after Put(k, v1): Get returned nothing", "format")
	})

	t.Run("multi-step joins with semicolons", func(t *testing.T) {
		t.Parallel()
		w := shrinker.Witness{
			Sequence: []string{"Put(k, v1)", "Put(k, v2)", "Get(k)"},
			Reason:   "got v1, want v2",
		}
		testkit.Equal(t, w.String(),
			"after Put(k, v1); Put(k, v2); Get(k): got v1, want v2",
			"format")
	})

	t.Run("empty sequence returns just the reason", func(t *testing.T) {
		t.Parallel()
		w := shrinker.Witness{Reason: "initial state violated invariant"}
		testkit.Equal(t, w.String(), "initial state violated invariant", "no prefix")
	})
}

func TestExtractWitness(t *testing.T) {
	t.Parallel()

	t.Run("formats sequence names plus reason", func(t *testing.T) {
		t.Parallel()
		steps := []shrinker.Step{
			{Name: "Put(k, v1)"},
			{Name: "Put(k, v2)"},
			{Name: "Get(k)"},
		}
		got := shrinker.ExtractWitness(steps, "got v1, want v2")
		testkit.Equal(t, got.Sequence, []string{"Put(k, v1)", "Put(k, v2)", "Get(k)"}, "names")
		testkit.Equal(t, got.Reason, "got v1, want v2", "reason")
	})

	t.Run("empty steps yields empty Sequence", func(t *testing.T) {
		t.Parallel()
		got := shrinker.ExtractWitness(nil, "initial-state failure")
		testkit.Equal(t, len(got.Sequence), 0, "empty")
		testkit.Equal(t, got.Reason, "initial-state failure", "reason preserved")
	})
}

func TestWriteWitness(t *testing.T) {
	t.Parallel()

	w := shrinker.Witness{Sequence: []string{"Put(k, v)"}, Reason: "Get returned nothing"}

	t.Run("writes <dir>/witness-<seed>.txt", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path, err := shrinker.WriteWitness(dir, "0xDEADBEEF", w)
		testkit.NoError(t, err, "write")
		testkit.True(t, strings.HasSuffix(path, "witness-0xDEADBEEF.txt"), "named file")
		body, err := os.ReadFile(path)
		testkit.NoError(t, err, "read back")
		testkit.Assert(t, string(body)).
			Contains("after Put(k, v):", "format preserved").
			HasSuffix("\n", "newline-terminated")
	})

	t.Run("empty seed becomes 'witness'", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path, err := shrinker.WriteWitness(dir, "", w)
		testkit.NoError(t, err, "write")
		testkit.True(t, strings.HasSuffix(path, "witness-witness.txt"), "default seed")
	})

	t.Run("empty dir errors", func(t *testing.T) {
		t.Parallel()
		_, err := shrinker.WriteWitness("", "seed", w)
		testkit.True(t, err != nil, "empty dir rejected")
	})

	t.Run("creates the directory if missing", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "nested", "deeper")
		path, err := shrinker.WriteWitness(dir, "x", w)
		testkit.NoError(t, err, "mkdir + write")
		_, statErr := os.Stat(path)
		testkit.NoError(t, statErr, "file exists")
	})
}

func TestWitnessFromCausalHull(t *testing.T) {
	t.Parallel()

	t.Run("witness uses causally-shrunk sequence", func(t *testing.T) {
		t.Parallel()
		// Synthesize a 5-step sequence with 2-step causal hull.
		all := []shrinker.Step{
			{Name: "Put(k, v1)", Writes: []string{"k"}},
			{Name: "Put(other, x)", Writes: []string{"other"}},
			{Name: "noise"},
			{Name: "Put(k, v2)", Writes: []string{"k"}},
			{Name: "Get(k)", Reads: []string{"k"}},
		}
		hull := shrinker.CausalHull(all, len(all)-1)
		got := shrinker.ExtractWitness(hull, "got v1, want v2").String()
		testkit.Equal(t, got, "after Put(k, v2); Get(k): got v1, want v2", "minimal")
	})
}
