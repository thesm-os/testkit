// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

// SnapshotConsumers captures the current consumer registry. Tests that
// mutate the registry (via [RegisterConsumer]) call this before
// registering and pair it with [RestoreConsumers] in t.Cleanup so
// repeated runs (go test -count=N, or sibling tests sharing the
// process) don't accumulate registrations.
func SnapshotConsumers() map[string][]Consumer {
	consumerMu.RLock()
	defer consumerMu.RUnlock()
	out := make(map[string][]Consumer, len(consumerRegistry))
	for k, v := range consumerRegistry {
		copied := make([]Consumer, len(v))
		copy(copied, v)
		out[k] = copied
	}
	return out
}

// RestoreConsumers replaces the consumer registry with snap.
func RestoreConsumers(snap map[string][]Consumer) {
	consumerMu.Lock()
	defer consumerMu.Unlock()
	consumerRegistry = make(map[string][]Consumer, len(snap))
	for k, v := range snap {
		copied := make([]Consumer, len(v))
		copy(copied, v)
		consumerRegistry[k] = copied
	}
}
