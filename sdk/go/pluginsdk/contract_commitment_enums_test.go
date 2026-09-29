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

package pluginsdk_test

import (
	"strings"
	"testing"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestContractCommitmentEnumValidators(t *testing.T) {
	if !pluginsdk.IsValidContractCommitmentBenefitCategory(
		pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_DISCOUNT) {
		t.Fatal("DISCOUNT should be valid")
	}
	if pluginsdk.IsValidContractCommitmentBenefitCategory(
		pbc.FocusContractCommitmentBenefitCategory(99)) {
		t.Fatal("unknown benefit category should be invalid")
	}
	if !pluginsdk.IsValidContractCommitmentFulfillmentInterval(
		pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_FULL_PERIOD) {
		t.Fatal("FULL_PERIOD should be valid")
	}
	if !pluginsdk.IsValidContractCommitmentLifecycleStatus(
		pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_CANCELED) {
		t.Fatal("CANCELED should be valid")
	}
	if !pluginsdk.IsValidContractCommitmentModel(
		pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_DISCONTINUOUS) {
		t.Fatal("DISCONTINUOUS should be valid")
	}
	if !pluginsdk.IsValidContractCommitmentOfferCategory(
		pbc.FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_NEGOTIATED) {
		t.Fatal("NEGOTIATED should be valid")
	}
	if !pluginsdk.IsValidContractCommitmentPaymentInterval(
		pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_ONE_TIME) {
		t.Fatal("ONE_TIME should be valid")
	}
	if !pluginsdk.IsValidContractCommitmentPaymentModel(
		pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_PARTIAL_UPFRONT) {
		t.Fatal("PARTIAL_UPFRONT should be valid")
	}
}

//nolint:gocognit // one sub-benchmark per enum; each body is the same membership check
func BenchmarkIsValidContractCommitmentEnums(b *testing.B) {
	benefit := pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_DISCOUNT
	interval := pbc.
		FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_MONTHLY
	status := pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_ACTIVE
	model := pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_CONTINUOUS
	offer := pbc.FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_PUBLIC
	payInterval := pbc.
		FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_MONTHLY
	payModel := pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_NO_UPFRONT

	b.Run("benefit", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentBenefitCategory(benefit) {
				b.Fatal("invalid")
			}
		}
	})
	b.Run("fulfillment", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentFulfillmentInterval(interval) {
				b.Fatal("invalid")
			}
		}
	})
	b.Run("lifecycle", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentLifecycleStatus(status) {
				b.Fatal("invalid")
			}
		}
	})
	b.Run("model", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentModel(model) {
				b.Fatal("invalid")
			}
		}
	})
	b.Run("offer", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentOfferCategory(offer) {
				b.Fatal("invalid")
			}
		}
	})
	b.Run("payment-interval", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentPaymentInterval(payInterval) {
				b.Fatal("invalid")
			}
		}
	})
	b.Run("payment-model", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if !pluginsdk.IsValidContractCommitmentPaymentModel(payModel) {
				b.Fatal("invalid")
			}
		}
	})
}

func TestFormatContractApplied(t *testing.T) {
	cost := 12.5
	raw, err := pluginsdk.FormatContractApplied([]pluginsdk.ContractAppliedElement{{
		ContractID:   "contract-1",
		CommitmentID: "commit-1",
		AppliedCost:  &cost,
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"Elements":[{"ContractId":"contract-1","ContractCommitmentId":"commit-1",` +
		`"ContractCommitmentAppliedCost":12.5}]}`
	if raw != want {
		t.Fatalf("got %s", raw)
	}
	if _, emptyErr := pluginsdk.FormatContractApplied(nil); emptyErr == nil {
		t.Fatal("expected an error for no elements")
	}
}

func TestFormatContractApplied_MetricRules(t *testing.T) {
	zero := 0.0
	hours := "Hours"
	quantity := 3.0
	cost := 1.0
	blank := " "

	raw, err := pluginsdk.FormatContractApplied([]pluginsdk.ContractAppliedElement{{
		ContractID:   "contract-1",
		CommitmentID: "commit-1",
		AppliedCost:  &zero,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, `"ContractCommitmentAppliedCost":0`) {
		t.Fatalf("zero cost omitted: %s", raw)
	}

	raw, err = pluginsdk.FormatContractApplied([]pluginsdk.ContractAppliedElement{{
		ContractID:      "contract-1",
		CommitmentID:    "commit-1",
		AppliedCost:     &cost,
		AppliedQuantity: &quantity,
		AppliedUnit:     &hours,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, `"ContractCommitmentAppliedQuantity":3`) ||
		!strings.Contains(raw, `"ContractCommitmentAppliedUnit":"Hours"`) {
		t.Fatalf("both metrics should be kept: %s", raw)
	}

	cases := []pluginsdk.ContractAppliedElement{
		{CommitmentID: "commit-1", AppliedCost: &cost},
		{ContractID: blank, CommitmentID: "commit-1", AppliedCost: &cost},
		{ContractID: "contract-1", CommitmentID: "commit-1"},
		{ContractID: "contract-1", CommitmentID: "commit-1", AppliedQuantity: &quantity},
		{ContractID: "contract-1", CommitmentID: "commit-1", AppliedCost: &cost, AppliedUnit: &hours},
		{ContractID: "contract-1", CommitmentID: "commit-1", AppliedQuantity: &zero, AppliedUnit: &blank},
	}
	for i, element := range cases {
		if _, caseErr := pluginsdk.FormatContractApplied([]pluginsdk.ContractAppliedElement{element}); caseErr == nil {
			t.Fatalf("case %d accepted an incomplete element", i)
		}
	}
}
