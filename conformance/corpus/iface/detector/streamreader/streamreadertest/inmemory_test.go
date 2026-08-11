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
// promise ends at the call: two checks, both about the call itself.
//
// Everything the stream is about happens when someone ranges it, and no
// generated check does — which is not a gap so much as the honest reading of a
// lazy return. The error is per element rather than per call, so even a
// cancelled context has nowhere to surface until the first yield.
func TestStreamReaderContract(t *testing.T) {
	t.Parallel()

	// No fixture: List takes nothing after its context, so there is no input to
	// derive and the generated struct is empty.
	streamreadertest.AssertStreamReaderContract(t,
		streamreadertest.StreamReaderSubject("in-memory", func() streamreader.StreamReader {
			return streamreadertest.NewInMemory()
		}),
		streamreadertest.StreamReaderSeed(func(_ context.Context, subject streamreader.StreamReader) error {
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			s := subject.(*streamreadertest.InMemory)
			s.Put(streamreader.Value{Key: "first", Body: "one"})
			s.Put(streamreader.Value{Key: "second", Body: "two"})
			return nil
		}),
		streamreadertest.StreamReaderOnList("yields what it holds, in order", func(
			tb testing.TB, subject streamreader.StreamReader,
		) {
			tb.Helper()
			var got []string
			for v, err := range subject.List(tb.Context()) {
				testkit.NoError(tb, err, "a healthy stream yields no per-element error")
				got = append(got, v.Key)
			}
			testkit.Equal(tb, got, []string{"first", "second"}, "in the order they were added")
		}),
		streamreadertest.StreamReaderOnList("stops when the consumer does", func(
			tb testing.TB, subject streamreader.StreamReader,
		) {
			tb.Helper()
			// The shape's own law: a consumer may break out, so an
			// implementation must not assume the sequence is drained. One that
			// did would deadlock or panic here rather than return.
			var seen int
			for range subject.List(tb.Context()) {
				seen++
				break
			}
			testkit.Equal(tb, seen, 1, "the range stopped after one element")
		}),
		streamreadertest.StreamReaderOnList("reports a cancelled context on the first yield", func(
			tb testing.TB, subject streamreader.StreamReader,
		) {
			tb.Helper()
			// Where the generated cancellation check would have gone, if the
			// error were on the call rather than on the element.
			ctx, cancel := context.WithCancel(tb.Context())
			cancel()
			for _, err := range subject.List(ctx) {
				testkit.ErrorIs(tb, err, context.Canceled,
					"a cancelled stream says so through its element error")
				break
			}
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestStreamReaderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	streamreadertest.AssertStreamReaderContract(t,
		streamreadertest.StreamReaderSubject("in-memory", func() streamreader.StreamReader {
			return streamreadertest.NewInMemory()
		}),
		streamreadertest.StreamReaderWithout("List/smoke"),
		streamreadertest.StreamReaderWithoutDouble(),
	)
}
