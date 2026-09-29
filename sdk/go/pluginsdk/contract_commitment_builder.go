package pluginsdk

import (
	"time"

	"google.golang.org/protobuf/proto"
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
// FOCUS requires it when the category is SPEND. USAGE may leave it empty.
// FOCUS 1.3 and 1.4 Section: Billing Currency.
func (b *ContractCommitmentBuilder) WithCurrency(currencyCode string) *ContractCommitmentBuilder {
	b.record.BillingCurrency = currencyCode
	return b
}

// WithApplicability sets the FOCUS 1.4 applicability column. applicability is a
// JSON object string. Build rejects anything that is not one JSON object.
func (b *ContractCommitmentBuilder) WithApplicability(applicability string) *ContractCommitmentBuilder {
	b.record.ContractCommitmentApplicability = applicability
	return b
}

// WithBenefitCategory sets the FOCUS 1.4 benefit category.
func (b *ContractCommitmentBuilder) WithBenefitCategory(
	category pbc.FocusContractCommitmentBenefitCategory,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentBenefitCategory = category
	return b
}

// WithCreated sets ContractCommitmentCreated.
func (b *ContractCommitmentBuilder) WithCreated(created time.Time) *ContractCommitmentBuilder {
	b.record.ContractCommitmentCreated = timestamppb.New(created)
	return b
}

// WithDiscountPercentage sets the discount fraction. Zero is stored as a present
// value, which FOCUS treats differently from null. Use ClearDiscountPercentage
// when the benefit category is Availability.
func (b *ContractCommitmentBuilder) WithDiscountPercentage(percentage float64) *ContractCommitmentBuilder {
	b.record.ContractCommitmentDiscountPercentage = proto.Float64(percentage)
	return b
}

// ClearDiscountPercentage marks the discount percentage null.
func (b *ContractCommitmentBuilder) ClearDiscountPercentage() *ContractCommitmentBuilder {
	b.record.ContractCommitmentDiscountPercentage = nil
	return b
}

// WithDurationType sets the FOCUS duration, for example "3 Years".
func (b *ContractCommitmentBuilder) WithDurationType(duration string) *ContractCommitmentBuilder {
	b.record.ContractCommitmentDurationType = duration
	return b
}

// WithFulfillmentInterval sets how often the commitment is fulfilled.
func (b *ContractCommitmentBuilder) WithFulfillmentInterval(
	interval pbc.FocusContractCommitmentFulfillmentInterval,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentFulfillmentInterval = interval
	return b
}

// WithLastUpdated sets ContractCommitmentLastUpdated.
func (b *ContractCommitmentBuilder) WithLastUpdated(updated time.Time) *ContractCommitmentBuilder {
	b.record.ContractCommitmentLastUpdated = timestamppb.New(updated)
	return b
}

// WithLifecycleStatus sets the FOCUS 1.4 lifecycle status.
func (b *ContractCommitmentBuilder) WithLifecycleStatus(
	status pbc.FocusContractCommitmentLifecycleStatus,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentLifecycleStatus = status
	return b
}

// WithModel sets Continuous or Discontinuous.
func (b *ContractCommitmentBuilder) WithModel(model pbc.FocusContractCommitmentModel) *ContractCommitmentBuilder {
	b.record.ContractCommitmentModel = model
	return b
}

// WithOfferCategory sets Public or Negotiated.
func (b *ContractCommitmentBuilder) WithOfferCategory(
	category pbc.FocusContractCommitmentOfferCategory,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentOfferCategory = category
	return b
}

// WithPaymentInterval sets how often the customer pays.
func (b *ContractCommitmentBuilder) WithPaymentInterval(
	interval pbc.FocusContractCommitmentPaymentInterval,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentPaymentInterval = interval
	return b
}

// WithPaymentModel sets No Upfront, Partial Upfront, or All Upfront.
func (b *ContractCommitmentBuilder) WithPaymentModel(
	model pbc.FocusContractCommitmentPaymentModel,
) *ContractCommitmentBuilder {
	b.record.ContractCommitmentPaymentModel = model
	return b
}

// WithPaymentUpfrontPercentage sets the upfront fraction. Zero is present, not null.
func (b *ContractCommitmentBuilder) WithPaymentUpfrontPercentage(percentage float64) *ContractCommitmentBuilder {
	b.record.ContractCommitmentPaymentUpfrontPercentage = proto.Float64(percentage)
	return b
}

// WithInvoiceIssuerName sets the entity that invoices the commitment.
func (b *ContractCommitmentBuilder) WithInvoiceIssuerName(name string) *ContractCommitmentBuilder {
	b.record.InvoiceIssuerName = name
	return b
}

// WithPricingCurrency sets the conditional pricing currency. Empty leaves it absent.
func (b *ContractCommitmentBuilder) WithPricingCurrency(currencyCode string) *ContractCommitmentBuilder {
	b.record.PricingCurrency = currencyCode
	return b
}

// WithPricingCurrencyCost sets the commitment cost denominated in the pricing currency.
// Zero is present, not null.
func (b *ContractCommitmentBuilder) WithPricingCurrencyCost(cost float64) *ContractCommitmentBuilder {
	b.record.PricingCurrencyContractCommitmentCost = proto.Float64(cost)
	return b
}

// WithServiceProviderName sets the service provider that offers the commitment.
func (b *ContractCommitmentBuilder) WithServiceProviderName(name string) *ContractCommitmentBuilder {
	b.record.ServiceProviderName = name
	return b
}

// WithDescription sets ContractCommitmentDescription. Empty means null.
func (b *ContractCommitmentBuilder) WithDescription(description string) *ContractCommitmentBuilder {
	b.record.ContractCommitmentDescription = description
	return b
}

// WithBaselineTerms fills the FOCUS 1.4 columns that do not allow nulls with a
// public, continuous, monthly, no-upfront discount created at at. The discount
// and upfront percentages are 0, which is present rather than null. Applicability
// is {"IsGlobalScope":true}. Call the specific setter afterwards to override one column.
func (b *ContractCommitmentBuilder) WithBaselineTerms(at time.Time) *ContractCommitmentBuilder {
	return b.
		WithApplicability(`{"IsGlobalScope":true}`).
		WithBenefitCategory(
			pbc.FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_DISCOUNT).
		WithCreated(at).
		WithDiscountPercentage(0).
		WithDurationType("1 Year").
		WithFulfillmentInterval(
			pbc.FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_MONTHLY).
		WithLastUpdated(at).
		WithLifecycleStatus(
			pbc.FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_ACTIVE).
		WithModel(pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_CONTINUOUS).
		WithOfferCategory(
			pbc.FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_PUBLIC).
		WithPaymentInterval(
			pbc.FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_MONTHLY).
		WithPaymentModel(
			pbc.FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_NO_UPFRONT).
		WithPaymentUpfrontPercentage(0).
		WithInvoiceIssuerName("Example Issuer").
		WithServiceProviderName("Example Provider")
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
