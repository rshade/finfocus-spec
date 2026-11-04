# Tasks: Spot Pricing Category and Interruption Risk on Recommendation

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md),
[data-model.md](data-model.md), [contract](contracts/recommendation-spot.md), [quickstart.md](quickstart.md)

**Tests**: Required (spec FR-009, FR-010, FR-015; Constitution Principle V). A test that names a symbol
or field that does not exist yet fails to compile until the proto change and `make generate`; that is the
expected red state.

**Format**: `[ID] [P?] [Story?]` followed by an action and exact file path. `[P]` means the task touches a
different file from every other open task and needs nothing unfinished.

**Rules to quote verbatim** (from data-model.md): `spot_interruption_risk_score` is checked in this order:
NaN or Inf is `ErrRecommendationSpotInvalidValue`; `|score| < 1e-9` is valid for every category; `score <
-1e-9` or `score > 1.0` is `ErrRecommendationSpotInvalidValue`; a non-zero score with a category other than
DYNAMIC is `ErrRecommendationSpotInvalidCategory`. `action_type` is never constrained by either field.

## Phase 1: Setup

**Goal**: Confirm the baseline and record the benchmark baseline before any change.

- [X] T001 Confirm in `proto/finfocus/v1/costsource.proto` that `Recommendation` ends at field 19
  (`secondary_reasons`) and that 20 and 21 are unused; confirm `git status` is clean on branch
  `604-recommendation-spot-category`.
- [X] T002 Add `BenchmarkValidateRecommendation` (valid recommendation, both new fields unset, `b.ReportAllocs()`)
  to `sdk/go/pluginsdk/helpers_test.go`, build its test binary from `main` in a scratch worktree, run it with
  `-benchmem -count=10`, and record ns/op and allocs/op in `specs/604-recommendation-spot-category/plan.md`
  under a new "Baseline" heading. No such benchmark exists today.

## Phase 2: Foundational Wire Contract (blocks all stories)

**Goal**: The two fields exist on the wire and in both generated SDKs before any SDK code uses them.

- [X] T003 Write the failing wire test in `sdk/go/recommendation_spot_test.go` (package `sdk_test`, modelled on
  `sdk/go/on_demand_equivalent_test.go`): `pricing_category` has field number 20 and `spot_interruption_risk_score`
  has field number 21 on the `Recommendation` descriptor; defaults are UNSPECIFIED and 0.0; a proto marshal round
  trip keeps `DYNAMIC` and `0.12`.
- [X] T004 Add `FocusPricingCategory pricing_category = 20;` and `double spot_interruption_risk_score = 21;` after
  `secondary_reasons` in `message Recommendation` in `proto/finfocus/v1/costsource.proto`, using the comment text in
  `contracts/recommendation-spot.md`. The comments MUST say: the category describes the recommended target state;
  DYNAMIC means spot; 0.0 means unknown; non-zero requires DYNAMIC; tolerance 1e-9 at zero and a strict upper bound
  of 1.0; `action_type` is not constrained; hosts annotate with risk and never reorder by it; validation is
  `pluginsdk.ValidateRecommendation`.
- [X] T005 Run `make generate` to regenerate `sdk/go/proto/finfocus/v1/costsource.pb.go` and
  `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts`; confirm with `git diff --stat` that
  only those two generated files and the proto changed, and that `buf.gen.yaml` still pins
  `buf.build/connectrpc/go:v1.21.0` (no `*.connect.go` rewrite).
- [X] T006 Run `buf lint` and `buf breaking --against '.git#branch=main'` (in a worktree, `git archive origin/main`
  into a scratch directory and run `buf breaking` against it); record a clean additive result in
  `specs/604-recommendation-spot-category/plan.md`.

**Checkpoint**: T003 passes; both generated SDKs carry the fields; no unrelated generated drift.

## Phase 3: User Story 1 - Say a recommendation targets spot capacity (Priority: P1) MVP

**Goal**: A plugin can build, validate, and return a spot recommendation, and a host reads both values back.

**Independent Test**: Build `NewSpotRecommendation("rec-1", 0.12)`, validate it with `ValidateRecommendation`,
return it through the bufconn harness, and read DYNAMIC and 0.12 back.

### Tests (write first, expect failure)

