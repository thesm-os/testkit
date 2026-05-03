// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"fmt"
	"sync"
	"testing"
)

// OrderTracker enforces call-order constraints across methods on a stub.
// Methods annotated with //testkit:order-after must be called after their
// prerequisite. In strict mode, violations fatal the test; in lenient mode,
// ordering is not enforced.
//
// Generated stubs embed an OrderTracker when any method has an order-after
// directive. Each method's dispatch calls [OrderTracker.AssertAfter] before
// proceeding and [OrderTracker.Record] after recording the call.
type OrderTracker struct {
	mu     sync.Mutex
	called map[string]bool
	strict bool
	tb     testing.TB
}

// NewOrderTracker returns a new [OrderTracker]. Pass strict=true to enforce
// ordering constraints; pass strict=false to skip enforcement (lenient mode).
//
//nolint:thelper // constructor, not a test helper — tb may be nil
func NewOrderTracker(tb testing.TB, strict bool) *OrderTracker {
	return &OrderTracker{
		called: make(map[string]bool),
		strict: strict,
		tb:     tb,
	}
}

// Record marks a method as having been called. In lenient mode (strict=false),
// Record is a no-op — no allocation, no map write.
func (o *OrderTracker) Record(method string) {
	if !o.strict {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.called[method] = true
}

// AssertAfter checks that the prerequisite method has been called. In strict
// mode, fatals the test if the prerequisite hasn't been called yet. In lenient
// mode, this is a no-op.
func (o *OrderTracker) AssertAfter(method, prerequisite string) {
	if !o.strict || o.tb == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.called[prerequisite] {
		o.tb.Fatalf("%s: must be called after %s (order-after constraint)", method, prerequisite)
	}
}

// Reset clears the call history.
func (o *OrderTracker) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.called = make(map[string]bool)
}

// Called reports whether the named method has been called.
func (o *OrderTracker) Called(method string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.called[method]
}

// String returns a debug representation of called methods.
func (o *OrderTracker) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	methods := make([]string, 0, len(o.called))
	for m := range o.called {
		methods = append(methods, m)
	}
	return fmt.Sprintf("OrderTracker{called: %v}", methods)
}
