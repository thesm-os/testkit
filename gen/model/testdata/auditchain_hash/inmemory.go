// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package auditchain_hash

import (
	"context"
	"iter"

	"go.thesmos.sh/testkit/model/refchain"
)

// InMemoryHashLog implements [HashLog] backed by [refchain.AppendOnly]
// with the SHA-512 hash function.
type InMemoryHashLog struct {
	chain *refchain.AppendOnly[Entry]
}

// NewInMemoryHashLog returns a ready-to-use [InMemoryHashLog].
func NewInMemoryHashLog() *InMemoryHashLog {
	return &InMemoryHashLog{chain: refchain.New[Entry](SHA512Hash())}
}

func (l *InMemoryHashLog) Append(ctx context.Context, entry Entry) error {
	return l.chain.Append(ctx, entry)
}

func (l *InMemoryHashLog) Replay(ctx context.Context) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx)
}

func (l *InMemoryHashLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}
