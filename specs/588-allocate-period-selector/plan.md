# Implementation Plan: Allocation Period and Partial-Selection Scope

**Branch**: `588-allocate-period-selector` | **Date**: 2026-10-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/588-allocate-period-selector/spec.md`

## Summary

Add `start`, `end`, and `selector` to `AllocateRequest` and an echoed `start` and `end` to
`AllocateResponse`, mirroring `GetStatsRequest`. Request validation enforces the window rule. Response
validation accepts a missing echo but rejects a wrong one. The reference allocator echoes and warns
under a selector, conformance gains two scenarios, and TypeScript and the docs follow. No invariant
changes.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf (timestamppb), google.golang.org/grpc,
connectrpc.com/connect, buf v1.32.1; no new dependencies

**Storage**: N/A

**Testing**: `go test`, bufconn conformance (`RunAllocatorConformance`), vitest + msw

**Target Platform**: Allocator plugins and hosts (Go, TypeScript)

**Project Type**: Protocol specification and SDK library

**Performance Goals**: No zero-allocation path is touched; allocation validation already allocates

**Constraints**: Additive only; the rules live in `sdk/go/testing/allocation.go` (052 pattern)

**Scale/Scope**: Five proto fields, two validator changes, reference allocator, two conformance
scenarios, TS test, docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | Proto and `make generate` first |
| II. Multi-provider consistency | PASS | Selector semantics shared with the usage source |
| III. Spec consumes, does not calculate | PASS | The period never changes cost; no new math |
| IV. Separation of concerns | PASS | Allocation math stays in the allocator; hosts only omit or label rows |
| V. Test first | PASS | Validator, refalloc, conformance, and TS tests fail first |
| VI. Backward compatibility | PASS | New fields only; a missing echo is accepted at runtime |
| VII. Documentation | PASS | `docs/allocator.md`, READMEs, CLAUDE.md |
| VIII. Performance | PASS | No hot path touched |
| IX. Observability and validation | PASS | Errors name the field and the problem |
| X. Established patterns | PASS | `invalidArgumentError`, `timestampAfter`, conformance scenario list |
| XI. Copyright headers | PASS | No new source files expected; any new one gets the header |
| XII. Capability declaration | PASS | Same capability (ALLOCATION) |
| XIII. Multi-language SDK sync | PASS | TS regenerated and tested |
| XIV. Documentation integrity | PASS | Godoc updated for changed validators |

Post-design re-check: PASS.

## Project Structure

### Documentation (this feature)

```text
specs/588-allocate-period-selector/
├── plan.md, research.md, data-model.md, quickstart.md
├── contracts/proto-diff.md
├── checklists/
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/allocation.proto                        # fields 5-7 and 5-6
sdk/go/proto/..., sdk/typescript/.../generated/           # regenerated
sdk/go/testing/allocation.go                              # window and echo rules
sdk/go/testing/allocation_test.go                         # rule tests
sdk/go/testing/allocator_conformance.go                   # two scenarios
sdk/go/testing/allocator_conformance_test.go              # scenario pass/fail tests
sdk/go/internal/refalloc/refalloc.go, refalloc_test.go    # echo and warning
sdk/typescript/packages/client/test/allocator.test.ts
docs/allocator.md, sdk/go/pluginsdk/README.md, sdk/go/testing/README.md, CLAUDE.md
```

**Structure Decision**: Existing layout. No new packages.

## Complexity Tracking

No constitution violations.
