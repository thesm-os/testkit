// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Action errors are diagnostic (SUT vs ref comparison), not wrapped.
package action

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model"
)

// Persister creates an action for a Persister-shaped method:
// `func(ctx, V) (ID, error)`. The returned ID is the lookup key
// for the paired sibling Reader. The action draws a value, calls
// both SUT and ref, and compares the returned IDs along with the
// error outcomes.
func Persister[T, V any, ID comparable](
	name string,
	values *rapid.Generator[V],
	save func(context.Context, T, V) (ID, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			v := values.Draw(rt, name+"_value")
			sutID, sutErr := save(rt.Context(), sut, v)
			refID, refErr := save(rt.Context(), ref, v)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, v, sutErr, refErr),
					Input:  v,
					Output: sutID,
				}
			}
			if sutErr == nil && sutID != refID {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT id=%v, ref id=%v", name, v, sutID, refID),
					Input:  v,
					Output: sutID,
				}
			}
			return model.ActionResult{Input: v, Output: sutID}
		},
	}
}

// Updater creates an action for an Updater-shaped method:
// `func(ctx, V) error` or `func(ctx, K, V) error`. Carrier shape
// is identical to Writer/CompositeWriter; the contract-tier
// distinction is enforced by laws over the paired sibling Reader.
// The action just compares error outcomes.
func Updater[T, V any](
	name string,
	values *rapid.Generator[V],
	update func(context.Context, T, V) error,
) model.Action[T] {
	return Writer(name, values, update)
}

// Upserter creates an action for an Upserter-shaped method:
// `func(ctx, V) error`. Idempotent insert-or-update; carrier shape
// matches Writer. The contract-tier distinction (idempotence) is
// enforced by laws.
func Upserter[T, V any](
	name string,
	values *rapid.Generator[V],
	upsert func(context.Context, T, V) error,
) model.Action[T] {
	return Writer(name, values, upsert)
}

// CompareAndSwap creates an action for a CompareAndSwap-shaped
// method: `func(ctx, V) error` where V carries a version field.
// Carrier shape matches Writer. The version-mismatch contract is
// enforced by laws and concurrent linearizability checks.
func CompareAndSwap[T, V any](
	name string,
	values *rapid.Generator[V],
	cas func(context.Context, T, V) error,
) model.Action[T] {
	return Writer(name, values, cas)
}

// Appender creates an action for an Appender-shaped method:
// `func(ctx, V) (Offset, error)`. The returned offset is asserted
// monotonic by a law; the action compares offsets and error
// outcomes per call.
func Appender[T, V any, Offset comparable](
	name string,
	values *rapid.Generator[V],
	appendFn func(context.Context, T, V) (Offset, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			v := values.Draw(rt, name+"_value")
			sutOff, sutErr := appendFn(rt.Context(), sut, v)
			refOff, refErr := appendFn(rt.Context(), ref, v)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, v, sutErr, refErr),
					Input:  v,
					Output: sutOff,
				}
			}
			if sutErr == nil && sutOff != refOff {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT offset=%v, ref offset=%v", name, v, sutOff, refOff),
					Input:  v,
					Output: sutOff,
				}
			}
			return model.ActionResult{Input: v, Output: sutOff}
		},
	}
}

// Watcher creates an action for a Watcher-shaped method that returns
// a notification channel. The action just exercises the watcher path
// — channel comparison is intractable across impls, so the contract
// (notification on trigger) is enforced by laws that bind the
// watcher with its trigger sibling.
//
// Both SUT and ref are invoked; the action returns success if both
// produce a non-nil channel and matching error outcomes.
func Watcher[T, V any](
	name string,
	open func(context.Context, T) (<-chan V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutCh, sutErr := open(rt.Context(), sut)
			refCh, refErr := open(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
				}
			}
			if sutErr == nil && (sutCh == nil) != (refCh == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s: SUT chan nil=%v, ref chan nil=%v", name, sutCh == nil, refCh == nil),
				}
			}
			return model.ActionResult{}
		},
	}
}

