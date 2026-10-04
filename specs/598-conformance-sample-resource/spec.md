# Feature Specification: Plugin-Supplied Sample Resource for Conformance

**Feature Branch**: `598-conformance-sample-resource`

**Created**: 2026-10-04

**Status**: Draft

**Input**: User description: "feat(testing): let plugins run conformance with their own sample
resource. The conformance suite tests every plugin with one hard-coded AWS descriptor; SuiteConfig
cannot supply a different resource. Add a configurable sample resource (default stays the AWS
descriptor), settable for the Basic, Standard, and Advanced runners and the pluginsdk wrappers, used
everywhere the suite builds a descriptor. The bare GetActualCost checks must use the sample resource
in GetActualCostRequest.resource, and the check description must say what happened to the bare
request. A test runs the suite against a mock that prices only a non-AWS provider. Update the
READMEs and the developer guide. Closes #625"

## Clarifications

`/speckit-clarify` was not required. The issue fixes the default, the runners that need the
setting, and the scope. The one open choice it delegates (replace the bare `GetActualCost` request
or keep it) is decided in FR-006.

### Issue claims verified against `origin/main` (d819094)

The issue was filed from measurements on a downstream plugin. Each claim was re-checked against
this repository before specifying, and the spec relies only on the verified ones:

- **Confirmed**: 23 hard-coded `aws/ec2/t3.micro/us-east-1` descriptors in non-test files under
  `sdk/go/testing/`; `SuiteConfig` has no resource setting; the three registered Spec Validation
  checks and the standalone `RunSpecValidation` fail on any `GetPricingSpec` error.
- **Corrected**: the code-acceptance lines the issue cites (`rpc_correctness.go:109,176,246`) are
  all `GetActualCost` checks. The projected-cost check accepts no error code at all.
- **Corrected**: there is no paginated `GetActualCost` conformance check. The bare-request checks
  are `RPCCorrectness_GetActualCostRPC` and `RPCCorrectness_GetActualCostBillingAccount` (both
  Standard level, not Basic), plus the Basic `RPCCorrectness_InvalidTimeRange`, which the issue
  does not mention; it passes a strict plugin today only because a bare request is rejected anyway.
- **Out of scope**: `contract.go:731` is a request-validator unit test with no plugin behind it;
  its provider value does not affect any plugin. The `harness.go` occurrences belong to the
  standalone `ErrorHandlingTestSuite`, not to the registered conformance checks.
- **Reproduced in-repo** with the existing mock, wrapped to accept one non-AWS provider and answer
  `InvalidArgument` for any other (the code plugins return for a provider they do not price):
  Basic 5 passed and 6 failed, Standard 13 passed and 9 failed. Every failure was the hard-coded
  descriptor or a bare `GetActualCost` request. The unwrapped mock passes because it never rejects
  an unknown provider on the cost RPCs, which is why the repository's own tests never caught this.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A plugin for any provider runs conformance against a resource it prices (Priority: P1)

A plugin author whose plugin prices a provider other than AWS wants to certify the plugin at Basic,
Standard, or Advanced level. They tell the suite which resource to use, one their plugin prices, and
every conformance check that sends a resource sends that one. The plugin passes or fails on its own
behavior, not on whether it prices AWS.

**Why this priority**: This is the defect. The specification is provider-independent, but its
conformance suite only certifies plugins that price one AWS instance type.

**Independent Test**: Run each conformance level against a reference plugin that prices only a
neutral, non-AWS provider and rejects every other provider with `InvalidArgument`. With that
provider's sample resource supplied, every level passes. Without it, the Basic run fails as it does
today.

**Acceptance Scenarios**:

1. **Given** a plugin that prices only a non-AWS provider, **When** its author runs Basic, Standard,
   or Advanced conformance with a sample resource for that provider, **Then** no check fails because
   of the resource's provider.
2. **Given** the same plugin, **When** the checks that send a resource run, **Then** each sends the
   supplied sample resource, including Spec Validation, RPC Correctness, Performance, and
   Concurrency.
3. **Given** a check that adds data to the resource (for example nested attributes), **When** it
   runs, **Then** it starts from the supplied sample resource and does not alter the author's copy.
