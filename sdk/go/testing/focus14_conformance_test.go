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

package testing_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// =============================================================================
// FOCUS 1.4 Cost and Usage Conformance Tests
// =============================================================================
//
// FOCUS 1.4 removes the ProviderName and PublisherName columns and adds
// InvoiceDetailId and CommitmentProgramEligibilityDetails to Cost and Usage rows.

const focus14Eligibility = `{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}`

// buildValidFocus14Record returns a builder for a valid FOCUS 1.4 cost row. It sets
// every mandatory column but names the provider only through service_provider_name,
// because FOCUS 1.4 removes ProviderName.
func buildValidFocus14Record() *pluginsdk.FocusRecordBuilder {
	billingStart := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	billingEnd := billingStart.AddDate(0, 1, 0)

	return pluginsdk.NewFocusRecordBuilder().
		WithIdentity("", "123456789012", "Production").
		WithServiceProvider("AWS").
		WithBillingPeriod(billingStart, billingEnd, "USD").
		WithChargePeriod(billingStart, billingStart.Add(24*time.Hour)).
		WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2").
		WithChargeDetails(
			pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		).
		WithChargeClassification(
			pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
			"EC2 usage",
			pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
		).
		WithUsage(100, "Hours").
		WithFinancials(100.00, 100.00, 95.00, "USD", "INV-2026-09").
		WithContractedCost(95.00)
}

// TestFocus14_ProviderRule verifies that service_provider_name satisfies the
// mandatory provider rule and that the deprecated provider_name still does.
func TestFocus14_ProviderRule(t *testing.T) {
	t.Run("service provider only (FOCUS 1.4)", func(t *testing.T) {
		record, err := buildValidFocus14Record().Build()
		require.NoError(t, err)
		//nolint:staticcheck // SA1019: asserting the removed column stays empty
		require.Empty(t, record.GetProviderName())
		require.Equal(t, "AWS", record.GetServiceProviderName())
	})

	t.Run("deprecated provider only (FOCUS 1.2)", func(t *testing.T) {
		record, err := buildValidFocus14Record().
			WithIdentity("AWS", "123456789012", "Production").
			WithServiceProvider("").
			Build()
		require.NoError(t, err)
		//nolint:staticcheck // SA1019: FOCUS 1.2 records keep working
		require.Equal(t, "AWS", record.GetProviderName())
	})

	t.Run("both set", func(t *testing.T) {
		_, err := buildValidFocus14Record().
			WithIdentity("AWS", "123456789012", "Production").
			Build()
		require.NoError(t, err)
	})

	t.Run("neither set", func(t *testing.T) {
		_, err := buildValidFocus14Record().WithServiceProvider("").Build()
		var valErr *pluginsdk.ValidationError
		require.ErrorAs(t, err, &valErr)
		require.Equal(t, "provider_name", valErr.FieldName)
		require.Contains(t, valErr.ExpectedValue, "service_provider_name")
	})
}

// TestFocus14_InvoiceDetailID verifies InvoiceDetailId round-trips through the
// builder and the wire, and requires InvoiceId.
func TestFocus14_InvoiceDetailID(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		record, err := buildValidFocus14Record().WithInvoiceDetailID("INV-2026-09-L3").Build()
		require.NoError(t, err)
		require.Equal(t, "INV-2026-09-L3", record.GetInvoiceDetailId())

		wire, err := proto.Marshal(record)
		require.NoError(t, err)
		decoded := &pbc.FocusCostRecord{}
		require.NoError(t, proto.Unmarshal(wire, decoded))
		require.True(t, proto.Equal(record, decoded), "record must survive the wire unchanged")
		require.NoError(t, pluginsdk.ValidateFocusRecord(decoded))
	})

	t.Run("requires invoice_id", func(t *testing.T) {
		_, err := buildValidFocus14Record().
			WithInvoice("", "AWS").
			WithInvoiceDetailID("INV-2026-09-L3").
			Build()
		require.ErrorIs(t, err, pluginsdk.ErrInvoiceIDMissingForInvoiceDetail)
		var valErr *pluginsdk.ValidationError
		require.ErrorAs(t, err, &valErr)
		require.Equal(t, "invoice_id", valErr.FieldName)
	})
}

// TestFocus14_CommitmentProgramEligibilityDetails verifies that only a well-formed
// JSON object is accepted and that it round-trips unchanged.
func TestFocus14_CommitmentProgramEligibilityDetails(t *testing.T) {
	t.Run("valid object round trip", func(t *testing.T) {
		record, err := buildValidFocus14Record().
			WithCommitmentProgramEligibilityDetails(focus14Eligibility).
			Build()
		require.NoError(t, err)
		require.JSONEq(t, focus14Eligibility, record.GetCommitmentProgramEligibilityDetails())

		wire, err := proto.Marshal(record)
		require.NoError(t, err)
		decoded := &pbc.FocusCostRecord{}
		require.NoError(t, proto.Unmarshal(wire, decoded))
		require.True(t, proto.Equal(record, decoded), "eligibility details must survive the wire unchanged")
	})

	for name, details := range map[string]string{
		"malformed":  `{"CommitmentPrograms":[`,
		"array":      `[{"ProgramType":"Savings Plan"}]`,
		"null":       `null`,
		"bare value": `"Savings Plan"`,
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := buildValidFocus14Record().
				WithCommitmentProgramEligibilityDetails(details).
				Build()
			require.ErrorIs(t, err, pluginsdk.ErrInvalidCommitmentProgramEligibilityDetails)
			var valErr *pluginsdk.ValidationError
			require.ErrorAs(t, err, &valErr)
			require.Equal(t, "commitment_program_eligibility_details", valErr.FieldName)
		})
	}
}

