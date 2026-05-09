// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestStripPrefix(t *testing.T) {
	t.Parallel()

	t.Run("strips matching type prefix", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stripPrefix("StatusPending", "Status"), "Pending",
			"prefix removed")
	})

	t.Run("strips trailing underscore after prefix", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stripPrefix("Status_Pending", "Status"), "Pending",
			"underscore stripped after prefix")
	})

	t.Run("returns input unchanged when prefix is absent", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stripPrefix("OtherName", "Status"), "OtherName",
			"no prefix → no change")
	})
}

func TestStripTestSuffix(t *testing.T) {
	t.Parallel()

	t.Run("trims _test.go", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stripTestSuffix("status.gen_test.go"), "status.gen",
			"_test.go path")
	})

	t.Run("falls back to .go", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stripTestSuffix("status.go"), "status",
			".go fallback")
	})
}

func TestEmitWireGolden_EmptyDataShortCircuits(t *testing.T) {
	t.Parallel()
	// HasContent() is false when there are no enums; emitWireGolden
	// short-circuits without producing a file.
	files, err := emitWireGolden(&Data{}, generator.Options{Output: "x.gen_test.go"})
	testkit.NoError(t, err, "no error on empty data")
	testkit.Len(t, files, 0, "no auxiliary files emitted")
}
