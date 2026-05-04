// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package auditchain exercises Pillar 4: chain directives with
// auto-derived append-only, hash-chain integrity, and replay
// determinism laws.
package auditchain

import (
	"context"
	"iter"
)

//go:generate testkit model -o auditlogtest/auditlog_model.gen.go AuditLog

// Entry is an append-only log entry.
type Entry struct {
	ID   string
	Data string
}

// AuditLog is an append-only chain with verify and replay.
type AuditLog interface {
	//testkit:appends
	Append(ctx context.Context, entry Entry) error

	//testkit:replays
	Replay(ctx context.Context) iter.Seq2[Entry, error]

	//testkit:verifies
	Verify(ctx context.Context) error
}
