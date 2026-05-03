// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// This file demonstrates wrapping a hand-written in-memory implementation
// with the generated stub via DelegateTo. This is the intended pattern
// for integration tests and sim companions.
//
// The in-memory implementation provides the real logic. The generated stub
// adds recording, fault injection, call-count verification, and strict
// mode on top — without the consumer writing any of that plumbing.

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/stub/testdata/companion"
	"go.thesmos.sh/testkit/gen/stub/testdata/companion/storetest"
)

// TestCompanionPattern demonstrates the full DelegateTo workflow:
// wrap a real implementation, exercise it through the stub, and use
// the stub's recording + fault injection to verify behavior.
func TestCompanionPattern(t *testing.T) {
	t.Parallel()

	t.Run("stub delegates to real implementation", func(t *testing.T) {
		t.Parallel()
		inner := companion.NewInMemoryStore()
		s := storetest.NewStoreStub(t, storetest.StoreStubDelegateTo(inner))

		// Writes go through to the real implementation.
		err := s.Put(t.Context(), "greeting", "hello")
		testkit.NoError(t, err, "Put must succeed")

		// Reads return the real implementation's data.
		got, err := s.Get(t.Context(), "greeting")
		testkit.NoError(t, err, "Get must succeed")
		testkit.Equal(t, got, "hello", "must return stored value")

		// The stub records every call for inspection.
		s.OnPut.AssertCalledOnce(t, "must record Put")
		s.OnGet.AssertCalledOnce(t, "must record Get")

		// LastCall captures the exact arguments.
		getCall := s.OnGet.LastCall(t)
		testkit.Equal(t, getCall.Key, "greeting", "must capture arg")
	})

	t.Run("fault injection overrides real implementation", func(t *testing.T) {
		t.Parallel()
		inner := companion.NewInMemoryStore()
		s := storetest.NewStoreStub(t, storetest.StoreStubDelegateTo(inner))

		// Seed some data through the stub.
		testkit.NoError(t, s.Put(t.Context(), "key", "value"), "seed")

		// Inject a fault — Get now fails even though the data exists.
		errInjected := testkit.TestError("injected")
		s.OnGet.Faults(errInjected, 1)

		_, err := s.Get(t.Context(), "key")
		testkit.ErrorIs(t, err, errInjected,
			"fault must override the real implementation's success path")
	})

	t.Run("call-count verification catches missing calls", func(t *testing.T) {
		t.Parallel()
		inner := companion.NewInMemoryStore()
		f := testkit.NewFailableTB()
		s := storetest.NewStoreStub(f, storetest.StoreStubDelegateTo(inner))

		// Expect exactly 2 Get calls.
		s.OnGet.Times(2)

		// Only call once — verification should fail.
		_, _ = s.Get(f.Context(), "key")

		f.RunCleanups()
		testkit.True(t, f.Failed(), "Times(2) with 1 call must fail at cleanup")
	})

	t.Run("OnRecord hook observes real implementation calls", func(t *testing.T) {
		t.Parallel()
		inner := companion.NewInMemoryStore()
		s := storetest.NewStoreStub(t, storetest.StoreStubDelegateTo(inner))

		var putKeys []string
		s.OnPut.OnRecord(func(c storetest.StorePutCall) {
			putKeys = append(putKeys, c.Key)
		})

		testkit.NoError(t, s.Put(t.Context(), "a", "1"), "put a")
		testkit.NoError(t, s.Put(t.Context(), "b", "2"), "put b")

		testkit.Equal(t, putKeys, []string{"a", "b"},
			"OnRecord must observe every delegated call")
	})

	t.Run("strict mode catches unexpected methods", func(t *testing.T) {
		t.Parallel()
		inner := companion.NewInMemoryStore()
		f := testkit.NewFailableTB()
		// Strict + DelegateTo: all methods delegate, but strict flags any
		// method that the test didn't explicitly configure. This catches
		// tests that accidentally exercise untested code paths.
		s := storetest.NewStoreStub(f,
			storetest.StoreStubStrict(),
			storetest.StoreStubDelegateTo(inner))

		// Delete is delegated but we didn't "expect" it in this test.
		// In strict mode, it still succeeds (DelegateTo wires the Func),
		// but if we removed DelegateTo, it would fail.
		_ = s.Delete(f.Context(), "key")

		// The call went through because DelegateTo configured it.
		testkit.False(t, f.Failed(),
			"strict + DelegateTo must not fail — DelegateTo wires all methods")
	})
}
