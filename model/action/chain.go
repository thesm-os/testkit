// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action

import (
	"context"
	"fmt"
	"iter"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/history"
)

// ChainAppend creates an action for an //testkit:appends method.
// Draws an entry, calls SUT and ref, compares error outcomes.
func ChainAppend[T, Entry any](
	name string,
	entries *rapid.Generator[Entry],
	appendFn func(context.Context, T, Entry) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureStructural,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			e := entries.Draw(rt, name+"_entry")
			sutErr := appendFn(rt.Context(), sut, e)
			refErr := appendFn(rt.Context(), ref, e)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s: SUT err=%w, ref err=%w", name, sutErr, refErr),
					Input: e,
				}
			}
			return model.ActionResult{Input: e}
		},
	}
}

// ChainAppendRecording is like [ChainAppend] but also records
// successful appends into a [history.History] for dropped-write
// detection by [law.AppendOnlyNoDrops].
func ChainAppendRecording[T any, K comparable, Entry any](
	name string,
	entries *rapid.Generator[Entry],
	hist *history.History[K, Entry],
	partKeyOf func(Entry) K,
	appendFn func(context.Context, T, Entry) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureStructural,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			e := entries.Draw(rt, name+"_entry")
			sutErr := appendFn(rt.Context(), sut, e)
			refErr := appendFn(rt.Context(), ref, e)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s: SUT err=%w, ref err=%w", name, sutErr, refErr),
					Input: e,
				}
			}
			if sutErr == nil {
				hist.Record(partKeyOf(e), e)
			}
			return model.ActionResult{Input: e}
		},
	}
}

// ChainVerify creates an action for an //testkit:verifies method.
func ChainVerify[T any](
	name string,
	verify func(context.Context, T) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureStructural,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutErr := verify(rt.Context(), sut)
			refErr := verify(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s: SUT err=%w, ref err=%w", name, sutErr, refErr),
				}
			}
			return model.ActionResult{}
		},
	}
}

// ChainReplay creates a partition-aware action for an //testkit:replays
// method. Draws a partition key from the history's known partitions,
// drains both SUT and ref iter.Seq2, sorts, compares.
func ChainReplay[T any, K comparable, Entry any](
	name string,
	hist *history.History[K, Entry],
	replay func(context.Context, T, K) iter.Seq2[Entry, error],
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureStructural,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			parts := hist.Partitions()
			if len(parts) == 0 {
				return model.ActionResult{} // nothing appended yet
			}
			partKey := rapid.SampledFrom(parts).Draw(rt, name+"_partition")

			sutEntries, sutErr := drainSeq2(replay(rt.Context(), sut, partKey))
			refEntries, refErr := drainSeq2(replay(rt.Context(), ref, partKey))
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s[%v]: SUT err=%w, ref err=%w", name, partKey, sutErr, refErr),
					Input:  partKey,
					Output: sutEntries,
				}
			}
			if sutErr == nil {
				sortByString(sutEntries)
				sortByString(refEntries)
				if diff := cmp.Diff(refEntries, sutEntries); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s[%v]: SUT/ref disagree:\n%s", name, partKey, diff),
						Input:  partKey,
						Output: sutEntries,
					}
				}
			}
			return model.ActionResult{Input: partKey, Output: sutEntries}
		},
	}
}

func drainSeq2[Entry any](seq iter.Seq2[Entry, error]) ([]Entry, error) {
	var out []Entry
	for e, err := range seq {
		if err != nil {
			return out, err
		}
		out = append(out, e)
	}
	return out, nil
}
