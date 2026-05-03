// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/richstruct"
	"go.thesmos.sh/testkit/gen/model/testdata/richstruct/storetest"
)

func TestInMemoryStoreModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 rich struct", func(t *testing.T) {
		t.Parallel()
		// Document has 20+ fields across every common Go type:
		//   - primitives: string, bool, int, int32, int64, float32, float64
		//   - named types: Priority (int), Tag (string)
		//   - nested structs: Address, GeoPoint, Metadata
		//   - pointer to struct: *GeoPoint
		//   - slices: []string, []Tag, []byte
		//   - maps: map[string]string (in Metadata.Labels and Attributes)
		//   - pointer to primitive: *string
		//   - unsigned: uint8, uint32
		//
		// rapid.MakeCustom generates ALL fields via reflection. Only ID
		// is overridden to draw from the key pool. ReadAfterWrite uses
		// cmp.Diff on the full struct, so any field corruption is caught.
		storetest.AssertStoreModel(t, func() richstruct.Store {
			return richstruct.NewInMemoryStore()
		})
	})
}
