// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/directive"
)

type fakeHandler struct {
	name   string
	output *directive.Output
}

func (h *fakeHandler) Name() string { return h.name }

func (h *fakeHandler) Process(_ directive.Context) (*directive.Output, error) {
	return h.output, nil
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register and Get", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		h := &fakeHandler{name: "errors"}
		r.Register(h)

		got := r.Get("errors")
		testkit.True(t, got == h, "must return registered handler")
	})

	t.Run("Get returns nil for unknown", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		testkit.True(t, r.Get("nonexistent") == nil, "must return nil")
	})

	t.Run("Names returns sorted list", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		r.Register(&fakeHandler{name: "errors"})
		r.Register(&fakeHandler{name: "concurrent"})
		r.Register(&fakeHandler{name: "allocs"})

		testkit.Equal(t, r.Names(), []string{"allocs", "concurrent", "errors"}, "must be sorted")
	})

	t.Run("duplicate registration panics", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		r.Register(&fakeHandler{name: "errors"})

		testkit.Panics(t, func() {
			r.Register(&fakeHandler{name: "errors"})
		}, "must panic on duplicate")
	})

	t.Run("empty registry has no names", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		testkit.Len(t, r.Names(), 0, "empty registry")
	})
}

func TestProcess(t *testing.T) {
	t.Parallel()

	t.Run("calls registered handler", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		r.Register(&fakeHandler{
			name: "errors",
			output: &directive.Output{
				Blocks: []directive.Block{
					{ExtensionPoint: "stub-options", Content: "// injected"},
				},
			},
		})

		out, err := r.Process("errors", directive.Context{
			Args:          []string{"ErrNotFound"},
			Generator:     "stub",
			MethodName:    "Get",
			InterfaceName: "Store",
		})
		testkit.NoError(t, err, "must succeed")
		testkit.Len(t, out.Blocks, 1, "must return one block")
		testkit.Equal(t, out.Blocks[0].ExtensionPoint, "stub-options", "extension point")
		testkit.Equal(t, out.Blocks[0].Content, "// injected", "content")
	})

	t.Run("unknown directive returns nil", func(t *testing.T) {
		t.Parallel()
		r := directive.NewRegistry()
		out, err := r.Process("unknown", directive.Context{})
		testkit.NoError(t, err, "must not error")
		testkit.True(t, out == nil, "must return nil for unknown")
	})
}
