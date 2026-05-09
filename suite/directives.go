// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/polling"
)

// Each AssertX runtime helper here owns the contract logic — t.Run,
// loops, deadlines, equality, recover-on-panic. Templates emit ONE
// call per directive; the closures they pass are minimal
// translators that only adapt the impl's specific signature to the
// runtime's expected signature. No control flow lives in generated
// code.

// AssertDeprecatedSmoke calls the deprecated method and asserts no
// panic. Logs the replacement at runtime. The closure invokes the
// method and discards its results.
func AssertDeprecatedSmoke[T any](
	methodName, replacement string,
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("deprecated", func(t *testing.T) {
			t.Parallel()
			t.Logf("%s is deprecated; use %s instead", methodName, replacement)
			impl := factory()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("deprecated method panicked: %v", r)
				}
			}()
			call(t.Context(), impl)
		})
	}
}

// AssertRetrySucceedsOnAttempt verifies the impl returns a transient
// error on the first N-1 calls and succeeds on the Nth. The closure
// captures the impl's trailing error.
func AssertRetrySucceedsOnAttempt[T any](
	n int,
	call func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("retry succeeds on attempt", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			for i := 1; i < n; i++ {
				err := call(t.Context(), impl)
				if err == nil {
					t.Fatalf("attempt %d: expected transient error, got nil", i)
				}
			}
			err := call(t.Context(), impl)
			testkit.NoError(t, err, "final attempt must succeed")
		})
	}
}

// AssertOrderAfter verifies the prerequisite-method contract: the
// carrier errors before the prereq runs, succeeds after.
func AssertOrderAfter[T any](
	target string,
	prerequisite func(ctx context.Context, impl T) error,
	call func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("order-after "+target, func(t *testing.T) {
			t.Run("fails before prerequisite", func(t *testing.T) {
				t.Parallel()
				impl := factory()
				defer func() { _ = recover() }()
				err := call(t.Context(), impl)
				testkit.True(t, err != nil,
					fmt.Sprintf("calling before %s must error or panic", target))
			})
			t.Run("succeeds after prerequisite", func(t *testing.T) {
				t.Parallel()
				impl := factory()
				_ = prerequisite(t.Context(), impl)
				err := call(t.Context(), impl)
				testkit.NoError(t, err, fmt.Sprintf("calling after %s must succeed", target))
			})
		})
	}
}

// AssertPartitionIsolation verifies two sequential writes succeed.
func AssertPartitionIsolation[T any](
	field string,
	call func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("partition isolation ("+field+")", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			testkit.NoError(t, call(t.Context(), impl), "first partition write")
			testkit.NoError(t, call(t.Context(), impl), "second partition write")
		})
	}
}

// AssertWrappedVia verifies error returns satisfy errors.Is against
// both the wrap target and the named sentinel.
func AssertWrappedVia[T any](
	target, sentinel error,
	call func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("wrapped-via", func(t *testing.T) {
			t.Parallel()
			err := call(t.Context(), factory())
			if err != nil {
				testkit.ErrorIs(t, err, target, "error must wrap the target")
				testkit.ErrorIs(t, err, sentinel, "error must also unwrap to the sentinel")
			}
		})
	}
}

// AssertIdempotentSecondCall verifies the second call doesn't panic.
// Paired-method state observation is the cross-method invariant
// directives' job, not this baseline.
func AssertIdempotentSecondCall[T any](
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("idempotent (second call)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			call(t.Context(), impl)
			call(t.Context(), impl)
		})
	}
}

// AssertPureImplIndependent samples the method on two independent
// impls and asserts the results are equal. The closure is a
// single-shot sampler; this helper owns the two-impl construction
// and the comparison.
func AssertPureImplIndependent[T, R any](
	sample func(ctx context.Context, impl T) (R, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("pure (impl-independent)", func(t *testing.T) {
			t.Parallel()
			ra, errA := sample(t.Context(), factory())
			testkit.NoError(t, errA, "pure call on impl A must not error")
			rb, errB := sample(t.Context(), factory())
			testkit.NoError(t, errB, "pure call on impl B must not error")
			testkit.Equal(t, ra, rb, "pure method must be impl-independent")
		})
	}
}

