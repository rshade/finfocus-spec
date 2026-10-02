---

description: "Task list for 557-price-options"
---

# Tasks: Alternative Retail Price Options

**Input**: Design documents from `specs/557-price-options/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required. Constitution Principle V makes test-first mandatory. In each story, write the test
tasks first and confirm they fail (compile failure or behavior failure) before the implementation
tasks.

**Organization**: Tasks are grouped by user story. Stories run in priority order: US1 and US3 (P1),
then US2 and US4 (P2).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: The user story the task belongs to (US1-US4 from spec.md)

## Path Conventions

- Proto: `proto/finfocus/v1/costsource.proto`
- Go SDK: `sdk/go/pluginsdk/` (package tests are `package pluginsdk_test`), `sdk/go/testing/`
- TypeScript client: `sdk/typescript/packages/client/`

---

## Phase 1: Setup

**Purpose**: Branch and baseline

- [X] T001 Create and switch to branch `557-price-options` from an up-to-date `main` (`git switch -c 557-price-options`)
- [X] T002 Record the baseline: run `make test` and `go test ./sdk/go/pluginsdk/ -run '^$' -bench
      'Validate(GetProjectedCost|EstimateCost)Response_Valid$' -benchmem -count 6` and save the output to the scratch
      directory (not the repo) for the 10% budget check in T040

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The proto contract and generated code that every story depends on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

**Test-first gate (Constitution V)**: Before T003, write the test tasks T009, T010, T011, T012,
T018, T019, T022, T023, T024, T025, T030 and T031. Run
`go test ./sdk/go/pluginsdk/ ./sdk/go/testing/` and confirm the build fails: `pbc.PriceOption`,
`PriceOptions`, `WithProjectedCostPriceOptions`, `WithEstimatePriceOptions`, the three
`ErrPriceOption*` sentinels, and the two mock fields do not exist yet. For TypeScript, run
`npx vitest run` in `sdk/typescript/packages/client` and confirm the `priceOptions` assertions fail
(`npm run lint` cannot show this: the client `tsconfig.json` excludes `test/`). Phase 2 then makes
the tests compile; the rejection, dry-run, mock, and option tests keep failing on behavior until
each story's implementation lands.

- [X] T003 Add the `PriceOption` message to `proto/finfocus/v1/costsource.proto` directly after
      `GetProjectedCostResponse`, with the message and field comments copied verbatim from
      `specs/557-price-options/contracts/proto-changes.md`. Fields: `FocusPricingCategory category = 1; string model =
      2; string term = 3; double unit_price = 4; double monthly_cost = 5; double upfront_cost = 6; double
      savings_fraction = 7;`
- [X] T004 Add `repeated PriceOption price_options = 16;` after `cost_breakdown = 15` in `GetProjectedCostResponse` in
      `proto/finfocus/v1/costsource.proto`, with the comment from contracts/proto-changes.md. The comment must say:
      "Field 17 is held for a later per-region price list. Do not use it for anything else." Do NOT add a `reserved
      17;` statement (research R2)
- [X] T005 Add `repeated PriceOption price_options = 6;` after `expires_at = 5` in `EstimateCostResponse` in
      `proto/finfocus/v1/costsource.proto`, with the field comment from contracts/proto-changes.md (the field 7 hold
      is by comment only, no `reserved 7;`). Replace the message's leading "Future versions may add an optional cost
      breakdown" paragraph with the version in contracts/proto-changes.md, which says a breakdown MUST use field 8 or
      later
- [X] T006 Run `make generate`. Confirm with `git status --short` that only
      `sdk/go/proto/finfocus/v1/costsource.pb.go` and
      `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts` changed. Restore any unrelated
      reformatted `*.connect.go` files with `git checkout` (CLAUDE.md 051 note)
- [X] T007 Run `bin/buf lint` and `bin/buf breaking --against '.git#branch=main'`; both must pass. If the git object
      cannot be read, `git archive main proto buf.yaml` into a scratch directory and run `bin/buf breaking --against
      <dir>`
- [X] T008 Add three sentinel errors to the `var` block in `sdk/go/pluginsdk/validation.go`, after the
      `ErrCostBreakdown*` group, with godoc as in contracts/sdk-helpers.md: `ErrPriceOptionNil =
      errors.New("price_options entry is nil")`, `ErrPriceOptionInvalidValue = errors.New("price_options value is
      invalid")`, and `ErrPriceOptionsWithDryRun = errors.New("price_options must be empty for dry-run responses")`

**Checkpoint**: `go build ./...` passes; `pbc.PriceOption`, `GetPriceOptions()` on both parents, and the TS
`PriceOption` type exist

---

## Phase 3: User Story 1 - Plugin reports alternative prices on a projected cost (Priority: P1) 🎯 MVP

**Goal**: A plugin can attach alternative prices to `GetProjectedCostResponse`. The validator accepts them without
touching `cost_per_month`, and the mock serves them over gRPC.

**Independent Test**: `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -run 'PriceOption'` passes. A response with
`cost_per_month` 70.08 and two options validates, and `cost_per_month` is still 70.08.

### Tests for User Story 1 ⚠️ (written in the Phase 2 test-first gate; confirm the expected failures before implementing)

- [X] T009 [P] [US1] Add `TestValidateGetProjectedCostResponse_PriceOptions` (table-driven) to
      `sdk/go/pluginsdk/validation_pricing_test.go`:
  - valid cases:
    - "two options": `UnitPrice` 0.096, `CostPerMonth` 70.08, Reservation `1 Year` (0.0573 / 41.83 / 502.00 /
      0.403125) and SavingsPlan `3 Years` (0.0612 / 44.68 / 0 / 0.3625); assert `CostPerMonth` is still 70.08 after
      validation
    - "with cost_breakdown": `{compute: 60.08, os_disk: 10.00}` plus the same two options
  - error case: "dry run": the same options plus `DryRunResult: &pbc.DryRunResponse{}`, which must return
    `ErrPriceOptionsWithDryRun` (`require.ErrorIs`)
- [X] T010 [P] [US1] Add `TestWithProjectedCostPriceOptions` to `sdk/go/pluginsdk/helpers_test.go`:
  - no arguments leave `PriceOptions` nil
  - two entries are copied
  - after the call, mutating the caller's `*pbc.PriceOption` (for example setting `UnitPrice` to 99) does not change
    the response (deep copy)
  - a nil argument is kept as a nil element, not dropped
  - `cost_per_month` set by `WithProjectedCostDetails` is unchanged
- [X] T011 [P] [US1] Create `sdk/go/testing/price_options_test.go` with the Apache 2.0 header copied from
      `sdk/go/testing/cost_breakdown_test.go`. Add these tests:
  - `TestGetProjectedCostPriceOptions`: a `MockPlugin` with three `ProjectedCostPriceOptions` returns all three over
    the `TestHarness`; `GetCostPerMonth()` equals a default mock's for the same resource;
    `pluginsdk.ValidateGetProjectedCostResponse` returns nil
  - `TestGetProjectedCostPriceOptions_Nil`: no options are returned
  - `TestGetProjectedCostPriceOptions_DryRun`: dry-run responses carry no options (copy the dry-run setup from
    `TestGetProjectedCostBreakdown_DryRun`)
  - `TestGetProjectedCostPriceOptions_Isolation`: call `plugin.GetProjectedCost(ctx, req)` directly
    (not over the harness, where serialization already isolates responses) twice. Set the first
    response's `PriceOptions[0].UnitPrice` to 99, then assert the second response and
    `plugin.ProjectedCostPriceOptions[0]` still have the configured `UnitPrice`
  - `TestBatchProjectedCostPriceOptions`: the `BatchCost` PROJECTED path (copy from `TestBatchProjectedCostBreakdown`)
    carries the options
- [X] T012 [P] [US1] In `sdk/typescript/packages/client/test/mocks/handlers.ts`, add `priceOptions` with two entries
      (Reservation `1 Year` and SavingsPlan `3 Years`, using proto3 JSON camelCase: `category:
      "FOCUS_PRICING_CATEGORY_COMMITTED"`, `unitPrice`, `monthlyCost`, `upfrontCost`, `savingsFraction`) to the
      GetProjectedCost handler. In `sdk/typescript/packages/client/test/integration.test.ts`, assert
      `response.priceOptions` has length 2 with those values and `response.costPerMonth` is still 150.0

### Implementation for User Story 1

- [X] T013 [US1] In `sdk/go/pluginsdk/validation.go`, add an unexported `validatePriceOptions(options
      []*pbc.PriceOption, isDryRun bool) error` with godoc. In this story it returns `ErrPriceOptionsWithDryRun` when
      `isDryRun`, then loops the slice by index (`for i := range options`). The per-entry checks are added in US4
      (T030). It must not read `cost_per_month` or `cost_breakdown`
- [X] T014 [US1] Wire step 9 into `ValidateGetProjectedCostResponse` in `sdk/go/pluginsdk/validation.go`, after the
      `cost_breakdown` block. Guard it at the call site: `if options := resp.GetPriceOptions(); len(options) > 0 { if
      err := validatePriceOptions(options, resp.GetDryRunResult() != nil); err != nil { return
      fmt.Errorf("GetProjectedCostResponse: %w", err) } }`. Add to the godoc list: "9. price_options validation (if
      set): empty for dry-run responses, no nil entries, finite non-negative prices, finite savings_fraction. Never
      compared to cost_per_month."
