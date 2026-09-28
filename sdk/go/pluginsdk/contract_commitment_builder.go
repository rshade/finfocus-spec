package pluginsdk

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// ContractCommitmentBuilder handles the construction of FOCUS 1.3 ContractCommitment records.
// This builder creates records for the Contract Commitment supplemental dataset,
// which tracks contractual obligations separately from cost line items.
//
// Reference: FOCUS 1.3 Contract Commitment Dataset.
type ContractCommitmentBuilder struct {
	record *pbc.ContractCommitment
}

// NewContractCommitmentBuilder creates a new builder instance for ContractCommitment records.
func NewContractCommitmentBuilder() *ContractCommitmentBuilder {
	return &ContractCommitmentBuilder{
		record: &pbc.ContractCommitment{},
	}
}

// WithIdentity sets the identity fields for the contract commitment.
// ContractCommitmentId is the unique identifier for this specific commitment (REQUIRED).
// ContractId is the identifier of the parent contract containing this commitment (REQUIRED).
// FOCUS 1.3 Section: Contract Commitment ID, Contract ID.
func (b *ContractCommitmentBuilder) WithIdentity(
	commitmentID, contractID string,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentId = commitmentID
	b.record.ContractId = contractID
	return b
}

// WithCategory sets the commitment category.
// Category indicates whether this is a SPEND or USAGE commitment.
// FOCUS 1.3 Section: Contract Commitment Category.
func (b *ContractCommitmentBuilder) WithCategory(
	category pbc.FocusContractCommitmentCategory,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentCategory = category
	return b
}

// WithType sets the provider-specific commitment type.
// Examples: "Reserved Instance", "Savings Plan", "Committed Use Discount", "Enterprise Agreement"
// FOCUS 1.3 Section: Contract Commitment Type.
func (b *ContractCommitmentBuilder) WithType(
	commitmentType string,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentType = commitmentType
	return b
}

// WithCommitmentPeriod sets the start and end of the commitment period.
// This is when the specific commitment obligations are active.
// FOCUS 1.3 Section: Contract Commitment Period Start/End.
func (b *ContractCommitmentBuilder) WithCommitmentPeriod(
	start, end time.Time,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentPeriodStart = timestamppb.New(start)
	b.record.ContractCommitmentPeriodEnd = timestamppb.New(end)
	return b
}

// WithContractPeriod sets the start and end of the overall contract period.
// This is when the parent contract agreement is active.
// FOCUS 1.3 Section: Contract Period Start/End.
func (b *ContractCommitmentBuilder) WithContractPeriod(
	start, end time.Time,
) *ContractCommitmentBuilder {
	b.record.ContractPeriodStart = timestamppb.New(start)
	b.record.ContractPeriodEnd = timestamppb.New(end)
	return b
}

// WithFinancials sets all financial fields for the commitment.
// - cost: Monetary amount of the commitment (for SPEND category)
// - quantity: Quantity amount of the commitment (for USAGE category)
// - unit: Unit of measure for quantity (e.g., "Hours", "GB", "vCPU-Hours")
// - currencyCode: ISO 4217 currency code for monetary values (REQUIRED)
//
// For granular control, use WithCost(), WithQuantity(), and WithCurrency() instead.
// FOCUS 1.3 Section: Contract Commitment Cost, Quantity, Unit, Billing Currency.
func (b *ContractCommitmentBuilder) WithFinancials(
	cost, quantity float64, unit, currencyCode string,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentCost = cost
	b.record.ContractCommitmentQuantity = quantity
	b.record.ContractCommitmentUnit = unit
	b.record.BillingCurrency = currencyCode
	return b
}

// WithCost sets the monetary commitment amount for SPEND category commitments.
// Use this for commitments based on spend thresholds (e.g., "$100,000/year").
// FOCUS 1.3 Section: Contract Commitment Cost.
func (b *ContractCommitmentBuilder) WithCost(cost float64) *ContractCommitmentBuilder {
	b.record.ContractCommitmentCost = cost
	return b
}

// WithQuantity sets the quantity and unit for USAGE category commitments.
// Use this for commitments based on consumption (e.g., "1000 vCPU-Hours").
// FOCUS 1.3 Section: Contract Commitment Quantity, Contract Commitment Unit.
func (b *ContractCommitmentBuilder) WithQuantity(quantity float64, unit string) *ContractCommitmentBuilder {
	b.record.ContractCommitmentQuantity = quantity
	b.record.ContractCommitmentUnit = unit
	return b
}

// WithCurrency sets the ISO 4217 currency code for the commitment.
// This field is REQUIRED for all commitments.
// FOCUS 1.3 Section: Billing Currency.
func (b *ContractCommitmentBuilder) WithCurrency(currencyCode string) *ContractCommitmentBuilder {
	b.record.BillingCurrency = currencyCode
	return b
}

// Build validates and returns the constructed ContractCommitment record.
// Returns an error if required fields are missing or validation rules are
// violated; the rules are those of ValidateContractCommitment.
func (b *ContractCommitmentBuilder) Build() (*pbc.ContractCommitment, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return b.record, nil
}

// validate checks the record with ValidateContractCommitment, the rule set
// hosts and the conformance suite also apply.
func (b *ContractCommitmentBuilder) validate() error {
	return plugintesting.ValidateContractCommitment(b.record)
}
