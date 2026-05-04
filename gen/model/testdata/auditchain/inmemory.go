// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package auditchain

import (
	"context"
	"iter"
	"sync/atomic"

	"go.thesmos.sh/testkit/model/refchain"
)

// InMemoryAuditLog implements [AuditLog] backed by [refchain.AppendOnly].
// Hash chain is real — Verify actually recomputes and validates.
type InMemoryAuditLog struct {
	chain *refchain.AppendOnly[Entry]
}

// NewInMemoryAuditLog returns a ready-to-use [InMemoryAuditLog].
func NewInMemoryAuditLog() *InMemoryAuditLog {
	return &InMemoryAuditLog{chain: refchain.New[Entry](nil)}
}

func (l *InMemoryAuditLog) Append(ctx context.Context, entry Entry) error {
	return l.chain.Append(ctx, entry)
}

func (l *InMemoryAuditLog) Replay(ctx context.Context) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx)
}

func (l *InMemoryAuditLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

// BrokenAuditLog silently drops every 3rd entry on Append.
type BrokenAuditLog struct {
	chain *refchain.AppendOnly[Entry]
	count atomic.Int64
}

// NewBrokenAuditLog returns an AuditLog that drops entries.
func NewBrokenAuditLog() *BrokenAuditLog {
	return &BrokenAuditLog{chain: refchain.New[Entry](nil)}
}

func (l *BrokenAuditLog) Append(ctx context.Context, entry Entry) error {
	n := l.count.Add(1)
	if n%3 == 0 {
		// BUG: silently drops every 3rd entry
		return nil
	}
	return l.chain.Append(ctx, entry)
}

func (l *BrokenAuditLog) Replay(ctx context.Context) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx)
}

func (l *BrokenAuditLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

// TamperedAuditLog corrupts the Verify result after every append.
// Verify returns a non-nil error, exercising HashChainIntegrityViaVerify
// negative path.
type TamperedAuditLog struct {
	chain    *refchain.AppendOnly[Entry]
	appended bool
}

// NewTamperedAuditLog returns an AuditLog with broken hash verification.
func NewTamperedAuditLog() *TamperedAuditLog {
	return &TamperedAuditLog{chain: refchain.New[Entry](nil)}
}

func (l *TamperedAuditLog) Append(ctx context.Context, entry Entry) error {
	err := l.chain.Append(ctx, entry)
	l.appended = true
	return err
}

func (l *TamperedAuditLog) Replay(ctx context.Context) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx)
}

func (l *TamperedAuditLog) Verify(_ context.Context) error {
	if l.appended {
		// BUG: always reports integrity failure after first append
		return refchain.ErrChainIntegrity
	}
	return nil
}
