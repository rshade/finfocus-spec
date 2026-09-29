# Feature Specification: Invoice Dataset RPCs

**Feature Branch**: `547-invoice-dataset-rpcs`

**Created**: 2026-09-28

**Status**: Complete

**Input**: GitHub issue 540, stage B of `.specify/assessments/focus-1-4-support/decision.md`

## Summary

`SupplementalDatasetService` already serves Contract Commitment. This feature adds
`GetBillingPeriods` and `GetInvoiceDetails` on that same service, behind one capability, because
Billing Period and Invoice Detail join. A plugin implements both methods or neither. The window,
page size, and token rules are the rules `GetContractCommitments` already enforces. Issue 540 is
complete only when a host can page both datasets over gRPC and Connect.

## Clarifications

### Session 2026-09-28

Answers come from the staged supplemental-dataset decision, spec 544, and the Billing Period and
Invoice Detail validators shipped in spec 546.

- Q: New service, or the existing one? → A: The existing `SupplementalDatasetService`. No second
  service. `GetContractCommitments` request and response fields do not change.
- Q: One capability or two? → A: One. `PLUGIN_CAPABILITY_INVOICE_DATA` is 17. Legacy metadata is
  `supports_invoice_data`. `InvoiceDatasetProvider` has both methods, so a plugin cannot serve one
  dataset without the other.
- Q: When does `Serve` register the service? → A: When `ContractCommitmentProvider` or
  `InvoiceDatasetProvider` is implemented. The RPC whose provider is nil returns `UNIMPLEMENTED`
  with message `method <RpcName> not implemented` on gRPC and Connect, with no `rpc error:` prefix.
  The other RPC still serves. The Connect health checker lists the service once. A cost-only plugin
  registers neither provider.
- Q: Do explicit capabilities turn the RPCs off? → A: No. `WithCapabilities` replaces inferred
  capabilities and legacy metadata. Registration follows the interfaces. There is no startup
  warning: invoice data normally comes from a plugin that is also a cost source.
- Q: What matches the window? → A: `[billing_period_start, billing_period_end)` overlaps
  `[start, end)`. Invoice lines use that same pair. Both request bounds are set or both are unset.
  A set window has valid timestamps and `end` strictly after `start`. A period that ends exactly at
  the window start does not match, and neither does one that starts exactly at the window end. An
  unset window matches every non-nil record. A nil record never matches.
- Q: Can a successful response omit a period bound? → A: No. `ValidateBillingPeriod` and
  `ValidateInvoiceDetail` require both bounds, and the end strictly after the start. The match
  helper treats a nil bound as open-ended so a caller can filter before validation. A response that
  passes the response validator has no open-ended record.
- Q: What are the page rules? → A: `page_size` 0 means 50. Above 1000 is served as 1000. Negative
  is `INVALID_ARGUMENT`. There is no return-everything mode. Tokens are opaque. Clients send a
  token back with the same `start`, `end`, and `page_size`. The server does not bind a token to
  the window. `PaginateBillingPeriods` and `PaginateInvoiceDetails` use the same offset tokens as
  `PaginateContractCommitments`: standard base64 of the decimal offset. A token that does not
  decode, or that decodes to a negative offset, is `INVALID_ARGUMENT`. An offset at or past the
  match count returns an empty page, an empty `next_page_token`, and the match count.
- Q: What is unique? → A: Within one walk, `(invoice_issuer_name, billing_period_start)` including
  seconds and nanos. The same issuer and start with a different end is a duplicate. Invoice lines
  are unique on `invoice_detail_id`. The response validator checks the page. A full walk checks
  the whole result.
- Q: Is `total_count` exact? → A: Yes. It is the match count across all pages, stable for one walk,
  and 0 only when none match. The response validator checks that it is not negative and not less
  than the page length, and that the page is no longer than the effective page size. Conformance
  checks exactness by walking.
- Q: Which record rules apply on the wire? → A: Every returned billing period passes
  `ValidateBillingPeriod`. Every returned invoice line passes `ValidateInvoiceDetail`. A refund
  charge category never appears in a successful response. Payment currency and payment-currency
  billed cost are all-or-nothing. A present zero settlement cost is valid. A nil record is invalid.
- Q: What does one call mean? → A: The source's current view of the window, in an order that is
  stable across calls. FOCUS Replacement / Overwrite: the host replaces what it holds for that
  window. The response has no correction-handling or delivery-handling field.
- Q: Where do the rules live, and what are the errors? → A: In `sdk/go/testing`, because that
  package cannot import `pluginsdk`. `pluginsdk` delegates. Request failures wrap
  `ErrInvalidBillingPeriodsRequest` or `ErrInvalidInvoiceDetailsRequest`. Response failures wrap
  `ErrInvalidBillingPeriodsResponse` or `ErrInvalidInvoiceDetailsResponse`. Each error is a plain
  error, starts with the sentinel text, carries `INVALID_ARGUMENT`, and has no `rpc error:` prefix.
  A provider may return it unchanged.