- [X] T015 [P] [US1] Add `WithProjectedCostPriceOptions(options ...*pbc.PriceOption) GetProjectedCostResponseOption`
      to `sdk/go/pluginsdk/helpers.go`, after `WithProjectedCostBreakdown`, with the godoc and example from
      contracts/sdk-helpers.md. With zero arguments it sets `resp.PriceOptions = nil`. Otherwise it builds a new slice
      with `proto.CloneOf(o)` for each non-nil entry and keeps nil entries as nil. It does not validate. Add
      `ExampleWithProjectedCostPriceOptions` to `sdk/go/pluginsdk/example_test.go` after
      `ExampleWithProjectedCostBreakdown`, with an `// Output:` block
- [X] T016 [US1] In `sdk/go/testing/mock_plugin.go`, add the `ProjectedCostPriceOptions []*pbc.PriceOption` field
      after `ProjectedCostBreakdown`, with the godoc from contracts/sdk-helpers.md, and add it to the `MockPlugin`
      type comment's field list. In `GetProjectedCost`, after the breakdown block, set `resp.PriceOptions` to deep
      copies (`proto.CloneOf`) when the slice is non-empty. Dry-run responses must not get options (place the copy
      where the breakdown is applied, which the dry-run path already skips; verify)
- [X] T017 [US1] Run `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -run 'PriceOption'` and `cd sdk/typescript && npm
      run build --workspaces && npx --workspace packages/client vitest run`; T009-T012 must pass

