# Tasks: Allocation Period and Partial-Selection Scope

**Input**: Design documents from `specs/588-allocate-period-selector/`

**Tests**: Required (constitution principle V). Test tasks run first and are seen failing: on compile before T002,
and on behavior after it.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Confirm the baseline: `go test ./sdk/go/testing/ ./sdk/go/internal/refalloc/ ./sdk/go/pluginsdk/ -run
  'Allocat'` passes

## Phase 2: Foundational (proto first)

- [X] T002 In proto/finfocus/v1/allocation.proto import `google/protobuf/timestamp.proto` and add `start = 5`, `end
  = 6`, `selector = 7` to `AllocateRequest` and `start = 5`, `end = 6` to `AllocateResponse`, with the comments in
  contracts/proto-diff.md
- [X] T003 Run `make generate`, `make buf-lint`, and `buf breaking` against main; all clean

## Phase 3: User Story 1 - Window on the request and its echo (P1) MVP

- [X] T004 [P] [US1] Add failing cases to sdk/go/testing/allocation_test.go: `ValidateAllocateRequest` accepts no
  window and start equal to end; rejects only start, only end, and start after end with
  `ErrInvalidAllocateRequest` and `codes.InvalidArgument`, each message naming the problem
- [X] T005 [P] [US1] Add a failing test in sdk/go/internal/refalloc/refalloc_test.go: a windowed request's response
  carries the identical start and end, an unwindowed one carries none, and the rows for the same fixture are
  identical with and without the window (FR-011)
- [X] T006 [US1] Implement the window rule in `ValidateAllocateRequest` (sdk/go/testing/allocation.go) using
  `timestampAfter`, and update its godoc
- [X] T007 [US1] Echo start and end in `refalloc.Allocate` (sdk/go/internal/refalloc/refalloc.go) with
  `proto.CloneOf`

## Phase 4: User Story 3 - Echo validation and older allocators (P1)

- [X] T008 [US3] Add failing cases to sdk/go/testing/allocation_test.go for `ValidateAllocateResponse`: no echo
  passes against a windowed request; an equal echo passes; a different start, a different end (nanos only), and a
  window on a response whose request had none each fail with `ErrInvalidAllocateResponse`, naming `start` or `end`
- [X] T009 [US3] Implement the echo rule in `ValidateAllocateResponse` and update its godoc

## Phase 5: User Story 2 - Selector (P1)

- [X] T010 [US2] Add a failing test in sdk/go/internal/refalloc/refalloc_test.go: with selector `{"namespace":
  "payments"}`, the response passes `ValidateAllocateResponse` and `CheckConservation` and its warnings contain
  "selector narrows workloads"; with no selector, no such warning
- [X] T011 [US2] Add the warning to `refalloc.Allocate` when `len(selector) > 0`; no computation change

## Phase 6: User Story 4 - Conformance (P2)

- [X] T012 [US4] Add failing tests in sdk/go/testing/allocator_conformance_test.go: the reference allocator passes
  `period_echoed` and `selector_keeps_invariants`; an allocator that drops the window fails `period_echoed`
- [X] T013 [US4] Add both scenarios to the scenario list in sdk/go/testing/allocator_conformance.go

## Phase 7: User Story 5 - TypeScript (P2)

- [X] T014 [US5] Add a case to sdk/typescript/packages/client/test/allocator.test.ts: a request with start, end, and
  a selector sends them (asserted in the request body), and the echoed start and end are readable (seen failing
  first)
- [X] T015 [US5] Run `cd sdk/typescript && npm run build && npm test && npm run lint`

## Phase 8: Polish

- [X] T016 [P] Document the window, the echo rule, the selector, and the host rule in docs/allocator.md (Request,
  Response, Host Verification)
- [X] T017 [P] Update the allocator sections of sdk/go/pluginsdk/README.md and sdk/go/testing/README.md (new
  conformance scenarios)
- [X] T018 [P] Add CLAUDE.md "Active Technologies", "Recent Changes", and a short pattern note by hand
- [X] T019 Gates: `make generate && git diff --exit-code -- sdk/`, `make buf-lint`, `buf breaking`, `make test`, `go
  test -tags=integration ./sdk/go/testing/`, `go test -run TestConformance ./sdk/go/testing/`, `make lint-go`, `make
  lint-markdown`, `make lint-yaml`, `make validate-npm`

## Dependencies & Execution Order

Phase 2 blocks everything. Tests precede their implementations (T004/T005 → T006/T007, T008 → T009, T010 → T011,
T012 → T013, T014 → T015). Phase 6 needs T007, T009, and T011. Polish comes last.

## Parallel Opportunities

T004 with T005; T016-T018.

## Implementation Strategy

MVP is Phases 2-4 (window, echo, and compatibility). Phase 5 adds the selector signal, Phase 6 enforces it for
allocator authors, Phase 7 closes SDK parity, then docs.

## Phase 9: Convergence

- [X] T020 Add a partial-selection note (selector meaning, invariants unchanged, hosts omit or label idle and
  cluster rows) to the allocator sections of sdk/go/pluginsdk/README.md and sdk/go/testing/README.md, linking
  docs/allocator.md#partial-selections, per FR-013 (partial)