4. **Given** a plugin author who uses the plugin SDK's conformance helpers rather than the testing
   package directly, **When** they supply a sample resource, **Then** it reaches the same checks.

---

### User Story 2 - Existing callers see no change (Priority: P1)

A plugin author, or this repository's own tests, runs conformance the way they do today without
naming a resource. The suite uses the same AWS descriptor as before, and every result is unchanged.

**Why this priority**: Backward compatibility is a constitutional rule, and every current caller is
in this state.

**Independent Test**: The existing conformance tests and mock pass unchanged, with no edits to their
call sites.

**Acceptance Scenarios**:

1. **Given** a caller that supplies no sample resource, **When** any level runs, **Then** the suite
   sends the current default descriptor and reports the same results as before this change.
2. **Given** existing code that calls the three level runners or the plugin SDK wrappers, **When**
   it is compiled against the new version, **Then** it compiles without change.

---

### User Story 3 - Actual-cost checks send pricing inputs (Priority: P2)

A list-price plugin cannot price an actual-cost request that carries only a resource id and a time
window, so it rejects the request as invalid. The actual-cost checks now send the sample resource in
the request's resource descriptor, so the plugin receives the inputs it needs.

**Why this priority**: These checks are Standard level. They block Standard certification for any
list-price plugin, whatever provider it prices.

**Independent Test**: Run Standard conformance against a reference plugin that rejects an
actual-cost request without a resource descriptor. The actual-cost checks pass.

**Acceptance Scenarios**:

1. **Given** the plain, billing-account, and invalid-time-range actual-cost checks, **When** they
   run, **Then** each
   request carries the sample resource as its resource descriptor, and still carries the resource
   id and time window it carries today.
2. **Given** a reader of the check list, **When** they read these checks' descriptions, **Then**
   each says the request carries the sample resource; the plain check also says a request without
   a resource descriptor is not checked.

---

### User Story 4 - Authors can find out how to do it (Priority: P3)

A plugin author reading the testing package README, the plugin SDK README, or the developer guide
learns how to supply a sample resource and why a plugin that does not price AWS needs one.

**Why this priority**: The setting exists only if authors can find it, and the constitution makes
stale docs a merge blocker.

**Independent Test**: Each of the three documents shows how to supply a sample resource, and the
example compiles.

**Acceptance Scenarios**:

1. **Given** the testing package README, the plugin SDK README, and the developer guide's
   conformance section, **When** an author reads them, **Then** each shows how to supply a sample
   resource and states the default.
2. **Given** those examples, **When** they are compiled, **Then** they compile against the
   published API.

---

### Edge Cases

- **Invalid sample resource** (no provider, no resource type, or a provider the contract validator
  rejects): the run stops before any check and returns an error that names the problem, rather than
  reporting every check as a plugin failure.
- **Author mutates the resource after starting a run**: the suite uses its own copy, so later edits
  do not change the run.
- **A check mutates the resource it sends**: each check works on its own copy, so one check cannot
  change what another sends.
- **Sample resource with attributes**: the checks that add their own attributes replace them on
  their copy; the other checks send the author's attributes unchanged.
- **Plugin rejects a request with no resource descriptor**: no conformance check sends such an
  actual-cost request any more (FR-006). The nil-resource check, which deliberately sends no
  descriptor to `Supports`/projected cost, is unchanged.
- **Plugin returns NotFound or Unavailable for the sample resource's actual cost**: still accepted,
  exactly as today.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The suite configuration MUST let a caller supply one sample resource descriptor.
  When none is supplied, the suite MUST use the current default (`aws`, `ec2`, `t3.micro`,
  `us-east-1`).
- **FR-002**: Callers MUST be able to supply the sample resource to the Basic, Standard, and
  Advanced levels in the testing package and in the plugin SDK, without changing the signature of
  any existing exported function, so existing direct calls and existing code that stores a runner in
  a typed function value both keep compiling.
- **FR-003**: Every registered conformance check that sends a resource descriptor today MUST send
  the configured sample resource instead: the three Spec Validation checks, the RPC Correctness
  checks for `Supports`, projected cost, projected cost with attributes, pricing spec, and actual
  cost with resource, the Performance latency checks, and the Concurrency checks.
