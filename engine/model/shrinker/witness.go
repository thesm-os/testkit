// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shrinker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Witness is a one-line failure predicate. It collapses the
// reduced action sequence plus the diagnostic reason into a single
// string suitable for `<artifactDir>/witness-<seed>.txt`.
//
// The format is:
//
//	after Step1; Step2; ...; StepN: Reason
//
// or just `Reason` when Sequence is empty (the failure fired on
// the initial state).
type Witness struct {
	// Sequence is the formatted action names in execution order.
	// One entry per causal-hull step.
	Sequence []string

	// Reason is the diagnostic message — the law's Check return,
	// typically.
	Reason string
}

// String returns the one-line predicate. Pre-allocated builder;
// stable formatting across runs.
func (w Witness) String() string {
	if len(w.Sequence) == 0 {
		return w.Reason
	}
	var b strings.Builder
	b.WriteString("after ")
	b.WriteString(strings.Join(w.Sequence, "; "))
	b.WriteString(": ")
	b.WriteString(w.Reason)
	return b.String()
}

// ExtractWitness builds a [Witness] from a step sequence plus the
// failure reason. The step list is typically the output of
// [CausalHull] — the minimal causally-connected slice. The names
// of each step appear in the witness in original order.
func ExtractWitness(steps []Step, reason string) Witness {
	seq := make([]string, len(steps))
	for i, s := range steps {
		seq[i] = s.Name
	}
	return Witness{Sequence: seq, Reason: reason}
}

// WriteWitness writes w.String() to <dir>/witness-<seed>.txt and
// returns the resolved path. Creates dir when it does not exist;
// overwrites any existing file at the target path. seed is the
// hex-formatted seed value the rapid runner surfaces; an empty
// seed becomes the literal "witness".
func WriteWitness(dir, seed string, w Witness) (string, error) {
	if dir == "" {
		return "", errors.New("shrinker: WriteWitness: dir is empty")
	}
	if seed == "" {
		seed = "witness"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("shrinker: WriteWitness: mkdir: %w", err)
	}
	path := filepath.Join(dir, "witness-"+seed+".txt")
	if err := os.WriteFile(path, []byte(w.String()+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("shrinker: WriteWitness: write: %w", err)
	}
	return path, nil
}
