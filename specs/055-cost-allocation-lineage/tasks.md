# Tasks: Standardized Cost Allocation Lineage Metadata

**Input**: Design documents from `/specs/055-cost-allocation-lineage/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/lineage-chain.md

**Tests**: Required. The issue's success criteria are round-trip tests, and constitution V
requires tests for proto changes.

**Organization**: Tasks are grouped by user story. The proto edit and regeneration are
foundational: every story consumes the generated types. This file documents an
already-completed implementation, so every task is marked done and maps to a real file or
test.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `proto/finfocus/v1/enums.proto` and `proto/finfocus/v1/costsource.proto`
- `sdk/go/pluginsdk/lineage_builder.go` and `sdk/go/pluginsdk/lineage_builder_test.go`
- Generated (never hand-edited): `sdk/go/proto/finfocus/v1/`,
  `sdk/typescript/packages/client/src/generated/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Keep the slice inside the existing module and toolchain

- [x] T001 Confirm `go.mod` stays at Go 1.27.1 with no new dependency; tests reuse the
  existing `google.golang.org/protobuf` protojson/protocmp and `github.com/google/go-cmp`
  entries
- [x] T002 Confirm `bin/buf` resolution via `mise` (buf 1.32.1) and that `make generate`
  regenerates Go and TypeScript bindings in one step

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The contract change and regenerated bindings every story depends on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Append the `LineageNodeType` enum to `proto/finfocus/v1/enums.proto` with values
  UNSPECIFIED=0, ORGANIZATION=1, ORGANIZATIONAL_UNIT=2, BILLING_ACCOUNT=3, SUB_ACCOUNT=4,
  RESOURCE_GROUP=5, RESOURCE=6, CUSTOM=7, in the existing enum comment style, noting that
  chains may skip, repeat, or stop at levels and that ids are not validated against provider
  APIs (`specs/055-cost-allocation-lineage/data-model.md`)
- [x] T004 Add the `LineageNode` message to `proto/finfocus/v1/costsource.proto` before
  `ResourceDescriptor`: `type`=1, `id`=2, `name`=3, `parent`=4 (leaf-to-root singly-linked
  list, nil ends the chain), `metadata`=5. Comments state the pass-through semantics and the
  recursion-limit caveat (`specs/055-cost-allocation-lineage/research.md`)
- [x] T005 [P] Add `LineageNode lineage = 11;` to `ResourceDescriptor` in
  `proto/finfocus/v1/costsource.proto` (fields 1–10 untouched), OPTIONAL, with the
  anti-guess comment block
- [x] T006 [P] Add `LineageNode lineage = 9;` to `ActualCostResult` in
  `proto/finfocus/v1/costsource.proto`. Field 8 (`expires_at`, spec 045) MUST NOT be touched;
  fields 1–8 untouched
- [x] T007 Run `buf lint` and confirm the edited protos pass
- [x] T008 Run `make generate` and verify `LineageNode` appears in
  `sdk/go/proto/finfocus/v1/costsource.pb.go`, `sdk/go/proto/finfocus/v1/enums.pb.go`,
  `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts`, and
  `.../enums_pb.ts`. Hand-edit nothing under generated trees

**Checkpoint**: Generated Go and TypeScript bindings expose `LineageNode`, `LineageNodeType`,
and both lineage fields. Story work can start.

---

## Phase 3: User Story 1 - Plugin Author Reports the Hierarchy With a Cost (Priority: P1) 🎯 MVP

