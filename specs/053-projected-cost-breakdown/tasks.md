# Tasks: Projected Cost Breakdown

**Input**: Design documents from `specs/053-projected-cost-breakdown/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: REQUIRED. Constitution V (Test-First Protocol) is non-negotiable, and it covers the proto
field too. The test tasks of Phases 3 to 6 (T006, T007, T010, T011, T013 to T015, T021) are written
**before** T002, and at that point they fail to compile because `CostBreakdown`,
`WithProjectedCostBreakdown` and the sentinel errors do not exist yet. That compile failure is the
"fails against the current proto" state. T002 to T004 then add the field, and the behavior tests
still fail until each story's implementation lands.

**Organization**: Tasks are grouped by user story (spec.md). US1 and US2 are both P1. US2 mostly
verifies that US1 introduced nothing breaking.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: The user story the task belongs to (US1 to US4)

## Path Conventions

This is a single-repository SDK. Paths are relative to the repo root: `proto/`, `sdk/go/`,
`sdk/typescript/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm a clean baseline before any change.

- [X] T001 Record the baseline: run `make test` and
  `go test ./sdk/go/pluginsdk/ -run '^$' -bench 'ValidateGetProjectedCostResponse' -benchmem`,
  and save the benchmark output to `specs/053-projected-cost-breakdown/baseline-bench.txt` for the
  SC-004 comparison in T030. Delete the file before the PR.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add the wire field. Every story's implementation depends on the generated
`CostBreakdown` field.

**⚠️ CRITICAL**: No user story *implementation* can begin until this phase is complete. The test tasks
listed below come first.

**Test-first gate (Constitution V)**: Before T002, write T006, T007, T010, T011, T013, T014, T015 and
T021. Run `go vet ./sdk/go/pluginsdk/ ./sdk/go/testing/` and
`npx vitest run test/integration.test.ts` in `sdk/typescript/packages/client`. Both MUST fail: Go on
the undefined `CostBreakdown` field, option and sentinels; vitest on the missing `costBreakdown`
(the client `tsconfig.json` excludes `test/`, so `tsc --noEmit` cannot see this failure). Record the failure
output in the PR body.

- [X] T002 Add `map<string, double> cost_breakdown = 15;` after `map<string, string> metadata = 14;`
  in `GetProjectedCostResponse` in `proto/finfocus/v1/costsource.proto`. Use the full field comment
  from `specs/053-projected-cost-breakdown/contracts/proto-changes.md` word for word: semantics,
  producer constraints, recommended keys, the EC2 example, and backward compatibility.
- [X] T003 Update the `EstimateCostResponse` message comment in
  `proto/finfocus/v1/costsource.proto`. Replace the "Future versions may add optional breakdown
  fields" sentence with the text in `contracts/proto-changes.md`, which says a future breakdown
  "MUST follow the rules of GetProjectedCostResponse.cost_breakdown".
- [X] T004 Run `make generate`. Confirm that `sdk/go/proto/finfocus/v1/costsource.pb.go` gains
  `CostBreakdown map[string]float64` and `GetCostBreakdown()`, and that
  `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts` gains
  `costBreakdown: { [key: string]: number }`. Restore any unrelated `*.connect.go` or other
  generated reformatting with `git checkout -- <file>`.
- [X] T005 Run `bin/buf lint` and `bin/buf breaking --against '.git#branch=main'`. Both must pass,
  since the change is additive (Constitution VI).

**Checkpoint**: The field exists in the Go and TypeScript bindings. `go build ./...` passes. The
pre-written tests compile but only pass with the stories' implementation tasks. Missing symbols
such as `WithProjectedCostBreakdown` and the sentinels may still block compiling a test package, and
that is expected until T008 and T016.

---

## Phase 3: User Story 1 - Plugin reports cost components (Priority: P1) 🎯 MVP

**Goal**: A plugin can set a named component breakdown, and a consumer receives it unchanged over
gRPC, including inside batch results.

