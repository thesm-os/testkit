// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/basic"
	"go.thesmos.sh/testkit/gen/model/testdata/basic/storetest"
	"go.thesmos.sh/testkit/model/law"
)

func TestInMemoryStoreModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 zero code", func(t *testing.T) {
		t.Parallel()
		// Tier 0: auto-synthesized reference (refmap.MapStore),
		// auto-derived actions, auto-derived ReadAfterWrite +
		// CountEqualsReference laws. No consumer code beyond the factory.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		})
	})

	t.Run("tier 2 custom law", func(t *testing.T) {
		t.Parallel()
		// Tier 2: add a domain-specific law. Count must never go negative.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		},
			storetest.StoreModelLaw(countNonNegative{}),
		)
	})

	t.Run("skip auto law", func(t *testing.T) {
		t.Parallel()
		// Opt out of an auto-derived law by ID.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		},
			storetest.StoreModelSkipLaw("AUTO-COUNT-EQUALS-REFERENCE"),
		)
	})

	t.Run("concurrent linearizability", func(t *testing.T) {
		t.Parallel()
		// Concurrent: 4 workers × 30 ops. Porcupine validates
		// linearizability of Get/Put/Delete across interleaved histories.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		},
			storetest.StoreModelConcurrent(4, 30),
		)
	})

	t.Run("concurrent catches non-linearizable store", func(t *testing.T) {
		t.Parallel()
		// Negative: NonLinearizableStore is thread-safe but Get reads
		// stale data. Porcupine must report Illegal via the generated API.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			storetest.AssertStoreModel(ft, func() basic.Store {
				return basic.NewNonLinearizableStore()
			},
				storetest.StoreModelConcurrent(4, 30),
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("concurrent negative test timed out")
		}
		if !ft.Failed() {
			t.Fatal("non-linearizable store should fail Porcupine check via generated API")
		}
	})
}

// countNonNegative is a Tier 2 domain-specific law: Count >= 0 always.
type countNonNegative struct{}

func (countNonNegative) ID() string    { return "CUSTOM-COUNT-NON-NEGATIVE" }
func (countNonNegative) REQID() string { return "" }

func (countNonNegative) Check(rt *rapid.T, sut, ref basic.Store) error {
	n, err := sut.Count(rt.Context())
	if err != nil {
		return err
	}
	if n < 0 {
		return fmt.Errorf("count is negative: %d", n)
	}
	return nil
}

// Compile-time check that countNonNegative satisfies Law[basic.Store].
var _ law.Law[basic.Store] = countNonNegative{}

func TestWithoutTrace(t *testing.T) {
	t.Parallel()
	// Verifies that WithoutTrace doesn't crash and still catches bugs.
	storetest.AssertStoreModel(t, func() basic.Store {
		return basic.NewInMemoryStore()
	},
		storetest.StoreModelWithoutTrace(),
	)
}

func TestArtifactDir(t *testing.T) {
	t.Parallel()
	// Verifies that WithArtifactDir + concurrent negative test writes
	// the Porcupine HTML artifact to the specified directory.
	artifactDir := t.TempDir()
	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		storetest.AssertStoreModel(ft, func() basic.Store {
			return basic.NewNonLinearizableStore()
		},
			storetest.StoreModelConcurrent(4, 30),
			storetest.StoreModelArtifactDir(artifactDir),
		)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("artifact test timed out")
	}
	if !ft.Failed() {
		t.Fatal("non-linearizable store should fail")
	}
	// Verify artifact was written.
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		t.Fatalf("reading artifact dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-linearizability.html") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected linearizability HTML artifact in %s, got: %v", artifactDir, entries)
	}
}
