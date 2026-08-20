// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamreadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamreader/streamreadertest"
)

// A method returning an iterator returns a function, so what the signature can
// promise ends at the call: one check, about the call itself.
//
// Everything the stream is about happens when someone ranges it, and no
// generated check does — which is not a gap so much as the honest reading of a
// lazy return. The error is per element rather than per call, so even a
// cancelled context has nowhere to surface until the first yield.
func TestStreamReaderContract(t *testing.T) {
	t.Parallel()

	streamreadertest.RunStreamReader(t,
		streamreadertest.StreamReaderHarness[*streamreadertest.InMemory]{
			Name: "in-memory",
			New: func() *streamreadertest.InMemory {
				s := streamreadertest.NewInMemory()
				s.Put(streamreader.Value{Key: "first", Body: "one"})
				s.Put(streamreader.Value{Key: "second", Body: "two"})
				return s
			},
		},
		streamreadertest.StreamReaderChecks{
			{
				Method: "List",
				Name:   "yields-what-it-holds-in-order",
				Claim:  "List yields what it holds, in order",
				Run: func(tb testing.TB, s streamreader.StreamReader, fx streamreadertest.StreamReaderFixture) {
					tb.Helper()
					var got []string
					for v, err := range s.List(tb.Context()) {
						testkit.NoError(tb, err, "a healthy stream yields no per-element error")
						got = append(got, v.Key)
					}
					testkit.Equal(tb, got, []string{"first", "second"}, "in the order they were added")
				},
			},
			{
				Method: "List",
				Name:   "stops-when-the-consumer-does",
				Claim:  "List stops when the consumer does",
				Run: func(tb testing.TB, s streamreader.StreamReader, fx streamreadertest.StreamReaderFixture) {
					tb.Helper()
					// The shape's own law: a consumer may break out, so an
					// implementation must not assume the sequence is drained.
					// One that did would deadlock or panic here rather than
					// return.
					var seen int
					for range s.List(tb.Context()) {
						seen++
						break
					}
					testkit.Equal(tb, seen, 1, "the range stopped after one element")
				},
			},
			{
				Method: "List",
				Name:   "cancelled-on-the-first-yield",
				Claim:  "List reports a cancelled context on the first yield",
				Run: func(tb testing.TB, s streamreader.StreamReader, fx streamreadertest.StreamReaderFixture) {
					tb.Helper()
					// Where the generated cancellation check would have gone, if
					// the error were on the call rather than on the element.
					ctx, cancel := context.WithCancel(tb.Context())
					cancel()
					for _, err := range s.List(ctx) {
						testkit.ErrorIs(tb, err, context.Canceled,
							"a cancelled stream says so through its element error")
						break
					}
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestStreamReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	streamreadertest.RunStreamReader(
		t,
		streamreadertest.StreamReaderHarness[*streamreadertest.InMemory]{
			Name: "in-memory",
			New:  streamreadertest.NewInMemory,
		},
		streamreadertest.StreamReaderSuite.Without(streamreadertest.StreamReaderSuite.Checks.List.Smoke()),
	)
}