**Independent Test**: A harness test configures the mock with `{compute, root_volume}` and receives
both entries on `GetProjectedCost`. Their sum equals `cost_per_month`.

### Tests for User Story 1 (write first, must fail) ⚠️

- [X] T006 [P] [US1] Add `TestWithProjectedCostBreakdown` to `sdk/go/pluginsdk/helpers_test.go`, a
  table test with these cases:
  - The EC2 example `{"compute": 7.592, "root_volume": 0.80}` combined with
    `WithProjectedCostDetails(0.0104, "USD", 8.392, "...")` sets both entries.
  - A nil map and an empty map leave `GetCostBreakdown()` empty.
  - Changing the caller's map after `NewGetProjectedCostResponse` returns does NOT change the
    response (copy semantics, research R8).
  - Option order does not matter: the breakdown option before or after the details option gives the
    same result.
- [X] T007 [P] [US1] Create `sdk/go/testing/cost_breakdown_test.go` (package `testing_test`, modeled on
  `sdk/go/testing/metadata_test.go`) with these tests:
  - `TestGetProjectedCostBreakdown`: start `plugintesting.NewTestHarness` with a `MockPlugin` whose
    `ProjectedCostBreakdown = map[string]float64{"compute": 3, "root_volume": 1}`, call
    `GetProjectedCost`, and assert both keys are present. Assert the values are in a 3:1 ratio, that
    they sum to `CostPerMonth` within `1e-9`, and that
    `pluginsdk.ValidateGetProjectedCostResponse(resp)` returns nil.
  - `TestGetProjectedCostBreakdown_Nil`: a mock without a breakdown returns an empty breakdown.
  - `TestGetProjectedCostBreakdown_DryRun`: with `dry_run=true`, the breakdown is empty.
  - `TestBatchProjectedCostBreakdown`: the batch path carries the breakdown inside
    `CostData.projected_cost`.

  Do not use `t.Parallel()` in subtests that share a harness.

### Implementation for User Story 1

- [X] T008 [US1] Add `WithProjectedCostBreakdown(breakdown map[string]float64)
  GetProjectedCostResponseOption` to `sdk/go/pluginsdk/helpers.go`, directly after
  `WithProjectedCostExpiresAt`. Copy the input map. A nil or empty input leaves the field empty. It
  does NOT panic and does NOT validate. Its godoc must follow `contracts/sdk-helpers.md`, including
  the EC2 usage example and the note "Call ValidateGetProjectedCostResponse on the result". T006 must
  now pass.
- [X] T009 [US1] Add `ProjectedCostBreakdown map[string]float64` to `MockPlugin` in
  `sdk/go/testing/mock_plugin.go`, and document it next to `ProjectedCostExpiresAtDuration` in the
  struct field list at about line 106. In `GetProjectedCost`, only on the non-dry-run path, when the
  map is non-empty: compute `weightSum`, then set each entry to
  `costPerMonth * w / weightSum`. Weights that are negative or non-finite, or that lack a positive
  finite sum, cannot be scaled to a valid breakdown, so return `codes.FailedPrecondition` for them
  (changed after PR review; the first version returned all zeros for a zero sum, which fails the sum
  rule when `cost_per_month > 0`). Leave the response unchanged when the map is nil (research R10).
  T007 must now pass.

**Checkpoint**: US1 is complete. Run `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -run
'CostBreakdown'` and it passes.

---

## Phase 4: User Story 2 - Existing plugins and consumers keep working (Priority: P1)

**Goal**: Plugins and hosts that know nothing about the breakdown are unaffected (FR-002, SC-002).

**Independent Test**: The full existing suite passes unchanged, and a response without a breakdown
still validates.

### Tests for User Story 2 ⚠️

- [X] T010 [P] [US2] Add backward-compatibility cases to `TestValidateGetProjectedCostResponse` in
  `sdk/go/pluginsdk/validation_pricing_test.go`, all expected valid:
  - No breakdown, `cost_per_month = 8.0`
  - No breakdown, `cost_per_month = 0`
  - No breakdown, with `DryRunResult` set
  - An empty non-nil map `map[string]float64{}`
