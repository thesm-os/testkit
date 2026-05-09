// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type byteCounter struct{}

var errStreamInvalid = errors.New("stream: invalid")

// ReadFrom counts the bytes drained from r. Returns errStreamInvalid
// when r is the sentinel "<invalid>" string-reader.
func (*byteCounter) ReadFrom(_ context.Context, r io.Reader) (int, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("byteCounter: read: %w", err)
	}
	if bytes.Equal(buf, []byte("INVALID")) {
		return 0, errStreamInvalid
	}
	return len(buf), nil
}

func streamConsumerCtx(t *testing.T) suite.StreamConsumerContext[*byteCounter, io.Reader, int] {
	t.Helper()
	return suite.StreamConsumerContext[*byteCounter, io.Reader, int]{
		T: t,
		StreamConsumerBindings: bindings.StreamConsumerBindings[*byteCounter, io.Reader, int]{
			Factory: func() *byteCounter { return &byteCounter{} },
			Call: func(ctx context.Context, b *byteCounter, r io.Reader) (int, error) {
				return b.ReadFrom(ctx, r)
			},
		},
	}
}

func TestStreamConsumer(t *testing.T) {
	t.Parallel()

	t.Run("Succeeds for a valid stream", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamConsumerSucceeds[*byteCounter, io.Reader, int](
			bytes.NewReader([]byte("hello")), 5)(streamConsumerCtx(t))
	})

	t.Run("RejectInvalid surfaces the sentinel", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamConsumerRejectInvalid[*byteCounter, io.Reader, int](
			bytes.NewReader([]byte("INVALID")), errStreamInvalid)(streamConsumerCtx(t))
	})

	t.Run("Consistent yields equal results across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamConsumerConsistent[*byteCounter, io.Reader, int](
			func() io.Reader { return bytes.NewReader([]byte("hello")) }, 4)(
			streamConsumerCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled", func(t *testing.T) {
		t.Parallel()
		ctx := suite.StreamConsumerContext[*byteCounter, io.Reader, int]{
			T: t,
			StreamConsumerBindings: bindings.StreamConsumerBindings[*byteCounter, io.Reader, int]{
				Factory: func() *byteCounter { return &byteCounter{} },
				Call: func(c context.Context, _ *byteCounter, _ io.Reader) (int, error) {
					return 0, c.Err()
				},
			},
		}
		suite.AssertStreamConsumerRespectsContext[*byteCounter, io.Reader, int](
			bytes.NewReader([]byte("x")))(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamConsumerConcurrentSafe[*byteCounter, io.Reader, int](
			func() io.Reader { return bytes.NewReader([]byte("hello")) }, 4, 50)(
			streamConsumerCtx(t))
	})

	t.Run("SucceedsWithDefault reads default sample bytes", func(t *testing.T) {
		t.Parallel()
		// "test-data" is 9 bytes (the default sample)
		suite.AssertStreamConsumerSucceedsWithDefault[*byteCounter, io.Reader, int](
			9)(streamConsumerCtx(t))
	})

	t.Run("RespectsContextWithDefault surfaces ctx.Canceled", func(t *testing.T) {
		t.Parallel()
		ctx := suite.StreamConsumerContext[*byteCounter, io.Reader, int]{
			T: t,
			StreamConsumerBindings: bindings.StreamConsumerBindings[*byteCounter, io.Reader, int]{
				Factory: func() *byteCounter { return &byteCounter{} },
				Call: func(c context.Context, _ *byteCounter, _ io.Reader) (int, error) {
					return 0, c.Err()
				},
			},
		}
		suite.AssertStreamConsumerRespectsContextWithDefault[*byteCounter, io.Reader, int]()(ctx)
	})

	t.Run("ConcurrentSafeWithDefault runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertStreamConsumerConcurrentSafeWithDefault[*byteCounter, io.Reader, int](
			4, 50)(streamConsumerCtx(t))
	})
}
