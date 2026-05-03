// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package weirdtest_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/weird"
	"go.thesmos.sh/testkit/gen/suite/testdata/weird/weirdtest"
	"go.thesmos.sh/testkit/suite"
)

func TestJSONCodecContract(t *testing.T) {
	t.Parallel()
	factory := func() weird.Codec { return weird.NewJSONCodec() }

	weirdtest.AssertCodecContract(t, factory,
		// Pure: ContentType returns "application/json".
		weirdtest.CodecOnContentType(
			suite.AssertDeterministic[weird.Codec, string](3),
		),

		// Predicate: Handles returns true for "application/json".
		weirdtest.CodecOnHandles(
			suite.AssertPredicateConsistent[weird.Codec](3),
		),

		// Unknown-shaped: Encode exercises io.Writer param.
		weirdtest.CodecOnEncode(func(t *testing.T, c weird.Codec) {
			var buf bytes.Buffer
			err := c.Encode(&buf, map[string]string{"key": "value"})
			testkit.NoError(t, err, "Encode must succeed")
			testkit.True(t, buf.Len() > 0, "Encode must write bytes")
		}),

		// Unknown-shaped: Decode exercises io.Reader param.
		weirdtest.CodecOnDecode(func(t *testing.T, c weird.Codec) {
			input := bytes.NewBufferString(`{"key":"value"}`)
			var out map[string]string
			err := c.Decode(input, &out)
			testkit.NoError(t, err, "Decode must succeed")
			testkit.Equal(t, out["key"], "value", "Decode must produce expected value")
		}),

		// Unknown-shaped: MarshalBinary returns bytes.
		weirdtest.CodecOnMarshalBinary(func(t *testing.T, c weird.Codec) {
			data, err := c.MarshalBinary("hello")
			testkit.NoError(t, err, "MarshalBinary must succeed")
			testkit.True(t, len(data) > 0, "MarshalBinary must return bytes")
		}),

		weirdtest.CodecCustom("encode-decode round-trip", func(t *testing.T, c weird.Codec) {
			var buf bytes.Buffer
			original := map[string]string{"round": "trip"}
			testkit.NoError(t, c.Encode(&buf, original), "encode")
			var decoded map[string]string
			testkit.NoError(t, c.Decode(&buf, &decoded), "decode")
			testkit.Equal(t, decoded["round"], "trip", "round-trip must preserve data")
		}),
	)
}

func TestInMemorySchedulerContract(t *testing.T) {
	t.Parallel()
	factory := func() weird.Scheduler { return weird.NewInMemoryScheduler("test-scheduler") }

	weirdtest.AssertSchedulerContract(t, factory,
		weirdtest.SchedulerPrePopulate(func(ctx context.Context, s weird.Scheduler) {
			_ = s.Schedule(ctx, "task-1", time.Second, func(context.Context) error { return nil })
		}),

		// Unknown-shaped: Schedule takes multiple non-ctx params.
		weirdtest.SchedulerOnSchedule(func(t *testing.T, s weird.Scheduler) {
			err := s.Schedule(t.Context(), "new-task", time.Minute, func(context.Context) error { return nil })
			testkit.NoError(t, err, "Schedule must succeed")
		}),

		// Deleter: Cancel removes a scheduled task.
		weirdtest.SchedulerOnCancel(
			suite.AssertDeleteSucceeds[weird.Scheduler, string]("task-1"),
			suite.AssertDeleteReturnsNotFound[weird.Scheduler, string]("nonexistent", weird.ErrNotScheduled),
		),

		// Reader: Status returns task info.
		weirdtest.SchedulerOnStatus(
			suite.AssertReturnsForKey[weird.Scheduler, string, weird.TaskStatus](
				"task-1", weird.TaskStatus{ID: "task-1", Running: true},
			),
			suite.AssertReturnsSentinel[weird.Scheduler, string, weird.TaskStatus](
				"nonexistent", weird.ErrNotScheduled,
			),
		),

		// Aggregator: Running returns task count.
		weirdtest.SchedulerOnRunning(
			suite.AssertAggregatorBounded[weird.Scheduler, int](0, 1000),
			suite.AssertAggregatorConsistent[weird.Scheduler, int](3),
		),

		// Lifecycle: Flush clears all tasks.
		weirdtest.SchedulerOnFlush(
			suite.AssertLifecycleSucceeds[weird.Scheduler](),
		),

		// StreamReader: Tasks iterates scheduled tasks.
		weirdtest.SchedulerOnTasks(
			suite.AssertStreamCompletes[weird.Scheduler, weird.TaskStatus](),
			suite.AssertStreamRespectsBreak[weird.Scheduler, weird.TaskStatus](),
			suite.AssertStreamReentrant[weird.Scheduler, weird.TaskStatus](),
		),

		// Pure: Name returns the scheduler name.
		weirdtest.SchedulerOnName(
			suite.AssertDeterministic[weird.Scheduler, string](3),
		),
	)
}
