// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestGenerateStub(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "stub")
	cfg := DefaultConfig()

	t.Run("generates stub for Store interface", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateStub(pkg, []string{"Store"}, cfg, Options{
			Output: "stubtest/in_memory_store.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		testkit.Len(t, result.Files, 2, "must produce impl + test files")

		impl := string(result.Files[0].Content)

		// Interface check.
		testkit.Assert(t, impl).Contains("var _ stub.Store = (*StoreStub)(nil)", "must have compile-time check")

		// Call types — prefixed with interface name.
		testkit.Assert(t, impl).
			Contains("type StoreGetCall struct", "must have StoreGetCall").
			Contains("type StorePutCall struct", "must have StorePutCall").
			Contains("type StoreDeleteCall struct", "must have StoreDeleteCall").
			Contains("type StoreListCall struct", "must have StoreListCall")

		// Per-method stubs — prefixed with interface name.
		testkit.Assert(t, impl).
			Contains("type StoreGetStub struct", "must have StoreGetStub").
			Contains("type StorePutStub struct", "must have StorePutStub")

		// Returns + Func methods.
		testkit.Assert(t, impl).
			Contains("func (s *StoreGetStub) Returns(", "must have StoreGetStub.Returns").
			Contains("func (s *StoreGetStub) Func(", "must have StoreGetStub.Func")

		// Constructor.
		testkit.Assert(t, impl).Contains("func NewStoreStub(", "must have constructor")

		// Options — prefixed with interface name.
		testkit.Assert(t, impl).
			Contains("StoreStubStrict()", "must have Strict option").
			Contains("StoreStubDelegateTo(", "must have DelegateTo option").
			Contains("WithStoreGet(", "must have WithStoreGet option").
			Contains("WithStorePut(", "must have WithStorePut option")

		// On* fields.
		testkit.Assert(t, impl).
			Contains("OnGet", "must have OnGet field").
			Contains("OnPut", "must have OnPut field")

		// Fault injection only on error-returning methods.
		testkit.Assert(t, impl).
			Contains("s.OnGet.ShouldFault()", "Get returns error, must check fault").
			Contains("s.OnPut.ShouldFault()", "Put returns error, must check fault").
			Contains("s.OnDelete.ShouldFault()", "Delete returns error, must check fault")
		// List does not return error — no fault check.
		testkit.Assert(t, impl).NotContains("s.OnList.ShouldFault()", "List has no error, must not check fault")
	})

	t.Run("has DO NOT EDIT header", func(t *testing.T) {
		t.Parallel()
		result, err := GenerateStub(pkg, []string{"Store"}, cfg, Options{
			Output: "stubtest/in_memory_store.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		for _, f := range result.Files {
			testkit.Assert(t, string(f.Content)).Contains("DO NOT EDIT", "must have header")
		}
	})

	t.Run("validates type is interface", func(t *testing.T) {
		t.Parallel()
		_, err := GenerateStub(pkg, []string{"Item"}, cfg, Options{
			Output: "stubtest/item.gen.go",
		})
		testkit.Error(t, err, "must fail for struct type")
	})

	t.Run("output is deterministic", func(t *testing.T) {
		t.Parallel()
		opts := Options{Output: "stubtest/in_memory_store.gen.go"}
		r1, err := GenerateStub(pkg, []string{"Store"}, cfg, opts)
		testkit.NoError(t, err, "first run")
		r2, err := GenerateStub(pkg, []string{"Store"}, cfg, opts)
		testkit.NoError(t, err, "second run")
		for i := range r1.Files {
			testkit.Equal(t, string(r1.Files[i].Content), string(r2.Files[i].Content),
				"output must be identical across runs")
		}
	})

	//nolint:paralleltest // writes fixture files
	t.Run("generate same-package fixtures", func(t *testing.T) {
		result, err := GenerateStub(pkg, []string{"Store"}, cfg, Options{
			Output: "store_stub.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		err = WriteResult(result, filepath.Join(testdataDir(t), "stub"), false)
		testkit.NoError(t, err, "writing must succeed")
	})

	//nolint:paralleltest // writes fixture files
	t.Run("generate cross-package fixtures", func(t *testing.T) {
		result, err := GenerateStub(pkg, []string{"Store"}, cfg, Options{
			Output: "stubtest/in_memory_store.gen.go",
		})
		testkit.NoError(t, err, "generation must succeed")
		err = WriteResult(result, filepath.Join(testdataDir(t), "stub"), false)
		testkit.NoError(t, err, "writing must succeed")
	})
}
