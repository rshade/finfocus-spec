# Tasks: Resource Descriptor on Actual Cost Requests

**Input**: Design documents from `specs/597-actual-cost-resource-descriptor/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required. Constitution Principle V (test-first) applies; each story writes its tests
before the code they cover. Only the rejection cases (T015, T016), the mock assertions (T005), and
the conformance registration (T012) can fail first; the valid-input and nil cases (T006, T007, T010)
pass before and after, and guard against regressions.

**Organization**: Tasks are grouped by user story so each story can be checked on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: The user story the task belongs to (US1-US4)

## Phase 1: Setup

- [X] T001 Confirm the worktree baseline: `make generate && git diff --exit-code -- sdk/` is clean and `go test
  ./sdk/go/pluginsdk/ ./sdk/go/testing/` passes before any change

## Phase 2: Foundational (proto first, blocks every story)

- [X] T002 Add `ResourceDescriptor resource = 11;` to `GetActualCostRequest` in proto/finfocus/v1/costsource.proto
  after `billing_account_id = 9`, with the comment from contracts/proto.md (same meaning as
  `GetProjectedCostRequest.resource` including `attributes` and host redaction; unset → fall back to `tags`,
  `resource_id`, `arn`; `tags` keep their meaning and hosts SHOULD keep sending them; when set, pricing dimensions
  come from `resource`, not `tags`; `resource_id` stays required; same `resource` on every page; dry run otherwise
  unchanged). Keep the field 9 sentence holding field 10 free
- [X] T003 Run `make generate` to regenerate sdk/go/proto/finfocus/v1/ and
  sdk/typescript/packages/client/src/generated/; restore any unrelated reformatted `*.connect.go` files with `git
  checkout`
- [X] T004 Run `make buf-lint` and `buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'`
  (in a worktree, fall back to `git archive origin/main proto buf.yaml` into a scratch dir); both must report nothing

**Checkpoint**: Field 11 exists in Go (`GetResource()`) and TypeScript (`resource?`) bindings.

## Phase 3: User Story 1 - List-price plugin prices actual cost from the descriptor (P1) MVP

**Goal**: A request carrying a descriptor validates and reaches the plugin, which reads it.

**Independent Test**: `go test ./sdk/go/testing/ -run TestMockPluginGetActualCost_Resource` and the valid-request
cases in `TestValidateActualCostRequest_Resource` / `TestValidateGetActualCostRequest`.

### Tests (write first, confirm they fail)

- [X] T005 [P] [US1] Create sdk/go/testing/mock_actual_cost_resource_test.go (Apache 2.0 header, `package
  testing_test`) with `TestMockPluginGetActualCost_Resource`: over a `TestHarness`, without `t.Parallel()` in
  subtests, assert (a) with `billing_account_id` set and a descriptor `aws`/`ec2`/`t3.micro`/`us-east-1`, every FOCUS
  record has `resource_type` `ec2`, `region_id` `us-east-1`, `sku_id` `t3.micro`; (b) costs are identical with and
  without the descriptor; (c) with no descriptor the record's `resource_type`, `region_id`, `sku_id` are empty, as
  before
- [X] T006 [P] [US1] Add valid cases to `TestValidateActualCostRequest_Resource` in sdk/go/pluginsdk/validation_test.go:
  a request with a full descriptor (tags plus a small `attributes` Struct) passes; a request with an empty
  `&pbc.ResourceDescriptor{}` passes (the `pluginsdk` layer checks lengths only)
- [X] T007 [P] [US1] Add a valid case to `TestValidateGetActualCostRequest` in sdk/go/testing/contract_test.go: a
  request with a descriptor carrying provider `aws`, resource type `ec2`, tags, and `attributes` passes

### Implementation

- [X] T008 [US1] In sdk/go/testing/mock_plugin.go `mockActualCostFocusRecord`, when `req.GetResource()` is non-nil set
  `ResourceType`, `RegionId`, `SkuId` from its `resource_type`, `region`, `sku`; leave every other field and all cost
  values unchanged. Update the dry-run comment in `GetActualCost` to note that plugins may read
  `resource.resource_type`, while the mock keeps returning default mappings
- [X] T009 [US1] Run T005-T007 and confirm they pass

