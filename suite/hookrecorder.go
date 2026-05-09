// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"sync"
)

// hookRecorderKey is the unexported context key under which a
// [HookRecorder] travels through call sites. Production code that
// fires hooks reads the recorder via [RecorderFromContext] — when
// absent (production path), the returned nil recorder's Record
// method is a no-op, so production cost is zero.
type hookRecorderKey struct{}

// HookRecorder is a typed registry impls write to during hook
// execution. Test code constructs a recorder, attaches it to a
// context via [WithHookRecorder] (a context decorator, not the
// Option of the same name), passes the context into the impl, and
// reads the captured fire counts via [HookRecorder.Count].
//
// Goroutine-safe: writes use a mutex, reads return a snapshot.
type HookRecorder struct {
	mu     sync.Mutex
	counts map[string]int
}

// NewHookRecorder returns an empty recorder ready for use.
func NewHookRecorder() *HookRecorder {
	return &HookRecorder{counts: make(map[string]int)}
}

// Record increments the fire count for the named hook. Safe to
// call from any goroutine. Calls to Record on a nil recorder are
// no-ops, so production code reading a recorder from context with
// no test wiring incurs no allocation overhead beyond the context
// lookup.
func (r *HookRecorder) Record(name string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[name]++
}

// Count returns the fire count for the named hook. Returns 0 when
// the hook hasn't fired or the recorder is nil.
func (r *HookRecorder) Count(name string) int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counts[name]
}

// ContextWithRecorder returns a context that carries `recorder`
// under [hookRecorderKey]. Tests use this to thread a recorder
// into impls that read it via [RecorderFromContext].
func ContextWithRecorder(ctx context.Context, recorder *HookRecorder) context.Context {
	return context.WithValue(ctx, hookRecorderKey{}, recorder)
}

// RecorderFromContext returns the [HookRecorder] attached to ctx,
// or nil when none is present. Production hook-firing code should
// call this and call [HookRecorder.Record] on the result — the nil
// guard inside Record makes the production path a no-op.
func RecorderFromContext(ctx context.Context) *HookRecorder {
	if v := ctx.Value(hookRecorderKey{}); v != nil {
		if r, ok := v.(*HookRecorder); ok {
			return r
		}
	}
	return nil
}
