# Research: FOCUS 1.4 Cost and Usage Columns

**Feature**: `055-focus-14-cost-usage-columns` | **Date**: 2026-09-28

The FOCUS 1.4 facts (column types, feature levels, nullability) come from
`.specify/assessments/focus-1-4-support/research.md`, which read the FOCUS_Spec repository at tag
`v1.4`. This file records the implementation decisions.

## R1: Provider conformance rule

- **Decision**: `validateMandatoryFields` passes when `service_provider_name` or the deprecated
  `provider_name` is non-empty, and fails only when both are empty. The failure keeps
  `FieldName = "provider_name"` and constraint `"required"`; the expected value becomes
  `"non-empty service_provider_name (or deprecated provider_name)"`.
- **Rationale**: FOCUS 1.4 removes `ProviderName`, and FOCUS 1.3 already made `ServiceProviderName`
  the replacement. Accepting either field only loosens validation, so it is compatible (issue
  "smaller option"). Keeping the field name avoids breaking hosts that match on
  `ValidationError.FieldName`.
- **Alternatives considered**: FOCUS 1.2, 1.3 and 1.4 validation profiles (more API surface, and no
  caller asked for strict 1.2 checks); reporting `service_provider_name` as the field (clearer for
  1.4 authors, but changes an existing observable value).

## R2: Validating the eligibility-details JSON without allocating

- **Decision**: `json.Valid` over a zero-copy byte view of the string
  (`unsafe.Slice(unsafe.StringData(s), len(s))`), then check that the first non-whitespace byte is
  `{`. Because `json.Valid` guarantees exactly one JSON value, a leading `{` means the value is an
  object. A code comment justifies the `unsafe` use: `json.Valid` reads the bytes and neither
  retains nor mutates them. The project's golangci-lint configuration does not flag this use
  (gosec G103 reported nothing, and a `nolint` directive was rejected by `nolintlint` as unused).
- **Evidence**: A throwaway benchmark (deleted) on a 2-program value measured
  `json.Valid([]byte(s))` at 110 ns/op, 96 B/op, 1 alloc/op, and the zero-copy view at 77 ns/op,
  0 B/op, 0 allocs/op (Go 1.27.1, amd64).
- **Rationale**: FR-009 and Constitution VIII require 0 allocs/op on the validator's happy path. The
  check runs only when the field is non-empty, so records without it pay one length comparison.
- **Revised after review**: the user chose plain `json.Valid([]byte(s))` to keep `unsafe` out of the
  SDK. Records that set the field pay 1 alloc/op (about 33 ns); records without it stay 0 allocs/op.
  FR-009 and SC-004 were narrowed to match.
- **Alternatives considered**: `json.Unmarshal` into a struct to also check `CommitmentPrograms` and
  `ProgramType` (allocates on every record); a hand-written JSON scanner (more code and risk than
  stdlib); validating only in `Build()` (hosts validating received records would miss it).

## R3: Where the new rules live

- **Decision**: A new `validateFocus14Rules(r, opts) []error` in `focus_conformance.go`, called from
  `validateBusinessRulesWithOptions` right after `validateFocus13Rules`. It honors FailFast and
  Aggregate modes, and `var errs []error` stays nil on the happy path.
- **Rationale**: Mirrors the 1.3 structure, keeps version-specific rules findable, and lets
  Aggregate mode report both 1.4 violations at once.

## R4: `invoice_detail_id` requires `invoice_id`

- **Decision**: When `invoice_detail_id` is non-empty and `invoice_id` is empty, return a
  `ValidationError` for field `invoice_id` wrapping `ErrInvoiceIDMissingForInvoiceDetail`.
- **Rationale**: FOCUS 1.4 defines InvoiceDetailId as unique within an InvoiceId and null when there
  is no invoice. The rule touches only the new field, so no existing record is affected. It mirrors
  `ErrCommitmentIDMissingForStatus`.

## R5: Changed FOCUS 1.3 columns

