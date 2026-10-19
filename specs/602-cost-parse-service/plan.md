# Implementation Plan: Cost Parse Service

**Branch**: `recommendations_history` | **Date**: 2026-10-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/602-cost-parse-service/spec.md`

**Status**: Implemented and validated on 2026-10-18. All constitution gates re-evaluated;
see [final evidence and requirement reconciliation](quickstart.md#final-validation-and-requirement-reconciliation).

Setup resolves the feature identifier as `602-cost-parse-service`; the actual Git branch remains
`recommendations_history`. The active directory is persisted in `.specify/feature.json` independently of Git naming.

## Summary

Add an optional CostParseService that turns cloud-native or Kubernetes input into existing priceable descriptors,
with source identities, unresolved-property paths, separate warnings, and advisory expiration. Define the contract
first, generate both SDKs, integrate the established optional-service infrastructure, and provide shared validation,
a separate mock, provider-neutral conformance, TypeScript client/REST support, and documentation. Vendor parsers,
Terraform/Pulumi ingestion, pricing calculations, and core selection implementation are outside this change.

Research resolves automatic capability discovery, strict oneof conflicts at decode time, plugin-owned filesystem
limits, dynamic unknown-property paths, gateway registration, and the distinction between registration and successful
handler execution. Details and alternatives are in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.27.1; Protocol Buffers v3; TypeScript with repository compiler tooling; Node 24.21.0+.

**Primary Dependencies**: Existing protobuf v1.36.12, gRPC v1.84.0, Connect Go v1.21.0, grpchealth v1.5.0,
protobuf-es and Connect-ES. Use repository-managed Buf and existing protojson/protowire; no new dependencies.

**Storage**: Stateless contract and validators. File reads are plugin responsibilities; no persistence is added.

**Testing**: Go external-package table tests, bufconn conformance, injected-listener gRPC/Connect integration,
TypeScript Vitest client/gateway tests, Buf lint/breaking, registry schema drift, documentation checks, benchmarks.

**Target Platform**: Existing plugin server platforms; browser/Node TypeScript clients; Node REST middleware.

**Project Type**: Protocol specification and multi-language plugin SDK.

**Performance Goals**: Existing validator operations retain their behavior and show no repeatable A/B slowdown.
New format/enum checks target zero allocations; collection validation may allocate in proportion to input size.
Each conformance call has a 10-second test deadline; this bounds test execution rather than promising vendor latency.
Benchmark valid, rejected, and near-limit parsing requests and responses with allocation reporting.

**Constraints**: Preserve existing fields and numbers; options encoded size <=65,536 bytes; format length 1–32 bytes;
input bounded by positive advertised byte limit and independent transport receive caps. No streaming, generated edits,
external parser dependencies, or SDK filesystem reads. Root containment remains a plugin contract obligation.

**Scale/Scope**: One service/method, six new messages, one severity enum, capability 19, three metadata keys,
two SDKs, one gateway extension, separate parser mock/harness/conformance. No universal vendor-format implementation.

## Constitution Check

*GATE: Evaluated before research and re-evaluated after Phase 1 design.*

| Principle | Pre-research gate | Post-design evidence / implementation gate |
| --- | --- | --- |
| I. Proto first | Pass: contract precedes adapters | Add parse.proto before implementation; generate all bindings; validate every message |
| II. Provider consistency | Pass: no new billing model | Generic descriptors and configurable neutral fixtures; format names remain open strings |
| III. No pricing calculation | Pass | Parsing returns descriptors only |
| IV. Separation | Pass | Vendor parsing and host selection stay outside SDK; no dependencies added |
| V. Test first | Pass: required sequence recorded | Write failing protocol/conformance tests first, then proto and generation, then implementation |
| VI. Compatibility | Pass: additive feature | Buf breaking check; preserve SourceFormat, descriptor and existing service contracts |
| VII. Documentation | Pass | Inline proto/godoc plus root, Go, TS and parsing documentation updated together |
| VIII. Performance | Pass | New benchmarks and prebuilt-binary A/B validation baseline; no unrelated hot-path edits |
| IX. Validation | Pass | Metadata, request, response, transport and conformance checks; no new metrics system |
| X. Patterns | Pass | Reuse optionalServices, scorer limits, error normalization and bufconn patterns |
| XI. Copyright | Pass | Apache headers on every new Go/proto source, including contract prototype |
| XII. Capabilities | Pass after code inspection | Auto-infer parsing; explicit parser-only override uses the constitution's permitted advanced case |
| XIII. SDK parity | Pass | Generate Go/TS, expose TypeScript client and add REST route, test transports |
| XIV. Documentation integrity | Pass | Compiling examples, exports, README coverage and Markdown checks in implementation validation |

Post-design result: all gates pass for the planned design; no constitutional exception is required. These are design
gates, not claims that implementation tests have run. The original inference sentence is superseded by research
section 1. A generated stub cannot be proven functional by startup introspection: health means registered provider,
and Unimplemented remains authoritative at invocation time.

## Project Structure

### Documentation (this feature)

```text
specs/602-cost-parse-service/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── contracts/
    ├── parse.proto
    └── sdk-and-errors.md
