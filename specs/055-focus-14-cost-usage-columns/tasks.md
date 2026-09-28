# Tasks: FOCUS 1.4 Cost and Usage Columns

**Input**: Design documents from `specs/055-focus-14-cost-usage-columns/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: REQUIRED. Constitution V (Test-First Protocol) is non-negotiable and covers the proto
fields too. The test tasks of Phases 3 to 6 are written **before** T003. At that point they fail
to compile, because `InvoiceDetailId`, `CommitmentProgramEligibilityDetails`, the two setters and the
two sentinels do not exist yet. That compile failure is the "fails against the current proto" state.
T003 to T005 then add the fields, and the behavior tests still fail until each story's
implementation lands.

**Organization**: Tasks are grouped by user story (spec.md). US1 and US2 are both P1.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: The user story the task belongs to (US1 to US4)

## Path Conventions

Single-repository SDK. Paths are relative to the repo root: `proto/`, `sdk/go/`, `sdk/typescript/`,
`docs/`.

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Record the baseline: run `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ ./sdk/go/jsonld/`
  and `go test ./sdk/go/pluginsdk/ -run '^$' -bench 'ValidateFocusRecord_ValidRecord|WithContractApplied' -benchmem`.
  Keep the output in the session only; do not commit it.
- [X] T002 Create `sdk/go/testing/focus14_conformance_test.go` with the Apache 2.0 header copied from
  `sdk/go/testing/allocation_test.go`, `package testing_test`, and a
  `buildValidFocus14Record() *pluginsdk.FocusRecordBuilder` helper that sets every mandatory column
  with `WithServiceProvider("AWS")` and **without** `WithIdentity`'s provider name (use
  `WithIdentity("", "123456789012", "Production")`).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the wire fields. Every story's implementation depends on the generated Go and TS
fields.

**Test-first gate (Constitution V)**: Before T003, write T006, T007, T009 to T011, T015 to T018 and
T021 to T023. Then run `go vet ./sdk/go/pluginsdk/ ./sdk/go/testing/ ./sdk/go/jsonld/` and
`npx vitest run` in `sdk/typescript/packages/client`. Go MUST fail to compile on the undefined
fields, setters and sentinels; vitest MUST fail on the missing `invoiceDetailId` and builder
methods. Record the failure output for the PR body.

- [X] T003 Edit `proto/finfocus/v1/focus.proto` exactly as `contracts/proto-changes.md` says: add
  `string invoice_detail_id = 67;` and `string commitment_program_eligibility_details = 68;` after
  `contract_applied = 66` under a "FOCUS 1.4 Additions" banner that reserves 69 to 80 by comment,
  and apply every comment-only change in that contract's table (fields 1, 19, 40, 51, 53, 55, 62,
  the message comment and the 1.3 numbering comment).
- [X] T004 Run `make generate`. Confirm `sdk/go/proto/finfocus/v1/focus.pb.go` gains
  `InvoiceDetailId`/`GetInvoiceDetailId()` and
  `CommitmentProgramEligibilityDetails`/`GetCommitmentProgramEligibilityDetails()`, and
  `sdk/typescript/packages/client/src/generated/finfocus/v1/focus_pb.ts` gains `invoiceDetailId` and
  `commitmentProgramEligibilityDetails`. Restore unrelated regenerated files with `git checkout --`.
- [X] T005 Run `bin/buf lint` and `bin/buf breaking --against '.git#branch=main'`; both must pass.

**Checkpoint**: `go build ./...` passes. Pre-written tests compile once T012, T013, T019 and T020 add
the missing sentinels and setters.

---

## Phase 3: User Story 1 - FOCUS 1.4 records without ProviderName pass (Priority: P1) 🎯 MVP

**Goal**: Rule V1 from data-model.md: "`service_provider_name` or `provider_name` is non-empty".

**Independent Test**: `go test ./sdk/go/testing/ -run 'TestFocus14_Provider' -v` passes.

### Tests for User Story 1 (write first) ⚠️

- [X] T006 [P] [US1] In `sdk/go/pluginsdk/focus_conformance_test.go`, extend
  `TestValidateFocusRecord_MandatoryFields`: the "missing provider_name" case now clears **both**
  `ProviderName` and `ServiceProviderName` and still expects field `provider_name`; add a case
  asserting a record with `ProviderName = ""` and `ServiceProviderName = "AWS"` is valid. Update
  `TestValidateFocusRecord_ErrorsAs_AdHocErrors`'s provider case the same way. Assert the error's
  `ExpectedValue` contains `service_provider_name`.
- [X] T007 [P] [US1] In `sdk/go/testing/focus14_conformance_test.go`, add
  `TestFocus14_ProviderRule` with subtests: service provider only → `Build()` succeeds; deprecated
  provider only (FOCUS 1.2 style) → succeeds; both set → succeeds; neither → fails with
  `*pluginsdk.ValidationError` whose `FieldName == "provider_name"`.

### Implementation for User Story 1

- [X] T008 [US1] In `sdk/go/pluginsdk/focus_conformance.go`, change `validateMandatoryFields` so the
  provider check fails only when `r.GetProviderName() == "" && r.GetServiceProviderName() == ""`,
  returning `NewValidationError("provider_name", "required", "", "non-empty service_provider_name (or deprecated provider_name)")`.
  Update the function godoc and the `ValidateFocusRecord` godoc (FOCUS 1.2 to 1.4). T006 and T007
  pass.

**Checkpoint**: US1 is complete and shippable alone.

---

## Phase 4: User Story 2 - Link cost rows to invoice lines (Priority: P1)

**Goal**: Field 67 end to end, plus rule V3: "If `invoice_detail_id` is non-empty, `invoice_id` is
non-empty".

**Independent Test**: `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ ./sdk/go/jsonld/ -run 'InvoiceDetail' -v`
passes.

### Tests for User Story 2 (write first) ⚠️

- [X] T009 [P] [US2] In `sdk/go/pluginsdk/focus_builder_test.go`, add
  `TestFocusRecordBuilder_WithInvoiceDetailID` (round trip with `WithInvoice("INV-1", "AWS")`;
  missing invoice ID → `errors.Is(err, pluginsdk.ErrInvoiceIDMissingForInvoiceDetail)` and
  `FieldName == "invoice_id"`; empty detail ID → no new error) and
  `BenchmarkFocusRecordBuilder_WithInvoiceDetailID` with `b.ReportAllocs()`, modeled on
  `BenchmarkFocusRecordBuilder_WithContractApplied`.
- [X] T010 [P] [US2] In `sdk/go/testing/focus14_conformance_test.go`, add
  `TestFocus14_InvoiceDetailID`: round trip through `Build()`, through `proto.Marshal`/`Unmarshal`
  with `proto.Equal`, and the missing-invoice-ID failure.
- [X] T011 [P] [US2] In `sdk/go/jsonld/serializer_test.go`, add `TestSerialize_Focus14Fields`
  asserting the output has `invoiceDetailId` and `commitmentProgramEligibilityDetails` (string
  values equal to the input) and omits both keys when empty.

### Implementation for User Story 2

- [X] T012 [US2] Add `ErrInvoiceIDMissingForInvoiceDetail` ("invoice_id required when
  invoice_detail_id is set") to the sentinel `var` block in `sdk/go/pluginsdk/focus_conformance.go`,
  and add `validateFocus14Rules(r *pbc.FocusCostRecord, opts ValidationOptions) []error`, called
  from `validateBusinessRulesWithOptions` right after `validateFocus13Rules`, honoring FailFast and
  Aggregate modes. Implement V3 there with `NewValidationErrorWithCause("invoice_id", ...)`.
- [X] T013 [US2] Add `WithInvoiceDetailID(invoiceDetailID string) *FocusRecordBuilder` to
  `sdk/go/pluginsdk/focus_builder.go` under a "FOCUS 1.4 Cost and Usage Builder Methods" banner,
  with the godoc from `contracts/sdk-api.md`. Update `WithInvoice`'s godoc to say `invoiceIssuer`
  carries the FOCUS 1.4 `InvoiceIssuerName` column.
- [X] T014 [US2] Add `InvoiceDetailID` and `CommitmentProgramEligibilityDetails` constants to
  `sdk/go/jsonld/vocabulary.go` (next to the invoice fields) and emit both with `fw.addString` in
  `serializeCostRecordFields` in `sdk/go/jsonld/serializer.go`, after `invoiceIssuer`. T009 to T011
  pass (T011's eligibility half passes once T004 lands).

**Checkpoint**: US1 and US2 are complete.

---

## Phase 5: User Story 3 - Report commitment program eligibility (Priority: P2)

**Goal**: Field 68 plus rule V2: "If `commitment_program_eligibility_details` is non-empty, it is
well-formed JSON whose top-level value is an object", allocation-free.

**Independent Test**: `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -run 'CommitmentProgramEligibility|Focus14' -v`
passes, and the new benchmarks show at most 1 alloc/op (only for the eligibility JSON check).

### Tests for User Story 3 (write first) ⚠️

- [X] T015 [P] [US3] In `sdk/go/pluginsdk/focus_builder_test.go`, add
  `TestFocusRecordBuilder_WithCommitmentProgramEligibilityDetails`, a table test. Valid:
  `{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}`, the same with leading/trailing
  whitespace, `{}`, and `""` (unset). Invalid, each `errors.Is(err, pluginsdk.ErrInvalidCommitmentProgramEligibilityDetails)`
  with `FieldName == "commitment_program_eligibility_details"`: truncated `{"CommitmentPrograms":[`,
  `[]`, `"text"`, `null`, `42`, whitespace only, `{"a":1}{"b":2}`, and a non-JSON word. Also add
  `BenchmarkFocusRecordBuilder_WithCommitmentProgramEligibilityDetails` with `b.ReportAllocs()`.
- [X] T016 [P] [US3] In `sdk/go/pluginsdk/focus_benchmark_test.go`, add
  `BenchmarkValidateFocusRecord_Focus14Fields` (the `createValidBenchmarkRecord()` record plus
  `ServiceProviderName`, `InvoiceId`, `InvoiceDetailId` and a two-program eligibility JSON) with
  `b.ReportAllocs()`, and `TestValidateFocusRecord_Focus14Allocs` asserting
  `testing.AllocsPerRun(100, ...)` is 0 without eligibility details and at most 1 with them.
- [X] T017 [P] [US3] In `sdk/go/testing/focus14_conformance_test.go`, add
  `TestFocus14_CommitmentProgramEligibilityDetails` (valid round trip, malformed and non-object
  failures) and `TestFocus14_AggregateMode` (a record violating V2 and V3 reports both errors with
  `ValidationModeAggregate`).
- [X] T018 [P] [US3] Add `ExampleFocusRecordBuilder_WithCommitmentProgramEligibilityDetails` to
  `sdk/go/pluginsdk/example_test.go` as described in `contracts/sdk-api.md`, with an `// Output:`
  block.

### Implementation for User Story 3

- [X] T019 [US3] Add `ErrInvalidCommitmentProgramEligibilityDetails` to the sentinel block and implement
  V2 in `validateFocus14Rules` in `sdk/go/pluginsdk/focus_conformance.go` via a helper
  `jsonObjectFailure(s string) string`, which returns the failure reason or "": `json.Valid([]byte(s))`
  (a zero-copy `unsafe` view was replaced after review), then the first non-whitespace byte must be `{`.
  The error's actual value is the fixed string `"malformed JSON"` or `"not a JSON object"`, never
  the input (research R2).
- [X] T020 [US3] Add `WithCommitmentProgramEligibilityDetails(detailsJSON string) *FocusRecordBuilder`
  to `sdk/go/pluginsdk/focus_builder.go` with the godoc from `contracts/sdk-api.md`. T015 to T018
  pass.

**Checkpoint**: All Go behavior is complete.

---

## Phase 6: User Story 4 - Discoverability: dry-run, docs, TypeScript (Priority: P3)

**Goal**: FR-010 to FR-013 and FR-015.

**Independent Test**: dry-run name tests, the mock/SDK parity test, and vitest pass; docs describe
the 1.4 columns.

### Tests for User Story 4 (write first) ⚠️

- [X] T021 [P] [US4] In `sdk/go/pluginsdk/dry_run_test.go`, add a `focus14Fields` check to
  `TestFocusFieldNames` for `invoice_detail_id` and `commitment_program_eligibility_details`.
- [X] T022 [P] [US4] In `sdk/go/testing/focus14_conformance_test.go`, add
  `TestFocus14_FieldNamesMatchProto` (every field of `(&pbc.FocusCostRecord{}).ProtoReflect().Descriptor()`
  is in `pluginsdk.FocusFieldNames()` and vice versa) and `TestFocus14_MockDryRunFieldParity` (a
  default `NewMockPlugin()` behind a `TestHarness` returns DryRun field mappings whose names equal
  `FocusFieldNames()` as a set). No `t.Parallel()` on harness subtests.
- [X] T023 [P] [US4] In `sdk/typescript/packages/client/test/mocks/handlers.ts`, add a `focusRecord`
  with `serviceProviderName: "AWS"`, `invoiceId: "INV-2026-09"`, `invoiceDetailId: "INV-2026-09-L3"`
  and `commitmentProgramEligibilityDetails: '{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}'`
  to the `GetActualCost` result. In `sdk/typescript/packages/client/test/integration.test.ts`, assert
  the client reads both values, and add builder tests for `withInvoiceDetailId` and
  `withCommitmentProgramEligibilityDetails` (valid object; `[]`, `null`, `"x"` and truncated JSON
  throw `ValidationError`).

### Implementation for User Story 4

- [X] T024 [US4] In `sdk/go/pluginsdk/dry_run.go`, append the two names under a
  `// FOCUS 1.4 Cost and Usage` group with field-number comments, and change "FOCUS 1.2/1.3" to
  "FOCUS 1.2-1.4" and "~66" to "~68" in the variable and `FocusFieldNames` godoc. Mirror the names in
  `generateDefaultFieldMappings` in `sdk/go/testing/mock_plugin.go`. T021 and T022 pass.
- [X] T025 [US4] In `sdk/typescript/packages/client/src/builders/focus-record.ts`, add
  `withInvoiceDetailId(id)` and `withCommitmentProgramEligibilityDetails(detailsJson)` per
  `contracts/sdk-api.md`. Run `npx vitest run` and `npx tsc --noEmit`; T023 passes.
- [X] T026 [P] [US4] In `docs/focus-columns.md`, retitle to "FOCUS 1.2-1.4 Column Reference", add
  FOCUS 1.4 to the references and summary tables, and add a `## FOCUS 1.4 Changes (Cost and Usage)`
  section: the two new columns with feature level, nullability and examples; removed
  `ProviderName`/`PublisherName` and how validation now treats them; `InvoiceIssuerName` carried by
  `invoice_issuer`; and the changed-columns table from research R5, marking each rule "enforced" or
  "producer responsibility".
- [X] T027 [P] [US4] In `sdk/go/pluginsdk/README.md`, add a `## FOCUS 1.4 Cost and Usage Columns`
  section after "FOCUS 1.3 Extensions" (and a TOC entry): the two setters by exact name, the three
  validation rules and two sentinels, the relaxed provider rule, and an example. Update the FOCUS 1.3
  "Deprecated Fields" note that validation accepts `service_provider_name`.
- [X] T028 [P] [US4] Update the `FocusRecordBuilder` type godoc in `sdk/go/pluginsdk/focus_builder.go`
  (version list and migration note: FOCUS 1.4 removes provider_name/publisher; set
  `WithServiceProvider`).

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T029 [P] Run `goimports -w` on changed Go files, then `golangci-lint run ./...`; fix every
  finding (baseline 0).
- [X] T030 [P] Run `npx markdownlint-cli2` on changed markdown (quickstart step 6) and fix findings.
- [X] T031 Run `make test` (rerun once if the only failure is the known
  `TestUsageSourceNotRegistered/connect` flake).
- [X] T032 Run every step of `specs/055-focus-14-cost-usage-columns/quickstart.md` and confirm each
  expected outcome; compare the step 4 numbers with T001.
- [X] T033 Update `CLAUDE.md` by hand (do not run update-agent-context): add 055 entries to "Active
  Technologies" and "Recent Changes" matching the existing format, update the FOCUS 1.3 section's
  `FocusFieldNames` "~66 FOCUS 1.2/1.3" mention, list `focus14_conformance_test.go` next to
  `focus13_conformance_test.go` in the testing framework list, and add a
  `### FOCUS 1.4 Cost and Usage Pattern (055-focus-14-cost-usage-columns)` note (zero-copy
  `json.Valid`, the relaxed provider rule keeping `FieldName "provider_name"`, the 69-80 reservation,
  and the proto/FocusFieldNames/mock parity test).
- [X] T034 Write `PR_MESSAGE.md` (gitignored) with title
  `feat(pluginsdk): add FOCUS 1.4 cost and usage columns` (commitlint caps the header at 72 characters), Summary /
  Test plan / Notes / Follow-ups, body lines ≤ 100 chars, no `#NNN` except the final
  `Closes #541`. Validate with `npx commitlint < PR_MESSAGE.md`. Do not edit CHANGELOG.md.

---

## Dependencies & Execution Order

- **Setup (T001, T002)**: no dependencies.
- **Test-first gate**: T006, T007, T009 to T011, T015 to T018, T021 to T023 are written after Setup
  and before T003, and must fail (compile failures in Go, assertion failures in vitest).
- **Foundational (T003 to T005)**: sequential; blocks all implementation tasks.
- **US1 (T008)**: after Foundational. Independent of the other stories.
- **US2 (T012 to T014)**: after Foundational. The `pluginsdk` test package also needs T019's
  sentinel and T020's setter to compile, so declare both sentinels and both setters together if
  compiling the package early.
- **US3 (T019, T020)**: after Foundational and T012 (shares `validateFocus14Rules`).
- **US4**: T024 and T025 after Foundational; T026 to T028 after US1 to US3 (they document final
  names).
- **Polish (T029 to T034)**: after all stories.

### Parallel Opportunities

- T006, T007, T009, T010, T011 are in different files.
- T021, T022, T023 are in different files.
- T026, T027, T028 are in different files.

---

## Implementation Strategy

### MVP First (US1)

The provider-rule fix (T008) is the smallest shippable unit and resolves the conformance bug alone.

### Incremental Delivery

1. Foundational + US1: the conformance contradiction is fixed and the fields exist.
2. US2: invoice linking with its rule.
3. US3: eligibility details with allocation-free JSON validation.
4. US4: dry-run, TypeScript, and docs; then Polish and the PR message.

## Notes

- Generated files are never edited by hand; always `make generate`.
- Do not commit; the user commits after T029 to T032 pass.
