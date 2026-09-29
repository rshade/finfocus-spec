# Implementation Plan: FOCUS 1.4 Billing Period and Invoice Detail

**Branch**: `546-focus-14-billing-invoice` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/546-focus-14-billing-invoice/spec.md`

## Summary

Phase 3 of FOCUS 1.4 (issue 540), delivering issue 543. Add `BillingPeriod` (6 columns) and
`InvoiceDetail` (22 FOCUS columns plus `extended_columns`) to `focus.proto`, with two new enums.
`optional double` keeps a zero settlement cost distinct from an omitted one. Per-record rules live
in `sdk/go/testing` and the builders delegate. No delivery RPC.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (client SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1.
No new modules. Currency checks reuse `currency.IsValid`.

**Storage**: N/A (stateless proto messages and validation)

**Testing**: `go test`, `testing.AllocsPerRun`, setter benchmarks, vitest

**Target Platform**: Go plugin SDK and TypeScript client

**Project Type**: Library. Wire contract and SDK helpers.

**Performance Goals**: Simple setters and `IsValid*` status checks stay at 0 allocs/op.
`ValidateBillingPeriod` and `ValidateInvoiceDetail` stay at 0 allocs/op on a valid record.

**Constraints**: Additive proto only. `buf breaking` ignores `focus.proto`; verify with the ignore
entry removed. Cross-row rules are not checked.

**Scale/Scope**: Two messages, two enums, Go and TypeScript builders, docs, JSON-LD.

## Constitution Check

- Proto first: `focus.proto` is edited and `make generate` regenerates Go and TypeScript.
- SDK parity: Go builders and validators, TypeScript builders, conformance in
  `focus14_conformance_test.go`, JSON-LD, and `docs/focus-columns.md`.
- No new billing mode, so no new cross-provider pricing examples.
- New Go files carry the Apache 2.0 header.
- Re-check after design: the same gates hold. No RPC, no new dependency, no removed field.

## Project Structure

### Source Code

```text
proto/finfocus/v1/focus.proto
sdk/go/pluginsdk/billing_period_builder.go
sdk/go/pluginsdk/invoice_detail_builder.go
sdk/go/pluginsdk/invoice_enums.go
sdk/go/testing/invoice_focus14.go
sdk/go/testing/focus14_conformance_test.go
sdk/go/jsonld/serializer.go
sdk/go/jsonld/vocabulary.go
sdk/typescript/packages/client/src/builders/invoice-datasets.ts
docs/focus-columns.md
```

**Structure Decision**: Rules stay in `sdk/go/testing` and the builders delegate, matching
contract commitments. Enum helpers live in `pluginsdk`. The TypeScript `build()` returns a clone
and does not re-implement the Go rule set.