**Checkpoint**: US1 is fully functional. Plugins can report alternatives on projected cost.

---

## Phase 4: User Story 3 - Existing plugins and hosts keep working (Priority: P1)

**Goal**: Responses without the field behave exactly as before, and an older host can decode responses that have it.

**Independent Test**: The whole existing validator suite passes unchanged, and a pre-feature schema decodes a response
with options and reads the same `cost_per_month`.

### Tests for User Story 3 ⚠️ (written in the Phase 2 test-first gate; confirm the expected failures before implementing)

- [X] T018 [P] [US3] Add `TestPriceOptionsWireCompat` to `sdk/go/testing/price_options_test.go`:
  - "EmptyListOmitsField": for `nil` and `[]*pbc.PriceOption{}`, the marshaled `GetProjectedCostResponse` has no field
    16 and the marshaled `EstimateCostResponse` has no field 6 (walk with `protowire.ConsumeField`, as
    `TestCostBreakdownWireCompat` does; add `priceOptionsProjectedFieldNumber = 16` and
    `priceOptionsEstimateFieldNumber = 6` constants)
  - "PopulatedListRoundTrips": `proto.Equal` after marshal and unmarshal, for both messages
  - "OlderConsumerDecodes": take `protodesc.ToFileDescriptorProto(pbc.File_finfocus_v1_costsource_proto)`, clone the file
    descriptor proto, remove field 16 from `GetProjectedCostResponse`, build it with `protodesc.NewFile` (resolve
    imports via `protoregistry.GlobalFiles`), decode the populated bytes into a `dynamicpb` message, and assert that
    `cost_per_month` equals the original and that the field 16 bytes are kept as unknown fields