```

[tasks.md](tasks.md) records all 57 completed implementation tasks.

### Source Code (repository root)

```text
proto/finfocus/v1/
├── parse.proto                           # new canonical service
└── enums.proto                           # additive capability 19
sdk/go/pluginsdk/
├── parse.go                              # delegates, options, adapters, common dispatch
├── parse_codec.go                        # strict binary input exclusivity
├── sdk.go                                # provider + optionalServices registration
├── plugin_info.go                        # inference, limits and capability bounds
├── capability_compat.go                  # supports_cost_parse
├── usage_source.go                       # inferred-only startup diagnostic
└── parse*_test.go                        # transport, codec, metadata, benchmarks
sdk/go/testing/
├── parse.go                              # shared request/response validation
├── parse_limits.go                       # ParseLimits and metadata helpers
├── parse_mock.go                         # MockCostParser
├── parse_conformance.go                  # harness and structural scenarios
└── parse*_test.go                        # independent negative cases and benchmarks
sdk/go/registry/
└── schema_test.go                        # capability/schema drift assertions
sdk/typescript/packages/
├── client/src/clients/cost-parse.ts
├── client/src/index.ts
├── client/test/cost-parse.test.ts
├── middleware/src/gateway.ts
└── middleware/test/                      # parsing route + JSON/error cases
schemas/
├── plugin_manifest.schema.json
└── plugin_registry.schema.json
docs/cost-parse.md
README.md
sdk/go/README.md
sdk/go/pluginsdk/README.md
sdk/go/testing/README.md
sdk/typescript/README.md
```

Generated Go files under sdk/go/proto and TypeScript bindings under client/src/generated change only through root
`make generate`. Keep the Connect generator pinned to v1.21.0; do not use the stale TypeScript-local Buf config.

**Structure Decision**: Extend existing protocol/SDK packages. Shared testing code must not import pluginsdk.
Keep decode guards and all parsing loops out of existing descriptor-validation hot paths.

## Phase 0: Research Results

[research.md](research.md) resolves all research questions using existing repository code and installed dependency
sources. No external technology selection or dependency upgrade is needed. Decisions cover capability inference,
severity, metadata strictness, oneof decoding, paths/unknowns, limits, unsupported handlers, clients, mocks, and performance.

## Phase 1: Design and Contracts

[contracts/parse.proto](contracts/parse.proto) is the proposed additive wire contract, not a generated file or a
second runtime source of truth. Copy it into the canonical proto tree during implementation and generate there.
[contracts/sdk-and-errors.md](contracts/sdk-and-errors.md) defines exported surfaces, ownership, error outcomes,
transport guard behavior and conformance parameters. [data-model.md](data-model.md) specifies invariants and boundaries.
[quickstart.md](quickstart.md) provides validation commands and expected outcomes after implementation.

Implementation sequencing for task generation:

1. Capture existing validation benchmark binaries. Write failing contract/conformance and raw conflicting-oneof tests;
   missing new generated symbols are an expected initial failure. Add the canonical proto and capability, regenerate,
   and make protocol-shape tests pass before writing provider logic.
2. Add ParseLimits/metadata helpers and validators in testing, then pluginsdk delegates. Test limits, paths, unknown
   traversal, descriptor constraints, nil entries, and response validity; add benchmarks.
3. Add provider dispatch, strict parse-request codec wrappers, optionalServices registration, capability inference,
   legacy compatibility, metadata startup validation, and parser-only diagnostics. Verify missing and stub providers,
   handler errors/details, and health registration using injected listeners.
4. Add separate mock, configurable harness and conformance fixtures. Prove rejecting raw conflicts in both tag orders,
   advertised mode handling, source uniqueness, unknown-value absence, descriptor validity, and 100-call determinism.
5. Add TypeScript client/limits helpers/exports and optional REST gateway service. Test both input arms, binary bytes,
   timestamp JSON, unsupported modes/formats, missing gateway client, and both-member JSON rejection.
6. Update both schema enums and drift/bounds tests, public docs and compiling examples. Run final validation and
   repeated A/B benchmarks, inspecting generation diff for unrelated changes.

All content-size checks use raw content bytes. Binary/JSON transport caps remain independent and may reject an
oversized encoded message before a plugin sees it; docs must explain the effective minimum and base64 overhead.
Plugin-specific option allowlists, path read accounting, and symlink containment cannot be enforced by generic validators.

## Complexity Tracking

No constitution violations. A narrowly scoped decode guard is necessary because ordinary oneof decoding loses evidence
of conflicting inputs. It delegates unrelated messages and introduces no global codec registration or generated edits.
Health describes registered interfaces; startup cannot safely distinguish inherited generated methods from working code.

## Implementation gate re-evaluation

All fourteen constitution gates pass with implementation evidence in quickstart.md: test-first protocol and
behavioral regressions; additive generated Go/TS bindings and Buf compatibility; neutral descriptors without
pricing calculations or vendor parsing; shared bounded validators and local codecs; automatic capabilities
with parser-only overrides; synchronized schemas/client/gateway/docs; Apache headers; exported godoc above
80%; compiling examples; and repeated baseline/candidate benchmarks with unchanged existing allocations.
Root containment and consumed-file accounting remain provider responsibilities and are demonstrated by the
separate synthetic mock. The original requirements-quality checklist remains read-only and unchanged.
