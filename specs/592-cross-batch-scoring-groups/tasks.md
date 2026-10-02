---

description: "Task list for cross-batch scoring sessions"
---

# Tasks: Cross-Batch Scoring Sessions

**Input**: Design documents in `specs/592-cross-batch-scoring-groups/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/session-contract.md

Tests come before implementation inside each story. Worktree-relative paths below.

## Phase 1: Setup

- [x] T001 Confirm baseline: run `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/` and `cd
      sdk/typescript/packages/client && npx vitest run recommendation-scorer` on a clean tree

## Phase 2: Foundational (proto first, blocks all stories)

- [x] T002 Add `string session_id = 4;` to `ScoreRecommendationsRequest` and `string session_id = 5;` to
      `ScoreRecommendationsResponse` in `proto/finfocus/v1/scoring.proto`, with comments stating: empty means no
      session; 1 to 128 printable ASCII characters (0x21-0x7E); one host operation, one pseudonymization key; echo
      semantics. Update the `duplicate_group_id` and `IDENTIFIER_MODE_PSEUDONYMIZED` comments to replace "within one
      response" and "within one request" with the session rules (FR-001, FR-002, FR-012)
- [x] T003 Run `make generate`, then `git status` to confirm only scoring bindings changed (Go, Connect, TypeScript
      `scoring_pb.ts`); restore unrelated regenerated files with `git checkout` (depends on T002)
- [x] T004 Run `buf breaking` against `origin/main` (archive `origin/main` proto into a scratch dir if
      `.git#branch=main` fails) and `make buf-lint`; expect clean (FR-013, SC-003)

**Checkpoint**: fields exist in all bindings.

## Phase 3: User Story 4 - Validation (P1)

**Goal**: shared validators know sessions. **Independent test**: validator table tests.

- [x] T005 [P] [US4] Add request tests in `sdk/go/testing/scoring_test.go`: `session_id` of 128 printable ASCII
      characters passes; 129 characters fails; a space, control character, and non-ASCII byte fail; empty passes; each
      failure wraps `ErrInvalidScoreRequest` and has `codes.InvalidArgument` (FR-007)
- [x] T006 [P] [US4] Add response tests in `sdk/go/testing/scoring_test.go`: with request session set, a lone-member
      `duplicate_group_id` passes; with no session the same response still fails with the "used by only one
      recommendation" error; echo equal passes; empty echo passes; echo differing from request fails with
      `ErrInvalidScoreResponse`; a non-empty echo for a session-less request fails (FR-004, FR-008)
- [x] T007 [US4] Implement `session_id` grammar check in `ValidateScoreRecommendationsRequest` in
      `sdk/go/testing/scoring.go`, with constant `maxSessionIDLength = 128`; zero allocations (FR-007)
- [x] T008 [US4] Implement echo check and session-conditional two-member rule in
      `ValidateScoreRecommendationsResponse` in `sdk/go/testing/scoring.go`; update the doc comment (FR-004, FR-008)
- [x] T009 [P] [US4] Update delegating doc comments in `sdk/go/pluginsdk/scorer.go` and add a pass-through test in
      `sdk/go/pluginsdk/scorer_serve_test.go` that an invalid `session_id` returns `InvalidArgument` on gRPC and
      Connect with no `rpc error:` prefix
- [x] T010 [US4] Add an allocation guard test for session-less valid request and response validation in
      `sdk/go/testing/scoring_test.go` (SC-004)

## Phase 4: User Story 1 - Grouping across batches (P1)

**Goal**: mock and conformance prove ids match across batches. **Independent test**: mock tests.

- [x] T011 [P] [US1] Add mock tests in `sdk/go/testing/scorer_mock_test.go`: a duplicate pair (same `resource.id` and
      action type) split into two single-item requests with one `session_id` gets equal non-empty ids; the same split
      under two different sessions gets different ids; a session-less request keeps the `dup-N` ids and clears a
      singleton; response echoes `session_id`; `IDENTIFIER_MODE_OMITTED` yields no ids even with a session (FR-003,
      FR-009, SC-001)
- [x] T012 [US1] Implement session-aware grouping in `sdk/go/testing/scorer_mock.go`: when `session_id` is set, id is
      `"s-" + first 16 hex characters of sha256(session_id + "\x00" + resource.id + "\x00" + action_type)`, applied to
      every scored member including singletons; echo `session_id` in the response (depends on T011)
