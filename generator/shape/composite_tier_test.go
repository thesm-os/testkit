// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
)

func TestPoolDetector(t *testing.T) {
	t.Parallel()

	t.Run("pool directive plus existing Put sibling promotes Get to Pool", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Conn struct{}
type I interface {
    Get(ctx context.Context) (Conn, error)
    Put(ctx context.Context, c Conn) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Get": {{Name: directive.Pool, Args: []string{"Put"}}},
		})
		testkit.Equal(t, got["Get"], "Pool", "Get promoted")
	})

	t.Run("missing Put sibling rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Conn struct{}
type I interface {
    Get(ctx context.Context) (Conn, error)
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Get": {{Name: directive.Pool, Args: []string{"Missing"}}},
		})
		testkit.Equal(t, got["Get"], "Aggregator", "no Put sibling → signature-tier")
	})
}

func TestCursorDetector(t *testing.T) {
	t.Parallel()

	t.Run("cursor directive plus Lifecycle Close promotes Next to Cursor", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Next() (string, error)
    Close(ctx context.Context) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Next": {{Name: directive.Cursor, Args: []string{"Close"}}},
		})
		testkit.Equal(t, got["Next"], "Cursor", "Next promoted")
	})

	t.Run("non-Lifecycle Close shape rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Next() (string, error)
    Close(ctx context.Context, k string) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Next": {{Name: directive.Cursor, Args: []string{"Close"}}},
		})
		testkit.Equal(t, got["Next"], "Aggregator", "Close isn't a finisher → fall back")
	})
}

func TestTwoPhaseDetector(t *testing.T) {
	t.Parallel()

	t.Run("two-phase directive plus Commit + Rollback siblings promotes Begin", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Tx struct{}
type I interface {
    Begin(ctx context.Context) (Tx, error)
    Commit(ctx context.Context, tx Tx) error
    Rollback(ctx context.Context, tx Tx) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Begin": {{Name: directive.TwoPhase, Args: []string{"Commit", "Rollback"}}},
		})
		testkit.Equal(t, got["Begin"], "TwoPhase", "Begin promoted")
	})

	t.Run("missing Rollback sibling rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Tx struct{}
type I interface {
    Begin(ctx context.Context) (Tx, error)
    Commit(ctx context.Context, tx Tx) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Begin": {{Name: directive.TwoPhase, Args: []string{"Commit", "Rollback"}}},
		})
		testkit.Equal(t, got["Begin"], "Aggregator", "missing Rollback → fall back")
	})

	t.Run("only one arg rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Tx struct{}
type I interface {
    Begin(ctx context.Context) (Tx, error)
    Commit(ctx context.Context, tx Tx) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Begin": {{Name: directive.TwoPhase, Args: []string{"Commit"}}},
		})
		testkit.Equal(t, got["Begin"], "Aggregator", "needs both Commit and Rollback")
	})
}

func TestSagaDetector(t *testing.T) {
	t.Parallel()

	t.Run("saga directive listing every step promotes the entry method", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Run(ctx context.Context) error
    Step1(ctx context.Context) error
    Step2(ctx context.Context) error
    Step3(ctx context.Context) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Run": {{Name: directive.Saga, Args: []string{"Step1", "Step2", "Step3"}}},
		})
		testkit.Equal(t, got["Run"], "Saga", "Run promoted")
	})

	t.Run("any missing step rejects the directive", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Run(ctx context.Context) error
    Step1(ctx context.Context) error
}
`
		got := classifyAllViaInterface(t, src, "I", map[string][]directive.Directive{
			"Run": {{Name: directive.Saga, Args: []string{"Step1", "Missing"}}},
		})
		testkit.Equal(t, got["Run"], "Lifecycle", "missing step → fall back")
	})
}