- [X] T019 [P] [US3] Add `TestValidateEstimateCostResponse_PriceOptionsOmitted` and a matching projected-cost case to
      `sdk/go/pluginsdk/validation_pricing_test.go`: responses with nil and with empty `PriceOptions` return the same
      result as before the feature (nil for valid input; the same sentinel for an existing invalid input such as a
      negative `CostPerMonth`)

### Implementation for User Story 3

- [X] T020 [US3] Run `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/`: every pre-existing test must pass unmodified
      (SC-002). If a pre-existing test fails, fix the implementation, not the test
- [X] T021 [US3] Re-run `bin/buf breaking` (as in T007) after all proto edits are final, and confirm `grep -n
      'reserved 17\|reserved 7' proto/finfocus/v1/costsource.proto` prints nothing

**Checkpoint**: Backward compatibility is proven on the wire and in validation

---

## Phase 5: User Story 2 - Plugin reports alternative prices on a cost estimate (Priority: P2)

**Goal**: The same capability on `EstimateCostResponse`. `savings_fraction` there compares monthly costs.

**Independent Test**: An estimate response with `cost_monthly` 70.08 and two options validates, and `cost_monthly` is
unchanged; the mock serves options on `EstimateCost`.

### Tests for User Story 2 ⚠️ (written in the Phase 2 test-first gate; confirm the expected failures before implementing)

- [X] T022 [P] [US2] Add `TestValidateEstimateCostResponse_PriceOptions` to
      `sdk/go/pluginsdk/validation_pricing_test.go`: `CostMonthly` 70.08 with Reservation `1 Year` (`MonthlyCost`
      41.83, `SavingsFraction` (70.08-41.83)/70.08) and SavingsPlan `3 Years` validates, and `CostMonthly` is still
      70.08
- [X] T023 [P] [US2] Add `TestWithEstimatePriceOptions` to `sdk/go/pluginsdk/helpers_test.go` with the same cases as
      T010, using `NewEstimateCostResponse` and `WithEstimateCost("USD", 70.08)`
- [X] T024 [P] [US2] Add `TestEstimateCostPriceOptions` and `TestEstimateCostPriceOptions_Nil` to
      `sdk/go/testing/price_options_test.go`. A `MockPlugin` with `EstimateCostPriceOptions` returns them over the
      harness, `GetCostMonthly()` equals a default mock's, and `pluginsdk.ValidateEstimateCostResponse` returns nil.
      Use an `EstimateCostRequest` that the default mock supports (copy one from an existing EstimateCost test in
      `sdk/go/testing/`)
- [X] T025 [P] [US2] Add the same two `priceOptions` entries to the EstimateCost handler in
      `sdk/typescript/packages/client/test/mocks/handlers.ts`, and assert them plus `costMonthly` still 200.0 in the
      `estimateCost` test in `sdk/typescript/packages/client/test/integration.test.ts`

### Implementation for User Story 2

