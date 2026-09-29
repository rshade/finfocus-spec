# Implementation Plan: FOCUS 1.4 Contract Commitment Columns

**Branch**: `545-focus-14-contract-commitment` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/545-focus-14-contract-commitment/spec.md`

## Summary

Phase 2 of FOCUS 1.4 (issue 540), delivering issue 542. `ContractCommitment` gains fields 13-30
and seven enums. Three `optional double` fields keep null distinct from zero. Applicability is a
JSON string. `ValidateContractCommitment` grows the per-record 1.4 rules and stays allocation-free
on valid input. `FormatContractApplied` emits the ContractApplied object. `WithContractApplied`
stays and is deprecated.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (client SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1.
No new modules. JSON object checks are a zero-allocation scan, not `encoding/json`.

**Storage**: N/A (stateless proto fields and validation)

**Testing**: `go test`, `testing.AllocsPerRun`, enum benchmarks, vitest

**Target Platform**: Go plugin SDK and TypeScript client

**Project Type**: Library. Wire contract and SDK helpers.

**Performance Goals**: Enum membership and `ValidateContractCommitment` stay at 0 allocs/op on
valid input.

**Constraints**: Additive proto only. `buf breaking` ignores `focus.proto`; the change adds fields
and enums. Deprecated `WithContractApplied` keeps its wire value.

**Scale/Scope**: One message, seven enums, Go and TypeScript helpers, docs, JSON-LD.

## Constitution Check

- Proto first: `focus.proto` is edited and `make generate` regenerates Go and TypeScript.
- SDK parity: Go builder and validators, TypeScript builder and `formatContractApplied`,
  conformance in `focus14_conformance_test.go`, JSON-LD, and `docs/focus-columns.md`.
- No new billing mode, so no new cross-provider pricing examples.
- New Go files carry the Apache 2.0 header.

## Project Structure

### Source Code

```text
proto/finfocus/v1/focus.proto
sdk/go/pluginsdk/contract_commitment_builder.go
sdk/go/pluginsdk/contract_commitment_enums.go
sdk/go/pluginsdk/contract_applied.go
sdk/go/testing/commitment_focus14.go
sdk/go/testing/focus14_conformance_test.go
sdk/go/jsonld/serializer.go
sdk/go/jsonld/vocabulary.go
sdk/typescript/packages/client/src/builders/contract-commitment.ts
docs/focus-columns.md
```

**Structure Decision**: Rules stay in `sdk/go/testing` and the builder delegates, matching the
supplemental-dataset split. Enum helpers live in `pluginsdk` because `testing` cannot import it.
