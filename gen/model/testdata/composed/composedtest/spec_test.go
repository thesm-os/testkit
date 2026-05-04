// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package composedtest_test

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/composed"
	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/action"
	"go.thesmos.sh/testkit/model/law"
)

// linked is the composed type for Store + Ledger.
type linked = model.Pair[composed.Store, composed.Ledger]

func TestStoreLedgerComposed(t *testing.T) {
	t.Parallel()

	t.Run("cross-interface invariant holds", func(t *testing.T) {
		t.Parallel()
		// Linked composition: Store.Put appends to the shared Ledger.
		// Cross-law checks that SUT's Ledger.Len matches ref's.
		model.Assert(t, linkedFactory,
			model.WithReference(linkedFactory),
			model.WithActions(composedActions()...),
			model.WithLaws(composedLaws()),
		)
	})

	t.Run("catches broken Store that skips Put append", func(t *testing.T) {
		t.Parallel()
		// Negative test: brokenStore.Put doesn't append to Ledger.
		// The cross-law must detect the mismatch.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			model.Assert(ft, brokenPutFactory,
				model.WithReference(linkedFactory),
				model.WithActions(composedActions()...),
				model.WithLaws(composedLaws()),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("cross-law should have caught missing Put→Ledger append")
		}
	})

	t.Run("catches broken Store that skips Delete append", func(t *testing.T) {
		t.Parallel()
		// Negative test: brokenStore.Delete doesn't append to Ledger.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			model.Assert(ft, brokenDeleteFactory,
				model.WithReference(linkedFactory),
				model.WithActions(composedActions()...),
				model.WithLaws(composedLaws()),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("cross-law should have caught missing Delete→Ledger append")
		}
	})
}

// linkedFactory creates a Store wired to a shared Ledger.
func linkedFactory() linked {
	ledger := composed.NewInMemoryLedger()
	store := composed.NewInMemoryStore(ledger)
	return linked{A: store, B: ledger}
}

// brokenPutFactory creates a Store that silently skips ledger appends on Put.
func brokenPutFactory() linked {
	ledger := composed.NewInMemoryLedger()
	store := &brokenPutStore{data: make(map[string]composed.Item), ledger: ledger}
	return linked{A: store, B: ledger}
}

// brokenDeleteFactory creates a Store that silently skips ledger appends on Delete.
func brokenDeleteFactory() linked {
	ledger := composed.NewInMemoryLedger()
	store := &brokenDeleteStore{data: make(map[string]composed.Item), ledger: ledger}
	return linked{A: store, B: ledger}
}

func composedActions() []model.Action[linked] {
	keyGen := rapid.SampledFrom([]string{"a", "b", "c"})
	valGen := rapid.MakeCustom[composed.Item](rapid.MakeConfig{
		Fields: map[reflect.Type]map[string]*rapid.Generator[any]{
			reflect.TypeOf(composed.Item{}): {
				"ID": keyGen.AsAny(),
			},
		},
	})

	// Lift single-interface actions into composed actions via LiftA/LiftB.
	return []model.Action[linked]{
		model.LiftA[composed.Store, composed.Ledger](
			action.Reader("Get", keyGen,
				func(ctx context.Context, s composed.Store, k string) (composed.Item, error) {
					return s.Get(ctx, k)
				},
			),
		),
		model.LiftA[composed.Store, composed.Ledger](
			action.Writer("Put", valGen,
				func(ctx context.Context, s composed.Store, v composed.Item) error {
					return s.Put(ctx, v)
				},
			),
		),
		model.LiftA[composed.Store, composed.Ledger](
			action.Deleter("Delete", keyGen,
				func(ctx context.Context, s composed.Store, k string) error {
					return s.Delete(ctx, k)
				},
			),
		),
		model.LiftB[composed.Store, composed.Ledger](
			action.Aggregator("Len",
				func(ctx context.Context, l composed.Ledger) (int, error) {
					return l.Len(ctx)
				},
			),
		),
	}
}

func composedLaws() *model.Registry[linked] {
	reg := model.NewRegistry[linked]()
	reg.Add(ledgerCountMatchesWrites{})
	return reg
}

// ledgerCountMatchesWrites checks that the SUT's Ledger.Len matches
// the reference's Ledger.Len. Since Store.Put and Store.Delete each
// append to the linked Ledger, this cross-interface law detects when
// the Store fails to record a write.
type ledgerCountMatchesWrites struct{}

func (ledgerCountMatchesWrites) ID() string    { return "CROSS-LEDGER-COUNTS-WRITES" }
func (ledgerCountMatchesWrites) REQID() string { return "" }

func (ledgerCountMatchesWrites) Check(rt *rapid.T, sut, ref linked) error {
	sutLen, sutErr := sut.B.Len(rt.Context())
	refLen, refErr := ref.B.Len(rt.Context())
	if sutErr != nil || refErr != nil {
		return fmt.Errorf("Ledger.Len: SUT err=%v, ref err=%v", sutErr, refErr)
	}
	if sutLen != refLen {
		return fmt.Errorf("Ledger.Len: SUT=%d, ref=%d", sutLen, refLen)
	}
	return nil
}

var _ law.Law[linked] = ledgerCountMatchesWrites{}

// brokenPutStore skips ledger append on Put. Delete works correctly.
type brokenPutStore struct {
	mu     sync.Mutex
	data   map[string]composed.Item
	ledger composed.Ledger
}

func (s *brokenPutStore) Get(_ context.Context, id string) (composed.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return composed.Item{}, composed.ErrNotFound
	}
	return v, nil
}

func (s *brokenPutStore) Put(_ context.Context, item composed.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	// BUG: silently skips ledger append
	return nil
}

func (s *brokenPutStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return s.ledger.Append(ctx, composed.Entry{ItemID: id, Action: "delete"})
}

// brokenDeleteStore skips ledger append on Delete. Put works correctly.
type brokenDeleteStore struct {
	mu     sync.Mutex
	data   map[string]composed.Item
	ledger composed.Ledger
}

func (s *brokenDeleteStore) Get(_ context.Context, id string) (composed.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return composed.Item{}, composed.ErrNotFound
	}
	return v, nil
}

func (s *brokenDeleteStore) Put(ctx context.Context, item composed.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	return s.ledger.Append(ctx, composed.Entry{ItemID: item.ID, Action: "put"})
}

func (s *brokenDeleteStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	// BUG: silently skips ledger append
	return nil
}
