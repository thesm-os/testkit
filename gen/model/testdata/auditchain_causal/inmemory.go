// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package auditchain_causal

import (
	"context"
	"fmt"
	"iter"

	"go.thesmos.sh/testkit/model/refchain"
)

// InMemoryCausalLog implements [CausalLog] with dependency validation.
// Append rejects entries whose deps aren't already present.
type InMemoryCausalLog struct {
	chain *refchain.AppendOnly[Entry]
	ids   map[string]bool
}

// NewInMemoryCausalLog returns a CausalLog that validates deps on append.
func NewInMemoryCausalLog() *InMemoryCausalLog {
	return &InMemoryCausalLog{
		chain: refchain.New[Entry](nil),
		ids:   make(map[string]bool),
	}
}

func (l *InMemoryCausalLog) Append(ctx context.Context, entry Entry) error {
	for _, dep := range entry.DependsOn {
		if !l.ids[dep] {
			return fmt.Errorf("dependency %s not found", dep)
		}
	}
	l.ids[entry.ID] = true
	return l.chain.Append(ctx, entry)
}

func (l *InMemoryCausalLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

func (l *InMemoryCausalLog) Replay(ctx context.Context) iter.Seq2[Entry, error] {
	return l.chain.Replay(ctx)
}

// ReorderingCausalLog validates deps on Append (same as InMemoryCausalLog)
// so action-level error comparison passes. But Replay returns entries in
// REVERSE order — violating causality. This exercises ReplayRespectsCausality
// law directly, not action-level mismatch.
type ReorderingCausalLog struct {
	chain   *refchain.AppendOnly[Entry]
	ids     map[string]bool
	entries []Entry
}

// NewReorderingCausalLog returns a CausalLog that reverses replay order.
func NewReorderingCausalLog() *ReorderingCausalLog {
	return &ReorderingCausalLog{
		chain: refchain.New[Entry](nil),
		ids:   make(map[string]bool),
	}
}

func (l *ReorderingCausalLog) Append(ctx context.Context, entry Entry) error {
	for _, dep := range entry.DependsOn {
		if !l.ids[dep] {
			return fmt.Errorf("dependency %s not found", dep)
		}
	}
	l.ids[entry.ID] = true
	l.entries = append(l.entries, entry)
	return l.chain.Append(ctx, entry)
}

func (l *ReorderingCausalLog) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

func (l *ReorderingCausalLog) Replay(_ context.Context) iter.Seq2[Entry, error] {
	// BUG: returns entries in REVERSE order — violates causality.
	return func(yield func(Entry, error) bool) {
		for i := len(l.entries) - 1; i >= 0; i-- {
			if !yield(l.entries[i], nil) {
				return
			}
		}
	}
}
