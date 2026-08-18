// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// timeoutScale is read once: the polling laws consult it on every
// check, and a mid-run change of scale would make two iterations of
// one property disagree about the budget they ran under.
var timeoutScale = sync.OnceValue(func() float64 {
	v := os.Getenv("TESTKIT_TIMEOUT_SCALE")
	if v == "" {
		return 1
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return 1
	}
	return f
})

// scaleTimeout applies TESTKIT_TIMEOUT_SCALE to a declared time
// budget. The declaration stays the contract's number; the scale is an
// operator's statement about the hardware — CI headroom, never a
// per-test tuning knob. Unset, unparsable, or non-positive values
// scale by 1.
func scaleTimeout(d time.Duration) time.Duration {
	return time.Duration(float64(d) * timeoutScale())
}
