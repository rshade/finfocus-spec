# Implementation Plan: On-Demand Equivalent for Spot Results

**Branch**: `main` (working tree) | **Date**: 2026-10-18 | **Spec**: [spec.md](spec.md)

**Input**: Issue #646 and the feature specification in this directory. Implementation already
exists in the working tree; this plan records its design and verification gates for review.

## Summary

Add a presence-tracked on-demand monthly counterpart to projected and estimated DYNAMIC
results. The plugin supplies both prices in one response. The contract adds optional doubles
at field 18 of `GetProjectedCostResponse` and field 8 of `EstimateCostResponse`. Regenerate
Go and TypeScript bindings. One shared rule in `sdk/go/testing` checks the field; both
`pluginsdk` validators and both conformance validators call it. Response options, mock
support, wire tests, and docs complete the change. Core savings math remains downstream.

## Technical Context

**Language/Version**: Go 1.27.1 (per `go.mod`), Protocol Buffers v3, TypeScript client

**Primary Dependencies**: Existing protobuf, gRPC, Buf v1.32.1, ConnectRPC, and Vitest;
no new dependencies

**Storage**: N/A; response-only wire field and in-memory helpers

**Testing**: Go table tests, gRPC `bufconn`, old-decoder protobuf test, TypeScript integration,
validator benchmarks, `make validate`, Buf lint and breaking checks

**Target Platform**: Plugin and host libraries using Go or TypeScript bindings

**Project Type**: Protobuf contract with generated SDKs and conformance tooling

**Performance Goals**: Both response validators and the shared rule retain 0 allocs/op on
valid and absent-field paths

**Constraints**: Additive wire change. Presence distinguishes unknown from real zero. A
present value is finite, non-negative, and DYNAMIC-only. No ordering or `price_options`
comparison. Do not edit generated code by hand, add request or batch fields, add dependencies,
or change `SpecVersion`, release-please files, or spot-risk semantics.

**Scale/Scope**: Two response fields, two generated bindings, one shared rule, two option
helpers, four validator call sites, one mock field, focused tests and docs

## Constitution Check

*Evaluated before design and rechecked after contracts; no exceptions needed.*

| Principle | Result | Design consequence |
| --- | --- | --- |
| I. Proto first | PASS | Tests define wire behavior; proto is canonical; bindings use `make generate`. |
| II. Multi-provider consistency | PASS | Equivalent is resource-neutral; no new billing mode. |
| III. Spec consumes | PASS | No savings calculator or second pricing request. |
| IV. Separation of concerns | PASS | Core computes and displays savings downstream. |
| V. Test first | PASS | Validation and presence tests precede their implementation. |
| VI. Backward compatibility | PASS | Unused field numbers, optional presence, `buf breaking` gate. |
| VII. Documentation | PASS | Proto comments, SDK READMEs, and developer guide explain the contract. |
| VIII. Performance | PASS | Nil pointer fast path and allocation benchmarks. |
| IX. Validation | PASS | Shared sentinels and rule for SDK and conformance. |
| X. Established patterns | PASS | Mirrors `region_prices` rule ownership and optional-double convention. |
| XI. Copyright headers | PASS | New Go source files carry the Apache 2.0 header. |
| XII. Capability declaration | PASS | No capability or provider interface changes. |
| XIII. SDK synchronization | PASS | Go and TypeScript bindings regenerate from the same proto. |
| XIV. Documentation integrity | PASS | Runnable example and Markdown lint gate. |

## Design and Data Flow

1. A plugin selects DYNAMIC pricing and, when known, attaches its STANDARD on-demand monthly
   counterpart. Both amounts share the response currency and monthly basis.
2. An absent pointer means unknown. A present zero survives gRPC, nested batch serialization,
   and the generated TypeScript optional property.
3. `ValidateOnDemandEquivalent(*float64, FocusPricingCategory)` accepts absence, rejects
   NaN/Inf or negative values, then rejects non-DYNAMIC categories. The SDK and conformance
   validators call the same rule; `pluginsdk` aliases its sentinels.
4. Both response options copy the value without panicking. The mock copies a configured
   pointer only for DYNAMIC responses. The consistency helper warns about missing data
   without making DYNAMIC responses invalid.
5. The value is advisory. It is never summed into the selected cost or cross-checked with
   `price_options`; consumers handle negative savings and a zero denominator.

## Project Structure

### Documentation and contracts

```text
specs/603-on-demand-equivalent/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── contracts/response-and-sdk.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

### Source and tests

```text
proto/finfocus/v1/costsource.proto
sdk/go/proto/finfocus/v1/costsource.pb.go                  # generated
sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts # generated
sdk/go/testing/on_demand_equivalent.go
sdk/go/testing/harness.go
sdk/go/testing/mock_plugin.go
sdk/go/pluginsdk/validation.go
sdk/go/pluginsdk/helpers.go
sdk/go/on_demand_equivalent_test.go
sdk/go/testing/on_demand_equivalent_test.go
sdk/go/pluginsdk/on_demand_equivalent_test.go
sdk/go/pluginsdk/example_test.go
sdk/typescript/packages/client/test/integration.test.ts
sdk/typescript/packages/client/test/mocks/handlers.ts
```

**Structure Decision**: Keep the shared rule in `sdk/go/testing`, as with `region_prices`,
and delegate from `pluginsdk`. Keep options and mock behavior in existing files. No new
runtime package or RPC is needed.

## Verification

1. Check generated Go `*float64` fields and optional TypeScript number properties.
2. Run `buf lint` and `buf breaking --against '.git#branch=main'`.
3. Run `go test ./sdk/go/...`, including both validators' edge matrix, wire presence,
   old-decoder behavior, legacy mock, default mock, and runnable example.
4. Build and test the TypeScript client.
5. Compare valid-path validator benchmarks before and after with `-benchmem`.
6. Run `make validate` and check the final diff for generated drift and whitespace.

The working-tree implementation passed Buf lint and `buf breaking --against
'.git#branch=main'`, confirming an additive wire change. The TypeScript client passed
207 tests. Both response validators remain at 0 B/op and 0 allocs/op. Baseline
estimate validation measured 1.852 ns/op; the current run measured 2.398 ns/op.
Baseline projected validation measured 5.463 ns/op; the current run measured
6.278 ns/op. These are single-run timings on a variable QEMU environment;
allocation counts are the relevant performance gate.
`make validate` passed with the existing cached `yamllint` executable on `PATH`;
Go lint reported 0 issues, Markdown lint reported 0 issues, and the schema and
example validation steps passed. `git diff --check` was clean.

## Complexity Tracking

No constitution violations or new abstractions requiring exceptions.
