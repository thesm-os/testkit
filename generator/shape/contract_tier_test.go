// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

// Contract-tier detectors all share the same shape:
//
//   - Single-method [shape.Classify] never fires them (Interface ctx
//     is nil, so the detector returns false).
//   - [shape.ClassifyInterface] supplies the InterfaceContext, the
//     directive is read off the carrier method, sibling existence/
//     shape is verified, and the contract-tier shape replaces the
//     pass-1 signature-tier result.
//
// One subtest per detector exercises the directive-driven happy path
// plus a representative reject path.

func TestPersisterDetector(t *testing.T) {
	t.Parallel()

	t.Run("Writer-with-result + sibling Reader promotes to Persister", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Save(ctx context.Context, v string) (int, error)
    Get(ctx context.Context, id int) (string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Save": {{Name: directive.Persister, Args: []string{"Get"}}},
		})
		testkit.Equal(t, got["Save"], "Persister", "Save promoted")
		testkit.Equal(t, got["Get"], "Reader", "Get unchanged")
	})

	t.Run("missing sibling rejects the promotion", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Save(ctx context.Context, v string) (int, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Save": {{Name: directive.Persister, Args: []string{"Missing"}}},
		})
		testkit.Equal(t, got["Save"], "Reader", "no sibling → falls back to signature-tier")
	})
}

func TestUpdaterDetector(t *testing.T) {
	t.Parallel()

	t.Run("Writer + sibling Reader promotes to Updater", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Item struct{ ID string }
type I interface {
    Update(ctx context.Context, item Item) error
    Get(ctx context.Context, id string) (Item, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Update": {{Name: directive.Updater, Args: []string{"Get"}}},
		})
		testkit.Equal(t, got["Update"], "Updater", "Update promoted")
	})

	t.Run("non-writer carrier rejects the promotion", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface {
    Update(k string) string
    Get(k string) string
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Update": {{Name: directive.Updater, Args: []string{"Get"}}},
		})
		testkit.Equal(t, got["Update"], "ReaderNoError", "no error return → not a Writer")
	})
}

func TestUpserterDetector(t *testing.T) {
	t.Parallel()

	t.Run("Writer + sibling Reader promotes to Upserter", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Item struct{ ID string }
type I interface {
    Upsert(ctx context.Context, item Item) error
    Get(ctx context.Context, id string) (Item, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Upsert": {{Name: directive.Upserter, Args: []string{"Get"}}},
		})
		testkit.Equal(t, got["Upsert"], "Upserter", "Upsert promoted")
	})
}

func TestCASDetector(t *testing.T) {
	t.Parallel()

	t.Run("Writer carrier promotes to CompareAndSwap", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Item struct{ ID, Version string }
type I interface {
    Put(ctx context.Context, item Item) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Put": {{Name: directive.CAS, Args: []string{"Version"}}},
		})
		testkit.Equal(t, got["Put"], "CompareAndSwap", "promoted")
	})

	t.Run("missing version-field arg rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Item struct{ ID string }
type I interface {
    Put(ctx context.Context, item Item) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Put": {{Name: directive.CAS}},
		})
		testkit.Equal(t, got["Put"], "Writer", "no arg → falls back")
	})
}

func TestAppenderDetector(t *testing.T) {
	t.Parallel()

	t.Run("single-input single-result with directive promotes to Appender", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Append(ctx context.Context, e string) (int, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Append": {{Name: directive.Appender}},
		})
		testkit.Equal(t, got["Append"], "Appender", "promoted")
	})

	t.Run("Off variant rejects the promotion", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Append(ctx context.Context, e string) (int, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Append": {{Name: directive.Appender, Off: true}},
		})
		testkit.Equal(t, got["Append"], "Reader", "off → signature-tier stands")
	})
}

func TestWatcherDetector(t *testing.T) {
	t.Parallel()

	t.Run("directive plus existing Trigger sibling promotes to Watcher", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Watch(ctx context.Context) (string, error)
    Set(ctx context.Context, v string) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Watch": {{Name: directive.Watcher, Args: []string{"Set"}}},
		})
		testkit.Equal(t, got["Watch"], "Watcher", "promoted")
	})

	t.Run("missing Trigger sibling rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Watch(ctx context.Context) (string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Watch": {{Name: directive.Watcher, Args: []string{"Missing"}}},
		})
		testkit.Equal(t, got["Watch"], "Aggregator", "no sibling → signature-tier stands")
	})
}

func TestPaginatorDetector(t *testing.T) {
	t.Parallel()

	t.Run("Reader carrier with pagination directive promotes to Paginator", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    List(ctx context.Context, cursor string) ([]string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"List": {{Name: directive.Pagination, Args: []string{"cursor"}}},
		})
		testkit.Equal(t, got["List"], "Paginator", "promoted")
	})
}

func TestGetOrComputeDetector(t *testing.T) {
	t.Parallel()

	t.Run("singleflight directive promotes to GetOrCompute", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Compute(ctx context.Context, k string, fn func() string) (string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Compute": {{Name: directive.Singleflight}},
		})
		testkit.Equal(t, got["Compute"], "GetOrCompute", "promoted")
	})
}

func TestTransactionFuncDetector(t *testing.T) {
	t.Parallel()

	t.Run("transaction directive promotes to TransactionFunc", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Tx interface{ Commit() error }
type I interface {
    InTx(ctx context.Context, fn func(Tx) error) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"InTx": {{Name: directive.Transaction}},
		})
		testkit.Equal(t, got["InTx"], "TransactionFunc", "promoted")
	})
}

func TestAcquireLeaseDetector(t *testing.T) {
	t.Parallel()

	t.Run("acquire directive plus Lifecycle Release promotes to AcquireLease", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Acquire(ctx context.Context) error
    Release(ctx context.Context) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Acquire": {{Name: directive.Acquire, Args: []string{"Release"}}},
		})
		testkit.Equal(t, got["Acquire"], "AcquireLease", "promoted")
	})

	t.Run("non-Lifecycle release shape rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Acquire(ctx context.Context) error
    Release(ctx context.Context, k string) (string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Acquire": {{Name: directive.Acquire, Args: []string{"Release"}}},
		})
		testkit.Equal(t, got["Acquire"], "Lifecycle", "release isn't a finisher → fall back")
	})
}

func TestPublisherDetector(t *testing.T) {
	t.Parallel()

	t.Run("publisher directive plus Subscribe sibling promotes to Publisher", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Publish(ctx context.Context, msg string) error
    Sub(ctx context.Context) (<-chan string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Publish": {{Name: directive.Publisher, Args: []string{"Sub"}}},
		})
		testkit.Equal(t, got["Publish"], "Publisher", "promoted")
	})
}

func TestSubscriberDetector(t *testing.T) {
	t.Parallel()

	t.Run("subscribe directive promotes to Subscriber", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Subscribe(ctx context.Context) (<-chan string, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Subscribe": {{Name: directive.Subscribe}},
		})
		testkit.Equal(t, got["Subscribe"], "Subscriber", "promoted")
	})
}

func TestContractTierGatesOnInterfaceContext(t *testing.T) {
	t.Parallel()

	t.Run("single-method Classify ignores contract-tier detectors", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    F(ctx context.Context, v string) (int, error)
}
`
		// classifyOne goes through Classify (no Interface ctx), so the
		// persister directive does NOT fire — falls through to Reader
		// at the signature tier.
		got := classifyOne(t, src, directive.Directive{Name: directive.Persister, Args: []string{"Get"}})
		testkit.Equal(t, got, "Reader", "no Interface ctx → contract-tier silent")
	})
}
