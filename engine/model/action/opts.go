// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action

import (
	"errors"
	"fmt"
)

// Opt configures an action's comparison beyond the default.
type Opt func(*opts)

type opts struct {
	sentinel error
}

// WithSentinel arms the identity half of a reader's error comparison: where
// both sides err, the two must also agree on whether the error is the
// declaration's own sentinel. Without it, agreement stops at presence — a
// subject answering a private error where the oracle answers the declared
// miss reads as agreement, and the wrong sentinel survives every sequence.
// The generator arms it from the declaration's stamped miss identity; a
// reference minting errors of its own stays presence-only, honestly.
func WithSentinel(err error) Opt {
	return func(o *opts) { o.sentinel = err }
}

// optsOf folds the options into their resolved form.
func optsOf(os []Opt) opts {
	var o opts
	for _, f := range os {
		f(&o)
	}
	return o
}

// identity reports the divergence where two errors — both non-nil, already
// past the presence check — disagree on being the armed sentinel. Nil where
// no sentinel is armed or the two agree.
func (o opts) identity(name string, input any, sutErr, refErr error) error {
	if o.sentinel == nil || errors.Is(sutErr, o.sentinel) == errors.Is(refErr, o.sentinel) {
		return nil
	}
	return fmt.Errorf(
		"%s(%v): SUT and ref agree the call fails and disagree on its identity: "+
			"SUT err=%w, ref err=%w (sentinel %w)",
		name, input, sutErr, refErr, o.sentinel,
	)
}
