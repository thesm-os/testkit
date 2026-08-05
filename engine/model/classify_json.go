// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/failure"
	"go.thesmos.sh/testkit/core/trace"
)

// modelKindToFailureKind maps a model-package FailureKind to the
// unified testkit/failure.Kind so the on-disk JSON shares the
// schema CI tooling reads.
func modelKindToFailureKind(k FailureKind) failure.Kind {
	switch k {
	case FailureStructural:
		return failure.KindStructural
	case FailureSemantic:
		return failure.KindSemantic
	case FailureInvariant:
		return failure.KindInvariant
	case FailureLiveness:
		return failure.KindLiveness
	default:
		return failure.KindUnclassified
	}
}

// ToUnifiedFailure converts a model-package Failure into a unified
// testkit/failure.Failure. The unified envelope carries the same
// classification, error, and any embedded REQ tag; per-kind
// details land in the Details map under the "lawID" and
// "stepRan"/"stepReported" keys for CI consumption.
//
// The unified Failure has no Subject/Generator/Seed by default;
// the caller (the runner) populates those before persistence.
func ToUnifiedFailure(f *Failure) *failure.Failure {
	uf := failure.New("model", modelKindToFailureKind(f.Kind), f.Err)
	uf.REQID = f.REQID
	uf.Details = map[string]any{
		"law_id":         f.LawID,
		"step_ran":       fmt.Sprintf("worker=%d index=%d", f.StepRan.WorkerID, f.StepRan.Index),
		"step_reported":  fmt.Sprintf("worker=%d index=%d", f.StepReported.WorkerID, f.StepReported.Index),
		"trace_events":   len(f.Trace),
		"artifact_paths": f.ArtifactPaths,
	}
	return uf
}

// emitClassifiedJSON resolves the artifact directory, writes the
// classified-Failure JSON, and returns the resulting path. Logs and
// returns "" when the override dir resolves empty or the write fails.
// Used by the runner's failure paths to complete the three-artifact
// dump (rapid failfile, Porcupine HTML, classified-Failure JSON).
func emitClassifiedJSON(rt rapid.TB, override string, f *Failure) string {
	dir := ResolveArtifactDir(override)
	if dir == "" {
		return ""
	}
	seed := sanitizeForFilename(rt.Name())
	path, err := WriteClassifiedFailure(dir, seed, f)
	if err != nil {
		rt.Logf("failed to write classified failure: %v", err)
		return ""
	}
	return path
}

// emitPerClientJSON splits a multi-client failure into per-client
// classified-Failure JSONs alongside the aggregate. The aggregate
// (already written by [emitClassifiedJSON]) covers the cross-client
// view; the per-client files filter the trace to one ClientID each
// so per-client downstream tooling (e.g., per-client REQ coverage)
// has a focused artifact.
//
// No-op when fewer than two distinct non-sequential ClientIDs appear
// in the trace; sequential (ClientID == -1) events never produce a
// per-client file. Returned paths are appended to f.ArtifactPaths by
// the caller.
func emitPerClientJSON(rt rapid.TB, override string, f *Failure) []string {
	dir := ResolveArtifactDir(override)
	if dir == "" {
		return nil
	}
	clients := distinctClients(f.Trace)
	if len(clients) < 2 {
		return nil
	}
	seed := sanitizeForFilename(rt.Name())
	paths := make([]string, 0, len(clients))
	for _, id := range clients {
		clientF := *f // shallow copy
		clientF.Trace = filterTraceByClient(f.Trace, id)
		clientSeed := fmt.Sprintf("%s-client%d", seed, id)
		path, err := WriteClassifiedFailure(dir, clientSeed, &clientF)
		if err != nil {
			rt.Logf("failed to write per-client failure for client %d: %v", id, err)
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

// distinctClients returns the sorted set of non-sequential ClientIDs
// observed in the trace. Sequential events (ClientID == -1) are
// excluded — only multi-client concurrent traces participate.
func distinctClients(events []trace.Event) []int {
	seen := make(map[int]struct{})
	for _, ev := range events {
		if ev.ClientID < 0 {
			continue
		}
		seen[ev.ClientID] = struct{}{}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// filterTraceByClient returns the subset of events emitted by the
// named client. Used by [emitPerClientJSON] to scope each per-client
// failure to that client's observations.
func filterTraceByClient(events []trace.Event, clientID int) []trace.Event {
	out := make([]trace.Event, 0, len(events))
	for _, ev := range events {
		if ev.ClientID == clientID {
			out = append(out, ev)
		}
	}
	return out
}

// WriteClassifiedFailure persists the unified-failure JSON
// representation of f to <dir>/failure-<seed>.json. Returns the
// resolved path. Creates dir when absent; overwrites any existing
// file at the target path. seed is the hex-formatted seed value
// the runner surfaces; an empty seed becomes the literal
// "classified".
//
// This is the third artifact in the three-artifact dump (rapid
// failfile, Porcupine HTML, classified-Failure JSON). CI tools
// parse the JSON to drive the unified PR-bot comment.
func WriteClassifiedFailure(dir, seed string, f *Failure) (string, error) {
	if dir == "" {
		return "", errors.New("model: WriteClassifiedFailure: dir is empty")
	}
	if seed == "" {
		seed = "classified"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("model: WriteClassifiedFailure: mkdir: %w", err)
	}
	uf := ToUnifiedFailure(f)
	buf, err := json.MarshalIndent(uf, "", "  ")
	if err != nil {
		return "", fmt.Errorf("model: WriteClassifiedFailure: marshal: %w", err)
	}
	path := filepath.Join(dir, "failure-"+seed+".json")
	if err := os.WriteFile(path, append(buf, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("model: WriteClassifiedFailure: write: %w", err)
	}
	return path, nil
}
