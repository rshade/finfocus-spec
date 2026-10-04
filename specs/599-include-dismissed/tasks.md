# Tasks: Include Dismissed Recommendations

**Input**: Design documents from `specs/599-include-dismissed/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/proto.md, quickstart.md

**Tests**: Required. Helper and mock tests are written before the filter they cover. The field-number
test compiles only after `make generate`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: The user story the task belongs to (US1-US3)

## Phase 1: Setup

- [x] T001 Confirm `make generate` is clean and `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/` passes
  before the proto edit

## Phase 2: Foundational (proto first)

- [x] T002 Add `bool include_dismissed = 8` to `GetRecommendationsRequest` in
  `proto/finfocus/v1/costsource.proto` using the comment in `contracts/proto.md`
- [x] T003 Run `make generate` and keep generated Go and TypeScript diffs that belong to this field
- [x] T004 Run `make buf-lint` and `buf breaking` against `main`; both must report nothing

## Phase 3: User Story 1 - Plugin returns its dismissed recommendations (P1)

- [x] T005 [US1] Write mock tests: dismissed IDs are omitted by default and returned when
  `include_dismissed` is true, in `sdk/go/testing/recommendations_visibility_test.go`
- [x] T006 [US1] Add `DismissedIDs` to `RecommendationsConfig` and apply the filter before pagination
  in `sdk/go/testing/mock_plugin.go`

## Phase 4: User Story 2 - Exclusion IDs win (P1)

- [x] T007 [US2] Extend the mock tests: an excluded ID is omitted when `include_dismissed` is true
  and when it is false
- [x] T008 [US2] Apply `excluded_recommendation_ids` after the dismissal filter in the mock

## Phase 5: User Story 3 - Helper, bindings, and docs (P2)

- [x] T009 [P] [US3] Write `ApplyRecommendationVisibility` tests, including the empty-list identity and
  field number 8, in `sdk/go/pluginsdk/recommendation_visibility_test.go`
- [x] T010 [US3] Implement `ApplyRecommendationVisibility` in `sdk/go/pluginsdk/helpers.go`
- [x] T011 [P] [US3] Round-trip `includeDismissed` in
  `sdk/typescript/packages/client/test/integration.test.ts`
- [x] T012 [US3] Document the field in `PLUGIN_DEVELOPER_GUIDE.md` next to
  `excluded_recommendation_ids`

## Phase 6: Polish

- [x] T013 Run `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/` and markdownlint on the edited guide
  and this spec directory

## Dependencies

T002 blocks T003. T003 blocks tests that name the generated getter. T005 precedes T006. T007 precedes
T008. T009 precedes T010. T011 and T012 need T003 only.