- [X] T011 [P] [US2] Add `TestCostBreakdownWireCompat` to `sdk/go/testing/cost_breakdown_test.go`.
  It must show that a response with an empty breakdown puts no field 15 bytes on the wire, so it is
  byte-identical to a pre-feature response:
  - `proto.Marshal` it, walk the bytes with `protowire.ConsumeField`, and assert that no field
    number 15 appears.
  - Assert that a response with a breakdown round-trips through `proto.Marshal` and `proto.Unmarshal`
    with `proto.Equal`.

  Wire compatibility for older readers is proven by `buf breaking` in T005, so this test does not
  simulate an older reader.

### Implementation for User Story 2

- [X] T012 [US2] Run `make test`. Every test that existed before this feature passes without
  changes. If one fails, fix the new code, not the old test.

**Checkpoint**: US1 and US2 are complete. The feature is safe to ship as an MVP.

---

## Phase 5: User Story 3 - Early feedback on inconsistent breakdowns (Priority: P2)

**Goal**: `ValidateGetProjectedCostResponse` rejects malformed breakdowns with named sentinel
errors, and allocates nothing on valid input (FR-004 to FR-007, FR-011, SC-003, SC-004).

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'CostBreakdown' -v` passes every case in the
quickstart step 2 table. The new benchmarks show `0 allocs/op`.

### Tests for User Story 3 (write first, must fail) ⚠️

- [X] T013 [P] [US3] Add `TestValidateCostBreakdown` to `sdk/go/pluginsdk/validation_pricing_test.go`,
  a table test using `require.ErrorIs` for failure cases and `require.NoError` for valid ones.

  Valid cases:
  - EC2: total 8.392, `{compute: 7.592, root_volume: 0.80}`
  - Single component: total 8.0, `{storage: 8.0}`
  - Rounding within the absolute tolerance: total 10.00, components summing to 10.005
  - Large total within the relative tolerance: total 50000, sum 50040 (tolerance 50)
  - Zero total: total 0, `{compute: 0}`
  - Exactly 32 entries summing to the total
  - A 64-byte key `a` + 63 × `b`
  - A key with digits and underscores: `data_transfer_2`

  Invalid cases:
  - `ErrCostBreakdownSumMismatch` for a total of 8.392 and a sum of 9.00. Assert the message contains
    both numbers.
  - `ErrCostBreakdownSumMismatch` for a total of 0 and `{compute: 0.5}`.
  - Valid (listed here because it pairs with the case above): a total of 0 and `{compute: 0.005}`,
    within the 0.01 absolute tolerance (spec edge case "Zero total with components").
  - `ErrCostBreakdownSumMismatch` for a total of 10.00 and a sum of 10.02, just outside 0.01.
  - `ErrCostBreakdownInvalidValue` for `{compute: -1}`, `{compute: math.NaN()}` and
    `{compute: math.Inf(1)}`. Assert the message names the key.
  - `ErrCostBreakdownInvalidKey` for `RootVolume`, `root-volume`, `1st`, `_x`, `""`, `root volume`, a
    65-byte key and `café`.
  - `ErrCostBreakdownTooManyEntries` for 33 entries.
  - `ErrCostBreakdownWithDryRun` when `DryRunResult` is non-nil and the breakdown is non-empty.
  - Precedence: `{compute: NaN}` with any total reports `ErrCostBreakdownInvalidValue`, not a sum
    mismatch (research R6).
- [X] T014 [P] [US3] Add benchmarks to `sdk/go/pluginsdk/validation_pricing_test.go`, next to
  `BenchmarkValidateGetProjectedCostResponse_WithMetadata32`, and call `b.ReportAllocs()` in each:
  - `BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown`: two entries, the EC2 example.
  - `BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown32`: 32 entries with 64-byte keys.
  - `BenchmarkValidateGetProjectedCostResponse_Invalid_CostBreakdownSum`: the error path.
- [X] T015 [P] [US3] Add a `TestValidateGetProjectedCostResponse_CostBreakdownZeroAlloc` test to
  `sdk/go/pluginsdk/validation_pricing_test.go`. It uses `testing.AllocsPerRun(100, ...)` on the
  32-entry valid response and asserts `== 0`, to guard SC-004 in CI.

### Implementation for User Story 3

- [X] T016 [US3] Add five sentinel errors to the error `var` block in
  `sdk/go/pluginsdk/validation.go`, next to `ErrMetadataTooManyEntries`, each with a godoc comment.
  Use the exact messages from `contracts/sdk-helpers.md`:
  - `ErrCostBreakdownTooManyEntries` ("cost_breakdown has too many entries")
  - `ErrCostBreakdownInvalidKey` ("cost_breakdown key is invalid")
  - `ErrCostBreakdownInvalidValue` ("cost_breakdown value is invalid")
  - `ErrCostBreakdownSumMismatch` ("cost_breakdown does not sum to cost_per_month")
  - `ErrCostBreakdownWithDryRun` ("cost_breakdown must be empty for dry-run responses")
- [X] T017 [US3] Add four unexported constants to the constants block in
  `sdk/go/pluginsdk/validation.go`, next to `maxMetadataEntries = 32`:
  - `maxCostBreakdownEntries = 32`
  - `maxCostBreakdownKeyLen = 64`
  - `costBreakdownAbsTolerance = 0.01`
  - `costBreakdownRelTolerance = 0.001`
- [X] T018 [US3] Implement `validateCostBreakdownKey(k string) error` in
  `sdk/go/pluginsdk/validation.go` as an allocation-free byte loop (no regexp, research R4):
  - Length is 1 to 64 bytes.
  - `k[0]` is in `'a'..'z'`.
  - Every later byte is in `'a'..'z'`, `'0'..'9'` or `'_'`.

  Errors wrap `ErrCostBreakdownInvalidKey` with `%q` of the key and the offending byte position.
- [X] T019 [US3] Implement
  `validateCostBreakdown(breakdown map[string]float64, costPerMonth float64, isDryRun bool) error`
  in `sdk/go/pluginsdk/validation.go`. Follow the order in `data-model.md` exactly:
  1. If the map is empty, return nil.
  2. If `isDryRun`, return `ErrCostBreakdownWithDryRun`.
  3. If `len > 32`, return `ErrCostBreakdownTooManyEntries`, formatted `got %d, max %d`.
  4. For each entry, validate the key, then check that the value is finite and `>= 0` (otherwise
     `ErrCostBreakdownInvalidValue` naming the key and value), then add it to `sum` (plain float64,
     research R3).
  5. Set `tol := max(costBreakdownAbsTolerance, costBreakdownRelTolerance*costPerMonth)`. If
     `math.Abs(sum-costPerMonth) > tol`, wrap `ErrCostBreakdownSumMismatch` with
     `sum %g, cost_per_month %g, tolerance %g`.
- [X] T020 [US3] In `ValidateGetProjectedCostResponse` in `sdk/go/pluginsdk/validation.go`, call
  `validateCostBreakdown(resp.GetCostBreakdown(), costPerMonth, resp.GetDryRunResult() != nil)`
  after `validateMetadataMap`, and wrap the error with the `"GetProjectedCostResponse: %w"` prefix.
  Extend the function's godoc numbered rule list with item 8, worded as in `contracts/sdk-helpers.md`.
  T013 to T015 must now pass.

**Checkpoint**: US3 is complete. Every quickstart step 2 case passes, and the benchmarks show
`0 allocs/op`.

---

## Phase 6: User Story 4 - Discoverable in docs and the TypeScript SDK (Priority: P3)

**Goal**: A plugin author can populate and validate a breakdown using only the documentation. A
TypeScript consumer can read it (FR-008, FR-010, FR-012, SC-005).

**Independent Test**: A reader of the README section can build and validate the EC2 example. The
TypeScript integration test reads `costBreakdown.compute`.

### Tests for User Story 4 (write first, must fail) ⚠️

- [X] T021 [P] [US4] In `sdk/typescript/packages/client/test/mocks/handlers.ts`, add
  `costBreakdown: { compute: 120.0, root_volume: 30.0 }` to the `GetProjectedCost` handler response,
  which has `costPerMonth: 150.0`. In `sdk/typescript/packages/client/test/integration.test.ts`,
  extend the projected-cost test to assert `response.costBreakdown.compute === 120.0`,
  `response.costBreakdown.root_volume === 30.0`, and that the values sum to `costPerMonth`.
- [X] T022 [P] [US4] Add `ExampleWithProjectedCostBreakdown` to
  `sdk/go/pluginsdk/example_test.go`. It builds the EC2 response, runs
  `ValidateGetProjectedCostResponse`, and prints the sorted keys and the validation result, with an
  `// Output:` block, so the documented example compiles and runs (Constitution XIV).

