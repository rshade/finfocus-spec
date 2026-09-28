# Feature Specification: Allocator Service (Allocate)

**Feature Branch**: `052-allocator-allocate`

**Created**: 2026-09-25

**Status**: Draft

**Input**: User description: "#506 — feat(proto): add AllocatorService.Allocate for cost allocation
plugins" (assessed as go in `.specify/assessments/allocator-allocate/decision.md`, Option B)

## Overview

FinFocus is adding in-cluster Kubernetes cost allocation. The host already gathers **usage** from a
usage source plugin (`GetStats`, #505) and prices the reported infrastructure (nodes, control plane)
through existing cost-source plugins. The step still missing is **dividing** that priced
infrastructure across the workloads that use it. The design keeps this math out of the host, so the
host stays free of Kubernetes (or any other domain's) semantics. That math belongs to a new kind of
plugin, an *allocator*.

This feature defines the allocator contract and makes it servable and discoverable through the
plugin SDK. It also gives both sides a way to trust the result. Hosts can check mechanically that no
money was lost or invented. Plugin authors get a conformance suite that exercises the cases where
allocators typically go wrong.

**Relationship to constitution principle III ("The Spec Consumes, It Does Not Calculate")**: this
feature adds no allocation math to the specification or to the SDK as a product feature. The
contract standardizes the *shape* of allocated cost and the invariants any allocator's output must
satisfy, which is the "standardized model for the final cost data" principle III asks for. The
calculation lives in allocator plugins, as principle III's rationale places it with specialized
providers. The only allocation logic in this repository is a minimal reference allocator that
exists to test the SDK and the conformance suite. It is not published as a product.

## Clarifications

### Session 2026-09-25

- Q: Which error code rejects mixed currencies among successfully priced resources? → A:
  invalid-argument. The request is inconsistent in itself, and every other input-validation
  failure (nonzero cost on an unpriced resource, a bad policy) already uses invalid-argument, so the
  request validator uses one code throughout. The finfocus SP2 plan's `FailedPrecondition` is to be
  aligned downstream.
- Q: How is a successfully priced resource with an empty currency handled? → A: It takes the single
  currency used by the other successfully priced resources. If every one is empty, the currency is
  `USD`. Output rows always carry the resolved currency explicitly, never empty, so hosts do not
  repeat the inference.
- Q: Can Go allocators name an unknown policy field's JSON path with the standard decoder? → A: No
  (verified on Go 1.27.1, including `GOEXPERIMENT=jsonv2`: `{"node_split":{"cpu":1}}` reports
  `unknown field "cpu"` with no path). The SDK provides a strict policy decoder that reports the
  path (FR-031), and the conformance suite asserts invalid-argument plus the field name, not an
  exact message.
- Q: Where do hosts get the conservation check and currency rule in production code? → A: From the
  plugin SDK package (FR-032). The testing package keeps the #506 names for plugin tests; the SDK
  wrappers delegate to it so both apply one rule.
- Q: Are duplicate priced resources valid? → A: No. Two priced resources with the same `kind` tag
  and `id` are an invalid request (invalid-argument), since an allocator keyed by node name would
  keep only one and fail conservation with a misleading error.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host Obtains a Cost Breakdown from an Allocator (Priority: P1)

The FinFocus host has usage rows from a usage source and a list of priced resources, each marked
as successfully priced or not. It sends both to an allocator plugin, together with the
organization's allocation policy (or none, to use the allocator's defaults), and receives cost rows
per workload. Each row has CPU and memory portions and a total. It also receives idle rows per node
for capacity no workload claimed, and cluster rows for shared infrastructure such as the control
plane. The response echoes the policy actually applied and a fingerprint of it, so the host can
show users which rules produced the numbers.

**Why this priority**: This is the step the whole Kubernetes cost breakdown depends on. Without it,
usage from #505 has no consumer and the host cannot show cost by namespace, controller, pod, node,
or label.

**Independent Test**: Serve the reference allocator through the SDK, send it a fixed two-node
cluster's usage and prices over each supported transport, and verify the rows, effective policy,
fingerprint, and warnings arrive intact and identical.

**Acceptance Scenarios**:

1. **Given** an allocator and a single priced node running two workloads, **When** the host
   requests allocation with no policy, **Then** it receives one row per workload, one idle row for
   the node, the allocator's default effective policy, and its fingerprint, and the row totals add
   up to the node's price.
2. **Given** three priced nodes and a priced control plane, **When** the host requests allocation,
   **Then** the response contains workload rows, one idle row per node, and a cluster row carrying
   the control plane's cost, and all totals add up to the sum of the four prices.
3. **Given** a node that could not be priced, **When** the host requests allocation, **Then** the
   workloads on that node receive zero-cost rows with a note, and the node's cost is excluded from
   the total the rows must add up to.
4. **Given** a policy document the allocator accepts, **When** the host requests allocation,
   **Then** the effective policy returned reflects the defaults with the document's overrides
   applied.
5. **Given** an allocator served by the SDK, **When** the host calls it over gRPC and separately
   over Connect, **Then** both calls return the same result, including the error code and message
   of failed calls.

---

### User Story 2 - Host Verifies an Allocator Did Not Lose or Invent Money (Priority: P1)

Before showing any allocated cost, the host checks that the allocator's rows add up to exactly what
was priced. Failed pricing is excluded, and floating-point rounding is tolerated. The host rejects
the result if the rows fall short, overshoot, mix currencies, or contain a negative cost. The check
comes from the SDK, so the host, the conformance suite, and every plugin's own tests apply exactly
the same rule.

**Why this priority**: The design's guiding rule is that a failure must never render as a plausible
number. An allocator bug that quietly drops or inflates cost would produce convincing but wrong
reports. This check is the only defense a domain-agnostic host has.

**Independent Test**: Feed the conservation check hand-built request and response pairs that
balance, overshoot, fall short, include unpriced resources, or have a zero total, and confirm it
accepts and rejects exactly as documented.

**Acceptance Scenarios**:

1. **Given** rows whose totals equal the sum of successfully priced costs, **When** the host checks
   conservation, **Then** the check passes.
2. **Given** rows totalling more than the priced sum by more than the tolerance, **When** checked,
   **Then** the check fails and states the expected total, the actual total, and the difference.
3. **Given** rows totalling less than the priced sum by more than the tolerance, **When** checked,
   **Then** the check fails likewise.
4. **Given** a request that includes an unpriced resource (with zero cost), **When** checked,
   **Then** that resource does not count toward the expected total.
5. **Given** a request where nothing was successfully priced and all rows are zero, **When**
   checked, **Then** the check passes despite the zero total.
6. **Given** a difference within the rounding tolerance, **When** checked, **Then** the check
   passes.

---

### User Story 3 - Plugin Developer Builds an Allocator with the SDK (Priority: P1)

A plugin developer implements a single allocate method on their plugin and starts it with the
standard SDK entry point. The SDK serves the allocator service alongside everything else (health
checks, tracing, interceptors) without extra wiring. It works the same way whether the plugin also
prices resources, reports usage, or does only allocation.

**Why this priority**: A contract the SDK cannot serve is a dead service. #505 established that
serving must ship with the protocol.

**Independent Test**: Write a minimal plugin that implements only the allocate method, serve it
with the SDK, and confirm the allocator service answers requests and reports healthy.

**Acceptance Scenarios**:

1. **Given** a plugin that implements the allocate method, **When** it is served with the SDK,
   **Then** the allocator service is registered on both transports and included in the Connect-mode
   health checker.
2. **Given** a plugin that does not implement the allocate method, **When** it is served, **Then**
   the allocator service is not registered and existing behavior is unchanged.
3. **Given** a plugin that implements both the usage-source and allocate methods, **When** it is
   served, **Then** both services are registered and answer independently.

---

### User Story 4 - Plugin Developer Proves an Allocator Is Correct (Priority: P2)

A plugin developer runs the SDK's allocator conformance suite against their implementation in
their own tests. The suite runs a standard set of scenarios as named subtests:

- single node, three nodes, and an empty cluster
- a fully packed node, an unpriced node, and a control plane
- workloads requesting more than a node can hold
- a policy with an unknown field, and a policy with an unknown version
- the empty request, and fingerprint stability

It reports which scenarios fail and why. A developer whose allocator passes has strong evidence it
will not be rejected by hosts.

**Why this priority**: Third-party allocators depend on this suite to catch the bugs the host would
otherwise reject at runtime. The host still verifies conservation without it, so it ranks below the
serving and verification stories.

**Independent Test**: Run the suite against the reference allocator (all pass) and against
deliberately broken allocators (each fails the scenario targeting its defect).

**Acceptance Scenarios**:

1. **Given** the reference allocator, **When** the conformance suite runs, **Then** every scenario
   passes.
2. **Given** an allocator that drops idle capacity, **When** the suite runs, **Then** the
   conservation and idle scenarios fail naming the shortfall.
3. **Given** an allocator that produces negative idle when workloads request more than a node holds,
   **When** the suite runs, **Then** the over-request scenario fails.
4. **Given** an allocator that silently ignores unknown policy fields, **When** the suite runs,
   **Then** the unknown-field scenario fails.
5. **Given** an allocator whose fingerprint differs between two identical calls, **When** the suite
   runs, **Then** the fingerprint-stability scenario fails.
6. **Given** an allocator whose fingerprint for an empty policy differs from its fingerprint for
   `{}`, **When** the suite runs, **Then** that scenario fails.

---

### User Story 5 - Host Discovers Allocators and Routes Correctly (Priority: P2)

The host inspects each plugin's metadata to find an allocator. An allocator advertises an
allocation capability. An allocation-only plugin must be distinguishable from a pricing plugin, so
the host never routes price queries to it.

**Why this priority**: Correct routing prevents confusing failures, but allocation still works when
the host is pointed at an allocator explicitly.

**Independent Test**: Query plugin metadata for an allocator with inferred capabilities and again
with explicitly declared capabilities, and verify the allocation capability and its legacy flag
appear in both cases.

**Acceptance Scenarios**:

1. **Given** a plugin that implements the allocate method, **When** the host asks for plugin info,
   **Then** the capability list includes allocation and the legacy metadata contains
   `supports_allocation=true`.
2. **Given** a plugin that explicitly declares only the allocation capability, **When** the host
   asks for plugin info, **Then** exactly that capability is reported.
3. **Given** the capability validity check, **When** it is asked about the allocation value (15),
   **Then** it reports it valid, and reports 16 invalid.
4. **Given** a plugin that implements the allocate method and declares no capabilities explicitly,
   **When** it starts, **Then** it is served normally and the SDK logs a warning that
   allocation-only plugins should declare capabilities explicitly.

---

### User Story 6 - TypeScript Consumers Call Allocators (Priority: P3)

A TypeScript client (browser or Node.js) needs to call allocators the same way it calls cost
sources and usage sources today.

**Why this priority**: Required by constitution principle XIII, but no TypeScript host of
allocation exists yet.

**Independent Test**: Use the TypeScript allocator client wrapper against a mocked allocator
service and verify an allocate request round-trips.

**Acceptance Scenarios**:

1. **Given** regenerated TypeScript bindings, **When** a developer uses the allocator client
   wrapper, **Then** they can call the allocate method with the same request shape as Go and receive
   rows, effective policy, fingerprint, and warnings.

---

### Edge Cases

- **Empty request**: no usage and no priced resources → no rows, but the effective policy and
  fingerprint are returned (hosts use this to display policy).
- **Priced resources but no usage**: every priced node's full cost is idle. The total still equals
  the priced sum.
- **Usage but nothing priced**: rows may be zero-cost with notes. The expected total is zero, and a
  zero total passes conservation.
- **Fully packed node**: an idle row is still emitted, with zero cost. Idle is never dropped.
- **Requests exceeding capacity**: idle never goes negative. The allocator scales workload shares
  so the node's cost is not exceeded.
- **Workload on a node with no priced resource** (for example Fargate, or an unknown provider):
  zero-cost row with a note explaining why.
- **Unpriced resource with nonzero cost**: an invalid request. It is rejected with invalid-argument
  before allocation.
- **Priced resource that is neither a node nor a control plane**: its cost still appears, as a
  cluster row with a note, so conservation holds.
- **Mixed currencies among successfully priced resources**: rejected with invalid-argument.
  Conservation across currencies is meaningless.
- **Empty currency on a successfully priced resource**: takes the single currency the others use.
  For example, `USD`, empty, and `USD` resolve to `USD`, while `USD`, empty, and `EUR` are still
  mixed and rejected. If all are empty, the currency is `USD`.
- **Unsuccessfully priced resources**: their currency is ignored, since they carry no cost.
- **Policy with an unknown field**: rejected with invalid-argument, and the message names the JSON
  path of the offending field (for example `node_split.cpu`). It never falls back to defaults.
- **Duplicate priced resources**: two priced resources with the same `kind` tag and `id` are
  rejected with invalid-argument before allocation.
- **Policy with an unknown or unsupported version**: rejected with invalid-argument.
- **Policy `{}` or omitted**: both mean "defaults" and produce identical effective policy and
  fingerprint.
- **Malformed policy JSON**: rejected with invalid-argument.
- **Cluster rows**: may put all cost in the total with zero CPU and memory portions. The rule that
  the total equals CPU plus memory is waived for cluster rows only.
- **Unspecified stats mode**: allowed. Allocation uses ratios of usage to capacity, so the result
  does not depend on the mode (see Assumptions).

## Requirements *(mandatory)*

### Functional Requirements

#### Contract

- **FR-001**: The protocol MUST define an allocator service, separate from the cost source and
  usage source services, with a single allocate operation.
- **FR-002**: An allocate request MUST carry usage rows (the same row type usage sources return),
  priced resources, an optional policy document, and the stats mode of the usage.
- **FR-003**: A priced resource MUST carry the existing resource descriptor, a cost for the
  normalized period, a currency, a flag saying whether pricing succeeded, and a note. A resource
  whose pricing failed MUST carry zero cost.
- **FR-004**: An allocate response MUST carry allocation rows, the effective policy applied, a
  fingerprint of that policy, and human-readable warnings.
- **FR-005**: Each allocation row MUST carry a subject (string map), a CPU cost, a memory cost, a
  total cost, a currency, and a note. The subject MUST carry a `kind` of `workload`, `__idle__`, or
  `__cluster__`. Idle rows MUST carry the `node` subject key. Subject keys follow the vocabulary
  #505 established.
- **FR-006**: The change MUST be additive only. Existing clients and plugins MUST keep working, and
  breaking-change detection MUST pass.

#### Allocation invariants (documented in the contract and enforced by SDK helpers)

- **FR-007 (Conservation)**: The sum of row totals MUST equal the sum of costs of successfully
  priced resources. The match is within a relative tolerance of one part per million, with an
  absolute floor of 1e-9 so that zero totals compare correctly.
- **FR-008**: Every row's total MUST equal its CPU cost plus its memory cost, within the same
  tolerance, except for cluster rows.
- **FR-009**: No row may carry a negative CPU, memory, or total cost.
- **FR-010**: Every successfully priced node MUST produce exactly one idle row, including when its
  idle cost is zero. Idle is never silently dropped.
- **FR-011**: All successfully priced resources and all rows MUST share one currency, the
  *resolved currency*:
  - **Resolution:** the resolved currency is the single non-empty currency among successfully
    priced resources, or `USD` when all of them are empty. A successfully priced resource with an
    empty currency takes the resolved currency.
  - **Rejection:** a request whose successfully priced resources carry more than one distinct
    non-empty currency MUST be rejected with invalid-argument.
  - **Rows:** every row MUST carry the resolved currency explicitly, never empty.

#### Policy

- **FR-012**: The policy document MUST be opaque to the host. Each allocator owns its policy
  schema. Every allocator's schema MUST include a top-level integer `version`.
- **FR-013**: An allocator MUST decode the policy strictly. It MUST reject an unknown field with
  invalid-argument naming the field's JSON path, reject an unknown `version` with invalid-argument,
  reject malformed JSON with invalid-argument, and merge accepted overrides onto its defaults. Go
  allocators satisfy the decoding rules through the SDK's strict policy decoder (FR-031); the
  `version` check stays with the allocator, which alone knows its supported versions.
- **FR-014**: An empty policy and `{}` MUST both yield the allocator's defaults and identical
  fingerprints.
- **FR-015**: The fingerprint MUST be a hex SHA-256 over the allocator's canonical form of the
  effective policy. It MUST be stable: the same effective policy always yields the same fingerprint
  from the same allocator. Fingerprints are NOT guaranteed comparable across different allocators.
- **FR-016**: A request with no usage and no priced resources MUST succeed with no rows, the
  effective policy, and its fingerprint.
- **FR-031**: The plugin SDK MUST provide a strict policy decoder that applies a JSON document onto a
  caller-supplied value already holding the allocator's defaults. It MUST treat an empty,
  whitespace-only, or `null` document as "no overrides"; reject unknown fields with an error naming
  the full JSON path (for example `node_split.cpu`); reject malformed JSON and trailing data; merge
  nested objects field by field; and replace arrays wholesale. The returned error is suitable for
  the allocator to return as invalid-argument.

#### SDK serving & discovery

- **FR-017**: The SDK MUST offer an optional allocator interface. When a plugin implements it, the
  SDK MUST serve the allocator service over both gRPC and Connect, include it in the Connect-mode
  health checker, and apply the same interceptors the cost service gets on each transport.
- **FR-018**: Errors returned by an allocator MUST reach clients with the same code and message
  over both transports.
- **FR-019**: Plugins implementing the allocator interface MUST have the allocation capability
  (value 15) inferred automatically, with the legacy metadata flag `supports_allocation`.
- **FR-020**: The capability validity check MUST accept 15 as the new upper bound and reject 16.
- **FR-021**: Capability inference MUST stay otherwise unchanged. The SDK MUST document that
  allocation-only plugins declare capabilities explicitly. It MUST log a startup warning when a
  plugin implements the allocator interface but configures no explicit capability list, matching
  the usage-source behavior from #505.

#### Testing support

- **FR-022**: The testing package MUST provide a conservation check that takes a request, a
  response, and a relative tolerance. It applies FR-007 (ignoring unsuccessfully priced resources)
  and reports the expected total, the actual total, and the difference on failure.
- **FR-023**: The testing package MUST provide a response validation helper that enforces FR-005
  and FR-008 through FR-011. It rejects: rows without a `kind` or with an unknown kind, idle rows
  without `node`, negative costs, a total not matching CPU plus memory on non-cluster rows, a row
  whose currency is empty or differs from the request's resolved currency, a missing idle row for a
  successfully priced node, a missing fingerprint, and a missing effective policy.
- **FR-024**: The testing package MUST provide a request validation helper that rejects a nil
  request, an unsuccessfully priced resource with nonzero cost, a negative cost, more than one
  distinct non-empty currency among successfully priced resources, and two priced resources with
  the same `kind` tag and `id`. Every rejection uses invalid-argument. The SDK MUST also expose
  the currency resolution rule from FR-011, so hosts and allocators compute the same resolved
  currency. Allocators can use these helpers to enforce input rules.
- **FR-025**: The testing package MUST provide an allocator conformance suite that runs, as named
  subtests, the scenarios in User Story 4. Each scenario's assertions MUST hold for any correct
  allocator regardless of its policy defaults: conservation, validity, and idle presence, not
  specific cost values. Policy-rejection scenarios assert the invalid-argument code and that the
  message contains the offending field name, not an exact message.
- **FR-026**: The in-memory test harness MUST let plugin tests call the allocate method.
- **FR-027**: The repository MUST include a minimal reference allocator, used only by the SDK's own
  tests, that passes the conformance suite.
- **FR-032**: The plugin SDK package (not only the testing package) MUST expose the conservation
  check (FR-022), the request validation (FR-024), and the currency resolution rule (FR-011), so
  hosts can call them from production code without importing test tooling. These MUST delegate to
  the testing package's implementations, so hosts, allocators, and the conformance suite apply one
  rule.

#### Multi-language & docs

- **FR-028**: TypeScript bindings MUST be regenerated and an allocator client wrapper added, with
  tests covering the allocate call.
- **FR-029**: The README capability table, the root README's service descriptions, the TypeScript
  SDK README, and the plugin developer guide MUST document the allocator service. The developer
  guide MUST gain a "Writing an allocator" section covering the invariants, policy rules,
  explicit capabilities, and the conformance suite.
- **FR-030**: The contract's inline comments MUST state every invariant (FR-007 through FR-016) and
  the error semantics.

### Key Entities

- **Allocator Service**: The new plugin-facing contract, with one allocate operation.
- **Allocate Request**: Usage rows, priced resources, an opaque policy document, and the stats mode.
- **Priced Resource**: A resource descriptor plus cost for the period, currency, a "pricing
  succeeded" flag, and a note. It is the host's output from pricing a usage source's priceable
  entries.
- **Allocate Response**: Allocation rows, effective policy, policy fingerprint, and warnings.
- **Allocation Row**: A subject map plus CPU, memory, and total cost, currency, and note. It is the
  atom of allocated cost. Its kinds are workload, idle (per node), and cluster (shared
  infrastructure).
- **Allocation Policy**: An allocator-owned, versioned JSON document. The host treats it as opaque
  bytes.
- **Policy Fingerprint**: A stable hex SHA-256 of the canonical effective policy, scoped to one
  allocator.
- **Allocation Capability**: A new plugin capability (value 15) with legacy flag
  `supports_allocation`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin author can go from an empty plugin to a served, health-checked allocator by
  embedding the SDK's base plugin, implementing exactly one method, and using the standard entry
  point, with no additional registration code.
- **SC-002**: 100% of allocate calls to an SDK-served allocator behave identically over both
  supported transports in the SDK's transport parity tests, including the error code and message
  of failed calls.
- **SC-003**: The conformance suite rejects 100% of a set of deliberately broken allocators. The
  set covers over-allocation, under-allocation, dropped idle, negative idle, ignored unknown policy
  fields, accepted unknown versions, and unstable fingerprints. Each broken allocator fails at least
  the scenario that targets its defect.
- **SC-004**: The reference allocator passes 100% of conformance scenarios.
- **SC-005**: Every rejection rule in FR-022 through FR-024 has at least one failing and one
  passing test case. The helpers produce zero false rejections on the documented valid examples,
  including zero totals, unpriced resources, and cluster rows.
- **SC-006**: Zero existing plugins, clients, or tests change behavior. The full existing test
  suite and breaking-change detection pass unchanged.
- **SC-007**: Go and TypeScript expose the allocate operation in the same release.
- **SC-008**: The planned Kubernetes allocator (finfocus SP2) can be validated against the
  conformance suite from a released SDK version with no local overrides. This is verified
  downstream after release.

## Assumptions

- The allocation math, including CPU/memory splitting, share scaling when requests exceed capacity,
  and idle computation, belongs to each allocator. The reference allocator uses the simplest method
  that satisfies the invariants and is not a template for production allocators.
- Allocation relies on ratios of usage to capacity, so the result is independent of the stats mode
  and of whether costs are monthly or for a window. The mode is passed through for allocators that
  choose to use it.
- The currency rule is part of the protocol, not allocator-specific: conservation cannot be
  verified across currencies. Existing cost messages describe currency as an ISO 4217 code but
  define no default for an empty value. The `USD` fallback in FR-011 applies only inside the
  allocator contract and does not change the meaning of other messages.
- The tolerance floor (1e-9 absolute) matches what finfocus core (SP3) plans. Core calls the plugin
  SDK's conservation check (FR-032) rather than its own, so the two never disagree.
- Fingerprints are compared only for the same allocator (policy display and change detection).
  A cross-language canonical JSON standard is out of scope.
- CPU and memory are the only cost categories in this version. GPU, storage, and network categories
  would be an additive change later. This version does not need to reserve fields for them.
- Idle and shared-cost redistribution policies, `__shared__`/`__unallocated__` kinds, and
  pre-allocated external sources (OpenCost, SP6) are out of scope.
- No concrete production allocator, CLI command, grouping, or rendering is in scope. Those live in
  finfocus (SP2, SP3).
- Shipping this alongside, or soon after, the release that includes `GetStats` is acceptable. The
  maintainer may prototype the Kubernetes allocator against the branch before that release.
- The identifier names proposed in #506 (`AllocatorProvider`, `CheckConservation`,
  `RunAllocatorConformance`) are kept, because finfocus plans are written against them. finfocus
  plans also code against these proposed names for the new helpers, which the plan should keep:
  `pluginsdk.DecodePolicy(data []byte, target any) error` (FR-031),
  `pluginsdk.CheckConservation(req, resp, relEpsilon)` (FR-032),
  `pluginsdk.ValidateAllocateRequest(req) error` (FR-032), and
  `pluginsdk.ResolveCurrency(priced []*pbc.PricedResource) (string, error)` (FR-011, FR-032).
- Hosts calling `Supports` on allocation-only plugins is the same separate SDK concern tracked for
  usage-only plugins (#507).

## Dependencies

- Usage source contract from #505 (usage rows, stats mode, subject keys, reserved `__idle__` and
  `__cluster__` kinds), merged on `main` and not yet in a tagged release.
- Existing resource descriptor message.
- Plugin SDK serving, capability inference, legacy capability mapping, and Connect error
  conversion.
- Design reference: rshade/finfocus `docs/superpowers/specs/2026-09-24-k8s-cost-allocation-design.md`.
