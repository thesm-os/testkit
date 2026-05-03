// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import "sync"

// faultInjector is the internal counter primitive used by [CountedFault].
type faultInjector struct {
	faultErr   error
	mu         sync.Mutex
	failEveryN int
	count      int
}

func newFaultInjector(err error, n int) faultInjector {
	return faultInjector{
		faultErr:   err,
		failEveryN: n,
	}
}

func (f *faultInjector) shouldFire() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failEveryN <= 0 {
		return false
	}
	f.count++
	return f.count%f.failEveryN == 0
}

func (f *faultInjector) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count = 0
}
