// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package auditchain_causal exercises Pillar 4 Phase B:
// causal ordering with //testkit:entry-id and //testkit:depends-on.
package auditchain_causal

import (
	"context"
	"iter"
)

//go:generate testkit model -o causallogtest/causallog_model.gen.go CausalLog

// Entry is a log entry with causal dependencies.
type Entry struct {
	ID        string
	DependsOn []string
	Data      string
}

// CausalLog is an append-only chain with causal ordering constraints.
type CausalLog interface {
	//testkit:appends
	Append(ctx context.Context, entry Entry) error

	//testkit:verifies
	Verify(ctx context.Context) error

	//testkit:replays
	//testkit:entry-id ID
	//testkit:depends-on DependsOn
	Replay(ctx context.Context) iter.Seq2[Entry, error]
}
