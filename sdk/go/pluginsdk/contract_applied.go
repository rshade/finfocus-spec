// Copyright 2026 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pluginsdk

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ContractAppliedElement is one object in the FOCUS ContractApplied Elements array.
// Nil applied-cost, quantity, and unit pointers are emitted as JSON null, which the
// column allows. ContractID is omitted when empty.
type ContractAppliedElement struct {
	ContractID      string
	CommitmentID    string
	AppliedCost     *float64
	AppliedQuantity *float64
	AppliedUnit     *string
}

type contractAppliedDocument struct {
	Elements []contractAppliedElement `json:"Elements"`
}

type contractAppliedElement struct {
	ContractID                        string   `json:"ContractID,omitempty"`
	ContractCommitmentID              string   `json:"ContractCommitmentID"`
	ContractCommitmentAppliedCost     *float64 `json:"ContractCommitmentAppliedCost"`
	ContractCommitmentAppliedQuantity *float64 `json:"ContractCommitmentAppliedQuantity"`
	ContractCommitmentAppliedUnit     *string  `json:"ContractCommitmentAppliedUnit"`
}

// FormatContractApplied returns the FOCUS 1.4 ContractAppliedObject JSON for elements.
// The document is {"Elements":[...]} and each element carries ContractCommitmentID,
// ContractCommitmentAppliedCost, ContractCommitmentAppliedQuantity, and
// ContractCommitmentAppliedUnit. An empty list or a missing commitment ID is an error.
func FormatContractApplied(elements []ContractAppliedElement) (string, error) {
	if len(elements) == 0 {
		return "", errors.New("contract applied requires at least one element")
	}
	document := contractAppliedDocument{Elements: make([]contractAppliedElement, len(elements))}
	for i, element := range elements {
		if element.CommitmentID == "" {
			return "", fmt.Errorf("contract applied element %d requires a commitment ID", i)
		}
		document.Elements[i] = contractAppliedElement{
			ContractID:                        element.ContractID,
			ContractCommitmentID:              element.CommitmentID,
			ContractCommitmentAppliedCost:     element.AppliedCost,
			ContractCommitmentAppliedQuantity: element.AppliedQuantity,
			ContractCommitmentAppliedUnit:     element.AppliedUnit,
		}
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("contract applied: %w", err)
	}
	return string(raw), nil
}

// WithContractAppliedObject stores a ContractApplied JSON object on the cost record.
// The setter does not validate; use FormatContractApplied to build a conforming value.
// The field stays a string, so a legacy bare commitment ID is still representable.
func (b *FocusRecordBuilder) WithContractAppliedObject(object string) *FocusRecordBuilder {
	b.record.ContractApplied = object
	return b
}