- [X] T007 [P] [US1] Write failing helper tests in `sdk/go/testing/recommendation_spot_test.go`: `NewSpotRecommendation`
  returns category DYNAMIC, a `MODIFY` action type, the given risk, and passes `ValidateRecommendationSpot`;
  `ValidateRecommendationSpot(DYNAMIC, 0.0)` and `(DYNAMIC, 0.12)` return nil; the valid path reports 0 allocs/op via
  `testing.AllocsPerRun`.
- [X] T008 [P] [US1] Write a failing bufconn round-trip test in `sdk/go/testing/recommendation_spot_test.go` that sets
  `MockPlugin.RecommendationsConfig.Recommendations` to `NewSpotRecommendation("rec-1", 0.12)`, calls
  `GetRecommendations` through `NewTestHarness`, and asserts both values arrive unchanged and that
  `Impact.EstimatedSavings` is untouched (savings stay in impact, FR-008).
- [X] T009 [P] [US1] Write a failing vitest in `sdk/typescript/packages/client/test/recommendation-spot.test.ts`
  that creates a `Recommendation` with `pricingCategory` DYNAMIC and `spotInterruptionRiskScore` 0.12, round trips it
  through `toJson`/`fromJson` (never `JSON.stringify`), and asserts the defaults on an empty message. Run it with
  `npx vitest run` (the client `tsconfig.json` excludes `test/`, so `tsc` cannot show the failure).
- [X] T010 [P] [US1] Write failing `ValidateRecommendation` cases in `sdk/go/pluginsdk/helpers_test.go`: a valid base
  recommendation with `(DYNAMIC, 0.12)` and `(DYNAMIC, 0.0)` returns nil.

### Implementation

- [X] T011 [US1] Create `sdk/go/testing/recommendation_spot.go` with the Apache 2.0 header: unexported
  `recommendationSpotEpsilon = 1e-9`, sentinels `ErrRecommendationSpotInvalidValue` and
  `ErrRecommendationSpotInvalidCategory` (godoc on each), `ValidateRecommendationSpot(category
  pbc.FocusPricingCategory, score float64) error` implementing the ordered rule quoted at the top of this file
  (wrap as `recommendation.spot_interruption_risk_score: <sentinel>: got ...`; no allocation on the valid path), and
  `NewSpotRecommendation(id string, risk float64) *pbc.Recommendation` (valid category COST, `MODIFY`, resource,
  impact, priority, DYNAMIC, given risk).
- [X] T012 [US1] In `sdk/go/pluginsdk/validation.go` add aliases `ErrRecommendationSpotInvalidValue =
  plugintesting.ErrRecommendationSpotInvalidValue` and `ErrRecommendationSpotInvalidCategory =
  plugintesting.ErrRecommendationSpotInvalidCategory` next to the other sentinel aliases.
- [X] T013 [US1] In `ValidateRecommendation` in `sdk/go/pluginsdk/helpers.go`, after the confidence-score check and
  before the metadata check, call an unexported `validateRecommendationSpot(rec)` helper only when
  `rec.GetPricingCategory()` is not `FOCUS_PRICING_CATEGORY_UNSPECIFIED` or `rec.GetSpotInterruptionRiskScore()` is not
  `0` (NaN is not equal to zero, so it still enters). The helper holds the `fmt.Errorf` wrap, never the public
  validator (research R4: an inline wrap behind a never-taken guard slowed a validator by 33% in spec 557).

**Checkpoint**: T007, T008, T009, T010 pass. User Story 1 works on its own.

## Phase 4: User Story 2 - Reject contradictory or malformed risk (Priority: P1)

**Goal**: Every invalid pair fails with the right named error; legacy and zero-risk pairs stay valid.

**Independent Test**: The full state table in `data-model.md` passes through both `ValidateRecommendationSpot` and
`ValidateRecommendation`.

### Tests (write first, expect failure)

- [X] T014 [P] [US2] Add a table-driven `TestValidateRecommendationSpot` in `sdk/go/testing/recommendation_spot_test.go`
  covering every cell of the state table (UNSPECIFIED, STANDARD, COMMITTED, DYNAMIC against 0.0, 0.5, 1.0, NaN, +Inf,
  -Inf, -0.1, 1.0000001, 1e-10, -1e-10, -1e-9 boundary): assert `errors.Is` against the right sentinel and that the
  message contains `recommendation.spot_interruption_risk_score`. One table, no duplicate rows.
