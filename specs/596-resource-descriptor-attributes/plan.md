# Implementation Plan: Structured Attributes on ResourceDescriptor

**Branch**: `596-resource-descriptor-attributes` | **Date**: 2026-10-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/596-resource-descriptor-attributes/spec.md`

## Summary

Add `google.protobuf.Struct attributes = 12` to `ResourceDescriptor`, so hosts can hand plugins nested inputs
on every cost RPC, not only on `EstimateCost`. Bound the field at 64 KiB of wire size in both descriptor
validators, and raise the tag value limit from 256 to 2048 in both. Add the generic
`pluginsdk.AttributeValue` dotted-path accessor, a Basic-level conformance test that sends nested attributes,
and the regenerated Go and TypeScript bindings. Update every document that describes how properties reach a
plugin or states tag limits. The batch and transport interaction is documented, not enforced
([research R3](./research.md#r3-batch-and-transport-interaction)).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf (`proto.Size`, `structpb`), google.golang.org/grpc,
buf v1.32.1; no new dependencies

**Storage**: N/A

**Testing**: `go test` (table tests, benchmarks, bufconn conformance), vitest for TypeScript

**Target Platform**: Plugin SDK library (Linux, macOS, and Windows plugin binaries)

**Project Type**: Protocol specification plus SDK libraries

**Performance Goals**: `ValidateResourceDescriptor` without attributes stays at 0 allocs/op.
`AttributeValue` makes 0 allocs/op.

**Constraints**: Additive proto only (`buf breaking` clean). Both validation layers stay in agreement.

**Scale/Scope**: 1 proto field, 2 constants changed, 2 constants and 1 sentinel added, 1 accessor,
1 conformance test, and about 12 docs audited

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Note |
|-----------|--------|------|
| I. Proto first | PASS | The proto change and `make generate` come before any SDK code. |
| II. Multi-provider | PASS | The field is generic, with no provider schema. The Pulumi secret signature is an example in a host rule, not a field shape. |
| III. Spec consumes | PASS | No pricing logic is added. |
| IV. Separation | PASS | Host population and redaction are core work (finfocus#1525). The spec states only the rule. |
| V. Test first | PASS | Boundary, accessor, and conformance tests are written before the code they cover. |
| VI. Backward compatibility | PASS | Additive field 12. A raised limit accepts strictly more input. |
| VII. Docs | PASS | FR-013 to FR-015 cover every doc found in [R9](./research.md#r9-documentation-inventory-fr-013-to-fr-015). |
| VIII. Performance | PASS | The size check is guarded by nil, and existing zero-alloc benchmarks guard it. The accessor gets a benchmark. |
| X. Patterns | PASS | Follows the `ErrTagValueTooLong` / `NewContractError` and `status.Errorf` patterns already in each validator. |
| XI. Headers | PASS | New Go files carry the Apache 2.0 header. |
| XII. Capabilities | N/A | No new capability or RPC. |
| XIII. SDK sync | PASS | The TypeScript bindings are regenerated, and a round-trip test is added. |
| XIV. Doc integrity | PASS | New exported symbols get godoc, and both READMEs list them. |

Post-design re-check: PASS, with no violations and nothing in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/596-resource-descriptor-attributes/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── proto.md
│   └── go-sdk.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto            # attributes = 12, plus the tags comment
sdk/go/proto/finfocus/v1/costsource.pb.go     # regenerated
sdk/typescript/packages/client/src/generated/ # regenerated
sdk/go/pluginsdk/batch.go                     # MaxTagValueLength, MaxAttributesBytes, validator
sdk/go/pluginsdk/attributes.go                # AttributeValue (new)
sdk/go/pluginsdk/attributes_test.go           # accessor table tests and benchmark (new)
sdk/go/pluginsdk/batch_test.go                # boundary tests
sdk/go/testing/contract.go                    # constants, sentinel, validator, contract-suite cases
sdk/go/testing/contract_test.go               # boundary tests
sdk/go/testing/rpc_correctness.go             # GetProjectedCostWithAttributes test
sdk/go/testing/*_test.go                      # delivery test and conformance test counts
sdk/typescript/packages/client/test/          # attributes round-trip test
docs/PROPERTY_MAPPING.md, PLUGIN_DEVELOPER_GUIDE.md, sdk/go/{pluginsdk,testing}/README.md,
README.md, sdk/typescript/README.md, CLAUDE.md, AGENTS.md, and the other docs from R9
```

**Structure Decision**: Existing packages only. The accessor gets its own file in `pluginsdk`, because it is
unrelated to batching.

## Complexity Tracking

No violations.
