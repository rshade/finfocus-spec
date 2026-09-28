# Tasks: Stop Mixed-Currency Recommendation Totals

**Input**: Design documents from `/specs/054-stop-mixed-currency-sum/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/recommendation-summary.md

**Tests**: Required. The parent asked for tests first, and the existing tests lock the behavior this slice replaces.

**Organization**: Tasks are grouped by user story. US2 and US3 edit the same two functions as
US1, so they are sequential on those files even though each story has its own tests.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `sdk/go/pluginsdk/helpers.go` and `sdk/go/pluginsdk/helpers_test.go`
- `sdk/go/testing/mock_plugin.go` and `sdk/go/testing/mock_plugin_test.go`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Keep the slice inside the existing module

- [x] T001 Confirm `go.mod` stays at Go 1.27.1 with no new dependency for this slice

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Guard the rules that must not move while the summary changes

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T002 Leave `ResolveCurrency` in `sdk/go/testing/allocation.go` and
  `sdk/go/pluginsdk/allocator.go` unchanged, including "Empty takes the resolved
  currency" on priced rows and the USD fallback when no non-empty priced currency
  exists (`specs/054-stop-mixed-currency-sum/data-model.md`, "What does not change")

**Checkpoint**: Allocation's empty-as-USD rule is untouched. Story work can start.

---

## Phase 3: User Story 1 - Mixed Grand Total Is Withheld (Priority: P1) 🎯 MVP

**Goal**: 100 USD + 50 EUR no longer summarizes to 150. The total is 0 and the currency
is empty. 100 USD + 50 USD stays 150 USD. A real zero still names its currency. The call
does not fail and the signatures do not change.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -count=1 -run TestCalculateRecommendationSummary`

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T003 [P] [US1] In `sdk/go/pluginsdk/helpers_test.go`
  `TestCalculateRecommendationSummaryMixedCurrency`, expect total estimated savings 0,
  currency empty, and count 2 for 100 USD + 50 EUR. Leave
  `TestCalculateRecommendationSummaryConsistentCurrency` expecting 150 USD
- [x] T004 [P] [US1] Add a case in `sdk/go/pluginsdk/helpers_test.go` where every savings
  amount is 0 in USD and the summary currency stays USD (real zero, not withheld)
- [x] T005 [P] [US1] In `sdk/go/testing/mock_plugin_test.go` `TestCalculateMockSummary`
  case "mixed currencies clears currency field", set `TotalEstimatedSavings` to 0 and
  keep currency empty. Leave the 50 and 75 bucket sums as they are for now

### Implementation for User Story 1

- [x] T006 [US1] In `sdk/go/pluginsdk/helpers.go` `CalculateRecommendationSummary`, withhold
  the page total when two or more distinct non-empty currencies appear: total estimated
  savings 0 and currency empty. With at most one non-empty currency, the total is the sum
  of every impact amount, including empty-currency amounts, and currency is that code or
  empty when none was stated. Compare currency strings exactly. Do not change the
  signature. Counts stay "Number of recommendations, including those with no savings
  impact"
- [x] T007 [P] [US1] Apply the same page-total rule in `sdk/go/testing/mock_plugin.go`
  `CalculateMockSummary` without changing its signature. Do not import `pluginsdk` from
  this file

**Checkpoint**: Mixed page total is 0 with an empty currency. Same-currency 150 USD still
passes. Bucket sums may still be the old added numbers until US2.

---

## Phase 4: User Story 2 - Mixed Buckets Store Zero (Priority: P2)

**Goal**: A bucket with one non-empty currency keeps its sum. A bucket with two or more
non-empty currencies stores 0. The 50 USD cost and 75 EUR performance fixture keeps those
buckets and only the grand total is 0.

**Independent Test**: `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -count=1 -run 'TestCalculateRecommendationSummary|TestCalculateMockSummary'`

### Tests for User Story 2 ⚠️

- [x] T008 [US2] Extend `TestCalculateRecommendationSummaryMixedCurrency` in
  `sdk/go/pluginsdk/helpers_test.go` so the shared cost category and the shared rightsize
  action store 0 (both rows are COST and RIGHTSIZE)
- [x] T009 [P] [US2] Add a case in `sdk/go/testing/mock_plugin_test.go`
  `TestCalculateMockSummary`: two COST/RIGHTSIZE rows (100 USD and 40 EUR) and one
  PERFORMANCE/TERMINATE row (25 USD). Expect cost savings 0, performance savings 25,
  rightsize savings 0, terminate savings 25, grand total 0, currency empty, and counts
  cost 2, performance 1, rightsize 2, terminate 1
- [x] T010 [US2] Add that same 100 USD + 40 EUR cost/rightsize plus 25 USD
  performance/terminate case to `sdk/go/pluginsdk/helpers_test.go`. Confirm the mock
  case "mixed currencies clears currency field" still expects cost 50, performance 75,
  and grand total 0

### Implementation for User Story 2

- [x] T011 [US2] In `sdk/go/pluginsdk/helpers.go`, apply the bucket rule from
  `specs/054-stop-mixed-currency-sum/data-model.md`: add every impact amount in the
  bucket, including empty-currency amounts; if the bucket's distinct non-empty
  currencies number two or more, store 0; otherwise store the sum. Other buckets are
  unaffected. Do not add a per-bucket currency field
