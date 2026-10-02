# Tasks: Caller-Supplied Billing Account ID on Actual Cost Requests

**Input**: Design documents from `specs/585-actual-cost-billing-account-id/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/proto-diff.md, quickstart.md

**Tests**: Required. Constitution principle V makes test-first mandatory for protocol changes.
T005, T006, T011, T012, and T016 are written and run before their implementation tasks, and each
is seen failing. Before T002 they fail to compile, and after T002 they fail on behavior. Phase 2
comes first only so that the tests can compile against the new field (principle I, proto first).

**Organization**: Tasks are grouped by user story. US1 and US2 share the mock change, so US2 is
mostly tests over the same code.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US4)

## Phase 1: Setup

- [X] T001 Confirm the worktree baseline: `make generate && git diff --exit-code -- sdk/` is clean and `go test
  ./sdk/go/testing/ ./sdk/go/pluginsdk/` passes before any change

## Phase 2: Foundational (proto first; blocks all stories)

- [X] T002 Add `string billing_account_id = 9;` to `GetActualCostRequest` after `page_token` in
  proto/finfocus/v1/costsource.proto, with the exact comment in contracts/proto-diff.md (empty means not supplied,
  "Plugins MUST NOT invent a value", echo rule when non-empty, not a filter, ignored on dry run, same value on every
  page, not in tags, field 10 left free). Do not add a `reserved` statement
- [X] T003 Run `make generate`; confirm sdk/go/proto/finfocus/v1/costsource.pb.go gains `BillingAccountId` /
  `GetBillingAccountId()` and sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts gains
  `billingAccountId: string`; restore unrelated regenerated files with `git checkout` if doc comments elsewhere were
  reformatted
- [X] T004 Run `make buf-lint` and `buf breaking` against main (archive `origin/main` into a scratch directory if the
  git ref form fails in the worktree); both must report nothing

**Checkpoint**: Generated types expose the field in Go and TypeScript (FR-001, FR-009, SC-005).

## Phase 3: User Story 1 - Host supplies the billing account (P1) MVP

**Goal**: A non-empty request id ends up in a FOCUS record that passes validation.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run TestActualCostBillingAccount` and
`go test ./sdk/go/testing/ -run TestMockActualCostBillingAccount`.

- [X] T005 [P] [US1] Write failing test `TestMockActualCostBillingAccount` in
  sdk/go/testing/mock_billing_account_test.go (package `testing_test`, Apache header): with `BillingAccountId:
  "ba-123"`, every result has a non-nil `FocusRecord` whose `BillingAccountId` is `"ba-123"`, `ResourceId` equals the
  request resource id, and `BilledCost` equals the result `Cost`
- [X] T006 [P] [US1] Write failing test `TestActualCostBillingAccount` in
  sdk/go/pluginsdk/actual_cost_billing_account_test.go (package `pluginsdk_test`, Apache header): call
  `MockPlugin.GetActualCost` through a `plugintesting.TestHarness` with `BillingAccountId: "ba-123"` and assert each
  `FocusRecord` passes `pluginsdk.ValidateFocusRecord`
- [X] T007 [US1] In sdk/go/testing/mock_plugin.go, add an unexported `mockActualCostFocusRecord(req, result)` that
  builds the record in data-model.md (billing_account_id from the request, service_provider_name = `PluginName`, UTC
  calendar-month billing period, one-hour charge period, `USD`, Usage / Regular, Compute service, consumed quantity =
  usage amount with unit `Hours`, billed/effective/list cost = result cost), and attach it in `GetActualCost` only when
  `req.GetBillingAccountId() != ""`, before pagination
- [X] T008 [US1] Run T005 and T006; both pass

**Checkpoint**: The acceptance criterion "a plugin test can pass a non-empty id through the request and
produce a FocusRecord that passes ValidateFocusRecord" holds.

## Phase 4: User Story 2 - Host does not know the billing account (P1)

**Goal**: Empty id means no FOCUS record, no invented id, and unchanged cost.

**Independent Test**: `go test ./sdk/go/testing/ -run TestMockActualCostBillingAccount`.

- [X] T009 [US2] Add subtests to sdk/go/testing/mock_billing_account_test.go: (a) empty id leaves `FocusRecord` nil on
  every result; (b) `Cost`, `UsageAmount`, and result count are identical with and without an id for the same range
  (SC-003); (c) with `SetActualCostDataPoints(24)`, a one-day range, `PageSize: 5`, and an id, more than one page is
  returned and every page's records carry the id; (d) `DryRun: true` with an id returns a `DryRunResult` and no results
- [X] T010 [US2] Run `make test` and confirm every pre-existing test passes unchanged (SC-002)

## Phase 5: User Story 3 - Plugin author checks the contract (P2)

