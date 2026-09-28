# Feature Specification: FOCUS 1.4 Cost and Usage Columns

**Feature Branch**: `055-focus-14-cost-usage-columns`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "GitHub issue #541 (phase 1 of 4 of the FOCUS 1.4 umbrella, issue 540):
bring the Cost and Usage dataset up to FOCUS 1.4. Add the two new columns (InvoiceDetailId and
CommitmentProgramEligibilityDetails) and resolve the conformance rule that still requires the
ProviderName column, which FOCUS 1.4 removes."

## Clarifications

### Session 2026-09-28

No human was available for this session. Each answer was taken from the issue text and
`.specify/assessments/focus-1-4-support/research.md`, or, where those are silent, is the most
conservative backward-compatible choice.

- Q: Should conformance gain FOCUS 1.2, 1.3 and 1.4 validation profiles, or should the single
  mandatory-field check accept `service_provider_name` in place of `provider_name`? → A: Accept
  either field in the single check; no profiles. The issue calls this the smaller option, it only
  loosens validation, and no caller has asked for strict 1.2 checking.
- Q: When both provider names are empty, which field name should the validation error report? →
  A: Keep reporting `provider_name` (unchanged field name, so hosts that match on it keep working),
  and change the expected-value text to name `service_provider_name` as the preferred fix.
- Q: Should the eligibility-details JSON check run only in `Build()`, or in the shared record
  validator that `Build()` calls? → A: In the shared record validator, so a host validating a
  received record applies the same rule. The field is new, so no previously valid record is
  affected. The check must stay allocation-free.
- Q: Should a record that sets `invoice_detail_id` without `invoice_id` be rejected? → A: Yes, as a
  hard validation error. FOCUS 1.4 defines the ID as unique within an InvoiceId and null when there
  is no invoice, and the rule touches only the new field.
- Q: Should the hand-written TypeScript `FocusRecordBuilder` gain setters for the two new columns?
  → A: Yes: `withInvoiceDetailId` and `withCommitmentProgramEligibilityDetails`, the latter
  throwing a `ValidationError` for a non-object or malformed JSON value, following the builder's
  eager-validation style.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - FOCUS 1.4 records without ProviderName pass conformance (Priority: P1)

A plugin author emits cost rows that follow FOCUS 1.4. FOCUS 1.4 removes the `ProviderName` and
`PublisherName` columns, so the rows identify the provider through `ServiceProviderName` only. Today
the SDK's record validator rejects every such row because it still requires the deprecated provider
name. After this feature, the row passes.

**Why this priority**: This is a correctness bug. A record that conforms to the current FOCUS
release fails the SDK's own conformance check, so a plugin author cannot follow 1.4 and pass
validation at the same time. It is also the smallest self-contained change.

**Independent Test**: Build a record with every mandatory column set, the service provider name set
and the deprecated provider name empty. Validate it. It passes. Clear both provider names and
validate again. It fails and names the missing provider column.

**Acceptance Scenarios**:

1. **Given** a record with `service_provider_name = "AWS"` and an empty `provider_name`, **When** it
   is validated, **Then** validation succeeds.
2. **Given** a FOCUS 1.2-style record with `provider_name = "AWS"` and an empty
   `service_provider_name`, **When** it is validated, **Then** validation still succeeds (no
   regression for existing plugins).
3. **Given** a record with both provider names empty, **When** it is validated, **Then** validation
   fails with an error whose field name is `provider_name` (unchanged) and whose expected value
   names `service_provider_name`.
4. **Given** a record built with the builder that sets only the service provider, **When** `Build()`
   is called, **Then** it returns the record without error.

---

### User Story 2 - Link cost rows to invoice lines (Priority: P1)

A plugin author whose provider issues payable invoices sets the FOCUS 1.4 `InvoiceDetailId` on each
cost row, so a host can reconcile cost rows against the invoice line items they contribute to.

**Why this priority**: This is one of the two new Cost and Usage columns in FOCUS 1.4 and the only
per-row link between cost data and invoice detail data. Hosts cannot reconcile without it.

**Independent Test**: Build a record with an invoice ID and an invoice detail ID. The value
round-trips through the builder, through the wire format, and through the JSON-LD serializer
unchanged.

**Acceptance Scenarios**:

1. **Given** a builder with `invoice_id = "INV-1"` and `invoice_detail_id = "INV-1-L3"`, **When**
   `Build()` is called, **Then** the record carries both values unchanged.
