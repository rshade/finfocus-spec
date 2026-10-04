---

description: "Tasks for the plugin-supplied conformance sample resource"
---

# Tasks: Plugin-Supplied Sample Resource for Conformance

**Input**: Design documents from `specs/598-conformance-sample-resource/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/go-api.md, quickstart.md

**Tests**: Required. The constitution (Principle V) makes this test-first: each story's tests are
written and fail before its implementation.

**Provider neutrality (FR-011)**: every new line in code, tests, and docs uses the `custom` provider
or a placeholder. The `aws` default may appear only in `DefaultSampleResource`. No other provider
name is added.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Confirm the worktree baseline: `make generate && git diff --exit-code -- sdk/` and `go test
  ./sdk/go/testing/ ./sdk/go/pluginsdk/` pass on the unmodified branch

---

## Phase 2: Foundational (blocks all stories)

- [X] T002 Add `SampleResource *pbc.ResourceDescriptor` to `SuiteConfig` with godoc "nil means
  DefaultSampleResource()", and add `DefaultSampleResource()` returning a new `CreateResourceDescriptor(providerAWS,
  ec2ResourceType, "t3.micro", "us-east-1")`, both with godoc, in sdk/go/testing/conformance.go
- [X] T003 Add unexported `sampleResource` to `TestHarness` and exported `SampleResource()` that returns
  `proto.CloneOf` of it, or `DefaultSampleResource()` when nil, with godoc, in sdk/go/testing/harness.go
- [X] T004 In `ConformanceSuite.Run` and `RunCategory`, validate a non-nil `config.SampleResource` with
  `ValidateResourceDescriptor` before creating the harness (return `fmt.Errorf("invalid sample resource: %w", err)`),
  then set `harness.sampleResource` to a copy, in sdk/go/testing/conformance.go

**Checkpoint**: code compiles; behavior unchanged (every check still uses its own descriptor).

---

## Phase 3: User Story 1 - A plugin for any provider runs conformance (Priority: P1) MVP

**Goal**: Callers supply a sample resource and every registered check that sends a descriptor uses it.

**Independent Test**: A strict `custom`-only plugin passes Basic, Standard, and Advanced with a
`custom` sample resource, and fails Basic without one.

### Tests for User Story 1 (write first, must fail)

- [X] T005 [US1] Create sdk/go/testing/sample_resource_test.go (`package testing_test`, Apache 2.0 header) with a
  `strictCustomPlugin` that embeds `*plugintesting.MockPlugin` (`SupportedProviders: []string{"custom"}`) and returns
  `codes.InvalidArgument` `"unsupported provider: %s"` from `GetProjectedCost`, `GetPricingSpec`, and `GetActualCost`
  for any provider other than `custom`, and `codes.InvalidArgument` from `GetActualCost` when `resource` is nil; plus
  a `customSample()` helper returning `CreateResourceDescriptor("custom", "instance", "standard", "region-1")`
- [X] T006 [US1] In sdk/go/testing/sample_resource_test.go add `TestSampleResourceAllLevels`: table over Basic,
  Standard, Advanced calling `RunConformance(plugin, level, WithSampleResource(customSample()))`, require 0 failures
  and list failing checks in the message
- [X] T007 [US1] In sdk/go/testing/sample_resource_test.go add `TestSampleResourceDefaultStillAWS`:
  `RunBasicConformance(strictCustomPlugin)` reports failures (guard that the default is unchanged and the defect is
  real)
- [X] T008 [US1] In sdk/go/testing/sample_resource_test.go add `TestSampleResourceInvalid`: table (empty provider,
  provider not in `ValidProviders`, empty resource type) each returns an error naming the field from `RunConformance`
  before any check runs (nil result); add an unknown-level row (`ConformanceLevel(99)`) that also returns an error
- [X] T009 [US1] In sdk/go/testing/sample_resource_test.go add `TestHarnessSampleResourceCopies`: mutating the value
  passed to `WithSampleResource` after the option is built, or the value returned by `SampleResource()`, does not
  change the next `SampleResource()` result; a harness from `NewTestHarness` returns the default
- [X] T010 [P] [US1] In sdk/go/pluginsdk/conformance_test.go add `customOnlyPlugin` (embeds `conformanceMockPlugin`;
  `GetProjectedCost` and `GetPricingSpec` return `codes.InvalidArgument` for any provider other than `custom`;
  `GetPricingSpec` echoes the request's provider and resource type in its spec) and
  `TestRunConformanceSampleResource`: it passes `pluginsdk.RunConformance(plugin, ConformanceLevelBasic,
  WithSampleResource(...))` with a `custom` descriptor, fails `RunBasicConformance(plugin)`, and a nil plugin returns
  `ErrNilPlugin`
- [X] T011 [US1] Run `go test -run 'TestSampleResource|TestHarnessSampleResource|TestRunConformanceSampleResource'
  ./sdk/go/testing/ ./sdk/go/pluginsdk/` and confirm they fail (compile errors count) before T012

### Implementation for User Story 1

- [X] T012 [US1] Add `ConformanceOption func(*SuiteConfig)`, `WithSampleResource(r)` (stores `proto.CloneOf(r)`, nil
  restores default), and `RunConformance(impl, level, opts...)` that builds the level preset (Basic: 60 s, 10
  parallel, benchmarks off, Spec Validation + RPC Correctness; Standard: 60 s, 10 parallel, benchmarks on, all four
  categories; Advanced: 120 s, 50 parallel, benchmarks on, 10 s duration, all four), applies opts, and returns an
  error for an unknown level; godoc on each new identifier; in sdk/go/testing/conformance.go
- [X] T013 [US1] Make `RunBasicConformance`, `RunStandardConformance`, `RunAdvancedConformance` call
  `RunConformance(impl, <level>)` with unchanged signatures in sdk/go/testing/conformance.go
- [X] T014 [P] [US1] Replace the hard-coded descriptor with `harness.SampleResource()` in the three Spec Validation
  checks and `RunSpecValidation` (which uses a fresh `NewTestHarness`, so it keeps the default) in
  sdk/go/testing/spec_validation.go
- [X] T015 [P] [US1] Replace the hard-coded descriptor with `harness.SampleResource()` in `testSupportsRPC`,
  `testGetProjectedCostRPC`, `testGetPricingSpecRPC`, `testGetActualCostWithResourceRPC`, and
  `testGetProjectedCostWithAttributesRPC` (set `Attributes` on the copy, replacing any author attributes) in
  sdk/go/testing/rpc_correctness.go
- [X] T016 [P] [US1] In the latency checks and `RunPerformanceBenchmarks`, take `harness.SampleResource()` once before
  `measureLatency` and reuse it inside the timed closure (FR-012) in sdk/go/testing/performance.go
- [X] T017 [P] [US1] In `runParallelRequests` and `validateConsistentResponses`, take `harness.SampleResource()` once
  before starting goroutines and send that descriptor (read-only under gRPC marshal, FR-012) in
  sdk/go/testing/concurrency.go
- [X] T018 [P] [US1] Add `type ConformanceOption = plugintesting.ConformanceOption`, `WithSampleResource` (forwards),
  and `RunConformance(plugin, level, opts...)` (nil check, then `plugintesting.RunConformance(NewServer(plugin),
  level, opts...)`) with godoc in sdk/go/pluginsdk/conformance.go

**Checkpoint**: T006's Basic row, T008, T009, and T010 pass; T007 still reports failures (default unchanged). T006's
Standard and Advanced rows stay red until T023, because the strict plugin rejects actual-cost requests without a
resource.

---

## Phase 4: User Story 2 - Existing callers see no change (Priority: P1)

**Goal**: The default path is byte-for-byte the old behavior.

**Independent Test**: existing tests pass with no call-site edits.

- [X] T019 [US2] In sdk/go/testing/sample_resource_test.go add `TestDefaultSampleResourceValid` (default passes
  `ValidateResourceDescriptor` and has provider `aws`, type `ec2`, SKU `t3.micro`, region `us-east-1`) and
  `TestRunnersStoreAsFuncValues` (assign the three `plugintesting` runners to a `func(pbc.CostSourceServiceServer)
  (*plugintesting.ConformanceResult, error)` variable); add the same func-value guard for the three `pluginsdk`
  runners in sdk/go/pluginsdk/conformance_test.go
- [X] T020 [US2] Run `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/` and `go test -v -run TestConformance
  ./sdk/go/testing/` with no edits to existing tests; any change in existing expectations is a defect in T012-T018
- [X] T021 [US2] Confirm `grep -nE 'CreateResourceDescriptor\((providerAWS|"aws")'
  sdk/go/testing/{spec_validation,rpc_correctness,performance,concurrency}.go` prints nothing (SC-003)

---

## Phase 5: User Story 3 - Actual-cost checks send pricing inputs (Priority: P2)

**Goal**: No conformance check sends an actual-cost request without a resource descriptor.

**Independent Test**: the strict plugin's `GetActualCost` rejects nil `resource`; Standard passes with a
`custom` sample.

- [X] T022 [US3] In sdk/go/testing/sample_resource_test.go add `TestActualCostChecksSendResource`: run
  `RunConformance(..., ConformanceLevelStandard, WithSampleResource(customSample()))` against the strict plugin and
  require `RPCCorrectness_GetActualCostRPC` and `RPCCorrectness_GetActualCostBillingAccount` results to succeed (fails
  before T023); after T023 the full T006 table passes
- [X] T023 [US3] Add `Resource: harness.SampleResource()` to the requests in `testGetActualCostRPC`,
  `testGetActualCostBillingAccountRPC`, and `testInvalidTimeRangeHandling` (found during implementation), keeping
  `ResourceId`, `Start`, `End`, and `BillingAccountId`; accepted codes
  stay NotFound and Unavailable (FR-007), in sdk/go/testing/rpc_correctness.go
- [X] T024 [US3] Update the three check descriptions per contracts/go-api.md ("... for the sample resource (sent as
  resource); requests without a resource are not checked") in sdk/go/testing/rpc_correctness.go

---

## Phase 6: User Story 4 - Documentation (Priority: P3)

- [X] T025 [P] [US4] Add a "Sample resource" subsection under "Running Conformance Tests" in sdk/go/testing/README.md:
  `RunConformance` + `WithSampleResource` example with a `custom` descriptor, the default, the copy rule, the
  actual-cost decision, and `TestHarness.SampleResource()` for custom checks; update the API list in "Conformance
  Suite"
- [X] T026 [P] [US4] Extend "Conformance Testing" in sdk/go/pluginsdk/README.md with `pluginsdk.RunConformance` and
  `pluginsdk.WithSampleResource`, and add a row to the levels table
- [X] T027 [P] [US4] Add a "Cost source conformance" subsection before the allocator conformance section in
  PLUGIN_DEVELOPER_GUIDE.md explaining why a plugin that does not price the default resource supplies its own, with a
  `custom` example; update the `RPCCorrectness_GetActualCostBillingAccount` mention near the billing-account section
  if it describes the request shape
- [X] T028 [US4] Compile every new Go snippet from T025-T027 as a standalone file in a scratch module dir (one file
  per snippet) against the worktree SDK; fix any that fail

---

## Phase 7: Polish & Cross-Cutting

- [X] T029 Add a "Conformance Sample Resource Pattern (598-conformance-sample-resource)" section and Active
  Technologies / Recent Changes entries by hand in CLAUDE.md (do not run update-agent-context.sh)
- [X] T030 Provider-neutrality gate (SC-005): `git diff origin/main -- . ':!specs' | grep -E '^\+' | grep -v
  'google\.golang\.org' | grep -niE '(azure|gcp|kubernetes|k8s|oracle|alibaba)'` prints nothing
- [X] T031 Run gates: `gofmt`/`goimports` on changed files, `make lint-go` (0 issues), `make test`, `go test -v
  -tags=integration ./sdk/go/testing/`, `go test -race -run
  'TestSampleResource|TestHarnessSampleResource|TestActualCostChecksSendResource' ./sdk/go/testing/` (FR-012), `make
  lint-markdown`, `make lint-yaml`, `make validate-npm`, `make generate && git diff --exit-code -- sdk/`
- [X] T032 Run quickstart.md sections 1-5 and record results

---

## Dependencies & Execution Order

- Phase 1 → Phase 2 → US1 (tests T005-T011, then T012-T018) → US2 and US3 (independent of each other) → US4 → Polish.
- T014-T018 touch different files and can run in parallel after T012.
- T023/T024 touch rpc_correctness.go after T015; run them sequentially after T015.
- T025-T027 are independent docs.

## Implementation Strategy

MVP is US1 (T001-T018): non-AWS plugins can certify at Basic. US3 then unblocks Standard for
list-price plugins; US2 guards compatibility throughout; US4 and Polish land in the same PR
(constitution VII).