**Checkpoint**: The mock reads the descriptor; valid descriptors pass both validators.

## Phase 4: User Story 2 - Existing hosts and plugins keep working (P1)

**Goal**: Requests without a descriptor behave as before; a plugin that ignores the descriptor passes
conformance.

**Independent Test**: `go test ./sdk/go/testing/ -run 'TestRPCCorrectness|TestConformance'` and the
nil-resource validator cases.

### Tests (write first)

- [X] T010 [P] [US2] Add nil-resource cases (`Resource: nil`) to `TestValidateActualCostRequest_Resource` in
  sdk/go/pluginsdk/validation_test.go and `TestValidateGetActualCostRequest` in sdk/go/testing/contract_test.go: both
  pass; add a precedence case where `resource_id` is empty and the descriptor is invalid, and assert the existing
  `resource_id` error is returned (descriptor checked last)
- [X] T011 [P] [US2] Add `BenchmarkValidateActualCostRequest_WithResource` (descriptor without attributes) next to
  `BenchmarkValidateActualCostRequest_Valid` in sdk/go/pluginsdk/validation_test.go, and an allocation test asserting
  0 allocs for the nil-resource and no-attributes shapes (`testing.AllocsPerRun`)
- [X] T012 [P] [US2] In sdk/go/testing/rpc_correctness_test.go add a test asserting
  `RPCCorrectness_GetActualCostWithResource` is registered at `ConformanceLevelStandard`, passes for
  `NewMockPlugin()`, and passes for a plugin that ignores `resource` (a wrapper embedding the mock whose
  `GetActualCost` clears `req.Resource` before delegating, or an equivalent); follow the pattern of
  `TestRPCCorrectnessGetActualCostBillingAccount`

### Implementation

- [X] T013 [US2] In sdk/go/testing/rpc_correctness.go add `testGetActualCostWithResourceRPC`: build a descriptor with
  `CreateResourceDescriptor(providerAWS, ec2ResourceType, "t3.micro", "us-east-1")`, tags `{"team": "platform"}`, and
  `conformanceAttributes()`; send it on `GetActualCostRequest` with `testResourceID` and
  `CreateTimeRange(HoursPerDay)`; succeed on `NotFound`/`Unavailable` or when `ValidateActualCostResponse` passes;
  never inspect whether the plugin used the descriptor. Register it as `RPCCorrectness_GetActualCostWithResource`
  (Standard) after `RPCCorrectness_GetActualCostBillingAccount` with a `createGetActualCostWithResourceRPCTest`
  factory
- [X] T014 [US2] Run T010-T012 plus `go test -v -run TestConformance ./sdk/go/testing/` and `go test ./sdk/go/testing/
  -run 'ArnField|BackwardCompatibility|BillingAccount|DryRun'`; all pass unchanged

**Checkpoint**: Backward compatibility shown by unchanged suites and the ignoring-plugin case.

## Phase 5: User Story 3 - Validation rejects an oversized or malformed descriptor (P2)

