// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hashertest

import "go.thesmos.sh/testkit/gen/testdata/samples"

// TestSampleDigest returns a Digest for testing. Defined in the output
// package to exercise the unqualified sample resolution path — the
// generator must emit this without a package qualifier.
func TestSampleDigest(_ samples.Hasher) samples.Digest {
	var d samples.Digest
	d[0] = 0xFF
	return d
}