**Goal**: Conformance enforces the echo rule (FR-004, FR-011).

**Independent Test**: `go test ./sdk/go/testing/ -run 'TestValidateActualCostBillingAccount|TestRPCCorrectness'`.

- [X] T011 [P] [US3] Write failing table test `TestValidateActualCostBillingAccount` in sdk/go/testing/contract_test.go:
  nil request or nil response is handled without panic; empty request id passes even with records; matching id passes;
  results without a record pass; a mismatched id at index 1 fails, `errors.Is(err,
  plugintesting.ErrBillingAccountIDMismatch)` holds, and the message names `results[1].focus_record.billing_account_id`
- [X] T012 [P] [US3] Write failing tests in sdk/go/testing/rpc_correctness_test.go: `RPCCorrectnessTests()` contains
  `RPCCorrectness_GetActualCostBillingAccount` at `ConformanceLevelStandard`; it passes against `NewMockPlugin()`; it
  fails against a plugin that wraps the mock and rewrites `FocusRecord.BillingAccountId`; it passes against `MockPlugin`
  with `ShouldErrorOnActualCost: true` (returns `codes.NotFound`)
- [X] T013 [US3] Add `ErrBillingAccountIDMismatch` and `ValidateActualCostBillingAccount(req *pbc.GetActualCostRequest,
  resp *pbc.GetActualCostResponse) error` with godoc to sdk/go/testing/contract.go, returning `NewContractError` with
  the indexed field name
- [X] T014 [US3] Add `testGetActualCostBillingAccountRPC` and its `RPCCorrectness_GetActualCostBillingAccount` entry
  (Standard level) to sdk/go/testing/rpc_correctness.go, sending a one-day range with a fixed test id and accepting
  NotFound / Unavailable like `testGetActualCostRPC`
- [X] T015 [US3] Run T011, T012, and `go test -v -run TestConformance ./sdk/go/testing/`; all pass

## Phase 6: User Story 4 - TypeScript host sets the id (P2)

**Goal**: The id reaches the plugin on every page from the TypeScript iterator (FR-012).

**Independent Test**: `cd sdk/typescript/packages/client && npx vitest run test/pagination.test.ts`.

- [X] T016 [US4] Add a case to the `actualCostIterator` describe block in
  sdk/typescript/packages/client/test/pagination.test.ts: a multi-page run with `billingAccountId: "ba-123"` records
  every request and asserts each carries `"ba-123"`
- [X] T017 [US4] Run `cd sdk/typescript && npm ci && npm run build && npm test`; the client package passes (ignore only
  the known middleware TS7006 error if it appears)

## Phase 7: Polish & Cross-Cutting

- [X] T018 [P] Update PLUGIN_DEVELOPER_GUIDE.md: add fields 6-9 to the `GetActualCostRequest` block and an
  implementation note on `billing_account_id` (empty means not supplied, do not invent, echo into FOCUS records, not a
  filter, never carried in `tags`, not trimmed by the SDK)
- [X] T019 [P] Update sdk/go/pluginsdk/README.md with a short "Billing account id" subsection near the GetActualCost
  material, including a compilable snippet that calls `WithIdentity("", req.GetBillingAccountId(), "")` plus
  `WithServiceProvider(...)` only when the id is non-empty, and otherwise leaves `FocusRecord` unset
- [X] T020 [P] Update sdk/go/testing/README.md: list `RPCCorrectness_GetActualCostBillingAccount` and document
  `ValidateActualCostBillingAccount`
- [X] T021 [P] Add CLAUDE.md "Active Technologies" and "Recent Changes" entries for 585 by hand (do not run
  update-agent-context.sh), plus a short "Actual Cost Billing Account Pattern (585)" note
- [X] T022 Run gates: `make generate && git diff --exit-code -- sdk/`, `make buf-lint`, `make test`, `go test -v
  -tags=integration ./sdk/go/testing/`, `make lint-go`, `make lint-markdown`, `make lint-yaml`, `make validate-npm`
- [X] T023 Run quickstart.md scenarios 1-5 and record results

## Dependencies & Execution Order

- Phase 2 blocks everything (proto first).
- US1 (T005-T008) before US2 (T009-T010): US2 tests the same mock change.
- US3 (T011-T015) depends on T007 for its passing case against the mock.
- US4 (T016-T017) depends only on Phase 2.
- Polish after the stories it documents.

## Parallel Opportunities

- T005 and T006 (different packages).
- T011 and T012 (different files), and US4 alongside US3.
- T018-T021 (different files).

## Implementation Strategy

MVP is Phase 2 plus US1: the field exists and the reference plugin proves a valid FOCUS record. US2
locks in backward compatibility, US3 makes the rule enforceable, US4 closes SDK parity, then docs.