- **FR-004**: The standalone runners that build their own descriptor today (`RunSpecValidation`,
  `RunPerformanceBenchmarks`, `RunConcurrencyTests`, `RunRPCCorrectness`) keep the default sample
  resource and their signatures; their behavior does not change. Authors who need a different
  resource use the level entry points (FR-002).
- **FR-005**: A conformance check MUST be able to read the sample resource as a copy it may modify,
  so custom checks written by plugin authors can use it too.
- **FR-006**: The three actual-cost checks that send a bare request today
  (`RPCCorrectness_GetActualCostRPC`, `RPCCorrectness_GetActualCostBillingAccount`, and
  `RPCCorrectness_InvalidTimeRange`) MUST send the sample resource in the request's resource
  descriptor, in addition to the resource id and window they send today. For the invalid time
  range check this makes the expected rejection come from the window, not from missing inputs.
  The bare request is **replaced, not kept**: a request with no pricing inputs has no correct
  answer for a list-price plugin, and keeping it while accepting `InvalidArgument` would make the
  check accept almost anything. Each check's description MUST say it sends the sample resource.
- **FR-007**: The codes each check accepts MUST NOT change. In particular, the projected-cost
  checks still accept no error.
- **FR-008**: A run with an invalid sample resource MUST fail before any check runs, with an error
  that names the invalid field, using the existing resource descriptor contract rules.
- **FR-009**: The repository MUST include a test that runs every conformance level against a
  reference plugin that prices only the neutral `custom` provider and answers `InvalidArgument` for
  any other provider and for an actual-cost request without a resource descriptor; with a `custom`
  sample resource every level passes, and without one the Basic level fails.
- **FR-010**: The testing package README, the plugin SDK README, and the developer guide's
  conformance section MUST document the setting, its default, and the FR-006 decision, with an
  example that compiles.
- **FR-011 (provider neutrality)**: The change MUST NOT add provider-specific content beyond the
  existing AWS default. New code, tests, and documentation MUST use the neutral `custom` provider
  or placeholders, and MUST NOT name any other provider's resource types, SKUs, regions, or
  plugins. Downstream measurements stay in the issue, not in the repository.
- **FR-012**: Checks that time plugin calls MUST NOT include the copying of the sample resource in
  their measurements, and checks that send from several goroutines MUST NOT let one goroutine's
  request change another's.

### Key Entities

- **Sample resource**: one resource descriptor that a plugin author asserts their plugin prices.
  It carries a provider, a resource type, and optionally a SKU, region, tags, and attributes. It
  has a default and is copied on every use.
- **Suite configuration**: the existing per-run settings (level, timeout, parallelism,
  benchmarks), extended with the sample resource.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin that prices only one non-AWS provider passes all three conformance levels
  once its author supplies a sample resource, down from 6 Basic and 9 Standard failures today.
- **SC-002**: 0 existing conformance, mock, or plugin SDK tests change their call sites or
  expectations.
- **SC-003**: 0 registered conformance checks send a hard-coded descriptor; every one that sends a
  descriptor uses the configured sample resource.
- **SC-004**: 0 conformance checks send an actual-cost request without a resource descriptor.
- **SC-005**: 0 lines added by this change, in code, tests, or documentation outside `specs/`,
  name a provider other than `aws` (the kept default) and `custom`.
- **SC-006**: All three documents show the setting with an example that compiles.

## Assumptions

- The AWS default stays for compatibility only; it is not a statement that conformance targets AWS.
- `custom` is already a valid provider in the contract rules, so the neutral test needs no
  provider-list change.
- Changing the existing mock's provider handling is out of scope; the neutral test uses a strict
  wrapper around it. The mock's leniency is why the repository's own tests never caught the
  defect, and that is recorded here rather than changed.
- The standalone `ErrorHandlingTestSuite` helpers are not registered conformance checks and keep
  their descriptor; `contract.go` validator tests are unaffected.
- The Performance latency checks time a call whether or not it errors. Changing that is out of
  scope and is listed as a follow-up in the pull request, not filed.
- Changing which codes the projected-cost checks accept is out of scope (per the issue).
- The companion issue #626 (handler errors rewrapped as `Internal`) is separate.
- This change touches no `.proto` file and no TypeScript code: there is no TypeScript conformance
  suite, so SDK parity (Principle XIII) does not apply.