// AssertCacheableRepeatedReads samples the method three times and
// asserts pairwise equality. The closure is a single-shot sampler;
// this helper owns the loop.
func AssertCacheableRepeatedReads[T, R any](
	sample func(ctx context.Context, impl T) (R, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("cacheable (repeated query matches)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			r1, err1 := sample(t.Context(), impl)
			testkit.NoError(t, err1, "cacheable call must not error")
			r2, err2 := sample(t.Context(), impl)
			testkit.NoError(t, err2, "cacheable call must not error")
			r3, err3 := sample(t.Context(), impl)
			testkit.NoError(t, err3, "cacheable call must not error")
			testkit.Equal(t, r1, r2, "cache hit 1: result 1 must equal result 2")
			testkit.Equal(t, r2, r3, "cache hit 2: result 2 must equal result 3")
		})
	}
}

// AssertMonotonicNonDecreasing samples the method [n] times and
// asserts each sample is >= its predecessor. The closure is a
// single-shot sampler.
func AssertMonotonicNonDecreasing[T any, R cmp.Ordered](
	n int,
	sample func(ctx context.Context, impl T) (R, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("monotonic (non-decreasing)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			var prev R
			for i := range n {
				cur, err := sample(t.Context(), impl)
				testkit.NoError(t, err, "monotonic sample must not error")
				if i > 0 && cur < prev {
					t.Fatalf("monotonic value must be non-decreasing: prev=%v cur=%v", prev, cur)
				}
				prev = cur
			}
		})
	}
}

// AssertConcurrentStrict drives 16 workers × 25 iters of the call
// closure; race detector finds violations under -race.
func AssertConcurrentStrict[T any](
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("concurrent (strict)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			var wg sync.WaitGroup
			for range 16 {
				wg.Go(func() {
					for range 25 {
						call(t.Context(), impl)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertConcurrentReadersParallel forks 32 readers; race detector
// catches RWMutex violations.
func AssertConcurrentReadersParallel[T any](
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("concurrent-readers", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			var wg sync.WaitGroup
			for range 32 {
				wg.Go(func() {
					call(t.Context(), impl)
				})
			}
			wg.Wait()
		})
	}
}

// AssertNilSafeNoPanic calls the method with zero/nil inputs and
// asserts no panic.
func AssertNilSafeNoPanic[T any](
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("nilsafe (zero inputs)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("nilsafe method panicked on zero inputs: %v", r)
				}
			}()
			call(t.Context(), impl)
		})
	}
}

// AssertAtomicNoTrace constructs two impls, runs the failing call on
// one, asserts state-equal. Falls back to reflect.DeepEqual when no
// stateEqual is supplied.
func AssertAtomicNoTrace[T any](
	failingCall func(ctx context.Context, impl T),
	stateEqual func(a, b T) bool,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("atomic (failed mutation no-op)", func(t *testing.T) {
			t.Parallel()
			pre := factory()
			post := factory()
			failingCall(t.Context(), post)
			eq := stateEqual
			if eq == nil {
				eq = func(a, b T) bool { return reflect.DeepEqual(a, b) }
			}
			testkit.True(t, eq(pre, post),
				"atomic: failed mutation must leave state equal to pre-call factory output")
		})
	}
}

// AssertBoundedRange samples the method once and asserts the result
// is in [min, max]. Closure is a single-shot sampler.
func AssertBoundedRange[T any, R cmp.Ordered](
	rangeDesc string,
	lower, upper R,
	sample func(ctx context.Context, impl T) (R, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("bounded ("+rangeDesc+")", func(t *testing.T) {
			t.Parallel()
			got, err := sample(t.Context(), factory())
			testkit.NoError(t, err, "bounded call must not error")
			if got < lower || got > upper {
				t.Fatalf("result %v not in [%v, %v]", got, lower, upper)
			}
		})
	}
}

// AssertTimeoutWithin spawns the call in a goroutine, selects on
// completion vs deadline. Owns the timing.
func AssertTimeoutWithin[T any](
	timeout time.Duration,
	desc string,
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("timeout (completes within "+desc+")", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			done := make(chan struct{})
			go func() {
				defer close(done)
				call(t.Context(), impl)
			}()
			select {
			case <-done:
			case <-time.After(timeout):
				t.Fatalf("timeout: did not complete within %s", desc)
			}
		})
	}
}

