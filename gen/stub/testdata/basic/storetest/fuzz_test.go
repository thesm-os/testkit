// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/stub/testdata/basic/storetest"
)

// FuzzStubWithTestingF verifies that creating a stub with *testing.F
// does not panic. Go's testing framework forbids f.Cleanup inside fuzz
// bodies — the stub must detect *testing.F and skip cleanup registration.
func FuzzStubWithTestingF(f *testing.F) {
	f.Add("key")
	f.Fuzz(func(t *testing.T, key string) {
		// Construct with f (the outer *testing.F) — must not panic.
		s := storetest.NewStoreStub(f)
		_, _ = s.Get(t.Context(), key)
	})
}
