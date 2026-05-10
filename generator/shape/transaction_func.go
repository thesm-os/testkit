// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// transactionFuncDetector promotes a method to TransactionFunc when
// `//testkit:transaction` is present. The contract is "the supplied
// callback runs inside a transaction; on error the work rolls back;
// no mid-transaction state is visible to concurrent observers."
//
// Carrier signature is typically `func(ctx, func(Tx) error) error`,
// but the detector accepts any structure with the directive present
// and at least one non-ctx parameter. Tx-type resolution lives in
// the spec consumer.
type transactionFuncDetector struct{}

func (transactionFuncDetector) Name() string  { return "TransactionFunc" }
func (transactionFuncDetector) Priority() int { return PriorityContractTransactionFunc }

func (transactionFuncDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Transaction)
	if !ok || d.Off {
		return Info{}, false
	}
	if len(s.NonCtxParams) == 0 {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   TransactionFunc,
		ValType: carrier.ValType,
	}, true
}
