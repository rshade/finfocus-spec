# FOCUS 1.2-1.4 Column Reference

This document provides a comprehensive reference for columns defined in the
FinOps FOCUS (FinOps Open Cost and Usage Specification) as implemented in
FinFocus, covering FOCUS 1.2 and the 1.3 and 1.4 additions.

References:

- FOCUS 1.2: <https://focus.finops.org/focus-specification/v1-2/>
- FOCUS 1.3: <https://focus.finops.org/focus-specification/v1-3/>
- FOCUS 1.4: <https://focus.finops.org/focus-specification/v1-4/>

## Column Summary

### FocusCostRecord Columns

| Level       | FOCUS 1.2 | FOCUS 1.3 Additions | FOCUS 1.4 Additions | Total  | Description                                 |
| ----------- | --------- | ------------------- | ------------------- | ------ | ------------------------------------------- |
| Mandatory   | 14        | 0                   | 0                   | 14     | Required for all cost records               |
| Recommended | 1         | 1                   | 0                   | 2      | Strongly suggested for completeness         |
| Conditional | 42        | 7                   | 2                   | 51     | Required when applicable conditions are met |
| **Total**   | **57**    | **8**               | **2**               | **67** | Complete FOCUS 1.4 Cost and Usage coverage  |

FOCUS 1.4 also removes `ProviderName` and `PublisherName` and makes `InvoiceId` Conditional. The
counts above keep them in their FOCUS 1.2 rows because the SDK still carries both fields. See
[FOCUS 1.4 Changes](#focus-14-changes-cost-and-usage).

### ContractCommitment Dataset (FOCUS 1.3)

| Level       | Count  | Description                          |
| ----------- | ------ | ------------------------------------ |
| Required    | 2      | Identity fields for commitments      |
| Conditional | 10     | Classification, periods, and amounts |
| **Total**   | **12** | Complete commitment dataset          |

### Deprecated Columns (FOCUS 1.3, removed in FOCUS 1.4)

| Column          | Replacement           | Notes                                                        |
| --------------- | --------------------- | ------------------------------------------------------------ |
| `ProviderName`  | `ServiceProviderName` | Supports marketplace scenarios; either one satisfies validation |
| `Publisher`     | `HostProviderName`    | Clarifies hosting vs service                                 |

## Mandatory Columns (14)

These columns MUST be present in every FOCUS-compliant cost record.

### Identity & Hierarchy

| Column                 | Type   | Description                      | Provider Mapping                                                           |
| ---------------------- | ------ | -------------------------------- | -------------------------------------------------------------------------- |
| **ProviderName**       | string | Cloud provider name (removed in FOCUS 1.4; `ServiceProviderName` satisfies the rule) | AWS, Azure, GCP, Kubernetes |
| **BillingAccountId**   | string | Billing account identifier       | AWS: Payer Account ID, Azure: Billing Account ID, GCP: Billing Account ID  |
| **BillingAccountName** | string | Display name for billing account | AWS: Account Alias, Azure: Billing Account Name, GCP: Billing Account Name |

### Billing Period

| Column                 | Type     | Description             | Provider Mapping           |
| ---------------------- | -------- | ----------------------- | -------------------------- |
| **BillingPeriodStart** | DateTime | Start of billing period | First day of billing month |
| **BillingPeriodEnd**   | DateTime | End of billing period   | Last day of billing month  |
| **BillingCurrency**    | string   | ISO 4217 currency code  | USD, EUR, GBP, etc.        |

### Charge Period

| Column                | Type     | Description       | Provider Mapping     |
| --------------------- | -------- | ----------------- | -------------------- |
| **ChargePeriodStart** | DateTime | When charge began | Line item start time |
| **ChargePeriodEnd**   | DateTime | When charge ended | Line item end time   |

### Charge Details

| Column                | Type   | Description                | Provider Mapping                                 |
| --------------------- | ------ | -------------------------- | ------------------------------------------------ |
| **ChargeCategory**    | Enum   | Nature of charge           | Usage, Purchase, Credit, Tax, Refund, Adjustment |
| **ChargeClass**       | Enum   | Classification             | Regular, Correction                              |
| **ChargeDescription** | string | Human-readable description | AWS: lineItem/LineItemDescription                |

### Service

| Column          | Type   | Description         | Provider Mapping                                           |
| --------------- | ------ | ------------------- | ---------------------------------------------------------- |
| **ServiceName** | string | Name of the service | AWS: EC2, S3; Azure: Virtual Machines; GCP: Compute Engine |

### Financial Amounts

| Column             | Type    | Description                           | Provider Mapping                             |
| ------------------ | ------- | ------------------------------------- | -------------------------------------------- |
| **BilledCost**     | Decimal | Amount billed                         | AWS: lineItem/BlendedCost, Azure: BilledCost |
| **ContractedCost** | Decimal | ContractedUnitPrice × PricingQuantity | Calculated from contracted rates             |

## Recommended Column (1)

This column is strongly recommended but not mandatory.

| Column              | Type | Description              | Provider Mapping               |
| ------------------- | ---- | ------------------------ | ------------------------------ |
| **ChargeFrequency** | Enum | How often charge applies | OneTime, Recurring, UsageBased |

## Conditional Columns (42)

These columns are required when specific conditions apply.

### Identity & Hierarchy - Conditional

| Column                 | Type   | Condition                   | Provider Mapping                                         |
| ---------------------- | ------ | --------------------------- | -------------------------------------------------------- |
| **SubAccountId**       | string | When sub-accounts exist     | AWS: Account ID, Azure: Subscription ID, GCP: Project ID |
| **SubAccountName**     | string | When sub-accounts exist     | Account/Subscription/Project name                        |
| **BillingAccountType** | string | When account types vary     | Enterprise, PayAsYouGo, etc.                             |
| **SubAccountType**     | string | When sub-account types vary | LinkedAccount, Subscription, Project                     |

### Location

| Column               | Type   | Condition                | Provider Mapping                                |
| -------------------- | ------ | ------------------------ | ----------------------------------------------- |
| **RegionId**         | string | When resource has region | AWS: us-east-1, Azure: eastus, GCP: us-central1 |
| **RegionName**       | string | When resource has region | US East (N. Virginia), East US, etc.            |
| **AvailabilityZone** | string | When AZ is applicable    | AWS: us-east-1a, Azure: 1, GCP: us-central1-a   |

### Resource Details

| Column           | Type   | Condition                     | Provider Mapping                               |
| ---------------- | ------ | ----------------------------- | ---------------------------------------------- |
| **ResourceId**   | string | When resource is identifiable | AWS: ARN, Azure: Resource ID, GCP: Resource ID |
| **ResourceName** | string | When resource has name        | User-assigned resource name                    |
| **ResourceType** | string | When resource type is known   | m5.large, Standard_D2s_v3, n1-standard-1       |

### SKU Details

| Column              | Type   | Condition                     | Provider Mapping                     |
| ------------------- | ------ | ----------------------------- | ------------------------------------ |
| **SkuId**           | string | When SKU is known             | Provider-specific SKU identifier     |
| **SkuPriceId**      | string | When SKU price is known       | Specific price list ID               |
| **SkuMeter**        | string | When meter exists             | AWS: BoxUsage, Azure: Meter Name     |
| **SkuPriceDetails** | string | When additional details exist | Operating system, license type, etc. |

### Pricing Details

| Column                                 | Type    | Condition                            | Provider Mapping                     |
| -------------------------------------- | ------- | ------------------------------------ | ------------------------------------ |
| **PricingQuantity**                    | Decimal | When pricing is quantifiable         | Number of units priced               |
| **PricingUnit**                        | string  | When pricing unit exists             | Hours, GB, Requests                  |
| **ListUnitPrice**                      | Decimal | When list price is known             | Public on-demand rate                |
| **ContractedUnitPrice**                | Decimal | When contracted price exists         | Reserved/committed rate              |
| **PricingCategory**                    | Enum    | When pricing model varies            | Standard, Committed, Dynamic, Other  |
| **PricingCurrency**                    | string  | When different from billing currency | ISO 4217 code                        |
| **PricingCurrencyContractedUnitPrice** | Decimal | When pricing currency differs        | Contracted price in pricing currency |
| **PricingCurrencyEffectiveCost**       | Decimal | When pricing currency differs        | Effective cost in pricing currency   |
| **PricingCurrencyListUnitPrice**       | Decimal | When pricing currency differs        | List price in pricing currency       |

### Financial Amounts - Conditional

| Column            | Type    | Condition                | Provider Mapping                  |
| ----------------- | ------- | ------------------------ | --------------------------------- |
| **ListCost**      | Decimal | When list price is known | On-demand cost at list price      |
| **EffectiveCost** | Decimal | When discounts apply     | Actual cost after all adjustments |

### Consumption/Usage

| Column               | Type    | Condition                | Provider Mapping                    |
| -------------------- | ------- | ------------------------ | ----------------------------------- |
| **ConsumedQuantity** | Decimal | When usage is measurable | AWS: UsageQuantity, Azure: Quantity |
| **ConsumedUnit**     | string  | When usage unit exists   | Hours, GB, Requests                 |

### Service - Conditional

| Column                 | Type   | Condition                            | Provider Mapping                                    |
| ---------------------- | ------ | ------------------------------------ | --------------------------------------------------- |
| **ServiceCategory**    | Enum   | When categorization is possible      | Compute, Storage, Network, Database, etc.           |
| **ServiceSubcategory** | string | When granular categorization exists  | Virtual Machine, Container, Serverless              |
| **Publisher**          | string | When publisher differs from provider | Amazon Web Services, Microsoft, Google, Third-party |

### Commitment Discounts

| Column                         | Type    | Condition                    | Provider Mapping                |
| ------------------------------ | ------- | ---------------------------- | ------------------------------- |
| **CommitmentDiscountCategory** | Enum    | When commitment exists       | Spend, Usage                    |
| **CommitmentDiscountId**       | string  | When commitment exists       | RI ID, Savings Plan ID          |
| **CommitmentDiscountName**     | string  | When commitment has name     | User-assigned name              |
| **CommitmentDiscountQuantity** | Decimal | When commitment has quantity | Amount purchased/consumed       |
| **CommitmentDiscountStatus**   | Enum    | When commitment applies      | Used, Unused                    |
| **CommitmentDiscountType**     | string  | When type is known           | Reserved Instance, Savings Plan |
| **CommitmentDiscountUnit**     | string  | When unit exists             | Hours, USD/Hour                 |

### Capacity Reservation

| Column                        | Type   | Condition                        | Provider Mapping    |
| ----------------------------- | ------ | -------------------------------- | ------------------- |
| **CapacityReservationId**     | string | When capacity reservation exists | CR ID from provider |
| **CapacityReservationStatus** | Enum   | When CR applies                  | Used, Unused        |

### Invoice Details

| Column            | Type   | Condition                   | Provider Mapping                      |
| ----------------- | ------ | --------------------------- | ------------------------------------- |
| **InvoiceId**     | string | When invoice exists (FOCUS 1.4: when the issuer supports payable invoices) | AWS: bill/InvoiceId, Azure: InvoiceId |
| **InvoiceIssuer** | string | When issuer is identifiable (named `InvoiceIssuerName` in FOCUS 1.4)       | Legal entity name                     |

### Metadata

| Column   | Type                 | Condition              | Provider Mapping             |
| -------- | -------------------- | ---------------------- | ---------------------------- |
| **Tags** | map\<string,string\> | When resource has tags | User-defined key-value pairs |

---

## FOCUS 1.3 Columns (8 New)

FOCUS 1.3 introduces 8 new columns for enhanced provider disambiguation,
split cost allocation, and contract commitment tracking.

### Provider Disambiguation (2 columns)

These columns replace deprecated fields to better support marketplace and
reseller scenarios.

| Column                  | Type   | Level       | Description                              | Provider Mapping                      |
| ----------------------- | ------ | ----------- | ---------------------------------------- | ------------------------------------- |
| **ServiceProviderName** | string | Conditional | Entity providing the service             | AWS, Azure, GCP, or ISV/Reseller name |
| **HostProviderName**    | string | Conditional | Entity hosting the underlying resource   | AWS, Azure, GCP (infrastructure host) |

**Deprecation Note**: `ProviderName` (field 1) and `Publisher` (field 55) are
deprecated in FOCUS 1.3. Use `ServiceProviderName` and `HostProviderName` instead.

#### Provider Disambiguation Example

```go
// Direct provider usage (traditional scenario)
builder.WithServiceProvider("AWS")
builder.WithHostProvider("AWS")

// Marketplace/Reseller scenario
builder.WithServiceProvider("Datadog")   // ISV providing the service
builder.WithHostProvider("AWS")          // AWS hosting the infrastructure
```

### Split Cost Allocation (5 columns)

These columns enable tracking of how shared resource costs are distributed
across workloads using organizational allocation methodologies.

| Column                     | Type                 | Level       | Description                              | Provider Mapping           |
| -------------------------- | -------------------- | ----------- | ---------------------------------------- | -------------------------- |
| **AllocatedMethodId**      | string               | Conditional | Identifier for allocation methodology    | Organization-defined ID    |
| **AllocatedMethodDetails** | string               | Recommended | Human-readable allocation description    | Free-text description      |
| **AllocatedResourceId**    | string               | Conditional | Resource receiving the allocated cost    | Target resource identifier |
| **AllocatedResourceName**  | string               | Conditional | Display name of target resource          | Target resource name       |
| **AllocatedTags**          | map\<string,string\> | Conditional | Tags associated with allocated resource  | Allocation metadata        |

**Validation Rule**: If `AllocatedMethodId` is populated, `AllocatedResourceId` MUST also be populated.

#### Allocation Example

```go
builder.WithAllocation("ALLOC-001", "CPU-weighted by namespace utilization")
builder.WithAllocatedResource("ns/payments-service", "Payments Service")
builder.WithAllocatedTags(map[string]string{
    "cost_center": "CC-1001",
    "team":        "payments",
})
```

### Contract Commitment Link (1 column)

This column links cost records to the ContractCommitment supplemental dataset.

| Column              | Type   | Level       | Description                            | Provider Mapping             |
| ------------------- | ------ | ----------- | -------------------------------------- | ---------------------------- |
| **ContractApplied** | string | Conditional | Reference to ContractCommitmentId      | Opaque commitment identifier |

#### Contract Link Example

```go
builder.WithContractApplied("ri-123456789")  // Links to ContractCommitment dataset
```

---

## FOCUS 1.4 Changes (Cost and Usage)

FOCUS 1.4 adds 2 columns to the Cost and Usage dataset, removes 2, and changes the rules of
several existing columns. Every change here is additive on the wire: no field was renamed,
renumbered, or removed. Field numbers 69 to 80 are reserved for future Cost and Usage columns.

### New Columns (2)

| Column                                  | Proto field (number)                          | Type             | Level       | Nulls | Builder method                                |
| --------------------------------------- | --------------------------------------------- | ---------------- | ----------- | ----- | --------------------------------------------- |
| **InvoiceDetailId**                     | `invoice_detail_id` (67)                      | string           | Conditional | Yes   | `WithInvoiceDetailID(id)`                     |
| **CommitmentProgramEligibilityDetails** | `commitment_program_eligibility_details` (68) | JSON object text | Conditional | Yes   | `WithCommitmentProgramEligibilityDetails(js)` |

- **InvoiceDetailId** links a cost row to the invoice line item it contributes to. It is required
  when the invoice issuer supports payable invoices, and null (empty) when there is no invoice or
  only a provisional one. It is unique within an InvoiceId, so the SDK requires `invoice_id`
  whenever it is set.
- **CommitmentProgramEligibilityDetails** lists the commitment programs the charge is eligible
  for, even when none was applied. It is required when the provider has commitment programs. It
  is stored as a JSON string, like `allocated_method_details`, and the SDK requires a well-formed
  JSON object when it is set.

#### FOCUS 1.4 Example

```go
builder.WithInvoice("INV-2026-09", "Amazon Web Services, Inc.").
    WithInvoiceDetailID("INV-2026-09-L3").
    WithCommitmentProgramEligibilityDetails(
        `{"CommitmentPrograms":[{"ProgramType":"Savings Plan"},{"ProgramType":"Reserved Instance"}]}`,
    )
```

`ProgramType` values are provider-defined. When `commitment_discount_type` is set, one
`ProgramType` should match it. Custom keys use the `x_` prefix.

### Removed Columns (2)

| Column          | Proto field (number)  | Status in this SDK                                                                   |
| --------------- | --------------------- | ------------------------------------------------------------------------------------ |
| `ProviderName`  | `provider_name` (1)   | Deprecated, kept on the wire. Validation accepts `service_provider_name` in its place |
| `PublisherName` | `publisher` (55)      | Deprecated, kept on the wire. Use `host_provider_name`                                 |

A FOCUS 1.4 record sets `service_provider_name` and leaves `provider_name` empty. The mandatory
provider check passes when either field is set. When both are empty, the validation error keeps
the field name `provider_name` and names `service_provider_name` as the expected value. Removing
fields 1 and 55 from the wire needs a v2 package.

### Renamed Column

`InvoiceIssuer` is named `InvoiceIssuerName` in FOCUS 1.4. The proto field keeps its name
(`invoice_issuer`, field 40) and `WithInvoice(invoiceID, invoiceIssuer)` is unchanged, so no
generated Go or TypeScript API breaks.

### Changed Columns

| Column                           | FOCUS 1.4 change                                                      | SDK behavior                |
| -------------------------------- | --------------------------------------------------------------------- | --------------------------- |
| **InvoiceId**                    | Recommended → Conditional (required when payable invoices exist)      | Producer responsibility     |
| **PricingCurrency**              | No longer nullable when the column applies                           | Producer responsibility     |
| **PricingCurrencyEffectiveCost** | Not nullable; must be the PricingCurrency equivalent of EffectiveCost | Producer responsibility     |
| **BilledCost**                   | 0 for non-invoicing entities; per-invoice sums within a tolerance     | Producer responsibility     |
| **EffectiveCost**                | Explicit equality with BilledCost by ChargeCategory; amortization     | Existing hierarchy rule only |
| **AllocatedMethodDetails**       | Becomes a JSON object (`AllocatedMethodDetailsObject`)                 | Not enforced (1.3 free text stays valid) |
| **ServiceProviderName**          | Presence moved to dataset-level rules                                 | Satisfies the provider rule |
| **HostProviderName**             | Must match ServiceProviderName when no separate host exists           | Producer responsibility     |

### Validation Summary

| Rule                                                                           | Enforced | Error (`errors.Is`)                             |
| ------------------------------------------------------------------------------ | -------- | ----------------------------------------------- |
| `service_provider_name` or `provider_name` is set                              | Yes      | `ValidationError` on `provider_name`            |
| `commitment_program_eligibility_details`, when set, is a well-formed JSON object | Yes      | `ErrInvalidCommitmentProgramEligibilityDetails` |
| `invoice_detail_id`, when set, has an `invoice_id`                              | Yes      | `ErrInvoiceIDMissingForInvoiceDetail`           |
| One `ProgramType` matches `commitment_discount_type`                           | No       | Producer responsibility                         |
| InvoiceDetailId is unique within an InvoiceId                                  | No       | Cross-row rule                                  |

The per-record checks allocate nothing on valid records.

---

## ContractCommitment Supplemental Dataset (FOCUS 1.3)

FOCUS 1.3 introduces the ContractCommitment dataset for tracking commitment-based
discount programs separately from cost records.

### Identity Fields (2 Required)

| Column                     | Type   | Level    | Description                           |
| -------------------------- | ------ | -------- | ------------------------------------- |
| **ContractCommitmentId**   | string | Required | Unique identifier for this commitment |
| **ContractId**             | string | Required | Parent contract identifier            |

### Classification Fields

| Column                         | Type   | Level       | Description                     |
| ------------------------------ | ------ | ----------- | ------------------------------- |
| **ContractCommitmentCategory** | Enum   | Conditional | SPEND or USAGE                  |
| **ContractCommitmentType**     | string | Conditional | Provider-specific type          |

### Commitment Period Fields

| Column                            | Type     | Level       | Description                   |
| --------------------------------- | -------- | ----------- | ----------------------------- |
| **ContractCommitmentPeriodStart** | DateTime | Conditional | Start of commitment period    |
| **ContractCommitmentPeriodEnd**   | DateTime | Conditional | End of commitment period      |

### Contract Period Fields

| Column                  | Type     | Level       | Description              |
| ----------------------- | -------- | ----------- | ------------------------ |
| **ContractPeriodStart** | DateTime | Conditional | Start of parent contract |
| **ContractPeriodEnd**   | DateTime | Conditional | End of parent contract   |

### Financial Fields

| Column                         | Type    | Level       | Description                     |
| ------------------------------ | ------- | ----------- | ------------------------------- |
| **ContractCommitmentCost**     | Decimal | Conditional | Monetary amount (SPEND)         |
| **ContractCommitmentQuantity** | Decimal | Conditional | Unit quantity (USAGE)           |
| **ContractCommitmentUnit**     | string  | Conditional | Unit of measure (Hours, GB)     |
| **BillingCurrency**            | string  | Required    | ISO 4217 currency code          |

### ContractCommitment Example

```go
import "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

commitment := pluginsdk.NewContractCommitmentBuilder().
    WithIdentity("ri-123456789", "contract-2024-001").
    WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
    WithType("Standard Reserved Instance").
    WithCommitmentPeriod(
        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
    ).
    WithFinancials(0, 8760, "Hours", "USD").  // USAGE: quantity=8760 hours, cost=0
    Build()
```

---

## Provider Mapping Examples

### AWS

```text
ServiceProviderName:    "AWS"              // FOCUS 1.3
HostProviderName:       "AWS"              // FOCUS 1.3
BillingAccountId:       "123456789012"
ServiceName:            "Amazon EC2"
ResourceId:             "arn:aws:ec2:us-east-1:123456789012:instance/i-0abc123"
RegionId:               "us-east-1"
CommitmentDiscountType: "Standard Reserved Instance"
ContractApplied:        "ri-123456789"     // FOCUS 1.3
```

### Azure

```text
ServiceProviderName:    "Azure"            // FOCUS 1.3
HostProviderName:       "Azure"            // FOCUS 1.3
BillingAccountId:       "ea12345678"
SubAccountId:           "00000000-0000-0000-0000-000000000000"
ServiceName:            "Virtual Machines"
ResourceId:             "/subscriptions/.../resourceGroups/.../providers/Microsoft.Compute/virtualMachines/vm-name"
RegionId:               "eastus"
CommitmentDiscountType: "Reservation"
ContractApplied:        "reservation-2024-001"  // FOCUS 1.3
```

### GCP

```text
ServiceProviderName:    "GCP"              // FOCUS 1.3
HostProviderName:       "GCP"              // FOCUS 1.3
BillingAccountId:       "012345-ABCDEF-678901"
SubAccountId:           "my-project-id"
ServiceName:            "Compute Engine"
ResourceId:             "projects/my-project/zones/us-central1-a/instances/instance-1"
RegionId:               "us-central1"
CommitmentDiscountType: "Committed Use Discount"
ContractApplied:        "cud-n1-standard-1"  // FOCUS 1.3
```

### Kubernetes (via Kubecost)

```text
ServiceProviderName:    "Kubernetes"       // FOCUS 1.3
HostProviderName:       "AWS"              // FOCUS 1.3 - underlying infrastructure
BillingAccountId:       "cluster-name"
SubAccountId:           "namespace-name"
ServiceName:            "Kubernetes Workload"
ResourceId:             "namespace/pod-name"
ServiceCategory:        FOCUS_SERVICE_CATEGORY_COMPUTE
```

### Marketplace/Reseller Scenario (FOCUS 1.3)

```text
ServiceProviderName:    "Datadog"          // ISV providing the service
HostProviderName:       "AWS"              // Cloud hosting infrastructure
BillingAccountId:       "123456789012"
ServiceName:            "Datadog APM"
```

## Common Use Cases

### Cost Allocation by Team

Use Tags to allocate costs:

```go
builder.WithTags(map[string]string{
    "team":        "platform",
    "cost_center": "CC-1001",
    "environment": "production",
})
```

### Reserved Instance Analysis

Track commitment utilization:

```go
builder.WithCommitmentDiscount(
    pbc.FocusCommitmentDiscountCategory_FOCUS_COMMITMENT_DISCOUNT_CATEGORY_USAGE,
    "ri-123456789",
    "EC2 m5.large RI",
)
builder.WithCommitmentDiscountDetails(
    730.0, // hours
    pbc.FocusCommitmentDiscountStatus_FOCUS_COMMITMENT_DISCOUNT_STATUS_USED,
    "Standard Reserved Instance",
    "Hours",
)
```

### Multi-Currency Billing

Handle pricing in different currencies:

```go
builder.WithBillingPeriod(start, end, "EUR")
builder.WithPricingCurrency("USD")
builder.WithPricingCurrencyPrices(
    0.089,  // contracted unit price in USD
    65.0,   // effective cost in USD
    0.10,   // list unit price in USD
)
```

### Capacity Reservation Tracking

Monitor capacity reservation utilization:

```go
builder.WithCapacityReservation(
    "cr-0abc123456789def0",
    pbc.FocusCapacityReservationStatus_FOCUS_CAPACITY_RESERVATION_STATUS_USED,
)
```

### Split Cost Allocation (FOCUS 1.3)

Allocate shared resource costs to specific workloads:

```go
builder.WithAllocation("ALLOC-001", "CPU-weighted by namespace utilization")
builder.WithAllocatedResource("ns/payments-service", "Payments Service")
builder.WithAllocatedTags(map[string]string{
    "team":        "payments",
    "cost_center": "CC-1001",
})
```

### Contract Commitment Tracking (FOCUS 1.3)

Link cost records to commitment discount programs:

```go
// In FocusCostRecord: link to commitment
costBuilder.WithContractApplied("ri-123456789")

// Separate ContractCommitment record for commitment details
commitmentBuilder := pluginsdk.NewContractCommitmentBuilder().
    WithIdentity("ri-123456789", "contract-2024-001").
    WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
    WithType("Standard Reserved Instance").
    WithFinancials(0, 8760, "Hours", "USD")
```

## Query Patterns

### Total Cost by Service

```sql
SELECT ServiceName, SUM(BilledCost) as TotalCost
FROM focus_records
GROUP BY ServiceName
ORDER BY TotalCost DESC
```

### Reserved Instance Utilization

```sql
SELECT
    CommitmentDiscountId,
    CommitmentDiscountName,
    SUM(CASE WHEN CommitmentDiscountStatus = 'USED' THEN BilledCost ELSE 0 END) as UsedCost,
    SUM(CASE WHEN CommitmentDiscountStatus = 'UNUSED' THEN BilledCost ELSE 0 END) as WastedCost
FROM focus_records
WHERE CommitmentDiscountId IS NOT NULL
GROUP BY CommitmentDiscountId, CommitmentDiscountName
```

### Cost by Tag

```sql
SELECT
    Tags['team'] as Team,
    SUM(BilledCost) as TotalCost
FROM focus_records
GROUP BY Tags['team']
```

### Regional Cost Distribution

```sql
SELECT
    ServiceProviderName,    -- FOCUS 1.3
    RegionId,
    SUM(BilledCost) as TotalCost
FROM focus_records
GROUP BY ServiceProviderName, RegionId
ORDER BY ServiceProviderName, TotalCost DESC
```

### Allocated Cost by Team (FOCUS 1.3)

```sql
SELECT
    AllocatedTags['team'] as Team,
    AllocatedResourceName,
    SUM(BilledCost) as AllocatedCost
FROM focus_records
WHERE AllocatedResourceId IS NOT NULL
GROUP BY AllocatedTags['team'], AllocatedResourceName
ORDER BY AllocatedCost DESC
```

### Contract Commitment Utilization (FOCUS 1.3)

```sql
SELECT
    cc.ContractCommitmentId,
    cc.ContractCommitmentType,
    cc.ContractCommitmentCost as CommittedAmount,
    SUM(f.BilledCost) as ActualSpend
FROM contract_commitments cc
LEFT JOIN focus_records f ON f.ContractApplied = cc.ContractCommitmentId
GROUP BY cc.ContractCommitmentId, cc.ContractCommitmentType, cc.ContractCommitmentCost
```
