# Implementation Plan: Allocation Row Provenance

**Branch**: `594-allocation-row-provenance` | **Date**: 2026-10-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/594-allocation-row-provenance/spec.md`

## Summary

Add three optional string fields to `AllocationRow` (`allocated_method_id` 7, `allocated_method_details` 8,
`allocated_resource_id` 9), mirroring `FocusCostRecord` fields 61-63. The response validator rejects a row
with a method id and no source resource id. The conformance suite gains a named scenario for the rule, the
reference allocator fills the fields, and docs and TypeScript tests cover the round trip. Field 10 is held
by comment for a later `LineageNode`.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new dependencies

**Storage**: N/A (stateless contract fields and validation)

**Testing**: `go test` (sdk/go/testing, sdk/go/pluginsdk, sdk/go/internal/refalloc), vitest in
`sdk/typescript/packages/client`, `buf breaking`

**Target Platform**: Plugin SDK library

**Project Type**: Proto-first spec repo with Go and TypeScript SDKs

**Performance Goals**: `ValidateAllocateResponse` stays allocation-free on valid input; no new per-row
allocation; benchmark compared with `main`

**Constraints**: Additive only; conservation and currency rules unchanged; no request-side or response-level
field (issue #579 owns those)

**Scale/Scope**: One message, one validator rule, one conformance scenario, one reference-allocator change

## Constitution Check

- I. Proto first: change `allocation.proto`, run `make generate`, then SDK code. PASS.
- III. Spec does not calculate: only carries values the allocator reports. PASS.
- V. Test first: tasks order failing tests before implementation. PASS.
- VI. Backward compatibility: new optional fields 7-9, no renumbering; `buf breaking` gate. PASS.
- VIII. Performance: validator rule is two string-length checks per row; benchmark gate. PASS.
- X. Follow established patterns: `ErrInvalidAllocateResponse` wrap in `validateAllocationRow`, scenario naming
  as in 052. PASS.
- XI. Headers: no new source files except none; spec docs only. PASS.
- XIII. SDK sync: Go SDK (validator, refalloc, conformance), TypeScript generated bindings plus a client
  test. PASS.
- XIV. Docs integrity: `docs/allocator.md` Rows section and pluginsdk README updated. PASS.

Post-design re-check: unchanged, PASS.

## Project Structure

```text
specs/594-allocation-row-provenance/   spec, plan, research, data-model, contracts, quickstart, tasks
proto/finfocus/v1/allocation.proto     AllocationRow fields 7-9, comment for 10
sdk/go/proto/**                        regenerated
sdk/typescript/packages/client/src/generated/**  regenerated
sdk/go/testing/allocation.go           provenance rule in validateAllocationRow
sdk/go/testing/allocator_conformance.go  row_provenance scenario
sdk/go/internal/refalloc/refalloc.go   fills method id and source resource id
sdk/typescript/packages/client/test/allocator.test.ts  round-trip test
docs/allocator.md                      Rows section
```

**Structure Decision**: Existing layout; no new packages.

## Complexity Tracking

No violations.