2. **Given** a record with an invoice detail ID but no invoice ID, **When** it is validated,
   **Then** validation fails, because an invoice detail ID is only unique within an invoice.
3. **Given** a record without an invoice detail ID, **When** it is validated, **Then** the new rule
   does not apply and validation behaves exactly as before.

---

### User Story 3 - Report commitment program eligibility (Priority: P2)

A plugin author whose provider offers commitment programs (for example, savings plans or reserved
instances) sets the FOCUS 1.4 `CommitmentProgramEligibilityDetails` JSON object on each cost row, so
a host can tell which commitment programs a charge was eligible for, even when none was applied.

**Why this priority**: The second new Cost and Usage column. It is conditional (only providers with
commitment programs need it), so it matters to fewer plugins than the invoice link.

**Independent Test**: Build a record with
`{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}`. `Build()` succeeds and the value
round-trips unchanged. Build a record with `{"CommitmentPrograms":[` (truncated). `Build()` fails.

**Acceptance Scenarios**:

1. **Given** a well-formed JSON object in the eligibility details, **When** `Build()` is called,
   **Then** it succeeds and the record carries the exact string.
2. **Given** malformed JSON, **When** `Build()` is called, **Then** it fails with an error naming
   `commitment_program_eligibility_details`.
3. **Given** well-formed JSON that is not an object (for example, an array or a bare string),
   **When** `Build()` is called, **Then** it fails, because FOCUS defines the column as a JSON
   object.
4. **Given** an empty value, **When** `Build()` is called, **Then** it succeeds (the column is
   nullable).

---

### User Story 4 - Discover the FOCUS 1.4 columns in dry-run, docs and the TypeScript SDK (Priority: P3)

A plugin author or host developer learns that the SDK supports FOCUS 1.4 Cost and Usage columns from
the dry-run field list, the column reference, and the TypeScript bindings.

**Why this priority**: Discoverability. The fields work without it, but authors would not know they
exist, and hosts using dry-run introspection would not see them.

**Independent Test**: The dry-run field list contains both new names. The column reference has a
FOCUS 1.4 section. A TypeScript client reading an actual-cost response receives both new fields.

**Acceptance Scenarios**:

1. **Given** the SDK, **When** a plugin asks for the list of FOCUS field names, **Then** it contains
   `invoice_detail_id` and `commitment_program_eligibility_details`.
2. **Given** a TypeScript client, **When** it receives an actual-cost result whose FOCUS record sets
   both new fields, **Then** it can read both values.
3. **Given** the column reference document, **When** a reader looks up `InvoiceIssuerName`, **Then**
   it explains that the column is carried by the existing `invoice_issuer` field.

---

### Edge Cases

- Both `provider_name` and `service_provider_name` set: valid; the existing once-per-process
  deprecation warning still applies in the builder.
- `invoice_detail_id` set on a row with no `invoice_id`: invalid (User Story 2, scenario 2).
- `commitment_program_eligibility_details` is whitespace only: treated as malformed JSON and fails.
- `commitment_program_eligibility_details` is `null` (the JSON literal): not an object, fails. An
  absent value is expressed by the empty string.
- Eligibility details whose `ProgramType` does not match `commitment_discount_type`: not checked by
  the per-record validator (it would need JSON decoding on every record); documented as a producer
  responsibility.
- An existing record that uses `allocated_method_details` for free text: still valid. FOCUS 1.4
  makes that column a JSON object, but enforcing it would reject data that 1.3 accepted.
- A record that was valid before this feature: still valid, because every change either loosens a
  rule or applies only to the two new fields.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The cost record MUST gain an `invoice_detail_id` field (FOCUS 1.4 `InvoiceDetailId`,
  string, conditional and nullable, empty meaning null).
- **FR-002**: The cost record MUST gain a `commitment_program_eligibility_details` field (FOCUS 1.4
  `CommitmentProgramEligibilityDetails`), carried as a JSON object string, following
  `allocated_method_details`.
- **FR-003**: The wire change MUST be additive only: no field is renamed, renumbered or removed, and
  the breaking-change check against `main` passes. Field numbers 69 to 80 MUST be documented as
  reserved for future Cost and Usage columns.
- **FR-004**: The existing `invoice_issuer` field MUST keep its name and number, and its
  documentation MUST state that it carries the FOCUS 1.4 `InvoiceIssuerName` column.
