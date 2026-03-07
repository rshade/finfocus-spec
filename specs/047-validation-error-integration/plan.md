# Implementation Plan: Integrate ValidationError with Validation Implementation

**Branch**: `047-validation-error-integration` | **Date**: 2026-03-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/047-validation-error-integration/spec.md`

## Summary

Integrate the existing `ValidationError` struct into the validation implementation in
`focus_conformance.go`. Currently, 24 error return sites use either sentinel errors (7) or
inline `errors.New`/`fmt.Errorf` strings (17). All will be converted to return `*ValidationError`
with structured field details, wrapping the original sentinel where applicable. An `Unwrap()`
method and new constructor will be added to `ValidationError` to support Go's error chain
traversal (`errors.Is` + `errors.As`).

## Technical Context

**Language/Version**: Go 1.25.8 (per go.mod)
**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc (existing, unchanged)
**Storage**: N/A (stateless validation functions)
**Testing**: `go test ./sdk/go/pluginsdk/...` with table-driven tests, benchmarks
**Target Platform**: Go SDK library (cross-platform)
**Project Type**: Library (Go SDK package)
**Performance Goals**: One allocation per validation error (struct creation); zero-allocation
sentinel errors preserved as package-level variables for `errors.Is` matching
**Constraints**: Full backward compatibility for `errors.Is` checks; existing test suite must pass
**Scale/Scope**: 24 error return sites across 1 file (`focus_conformance.go`), 1 type modification
(`validation_error.go`), test updates across 2 test files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Proto-First | N/A | No proto changes; Go SDK-only refactoring |
| II. Multi-Provider | N/A | No provider-specific changes |
| III. Spec Consumes | PASS | No pricing logic added |
| IV. Separation of Concerns | PASS | SDK-internal refactoring only |
| V. Test-First | APPLICABLE | Must write tests for `Unwrap()` and `errors.As` extraction before modifying validation |
| VI. Backward Compatibility | PASS | `errors.Is` preserved via `Unwrap()`; no proto changes |
| VII. Documentation | APPLICABLE | Must update godoc comments on `ValidationError` |
| VIII. Performance | APPLICABLE | Must benchmark `ValidationError` creation overhead |
| IX. Observability | PASS | Richer error messages improve diagnostic output |
| X. Follow Established Patterns | PASS | Extends existing `ValidationError` pattern |
| XI. Copyright Headers | APPLICABLE | Maintain headers on modified files |
| XII. Capability Discovery | N/A | Not related to capabilities |
| XIII. Multi-Language SDK | N/A | TypeScript SDK has no equivalent validation layer |
| XIV. Documentation Integrity | APPLICABLE | Must update README.md examples if they reference validation errors |

**Gate Result**: PASS - No violations. Applicable principles will be addressed during implementation.

## Project Structure

### Documentation (this feature)

```text
specs/047-validation-error-integration/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
sdk/go/pluginsdk/
├── validation_error.go       # MODIFY: Add Unwrap(), Err field, new constructor
├── validation_error_test.go  # MODIFY: Add Unwrap/errors.Is/errors.As tests
├── focus_conformance.go      # MODIFY: Convert 24 error sites to *ValidationError
├── focus_conformance_test.go # MODIFY: Update error assertions for new Error() format
├── validation_options.go     # UNCHANGED
├── validation_test.go        # REVIEW: May need Error() format updates
└── manifest.go               # UNCHANGED (uses pbc.ValidationError, different type)
```

**Structure Decision**: All changes are contained within the existing `sdk/go/pluginsdk/` package.
No new files or directories needed. The `manifest.go` file uses `pbc.ValidationError` (protobuf
type), which is distinct from the SDK's `pluginsdk.ValidationError` struct and is not affected.

## Complexity Tracking

No constitution violations to justify.
