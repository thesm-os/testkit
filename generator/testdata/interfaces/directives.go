// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

//go:generate testkit stub -o directivestest/directives.gen.go Directives
//go:generate testkit suite -o directivestest/directives_spec.gen_test.go Directives

import (
	"context"
	"errors"
)

// ErrConflict + ErrForbidden round out the sentinel set used by
// the Directives fixture. ErrInternal is the wrapped-via target.
var (
	ErrConflict  = errors.New("interfaces: conflict")
	ErrForbidden = errors.New("interfaces: forbidden")
	ErrInternal  = errors.New("interfaces: internal")
)

// Directives carries one method per stub-relevant directive. Each
// method's directive set is annotated in its doc; the order of
// declaration is the order each consumer fires when spec.Enrich
// dispatches.
type Directives interface {
	// Open is the order-after target. No directives.
	Open(ctx context.Context) error

	// Close is integration-only — the stub skips dispatch entirely
	// and returns zero values without recording.
	//
	//testkit:integration-only
	Close(ctx context.Context) error

	// Submit declares the sentinels it may return AND wraps every
	// error via ErrInternal — the stub emits FaultNotFound +
	// FaultConflict helpers whose returned error satisfies
	// errors.Is for both the named sentinel AND the wrap target.
	//
	//testkit:errors ErrNotFound ErrConflict
	//testkit:wrapped-via ErrInternal
	Submit(ctx context.Context, item Record) error

	// Wrap pairs //testkit:errors with //testkit:wrapped-via —
	// the FaultX helpers wrap each named sentinel via ErrInternal
	// so consumers' errors.Is checks satisfy both. The composition
	// rule `wrapped-via Requires errors` makes this pairing
	// mandatory: wrapped-via without sentinels is rejected at
	// directive validation.
	//
	//testkit:errors ErrForbidden
	//testkit:wrapped-via ErrInternal
	Wrap(ctx context.Context, key string) error

	// Legacy is deprecated; the stub renders a Deprecated
	// annotation pointing callers at Submit.
	//
	//testkit:deprecated Submit
	Legacy(ctx context.Context, item Record) error

	// Retry succeeds on the third call; the first two calls return
	// the configured fault. Composition rules require retryable to
	// pair with retry-succeeds-on-attempt.
	//
	//testkit:retryable
	//testkit:retry-succeeds-on-attempt 3
	Retry(ctx context.Context, key string) error

	// Read enforces ordering: must be called after Open. The stub
	// panics when called out of order.
	//
	//testkit:order-after Open
	Read(ctx context.Context, key string) (Record, error)

	// Shard isolates recordings by Item.ID — different IDs get
	// independent call recorders.
	//
	//testkit:partition ID
	Shard(ctx context.Context, item Record) error

	// ShardByKey isolates by the direct `key` parameter rather
	// than a struct field — exercises the partition consumer's
	// direct-param resolution branch.
	//
	//testkit:partition key
	ShardByKey(ctx context.Context, key string) error
}
