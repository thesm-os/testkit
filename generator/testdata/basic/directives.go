// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import "context"

// Workflow exercises the full directive vocabulary the stub
// generator consumes — one method per directive (or directive
// family) so the spec consumers can be tested independently.
//
// The directives reference symbols already declared elsewhere in
// the basic package (ErrNotFound, ErrConflict in errors.go) and the
// storage sibling pkg (storage.SampleKey for cross-package tests
// elsewhere). Stub-irrelevant directives (atomic, idempotent, ...)
// stay off this fixture so consumers can be exercised in isolation.
type Workflow interface {
	// Open uses no directives. Acts as the order-after target.
	Open(ctx context.Context) error

	// Close is integration-only — stub skips it entirely.
	//
	//testkit:integration-only
	Close(ctx context.Context) error

	// Submit declares the sentinels it may return; the stub emits
	// FaultErrNotFound + FaultErrConflict helpers.
	//
	//testkit:errors ErrNotFound ErrConflict
	Submit(ctx context.Context, item Item) error

	// Wrap returns errors wrapped via ErrForbidden — the stub's
	// fault helpers wrap their sentinel via this outer error.
	//
	//testkit:wrapped-via ErrForbidden
	Wrap(ctx context.Context, key string) error

	// Legacy is deprecated; the stub emits a Deprecated annotation
	// pointing callers at Submit.
	//
	//testkit:deprecated Submit
	Legacy(ctx context.Context, item Item) error

	// Retry succeeds on the third call; the stub fails the first
	// two then returns the configured success.
	//
	//testkit:retry-succeeds-on-attempt 3
	Retry(ctx context.Context, key string) error

	// Read enforces ordering: must be called after Open. The stub
	// panics at runtime when called out of order.
	//
	//testkit:order-after Open
	Read(ctx context.Context, key string) (Item, error)

	// Shard isolates recordings by Item.ID — different IDs get
	// independent call recorders.
	//
	//testkit:partition ID
	Shard(ctx context.Context, item Item) error

	// ShardByKey isolates by the direct `key` parameter rather
	// than a struct field — exercises the partition consumer's
	// direct-param resolution branch.
	//
	//testkit:partition key
	ShardByKey(ctx context.Context, key string) error

	// Batch demonstrates a variadic method — used by the partition
	// consumer's variadic-skip branch. The trailing variadic param
	// must be excluded from the param walk because its type is a
	// slice, not the underlying element.
	//
	//testkit:partition tenant
	Batch(ctx context.Context, tenant string, items ...Item) error

	// ShardAnon has anonymous params (no name) and a non-struct
	// param the partition walker must skip before reaching the
	// struct param holding the partition field. Exercises both the
	// non-struct-skip branch and the ParamName fallback for unnamed
	// params.
	//
	//testkit:partition ID
	ShardAnon(context.Context, string, Item) error
}