### Implementation for User Story 4

- [X] T023 [US4] Add a `## Cost Breakdown Helpers (cost_breakdown)` section to
  `sdk/go/pluginsdk/README.md`, directly after `## Caching Hint Helpers (expires_at)` and before
  `## Pagination Helpers`, and add it to the Table of Contents. It must contain:
  - The field's purpose, and that an empty map means "no breakdown available".
  - The rules table from `data-model.md`: key format `[a-z][a-z0-9_]*` of 1 to 64 bytes, at most 32
    entries, finite values `>= 0`, sum within `max(0.01, 0.001 × cost_per_month)` as a hard error,
    and empty for dry-run responses.
  - The recommended vocabulary: `compute`, `storage`, `root_volume`, `network`, `license`, `request`,
    `data_transfer`. State that consumers MUST accept others.
  - The EC2 and EBS examples, using `WithProjectedCostBreakdown` and
    `ValidateGetProjectedCostResponse`.
  - The five sentinel errors, for `errors.Is`.
  - A note that `billing_detail` is unchanged.
- [X] T024 [P] [US4] If `sdk/go/testing/README.md` documents `MockPlugin` configuration fields, add
  `ProjectedCostBreakdown` there with its weight-scaling semantics. Otherwise skip this task and
  record that in the PR.
