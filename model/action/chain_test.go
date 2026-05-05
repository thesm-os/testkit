// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"iter"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/action"
	"go.thesmos.sh/testkit/model/history"
	"go.thesmos.sh/testkit/model/refchain"
)

type entry struct {
	ID   string
	Data string
}

type chainIface interface {
	Append(context.Context, entry) error
	Replay(context.Context) iter.Seq2[entry, error]
	Verify(context.Context) error
}

func TestChainAppendRecording(t *testing.T) {
	t.Parallel()
	hist := history.New[struct{}, entry]()
	entryGen := rapid.Just(entry{ID: "test", Data: "data"})

	a := action.ChainAppendRecording(
		"Append", entryGen, hist,
		func(_ entry) struct{} { return struct{}{} },
		func(ctx context.Context, impl chainIface, e entry) error {
			return impl.Append(ctx, e)
		},
	)

	sut := refchain.New[entry](nil)
	ref := refchain.New[entry](nil)

	rapid.Check(t, func(rt *rapid.T) {
		a.Run(rt, sut, ref)
	})

	if hist.TotalLen() == 0 {
		t.Fatal("history should have recorded appends")
	}
}

func TestChainVerify(t *testing.T) {
	t.Parallel()

	a := action.ChainVerify("Verify",
		func(ctx context.Context, impl chainIface) error {
			return impl.Verify(ctx)
		},
	)

	sut := refchain.New[entry](nil)
	ref := refchain.New[entry](nil)
	_ = sut.Append(t.Context(), entry{ID: "1"})
	_ = ref.Append(t.Context(), entry{ID: "1"})

	rapid.Check(t, func(rt *rapid.T) {
		a.Run(rt, sut, ref)
		// Should not fatal — both chains are valid.
	})
}

func TestChainReplay(t *testing.T) {
	t.Parallel()
	hist := history.New[struct{}, entry]()
	hist.Record(struct{}{}, entry{ID: "1"})

	a := action.ChainReplay("Replay", hist,
		func(ctx context.Context, impl chainIface, _ struct{}) iter.Seq2[entry, error] {
			return impl.Replay(ctx)
		},
	)

	sut := refchain.New[entry](nil)
	ref := refchain.New[entry](nil)
	_ = sut.Append(t.Context(), entry{ID: "1"})
	_ = ref.Append(t.Context(), entry{ID: "1"})

	rapid.Check(t, func(rt *rapid.T) {
		a.Run(rt, sut, ref)
		// Should not fatal — both chains replay identically.
	})
}
