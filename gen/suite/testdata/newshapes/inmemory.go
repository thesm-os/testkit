// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package newshapes

import "context"

// InMemoryDevice implements [Device] for testing.
type InMemoryDevice struct {
	counters map[string]int64
	err      error
}

func NewInMemoryDevice() *InMemoryDevice {
	return &InMemoryDevice{counters: make(map[string]int64)}
}

func (d *InMemoryDevice) Increment(_ context.Context, delta int64) {
	d.counters["default"] += delta
}

func (d *InMemoryDevice) Load(_ context.Context, key string) (int64, bool) {
	v, ok := d.counters[key]
	return v, ok
}

func (d *InMemoryDevice) Inspect(key string) (int64, Metadata, bool) {
	v, ok := d.counters[key]
	if !ok {
		return 0, Metadata{}, false
	}
	return v, Metadata{Firmware: "v1.0"}, true
}

func (d *InMemoryDevice) Err() error {
	return d.err
}
