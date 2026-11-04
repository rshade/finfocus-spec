# Tasks: Plugin SDK Decline Log Levels

**Input**: Design documents from `specs/605-sdk-decline-log-level/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/log-levels.md

**Tests**: Required. Constitution V (test-first): every test task runs and FAILS before its implementation task.

**Format**: `- [ ] T### [P?] [US#] description with file path`. All paths are under `sdk/go/pluginsdk/` unless
shown. New Go files start with the Apache 2.0 header used by `handler_status_test.go`.

## Phase 1: Setup

- [x] T001 Confirm the baseline: run `go test ./sdk/go/pluginsdk/` and `golangci-lint run ./sdk/go/pluginsdk/...`
      in the worktree; both must pass (lint baseline is 0 issues). Read `handlerStatus` and
      `logHandlerFallback` in `sdk/go/pluginsdk/sdk.go` and `TracingUnaryServerInterceptorWithLogger` in
      `sdk/go/pluginsdk/logging.go`.

## Phase 2: Foundational (blocks every story)

- [x] T002 Write `sdk/go/pluginsdk/log_level_test.go` (package `pluginsdk`, `//nolint:testpackage` like
      `handler_status_test.go`): `TestHandlerLogLevel` table over all 16 non-OK gRPC codes
      (`codes.Canceled` through `codes.Unauthenticated`), wrapped with `fmt.Errorf("%w")` as well as bare, plus a
      plain `errors.New`. Expect `zerolog.DebugLevel` for `InvalidArgument`, `Unimplemented`, `NotFound`,
      `FailedPrecondition`; `zerolog.ErrorLevel` for every other code and for the plain error. Add
      `BenchmarkHandlerLogLevel` and `testing.AllocsPerRun` assertion of 0 allocs (FR-010). It must not compile
      yet.
- [x] T003 Create `sdk/go/pluginsdk/log_level.go` with `declined(code codes.Code) bool` (switch on the four
      decline codes) and `handlerLogLevel(err error) zerolog.Level` (`DebugLevel` when
      `declined(status.Code(err))`, else `ErrorLevel`), with doc comments. T002 now passes.

**Checkpoint**: classifier proven; stories can proceed.

## Phase 3: User Story 1 - Quiet logs for contract-allowed declines (P1)

**Goal**: declines produce no ERROR line from the interceptor or any wrapper.
**Independent test**: capture logs, return each decline code, assert no `"level":"error"` entry.

### Tests first (must FAIL before T006 and T007)

- [x] T004 [P] [US1] In `sdk/go/pluginsdk/logging_test.go` add `TestTracingUnaryServerInterceptor_DeclineLevel`:
      drive `TracingUnaryServerInterceptorWithLogger` with a `zerolog` logger writing JSON to a `bytes.Buffer`,
      a handler returning `status.Error(code, "x")` for each of the four decline codes, and assert the buffer
      holds no `"level":"error"` line and (at debug level) one `"message":"rpc failed"` line carrying
      `trace_id`, `operation`, and `error` (covers US3 too). Assert the returned error is the handler's error.
- [x] T005 [P] [US1] In `sdk/go/pluginsdk/handler_status_test.go` add `TestWrapperDeclineLevel`: reuse
      `errorHandlerPlugin` and the helper at line ~231 that builds a `Server` with a chosen error; with the
      server logger writing JSON to a buffer, call each of the eight wrapped RPCs (`Supports`, `DryRun`,
      `GetRecommendations`, `GetBudgets`, `DismissRecommendation`, `BatchCost`, `ResolveResourceTypes`,
      `GetPluginInfo`) with `InvalidArgument`, `NotFound`, `FailedPrecondition`, and `Unimplemented`, and
      assert no `"level":"error"` entry. Assert the returned status code and message equal the pre-change
      values (FR-006): for non-`Unimplemented` codes the handler's own status.

### Implementation

- [x] T006 [US1] In `sdk/go/pluginsdk/logging.go` replace `logger.Error()` in the `rpc failed` branch with
      `logger.WithLevel(handlerLogLevel(err))`, keeping the `traceID != ""` guard, fields, and message.
- [x] T007 [US1] In `sdk/go/pluginsdk/sdk.go` replace `s.logger.Error()` with
      `s.logger.WithLevel(handlerLogLevel(clientErr))` at the eight handler-error sites: GetPluginInfo
      (~578), DryRun (~780), Supports (~882), GetRecommendations (~968), GetBudgets (~1047),
      DismissRecommendation (~1107), BatchCost (~1179), ResolveResourceTypes (~1239). Keep fields and messages.
      Do not touch the nil-response or other `Error()` sites.
- [x] T008 [US1] Run `go test ./sdk/go/pluginsdk/ -run 'DeclineLevel|HandlerLogLevel'`; T004 and T005 pass.

**Checkpoint**: US1 delivered.

## Phase 4: User Story 2 - Real faults stay loud (P1)

**Goal**: faults and non-status errors still log at ERROR.
**Independent test**: same harness with fault codes.

- [x] T009 [P] [US2] In `sdk/go/pluginsdk/logging_test.go` add `TestTracingUnaryServerInterceptor_FaultLevel`:
      table of `Internal`, `Unknown`, `Unavailable`, `DataLoss`, `PermissionDenied`, `DeadlineExceeded`, a
      wrapped `Internal`, and `errors.New("boom")`; assert exactly one `"level":"error"` `rpc failed` line each.
