# Tasks: Omitted Fields on Scoring Requests

**Input**: `specs/593-omitted-fields/` (spec.md, plan.md, data-model.md, contracts/)
**Tests**: Required (constitution V, test first).

## Phase 1: Setup

- [x] T001 Confirm request field 4 is free on this base and that `../finfocus-spec-574` uses request field 4 and
      response field 5 (read-only); record the result in `specs/593-omitted-fields/research.md`

## Phase 2: Foundational (proto)

- [x] T002 Add `repeated string omitted_fields = 5` with the FR-002 doc comment to `ScoreRecommendationsRequest` in
      `proto/finfocus/v1/scoring.proto`
- [x] T003 Run `make generate`; confirm Go, Connect, and TypeScript bindings changed only for this field (restore
      unrelated regenerated files with `git checkout`)
- [x] T004 Run `buf breaking` against `origin/main` archive and `make buf-lint`

## Phase 3: US1 Signal and US2 Validation (P1)

**Goal**: the request carries the list and the shared validator enforces it.

- [x] T005 [P] [US2] Write failing table tests in `sdk/go/testing/scoring_test.go`: valid (`resource.tags`,
      `metadata`, nested `kubernetes.cluster_id`, whole-message `resource`, oneof name `action_detail`), empty list,
      empty entry, unknown path, path through a map/scalar leaf, uppercase or bad segment, duplicate, over 128 bytes,
      over 64 entries; each invalid case wraps `ErrInvalidScoreRequest` with `InvalidArgument` and names
      `omitted_fields[i]`
- [x] T006 [P] [US2] Write benchmarks in `sdk/go/testing/scoring_test.go` for `ValidateScoreRecommendationsRequest`
      with and without `omitted_fields`; the empty-list path must match `main` in allocs/op
- [x] T007 [US2] Implement `validateOmittedFields` (accepting the `action_detail` oneof name) in
      `sdk/go/testing/scoring.go` with constants `maxOmittedFields = 64`, `maxOmittedFieldBytes = 128`, resolving
      paths against the `pbc.Recommendation` descriptor; call it from `ValidateScoreRecommendationsRequest` guarded by
      `len() > 0` (keep any `fmt.Errorf` wrap out of the public function; check frame size)
- [x] T008 [US1] Add a `pluginsdk` test in `sdk/go/pluginsdk/scorer_serve_test.go` showing `omitted_fields` reaches
      the scorer over gRPC and Connect and a malformed list returns `InvalidArgument`

## Phase 4: US3 Mock and conformance (P2)

- [x] T009 [P] [US3] Write failing mock tests in `sdk/go/testing/scorer_mock_test.go`: with `resource.utilization` and
      `reasoning` cleared, `insufficient_evidence` is the thin value when not omitted and the enough value when listed
- [x] T010 [US3] Make `mockInsufficientEvidence` in `sdk/go/testing/scorer_mock.go` honor `omitted_fields` for
      `resource.utilization` and `reasoning`
- [x] T011 [P] [US3] Add scenarios `omitted_fields_accepted` and `omitted_fields_rejected` (empty entry, unknown path,
      duplicate) to `scorerScenarios` in `sdk/go/testing/scorer_conformance.go`, with no score-value comparison
- [x] T012 [US3] Add broken-scorer tests in `sdk/go/testing/scorer_conformance_test.go` (a scorer that accepts a
      malformed list fails `omitted_fields_rejected`; one that errors on a valid list fails `omitted_fields_accepted`)

## Phase 5: US4 Docs and TypeScript (P2)

- [x] T013 [P] [US4] Update `docs/recommendation-scoring.md`: request table row and rule, replace "the scorer cannot
      tell a removed field from an empty one", and add `omitted_fields` to the caching key table
- [x] T014 [P] [US4] Update `sdk/go/pluginsdk/README.md` and `sdk/go/testing/README.md` scorer sections
- [x] T015 [P] [US4] Add a field round-trip test to
      `sdk/typescript/packages/client/test/recommendation-scorer.test.ts`; run `npm run build` and `npx vitest run`

## Phase 6: Gates and delivery

- [x] T016 Run `make test`, `go test -v -tags=integration ./sdk/go/testing/`, `go test -v -run TestConformance
      ./sdk/go/testing/`; on any failure capture the test name with `-v`
- [x] T017 Run `make lint-go`, `make lint-markdown`, `make lint-yaml`, `make validate-npm`, and the validator
      benchmark A/B against `main`
- [x] T018 Update `CLAUDE.md` Recent Changes and add a short "Scorer Omitted Fields Pattern" note
- [x] T019 Write `PR_MESSAGE.md` (Closes #576, no trailers) and validate with `cat PR_MESSAGE.md | npx commitlint`
- [x] T020 List the exact `git add`, `git commit`, `git push`, `gh pr create` commands for the user (not run here)

## Dependencies

T001 -> T002 -> T003 -> T004 -> Phase 3 -> Phase 4 -> Phase 5 -> Phase 6. T005/T006 before T007; T009 before T010.
