// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// poisonAccessorDetector matches the exact signature `func() error`:
// no ctx, no params, error-only return. Models the lazy-error-getter
// idiom (`Err()`, `LastError()`).
//
// Fires before Lifecycle/VoidLifecycle to claim this signature
// outright. Lifecycle requires ctx; VoidLifecycle requires no
// return.
type poisonAccessorDetector struct{}

func (poisonAccessorDetector) Name() string  { return "PoisonAccessor" }
func (poisonAccessorDetector) Priority() int { return PriorityPoisonAccessor }

func (poisonAccessorDetector) Detect(s Signature) (Info, bool) {
	if s.HasCtx || s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 0 {
		return Info{}, false
	}
	return Info{Shape: PoisonAccessor}, true
}