- [X] T015 [P] [US2] Add a drift test in `sdk/go/pluginsdk/helpers_test.go` asserting the `testing` package's
  `recommendationSpotEpsilon` behaves like `spotRiskEpsilon`: probe `(DYNAMIC, 9e-10)` valid, `(STANDARD, 9e-10)` valid,
  `(STANDARD, 1.1e-9)` invalid category, `(DYNAMIC, -1.1e-9)` invalid value. Use the exported rule, not the unexported const.
- [X] T016 [P] [US2] Add `ValidateRecommendation` cases in `sdk/go/pluginsdk/helpers_test.go` for each invalid cell
  through the public validator (wrapped error still satisfies `errors.Is` for the pluginsdk alias), plus: nil
  recommendation keeps `recommendation cannot be nil`; a default recommendation with the pair unset stays valid; an
  invalid pair is reported only after the earlier required-field checks pass.
- [X] T017 [P] [US2] Add `TestScoreRequestIgnoresSpotPair` in `sdk/go/testing/recommendation_spot_test.go`: a
  `ScoreRecommendationsRequest` whose recommendation has `(STANDARD, 0.5)` passes `ValidateScoreRecommendationsRequest`
  (FR-016, clarification Q1).

### Implementation

- [X] T018 [US2] Make T014 to T017 pass; if any fails, fix `ValidateRecommendationSpot` or the helper from T011 and
  T013 (no new production file expected). Confirm `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/` passes.

**Checkpoint**: Rules complete and pinned for both packages; scorer untouched.

## Phase 5: User Story 3 - Verify plugins through conformance (Priority: P2)

**Goal**: The shared Standard conformance suite checks returned recommendations.

**Independent Test**: A mock returning valid spot recommendations and a mock that never sets the fields both pass;
one returning a non-zero risk without DYNAMIC fails with the rule named.

### Tests (write first, expect failure)

- [X] T019 [P] [US3] Add `TestRPCCorrectness_GetRecommendationsSpot` in `sdk/go/testing/rpc_correctness_test.go`
  (modelled on `TestRPCCorrectness_GetActualCostWithResource`): look the test up in `RPCCorrectnessTests()` by name,
  assert `MinLevel` is Standard and `Category` is `CategoryRPCCorrectness`, then run it against four plugins: default
  mock (passes), mock with `NewSpotRecommendation("r", 0.2)` (passes), mock with an empty recommendation list (passes),
  and a wrapper plugin returning `(STANDARD, 0.5)` (fails; `result.Error` or `Details` names
  `spot_interruption_risk_score`). Also assert it is skipped when the plugin does not declare
  `PLUGIN_CAPABILITY_RECOMMENDATIONS`, following how `withCapability` is used for the other entries.

### Implementation

- [X] T020 [US3] In `sdk/go/testing/rpc_correctness.go` add `testGetRecommendationsSpot(harness *TestHarness) TestResult`
  (calls `harness.Client().GetRecommendations` with an empty `GetRecommendationsRequest`, applies
  `ValidateRecommendationSpot` to every returned recommendation, reports the first failure with the recommendation id,
  treats `Unimplemented` like `testGetBudgetsRPC` does) and register `RPCCorrectness_GetRecommendationsSpot`
  (`Category: CategoryRPCCorrectness`, `MinLevel: ConformanceLevelStandard`, with its own capability gate instead of
  `withCapability`, which would fail plugins that declare no recommendations; see research R5) in
  `RPCCorrectnessTests()`. Use a `MethodGetRecommendations` constant if one exists, otherwise the existing
  method-name constant style.
- [X] T021 [US3] Run `go test ./sdk/go/testing/` and check any test that counts the registered RPC correctness tests or
  the Standard-level test total; update those counts and nothing else.

**Checkpoint**: A plugin author gets the new check at Standard level; plugins that never set the fields pass.

## Phase 6: User Story 4 - Adopt the change with clear guidance (Priority: P2)

**Goal**: Maintainers find the fields, rules, and upgrade guidance without reading code.

**Independent Test**: Follow only the guide section to set the fields in a plugin and read them in a host; every Go
snippet compiles.

- [X] T022 [P] [US4] Add a "Spot recommendations" subsection to the recommendation section of
  `PLUGIN_DEVELOPER_GUIDE.md`: the two fields, DYNAMIC requirement, zero-means-unknown, tolerance, that `action_type`
  stays `MODIFY` or `MIGRATE`, that savings stay in `impact`, that hosts annotate and never reorder by risk, plus a short
  "Upgrading" paragraph (additive, minor release, old plugins and hosts unaffected, scorer unaffected).
