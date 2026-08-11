// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamconsumertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamconsumer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamconsumer/streamconsumertest"
)

// Two interfaces in one package, so two harnesses and two subjects — which is
// the arrangement, not an accident: the consumer needs something to consume, and
// the thing it consumes is a contract in its own right.
func TestStreamConsumerContract(t *testing.T) {
	t.Parallel()

	streamconsumertest.AssertStreamConsumerContract(t,
		streamconsumertest.StreamConsumerSubject("in-memory",
			func() streamconsumer.StreamConsumer {
				return streamconsumertest.NewInMemory()
			}),
		streamconsumertest.StreamConsumerOnIngest("drains the source and counts what it took", func(
			tb testing.TB, subject streamconsumer.StreamConsumer, src streamconsumer.Source,
		) {
			tb.Helper()
			// The derived src is nil — an interface parameter is a type no
			// literal can be written for — so a check wanting a real stream
			// builds one. That is the whole reason this extension point exists.
			got, err := subject.Ingest(tb.Context(), streamconsumertest.NewSliceSource(
				streamconsumer.Value{Key: "a", Body: "one"},
				streamconsumer.Value{Key: "b", Body: "two"},
			))
			testkit.NoError(tb, err, "a readable source is ingested")
			testkit.Equal(tb, got, 2, "and every element is counted")
		}),
		streamconsumertest.StreamConsumerOnIngest("reports the zero count when handed nothing", func(
			tb testing.TB, subject streamconsumer.StreamConsumer, src streamconsumer.Source,
		) {
			tb.Helper()
			got, err := subject.Ingest(tb.Context(), nil)
			testkit.ErrorIs(tb, err, streamconsumertest.ErrNoSource,
				"a nil source is a failed ingest rather than an empty one")
			testkit.Equal(tb, got, 0, "and carries the zero count beside it")
		}),
		streamconsumertest.StreamConsumerOnIngest("refuses a source that is not there", func(
			tb testing.TB, subject streamconsumer.StreamConsumer, src streamconsumer.Source,
		) {
			tb.Helper()
			// A nil source reaches production through a caller whose own
			// construction failed, and draining it is a panic rather than a
			// count of zero.
			got, err := subject.Ingest(tb.Context(), nil)
			testkit.Error(tb, err, "a missing source is refused")
			testkit.Equal(tb, got, 0, "with nothing counted beside it")
		}),
		streamconsumertest.StreamConsumerOnIngest("stops on a source that fails mid-drain", func(
			tb testing.TB, subject streamconsumer.StreamConsumer, src streamconsumer.Source,
		) {
			tb.Helper()
			// A source that fails partway is the ordinary network case, and the
			// count that comes back with the error is what tells a caller
			// whether to resume or restart.
			got, err := subject.Ingest(tb.Context(), &failingSource{})
			testkit.ErrorIs(tb, err, streamconsumertest.ErrSourceFailed,
				"the source's failure is reported")
			testkit.Equal(tb, got, 0,
				"with nothing counted beside it, since a partial drain is not a count")
		}),
	)
}

// The stream being consumed answers to its own contract, which is what makes
// the asymmetry in this fixture worth having: a produced stream is an
// iter.Seq2 return and generates almost nothing, while a consumed one is an
// interface parameter and generates a whole second harness.
func TestSourceContract(t *testing.T) {
	t.Parallel()

	streamconsumertest.AssertSourceContract(t,
		streamconsumertest.SourceSubject("slice", func() streamconsumer.Source {
			return streamconsumertest.NewSliceSource(
				streamconsumer.Value{Key: "a", Body: "one"},
			)
		}),
		streamconsumertest.SourceOnNext("reports exhaustion through its flag", func(
			tb testing.TB, subject streamconsumer.Source,
		) {
			tb.Helper()
			v, ok, err := subject.Next(tb.Context())
			testkit.NoError(tb, err, "the first element reads cleanly")
			testkit.True(tb, ok, "and the flag says there was one")
			testkit.Equal(tb, v.Key, "a", "carrying what the source held")

			v, ok, err = subject.Next(tb.Context())
			testkit.NoError(tb, err, "exhaustion is not a failure")
			testkit.False(tb, ok, "the flag says the stream is done")
			testkit.Equal(tb, v, streamconsumer.Value{},
				"and the value slot is the zero rather than the last element again")
		}),
	)
}

// failingSource yields one element and then fails, which is the case a count
// returned beside an error would misreport.
//
// A pointer receiver because the state has to advance: a value receiver would
// serve the first element forever and the ingest would never terminate.
type failingSource struct{ served bool }

func (f *failingSource) Next(context.Context) (streamconsumer.Value, bool, error) {
	if !f.served {
		f.served = true
		return streamconsumer.Value{Key: "a"}, true, nil
	}
	return streamconsumer.Value{}, false, streamconsumertest.ErrSourceFailed
}

// Declining the double is separate from dropping a check.
func TestStreamConsumerContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	streamconsumertest.AssertStreamConsumerContract(t,
		streamconsumertest.StreamConsumerSubject("in-memory",
			func() streamconsumer.StreamConsumer {
				return streamconsumertest.NewInMemory()
			}),
		streamconsumertest.StreamConsumerWithout("Ingest/smoke"),
		streamconsumertest.StreamConsumerWithoutDouble(),
	)
}