- Q: What must not allocate? → A: A valid request check, a window check, and a duplicate check on a
  page of at most 64 records. A larger page may allocate one map. Simple capability checks stay at
  0 allocs/op.
- Q: Is the reference producer a `MockPlugin` method? → A: No. `MockInvoiceDatasetSource` is its
  own type, so `MockPlugin` capabilities and served services do not change. Construction rejects a
  nil, invalid, or duplicate record. `Get` copies the page it returns.
- Q: What does the TypeScript client do? → A: `getBillingPeriods` / `billingPeriods` and
  `getInvoiceDetails` / `invoiceDetails`, beside the existing commitment methods. Iterators clone
  the request, treat a missing or zero `pageSize` as 50, and send a negative `pageSize` unchanged.
  They follow `nextPageToken`, throw after 10 consecutive empty pages that still carry a token, and
  throw `Pagination safety: repeated page token` before yielding a page whose non-empty token was
  already returned. The client does not re-validate.

## User Scenarios

### User Story 1 - Page a billing period (Priority: P1)

A host asks for the billing periods that overlap a month and walks the pages.

**Independent Test**: A reference source with three periods returns the two that overlap, with
`total_count` 2, and the second page continues from an opaque token.

**Acceptance Scenarios**:

1. **Given** periods May, June, and June-from-another-issuer, **When** the host requests June with
   page size 1, **Then** page one is the June period for the first issuer, `total_count` is 2, and
   page two is the other issuer with an empty next token.
2. **Given** the same source, **When** the host sends a well-formed offset past the match count,
   **Then** the page is empty, the next token is empty, and `total_count` is unchanged.
3. **Given** two periods with the same issuer and start and different ends, **When** the response
   validator runs, **Then** it rejects the page as a duplicate.

### User Story 2 - Page an invoice line (Priority: P1)

A host asks for the invoice lines whose billing period overlaps that same month.

**Independent Test**: A refund category never appears in a successful response. A half-set
settlement currency is rejected. A present zero settlement cost is accepted.

**Acceptance Scenarios**:

1. **Given** one June line and one 2024 line, **When** the host requests June, **Then** only the
   June line is returned and `total_count` is 1.
2. **Given** a line with payment currency EUR and a present zero billed cost, **When** the response
   validator runs, **Then** it accepts the line.
3. **Given** a line with payment currency set and the billed cost unset, or a refund category,
   **When** the response validator runs, **Then** it rejects the page.

### User Story 3 - Serve only one dataset (Priority: P2)

A commitment-only plugin stays registered for `GetContractCommitments`. The two new RPCs return
`UNIMPLEMENTED`. An invoice-only plugin does the reverse. A plugin that implements both serves all
three. A cost-only plugin registers neither.

**Independent Test**: gRPC and Connect report the same status code and the same message for the
missing RPC.

**Acceptance Scenarios**:

1. **Given** a commitment-only plugin, **When** a host calls either invoice RPC, **Then** both
   transports return `UNIMPLEMENTED` and `method GetBillingPeriods not implemented` or
   `method GetInvoiceDetails not implemented`, and `GetContractCommitments` still succeeds.
2. **Given** an invoice-only plugin, **When** a host calls `GetContractCommitments`, **Then** the
   message is `method GetContractCommitments not implemented`, both invoice RPCs succeed, and the
   Connect health check reports the supplemental service `SERVING`.
3. **Given** a plugin that implements both providers, **When** a host calls all three RPCs,
   **Then** each succeeds and `GetPluginInfo` lists capabilities 16 and 17.
4. **Given** an invoice plugin whose `PluginInfo` lists only actual costs, **When** a host calls
   `GetBillingPeriods`, **Then** the RPC still serves. `GetPluginInfo` lists only the explicit
   capability.

## Edge Cases

- A nil request is `INVALID_ARGUMENT` (`request is nil`). A nil response is `response is nil`.
- Setting only `start` or only `end` is `INVALID_ARGUMENT`. So is an inverted window, an invalid
  timestamp, and a negative page size. A page size above 1000 is valid and served as 1000.
- An empty source returns an empty list, an empty next token, and `total_count` 0.
- A token past the end of the filtered list is an empty page, not an error. A token that does not
  decode is an error.
- Two periods that share an issuer and a start, including equal nanos, are duplicates even when
  their ends differ. A one-nanosecond difference in the start is a different period.
- A nil period bound matches as open-ended in the helper. The response validator rejects that
  record because the per-record rules require both bounds.
- `total_count` less than the page length is invalid. A page longer than the effective page size
  is invalid. The validator does not, by itself, prove that `total_count` equals the full match
  count.
- Order is stable: page size 1 returns the same sequence as one large page.
- The reference producer refuses to start when given a nil, invalid, or duplicate record.

## Requirements

- **FR-001**: `GetBillingPeriods` and `GetInvoiceDetails` are RPCs on `SupplementalDatasetService`.
  Each request has `start` = 1, `end` = 2, `page_size` = 3, and `page_token` = 4, with the same
  meaning as `GetContractCommitmentsRequest`. `GetBillingPeriodsResponse` has
  `billing_periods` = 1, `next_page_token` = 2, and `total_count` = 3.
  `GetInvoiceDetailsResponse` has `invoice_details` = 1 and the same token and total fields.
