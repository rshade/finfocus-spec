# Implementation Plan: Shared Conformance Harness and Duplicate-Key Check

**Branch**: `590-shared-supplemental-harness` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/590-shared-supplemental-harness/spec.md`

## Summary

Replace six copies of the bufconn start/stop/client lifecycle in `sdk/go/testing` with one
unexported generic `bufconnHarness[C]`, embedded by value in each exported harness. Replace
three copies of the pairwise-then-map duplicate scan with one generic
`findDuplicate[T, K]` that returns indices, so each dataset keeps its own error text. No
exported identifier, validation rule, error message, or wire behavior changes (see
[research.md](research.md)).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod)

**Primary Dependencies**: google.golang.org/grpc (bufconn), generated `pbc` clients and
registrars; no new dependencies

**Storage**: N/A

**Testing**: `go test` (unit, `-tags=integration`, conformance), `testing.AllocsPerRun`
allocation guards, benchmarks A/B against a prebuilt `main` test binary

**Target Platform**: Go SDK library (`sdk/go/testing`)

**Project Type**: library (SDK test-support package)

**Performance Goals**: response validators stay 0 allocs/op on valid pages of up to 64
records (hard gate); `BenchmarkValidateGetContractCommitmentsResponse` A/B is informational
and is reported in the PR, with any slowdown above 10% on `page_50` explained

**Constraints**: source-compatible exported API; `sdk/go/testing` cannot import `pluginsdk`;
golangci-lint baseline 0 issues

**Scale/Scope**: 9 existing source files edited, 1 new source file, 2 new test files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | Pass (N/A) | No `.proto` change |
| II. Multi-provider consistency | Pass (N/A) | No provider-specific behavior |
| III. Consumes, does not calculate | Pass | No cost logic |
| IV. Separation of concerns | Pass | Rules stay in `sdk/go/testing`; `pluginsdk` delegation unchanged |
| V. Test-first | Pass | `findDuplicate` and harness lifecycle tests are written before the helpers and fail to compile first; existing suites are the behavior lock |
| VI. Backward compatibility | Pass | Exported names, constructors, and method sets unchanged ([contract](contracts/exported-api.md)) |
| VII. Documentation | Pass | No user-facing doc changes; the README is checked for harness internals |
| VIII. Performance | Pass | Allocation guards unchanged; benchmark A/B recorded |
| IX. Observability and validation | Pass | Error text and codes unchanged |
| X. Established patterns | Pass | Follows the existing generic helpers `paginateRecords[T]` and `validatePagedResponse` |
| XI. Copyright headers | Pass | New files get the Apache 2.0 header |
| XII. Capability declaration | Pass (N/A) | No capability change |
| XIII. Multi-language SDK sync | Pass (N/A) | Go test-support internals only; TypeScript has no equivalent |
| XIV. Documentation integrity | Pass | CLAUDE.md pattern notes are updated to point at the shared helpers |

Post-design re-check: no violations. Complexity Tracking is not needed.

## Project Structure

### Documentation (this feature)

```text
specs/590-shared-supplemental-harness/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/exported-api.md
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
sdk/go/testing/
├── bufconn_harness.go              # NEW: bufconnHarness[C], newBufconnHarness, dial
├── bufconn_harness_test.go         # NEW: lifecycle tests (internal package)
├── supplemental.go                 # findDuplicate; checkDuplicateCommitmentIDs delegates
├── find_duplicate_test.go          # NEW: table tests (internal package)
├── invoice_dataset.go              # billing-period and invoice-detail checks delegate
├── harness.go                      # TestHarness embeds bufconnHarness
├── spec_validation.go              # createClientConnection uses h.dial()
├── allocator_conformance.go        # AllocatorHarness embeds bufconnHarness
├── scorer_conformance.go           # ScorerHarness embeds bufconnHarness
├── usage_source.go                 # UsageSourceHarness embeds bufconnHarness
├── contract_commitment_conformance.go  # ContractCommitmentHarness embeds bufconnHarness
└── invoice_dataset_conformance.go  # InvoiceDatasetHarness embeds bufconnHarness
```

**Structure Decision**: All changes stay inside `sdk/go/testing`. Internal tests use
`package testing` (as `export_test.go` already does), so the unexported helpers are tested
directly without new exports.