- [X] T025 [P] [US4] Check `docs/` and `PLUGIN_DEVELOPER_GUIDE.md` for a reference to
  `GetProjectedCostResponse` fields, for example `metadata` or `estimate_quality`. Where one exists,
  add a short `cost_breakdown` entry that links to the README section. Also mention that
  `billing_detail` no longer needs to carry component costs for machines to parse.
- [X] T026 [US4] In `sdk/typescript/packages/client`, run `npx vitest run` and `npx tsc --noEmit`.
  T021 must pass. Do not use `npm run build`, because of the known tsup TS5101 failure noted in
  CLAUDE.md.

**Checkpoint**: All four stories are complete.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T027 [P] Run `goimports -w` on the changed Go files, then run `golangci-lint run ./...` (the
  baseline is 0 issues, so fix every finding). Watch for `testifylint`, which requires `require.Len`,
  and `gocognit` on `validateCostBreakdown`, whose threshold is 20. Split the per-entry check into a
  helper rather than adding `nolint`.
- [X] T028 [P] Run `make lint-markdown` and `make lint-yaml`, and fix any findings in the README,
  docs and spec files.
- [X] T029 Run `make test` and `go test ./sdk/go/testing/ -v -run 'CostBreakdown|Conformance'`.
  Everything must pass.
- [X] T030 Run `go test ./sdk/go/pluginsdk/ -run '^$' -bench 'ValidateGetProjectedCostResponse'
  -benchmem`. Compare it with `baseline-bench.txt` from T001: `_Valid` must be within 10%, and the new
  breakdown benchmarks must show `0 allocs/op` (SC-004). Then delete `baseline-bench.txt`.
- [X] T031 Run every step in `specs/053-projected-cost-breakdown/quickstart.md` from 1 to 7 and
  confirm each expected outcome.