**Goal**: A plugin can assemble and attach an ordered chain — full, partial, custom, or deep —
to a cost result or a resource descriptor, with per-level attributes on the intended level.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -count=1 -run 'TestLineageBuilder'`

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T009 [P] [US1] Add `TestLineageBuilderChainOrder` in
  `sdk/go/pluginsdk/lineage_builder_test.go`: build the issue's four-level chain (resource
  `i-0abc123def456`, sub-account `123456789012`, billing account `999988887777`,
  organization `o-exampleorg`) and assert walking `Parent` pointers yields exact types, ids,
  and names leaf-to-root, with a nil parent at the root
- [x] T010 [P] [US1] Add table-driven `TestLineageBuilderMetadataPlacement` in
  `sdk/go/pluginsdk/lineage_builder_test.go`: metadata before any `WithParent` lands on the
  resource node; metadata after `WithParent(SUB_ACCOUNT)` lands on the sub-account, not the
  billing account added later; metadata on every level
- [x] T011 [P] [US1] Add `TestLineageBuilderPartialChain` in
  `sdk/go/pluginsdk/lineage_builder_test.go`: resource-only chain builds a non-nil leaf of
  type RESOURCE with a nil parent
- [x] T012 [P] [US1] Add `TestLineageBuilderCustomAndDeepChain` in
  `sdk/go/pluginsdk/lineage_builder_test.go`: a ten-level chain (nine CUSTOM parents plus an
  ORGANIZATION top) walks in order via a cycle-guarded helper

### Implementation for User Story 1

- [x] T013 [US1] Create `sdk/go/pluginsdk/lineage_builder.go` with `LineageBuilder` tracking
  both `leaf` and `top` pointers (O(1) per call, fixing the issue sketch's chain re-walk),
  `NewLineageBuilder(resourceID, resourceName)` creating the RESOURCE leaf,
  `WithParent(nodeType, id, name)` linking above the current top, `WithMetadata(key, value)`
  targeting the current top, and `Build()` returning the leaf with no validation. Godoc states
  the pass-through semantics, matching `focus_builder.go` style

**Checkpoint**: Plugins can build and attach any chain shape. US1 tests pass.

---

## Phase 4: User Story 2 - Host Reads the Chain Back in Order (Priority: P2)

**Goal**: A chain survives transport with order, ids, names, classifications, and per-level
attributes intact, and disagreement with the flat account fields changes nothing.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -count=1 -run 'TestLineageBuilderJSONRoundTrip|TestLineageBuilderIntegrationRoundTrip|TestActualCostResultLineageDisagreementRoundTrip'`

### Tests for User Story 2 ⚠️

- [x] T014 [P] [US2] Add `TestLineageBuilderJSONRoundTrip` in
  `sdk/go/pluginsdk/lineage_builder_test.go`: an `ActualCostResult` carrying the four-level
  chain round-trips through `protojson` with full equality (`cmp.Diff` +
  `protocmp.Transform()`, the `manifest_test.go` pattern)
- [x] T015 [P] [US2] Add `TestLineageBuilderIntegrationRoundTrip` in
  `sdk/go/pluginsdk/lineage_builder_test.go` with subtests for an `ActualCostResult` and a
  `ResourceDescriptor` each carrying the four-level chain, plus an explicit subtest asserting
  order, ids, names, and per-level metadata (`region` on the leaf, `cost_center` on the
  sub-account) after the round trip
- [x] T016 [P] [US2] Add `TestActualCostResultLineageDisagreementRoundTrip` in
  `sdk/go/pluginsdk/lineage_builder_test.go`: a chain ending at a CUSTOM node beside a
  `FocusCostRecord` naming different `BillingAccountId`/`SubAccountId` round-trips without
  error, and the FOCUS fields survive untouched (pass-through, no consistency check)

### Implementation for User Story 2

- [x] T017 [US2] Confirm no host-side or SDK code path infers, validates, or cross-checks
  lineage; the proto comments on `LineageNode` and both lineage fields are the only statement
  of the rule set and they match `specs/055-cost-allocation-lineage/contracts/lineage-chain.md`

**Checkpoint**: Round-trip matrix in `contracts/lineage-chain.md` is locked by passing tests.

---

## Phase 5: User Story 3 - Hosts That Ignore Lineage Are Unaffected (Priority: P3)

**Goal**: No-lineage messages are byte-identical to before; pre-slice readers ignore the new
field; no plugin is forced to change.

**Independent Test**: The full pre-existing SDK suite passes unmodified, and the new fields
are optional additions on never-used numbers.

### Tests for User Story 3 ⚠️

- [x] T018 [US3] Run `go test ./sdk/go/... -count=1` and confirm every pre-existing package
  passes without modification (pluginsdk, testing, registry, jsonld, pricing, mapping,
  utilization)