// Paginator creates an action for a Paginator-shaped method:
// `func(ctx, Cursor) ([]V, Cursor, error)`. The action drains the
// paginator from a drawn starting cursor, collecting every page
// from both SUT and ref, then compares the concatenated result.
//
// PageLimit caps the drain loop to prevent runaway iteration on a
// buggy paginator that never terminates.
func Paginator[T any, Cursor comparable, V any](
	name string,
	cursors *rapid.Generator[Cursor],
	page func(context.Context, T, Cursor) ([]V, Cursor, error),
	pageLimit int,
) model.Action[T] {
	if pageLimit <= 0 {
		pageLimit = 100
	}
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			start := cursors.Draw(rt, name+"_cursor")
			var zero Cursor

			drain := func(impl T) ([]V, error) {
				var all []V
				cur := start
				for i := 0; i < pageLimit; i++ {
					items, next, err := page(rt.Context(), impl, cur)
					if err != nil {
						return all, err
					}
					all = append(all, items...)
					if next == zero {
						return all, nil
					}
					cur = next
				}
				return all, fmt.Errorf("%s: page limit %d exceeded", name, pageLimit)
			}

			sutAll, sutErr := drain(sut)
			refAll, refErr := drain(ref)
			// Drain errors (page-limit overflow, mid-drain failure)
			// are always faults: a paginator that doesn't terminate
			// or that errors mid-drain is a bug regardless of impl
			// agreement. Surface unconditionally.
			if sutErr != nil || refErr != nil {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Input:  start,
					Output: sutAll,
				}
			}
			if diff := cmp.Diff(refAll, sutAll); diff != "" {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT/ref disagree:\n%s", name, diff),
					Input:  start,
					Output: sutAll,
				}
			}
			return model.ActionResult{Input: start, Output: sutAll}
		},
	}
}

// GetOrCompute creates an action for a GetOrCompute-shaped method:
// `func(ctx, K, func() V) (V, error)`. Both SUT and ref receive the
// same compute callback, so single-flight coalescing semantics are
// observable via the call counter the consumer threads through the
// callback. The action compares returned values and error outcomes.
func GetOrCompute[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	compute func() V,
	call func(context.Context, T, K, func() V) (V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutGot, sutErr := call(rt.Context(), sut, k, compute)
			refGot, refErr := call(rt.Context(), ref, k, compute)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, k, sutErr, refErr),
					Input:  k,
					Output: sutGot,
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v): SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: sutGot,
					}
				}
			}
			return model.ActionResult{Input: k, Output: sutGot}
		},
	}
}

// TransactionFunc creates an action for a TransactionFunc-shaped
// method: `func(ctx, func(Tx) error) error`. The body callback is
// invoked under the SUT's transaction and the ref's transaction
// independently; the action compares the outer-error outcomes.
//
// Tx-internal observations are made by the body callback itself
// (passed in by the consumer), which can record what state the
// transaction sees and assert per-tx invariants.
func TransactionFunc[T, Tx any](
	name string,
	body func(Tx) error,
	call func(context.Context, T, func(Tx) error) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutErr := call(rt.Context(), sut, body)
			refErr := call(rt.Context(), ref, body)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err: fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
				}
			}
			return model.ActionResult{}
		},
	}
}

// AcquireLease creates an action for an AcquireLease-shaped method:
// `func(ctx) error`. Carrier shape matches Lifecycle; the contract
// (double-acquire blocks/errors, release on cancel) is enforced by
// laws that bind the acquire with its release sibling.
func AcquireLease[T any](
	name string,
	acquire func(context.Context, T) error,
) model.Action[T] {
	return Lifecycle(name, acquire)
}

// Publisher creates an action for a Publisher-shaped method:
// `func(ctx, Msg) error`. Carrier shape matches Writer; delivery
// semantics are enforced by laws that pair the publisher with its
// subscriber.
func Publisher[T, Msg any](
	name string,
	messages *rapid.Generator[Msg],
	publish func(context.Context, T, Msg) error,
) model.Action[T] {
	return Writer(name, messages, publish)
}

// Subscriber creates an action for a Subscriber-shaped method that
// returns a delivery channel. Like [Watcher], the action just
// exercises the channel-open path — delivery semantics are
// enforced by laws.
func Subscriber[T, Msg any](
	name string,
	subscribe func(context.Context, T) (<-chan Msg, error),
) model.Action[T] {
	return Watcher(name, subscribe)
}
