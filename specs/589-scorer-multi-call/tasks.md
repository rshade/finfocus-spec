# Tasks: Multi-Call Scorer Identity and Per-Item Error Semantics

**Input**: Design documents from `specs/589-scorer-multi-call/`

**Tests**: Required (constitution principle V). Test tasks run first and are seen failing: on compile before T002,
and on behavior after it.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Confirm the baseline: `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/ -run 'Scor'` passes

## Phase 2: Foundational (proto first)

- [X] T002 In proto/finfocus/v1/scoring.proto add `provider_request_ids = 5` and `models = 6` to `ScorerInfo`, mark
  `provider_request_id = 4` `[deprecated = true]`, and update the `model`, `provider_request_id`, and
  `RecommendationScoreResult.error` comments per contracts/proto-diff.md
- [X] T003 Run `make generate`, `make buf-lint`, and `buf breaking` against main; all clean

## Phase 3: User Stories 1-3 - Rules (P1) MVP

- [X] T004 [US1] [US2] [US3] Add failing table cases to sdk/go/testing/scoring_test.go for
  `ValidateScoreRecommendationsResponse`: three request ids and two models (primary first) pass; only the deprecated
  single id passes; an empty `provider_request_ids[1]`, an empty `models[1]`, a list without `model`, and
  `models[0] != model` fail with `ErrInvalidScoreResponse` naming the field and index; a per-item error with
  `resource_type_unsupported` fails naming `results[i].error.resource_type_unsupported`; a per-item error with a code
  and message passes
- [X] T005 [US1] [US2] [US3] Implement `validateScorerInfo` (called from `validateScoreEnvelope`) and the flag check
  in `validateScoreResult` in sdk/go/testing/scoring.go, and update the validator godoc

## Phase 4: User Story 4 - Mock, conformance, TypeScript (P2)

- [X] T006 [P] [US4] Add failing tests in sdk/go/testing/scorer_mock_test.go: `WithScorerModels("a", "b")` sets
  `models` and `model = "a"`; `WithScorerProviderRequestIDs("r1", "r2")` sets the list and the deprecated single id
  to "r1"; empty arguments are ignored; the responses pass validation; and `RunScorerConformance` passes on a
  mock configured with both options (US4 AC1)
- [X] T007 [US4] Implement both options in sdk/go/testing/scorer_mock.go with godoc (set the deprecated field with a
  targeted `//nolint:staticcheck` that gives the reason)
- [X] T008 [US4] Add three broken scorers to `brokenScorers()` in sdk/go/testing/scorer_conformance_test.go (empty
  request id, mismatched primary model, unsupported-type flag on a per-item error), each listed with the scenario
  that catches it, and confirm they fail
- [X] T009 [US4] Add a case to sdk/typescript/packages/client/test/recommendation-scorer.test.ts reading
  `providerRequestIds` and `models` from a response (seen failing first); run `npm run build && npm test && npm run lint`

## Phase 5: Polish

- [X] T010 [P] Update docs/recommendation-scoring.md: the Response `scorer` row, `scorer.models` in the cache key
  table, `provider_request_ids` as a log field with the deprecated scalar, and the per-item error guidance (hosts
  surface the message; scorers never set `resource_type_unsupported`)
- [X] T011 [P] Document the two mock options and the new rules in sdk/go/testing/README.md
- [X] T012 [P] Add CLAUDE.md "Active Technologies", "Recent Changes", and a short pattern note by hand
- [X] T013 Gates: `make generate && git diff --exit-code -- sdk/`, `make buf-lint`, `buf breaking`, `make test`, `go
  test -tags=integration ./sdk/go/testing/`, `go test -run TestConformance ./sdk/go/testing/`, `make lint-go`, `make
  lint-markdown`, `make lint-yaml`, `make validate-npm`

## Dependencies & Execution Order

Phase 2 blocks everything. T004 precedes T005; T006 precedes T007; T008 needs T005 and T007. Polish comes last.

## Parallel Opportunities

T010-T012.

## Implementation Strategy

MVP is Phases 2-3 (fields and rules). Phase 4 closes test-framework and SDK parity, then docs.
