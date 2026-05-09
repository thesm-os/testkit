// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
)

// TestKeyMapContract closes the loop on the two-parameter generic
// suite generator. AssertKeyMapContract instantiates at K=string,
// V=int. The factory pre-seeds the zero K → zero V pair so the
// Reader baseline lands on a real entry.
//
// The contract emits AssertReturnsForKey (sample → zeroV) but
// omits AssertReturnsSentinel — the suite generator's
// SampleEqualsZero predicate detects that sample key == zero key
// for generic Reader and skips the conflicting sentinel assertion.
func TestKeyMapContract(t *testing.T) {
	t.Parallel()
	AssertKeyMapContract(t, func() generics.KeyMap[string, int] {
		m := generics.NewInMemoryKeyMap[string, int]()
		_ = m.Set(context.Background(), "", 0)
		return m
	})
}
