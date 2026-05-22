// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.thesmos.sh/testkit/suite"
)

type directive struct {
	mu    sync.Mutex
	count int
}

func directiveFactory() *directive { return &directive{} }

func TestAssertDeprecatedSmoke(t *testing.T) {
	t.Parallel()
	fn := suite.AssertDeprecatedSmoke("Old", "New", func(_ context.Context, d *directive) {
		d.mu.Lock()
		d.count++
		d.mu.Unlock()
	})
	fn(t, directiveFactory)
}

func TestAssertRetrySucceedsOnAttempt(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	fn := suite.AssertRetrySucceedsOnAttempt(3, func(_ context.Context, _ *directive) error {
		if calls.Add(1) < 3 {
			return errors.New("transient")
		}
		return nil
	})
	fn(t, directiveFactory)
}

func TestAssertOrderAfter(t *testing.T) {
	t.Parallel()

	type ordered struct {
		mu     sync.Mutex
		opened bool
	}
	factory := func() *ordered { return &ordered{} }

	fn := suite.AssertOrderAfter(
		"Open",
		func(_ context.Context, o *ordered) error {
			o.mu.Lock()
			defer o.mu.Unlock()
			o.opened = true
			return nil
		},
		func(_ context.Context, o *ordered) error {
			o.mu.Lock()
			defer o.mu.Unlock()
			if !o.opened {
				return errors.New("not opened")
			}
			return nil
		},
	)
	fn(t, factory)
}

func TestAssertPartitionIsolation(t *testing.T) {
	t.Parallel()
	fn := suite.AssertPartitionIsolation("field", func(_ context.Context, d *directive) error {
		d.mu.Lock()
		d.count++
		d.mu.Unlock()
		return nil
	})
	fn(t, directiveFactory)
}

func TestAssertWrappedVia(t *testing.T) {
	t.Parallel()
	target := errors.New("target")
	sentinel := errors.New("sentinel")
	fn := suite.AssertWrappedVia(target, func(_ context.Context, _ *directive) error {
		return errors.Join(target, sentinel)
	}, sentinel)
	fn(t, directiveFactory)
}

func TestAssertIdempotentSecondCall(t *testing.T) {
	t.Parallel()
	fn := suite.AssertIdempotentSecondCall(func(_ context.Context, d *directive) {
		d.mu.Lock()
		d.count++
		d.mu.Unlock()
	})
	fn(t, directiveFactory)
}

func TestAssertPureImplIndependent(t *testing.T) {
	t.Parallel()
	fn := suite.AssertPureImplIndependent(func(_ context.Context, _ *directive) (int, error) {
		return 42, nil
	})
	fn(t, directiveFactory)
}

func TestAssertCacheableRepeatedReads(t *testing.T) {
	t.Parallel()
	fn := suite.AssertCacheableRepeatedReads(func(_ context.Context, _ *directive) (string, error) {
		return "cached", nil
	})
	fn(t, directiveFactory)
}

func TestAssertMonotonicNonDecreasing(t *testing.T) {
	t.Parallel()
	var counter atomic.Int64
	fn := suite.AssertMonotonicNonDecreasing(5, func(_ context.Context, _ *directive) (int64, error) {
		return counter.Add(1), nil
	})
	fn(t, directiveFactory)
}

func TestAssertConcurrentStrict(t *testing.T) {
	t.Parallel()
	fn := suite.AssertConcurrentStrict(func(_ context.Context, d *directive) {
		d.mu.Lock()
		d.count++
		d.mu.Unlock()
	})
	fn(t, directiveFactory)
}

func TestAssertConcurrentReadersParallel(t *testing.T) {
	t.Parallel()
	fn := suite.AssertConcurrentReadersParallel(func(_ context.Context, d *directive) {
		d.mu.Lock()
		_ = d.count
		d.mu.Unlock()
	})
	fn(t, directiveFactory)
}

func TestAssertNilSafeNoPanic(t *testing.T) {
	t.Parallel()
	fn := suite.AssertNilSafeNoPanic(func(_ context.Context, _ *directive) {
		// no-op with zero inputs
	})
	fn(t, directiveFactory)
}

func TestAssertAtomicNoTrace(t *testing.T) {
	t.Parallel()
	fn := suite.AssertAtomicNoTrace(
		func(_ context.Context, _ *directive) {
			// failing call that does not mutate state
		},
		nil, // use default reflect.DeepEqual
	)
	fn(t, directiveFactory)
}

func TestAssertBoundedRange(t *testing.T) {
	t.Parallel()
	fn := suite.AssertBoundedRange("0-100", 0, 100, func(_ context.Context, _ *directive) (int, error) {
		return 50, nil
	})
	fn(t, directiveFactory)
}

func TestAssertTimeoutWithin(t *testing.T) {
	t.Parallel()
	fn := suite.AssertTimeoutWithin(time.Second, "1s", func(_ context.Context, _ *directive) {
		// completes immediately
	})
	fn(t, directiveFactory)
}

func TestAssertSideEffectObservable(t *testing.T) {
	t.Parallel()
	fn := suite.AssertSideEffectObservable(
		"count",
		func(_ context.Context, d *directive) any {
			d.mu.Lock()
			defer d.mu.Unlock()
			return d.count
		},
		func(_ context.Context, d *directive) {
			d.mu.Lock()
			d.count++
			d.mu.Unlock()
		},
	)
	fn(t, directiveFactory)
}

func TestAssertValidatesZeroInput(t *testing.T) {
	t.Parallel()
	fn := suite.AssertValidatesZeroInput("name", func(_ context.Context, _ *directive) error {
		return errors.New("name is required")
	})
	fn(t, directiveFactory)
}

func TestAssertHooksFire(t *testing.T) {
	t.Parallel()
	fn := suite.AssertHooksFire([]string{"beforeSave", "afterSave"}, func(ctx context.Context, _ *directive) {
		recorder := suite.RecorderFromContext(ctx)
		recorder.Record("beforeSave")
		recorder.Record("afterSave")
	})
	fn(t, directiveFactory)
}

func TestAssertEventuallyConverges(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	fn := suite.AssertEventuallyConverges(5*time.Second, "5s", func(_ context.Context, _ *directive) any {
		c := calls.Add(1)
		if c <= 2 {
			return c
		}
		return int64(99) // stabilizes
	})
	fn(t, directiveFactory)
}

func TestAssertScopeAuthRequired(t *testing.T) {
	t.Parallel()

	type scopeKey struct{}
	unauthorized := errors.New("unauthorized")

	fn := suite.AssertScopeAuthRequired(
		"admin",
		func(scope string) context.Context {
			//nolint:usetesting // scopeContext is not a test helper; it builds a context with auth data
			return context.WithValue(context.Background(), scopeKey{}, scope)
		},
		unauthorized,
		func(ctx context.Context, _ *directive) error {
			v, _ := ctx.Value(scopeKey{}).(string)
			if v != "admin" {
				return fmt.Errorf("no auth: %w", unauthorized)
			}
			return nil
		},
	)
	fn(t, directiveFactory)
}

func TestAssertLeaseAcquireRelease(t *testing.T) {
	t.Parallel()

	type lease struct {
		mu       sync.Mutex
		acquired bool
	}
	factory := func() *lease { return &lease{} }

	fn := suite.AssertLeaseAcquireRelease(
		func(_ context.Context, l *lease) error {
			l.mu.Lock()
			defer l.mu.Unlock()
			if l.acquired {
				return errors.New("already acquired")
			}
			l.acquired = true
			return nil
		},
		func(_ context.Context, l *lease) error {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.acquired = false
			return nil
		},
	)
	fn(t, factory)
}
