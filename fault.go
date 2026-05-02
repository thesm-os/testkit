// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import "sync"

// FaultInjector provides deterministic, counter-based fault injection. It is
// designed to be embedded in generated stubs — all exported names are prefixed
// with "Fault" to prevent collisions with interface methods.
//
//	fi := testkit.NewFaultInjector(errBoom, 3) // fires on 3rd, 6th, 9th...
//	if fi.FaultShouldFire() {
//	    return fi.FaultErr
//	}
type FaultInjector struct {
	// FaultErr is the error returned when the fault fires. Exported so
	// generated code can read it directly.
	FaultErr error

	mu         sync.Mutex
	failEveryN int
	count      int
}

// NewFaultInjector returns a [FaultInjector] that fires on every nth call to
// [FaultInjector.FaultShouldFire]. A value of n <= 0 disables the injector
// (FaultShouldFire always returns false).
func NewFaultInjector(err error, n int) FaultInjector {
	return FaultInjector{
		FaultErr:   err,
		failEveryN: n,
	}
}

// FaultShouldFire increments the internal counter and reports whether the
// fault should fire on this call. Returns true on every nth call when n > 0.
func (f *FaultInjector) FaultShouldFire() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failEveryN <= 0 {
		return false
	}
	f.count++
	return f.count%f.failEveryN == 0
}

// FaultCount returns the number of times [FaultInjector.FaultShouldFire] has
// been called.
func (f *FaultInjector) FaultCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}

// FaultReset zeroes the call counter.
func (f *FaultInjector) FaultReset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count = 0
}