| Column | 1.4 change | Decision |
| :--- | :--- | :--- |
| InvoiceId | Recommended → Conditional | Comment and docs only. The condition (issuer supports payable invoices) is not visible on a row |
| PricingCurrency | Allows nulls → False | Comment and docs only. proto3 strings cannot express null, and it is a Conditional column, so the empty string still means "not applicable" |
| PricingCurrencyEffectiveCost | Allows nulls → False; must equal EffectiveCost in PricingCurrency | Docs only. The equivalence needs an exchange rate |
| BilledCost | 0 for non-invoicing entities; per-invoice sums | Docs only; cross-row rules |
| EffectiveCost | Equality with BilledCost by ChargeCategory; amortization | Docs only; the existing hierarchy rule is unchanged |
| AllocatedMethodDetails | Becomes a JSON object | Comment and docs only. Enforcing it would reject 1.3 free-text values |
| ServiceProviderName, HostProviderName | Presence moved to dataset level | R1 covers ServiceProviderName; docs only otherwise |

No changed column gains a new enforced rule, so every record that was valid stays valid (SC-001).

## R6: InvoiceIssuerName

- **Decision**: Keep `invoice_issuer = 40` and `WithInvoice(invoiceID, invoiceIssuer)`; update their
  comments and godoc to say they carry the FOCUS 1.4 `InvoiceIssuerName` column. No alias setter,
  and the JSON-LD key stays `invoiceIssuer`.
- **Rationale**: Renaming breaks the generated Go and TS APIs and existing JSON-LD consumers. An
  alias setter adds surface without new capability.

## R7: Dry-run field names and the mock plugin

- **Decision**: Append `invoice_detail_id` and `commitment_program_eligibility_details` under a
  "FOCUS 1.4 Cost and Usage" group in `focusFieldNames` and in the mock plugin's
  `generateDefaultFieldMappings`. Update the wording from "FOCUS 1.2/1.3" to "FOCUS 1.2-1.4" and
  the count from ~66 to ~68.
- **Rationale**: Keeps the mock and the SDK describing the same field set (lesson from issue 537:
  test doubles must not disagree with the SDK).

## R8: JSON-LD

- **Decision**: Add `InvoiceDetailID = "focus:invoiceDetailId"` and
  `CommitmentProgramEligibilityDetails = "focus:commitmentProgramEligibilityDetails"` to
  `vocabulary.go`, and emit both as strings in `serializeFocusFields` next to the invoice and
  contract fields. The eligibility JSON is emitted as a string, like `allocatedMethodDetails`.
- **Rationale**: Consistent with how 1.3 JSON-bearing columns are serialized; the serializer skips
  empty strings, so existing output is unchanged.

## R9: Setter performance

- **Decision**: Both setters are single string assignments returning the builder, with no
  validation. Benchmarks `BenchmarkFocusRecordBuilder_WithInvoiceDetailID` and
  `BenchmarkFocusRecordBuilder_WithCommitmentProgramEligibilityDetails` follow
  `BenchmarkFocusRecordBuilder_WithContractApplied`, plus `BenchmarkValidateFocusRecord_Focus14Fields`
  for the validator and an `AllocsPerRun` test (0 without eligibility details, at most 1 with).
- **Rationale**: Issue acceptance criterion "setter benchmarks stay under 1 ns/op with 0 allocs/op".

## R10: TypeScript

- **Decision**: `make generate` adds `invoiceDetailId` and `commitmentProgramEligibilityDetails` to
  `FocusCostRecord` in `focus_pb.ts`. The hand-written `FocusRecordBuilder` gains
  `withInvoiceDetailId(id)` (throws on empty, like `withResourceId`) and
  `withCommitmentProgramEligibilityDetails(json)` (throws a `ValidationError` unless `JSON.parse`
  yields a non-null, non-array object). The msw `GetActualCost` handler returns a `focusRecord` with
  both fields, and the integration test reads them.
- **Rationale**: Constitution XIII. The TS builder validates eagerly in setters already, so it
  follows that style rather than the Go `Build()` style.

## R11: Test-first ordering

- **Decision**: Follow the 053 test-first gate. The Go tests (builder, conformance, dry-run, JSON-LD,
  `focus14_conformance_test.go`) and the TS test are written before the proto change. They fail to
  compile or fail on assertions (`InvoiceDetailId` field, setters and sentinels missing; vitest reads
  `undefined`). The proto change and `make generate` come next, then the SDK code.
