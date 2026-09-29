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
	"strings"
)

// ContractAppliedElement is one object in the FOCUS 1.4 ContractApplied Elements array.
// A nil applied cost or quantity is omitted. A non-nil pointer is written, including zero.
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
	ContractID                        string   `json:"ContractId"`
	ContractCommitmentID              string   `json:"ContractCommitmentId"`
	ContractCommitmentAppliedCost     *float64 `json:"ContractCommitmentAppliedCost,omitempty"`
	ContractCommitmentAppliedQuantity *float64 `json:"ContractCommitmentAppliedQuantity,omitempty"`
	ContractCommitmentAppliedUnit     *string  `json:"ContractCommitmentAppliedUnit,omitempty"`
}

// FormatContractApplied returns the FOCUS 1.4 ContractAppliedObject JSON for elements.
// Each element includes ContractId and ContractCommitmentId. It includes a cost, a
// quantity with a unit, or both. An empty list or a missing identifier is an error.
func FormatContractApplied(elements []ContractAppliedElement) (string, error) {
	if len(elements) == 0 {
		return "", errors.New("contract applied requires at least one element")
	}
	document := contractAppliedDocument{Elements: make([]contractAppliedElement, len(elements))}
	for i, element := range elements {
		encoded, err := appliedElement(i, element)
		if err != nil {
			return "", err
		}
		document.Elements[i] = encoded
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("contract applied: %w", err)
	}
	return string(raw), nil
}

func appliedElement(index int, element ContractAppliedElement) (contractAppliedElement, error) {
	if strings.TrimSpace(element.ContractID) == "" {
		return contractAppliedElement{}, fmt.Errorf("contract applied element %d requires a contract ID", index)
	}
	if strings.TrimSpace(element.CommitmentID) == "" {
		return contractAppliedElement{}, fmt.Errorf("contract applied element %d requires a commitment ID", index)
	}
	if err := validateAppliedMetric(index, element); err != nil {
		return contractAppliedElement{}, err
	}
	return contractAppliedElement{
		ContractID:                        element.ContractID,
		ContractCommitmentID:              element.CommitmentID,
		ContractCommitmentAppliedCost:     element.AppliedCost,
		ContractCommitmentAppliedQuantity: element.AppliedQuantity,
		ContractCommitmentAppliedUnit:     element.AppliedUnit,
	}, nil
}

// validateAppliedMetric applies the FOCUS 1.4 conditions. Cost is required when
// quantity is absent. Quantity requires a unit. Zero is present. Both may be set.
func validateAppliedMetric(index int, element ContractAppliedElement) error {
	hasCost := element.AppliedCost != nil
	hasQuantity := element.AppliedQuantity != nil
	unitProvided := element.AppliedUnit != nil
	hasUnit := unitProvided && strings.TrimSpace(*element.AppliedUnit) != ""
	if !hasCost && !hasQuantity {
		return fmt.Errorf("contract applied element %d requires a cost or a quantity", index)
	}
	if hasQuantity && !hasUnit {
		return fmt.Errorf("contract applied element %d requires a unit when quantity is set", index)
	}
	if !hasQuantity && unitProvided {
		return fmt.Errorf("contract applied element %d unit requires a quantity", index)
	}
	return nil
}

// WithContractAppliedObject stores a ContractApplied JSON object on the cost record.
// The setter does not validate; use FormatContractApplied to build a conforming value.
// The field stays a string, so a legacy bare commitment ID is still representable.
func (b *FocusRecordBuilder) WithContractAppliedObject(object string) *FocusRecordBuilder {
	b.record.ContractApplied = object
	return b
}
