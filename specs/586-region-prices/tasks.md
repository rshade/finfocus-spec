# Tasks: Per-Region Retail Prices on Cost Responses

**Input**: Design documents from `specs/586-region-prices/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/proto-diff.md, quickstart.md

**Tests**: Required (constitution principle V). Test tasks run before their implementation tasks, and each test is
seen failing. Before T002 the tests fail to compile; after it they fail on behavior. Phase 2 comes first only so the
tests can compile against the new types (principle I, proto first).

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Record the baseline: `make generate && git diff --exit-code -- sdk/` is clean, `go test ./sdk/go/testing/
  ./sdk/go/pluginsdk/` passes, and a `main` worktree benchmark baseline exists for
  `BenchmarkValidateGetProjectedCostResponse_Valid` and `BenchmarkValidateEstimateCostResponse_Valid` (`-count 6`)

## Phase 2: Foundational (proto first)

- [X] T002 Add `message RegionPrice` and `repeated RegionPrice region_prices = 17` on `GetProjectedCostResponse` and
  `= 7` on `EstimateCostResponse` in proto/finfocus/v1/costsource.proto, with the comments in contracts/proto-diff.md
  (advisory, never summed, empty means no other regions, omit unpriced regions, empty on dry run). Leave 16 and 6
  free by comment; no `reserved`
- [X] T003 Run `make generate`; confirm Go and TS types expose `RegionPrices` / `regionPrices`, and revert any
  unrelated regenerated files
- [X] T004 Run `make buf-lint` and `buf breaking` against main; both clean

## Phase 3: User Stories 1 and 3 - Rows validate, bad rows are rejected (P1, P2) MVP

- [X] T005 [P] [US1] [US3] Write failing table test `TestValidateRegionPrices` in sdk/go/testing/region_price_test.go
  (package `testing_test`, Apache header): nil and empty slices pass; two valid rows pass; a zero price passes; a row
  currency that differs from any parent passes; each of nil row, empty region, NaN, +Inf, and negative `unit_price`,
  NaN `monthly_cost`, empty currency, and `"XXQ"` currency fails with `errors.Is(err,
  plugintesting.ErrInvalidRegionPrice)` and a message containing `region_prices[1].<field>`, or `region_prices[1]`
  for the nil row
- [X] T006 [P] [US1] [US3] Write failing tests in sdk/go/pluginsdk/region_price_test.go (package `pluginsdk_test`,
  Apache header): `ValidateGetProjectedCostResponse` passes with `cost_per_month` 70.08 and rows 65.70 and 80.30, and
  `cost_per_month` is unchanged; `ValidateEstimateCostResponse` passes the same way with `cost_monthly`; a bad row
  fails both with `pluginsdk.ErrInvalidRegionPrice`; rows plus `DryRunResult` fail projected validation with
  `pluginsdk.ErrRegionPricesWithDryRun`; responses without rows still pass
- [X] T007 [US3] Implement sdk/go/testing/region_price.go: `ErrInvalidRegionPrice`, `ErrRegionPricesWithDryRun`, and
  `ValidateRegionPrices(rows []*pbc.RegionPrice) error`, fail-fast in row order, 0 allocs on valid input, using
  `currency.IsValid`
- [X] T008 [US3] In sdk/go/pluginsdk/validation.go alias both sentinels, and call `ValidateRegionPrices` from
  `ValidateGetProjectedCostResponse` (with the dry-run rule) and `ValidateEstimateCostResponse`, each guarded by
  `len(rows) > 0` at the call site; update both godoc validation-order lists
- [X] T009 [US3] In sdk/go/testing/harness.go make `ValidateProjectedCostResponse` (with the dry-run rule) and
  `ValidateEstimateCostResponse` apply the same rules, guarded the same way
- [X] T010 [US1] [US3] Run T005 and T006; both pass

## Phase 4: User Story 2 - Existing responses stay valid (P1)

- [X] T011 [US2] Add `BenchmarkValidateGetProjectedCostResponse_WithRegionPrices` and
  `BenchmarkValidateEstimateCostResponse_WithRegionPrices` (four rows) to sdk/go/pluginsdk/region_price_test.go; both
  report 0 allocs/op
- [X] T012 [US2] A/B the two `_Valid` benchmarks against the T001 baseline with prebuilt test binaries (`-count 6`);
  the no-rows path stays within noise and at 0 allocs/op (SC-004)
- [X] T013 [US2] Run `make test`; every existing test passes unchanged (SC-002)

## Phase 5: User Story 4 - SDK options, mock, TypeScript (P2)

- [X] T014 [P] [US4] Write failing option tests in sdk/go/pluginsdk/region_price_test.go:
  `WithProjectedCostRegionPrices` and `WithEstimateCostRegionPrices` set copies of two rows (editing the caller's row
  afterwards does not change the response), and calling either with no rows leaves the field nil
- [X] T015 [P] [US4] Write failing test `TestMockRegionPrices` in sdk/go/testing/region_price_test.go: with
  `MockPlugin.RegionPrices` set, GetProjectedCost and EstimateCost over the harness return the rows and pass the
  harness validators; a dry-run GetProjectedCost returns no rows; unset leaves both empty. Also assert that
  `plugintesting.ValidateProjectedCostResponse` and `ValidateEstimateCostResponse` reject a response with a NaN row
  (`ErrInvalidRegionPrice`) and that the projected validator rejects rows with `DryRunResult` set
  (`ErrRegionPricesWithDryRun`) (FR-012)
- [X] T016 [US4] Implement `WithProjectedCostRegionPrices` and `WithEstimateCostRegionPrices` in
  sdk/go/pluginsdk/helpers.go with godoc, cloning rows with `proto.Clone`
- [X] T017 [US4] Add `MockPlugin.RegionPrices` to sdk/go/testing/mock_plugin.go, cloned onto non-dry-run projected
  cost responses and onto estimate responses, and list it in the struct doc comment
- [X] T018 [US4] Add two `regionPrices` rows to the GetProjectedCost handler in
  sdk/typescript/packages/client/test/mocks/handlers.ts, and assert them (region, prices, currency, and
  `costPerMonth` unchanged) in sdk/typescript/packages/client/test/integration.test.ts
- [X] T019 [US4] Run T014, T015, and `cd sdk/typescript && npm run build && npm test && npm run lint`; all pass

## Phase 6: Polish

- [X] T020 [P] Document `region_prices` in PLUGIN_DEVELOPER_GUIDE.md (projected cost response notes)
- [X] T021 [P] Add a "Region Prices" subsection to sdk/go/pluginsdk/README.md near the cost breakdown section, with an
  example using `WithProjectedCostRegionPrices`, the row rules, and the no-sum rule
- [X] T022 [P] Document `ValidateRegionPrices`, the sentinels, and `MockPlugin.RegionPrices` in
  sdk/go/testing/README.md
- [X] T023 [P] Add CLAUDE.md "Active Technologies", "Recent Changes", and a short "Region Prices Pattern (586)" note
  by hand
- [X] T024 Gates: `make generate && git diff --exit-code -- sdk/`, `make buf-lint`, `buf breaking`, `make test`, `go
  test -tags=integration ./sdk/go/testing/`, `go test -run TestConformance ./sdk/go/testing/`, `make lint-go`, `make
  lint-markdown`, `make lint-yaml`, `make validate-npm`
- [X] T025 Run quickstart.md scenarios 1-5

## Dependencies & Execution Order

- Phase 2 blocks everything.
- T005 and T006 precede T007-T009. T011-T013 need T008.
- T014 and T015 precede T016 and T017. T018 needs only Phase 2.
- Polish comes after the stories.

## Parallel Opportunities

T005 with T006; T014 with T015; T020-T023.

## Implementation Strategy

MVP is Phase 2 plus Phase 3: the field exists and validation enforces the row rules without a sum rule. Phase 4
proves no regression, Phase 5 closes SDK parity, then docs.

## Phase 7: Convergence

- [X] T026 Restate FR-016 and SC-004 as a measurable bound (0 allocs/op with and without rows, and no non-inlined call
  added to the no-rows path) and record the A/B numbers (main 5.65 ns, layout-only 6.08 ns, branch 6.30 ns on
  `BenchmarkValidateGetProjectedCostResponse_Valid`) in research.md R4 per SC-004 (partial)