- [x] T012 [P] [US2] Apply that same bucket rule in `sdk/go/testing/mock_plugin.go`
  `CalculateMockSummary`

**Checkpoint**: COST/USD 50 plus PERFORMANCE/EUR 75 keeps those bucket sums and withholds
only the grand total. Two COST/RIGHTSIZE rows in USD and EUR withhold that category, that
action, and the grand total.

---

## Phase 5: User Story 3 - Empty Currency Is Not a Second Currency (Priority: P3)

**Goal**: An empty currency does not count as a second currency and its amount is still
included. XXX stays a normal non-empty code.

**Independent Test**: The new empty-currency and XXX cases pass in both packages, and
allocation tests are unchanged.

### Tests for User Story 3 ⚠️

- [x] T013 [P] [US3] In `sdk/go/pluginsdk/helpers_test.go`, add these cases: 100 USD
  plus 50 with an empty currency summarizes to 150 USD; 100 XXX plus 50 USD withholds
  the total (0 and empty currency); only empty currencies adding to 150 stay 150 with
  an empty currency; "usd" plus "USD" withholds because comparison is exact; one
  recommendation with no impact plus one 10 USD impact has count 2, total 10, and
  currency USD
- [x] T014 [P] [US3] Add the same cases to `TestCalculateMockSummary` in
  `sdk/go/testing/mock_plugin_test.go`

### Implementation for User Story 3

- [x] T015 [US3] If T013 or T014 fails, adjust `sdk/go/pluginsdk/helpers.go` and
  `sdk/go/testing/mock_plugin.go` so an empty string does not count as a distinct
  currency and any other non-empty code, including XXX, does. Do not map empty to USD
  in these functions

**Checkpoint**: Empty-currency amounts fold into a single-currency total. XXX plus USD
withholds. All-blank input still sums with an empty currency.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Comments, the one allowed doc fix, and formatting

- [x] T016 Update the doc comments on `CalculateRecommendationSummary` in
  `sdk/go/pluginsdk/helpers.go` and `CalculateMockSummary` in
  `sdk/go/testing/mock_plugin.go` so both describe the withheld total and the bucket rule
  in the same words. Keep the import-cycle note on the mock copy
- [x] T017 [P] Search `README.md`, `PLUGIN_DEVELOPER_GUIDE.md`, and
  `sdk/go/pluginsdk/README.md` for a sentence that says mixed recommendation currencies
  are summed and only the currency label is cleared. Correct that sentence if it exists.
  Do not rewrite the currency-converter FAQ, FOCUS SQL samples, or `ROADMAP.md`
- [x] T018 Run `gofmt` on the edited Go files and `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -count=1`
  as in `specs/054-stop-mixed-currency-sum/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup. Blocks all stories
- **User Story 1 (Phase 3)**: Depends on Foundational. No dependency on US2 or US3
- **User Story 2 (Phase 4)**: Depends on US1 because it changes the same functions
- **User Story 3 (Phase 5)**: Depends on US2 for the same reason
- **Polish (Phase 6)**: Depends on US1, US2, and US3

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational. MVP
- **User Story 2 (P2)**: After US1. Not safe to edit `helpers.go` or `mock_plugin.go` in parallel with US1
- **User Story 3 (P3)**: After US2

### Within Each User Story

- Tests are written and failing before the implementation tasks in that story
- The two function bodies may be edited in parallel with each other (`[P]`) after tests exist
- Do not extract a third copy

### Parallel Opportunities

- T003, T004, and T005 touch different test files or different cases and can run together
- T006 and T007 can run together after those tests exist
- T009 can run alongside T008 only if they are sequenced in `helpers_test.go`; T009 is the mock file and can run with T008
- T011 and T012 can run together
- T013 and T014 can run together
- T017 can run alongside T016

---

## Parallel Example: User Story 1

```text
T003 helpers_test.go mixed grand total
T004 helpers_test.go real zero in USD
T005 mock_plugin_test.go grand total 0, buckets still 50 and 75
```

After those tests fail for the right reason:

```text
T006 helpers.go page total
T007 mock_plugin.go page total
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Finish Phase 1 and Phase 2
2. Finish Phase 3
3. Stop and run `TestCalculateRecommendationSummary` plus the mock mixed-currency case
4. Continue to US2 and US3 before commit so bucket and empty-currency rules are not left half-done

### Incremental Delivery

1. US1 removes the false 150 on the grand total
2. US2 stops a mixed bucket from keeping that same false sum
3. US3 locks empty currency and XXX
4. Polish syncs the two comments and fixes a doc sentence only if one exists

---

## Out of scope (do not add tasks)

Rejected if analysis or a later pass proposes them:

- Any edit under `proto/`, `sdk/go/proto/`, or `sdk/typescript/`
- `make generate`
- FX rates, conversion helpers, or a money library
- Decimal or fixed-point amounts
- A currency field on actual-cost results
- A per-currency subtotal map on the summary
- Changing allocation's empty-as-USD rule
- Rejecting a FOCUS row whose billing currency differs from its pricing currency
- Sharing one function across `pluginsdk` and `testing`

## Notes

- [P] tasks = different files, no dependencies
- Each story's tests should fail before that story's implementation
- Commit after the slice is whole, not after every task, unless the parent asks otherwise
- Avoid: vague tasks, same-file conflicts, a new package for two copies
