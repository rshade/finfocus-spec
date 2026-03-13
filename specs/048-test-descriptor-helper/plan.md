# Implementation Plan: Extract Test Helper for ResourceDescriptor Creation

**Branch**: `048-test-descriptor-helper` | **Date**: 2026-03-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/048-test-descriptor-helper/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See
`.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extract a `newTestDescriptor(id, arn string, opts ...pluginsdk.ResourceDescriptorOption)` helper
function in `helpers_test.go` that provides standard test defaults (provider=`"aws"`,
resourceType=`"ec2"`, sku=`"t3.micro"`, region=`"us-east-1"`), then refactor existing tests to
use it where appropriate. This is a pure test-internal refactoring with zero behavior changes.

## Technical Context

**Language/Version**: Go 1.25.8 (per go.mod)
**Primary Dependencies**: `github.com/stretchr/testify`, `google.golang.org/protobuf` (existing, unchanged)
**Storage**: N/A (test-only refactoring)
**Testing**: `go test ./sdk/go/pluginsdk/...`
**Target Platform**: All Go-supported platforms
**Project Type**: Library (Go SDK test code)
**Performance Goals**: N/A (test helper, no production code)
**Constraints**: Zero behavior changes in existing tests; unexported helper only
**Scale/Scope**: Single file (`helpers_test.go`, ~1920 lines)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Proto Specification-First | N/A | No proto changes |
| II. Multi-Provider Consistency | N/A | No new provider features |
| III. Spec Consumes, Not Calculates | N/A | Test-only change |
| IV. Strict Separation of Concerns | PASS | Helper is test-internal (unexported) |
| V. Test-First Protocol | PASS | Refactoring existing tests; all must pass identically |
| VI. Protobuf Backward Compatibility | N/A | No proto changes |
| VII. Documentation & Identity | PASS | No documentation changes needed (internal test helper) |
| VIII. Performance as Requirement | N/A | Test code only |
| IX. Observability & Validation | N/A | No observability changes |
| X. Follow Established Patterns | PASS | Follows existing functional options pattern |
| XI. Mandatory Copyright Headers | N/A | No new files created |
| XII. Automatic Capability Declaration | N/A | No capability changes |
| XIII. Multi-Language SDK Sync | N/A | No proto/API changes |
| XIV. Documentation Integrity | N/A | No exported API changes |

**Result**: All gates PASS or N/A. No violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/048-test-descriptor-helper/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
└── quickstart.md        # Phase 1 output
```

### Source Code (repository root)

```text
sdk/go/pluginsdk/
└── helpers_test.go      # Only file modified (add helper + refactor callsites)
```

**Structure Decision**: No new files or directories. The helper function is added to the existing
`helpers_test.go` file as an unexported function in the `pluginsdk_test` package. No contracts
directory is needed since there are no API changes.

## Complexity Tracking

No violations to justify. This is a minimal-scope refactoring of test code.
