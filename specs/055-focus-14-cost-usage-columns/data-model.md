# Data Model: FOCUS 1.4 Cost and Usage Columns

**Feature**: `055-focus-14-cost-usage-columns` | **Date**: 2026-09-28

## FocusCostRecord: new fields

| # | Field | Proto type | FOCUS 1.4 column | Feature level | Nulls | Empty value means |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 67 | `invoice_detail_id` | `string` | InvoiceDetailId | Conditional (issuer supports payable invoices) | Yes | null: no invoice or provisional invoice |
| 68 | `commitment_program_eligibility_details` | `string` (JSON object) | CommitmentProgramEligibilityDetails | Conditional (provider has commitment programs) | Yes | null: not eligible, or provider has no programs |

Fields 69 to 80 are reserved by comment for future Cost and Usage columns (1.4.x or 1.5).

Example eligibility value, from FOCUS 1.4:

```json
{"CommitmentPrograms":[{"ProgramType":"Savings Plan"},{"ProgramType":"Reserved Instance"}]}
```

## FocusCostRecord: existing fields with changed documentation

| # | Field | Change |
| :--- | :--- | :--- |
| 1 | `provider_name` | Removed from FOCUS 1.4; stays on the wire, deprecated, until a v2 package. Satisfies the provider rule only as a fallback |
| 55 | `publisher` | Removed from FOCUS 1.4; stays on the wire, deprecated |
| 19 | `invoice_id` | FOCUS 1.4: Conditional (was Recommended); required context for `invoice_detail_id` |
| 40 | `invoice_issuer` | Carries the FOCUS 1.4 `InvoiceIssuerName` column; name and number unchanged |
| 51 | `pricing_currency` | FOCUS 1.4: not nullable when the column applies |
| 53 | `pricing_currency_effective_cost` | FOCUS 1.4: not nullable when the column applies; the PricingCurrency equivalent of EffectiveCost |
| 59 | `service_provider_name` | Satisfies the mandatory provider rule (R1) |
| 62 | `allocated_method_details` | FOCUS 1.4 defines it as a JSON object; the SDK does not enforce this, so 1.3 free text stays valid |

## Validation rules

Checked in this order by `ValidateFocusRecordWithOptions` (and therefore by `Build()`):

| ID | Rule | Error field | Sentinel (`errors.Is`) | Where |
| :--- | :--- | :--- | :--- | :--- |
| V1 | `service_provider_name` or `provider_name` is non-empty (replaces "provider_name required") | `provider_name` | none (plain `ValidationError`, as before) | `validateMandatoryFields` |
| V2 | If `commitment_program_eligibility_details` is non-empty, it is well-formed JSON whose top-level value is an object | `commitment_program_eligibility_details` | `ErrInvalidCommitmentProgramEligibilityDetails` | `validateFocus14Rules` |
| V3 | If `invoice_detail_id` is non-empty, `invoice_id` is non-empty | `invoice_id` | `ErrInvoiceIDMissingForInvoiceDetail` | `validateFocus14Rules` |

V2 and V3 run after the FOCUS 1.3 rules and before the contextual FinOps rules. In Aggregate mode
both can be reported. V2's actual-value text is a fixed string (`"malformed JSON"` or
`"not a JSON object"`), never the input, so the error does not echo arbitrary payloads.

Not enforced (documented producer responsibilities): one `ProgramType` matches
`commitment_discount_type` when set; no term or payment data in the eligibility JSON; custom keys
use the `x_` prefix; InvoiceDetailId uniqueness within an InvoiceId (cross-row); every changed 1.3
column rule in research R5.

## Dry-run field list

`FocusFieldNames()` grows from 66 to 68 names, adding `invoice_detail_id` and
`commitment_program_eligibility_details` in a "FOCUS 1.4 Cost and Usage" group at the end. The mock
plugin's default field mappings contain the same 68 names.

## JSON-LD

| Proto field | JSON-LD key | Vocabulary constant |
| :--- | :--- | :--- |
| `invoice_detail_id` | `invoiceDetailId` | `InvoiceDetailID = "focus:invoiceDetailId"` |
| `commitment_program_eligibility_details` | `commitmentProgramEligibilityDetails` (string) | `CommitmentProgramEligibilityDetails = "focus:commitmentProgramEligibilityDetails"` |