- [X] T026 [US2] Wire `validatePriceOptions(options, false)` into `ValidateEstimateCostResponse` in
      `sdk/go/pluginsdk/validation.go` after the spot-risk check, with the same `len() > 0` call-site guard and an
      `fmt.Errorf("EstimateCostResponse: %w", err)` wrap. Add to its godoc order list: "3. price_options validation
      (if set): no nil entries, finite non-negative prices, finite savings_fraction. Never compared to cost_monthly."
- [X] T027 [P] [US2] Add `WithEstimatePriceOptions(options ...*pbc.PriceOption) EstimateCostResponseOption` to
      `sdk/go/pluginsdk/helpers.go` after `WithEstimateCost`, with the same copy semantics as T015 and godoc from
      contracts/sdk-helpers.md
- [X] T028 [US2] In `sdk/go/testing/mock_plugin.go`, add the `EstimateCostPriceOptions []*pbc.PriceOption` field after
      `EstimateCostExpiresAtDuration`, with godoc, and list it in the type comment. In `EstimateCost`, after the
      expires_at block, deep-copy it into `resp.PriceOptions` when non-empty. If T015 and T016 duplicated the clone
      loop, extract one unexported helper in `mock_plugin.go` (for example `clonePriceOptions`) used by both
- [X] T029 [US2] Run `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -run 'PriceOption'` and the TS vitest run from
      T017; T022-T025 must pass

**Checkpoint**: US1, US2, and US3 all work independently

---

## Phase 6: User Story 4 - Validator rejects malformed alternative prices (Priority: P2)