- **FR-005**: The mandatory-field check MUST accept a record whose provider is identified by
  `service_provider_name`, by the deprecated `provider_name`, or by both. It MUST reject a record
  where both are empty. There are no per-version validation profiles. The error keeps the field
  name `provider_name` and names `service_provider_name` as the expected value.
- **FR-006**: The builder MUST offer a setter for the invoice detail ID and a setter for the
  commitment program eligibility details. Both setters MUST be allocation-free and MUST NOT validate.
- **FR-007**: The record validator, and therefore `Build()`, MUST reject eligibility details that are
  not a well-formed JSON object, and the error MUST name `commitment_program_eligibility_details`.
- **FR-008**: The record validator MUST reject a record that sets `invoice_detail_id` without
  `invoice_id`.
- **FR-009**: The record validator MUST stay allocation-free on valid records that do not set
  `commitment_program_eligibility_details`, which includes every FOCUS 1.3 record. Checking a
  non-empty eligibility value MAY cost one allocation (the byte conversion for the JSON check); the
  SDK does not use `unsafe` to avoid it.
- **FR-010**: The dry-run FOCUS field list MUST include both new field names, and its documentation
  MUST describe it as covering FOCUS 1.2 to 1.4.
- **FR-011**: The JSON-LD vocabulary and serializer MUST emit both new fields.
- **FR-012**: The Go and TypeScript bindings MUST be regenerated, and the TypeScript client MUST be
  able to read both fields from a received cost record. The TypeScript `FocusRecordBuilder` MUST
  gain `withInvoiceDetailId` and `withCommitmentProgramEligibilityDetails`; the latter throws a
  `ValidationError` for a value that is not a well-formed JSON object.
- **FR-013**: The column reference MUST add a FOCUS 1.4 section covering the two new columns, the
  removed `ProviderName` and `PublisherName`, the `InvoiceIssuerName` naming, and the changed 1.3
  columns that affect cost rows (InvoiceId becomes Conditional; PricingCurrency and
  PricingCurrencyEffectiveCost become non-nullable; BilledCost, EffectiveCost and
  AllocatedMethodDetails rule changes), stating which are enforced and which are producer
  responsibilities.
- **FR-014**: A FOCUS 1.4 conformance test file in the testing package MUST cover FR-005 to FR-008
  and FR-010; FR-009 is covered by benchmarks and an allocation test in the SDK package.
- **FR-015**: The mock plugin's default dry-run field mappings MUST include the new field names, so
  test doubles describe the same field set as the SDK.

### Key Entities

- **FocusCostRecord**: one Cost and Usage row. Gains `invoice_detail_id` (links the row to one line
  of an invoice, unique within `invoice_id`) and `commitment_program_eligibility_details` (a JSON
  object listing the commitment programs the charge was eligible for).
- **Invoice Detail line** (external, not modeled in this feature): the invoice line a cost row
  contributes to. Modeled in a later phase of the umbrella issue.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A FOCUS 1.4 record that sets only the service provider name passes validation (0
  errors), and 100% of records that were valid before this feature are still valid.
- **SC-002**: Both new fields round-trip unchanged through the builder, the wire format, the JSON-LD
  serializer and the TypeScript client.
- **SC-003**: Malformed or non-object eligibility details fail `Build()` in 100% of the tested
  cases.
- **SC-004**: The new setters cost less than 1 ns and 0 allocations per call. Validating a valid
  record costs 0 allocations without eligibility details and at most 1 with them.
- **SC-005**: The breaking-change check against `main` reports no breaking changes.

## Assumptions

- FOCUS 1.4 (tag `v1.4` of the FOCUS specification) is the reference, as summarized in
  `.specify/assessments/focus-1-4-support/research.md`.
- Fields 1 (`provider_name`) and 55 (`publisher`) stay on the wire, deprecated. Removing them needs
  a v2 package and is out of scope.
- Rules that span rows or datasets (invoice sums, rounding tolerance, ProgramType matching
  CommitmentDiscountType, PricingCurrencyEffectiveCost equivalence) are not per-record checks and
  are documented rather than enforced.
- Changed 1.3 columns are not tightened when that would reject records 1.3 accepted.
- Out of scope: the ContractApplied JSON helper (phase 2), the Contract Commitment additions, and
  the Billing Period and Invoice Detail datasets.
