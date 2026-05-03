// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaultstest

import "go.thesmos.sh/testkit/gen/builder/testdata/defaults"

// RequestDefaults returns canonical test defaults for [defaults.Request].
// The generated NewRequest() builder uses this automatically.
func RequestDefaults() defaults.Request {
	return defaults.Request{
		RunID: "test-run-id",
		Token: 42,
		Data:  []byte("test-data"),
	}
}
