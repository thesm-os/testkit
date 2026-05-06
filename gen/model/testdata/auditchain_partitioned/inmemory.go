// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package auditchain_partitioned

import (
	"context"
	"iter"

	"go.thesmos.sh/testkit/model/refchain"
)

// InMemoryPartitionedLog implements [PartitionedLog] backed by
// [refchain.PartitionedAppendOnly].
type InMemoryPartitionedLog struct {
	chain *refchain.PartitionedAppendOnly[string, Entry]
}

// NewInMemoryPartitionedLog returns a ready-to-use implementation.
func NewInMemoryPartitionedLog() *InMemoryPartitionedLog {
	return &InMemoryPartitionedLog{
		chain: refchain.NewPartitioned(
			func(e Entry) string { return e.Tenant },
			nil,
		),
	}
}

func (l *InMemoryPartitionedLog) Append(ctx context.Context, entry Entry) error {
	return l.chain.Append(ctx, entry)
}

func (l *InMemoryPartitionedLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

func (l *InMemoryPartitionedLog) Replay(ctx context.Context, tenant string) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx, tenant)
}

// LeakyPartitionedLog appends entries to the wrong partition.
// Replay("A") returns entries that were appended with Tenant="B".
type LeakyPartitionedLog struct {
	chain *refchain.PartitionedAppendOnly[string, Entry]
}

// NewLeakyPartitionedLog returns a PartitionedLog with cross-partition leaks.
func NewLeakyPartitionedLog() *LeakyPartitionedLog {
	return &LeakyPartitionedLog{
		chain: refchain.NewPartitioned(
			// BUG: always routes to "leaked" partition regardless of entry.Tenant
			func(_ Entry) string { return "leaked" },
			nil,
		),
	}
}

func (l *LeakyPartitionedLog) Append(ctx context.Context, entry Entry) error {
	return l.chain.Append(ctx, entry)
}

func (l *LeakyPartitionedLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

func (l *LeakyPartitionedLog) Replay(ctx context.Context, tenant string) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx, tenant)
}