### Implementation for User Story 3

- [x] T019 [US3] Verify the wire-compatibility invariants: `ActualCostResult` field 8 still
  `expires_at`; both lineage fields optional with no presence requirement; no RPC, no
  `PluginCapability` entry, no conformance-harness change (`git diff` limited to the two proto
  files, the four regenerated files, and the two new pluginsdk files)

**Checkpoint**: Backward compatibility holds by construction and by suite.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Lint, formatting, TypeScript check, and validation runs

- [x] T020 [P] Run `golangci-lint run ./sdk/go/pluginsdk/...` and fix findings in the new
  files (resolved: gocognit, govet shadow, gochecknoglobals in the first test draft via a
  shared `roundTripProtojson` helper and a `wantFourLevelChain()` function)
- [x] T021 [P] Run `go build ./...` and `gofmt -l` on the new files; confirm no lines over
  120 characters (goimports/golines convention)
- [x] T022 [P] Run the TypeScript client type check (`npx tsc --noEmit` in
  `sdk/typescript/packages/client`, its `lint` script) against the regenerated bindings
- [x] T023 Run the validation sequence in `specs/055-cost-allocation-lineage/quickstart.md`
  end to end: focused test run, full `go test ./sdk/go/...`, `golangci-lint`, `buf lint`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup. Blocks all stories
- **User Story 1 (Phase 3)**: Depends on Foundational (needs the generated `pbc.LineageNode`)
- **User Story 2 (Phase 4)**: Depends on US1 (round-trips exercise the builder output)
- **User Story 3 (Phase 5)**: Depends on Foundational only; safe to run any time after T008
- **Polish (Phase 6)**: Depends on US1 and US2

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Foundational. MVP
- **User Story 2 (P2)**: After US1; its tests build chains through the US1 builder
- **User Story 3 (P3)**: Independent verification; no file overlap with US1/US2

### Within Each User Story

- Tests are written and failing before the implementation tasks in that story
- US1 test tasks (T009–T012) share one new test file but are independent cases
- US2 tests (T014–T016) are additive cases in the same file; no conflicts with US1 cases

### Parallel Opportunities

- T005 and T006 edit different messages in `costsource.proto` (sequenced in practice to keep
  one file edit per step)
- T009–T012 are independent test cases
- T014–T016 are independent test cases
- T020, T021, T022 run in parallel

---

## Parallel Example: User Story 1

```text
T009 chain order test
T010 metadata placement test
T011 partial chain test
T012 custom/deep chain test
```

After those fail for the right reason (no `LineageBuilder` yet):

```text
T013 lineage_builder.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Finish Phase 1 and Phase 2
2. Finish Phase 3
3. Stop and run `go test ./sdk/go/pluginsdk/ -run TestLineageBuilder`
4. Continue to US2 and US3 before commit so the round-trip and compatibility guarantees are
   not left unproven

### Incremental Delivery

1. Foundational lands the contract and bindings
2. US1 lets a plugin author build and attach a chain
3. US2 locks transport fidelity and the no-consistency-check rule
4. US3 proves nothing existing moved
5. Polish runs the lint and validation gates

---

## Out of scope (do not add tasks)

Rejected if analysis or a later pass proposes them:

- Lineage on `GetProjectedCostResponse`, `EstimateCostResponse`, batch wrappers, or
  `FocusCostRecord`
- A consistency check between lineage and `billing_account_id` / `sub_account_id` (warning or
  failure)
- A stored `depth` field or a `WalkLineage` contract helper
- Provider-API validation of node ids
- A new RPC or `PluginCapability` entry
- Reusing `ActualCostResult` field 8
- A conformance-harness fixture for lineage
- Documentation pages beyond the proto comments and this spec directory

## Notes

- [P] tasks = different files, no dependencies
- Each story's tests were written to fail before that story's implementation
- This tasks file is retrospective: the implementation it describes is complete and verified
  (`go test ./sdk/go/...` green, `golangci-lint` clean, `buf lint` clean, `tsc --noEmit`
  clean)
- Avoid: vague tasks, editing generated code by hand, a depth field, a consistency check