// TestFocus14_AggregateMode verifies both FOCUS 1.4 rules are reported together in
// aggregate mode.
func TestFocus14_AggregateMode(t *testing.T) {
	record, err := buildValidFocus14Record().Build()
	require.NoError(t, err)
	record.InvoiceId = ""
	record.InvoiceDetailId = "INV-2026-09-L3"
	record.CommitmentProgramEligibilityDetails = `[]`

	errs := pluginsdk.ValidateFocusRecordWithOptions(record, pluginsdk.ValidationOptions{
		Mode: pluginsdk.ValidationModeAggregate,
	})

	var sawEligibility, sawInvoice bool
	for _, e := range errs {
		sawEligibility = sawEligibility || errors.Is(e, pluginsdk.ErrInvalidCommitmentProgramEligibilityDetails)
		sawInvoice = sawInvoice || errors.Is(e, pluginsdk.ErrInvoiceIDMissingForInvoiceDetail)
	}
	require.True(t, sawEligibility, "aggregate mode must report the eligibility error: %v", errs)
	require.True(t, sawInvoice, "aggregate mode must report the invoice error: %v", errs)
}

// TestFocus14_FieldNamesMatchProto verifies FocusFieldNames lists exactly the fields
// of FocusCostRecord, including the FOCUS 1.4 additions.
func TestFocus14_FieldNamesMatchProto(t *testing.T) {
	fields := (&pbc.FocusCostRecord{}).ProtoReflect().Descriptor().Fields()
	protoNames := make([]string, 0, fields.Len())
	for i := range fields.Len() {
		protoNames = append(protoNames, string(fields.Get(i).Name()))
	}

	sdkNames := append([]string(nil), pluginsdk.FocusFieldNames()...)
	sort.Strings(protoNames)
	sort.Strings(sdkNames)
	require.Equal(t, protoNames, sdkNames)
	require.Contains(t, sdkNames, "invoice_detail_id")
	require.Contains(t, sdkNames, "commitment_program_eligibility_details")
}

// TestFocus14_MockDryRunFieldParity verifies the mock plugin's default dry-run field
// mappings describe the same field set as the SDK.
func TestFocus14_MockDryRunFieldParity(t *testing.T) {
	harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
	harness.Start(t)
	defer harness.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := harness.Client().DryRun(ctx, &pbc.DryRunRequest{
		Resource: &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2", Region: "us-east-1"},
	})
	require.NoError(t, err)

	mockNames := make([]string, 0, len(resp.GetFieldMappings()))
	for _, m := range resp.GetFieldMappings() {
		mockNames = append(mockNames, m.GetFieldName())
	}
	sdkNames := append([]string(nil), pluginsdk.FocusFieldNames()...)
	sort.Strings(mockNames)
	sort.Strings(sdkNames)
	require.Equal(t, sdkNames, mockNames)
}

// TestFocus14_ContractCommitmentColumns checks fields 13-30 and that a baseline
// spend commitment distinguishes a present 0 discount from an unset one.
func TestFocus14_ContractCommitmentColumns(t *testing.T) {
	fields := (&pbc.ContractCommitment{}).ProtoReflect().Descriptor().Fields()
	want := map[int]string{
		13: "contract_commitment_applicability",
		14: "contract_commitment_benefit_category",
		15: "contract_commitment_created",
		16: "contract_commitment_discount_percentage",
		17: "contract_commitment_duration_type",
		18: "contract_commitment_fulfillment_interval",
		19: "contract_commitment_last_updated",
		20: "contract_commitment_lifecycle_status",
		21: "contract_commitment_model",
		22: "contract_commitment_offer_category",
		23: "contract_commitment_payment_interval",
		24: "contract_commitment_payment_model",
		25: "contract_commitment_payment_upfront_percentage",
		26: "invoice_issuer_name",
		27: "pricing_currency",
		28: "pricing_currency_contract_commitment_cost",
		29: "service_provider_name",
		30: "contract_commitment_description",
	}
	for number, name := range want {
		field := fields.ByNumber(protoreflect.FieldNumber(number))
		require.NotNil(t, field, "field %d", number)
		require.Equal(t, name, string(field.Name()))
	}

	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	record, err := pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commit-1", "contract-1").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
		WithFinancials(10, 0, "", "USD").
		WithBaselineTerms(at).
		Build()
	require.NoError(t, err)
	require.NotNil(t, record.ContractCommitmentDiscountPercentage)
	require.Zero(t, record.GetContractCommitmentDiscountPercentage())

	record.ContractCommitmentDiscountPercentage = nil
	err = plugintesting.ValidateContractCommitment(record)
	require.ErrorContains(t, err, "contract_commitment_discount_percentage is required")
}
