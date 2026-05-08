// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// voidLifecycleDetector matches `func()` and `func(ctx)` — no
// non-ctx params, no return. Models `Reset()`, `Close()` (when
// they don't return error), and parameterless lifecycle hooks
// (ANALYSIS.md G31).
//
// Fires above Mutator to claim no-arg void cases. Mutator requires
// at least one non-ctx param.
type voidLifecycleDetector struct{}

func (voidLifecycleDetector) Name() string  { return "VoidLifecycle" }
func (voidLifecycleDetector) Priority() int { return PriorityVoidLifecycle }

func (voidLifecycleDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if len(s.AllResults) != 0 {
		return Info{}, false
	}
	return Info{Shape: VoidLifecycle}, true
}
