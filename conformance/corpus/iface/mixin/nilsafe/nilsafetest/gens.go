// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nilsafetest

import (
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe"
	"go.thesmos.sh/testkit/engine/model"
)

// PayloadGen is the gen= supply: the pointer payloads no reflection walk can
// spell — arbitrary bodies behind the pointer, and nil itself, which is the
// nilsafe claim's whole subject. Half the drawn writes carry a value nothing
// hand-written would think to store.
func PayloadGen() *model.Generator[*nilsafe.Payload] {
	return model.Ptr(model.Make[nilsafe.Payload](), true)
}