**Goal**: NaN, infinity, negative prices, and nil entries are rejected with an error that names the entry and field.
Legitimate edge values are accepted.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'PriceOption'` passes every row of the quickstart.md step 2
table, for both validators.

### Tests for User Story 4 ⚠️ (written in the Phase 2 test-first gate; confirm the expected failures before implementing)

- [X] T030 [P] [US4] Add `TestValidatePriceOptions_Rules` (table-driven) to
      `sdk/go/pluginsdk/validation_pricing_test.go`. Run each case through both `ValidateGetProjectedCostResponse` and
      `ValidateEstimateCostResponse`, with the bad entry at index 1 of two entries. Expected results:
  - `ErrPriceOptionInvalidValue`:
    - `unit_price`, `monthly_cost`, `upfront_cost`: NaN, +Inf, -Inf, -0.01
    - `savings_fraction`: NaN, +Inf, -Inf
    - assert with `require.ErrorIs`, and assert the message contains `price_options[1].<field>` (FR-011)
  - `ErrPriceOptionNil`: a nil entry (message contains `price_options[1]`)
  - valid (nil error):
    - `savings_fraction` -0.25
    - `savings_fraction` 0 with `cost_per_month`/`cost_monthly` 0
    - `category` `pbc.FocusPricingCategory(99)`
    - empty `model` and `term`
    - `upfront_cost` 0
    - `-0.0` for each price field (it is not `< 0`)
  - a first-failure-wins case: index 0 has a NaN `unit_price` and index 1 is nil; expect index 0's error
- [X] T031 [P] [US4] Add `TestValidatePriceOptions_ZeroAlloc` to `sdk/go/pluginsdk/validation_pricing_test.go`, using
      `testing.AllocsPerRun(100, ...)` on a valid projected response with four options and on a valid estimate
      response with four options; both must be 0 (FR-014)

### Implementation for User Story 4

- [X] T032 [US4] Complete the per-entry loop in `validatePriceOptions` in `sdk/go/pluginsdk/validation.go`. For each
      index `i`, in this order:
  1. if nil: `fmt.Errorf("%w: price_options[%d]", ErrPriceOptionNil, i)`
  2. `unit_price`, then `monthly_cost`, then `upfront_cost`: if `math.IsNaN(v) || math.IsInf(v, 0) || v < 0`, return
     `fmt.Errorf("%w: price_options[%d].%s %v must be finite and non-negative", ErrPriceOptionInvalidValue, i, field,
     v)`
  3. `savings_fraction`: if NaN or Inf, return `fmt.Errorf("%w: price_options[%d].savings_fraction %v must be finite",
     ErrPriceOptionInvalidValue, i, v)`

  Do not check `savings_fraction` range, `category`, `model`, or `term`, and do not compare to the primary cost
  (research R4). Keep the happy path allocation-free: read values with getters and avoid building a slice of field
  names per call; a small unexported helper `checkPriceOptionAmount(i int, field string, v float64) error` with a
  string-literal field name is fine. Keep the function within the funlen and gocognit limits in `.golangci.yml`
- [X] T033 [US4] Run `go test ./sdk/go/pluginsdk/ -run 'PriceOption' -v`; T030 and T031 must pass

**Checkpoint**: All four stories are complete

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Benchmarks, documentation, and the full gate

- [X] T034 [P] Add `BenchmarkValidateGetProjectedCostResponse_WithPriceOptions` and
      `BenchmarkValidateEstimateCostResponse_WithPriceOptions` (four valid options, `b.ReportAllocs()`) to
      `sdk/go/pluginsdk/validation_pricing_test.go`, next to
      `BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown`
- [X] T035 [P] Add `BenchmarkWithProjectedCostPriceOptions` to `sdk/go/pluginsdk/helpers_test.go`, next to
      `BenchmarkWithProjectedCostBreakdown`
- [X] T036 [P] Add a "Price Option Helpers (price_options)" section to `sdk/go/pluginsdk/README.md`, after "Cost
      Breakdown Helpers", with a TOC entry. Include:
  - one paragraph stating that the list is advisory and never summed into `cost_per_month`/`cost_monthly`, and how it
    differs from `cost_breakdown`
  - a rules table: field, rule, sentinel. Rows: nil entry → `ErrPriceOptionNil`;
    `unit_price`/`monthly_cost`/`upfront_cost` "finite and `>= 0`" → `ErrPriceOptionInvalidValue`; `savings_fraction`
    "finite, may be negative" → `ErrPriceOptionInvalidValue`; dry-run → `ErrPriceOptionsWithDryRun`
  - the `savings_fraction` basis for each RPC, and the term-total reservation rule
  - an Azure example (Consumption, Reservation, SavingsPlan) and an AWS example (On-Demand, Reserved Instance, Savings
    Plans) using `WithProjectedCostPriceOptions` and `WithEstimatePriceOptions` with exact symbol names (Principles
    II, XIV)
- [X] T037 [P] Add a paragraph and short example for `ProjectedCostPriceOptions` and `EstimateCostPriceOptions` to
      `sdk/go/testing/README.md`, after the `ProjectedCostBreakdown` paragraph. State that entries are returned as
      configured, are not validated, and are deep-copied per response
- [X] T038 [P] Add a bullet after the `cost_breakdown` bullet in `PLUGIN_DEVELOPER_GUIDE.md`: to report other purchase
      options (on-demand, reservation, savings plan), set `price_options`; it is advisory and never summed; link to
      the new pluginsdk README section
- [X] T039 Update `CLAUDE.md`: replace the "557-price-options (planned)" Recent Changes entry with the final summary,
      and add a "### Price Options Pattern (557-price-options)" section after "Cost Breakdown Pattern" covering:
      fields 16/6 and comment-held 17/7 (no `reserved`), validator rules and sentinels, the `len() > 0` guard,
      `proto.CloneOf` copy semantics, the mock returning entries unvalidated, and no change to
      `plugintesting.Validate*`
- [X] T040 Compare `_Valid` benchmarks against the T002 baseline (A/B against a `main` worktree with prebuilt test
      binaries if the box is loaded, per the CLAUDE.md 053 note). Both must stay within 10%, and every `PriceOptions`
      benchmark must show `0 allocs/op`
- [X] T041 Run `gofmt -l ./sdk/go` (must print nothing), then `golangci-lint run ./...` (allow over 5 minutes),
  and fix all findings
- [X] T042 Run `make test`. If `TestUsageSourceNotRegistered/connect` fails, rerun it in isolation before
  treating it as a regression (CLAUDE.md 053 note)
- [X] T043 Run `make lint-markdown` and `make validate-npm`, and run `npm run lint` in `sdk/typescript/packages/client`
- [X] T044 Walk through every step in `specs/557-price-options/quickstart.md` and confirm each
  expected result. Then mark this file's tasks complete and set `**Status**:` in
  `specs/557-price-options/spec.md` to `Complete`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none
- **Test-first gate**: the 12 test tasks listed in Phase 2 are written after Setup and before T003,
  and must fail to compile or fail on behavior (Constitution V)
- **Foundational (Phase 2)**: depends on Setup and blocks every story. T003 → T004 → T005 (same file)
  → T006 → T007; T008 can start once T006 has run
- **US1 (Phase 3)**: depends on Phase 2
- **US3 (Phase 4)**: depends on Phase 2. T018 builds its own populated response and does not need US1
  code; T020 is most meaningful after US1 lands
- **US2 (Phase 5)**: depends on Phase 2 and T013 (`validatePriceOptions` exists)
- **US4 (Phase 6)**: depends on T014 and T026 (both validators call `validatePriceOptions`), so it runs
  after US1 and US2 wiring
- **Polish (Phase 7)**: depends on all stories

### Within Each Story

- Tests already exist from the gate; after Phase 2 they compile, and each story's implementation
  turns its failing tests green
- `validation.go` tasks are sequential (T013 → T014 → T026 → T032) because they share one
  file
- `helpers.go` tasks (T015, T027) and `mock_plugin.go` tasks (T016, T028) are sequential within
  each file

### Parallel Opportunities

- T009, T010, T011, and T012 (US1 tests) touch four different files
- T015 (helpers.go) can run alongside T013/T014 (validation.go)
- T018 and T019 (US3 tests) run in parallel with each other
- T022-T025 (US2 tests) touch four different files
- T030 and T031 are both in `validation_pricing_test.go`; marked [P] against other files, but
  write them in one sitting
- T034-T038 (benchmarks and docs) are all different files

---

## Parallel Example: User Story 1

```bash
# Tests first, in parallel (four files):
Task: "T009 projected-cost validator table test in sdk/go/pluginsdk/validation_pricing_test.go"
Task: "T010 WithProjectedCostPriceOptions copy test in sdk/go/pluginsdk/helpers_test.go"
Task: "T011 mock harness tests in sdk/go/testing/price_options_test.go"
Task: "T012 TS priceOptions round-trip in sdk/typescript/packages/client/test/"

