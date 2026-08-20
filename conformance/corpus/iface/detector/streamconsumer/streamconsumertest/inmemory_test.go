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

// Two interfaces in one package, so two runs and two subjects — which is
// the arrangement, not an accident: the consumer needs something to consume, and
// the thing it consumes is a contract in its own right.
//
// Ingest takes a Source, which no literal can be written for, so every family
// the rules reached for it was refused; the header lists both. The rows below
// build the sources the claims need.
func TestStreamConsumerContract(t *testing.T) {
	t.Parallel()

	streamconsumertest.RunStreamConsumer(t,
		streamconsumertest.StreamConsumerHarness[*streamconsumertest.InMemory]{
			Name: "in-memory", New: streamconsumertest.NewInMemory,
		},
		streamconsumertest.StreamConsumerChecks{
			{
				Method: "Ingest",
				Name:   "drains-and-counts",
				Claim:  "Ingest drains the source and counts what it took",
				Run: func(tb testing.TB, s streamconsumer.StreamConsumer, fx streamconsumertest.StreamConsumerFixture) {
					tb.Helper()
					got, err := s.Ingest(tb.Context(), streamconsumertest.NewSliceSource(
						streamconsumer.Value{Key: "a", Body: "one"},
						streamconsumer.Value{Key: "b", Body: "two"},
					))
					testkit.NoError(tb, err, "a readable source is ingested")
					testkit.Equal(tb, got, 2, "and every element is counted")
				},
			},
			{
				Method: "Ingest",
				Name:   "refuses-a-missing-source",
				Claim:  "Ingest refuses a source that is not there",
				Run: func(tb testing.TB, s streamconsumer.StreamConsumer, fx streamconsumertest.StreamConsumerFixture) {
					tb.Helper()
					// A nil source reaches production through a caller whose own
					// construction failed, and draining it is a panic rather than
					// a count of zero.
					got, err := s.Ingest(tb.Context(), nil)
					testkit.ErrorIs(tb, err, streamconsumertest.ErrNoSource,
						"a nil source is a failed ingest rather than an empty one")
					testkit.Equal(tb, got, 0, "and carries the zero count beside it")
				},
			},
			{
				Method: "Ingest",
				Name:   "stops-on-a-failing-source",
				Claim:  "Ingest stops on a source that fails mid-drain",
				Run: func(tb testing.TB, s streamconsumer.StreamConsumer, fx streamconsumertest.StreamConsumerFixture) {
					tb.Helper()
					// A source that fails partway is the ordinary network case,
					// and the count that comes back with the error is what tells
					// a caller whether to resume or restart.
					got, err := s.Ingest(tb.Context(), &failingSource{})
					testkit.ErrorIs(tb, err, streamconsumertest.ErrSourceFailed,
						"the source's failure is reported")
					testkit.Equal(tb, got, 0,
						"with nothing counted beside it, since a partial drain is not a count")
				},
			},
		},
	)
}

// The stream being consumed answers to its own contract, which is what makes
// the asymmetry in this fixture worth having: a produced stream is an
// iter.Seq2 return and generates almost nothing, while a consumed one is an
// interface parameter and generates a whole second harness.
func TestSourceContract(t *testing.T) {
	t.Parallel()

	streamconsumertest.RunSource(t,
		streamconsumertest.SourceHarness[streamconsumer.Source]{
			Name: "slice",
			New: func() streamconsumer.Source {
				return streamconsumertest.NewSliceSource(
					streamconsumer.Value{Key: "a", Body: "one"},
				)
			},
		},
		streamconsumertest.SourceChecks{
			{
				Method: "Next",
				Name:   "reports-exhaustion-through-its-flag",
				Claim:  "Next reports exhaustion through its flag",
				Run: func(tb testing.TB, s streamconsumer.Source, fx streamconsumertest.SourceFixture) {
					tb.Helper()
					v, ok, err := s.Next(tb.Context())
					testkit.NoError(tb, err, "the first element reads cleanly")
					testkit.True(tb, ok, "and the flag says there was one")
					testkit.Equal(tb, v.Key, "a", "carrying what the source held")

					v, ok, err = s.Next(tb.Context())
					testkit.NoError(tb, err, "exhaustion is not a failure")
					testkit.False(tb, ok, "the flag says the stream is done")
					testkit.Equal(tb, v, streamconsumer.Value{},
						"and the value slot is the zero rather than the last element again")
				},
			},
		},
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

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
//
// Against Source rather than StreamConsumer: Ingest's only argument admits no
// literal, so StreamConsumer derives nothing and has no index entry to name.
func TestSourceContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	streamconsumertest.RunSource(t,
		streamconsumertest.SourceHarness[streamconsumer.Source]{
			Name: "slice",
			New: func() streamconsumer.Source {
				return streamconsumertest.NewSliceSource(
					streamconsumer.Value{Key: "a", Body: "one"},
				)
			},
		},
		streamconsumertest.SourceSuite.Without(streamconsumertest.SourceSuite.Checks.Next.Smoke()),
	)
}
