# Tasks: FOCUS 1.4 Contract Commitment Columns

**Input**: Design documents from `specs/545-focus-14-contract-commitment/`

**Prerequisites**: plan.md, spec.md

**Tests**: REQUIRED. Conformance and validator cases landed with the implementation.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Contract commitment columns

- [x] T001 Add fields 13-30 and the seven FOCUS 1.4 enums to `proto/finfocus/v1/focus.proto`,
  then regenerate.
- [x] T002 [US1] Enforce the per-record 1.4 rules in `ValidateContractCommitment`, including
  null-versus-zero and the spend-only billing currency rule, at 0 allocs/op on valid input.
- [x] T003 [US1] Add builder setters, `WithBaselineTerms`, and the seven `IsValid*` helpers.
- [x] T004 [P] [US1] Serialize the new columns in JSON-LD and document them in `docs/focus-columns.md`.

## Phase 2: Contract applied object

- [x] T005 [US2] Add `FormatContractApplied` and `WithContractAppliedObject`. Deprecate
  `WithContractApplied` without changing the value it stores.
- [x] T006 [P] [US2] Add the TypeScript builder and `formatContractApplied`, and cover them with vitest.

## Phase 3: Conformance

- [x] T007 [US1] Assert fields 13-30 and the zero-versus-null discount in
  `sdk/go/testing/focus14_conformance_test.go`.
- [x] T008 Update existing commitment fixtures so records that must validate include the 1.4 columns.
