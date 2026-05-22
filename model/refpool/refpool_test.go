// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package refpool_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/refpool"
)

type conn struct{ id int }

var errDoublePut = errors.New("double put")

func TestBalancedPool(t *testing.T) {
	t.Parallel()

	t.Run("Get then Put balances", func(t *testing.T) {
		t.Parallel()
		next := 0
		p := refpool.NewBalancedPool(
			func() *conn { next++; return &conn{id: next} },
			func(c *conn) any { return c },
			errDoublePut,
		)
		c, _ := p.Get(t.Context())
		testkit.NoError(t, p.Put(t.Context(), c), "put")
		testkit.True(t, p.Balanced(), "balanced after one cycle")
	})

	t.Run("free pool reuses returned resources", func(t *testing.T) {
		t.Parallel()
		next := 0
		p := refpool.NewBalancedPool(
			func() *conn { next++; return &conn{id: next} },
			func(c *conn) any { return c },
			errDoublePut,
		)
		c1, _ := p.Get(t.Context())
		_ = p.Put(t.Context(), c1)
		c2, _ := p.Get(t.Context())
		testkit.Equal(t, c1, c2, "reused")
		testkit.Equal(t, next, 1, "make called once")
	})

	t.Run("double Put returns the configured error", func(t *testing.T) {
		t.Parallel()
		p := refpool.NewBalancedPool(
			func() *conn { return &conn{} },
			func(c *conn) any { return c },
			errDoublePut,
		)
		c, _ := p.Get(t.Context())
		_ = p.Put(t.Context(), c)
		err := p.Put(t.Context(), c)
		testkit.True(t, errors.Is(err, errDoublePut), "double-put error")
	})

	t.Run("Stats tracks lifetime counts and outstanding", func(t *testing.T) {
		t.Parallel()
		p := refpool.NewBalancedPool(
			func() *conn { return &conn{} },
			func(c *conn) any { return c },
			errDoublePut,
		)
		c1, _ := p.Get(t.Context())
		_, _ = p.Get(t.Context())
		_ = p.Put(t.Context(), c1)
		g, pu, out := p.Stats()
		testkit.Equal(t, g, 2, "two gets")
		testkit.Equal(t, pu, 1, "one put")
		testkit.Equal(t, out, 1, "one outstanding")
	})
}