- **FR-002**: Both bounds are set or both are unset. A set window is valid and half-open, with
  `end` strictly after `start`. `page_size` below 0 is `INVALID_ARGUMENT`. 0 means 50. Above 1000
  is served as 1000. There is no return-everything mode.
- **FR-003**: A billing period matches when `[billing_period_start, billing_period_end)` overlaps
  `[start, end)`. An invoice line matches on that same pair. Comparison is seconds, then nanos.
  An unset window matches every non-nil record. Equal endpoints do not overlap. A nil record does
  not match. A successful response contains only records whose both bounds are set and whose end
  is strictly after the start.
- **FR-004**: Tokens are opaque offsets in the `PaginateContractCommitments` encoding. A token that
  does not decode to a non-negative offset is `INVALID_ARGUMENT`. An offset at or past the match
  count returns an empty page, an empty next token, and that count. `next_page_token` is empty on
  the last page. Clients pass a token back with the same window and page size.
- **FR-005**: Every returned billing period passes `ValidateBillingPeriod`. Every returned invoice
  line passes `ValidateInvoiceDetail`. `total_count` is the exact match count. The response
  validator enforces a non-negative total, a total not less than the page length, the page-size
  bound, per-record validity, and the window. A full walk checks that the total equals the number
  of records returned.
- **FR-006**: Within one walk, `(invoice_issuer_name, billing_period_start)` is unique, and
  `invoice_detail_id` is unique. The response validator enforces that on the page. The same issuer
  and start with a different end is still a duplicate.
- **FR-007**: `PLUGIN_CAPABILITY_INVOICE_DATA` is 17. Legacy metadata is `supports_invoice_data`.
  `Serve` infers the capability when the plugin implements `InvoiceDatasetProvider`, unless
  `PluginInfo` sets capabilities explicitly. Explicit capabilities replace inference and do not
  unregister the service. `Serve` registers the service when that provider or
  `ContractCommitmentProvider` is present, and adds the service to the Connect health checker once.
- **FR-008**: The RPC whose provider is nil returns `UNIMPLEMENTED` on gRPC and Connect. The
  message is `method GetBillingPeriods not implemented`, `method GetInvoiceDetails not
  implemented`, or `method GetContractCommitments not implemented`. The message has no
  `rpc error:` prefix. The other RPC still serves. A plugin that implements both providers serves
  all three RPCs, and discovery lists capabilities 16 and 17.
- **FR-009**: Request failures wrap `ErrInvalidBillingPeriodsRequest` or
  `ErrInvalidInvoiceDetailsRequest`. Response failures wrap `ErrInvalidBillingPeriodsResponse` or
  `ErrInvalidInvoiceDetailsResponse`. Pagination failures use the request sentinel. Each error
  carries `INVALID_ARGUMENT`, starts with the sentinel text, and has no `rpc error:` prefix. gRPC
  and Connect report the same code and the same message.
- **FR-010**: Valid request checks, window checks, and duplicate checks on a page of at most 64
  records allocate nothing. Above 64, the duplicate check may allocate one map.
- **FR-011**: `MockInvoiceDatasetSource` is a separate type. It rejects a nil, invalid, or
  duplicate record at construction, filters by the window, pages with the shared helpers, and
  returns copies. `RunInvoiceDatasetConformance` runs, for both RPCs: full walk, stable order,
  window filter, one-bound window, inverted window, negative page size, and malformed token.
- **FR-012**: The TypeScript client exposes `getBillingPeriods`, `billingPeriods`,
  `getInvoiceDetails`, and `invoiceDetails`. Iterators clone the caller's request, follow
  `nextPageToken`, and use page size 50 when `pageSize` is unset or zero. A negative `pageSize` is
  sent unchanged. They throw `Pagination safety: exceeded 10 consecutive empty pages` after that
  many empty pages that still carry a token, and `Pagination safety: repeated page token` before
  yielding a page whose non-empty token was already returned.
- **FR-013**: Plugin, testing, and project docs name the provider, capability 17, the helpers, the
  mock, the conformance entry point, and the TypeScript methods.

## Out of Scope

- Correction Handling and Delivery Handling fields. Responses stay Replacement / Overwrite.
- Joins from a cost row to these records, invoice totals, and status transitions.
- Changing `GetContractCommitments` request or response fields, or its error text.
- A second service. Both RPCs stay on `SupplementalDatasetService`.
- Changing `GetActualCost`.

## Success Criteria

- **SC-001**: A plugin that implements `InvoiceDatasetProvider` is discovered with capability 17,
  unless capabilities are set explicitly, and serves both RPCs over gRPC and Connect.
- **SC-002**: `buf breaking` against `origin/main` passes. The new RPCs and capability 17 are
  additive.
- **SC-003**: `go test ./...`, `golangci-lint run ./...`, and the TypeScript client build and tests
  pass.
- **SC-004**: A commitment-only plugin, an invoice-only plugin, and a plugin that implements both
  each behave as FR-008 describes, on both transports.
