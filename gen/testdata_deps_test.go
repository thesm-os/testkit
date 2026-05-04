// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

// Blank imports to keep go mod tidy from dropping dependencies
// used only by testdata packages (which tidy skips).
import (
	_ "pgregory.net/rapid"

	_ "go.thesmos.sh/testkit/model"
	_ "go.thesmos.sh/testkit/model/action"
	_ "go.thesmos.sh/testkit/model/law"
	_ "go.thesmos.sh/testkit/model/refmap"
)