# Then implementation (two independent files in parallel):
Task: "T013+T014 validatePriceOptions and wiring in sdk/go/pluginsdk/validation.go"
Task: "T015 WithProjectedCostPriceOptions in sdk/go/pluginsdk/helpers.go"
```

---

## Implementation Strategy

### MVP First (US1 + US3)

1. Phase 1 Setup and Phase 2 Foundational
2. Phase 3 (US1): projected cost options, validator wiring, option, and mock
3. Phase 4 (US3): prove backward compatibility
4. **STOP and VALIDATE**: quickstart steps 1 and 4 for projected cost

Do not release the MVP without US4. Until T032 lands, the validator accepts NaN in an option.

### Incremental Delivery

1. Foundation → US1 → US3 (MVP, internal only)
2. US2: estimate parity
3. US4: rejection rules (required before release)
4. Polish: benchmarks, docs, full gate, then one PR

---

## Notes

- [P] = different files, no dependency on incomplete tasks
- Never edit generated files (`costsource.pb.go`, `costsource_pb.ts`) by hand; regenerate
- Do not edit `CHANGELOG.md` (release-please owns it); use a `feat(proto):` conventional commit
- Do not add a capability enum, a TS builder, or checks to `plugintesting.Validate*` (plan: Conformance scope)
- Commit only when the user asks