- [X] T023 [P] [US4] Add a recommendation validation paragraph with a compiling snippet using
  `pluginsdk.ValidateRecommendation` and `errors.Is(err, pluginsdk.ErrRecommendationSpotInvalidCategory)` to
  `sdk/go/pluginsdk/README.md`, next to the `RecommendationsProvider` section.
- [X] T024 [P] [US4] Document `ValidateRecommendationSpot`, `NewSpotRecommendation`, and the new conformance check in
  `sdk/go/testing/README.md`, and add the check to the Standard-level list in `docs/conformance.md` if that file lists
  checks by name.
- [X] T025 [US4] Compile-check every Go snippet added in T022 and T023 (extract to a scratch `main` package under the
  scratchpad directory and run `go vet`), and run `npx markdownlint-cli2` on the three edited markdown files.

**Checkpoint**: Docs match exported symbols exactly (Constitution XIV).

## Phase 7: Polish and Cross-Cutting Concerns

- [X] T026 Re-run the T002 benchmark on the branch against the recorded `main` baseline, using prebuilt binaries and
  `-count=10`; confirm no measurable change and 0 added allocs/op with the fields unset (FR-015, SC-006); if the
  unset path regressed, check the frame size with `go build -gcflags=-S` and move the wrap into the helper; record
  the result in `specs/604-recommendation-spot-category/plan.md`.
- [X] T027 Add a "Recommendation Spot Fields Pattern (604-recommendation-spot-category)" entry and a Recent Changes
  bullet to `CLAUDE.md` by hand (do not run `.specify/scripts/bash/update-agent-context.sh`, it mangles wrapped
  bullets); cover the shared-rule location, the two sentinels, the guarded call site, the Standard conformance check,
  the scorer exemption, and that `SpecVersion` is bumped by release-please. Do not touch `CHANGELOG.md`.
- [X] T028 Run the full gate: `golangci-lint run ./...`, `make test`, `make lint-markdown`, `buf lint`, and
  `npx vitest run` in `sdk/typescript/packages/client`. `TestUsageSourceNotRegistered/connect` and
  `TestAllocatorServe_NotRegisteredWithoutProvider/connect` flake under load; rerun them in isolation before blaming
  this change. Fix any finding; the lint baseline is 0 issues.
- [X] T029 Write `PR_MESSAGE.md` with the conventional header `feat(proto): add spot category and interruption
  risk to Recommendation` (header at most 72 characters, body lines at most 100, no `specs` scope, no attribution
  trailers) and validate with `cat PR_MESSAGE.md | npx commitlint`. Do not commit unless asked.
- [X] T030 Walk `quickstart.md` scenarios 1 to 7 and tick each against real output; mark `Status` in `spec.md` as
  Implemented only after all pass.

## Dependencies and Execution Order

- Phase 1 (T001, T002) first. T002 needs only the test file, so it does not depend on T001.
- Phase 2 blocks everything: T003 before T004, T004 before T005, T005 before T006.
- US1 and US2 share `recommendation_spot.go` (T011), so US2 implementation waits for US1 implementation. Their
  test tasks can be written in parallel.
- US3 needs T011 (rule) and T020 needs the mock path from T008's config route; it does not need US2 tests.
- US4 documents symbols from T011 and T020, so it follows them.
- Polish is last; T026 needs T013 in place.

```text
Setup -> Foundational -> US1 -> US2 -> Polish
                           \-> US3 -> Polish
                           \-> US4 -> Polish   (after US1 and US3 symbols exist)
```

## Parallel Opportunities

- After T006: T007, T008, T009, T010 touch four different files and can be written together.
- After T013: T014, T015, T016, T017 are different test files or distinct additions and can run together; US3's T019
  can start in parallel with them.
- T022, T023, T024 touch three different documents.

## Implementation Strategy

1. **MVP**: Phases 1 to 3 give a valid, round-trippable spot recommendation with the happy-path rule. Stop and demo.
2. Add US2 so invalid data is rejected, then US3 so ecosystem plugins get checked, then US4 docs.
3. Phase 7 gates the release. The `feat(proto):` commit triggers release-please to propose the minor release and
   `SpecVersion` bump; cutting that release PR is the final acceptance item from the issue and happens after merge.
