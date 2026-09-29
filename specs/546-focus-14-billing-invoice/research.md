# Research: FOCUS 1.4 Billing Period and Invoice Detail

## Decision: Follow the published v1.4 column table

Issue 543 says 17 mandatory invoice columns (4 nullable) and 5 conditional. The v1.4 tag of
`FOCUS_Spec` (`specification/datasets/invoice_detail/dataset.md`) lists 18 always-present columns
and 4 conditional ones. The always-present nullable columns are description, grain, issue date,
and payment due date. The conditional columns are PaymentCurrency, PaymentCurrencyBilledCost,
PaymentCurrencyInvoiceDetailId, and PurchaseOrderNumber.

The issue's field list already contains every published column. This spec follows the published
table. Rationale: a plugin that omits ReferenceInvoiceId or treats PaymentCurrency as mandatory
would not match FOCUS 1.4. No alternative was left open.

## Decision: Payment currency pair is all or nothing

PaymentCurrency and PaymentCurrencyBilledCost share the condition "the issuer supports billing
and payment in different currencies". A single record therefore has both or neither.
PaymentCurrencyInvoiceDetailId has a different condition (aggregation levels differ) and stays
independent. When that id is set and the settlement cost is non-zero, FOCUS requires the id to
equal InvoiceDetailId.

Zero settlement cost is a real value (detail lines before an exchange rate). `optional double`
keeps it distinct from an omitted column, the same way commitment percentages work.

## Decision: Message only

Issue 543 puts delivery in a later phase. `SupplementalDatasetService` is not extended.
`CLAUDE.md` previously described issue 543 as the invoice RPC stage. That note is corrected:
the RPC, if added later, belongs on the existing service.

## Decision: Grain is a string map

FOCUS defines InvoiceDetailGrain as a JSON key-value object. Spec 026 stored Tags as
`map<string, string>`. The same representation is used here. An empty map is null. Keys must be
one of ContractId, RegionId, ResourceId, ResourceType, ServiceName, SkuId, SkuMeter, SkuPriceId,
SubAccountId, or start with `x_`.

## Decision: Custom monetary columns are `extended_columns`

FOCUS requires a custom column for any invoice monetary metric that has no FOCUS column. Field 23
is `map<string, string>` with `x_` keys and decimal values. Empty is valid, because a record
cannot know whether the invoice has extra metrics.

## Decision: Per-record rules only

Out of scope, matching the issue: invoice sums and rounding tolerance, joins to cost rows or
billing periods, uniqueness of InvoiceDetailId within an InvoiceId, and one-way status changes
(Closed back to Open, Issued back to Open).

## Decision: TypeScript build clones

`ContractCommitmentBuilder.build` returns a clone and does not repeat `ValidateContractCommitment`.
The invoice and billing-period builders do the same. The Go validator is the rule set.