**Goal**: Both layers apply the descriptor limits when the field is set.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run TestValidateActualCostRequest` and
`go test ./sdk/go/testing/ -run 'TestValidateGetActualCostRequest|TestContract'`.

### Tests (write first)

- [X] T015 [P] [US3] In sdk/go/pluginsdk/validation_test.go add rejection cases to `TestValidateActualCostRequest_Resource`:
  `attributes` whose `proto.Size` exceeds `MaxAttributesBytes` (65536) → `codes.InvalidArgument`; a tag value longer
  than `MaxTagValueLength` (2048 bytes) → `codes.InvalidArgument`; a `resource_type` longer than
  `MaxResourceTypeLength` → `codes.InvalidArgument`
- [X] T016 [P] [US3] In sdk/go/testing/contract_test.go add rejection cases to `TestValidateGetActualCostRequest`:
  oversized `attributes` → `errors.Is(err, ErrAttributesTooLarge)`; descriptor with empty provider →
  `ErrEmptyProvider`; descriptor with empty resource type → `ErrEmptyResourceType`

### Implementation

- [X] T017 [US3] In sdk/go/pluginsdk/validation.go `ValidateActualCostRequest`, after the time range check add `if
  resource := req.GetResource(); resource != nil { return ValidateResourceDescriptor(resource) }`, returning the
  status error unchanged; add step 6 to the doc comment's validation order
- [X] T018 [US3] In sdk/go/testing/contract.go `ValidateGetActualCostRequest`, after `ValidateTags` add the same
  nil-guarded call to `ValidateResourceDescriptor`, returning its `ContractError`; add contract-suite cases
  `GetActualCostRequest_WithResourceAccepted` and `GetActualCostRequest_OversizedAttributesRejected` in
  `registerRequestTests`
- [X] T019 [US3] Run T015-T016 and the contract suite; then `go test ./sdk/go/pluginsdk/ -run '^$' -bench
  BenchmarkValidateActualCostRequest -benchmem` and confirm `_Valid` and `_WithResource` report 0 allocs/op

**Checkpoint**: Descriptor limits enforced on the actual path, same errors as other paths.

## Phase 6: User Story 4 - TypeScript host sends the descriptor on every page (P2)

**Goal**: `actualCostIterator` carries `resource` on every page.

**Independent Test**: `cd sdk/typescript/packages/client && npx vitest run test/pagination.test.ts`.

- [X] T020 [US4] In sdk/typescript/packages/client/test/pagination.test.ts add `it('sends resource on every page
  request', ...)` next to the billingAccountId test: build the request with a `resource` (provider, resourceType, sku,
  region, attributes), serve two pages through msw, and assert each page body's `resource` deep-equals the first
- [X] T021 [US4] Run `cd sdk/typescript && npm ci && npm run build && npm test`; no source change expected since the
  iterator clones the whole request

**Checkpoint**: TS parity shown.

## Phase 7: Polish and Documentation

- [X] T022 [P] Update docs/PROPERTY_MAPPING.md: add a section describing what reaches `GetActualCost` today
  (`resource_id`, `arn`, cloud `tags`, host-injected `sku`/`region`/`provider`/`resource_type` tags) and with
  `resource` (the full descriptor, same as projected), plus the fallback and precedence rules
- [X] T023 [P] Update PLUGIN_DEVELOPER_GUIDE.md `GetActualCost RPC` section: add `resource = 11` to the message
  excerpt and a rules list (prefer `resource`, fall back to `tags`/`resource_id`/`arn`, tags are billing labels, same
  `resource` on every page)
- [X] T024 [P] Update sdk/go/pluginsdk/README.md (actual cost section: `req.GetResource()` preferred with fallback,
  `ValidateActualCostRequest` step 6) and sdk/go/testing/README.md (new conformance check in the RPC correctness list,
  contract validator descriptor rule)
- [X] T025 [P] Update sdk/typescript/README.md actual cost example to show `resource` set with
  `ResourceDescriptorBuilder` or a plain object; check the example compiles with `tsc --ignoreConfig --strict
  --noEmit` against the built client
- [X] T026 Add a "Recent Changes" entry and an "Actual Cost Resource Pattern (597-actual-cost-resource-descriptor)"
  note to CLAUDE.md, mirrored in AGENTS.md and GEMINI.md, by hand
- [X] T027 Run quickstart.md end to end: `make generate && git diff --exit-code -- sdk/` (re-running generation
  changes nothing beyond T003's output), `make buf-lint`, `buf breaking`, `make test`, `go test -v -tags=integration
  ./sdk/go/testing/`, `make lint-go`, `make lint-markdown`, `make lint-yaml`, `make validate-npm`

## Dependencies and Execution Order

- Phase 1 → Phase 2 (proto) → all stories.
- US1 (Phase 3) and US3 (Phase 5) both edit sdk/go/pluginsdk/validation_test.go and
  sdk/go/testing/contract_test.go; run them in sequence, not in parallel.
- US2 depends on US1's mock change only for its "mock passes" assertion; otherwise independent.
- US4 depends only on Phase 2.
- Phase 7 after all stories.

## Parallel Opportunities

- T005, T006, T007 (different files).
- T010, T011, T012 once US1 is done.
- T020 can run alongside any Go story after Phase 2.
- T022-T025 in parallel.

## Implementation Strategy

MVP is Phase 2 + US1: the field exists, validates, and the mock reads it. US2 proves compatibility,
US3 adds the rejection rules, US4 pins TypeScript parity, and Phase 7 finishes docs and gates.
