# Implementation Plan: Standardized Cost Allocation Lineage Metadata

**Branch**: `055-cost-allocation-lineage` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/055-cost-allocation-lineage/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add an optional, pass-through lineage chain to the cost contract. `enums.proto` gains a
`LineageNodeType` classification enum. `costsource.proto` gains a recursive `LineageNode`
message (type, id, name, parent link, attributes) and two new optional fields:
`ResourceDescriptor.lineage = 11` and `ActualCostResult.lineage = 9` (field 8 is the existing
`expires_at` caching hint and is not touched). Both fields' comments state the anti-guess
boundary: no inferred parents, no provider-API validation, partial chains valid, disagreement
with the flat account fields not an error. Generated Go and TypeScript bindings are
regenerated with `make generate`. A fluent `pluginsdk.LineageBuilder` assembles caller-supplied
chains (leaf plus current-top tracking, so attributes land on the most recently added node)
with unit and JSON round-trip tests. No new RPC, no capability, no consistency checks.

## Technical Context

**Language/Version**: Go 1.27.1 (go.mod)

**Primary Dependencies**: Existing `google.golang.org/protobuf` (protojson, protocmp in tests).
No new dependencies.

**Storage**: N/A (wire contract and in-memory builder only)

**Testing**: `go test ./sdk/go/...`; focused runs under `sdk/go/pluginsdk`

**Target Platform**: Protocol specification plus Go/TypeScript SDKs used by FinFocus plugins
and hosts

**Project Type**: Protocol repository with generated SDK bindings

**Performance Goals**: Builder is O(1) per call (leaf and top pointers, no chain re-walking).
No new hot path; lineage is optional payload on existing messages.

**Constraints**: Never edit generated code (`sdk/go/proto/`, TypeScript `generated/`); regenerate
via `make generate`. Never reuse `ActualCostResult` field 8. Proto comments follow the house
style (REQUIRED/OPTIONAL notes, semantics, examples). No consistency check against
`billing_account_id`/`sub_account_id`. No unlimited-depth promise in docs.

**Scale/Scope**: Two proto files, four regenerated files, one new SDK source file
(`sdk/go/pluginsdk/lineage_builder.go`), one new test file
(`sdk/go/pluginsdk/lineage_builder_test.go`).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Proto first**: Pass. The change begins in `proto/finfocus/v1/enums.proto` and
  `proto/finfocus/v1/costsource.proto`. Generated code is only regenerated, never hand-edited.
  No PricingSpec change, so no JSON schema update is owed.
- **II. Multi-provider consistency**: Pass. Node classifications carry AWS, Azure, GCP, and
  Kubernetes examples in the enum comments. `LineageNode` is provider-agnostic.
- **III. The spec does not calculate**: Pass. Lineage is reported pass-through data. No
  pricing, allocation, or hierarchy derivation is added.
- **IV. Separation of concerns**: Pass. Contract plus plugin-SDK builder only. No host
  application logic, no drill-down UI.
- **V. Test-first**: Pass with note. This is a retroactive pipeline: implementation already
  exists with tests (`lineage_builder_test.go`), including protojson round-trip coverage that
  locks the wire shape. In a forward run, the round-trip tests would have preceded the proto
  edit.
- **VI. Protobuf compatibility**: Pass. Both fields use never-used numbers (11 and 9). Field 8
  (`expires_at`) is untouched. `buf breaking` is satisfied: additions only.
- **VII. Documentation currency**: Pass. Proto comments document every new element inline in
  house style. This spec directory is the same-change documentation.
- **VIII. Performance**: Pass with note. The builder is pointer assembly, no per-call chain
  walks. No new core SDK logic, so no new benchmark; payload growth on 1000-result pages is
  documented as an edge case in the spec.
- **IX. Observability**: Pass. No new logs or metrics; an absent or partial chain is visible on
  the message itself.
- **X. Established patterns**: Pass. Enum style matches existing `enums.proto` blocks
  (UNSPECIFIED default, per-value comments). Builder style matches `focus_builder.go` (fluent
  `With*` methods, `Build()` terminal). Test style matches `manifest_test.go`
  (`cmp.Diff` + `protocmp.Transform()`).
- **XIII. Multi-language SDK sync**: Pass. `make generate` regenerates Go and TypeScript
  bindings in one step. No new RPC, so no client-wrapper obligation.
- **XIV. Documentation integrity**: Pass. Exported builder symbols carry godoc comments.
  `specs/055-cost-allocation-lineage/` carries status, plan, and tasks.

## Project Structure

### Documentation (this feature)

```text
specs/055-cost-allocation-lineage/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── lineage-chain.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/
├── enums.proto             # LineageNodeType enum (appended)
└── costsource.proto        # LineageNode message; ResourceDescriptor.lineage = 11;
                            # ActualCostResult.lineage = 9

sdk/go/proto/finfocus/v1/                          # regenerated (do not hand-edit)
sdk/typescript/packages/client/src/generated/      # regenerated (do not hand-edit)

sdk/go/pluginsdk/
├── lineage_builder.go        # LineageBuilder (new)
└── lineage_builder_test.go   # chain order, metadata placement, partial chain,
                              # custom/deep chain, JSON round trips (new)
```

**Structure Decision**: Extend the two existing proto files in place; regenerate bindings into
their existing generated trees; add one builder and one test file alongside `focus_builder.go`
in the existing `pluginsdk` package. No new package, no new proto file, no schema changes under
`schemas/`.

## Complexity Tracking

No constitution violation requires justification. The recursive `LineageNode.parent` encoding
is the simplest representation of an ordered chain and is bounded in documentation, not by a
depth field the contract does not need.

## Post-Design Re-check

Design artifacts add no RPC, no capability, no consistency rule, no depth counter, and no
shared helper beyond the one builder. Gates above still pass. Rejected follow-ons, if a later
stage proposes them: lineage on projections/estimates/batch wrappers, a conformance check
against `billing_account_id`, a stored `depth` field, a `WalkLineage` helper in the contract,
and reusing field 8.