- [x] T013 [P] [US1] Add conformance scenarios in `sdk/go/testing/scorer_conformance.go`:
      `scorerCheckSessionAcrossBatches` (duplicate pair split across two calls of one session expects equal non-empty
      ids; the scenario function returns nil, which the runner reports as a pass, when the scorer does not echo the
      session, does not support `SCORE_SIGNAL_DUPLICATE_GROUP`, or does not group the pair in a single call, since the
      scorer owns its duplicate rule), `scorerCheckSessionIsolation` (two sessions produce different ids for the same
      pair), `scorerCheckSessionEcho` (echo equals request or is empty) (FR-010)
- [x] T014 [US1] Add scenario entries to the list near `sdk/go/testing/scorer_conformance.go:183` (names
      `session_across_batches`, `session_isolation`, `session_echo`), expose them in `sdk/go/testing/export_test.go`
      as the existing scenarios are, and add broken-scorer tests in `sdk/go/testing/scorer_conformance_test.go`: a
      scorer that changes ids per call fails across-batches, a scorer that ignores the session id and does not echo
      passes, a wrong echo fails
- [x] T015 [US1] Update `sdk/go/testing/README.md` scorer section with the session options

## Phase 5: User Story 2 - Stable tokens (P1)

**Goal**: contract states the pseudonymization duty. **Independent test**: docs lint plus review.

- [x] T016 [US2] In `docs/recommendation-scoring.md` update the `duplicate_group_id` rules (lines near "meaningful
      only within one response") and the `IDENTIFIER_MODE_PSEUDONYMIZED` row ("identical for the same value within one
      request") to the session rules, and add a "Sessions and Batches" section copying
      [contracts/session-contract.md](contracts/session-contract.md) as normative text (FR-005, FR-012)

## Phase 6: User Story 3 - Backward compatibility (P1)

- [x] T017 [US3] Run the existing scorer suites unchanged (`go test ./sdk/go/testing/ ./sdk/go/pluginsdk/ -run
      'Score|Scorer'`) and confirm no existing test needed editing; record the result in the PR message (SC-002)

## Phase 7: User Story 6 - Cached items (P2)

- [x] T018 [US6] In `docs/recommendation-scoring.md` add the cached-item paragraph (absent items are never grouped;
      host may send cached items and discard other scores) and extend the "Caching Scores" list so
      `duplicate_group_id` from a session is still never cached per recommendation (FR-006, FR-012)

## Phase 8: TypeScript parity (serves US1, US4)

- [x] T019 [P] Add tests in `sdk/typescript/packages/client/test/recommendation-scorer.test.ts` that `sessionId`
      round-trips on request and response through the client
- [x] T020 Update `sdk/typescript/packages/client/src/clients/recommendation-scorer.ts` (and `src/index.ts` only if a
      type export is needed) so request and response types expose `sessionId`; run `npm run build && npx vitest run
      recommendation-scorer` in `sdk/typescript/packages/client`

## Phase 9: Polish

- [x] T021 Add a "Recent Changes" and Active Technologies entry for 592-cross-batch-scoring-groups to `CLAUDE.md` and
      note the session rule under the Recommendation Scorer pattern; keep it to a few lines to ease the merge with
      spec 591
- [ ] T022 Run gates: `make generate && git diff --exit-code -- sdk/`, `make buf-lint`, `buf breaking`, `make test`,
      `go test -v -tags=integration ./sdk/go/testing/`, `go test -v -run TestConformance ./sdk/go/testing/`, `make
      lint-go`, `make lint-markdown`, `make lint-yaml`, `make validate-npm`, TypeScript build and tests, and `go test
      -bench=. -benchmem ./sdk/go/testing/ -run '^$' -bench Score` against `main` for the validators
- [ ] T023 Write `PR_MESSAGE.md` (conventional commit, `Closes #574`, conflict notes for #573) and validate with `cat
      PR_MESSAGE.md | npx commitlint`

## Dependencies

T002 -> T003 -> T004 -> all stories. Phases 3 and 4 touch `scoring.go` and `scorer_mock.go` separately and can overlap
after T004; T013 needs T012. Phases 5 and 7 edit the same docs file: do T016 before T018. T019 needs T003. T022 and
T023 last.

## Parallel Example

After T004: T005, T006, T011, T013, T019 are all test-first tasks in different files.

## Implementation Strategy

MVP is Phases 2 to 4 (wire, validators, mock, conformance). Docs and TypeScript follow in the same PR for parity.
