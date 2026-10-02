# Tasks: Allocation Row Provenance

**Input**: Design documents from `specs/594-allocation-row-provenance/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/allocation-row.md

Test-first (constitution V): within each story, tests are written and shown failing before implementation.
Proto first (constitution I): the proto change is Foundational because tests need the generated symbols.
`CHANGELOG.md` is never edited (release-please owns it).

## Phase 1: Setup

- [x] T001 Confirm worktree is on branch `594-allocation-row-provenance`, run `npm ci` and `make generate`, and record
  that `git status` is clean apart from `specs/` and `CLAUDE.md`

## Phase 2: Foundational (blocks all stories)

- [x] T002 Edit `proto/finfocus/v1/allocation.proto`: add to `AllocationRow` `string allocated_method_id = 7;`,
  `string allocated_method_details = 8;`, `string allocated_resource_id = 9;` with comments naming FOCUS 1.3
  AllocatedMethodId / AllocatedMethodDetails / AllocatedResourceId, the rule "a non-empty allocated_method_id requires
  a non-empty allocated_resource_id", "opaque strings, conventionally the resource.id of the priced resource the cost
  came from", and a comment holding field 10 for a later LineageNode (not `reserved`); also add the rule to the
  `AllocatorService` invariants list
- [x] T003 Run `make generate`; confirm only allocation-related files under `sdk/go/proto/` and
  `sdk/typescript/packages/client/src/generated/` changed (restore unrelated reformatted generated files with `git
  checkout`)

## Phase 3: User Story 1 - Host maps allocation rows to FOCUS columns (P1)

**Goal**: rows carry method id, details, source resource id end to end; reference allocator fills them.
**Independent Test**: reference allocator over gRPC returns provenance on every row.

- [x] T004 [US1] Write failing test in `sdk/go/internal/refalloc/refalloc_test.go`: every row from a two-node fixture
  has `AllocatedMethodId == "refalloc.proportional"` and a non-empty `AllocatedResourceId` (node id for workload and
  idle rows, priced resource id for cluster rows), and `ValidateAllocateResponse` still passes
- [x] T005 [US1] Implement in `sdk/go/internal/refalloc/refalloc.go`: set the method id constant and source resource
  id in `rowBuilder.row` and the cluster-row literal (about line 316); keep conservation math untouched
- [x] T006 [P] [US1] Add a transport round-trip assertion in `sdk/go/pluginsdk/allocator_serve_test.go` (gRPC and
  Connect) that provenance values survive Serve

## Phase 4: User Story 2 - Validator and conformance (P1)

**Goal**: method id without source resource id is rejected and flagged by name.
**Independent Test**: validate method-only, resource-only, full, empty rows.

- [x] T007 [US2] Write failing table test in `sdk/go/testing/allocation_test.go`: method-only row fails with
  `ErrInvalidAllocateResponse` and message containing `rows[0]` and `allocated_method_id`; resource-only,
  details-only, full, and empty rows pass; a valid provenance row leaves conservation unchanged
- [x] T008 [US2] Implement the rule in `validateAllocationRow` in `sdk/go/testing/allocation.go` (`if
  row.GetAllocatedMethodId() != "" && row.GetAllocatedResourceId() == ""` returns `fmt.Errorf("%w: rows[%d]:
  allocated_method_id requires allocated_resource_id", ErrInvalidAllocateResponse, i)`); do not touch
  `ValidateAllocateResponse` itself
- [x] T009 [US2] Write failing test in `sdk/go/testing/allocator_conformance_test.go` using
  `RunAllocatorScenariosForTest`: a broken allocator emitting a method-only row fails scenario `row_provenance` by
  name while a compliant one passes
- [x] T010 [US2] Add `scenarioRowProvenance` to `sdk/go/testing/allocator_conformance.go` (single-node fixture, check
  every returned row with a dedicated message) and register `row_provenance` in `allocatorScenarios()`; the scenario
  must not require an allocator to emit provenance
- [x] T011 [P] [US2] Add a `BenchmarkValidateAllocateResponse` variant row set with provenance in
  `sdk/go/testing/allocation_test.go` and a new `testing.AllocsPerRun` assertion (pattern in
  `sdk/go/testing/invoice_focus14_test.go:49`) that `ValidateAllocateResponse` on provenance-bearing valid rows
  allocates no more than the same response without provenance

## Phase 5: User Story 3 - TypeScript client and docs (P2)

**Goal**: TS exposes the values; docs describe them.
**Independent Test**: vitest round-trip and markdownlint.

- [x] T012 [US3] Write test in `sdk/typescript/packages/client/test/allocator.test.ts`: a row with the three values
  round-trips through the AllocatorClient (proto3 JSON and binary as the file already does) and an unset row decodes
  to empty strings; run `npx vitest run` and show it fails only if bindings are missing
- [x] T013 [P] [US3] Update `docs/allocator.md` Rows section (about line 161): list the three fields, FOCUS 1.3 column
  mapping, the method-needs-source rule, opaque-string convention, and that a lineage chain is a later addition
- [x] T014 [P] [US3] Update `sdk/go/pluginsdk/README.md` ("Allocation-Only Plugins", about line 949) and
  `sdk/typescript/README.md` (allocator section, about line 212) with the three fields and a link to
  `docs/allocator.md`

## Phase 6: Polish and gates

- [x] T015 Run `make generate && git diff --exit-code -- sdk/` style check: regenerate again and confirm no further
  diff after T003
- [x] T016 Run `make buf-lint`
- [x] T017 Run `buf breaking` against `main` (use `git archive origin/main proto buf.yaml` into a scratch dir if the
  git-branch form fails in a worktree); expect no findings
- [x] T018 Run `make test`; on failure capture the failing test with `-v` and do not call it flaky without evidence;
  on pass record the package `ok` lines
- [x] T019 Run `go test -v -tags=integration ./sdk/go/testing/` and `go test -v -run TestConformance ./sdk/go/testing/`
- [x] T020 Run `make lint-go` (extended timeout) with 0 findings
- [x] T021 Run `make lint-markdown` and `make lint-yaml`
- [x] T022 Run `make validate-npm`
- [x] T023 Run `cd sdk/typescript && npm ci && npm run build && npm test` (the middleware TS7006 build error, if
  present, is pre-existing)
- [x] T024 Benchmark A/B: build test binaries for `sdk/go/testing` on this branch and on an `origin/main` worktree,
  run `-bench=BenchmarkValidateAllocateResponse -benchmem -count=10` and compare; no regression beyond noise and 0
  extra allocs
- [x] T025 Run `/speckit-analyze` until clean and `/speckit-converge` until it appends no tasks
- [x] T026 Write `PR_MESSAGE.md` (conventional commit `feat(proto): ...`, body references
  `specs/594-allocation-row-provenance/`, `Closes #578`, no Claude-Session or Co-Authored-By trailer) and validate
  with `cat PR_MESSAGE.md | npx commitlint`
- [x] T027 List the exact `git add <named files>`, `git commit -F PR_MESSAGE.md`, `git push -u origin
  594-allocation-row-provenance`, and `gh pr create` commands for the user; do not run them

## Dependencies

T001 -> T002 -> T003 -> (US1, US2, US3 can proceed in parallel after T003; US1 and US2 touch different files) -> Phase
  6.
Within a story: test task precedes its implementation task.
