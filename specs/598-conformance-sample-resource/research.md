# Research: Plugin-Supplied Sample Resource for Conformance

## R1. How a caller supplies the sample resource to a level

- **Decision**: Add one entry point per package, `RunConformance(impl, level, opts ...ConformanceOption)`,
  with `WithSampleResource(*pbc.ResourceDescriptor)` as the first option. The existing
  `RunBasicConformance`, `RunStandardConformance`, and `RunAdvancedConformance` keep their signatures
  and become one-line calls to `RunConformance` with no options. `pluginsdk` gets the matching
  `RunConformance(plugin, level, opts ...ConformanceOption)`, with `ConformanceOption` as a type alias
  and `WithSampleResource` as a forwarding function.
- **Rationale**: Adding a variadic parameter to the three existing runners, as the issue suggests,
  is source compatible for direct calls but not for code that stores a runner in a typed function
  value (`func(pbc.CostSourceServiceServer) (*ConformanceResult, error)`); `apidiff` reports it as an
  incompatible change. A new function breaks nothing (FR-002, SC-002). One function with a level
  argument avoids three more near-identical functions per package.
- **Alternatives considered**:
  - Variadic options on the existing runners: rejected for the compatibility reason above.
  - `RunBasicConformanceWithConfig` and siblings taking a `SuiteConfig`: six new functions, and
    callers would have to restate per-level timeouts and parallelism to set one field.
  - Only a `SuiteConfig` field, with authors building the suite by hand (`NewConformanceSuiteWithConfig`
    plus `Register*Tests`): works today but is undocumented and makes authors repeat which categories
    each level registers. The field is still added (R2), so that path also works.

## R2. Where the resource lives

- **Decision**: `SuiteConfig.SampleResource *pbc.ResourceDescriptor`; nil means the default.
  `DefaultSampleResource()` returns a fresh copy of the current descriptor
  (`aws`, `ec2`, `t3.micro`, `us-east-1`). `DefaultSuiteConfig()` leaves the field nil.
- **Rationale**: Keeps `SuiteConfig` the single description of a run, so `NewConformanceSuiteWithConfig`
  callers get the feature too. Nil-means-default keeps every existing `SuiteConfig{...}` literal valid
  and unchanged in behavior.

## R3. How checks read it

- **Decision**: The harness carries the resource. `TestHarness` gains an unexported field and an
  exported `SampleResource() *pbc.ResourceDescriptor` that returns a deep copy (`proto.CloneOf`), or a
  fresh default when unset. `ConformanceSuite.Run` and `RunCategory` set the field from the config.
- **Rationale**: Every check has the signature `func(*TestHarness) TestResult`, and external checks
  (for example `DryRunBasicConformanceTest`) depend on it, so the signature cannot change. A copy per
  call means no check can alter what another sends or what the author holds (edge cases). The
  standalone runners (`RunSpecValidation`, `RunPerformanceBenchmarks`, `RunConcurrencyTests`,
  `RunRPCCorrectness`) read the same method and so keep the default with no API change (FR-004).
- **Alternatives considered**: a package-level variable (not safe across concurrent runs); passing the
  config into each `TestFunc` (breaks the exported check signature).

## R4. Copy cost in latency checks

- **Decision**: Performance and concurrency checks take the copy once, before the timed loop or before
  starting goroutines, and send that one descriptor on every iteration.
- **Rationale**: The latency baselines must measure the plugin, not `proto.Clone`. Sending the same
  descriptor from several goroutines is safe because gRPC marshaling only reads it. This matches what
  the checks do today with their hard-coded descriptor.

## R5. Validating the sample resource

- **Decision**: `ConformanceSuite.Run` and `RunCategory` call `ValidateResourceDescriptor` on the
  configured resource before creating the harness and return
  `fmt.Errorf("invalid sample resource: %w", err)`. The `ContractError` names the field (FR-008).
- **Rationale**: An invalid sample would otherwise surface as many plugin failures. The contract rules
  already accept `custom`, so no provider list changes (FR-011). The default passes these rules
  (a unit test pins this).

## R6. The bare `GetActualCost` request

- **Decision**: Replace it. `RPCCorrectness_GetActualCostRPC`,
  `RPCCorrectness_GetActualCostBillingAccount`, and `RPCCorrectness_InvalidTimeRange` (found during
  implementation; it also sent a bare request) add `Resource: harness.SampleResource()` and keep
  `ResourceId`, `Start`, `End` (and `BillingAccountId`). Descriptions are updated to say the request
  carries the sample resource and that a request without one is not checked.
- **Rationale**: Spec 597 defines `resource` as preferred when set, with fallback to `resource_id` and
  tags; plugins that ignore it are unaffected, and the mock's FOCUS record already reads it. A bare
  request has no pricing inputs, so no answer is "correct" for a list-price plugin. Accepting
  `InvalidArgument` on a kept bare case would let the check pass almost any plugin.
- **Accepted codes unchanged** (FR-007): NotFound and Unavailable remain the only accepted errors.

## R7. The neutral reference plugin (FR-009, FR-011)

- **Decision**: A test-only wrapper in `sdk/go/testing` external tests (`package testing_test`) embeds
  `*plugintesting.MockPlugin` with `SupportedProviders = []string{"custom"}` and overrides
  `GetProjectedCost`, `GetPricingSpec`, and `GetActualCost` to return
  `InvalidArgument "unsupported provider: <p>"` for any other provider, and `InvalidArgument` for an
  actual-cost request without a resource. The sample resource uses only neutral values:
  `CreateResourceDescriptor("custom", "instance", "standard", "region-1")`.
- **Rationale**: The plain mock never rejects an unknown provider on the cost RPCs, so it cannot show
  the defect; a strict wrapper reproduces real plugin behavior (verified during research: Basic 6 and
  Standard 9 failures with the hard-coded descriptor). `custom` is the only valid provider that names no
  vendor. Changing the mock itself is out of scope.
- **Alternatives considered**: `kubernetes` or `gcp` (names a real platform; the maintainer asked that
  this change not tilt the spec toward any one); changing `MockPlugin` to reject unknown providers
  (changes behavior other tests rely on).

## R8. What is not changed

- `ErrorHandlingTestSuite` in `harness.go` (standalone helpers, not registered checks) and the
  `contract.go` validator tests keep their descriptors; neither runs the conformance levels.
- `RPCCorrectness_NilResource` still sends no descriptor on purpose.
- No `.proto`, generated code, or TypeScript change; there is no TypeScript conformance suite.
