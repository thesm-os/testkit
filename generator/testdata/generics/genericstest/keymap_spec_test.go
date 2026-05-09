// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"testing"
)

// TestKeyMapContract is intentionally a Skip pending a generator
// fix. The Reader baseline emits two contract assertions keyed on
// the sample input:
//
//	AssertReturnsForKey   (*new(K), *new(V))         — expects (zeroV, nil)
//	AssertReturnsSentinel (*new(K), ErrNotFound)     — expects (anything, ErrNotFound)
//
// For non-generic K the sample literal ("test-key") differs from
// the zero literal (""), so the two assertions land on different
// keys. For generic K, [spec.Method.SampleParamAt] and ZeroParamAt
// both render `*new(K)`, so the two assertions hit the same key
// and become mutually exclusive — no factory can satisfy both.
//
// Fix path: emit the Sentinel assertion conditionally for generic
// shapes (skip when sample key == zero key), or render distinct
// "invalid-K" literal alongside SampleParamAt. Tracked as a
// generator follow-up.
func TestKeyMapContract(t *testing.T) {
	t.Skip("generator emits sample==zero key for generic Reader; AssertReturnsForKey conflicts with AssertReturnsSentinel until the generator distinguishes them")
}
