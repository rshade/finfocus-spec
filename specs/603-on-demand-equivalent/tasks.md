# Tasks: On-Demand Equivalent for Spot Results

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md),
[data-model.md](data-model.md), [response and SDK contract](contracts/response-and-sdk.md),
and [quickstart.md](quickstart.md)

**Status**: Planning checklist for issue #646. Implementation is already present in the
working tree. Reconcile each task with the current diff and verification evidence before
marking it complete; unchecked boxes do not imply its code is missing.

**Tests**: Required by the specification and Constitution Principle V. A new-field test may
fail to compile until the proto is regenerated; this is the expected red state.

**Format**: `[ID] [P?] [Story?]` followed by an action and exact file path. `[P]` means
the task can run independently after its listed prerequisites.

## Phase 1: Setup

**Goal**: Establish the pre-change contract and benchmark baseline.

- [X] T001 Inspect `proto/finfocus/v1/costsource.proto`, `sdk/go/pluginsdk/validation.go`,
  `sdk/go/testing/harness.go`, and `git status` to confirm field numbers 18 and 8 are unused and the baseline
  worktree is understood.
- [X] T002 Record the two valid-path validator benchmarks from the baseline checkout in
  `specs/603-on-demand-equivalent/plan.md`; use the command in `quickstart.md` with `-benchmem`.

## Phase 2: Foundational Wire Contract

**Goal**: Establish the optional fields and generated language types before SDK code.

- [X] T003 Write the failing presence and field-number contract cases for both responses in
  `sdk/go/on_demand_equivalent_test.go`; check absent versus present zero and field numbers 18 and 8.
- [X] T004 Add `optional double on_demand_equivalent_monthly_cost` at field 18 and field 8 in
  `proto/finfocus/v1/costsource.proto`; document currency, 730-hour basis, absence versus zero, DYNAMIC-only, finite
  non-negative values, advisory relationship to `price_options`, and legacy behavior; move the future cost-breakdown
  field note to 9 or later.
- [X] T005 Run `make generate` for `sdk/go/proto/finfocus/v1/costsource.pb.go` and
  `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts`; confirm Go pointers, optional
  TypeScript numbers, and no unrelated generated drift.
- [X] T006 Run `buf lint` and `buf breaking --against '.git#branch=main'` for `proto/finfocus/v1/costsource.proto`
  and record a clean additive-wire result in `specs/603-on-demand-equivalent/plan.md`.

**Checkpoint**: Both generated responses represent presence; prior request and batch messages
are unchanged.

## Phase 3: User Story 1 - Supply a Spot Counterpart (Priority: P1)

**Goal**: A DYNAMIC result can carry the plugin-supplied monthly on-demand counterpart.

**Independent Test**: A configured mock returns the equivalent on both RPCs; a present zero
is distinguishable from absence in Go and the TypeScript client.

### Tests

- [X] T007 [P] [US1] Write failing option tests in `sdk/go/pluginsdk/on_demand_equivalent_test.go` for copied
  pointer presence, zero, and no panic on invalid arguments.
- [X] T008 [P] [US1] Write failing mock tests in `sdk/go/testing/on_demand_equivalent_test.go` proving configured
  zero is present only for DYNAMIC projected and estimate responses.
- [X] T009 [P] [US1] Add client assertions for both optional response fields in
  `sdk/typescript/packages/client/test/integration.test.ts` and DYNAMIC JSON fixtures in
  `sdk/typescript/packages/client/test/mocks/handlers.ts`.

### Implementation

- [X] T010 [US1] Implement `WithOnDemandEquivalent(float64)` and `WithProjectedCostOnDemandEquivalent(float64)` in
  `sdk/go/pluginsdk/helpers.go`; each stores a copied pointer and defers validation without panicking.
- [X] T011 [US1] Add `MockPlugin.OnDemandEquivalentMonthlyCost *float64` in `sdk/go/testing/mock_plugin.go`; copy it
  to both RPC responses only when resolved pricing is DYNAMIC and leave dry-run and non-DYNAMIC responses unset.
- [X] T012 [US1] Add a runnable example covering both options and pointer presence in
  `sdk/go/pluginsdk/example_test.go`; run the focused Go tests in `quickstart.md`.

**Checkpoint**: Both RPC shapes can carry an optional counterpart without changing the
selected cost or pricing category.

## Phase 4: User Story 2 - Reject Invalid Equivalents (Priority: P1)

**Goal**: Plugin and conformance validators reject the same invalid values and categories.

**Independent Test**: Run the same table across projected and estimate responses in both
packages; valid DYNAMIC and absent legacy cases pass, and errors match with `errors.Is`.

### Tests

- [X] T013 [P] [US2] Write the conformance matrix in `sdk/go/testing/on_demand_equivalent_test.go` for DYNAMIC
  value/unset/zero, STANDARD/COMMITTED/UNSPECIFIED present values, STANDARD present zero, NaN, ±Inf, -0.01,
  value-before-category precedence, dry-run presence, and spot price above equivalent.
