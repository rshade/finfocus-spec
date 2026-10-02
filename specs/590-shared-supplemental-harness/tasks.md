---

description: "Task list for 590-shared-supplemental-harness"
---

# Tasks: Shared Conformance Harness and Duplicate-Key Check

**Input**: Design documents from `specs/590-shared-supplemental-harness/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/exported-api.md

**Tests**: Required. The constitution's Principle V makes the test-first approach mandatory,
and each story writes its failing tests before the helper exists.

**Organization**: Tasks are grouped by user story. All paths are under `sdk/go/testing/`.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Record the baseline: from the worktree root, run `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/` and keep a
  `main` test binary (`go test -c -o <scratch>/testing.base.test ./sdk/go/testing/`) for the A/B in T019

## Phase 2: Foundational

No shared prerequisites: US1 and US2 touch disjoint files.

## Phase 3: User Story 1 - One duplicate-key strategy (Priority: P1) MVP

**Goal**: One `findDuplicate[T any, K comparable]` backs the contract commitment, billing
period, and invoice detail duplicate checks, with identical results and messages, and 0
allocs at 64 records or fewer.

**Independent Test**: `go test -run 'TestFindDuplicate|AllocationFree|Duplicate' ./sdk/go/testing/`

### Tests for User Story 1 (write first, must fail)

- [X] T002 [US1] Create `sdk/go/testing/find_duplicate_test.go` (`package testing`, Apache 2.0 header) with a table
  test `TestFindDuplicate` over `[]string` keys covering: empty, one record, no duplicate at sizes 64 and 65, one
  duplicate at sizes 64 and 65 (same `(i, first)` on both), several duplicates (lowest later index wins, matched to
  the earliest occurrence), an empty-string key, and a duplicate where the first occurrence repeats three times.
  Assert `(i, first, ok)` exactly
- [X] T003 [US1] In `sdk/go/testing/find_duplicate_test.go`, add `TestFindDuplicateCompositeKey` using
  `billingPeriodIdentity` over `[]*pbc.BillingPeriod`: same issuer and start with a different end is a duplicate, and
  the same issuer with a start differing only in nanos is not. Add `TestFindDuplicateAllocationFree`, which asserts
  `testing.AllocsPerRun` is 0 for a 64-record `[]*pbc.ContractCommitment` page with a method-expression key and for a
  64-record billing-period page with `billingPeriodIdentity`
- [X] T004 [P] [US1] In `sdk/go/testing/invoice_dataset_test.go` `TestInvoiceDatasetRPCValidatorsAllocationFree`, add
  "billing response 64" and "invoice response 64" cases using 64 distinct valid records (`PageSize: 64`). The current
  cases use one-record pages and never reach the pairwise loop
- [X] T005 [US1] Run `go test ./sdk/go/testing/` and confirm that T002 and T003 fail to compile (`undefined:
  findDuplicate`) and that T004 passes against the old code (it is a lock, not a red test)

### Implementation for User Story 1

- [X] T006 [US1] Add `findDuplicate[T any, K comparable](list []T, key func(T) K) (i, first int, ok bool)` to
  `sdk/go/testing/supplemental.go` directly below `pairwiseDuplicateLimit`: pairwise when `len(list) <=
  pairwiseDuplicateLimit` (hoist `key(list[i])` out of the inner loop), otherwise `make(map[K]int, len(list))` storing
  the first index per key. Doc comment states the reported pair and the zero-allocation path
- [X] T007 [US1] Rewrite `checkDuplicateCommitmentIDs` in `sdk/go/testing/supplemental.go` to call
  `findDuplicate(list, (*pbc.ContractCommitment).GetContractCommitmentId)` and return `duplicateError(i, first,
  list[i].GetContractCommitmentId())`. The message text is unchanged
- [X] T008 [US1] Rewrite `checkDuplicateBillingPeriods` and `checkDuplicateInvoiceDetails` in
  `sdk/go/testing/invoice_dataset.go` to call `findDuplicate` (keys `billingPeriodIdentity` and
  `(*pbc.InvoiceDetail).GetInvoiceDetailId`), keeping `duplicateBillingPeriod` and `duplicateInvoiceDetail` unchanged.
  Delete `sameBillingPeriodIdentity` (its only caller is the replaced loop)
- [X] T009 [US1] Run `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/` and confirm that T002 to T004,
  `TestContractCommitmentValidatorsAllocationFree`, `TestInvoiceDatasetValidatorsAllocationFree`, and all existing
  duplicate-message tests pass

**Checkpoint**: US1 is complete and shippable alone.

## Phase 4: User Story 2 - One harness implementation (Priority: P2)

**Goal**: The six exported harnesses embed one generic `bufconnHarness[C]`, with an
unchanged exported API.

**Independent Test**: `go test -run 'TestBufconnHarness|Conformance|Harness' ./sdk/go/testing/` plus quickstart step 4.

### Tests for User Story 2 (write first, must fail)

- [X] T010 [US2] Create `sdk/go/testing/bufconn_harness_test.go` (`package testing`, Apache 2.0 header) with
  `TestBufconnHarness`. Build a harness with `newBufconnHarness` that registers `MockContractCommitmentSource` through
  the existing `contractCommitmentAdapter` and `pbc.NewSupplementalDatasetServiceClient`. Subtests: `Start` then one
  `GetContractCommitments` call succeeds; `GetBillingPeriods` returns `codes.Unimplemented`; `Stop` twice does not
  panic; `Stop` without `Start` does not panic; `Client()` before `Start` is nil. Confirm it fails to compile

### Implementation for User Story 2

- [X] T011 [US2] Create `sdk/go/testing/bufconn_harness.go` (Apache 2.0 header) with `bufconnHarness[C any]` (fields
  `server`, `listener`, `conn`, `client`, `newClient`), `newBufconnHarness[C any](register func(*grpc.Server),
  newClient func(grpc.ClientConnInterface) C) bufconnHarness[C]` (listens on `bufSize` and serves in a goroutine),
  `dial() (*grpc.ClientConn, error)` holding the single `//nolint:staticcheck // grpc.NewClient doesn't work with
  bufconn` `grpc.DialContext`, and the methods `Start(t testing.TB)` (calls `t.Fatalf("Failed to dial bufnet: %v",
  err)` on error), `Stop()`, and `Client() C` with the existing semantics
- [X] T012 [US2] Convert `ContractCommitmentHarness` in `sdk/go/testing/contract_commitment_conformance.go` and
  `InvoiceDatasetHarness` in `sdk/go/testing/invoice_dataset_conformance.go` to embed
  `bufconnHarness[pbc.SupplementalDatasetServiceClient]`. Constructors register their adapter through
  `pbc.RegisterSupplementalDatasetServiceServer`. Delete their `Start`/`Stop`/`Client` methods and keep the type doc
  comments. Remove now-unused imports
- [X] T013 [US2] Convert `TestHarness` in `sdk/go/testing/harness.go` to embed
  `bufconnHarness[pbc.CostSourceServiceClient]`, and change `createClientConnection` in
  `sdk/go/testing/spec_validation.go` to `return h.dial()` (drop its `nolint` and unused imports)
- [X] T014 [US2] Convert `AllocatorHarness` (`sdk/go/testing/allocator_conformance.go`), `ScorerHarness`
  (`sdk/go/testing/scorer_conformance.go`), and `UsageSourceHarness` (`sdk/go/testing/usage_source.go`) the same way.
  Touch only the harness block (the conflict surface with #579/#580)
- [X] T015 [US2] Run `go build ./...` and `go test ./sdk/go/...`, and run quickstart step 4 (`go doc`) for all six
  harnesses. If `go doc` does not list the promoted `Start`/`Stop`/`Client`, add thin exported wrapper methods instead
  and record that in research.md R2

**Checkpoint**: US1 and US2 are both complete.

## Phase 5: Polish and Cross-Cutting

- [X] T016 [P] In `sdk/go/testing/README.md` § Test Harness, replace the `TestHarness` struct snippet that lists the
  private fields `server`, `listener`, `client`, and `conn` with a sentence saying that every harness shares one
  in-memory bufconn lifecycle (`Start`, `Stop`, `Client`). `sdk/go/testing/CLAUDE.md` mentions only bufconn and needs
  no change
- [X] T017 [P] Add a short pattern entry and a Recent Changes bullet for 590 to `CLAUDE.md` by hand, in the existing
  style (the `update-agent-context.sh` script mangles multi-line entries)
- [X] T018 Run quickstart steps 1-5: `make test`, `go test -tags=integration ./sdk/go/testing/`, `go test -run
  TestConformance ./sdk/go/testing/`, `golangci-lint run ./...` (0 issues), `make lint-markdown`, and the two `grep`
  counts
- [X] T019 Run the `BenchmarkValidateGetContractCommitmentsResponse` A/B (`-count 6`) against the T001 binary and
  record the result in the PR. Explain any `page_50` slowdown above 10%

## Dependencies and Execution Order

- T001 comes first. US1 (T002-T009) and US2 (T010-T015) touch disjoint files and can run in either order. US1 is the
  MVP.
- Within US1: T002-T004 → T005 → T006 → T007 and T008 → T009.
- Within US2: T010 → T011 → T012-T014 → T015.
- Polish (T016-T019) comes after both stories.

## Parallel Opportunities

- T004 runs in parallel with T002 and T003 (different file).
- T012, T013, and T014 edit different files, but all depend on T011. Run them in sequence to
  keep one build state.
- T016 and T017 run in parallel.

## Implementation Strategy

MVP first: finish US1, then run T009. US2 follows as a second increment in the same PR.
Then polish and the full gate.
