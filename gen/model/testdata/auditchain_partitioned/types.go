// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package auditchain_partitioned exercises Pillar 4 Phase B:
// per-partition chain with //testkit:partition-by directive.
package auditchain_partitioned

import (
	"context"
	"iter"
)

//go:generate testkit model -o partlogtest/partlog_model.gen.go PartitionedLog

// Entry is a partitioned log entry.
type Entry struct {
	Tenant string
	Seq    int
	Data   string
}

// PartitionedLog is a per-tenant append-only chain.
type PartitionedLog interface {
	//testkit:appends
	Append(ctx context.Context, entry Entry) error

	//testkit:verifies
	Verify(ctx context.Context) error

	//testkit:replays
	//testkit:partition-by Tenant
	Replay(ctx context.Context, tenant string) iter.Seq2[Entry, error]
}