- [X] T014 [P] [US2] Write the SDK matrix, warning cases, and invalid-option no-panic cases in
  `sdk/go/pluginsdk/on_demand_equivalent_test.go` for both response validators and
  `CheckOnDemandEquivalentConsistency`; include legacy UNSPECIFIED with risk zero and both no-warning cases.

### Implementation

- [X] T015 [US2] Implement `ValidateOnDemandEquivalent(*float64, FocusPricingCategory)` and
  `ErrOnDemandEquivalentInvalidValue`/`ErrOnDemandEquivalentInvalidCategory` in
  `sdk/go/testing/on_demand_equivalent.go`; return nil for absence, check NaN/Inf then strict `< 0`, then DYNAMIC
  category, and wrap errors with field context.
- [X] T016 [US2] Call the shared rule from `ValidateProjectedCostResponse` and `ValidateEstimateCostResponse` in
  `sdk/go/testing/harness.go` without changing existing spot-risk validation.
- [X] T017 [US2] Alias both sentinels and call the shared rule immediately after spot-risk validation in both
  response validators in `sdk/go/pluginsdk/validation.go`; add `CheckOnDemandEquivalentConsistency` warnings for
  non-DYNAMIC presence and DYNAMIC absence only.
- [X] T018 [US2] Run the focused Go tests in `quickstart.md` and verify zero allocations for valid and nil
  `ValidateOnDemandEquivalent` paths in `sdk/go/testing/on_demand_equivalent_test.go`.

**Checkpoint**: SDK and conformance return the same sentinel identities. DYNAMIC with an
unknown equivalent remains valid.

## Phase 5: User Story 3 - Preserve Legacy Interoperability (Priority: P1)

**Goal**: Absent remains absent, present zero survives the wire, and old decoders ignore the
new field while reading all prior fields.

**Independent Test**: A real bufconn client receives both presence states on both RPCs;
batch serialization and an old descriptor retain the expected states.

- [X] T019 [P] [US3] Add bufconn tests for both RPCs and nested `BatchCostResponse` entries in
  `sdk/go/testing/on_demand_equivalent_test.go`; assert pointer presence for unset and present zero.
- [X] T020 [P] [US3] Add old-decoder protobuf tests for both response messages in
  `sdk/go/on_demand_equivalent_test.go`; verify unknown fields are ignored and old-to-new decoding leaves the
  pointer nil.
- [X] T021 [US3] Add a `MockLegacyPlugin` harness case and default `MockPlugin` validation case in
  `sdk/go/testing/on_demand_equivalent_test.go`; confirm both leave the new field absent.
- [X] T022 [US3] Run `go test ./sdk/go/...` and `buf breaking --against '.git#branch=main'` to check legacy and wire
  compatibility across the SDK.

**Checkpoint**: Existing plugins validate unchanged and batch needs no new field.

## Phase 6: Polish and Cross-Cutting Checks

- [X] T023 [P] Document producer and host guidance in `sdk/go/pluginsdk/README.md`, `sdk/go/testing/README.md`,
  `PLUGIN_DEVELOPER_GUIDE.md`, and `CLAUDE.md`; avoid `CHANGELOG.md`, `release-please-config.json`, and
  `sdk/go/pluginsdk/version.go`.
- [X] T024 Run the TypeScript client build and full tests in `sdk/typescript/packages/client/`; run both Go
  validator benchmarks with `-benchmem` and record before/after allocation results in
  `specs/603-on-demand-equivalent/plan.md`.
- [X] T025 Run `make validate`, `git diff --check`, and a generated-file diff review for
  `sdk/go/proto/finfocus/v1/costsource.pb.go` and
  `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts`; verify docs and schemas before
  completion.

## Dependencies and Execution Order

- Setup T001-T002 precedes wire tasks T003-T006. T003 must show the expected red state before
  T004-T005. All user stories require generated fields from T005.
- US1: T007-T009 are independent test edits after T005; T010-T012 follow their tests.
- US2: T013 and T014 are independent test edits after T005; T015 precedes T016-T018.
- US3: T019 and T020 can be written independently after T005; T019 and T021 use the mock from
  T011. T022 follows the compatibility tests.
- T023 can proceed after the wire and helper names are fixed. T024-T025 follow all desired
  story tasks and documentation.

## Parallel Opportunities

- After T005, T007, T008, and T009 touch separate Go SDK, mock, and TypeScript test files.
- After T005, T013 and T014 cover separate conformance and plugin SDK validators.
- After T011, T019 and T020 exercise independent bufconn/batch and old-decoder paths.

## Implementation Strategy

The minimum useful slice is T003-T012: both response types expose the plugin-supplied value
and preserve presence. T013-T018 add the invalid-data guard before host use. T019-T022 prove
legacy compatibility. T023-T025 close documentation, performance, and repository gates. Review
the existing working-tree implementation against each task before marking its checkbox.
