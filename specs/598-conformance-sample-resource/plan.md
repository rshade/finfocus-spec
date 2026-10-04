# Implementation Plan: Plugin-Supplied Sample Resource for Conformance

**Branch**: `598-conformance-sample-resource` | **Date**: 2026-10-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/598-conformance-sample-resource/spec.md`

## Summary

The cost-source conformance suite sends one hard-coded `aws/ec2/t3.micro/us-east-1` descriptor from
every check, so a plugin that does not price AWS cannot pass Basic level. This change lets the caller
supply the sample resource: a `SuiteConfig.SampleResource` field (nil means the current default), a
`RunConformance(impl, level, opts...)` entry point with `WithSampleResource` in both `sdk/go/testing`
and `sdk/go/pluginsdk`, and a `TestHarness.SampleResource()` copy that every registered check reads.
The two bare `GetActualCost` checks now send the sample resource in `GetActualCostRequest.resource`.
No proto, generated code, or TypeScript change. Design decisions are in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod)

**Primary Dependencies**: google.golang.org/protobuf (`proto.CloneOf`), google.golang.org/grpc;
no new dependencies

**Storage**: N/A

**Testing**: `go test` (testify), bufconn harness; `make test`, `make lint-go`

**Target Platform**: Go library (plugin SDK test tooling)

**Project Type**: library

**Performance Goals**: latency checks must not time descriptor copying (copy once, outside the loop)

**Constraints**: no exported signature change (SC-002); provider neutrality (FR-011): new code, tests,
and docs use only `custom` or placeholders besides the kept `aws` default

**Scale/Scope**: 16 check call sites in 4 files of `sdk/go/testing` plus `conformance.go` and
`harness.go`, one file in `pluginsdk`, three docs; the `ErrorHandlingTestSuite` sites stay as they are

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Note |
| --- | --- | --- |
| I. Proto first | N/A | No `.proto` change |
| II. Multi-provider consistency | Pass | Removes an AWS-only assumption from conformance; adds no provider-specific content |
| III. Spec consumes | Pass | No pricing logic |
| IV. Separation of concerns | Pass | Test tooling only |
| V. Test first | Pass | Neutral-plugin tests are written first and fail on the hard-coded descriptor |
| VI. Backward compatibility | Pass | Only new identifiers; nil field means old behavior; buf unaffected |
| VII. Documentation | Pass | testing README, pluginsdk README, developer guide in the same PR |
| VIII. Performance | Pass | Copy taken outside timed loops; no hot-path SDK code touched |
| X. Established patterns | Pass | Follows `NewConformanceSuiteWithConfig` / `SuiteConfig`; option style matches pluginsdk |
| XI. Headers | Pass | New test file carries the Apache 2.0 header |
| XIII. SDK sync | N/A | No RPC change; no TypeScript conformance suite exists |
| XIV. Doc integrity | Pass | New exported identifiers get godoc; README examples compile |

Post-design re-check: no change; no violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/598-conformance-sample-resource/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/go-api.md
├── checklists/requirements.md
└── tasks.md            # /speckit-tasks
```

### Source Code (repository root)

```text
sdk/go/testing/
├── conformance.go          # SampleResource field, DefaultSampleResource, ConformanceOption,
│                           # WithSampleResource, RunConformance, validation in Run/RunCategory
├── harness.go              # TestHarness.sampleResource + SampleResource()
├── spec_validation.go      # 3 checks + RunSpecValidation read the harness resource
├── rpc_correctness.go      # Supports, projected (x2), pricing spec, actual cost (x3)
├── performance.go          # latency checks + RunPerformanceBenchmarks
├── concurrency.go          # parallel and consistency checks
├── sample_resource_test.go # new: neutral strict plugin, all levels, validation, copy semantics
└── README.md
sdk/go/pluginsdk/
├── conformance.go          # ConformanceOption alias, WithSampleResource, RunConformance
├── conformance_test.go     # neutral plugin through pluginsdk.RunConformance
└── README.md
PLUGIN_DEVELOPER_GUIDE.md   # cost-source conformance subsection
CLAUDE.md                   # pattern notes (by hand)
```

**Structure Decision**: Existing packages only; one new test file.

## Complexity Tracking

No constitution violations.
