# Data Model: Supplemental Dataset Service (Contract Commitments)

## GetContractCommitmentsRequest (new)

| Field | Type | Rules |
| --- | --- | --- |
| `start` | Timestamp | Q1: set together with `end` or not at all. Q2: valid timestamp. |
| `end` | Timestamp | Q1; Q2; Q3: strictly after `start`. |
| `page_size` | int32 | Q4: not negative. 0 → 50; above 1000 → 1000. |
| `page_token` | string | Opaque; Q5: a token the source did not issue is rejected by pagination. |

## GetContractCommitmentsResponse (new)

| Field | Type | Rules |
| --- | --- | --- |
| `commitments` | repeated ContractCommitment | P1: at most the effective page size. P2: no nil entries. P3: each valid (C1–C8). P4: unique `contract_commitment_id`. P5: each matches the request window (W1). |
| `next_page_token` | string | Empty on the last page. |
| `total_count` | int32 | P6: not negative and not less than `len(commitments)`. Exact across pages (checked by conformance, not by the single-response validator). |

## ContractCommitment (existing, unchanged)

Rules shared by `ContractCommitmentBuilder.Build` and `ValidateContractCommitment`, with the
builder's existing messages:

| Rule | Check | Message |
| --- | --- | --- |
| C0 | record not nil | `contract commitment is nil` |
| C1 | `contract_commitment_id` non-empty | `contract_commitment_id is required` |
| C2 | `contract_id` non-empty | `contract_id is required` |
| C3 | `billing_currency` non-empty | `billing_currency is required` |
| C4 | category SPEND or USAGE | `contract_commitment_category must be SPEND or USAGE, got …` |
| C5 | `billing_currency` is ISO 4217 | `billing_currency must be a valid ISO 4217 currency code, got …` |
| C6 | period end ≥ start (commitment and contract periods, when both set) | `…_end (…) must be >= …_start (…)` |
| C7 | cost and quantity non-negative | `contract_commitment_cost must be non-negative` (same for quantity) |
| C8 (new) | cost and quantity finite | `contract_commitment_cost must be finite` (same for quantity) |

Rule order is unchanged, and C8 is checked with C7.

## Window matching (W1)

For window `[ws, we)` (both unset → everything matches):

1. If `contract_commitment_period_start` or `contract_commitment_period_end` is set, the period is
   `[commitment start, commitment end)`; otherwise `[contract start, contract end)`.
2. An unset period start is −∞ and an unset end is +∞.
3. Match when `period start < we` and `period end > ws`.

## Capability

`PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16`, legacy key `supports_contract_commitments`,
inferred from `ContractCommitmentProvider`.

## MockContractCommitmentSource

Holds deep copies of the configured commitments in the configured order. Construction fails for a
nil, invalid (C0–C8), or duplicate-ID commitment. Each call: validate the request (Q1–Q4), filter
by W1, paginate (Q5), and return the page with the filtered total.
