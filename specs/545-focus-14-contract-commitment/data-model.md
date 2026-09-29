# Data Model: FOCUS 1.4 Contract Commitment Columns

## ContractCommitment fields 13-30

| Field | Wire type | Null |
| --- | --- | --- |
| 13 contract_commitment_applicability | string, one JSON object | not allowed |
| 14 contract_commitment_benefit_category | enum | not allowed |
| 15 contract_commitment_created | Timestamp | not allowed |
| 16 contract_commitment_discount_percentage | optional double | allowed, except Discount |
| 17 contract_commitment_duration_type | string `[positive integer] [unit]` | not allowed |
| 18 contract_commitment_fulfillment_interval | enum | not allowed |
| 19 contract_commitment_last_updated | Timestamp | not allowed, must be >= created |
| 20 contract_commitment_lifecycle_status | enum | not allowed |
| 21 contract_commitment_model | enum | not allowed |
| 22 contract_commitment_offer_category | enum | not allowed |
| 23 contract_commitment_payment_interval | enum | not allowed |
| 24 contract_commitment_payment_model | enum | not allowed |
| 25 contract_commitment_payment_upfront_percentage | optional double | required once the model is set |
| 26 invoice_issuer_name | string | not allowed |
| 27 pricing_currency | string | allowed |
| 28 pricing_currency_contract_commitment_cost | optional double | required for spend when 27 is set |
| 29 service_provider_name | string | not allowed |
| 30 contract_commitment_description | string | allowed |

## Enumerations

Each enumeration has `UNSPECIFIED = 0` plus the FOCUS allowed values. Benefit category:
Discount, Entitlement, Availability, Other. Fulfillment interval: Hourly, Daily, Weekly,
Monthly, Quarterly, Semi-Annual, Annual, Full Period, Transactional, Custom. Lifecycle:
Proposed, Pending, Active, Exhausted, Expired, Canceled, Superseded. Model: Continuous,
Discontinuous. Offer: Public, Negotiated. Payment interval: One-Time, Monthly, Quarterly,
Semi-Annual, Annual, Custom. Payment model: No Upfront, Partial Upfront, All Upfront.

## Contract applied object

```text
{"Elements":[{"ContractId":"...","ContractCommitmentId":"...",
  "ContractCommitmentAppliedCost":12.5}]}
```

ContractId and ContractCommitmentId are required. An absent applied metric is omitted.
A zero cost or quantity is written as 0. A quantity requires a unit. Cost and quantity
may both be set.
