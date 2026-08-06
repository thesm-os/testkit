// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package handwritten

import "errors"

// ErrUnknownWeekday is the author's own parse sentinel, named the way the
// generator would have named it — which is what makes a generated one a
// redeclaration rather than a harmless duplicate.
var ErrUnknownWeekday = errors.New("handwritten: unknown weekday")