// AssertSideEffectObservable reads observable state via [observe]
// before and after invoking [mutate]; asserts they differ via
// reflect.DeepEqual.
func AssertSideEffectObservable[T any](
	target string,
	observe func(ctx context.Context, impl T) any,
	mutate func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("sideeffect observable via "+target, func(t *testing.T) {
			t.Parallel()
			impl := factory()
			before := observe(t.Context(), impl)
			mutate(t.Context(), impl)
			after := observe(t.Context(), impl)
			testkit.True(t, !reflect.DeepEqual(before, after),
				fmt.Sprintf("side effect must be observable: before=%v after=%v", before, after))
		})
	}
}

// AssertValidatesZeroInput calls with zero/invalid input and
// asserts a non-nil error.
func AssertValidatesZeroInput[T any](
	field string,
	call func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("validates "+field, func(t *testing.T) {
			t.Parallel()
			err := call(t.Context(), factory())
			testkit.True(t, err != nil,
				fmt.Sprintf("must reject zero %s with a non-nil error", field))
		})
	}
}

// AssertHooksFire constructs a HookRecorder, threads it through the
// call closure, asserts each declared hook fired.
func AssertHooksFire[T any](
	hookNames []string,
	call func(ctx context.Context, impl T),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("hooks fire", func(t *testing.T) {
			t.Parallel()
			recorder := NewHookRecorder()
			ctx := ContextWithRecorder(t.Context(), recorder)
			call(ctx, factory())
			for _, name := range hookNames {
				testkit.True(t, recorder.Count(name) > 0,
					fmt.Sprintf("hook %s must fire during method invocation", name))
			}
		})
	}
}

// AssertEventuallyConverges polls the [sample] closure until two
// consecutive calls return reflect.DeepEqual values, or fails when
// the deadline expires. Uses [polling.RetryUntil] for backoff —
// no time.Sleep at the contract level.
func AssertEventuallyConverges[T any](
	timeout time.Duration,
	desc string,
	sample func(ctx context.Context, impl T) any,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("eventually (converges within "+desc+")", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			var (
				last     any
				haveLast bool
			)
			polling.RetryUntil(t, timeout, func() bool {
				cur := sample(t.Context(), impl)
				if haveLast && reflect.DeepEqual(cur, last) {
					return true
				}
				last = cur
				haveLast = true
				return false
			}, "eventually: did not converge within deadline")
		})
	}
}

// AssertScopeAuthRequired verifies unauthorized calls return the
// configured sentinel; authorized calls succeed. Both options must
// be supplied.
func AssertScopeAuthRequired[T any](
	scopeName string,
	scopeContext func(scope string) context.Context,
	unauthorized error,
	call func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("scope "+scopeName, func(t *testing.T) {
			t.Parallel()
			if scopeContext == nil {
				t.Fatal("scope contract: missing WithScopeContext option; supply it to verify")
			}
			if unauthorized == nil {
				t.Fatal("scope contract: missing WithScopeUnauthorized option; supply it to verify")
			}
			t.Run("unauthorized returns sentinel", func(t *testing.T) {
				t.Parallel()
				err := call(t.Context(), factory())
				testkit.ErrorIs(t, err, unauthorized,
					"unauthorized call must surface the unauthorized sentinel")
			})
			t.Run("authorized succeeds", func(t *testing.T) {
				t.Parallel()
				err := call(scopeContext(scopeName), factory())
				testkit.NoError(t, err, "authorized call must succeed")
			})
		})
	}
}

// AssertLeaseAcquireRelease verifies acquire/release/acquire works
// and double-acquire-without-release fails.
func AssertLeaseAcquireRelease[T any](
	acquire, release func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("lease (acquire-release-acquire)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			testkit.NoError(t, acquire(t.Context(), impl), "first acquire must succeed")
			testkit.NoError(t, release(t.Context(), impl), "release must succeed")
			testkit.NoError(t, acquire(t.Context(), impl), "second acquire after release must succeed")
		})
		t.Run("lease (double-acquire fails)", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			testkit.NoError(t, acquire(t.Context(), impl), "first acquire must succeed")
			err := acquire(t.Context(), impl)
			testkit.True(t, err != nil && !errors.Is(err, context.Canceled),
				fmt.Sprintf("second acquire without release must fail (got %v)", err))
		})
	}
}