- [X] T032 Update `CLAUDE.md`. Change the `053-projected-cost-breakdown` Recent Changes entry from
  "Planned" to "Added". Add a short `### Cost Breakdown Pattern (053-projected-cost-breakdown)` note
  under Project-Specific Patterns covering the sum tolerance constants, the fact that the option does
  not validate, and the mock plugin's weight scaling.
- [X] T033 Write `PR_MESSAGE.md` with a conventional-commit title
  `feat(proto): add cost_breakdown map to GetProjectedCostResponse`, and `Closes #433` in the body.
  Validate it with `cat PR_MESSAGE.md | npx commitlint`. List the aws-public producer and the Core
  display work as follow-ups in the body; do not file issues. Do not edit CHANGELOG.md, because
  release-please owns it.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (T001)**: No dependencies.
- **Test-first gate**: T006, T007, T010, T011, T013, T014, T015 and T021 are written after Setup and
  before T002, and must fail to compile (Constitution V).
- **Foundational (T002 to T005)**: Depend on the test-first gate and are strictly sequential: proto,
  then generate, then buf. They block every story's implementation tasks.
- **US1 (T006 to T009)**: Depends on Foundational. T006 and T013 share the `sdk/go/pluginsdk` test
  package, so the US1 checkpoint also needs T016 (sentinel declarations only, no behavior) for that
  package to compile.
- **US2 (T010 to T012)**: Depends on Foundational. T012 should run after US1, so it covers the new
  code.
- **US3 (T013 to T020)**: Depends on Foundational. It does not depend on US1, but T007's
  "validator returns nil" assertion only becomes meaningful once T020 lands.
- **US4 (T021 to T026)**: T021 and T026 depend only on Foundational. T022 to T025 depend on US1 (the
  option) and US3 (the error names).
- **Polish (T027 to T033)**: Depends on all stories.

### Within Each Story

- Tests are written first, before T002, and fail to compile. After Foundational they compile and
  fail on behavior, and then the implementation makes them pass.
- US3: T016 and T017 (declarations) come before T018 (key check), then T019 (map check), then T020
  (wiring).

### Parallel Opportunities

- T006 and T007 are in different files.
- T010 and T011 are in different files.
- T013, T014 and T015 are in the same file, so write them in one editing pass, or in parallel if
  they go in separate test functions merged carefully.
- T021 and T022, then T024 and T025, are all in different files.
- After Foundational, US1 and US3 can proceed in parallel: US1 touches `helpers*.go` and
  `sdk/go/testing/`, while US3 touches `validation*.go`.

---

## Parallel Example: after Phase 2

```bash
# Developer A, US1:
Task: "T006 TestWithProjectedCostBreakdown in sdk/go/pluginsdk/helpers_test.go"
Task: "T007 harness tests in sdk/go/testing/cost_breakdown_test.go"

# Developer B, US3:
Task: "T013 TestValidateCostBreakdown in sdk/go/pluginsdk/validation_pricing_test.go"

# Developer C, US4 (TypeScript only):
Task: "T021 costBreakdown in test/mocks/handlers.ts + integration.test.ts"
```

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Phase 1 (baseline), then Phase 2 (proto and generate).
2. US1: the option, the mock and the round-trip tests.
3. US2: backward compatibility is confirmed.
4. **Stop and validate**: plugins can already report breakdowns. Without US3 nothing enforces the
   sum rule yet, so do not release without US3, because the clarified hard error is part of the
   contract.

### Incremental Delivery

1. Foundational, US1 and US2: the field works end to end.
2. US3: validation is enforced. This is the minimum releasable unit, because the proto comment
   promises validator enforcement.
3. US4: docs and the TypeScript test.
4. Polish, then the PR.

---

## Notes

- Generated files (`*.pb.go`, `*_pb.ts`) are never edited by hand. Always use `make generate`.
- Keep test files' header style consistent with their neighbors in the same package.
- Do not commit. The user commits, after T027 to T031 pass.
