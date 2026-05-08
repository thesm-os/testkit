// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// lifecycleDetector matches `func(ctx) error` — ctx required, no
// other parameters, error-only return. Models Open/Close/Flush/
// Sync style hooks that observe ctx and report failure.
//
// PoisonAccessor (priority 830) claims the no-ctx form `func() error`;
// VoidLifecycle (priority 810) claims the no-error form
// `func()` / `func(ctx)`.
type lifecycleDetector struct{}

func (lifecycleDetector) Name() string  { return "Lifecycle" }
func (lifecycleDetector) Priority() int { return PriorityLifecycle }

func (lifecycleDetector) Detect(s Signature) (Info, bool) {
	if !s.HasCtx || s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 0 {
		return Info{}, false
	}
	return Info{Shape: Lifecycle}, true
}