- [x] T010 [P] [US2] In `sdk/go/pluginsdk/handler_status_test.go` add `TestWrapperFaultLevel`: the eight
      wrapped RPCs with `Internal`, `Unknown`, `Unavailable`, `PermissionDenied`, and a plain error; assert an
      ERROR entry from the wrapper, and that a plain error or `Unknown` reaches the host as `Internal` with the
      generic message.
- [x] T011 [US2] Run both tests; they pass with the T006/T007 code (no further change expected). If one
      fails, fix the classifier input, not the test.

**Checkpoint**: US1 and US2 pass together.

## Phase 5: User Story 3 - Decline traceable at debug (P2)

- [x] T012 [US3] Confirm T004 asserts the debug entry fields (`trace_id`, `operation`, `error`). Add the same
      field assertion to `TestWrapperDeclineLevel` for one wrapper entry (fields `provider`, `resource_type`
      for `Supports`). No production change.

## Phase 6: Polish and cross-cutting

- [x] T013 [P] Add a "Log levels for handler errors" note to `sdk/go/pluginsdk/README.md` (table from
      `contracts/log-levels.md`, and the sentence that plugins must rebuild on the new SDK to pick it up).
- [x] T014 [P] Add the same fact to the "Common Issues" or "Project-Specific Patterns" part of `CLAUDE.md`
      (one short bullet: declines log at debug, `handlerLogLevel` is the one rule, new wrapped RPCs must use it).
- [x] T015 Check that `log_level.go` and `log_level_test.go` carry the Apache 2.0 header (copy it from
      `handler_status_test.go`), then run `goimports -w` on changed Go files, then
      `golangci-lint run ./sdk/go/pluginsdk/...`; fix findings
      in your change only (baseline is 0; do not edit `.golangci.yml`).
- [x] T016 Run `make test`, `go test -v -tags=integration ./sdk/go/testing/`, and
      `go test -v -run TestConformance ./sdk/go/testing/`. Known flake: `TestUsageSourceNotRegistered/connect`
      and `TestAllocatorServe_NotRegisteredWithoutProvider/connect`; re-run in isolation before blaming the change.
- [x] T017 Run `go test -bench 'HandlerLogLevel|ValidateGetProjectedCostResponse' -benchmem ./sdk/go/pluginsdk/`;
      the classifier is 0 allocs/op and untouched validators did not move.
- [x] T018 Run `make buf-lint`, `make lint-markdown`, and `make generate && git diff --exit-code -- sdk/`
      (expect no generated change: no proto edit).
- [x] T019 Write `PR_MESSAGE.md` (gitignored): `fix(pluginsdk): log handled declines at debug instead of ERROR`,
      body naming the four codes, the eight wrappers, "plugins must rebuild on the new SDK to pick up this
      change", and `Closes #664`. Validate with `cat PR_MESSAGE.md | npx commitlint`. No session trailers.

## Dependencies and Order

- T001 → T002 → T003 → (T004, T005 in parallel) → T006 → T007 → T008.
- T009 and T010 can be written alongside T004/T005 (same files, so edit sequentially).
- T012 after T008. Polish after all stories. T013 and T014 are parallel.

## Implementation Strategy

MVP is Phases 1-3 (US1). US2 only adds guard tests because the classifier already keeps faults at ERROR.
Each test task is run red before its implementation task.

## Phase 7: Convergence

- [x] T020 Add the missing tenth call site: add `GetRecommendationHistory` to `wrappedRPCs()` in
      `sdk/go/pluginsdk/handler_status_test.go` (its provider interface is `RecommendationHistoryProvider`;
      give `errorHandlerPlugin` that method returning `p.err`), run `TestWrapperDeclineLevel` red, then change
      `s.logger.Error()` at `sdk/go/pluginsdk/recommendation_history.go:40` to
      `s.logger.WithLevel(handlerLogLevel(clientErr))` per FR-004 (missing)
- [x] T021 Widen the code coverage of the interceptor and wrapper tests to all 16 non-OK gRPC codes, each with its
      expected level (debug for the four declines, error for the rest): drive `TestWrapperDeclineLevel` and
      `TestWrapperFaultLevel` in `sdk/go/pluginsdk/handler_status_test.go` and the two interceptor tests in
      `sdk/go/pluginsdk/logging_test.go` from the full code list per FR-008 and SC-003 (partial)
- [x] T022 Correct "eight" to "nine" wrappers and add `GetRecommendationHistory` to the RPC lists in
      `specs/605-sdk-decline-log-level/spec.md` (FR-004, Assumptions), `plan.md`, `research.md`,
      `contracts/log-levels.md`, `CLAUDE.md` (Recent Changes, Active pattern bullet), `AGENTS.md`, `GEMINI.md`,
      and `PR_MESSAGE.md`; revalidate with `cat PR_MESSAGE.md | npx commitlint` per FR-004 (partial)
- [x] T023 Re-run `golangci-lint run ./sdk/go/pluginsdk/...`, `make test`, and `make lint-markdown` after
      T020-T022 per the Phase 6 gates (partial)
