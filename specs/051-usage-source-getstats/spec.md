# Feature Specification: Usage Source Service (GetStats)

**Feature Branch**: `051-usage-source-getstats`

**Created**: 2026-09-25

**Status**: Draft

**Input**: User description: "#505 — feat(proto): add UsageSourceService.GetStats for workload usage plugins"

## Overview

FinFocus is adding in-cluster Kubernetes cost allocation. Allocation joins **usage** (how much CPU and
memory each workload requests or consumes) with **prices** (what the underlying nodes cost, already
available from existing cost-source plugins). Usage comes from a new kind of plugin, a *usage source*,
backed by the Kubernetes API, Prometheus, Datadog, or similar systems. Usage sources have no prices,
so they get their own service contract rather than a stubbed-out cost-source contract.

This feature defines that contract, makes it servable and discoverable through the plugin SDK, and
gives plugin authors the tools to verify their responses. Allocation itself is tracked separately
(#506).

## Clarifications

### Session 2026-09-25

- Q: How should the SDK handle usage-only plugins that inherit pricing capabilities by inference
  (FR-013)? → A: Document that usage-only plugins must declare capabilities explicitly, and log a
  startup warning when a plugin implements the usage-source interface without an explicit capability
  list. Inference is unchanged, because a plugin may legitimately both price resources and report
  usage.
- Q: How does the selector distinguish a namespace from label filters? → A: `namespace` is a
  reserved selector key; every other key is an exact-match label filter.
- Q: What do two rows with the same subject and metric mean? → A: They are invalid; the validation
  helper rejects them. Sources aggregate before responding.
- Q: Is `kind` required on every row? → A: Yes; a row without `kind` is rejected.
- Q: In historical mode, are node allocatable rows integrated too? → A: Yes; every amount in a
  historical response, including `cpu_allocatable`/`mem_allocatable`, is integrated
  resource-hours, so usage and capacity stay directly comparable.
- Q: Can a `kind=node` row omit the `node` key? → A: No (added during plan review); without it the
  row's capacity cannot be joined to a priceable node, so the validator rejects it.
- Q: Must error codes match across transports? → A: Yes (added during plan review); connect-go
  reports gRPC `status` errors as `Unknown`, so the SDK converts them explicitly (research R8).
- Q: Does the SDK validate stats requests? → A: Yes (added during analysis). Constitution I
  requires validation for every message, and every existing request has a `Validate*Request`
  helper, so the testing package gains `ValidateGetStatsRequest` for the window rules (research R11).
- Q: Can hosts always tell a usage-only plugin from a pricing plugin? → A: Only when the plugin
  declares capabilities explicitly (added during analysis). Inference always adds the four pricing
  capabilities, so an inferred usage-only plugin looks like a pricing plugin; the startup warning
  (FR-013) exists to push authors to the explicit form.
- Q: Do interceptors apply to the usage service in Connect mode? → A: It gets exactly what the cost
  service gets (added during analysis). In gRPC mode that is the tracing interceptor plus
  `ServeConfig.UnaryInterceptors`; in Connect mode the cost service has no interceptors today, so
  neither does the usage service.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host Retrieves Workload Usage from a Usage Source (Priority: P1)

The FinFocus host needs to know, for a given cluster scope, how much CPU and memory each workload
requests and which priceable resources (nodes, control plane) that usage runs on. It calls a usage
source plugin, asking either for current per-hour run rates or for usage integrated over a past time
window, optionally narrowed to a namespace or label selector. It receives usage rows keyed by
subject (cluster, namespace, controller, pod, node, labels), plus a list of priceable resources in
the same descriptor form the host already prices through cost-source plugins.

**Why this priority**: This is the contract the whole allocation feature depends on. Without it, the
host has no standard way to obtain usage and cannot join usage with prices.

**Independent Test**: Serve a reference usage source that returns fixed rows through the SDK, call it
from a client over each supported transport, and verify the rows, priceable resources, mode, and
warnings arrive intact.

**Acceptance Scenarios**:

1. **Given** a usage source serving a two-node cluster, **When** the host requests stats with no time
   window, **Then** the response is marked run-rate and contains per-hour `cpu_request` (core) and
   `mem_request` (GiB) rows per workload, `cpu_allocatable`/`mem_allocatable` rows per node, and one
   priceable entry per priceable node whose identifier equals that node's `node` subject value.
2. **Given** the same source, **When** the host requests stats for a start and end time, **Then** the
   response is marked historical and all amounts, including node allocatable rows, are integrated
   resource-hours (`core-hours`, `GiB-hours`).
3. **Given** a selector of `namespace=payments`, **When** the host requests stats, **Then** only
   workload rows in that namespace are returned.
4. **Given** a source that supports only run-rate, **When** the host requests a historical window,
   **Then** the call fails with an invalid-argument error.
5. **Given** a source whose credentials lack permission to list pods, **When** the host requests
   stats, **Then** the call fails with a permission-denied error that names the missing verb and
   resource.
6. **Given** a usage source served by the SDK, **When** the host calls it over gRPC and separately
   over Connect, **Then** both calls return the same result.

---

### User Story 2 - Plugin Developer Builds a Usage Source with the SDK (Priority: P1)

A plugin developer writing a Prometheus-backed usage source implements a single stats method on their
plugin and starts it with the standard SDK entry point. The SDK serves the usage service alongside
everything else (health checks, tracing, interceptors) without extra wiring. Shared constants for
subject keys, kinds, and metric names keep their output consistent with every other usage source.

**Why this priority**: The protocol is unusable in practice unless the SDK can serve it; the issue
explicitly calls out that "protos alone ship a dead service".

**Independent Test**: Write a minimal plugin that implements only the stats method, serve it with the
SDK, and confirm the usage service answers requests and reports healthy.

**Acceptance Scenarios**:

1. **Given** a plugin that implements the stats method, **When** it is served with the SDK, **Then**
   the usage service is registered on both transports and included in the Connect-mode health
   checker (gRPC mode has no health service).
2. **Given** a plugin that does not implement the stats method, **When** it is served, **Then** the
   usage service is not registered and existing behaviour is unchanged.
3. **Given** a plugin author using the SDK's subject, kind, and metric constants, **When** they build
   rows, **Then** the keys and names match the documented vocabulary exactly.

---

### User Story 3 - Host Discovers Usage Sources and Routes Correctly (Priority: P2)

The host inspects each plugin's metadata to decide what to ask it. A usage source must advertise a
usage-stats capability so the host knows to call it for usage. A usage-only plugin must not appear to
offer pricing, so the host never routes price queries to a plugin that cannot answer them.

**Why this priority**: Correct routing prevents hard-to-diagnose failures in the host, but the service
still works when the host is configured manually.

**Independent Test**: Query plugin metadata for a usage-stats plugin with inferred capabilities and
again with explicitly declared capabilities; verify the capability appears in both cases along with
its legacy metadata flag.

**Acceptance Scenarios**:

1. **Given** a plugin that implements the stats method, **When** the host asks for plugin info,
   **Then** the capability list includes usage-stats and the legacy metadata contains
   `supports_usage_stats=true`.
2. **Given** a plugin that explicitly declares only the usage-stats capability, **When** the host
   asks for plugin info, **Then** exactly that capability is reported and no pricing capabilities
   appear.
3. **Given** the capability validity check, **When** it is asked about the new usage-stats value,
   **Then** it reports it as valid, and a value one above it as invalid.
4. **Given** a plugin that implements the stats method and declares no capabilities explicitly,
   **When** it starts, **Then** it is served normally with inferred capabilities and the SDK logs a
   warning that usage-only plugins should declare capabilities explicitly.

---

### User Story 4 - Plugin Developer Verifies Responses in Tests (Priority: P2)

A plugin developer wants confidence that their usage source produces well-formed output before the
host consumes it. They call a validation helper on their responses in unit tests, and use an
in-memory test harness to call their stats method end-to-end without a network.

**Why this priority**: Malformed usage (a misspelled subject key, a node that can never be joined to
its price) silently corrupts allocation downstream. Catching it in plugin tests is far cheaper.

**Independent Test**: Feed the validation helper a set of good and bad responses and check each
rejection rule fires, and only then.

**Acceptance Scenarios**:

1. **Given** a response with a subject key that is neither documented nor `label.`-prefixed, **When**
   validated, **Then** it is rejected naming the key.
2. **Given** a response with a negative amount, **When** validated, **Then** it is rejected.
3. **Given** a row whose `kind` is not `workload` or `node`, **When** validated, **Then** it is
   rejected.
4. **Given** a priceable entry with no identifier, **When** validated, **Then** it is rejected.
5. **Given** a priceable node whose identifier matches no row's `node` subject, **When** validated,
   **Then** it is rejected.
6. **Given** a node that reports capacity rows but has no priceable entry (for example Fargate or an
   unknown provider), **When** validated, **Then** it is accepted.
7. **Given** a row with no `kind` subject key, **When** validated, **Then** it is rejected.
8. **Given** two rows with the same subject and metric, **When** validated, **Then** they are
   rejected as duplicates.
9. **Given** a `kind=node` row with no `node` subject key, **When** validated, **Then** it is
   rejected, because its capacity could never be joined to a price.
10. **Given** a usage source under test, **When** the developer starts the in-memory harness, **Then**
    they can call the stats method and receive its response.

---

### User Story 5 - TypeScript Consumers Call Usage Sources (Priority: P3)

A TypeScript client (browser or Node.js) needs to query usage sources the same way it queries cost
sources today.

**Why this priority**: Required by the constitution's SDK-synchronization principle, but no
TypeScript consumer of usage data exists yet.

**Independent Test**: Use the TypeScript client wrapper against a mocked usage service and verify a
stats request round-trips.

**Acceptance Scenarios**:

1. **Given** regenerated TypeScript bindings, **When** a developer uses the usage-source client
   wrapper, **Then** they can call the stats method with the same request shape as Go.

---

### Edge Cases

- **Partial time window**: only one of start/end set → invalid-argument error (mode cannot be
  determined).
- **Inverted window**: start after end → invalid-argument error.
- **Unspecified mode in response**: a response that does not state run-rate or historical is
  rejected by the validation helper, since amounts are uninterpretable without it.
- **Empty cluster or selector matching nothing**: valid, empty rows, no error.
- **Unauthenticated source**: unauthenticated error, distinct from permission-denied.
- **Unknown metric requested**: the source ignores it and adds a warning rather than failing the
  whole call.
- **Label keys**: arbitrary `label.<key>` subject keys are always accepted, including keys containing
  dots or slashes (e.g. `label.app.kubernetes.io/name`).
- **Unpriceable nodes**: capacity rows with no matching priceable entry are valid (see Story 4,
  scenario 6).
- **Allocator-only kinds**: `__idle__` and `__cluster__` are allocation output kinds (#506) and are
  rejected in usage-source rows.
- **Control plane**: a priceable entry tagged `kind=cluster` is not a node and needs no matching
  `node` subject.
- **Duplicate rows**: two rows with an identical subject map and metric are invalid; a source that
  reads per-container data aggregates to one row per subject and metric before responding.
- **Label named `namespace`**: cannot be filtered through the selector, because `namespace` is a
  reserved selector key.

## Requirements *(mandatory)*

### Functional Requirements

#### Contract

- **FR-001**: The protocol MUST define a usage source service, separate from the cost source service,
  with a single stats operation.
- **FR-002**: A stats request MUST carry a scope (such as a cluster identifier or kubeconfig
  context), an optional start and end time, a selector map, and an optional list of metrics (empty
  meaning source defaults). In the selector, `namespace` is a reserved key restricting workloads to
  that namespace; every other key is an exact-match label filter, and all entries must match.
- **FR-003**: A stats response MUST carry usage rows, priceable resources, the mode actually served
  (run-rate or historical), and human-readable warnings.
- **FR-004**: Each usage row MUST carry a subject (string map), metric name, non-negative amount, and
  unit. Amounts are per-hour rates in run-rate mode and integrated resource-hours in historical mode;
  this applies to every row, including node allocatable rows. Every row MUST carry a `kind`, and a
  subject and metric pair MUST appear at most once per response.
- **FR-005**: Priceable resources MUST reuse the existing resource descriptor so hosts price them
  through existing cost-source plugins unchanged.
- **FR-006**: Subject keys, kinds, metric names, units, priceable tagging, and error semantics MUST
  be documented in the contract's inline comments and in `docs/`.
- **FR-007**: The change MUST be additive only; existing clients and plugins MUST keep working, and
  breaking-change detection MUST pass.

#### Error semantics

- **FR-008**: A source that cannot serve the requested mode, or receives a partial or inverted time
  window, MUST return invalid-argument.
- **FR-009**: Permission failures MUST return permission-denied naming the missing verb and resource;
  missing credentials MUST return unauthenticated.

#### SDK serving & discovery

- **FR-010**: The SDK MUST offer an optional usage-source interface. When a plugin implements it, the
  SDK MUST serve the usage service over both gRPC and Connect, include it in the Connect-mode health
  checker, and apply the same interceptors as the cost service. In gRPC mode that means the
  built-in tracing interceptor followed by `ServeConfig.UnaryInterceptors`. In Connect mode it means
  the same handler options as the cost handler, which today carry no interceptors.
- **FR-011**: Plugins implementing the usage-source interface MUST have a usage-stats capability
  (value 14) inferred automatically, with the legacy metadata flag `supports_usage_stats`.
- **FR-012**: The capability validity check MUST accept value 14 as the new upper bound (moving to 15
  when #506 lands).
- **FR-013**: Capability inference MUST stay unchanged (a plugin may both price and report usage).
  The SDK MUST document that usage-only plugins declare capabilities explicitly, and MUST log a
  warning at startup when a plugin implements the usage-source interface but no explicit capability
  list is configured.
- **FR-014**: The SDK MUST export named constants for all documented subject keys, the `label.`
  prefix, kinds (`workload`, `node`, `__idle__`, `__cluster__`), and metric names (`cpu_request`,
  `mem_request`, `cpu_allocatable`, `mem_allocatable`, `cpu_usage`, `mem_usage`).

#### Testing support

- **FR-015**: The testing package MUST provide a response validation helper that rejects: unknown
  subject keys that are not `label.`-prefixed, negative amounts, rows with a missing `kind` or a
  `kind` other than `workload` or `node`, `kind=node` rows without a non-empty `node` subject key,
  duplicate rows (same subject and metric), priceable
  entries without an identifier, priceable `kind=node` entries whose identifier matches no row's
  `node` subject, and responses with an unspecified mode.
- **FR-016**: The validation helper MUST accept nodes that report capacity without a priceable entry.
- **FR-017**: The in-memory test harness MUST let plugin tests call the stats method.
- **FR-020**: The testing package MUST provide a stats request validation helper that rejects a nil
  request, a window with only one of start and end set, and a window whose start is after its end.
  A request with neither start nor end set (run-rate) is valid. Sources can use it to enforce
  FR-008's window rules.

#### Multi-language & docs

- **FR-018**: TypeScript bindings MUST be regenerated and a usage-source client wrapper added.
- **FR-019**: The README capability table and the plugin developer guide MUST document the new
  capability and the rule for usage-only plugins. The root README's service descriptions and the
  TypeScript SDK README MUST describe the usage source service and its client wrapper.

### Key Entities

- **Usage Source Service**: The new plugin-facing contract; one stats operation.
- **Stats Request**: Scope, optional time window, selector, requested metrics.
- **Stats Response**: Usage rows, priceable resources, served mode, warnings.
- **Usage Row**: Subject map + metric + amount + unit; the atom of usage.
- **Subject**: String map identifying what the usage belongs to (`cluster`, `namespace`,
  `controller_kind`, `controller`, `pod`, `node`, `label.<key>`, `kind`). Strings keep grouping
  flexible and non-Kubernetes sources schema-free.
- **Stats Mode**: Run-rate (point-in-time per-hour) or historical (integrated over the window).
- **Priceable Resource**: Existing resource descriptor for a node (`tags.kind=node`, identifier =
  node name, plus `provider_id` and `capacity_type` tags) or control plane (`tags.kind=cluster`).
- **Usage-Stats Capability**: New plugin capability (value 14) and legacy flag
  `supports_usage_stats`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin author can go from an empty plugin to a served, health-checked usage source by
  embedding the SDK's base plugin, implementing exactly one method, and using the standard entry
  point — no additional registration code.
- **SC-002**: 100% of stats calls to an SDK-served usage source behave identically over both
  supported transports in the SDK's transport parity tests, including the error code and message of
  failed calls.
- **SC-003**: Every rejection rule in FR-015 and FR-020 has at least one failing and one passing
  test case, and the validation helpers produce zero false rejections on the documented valid
  examples (including unpriceable nodes, control-plane entries, and run-rate requests).
- **SC-004**: Zero existing plugins, clients, or tests change behaviour: the full existing test suite
  and breaking-change detection pass unchanged.
- **SC-005**: A host can tell a usage-only plugin from a pricing plugin using plugin metadata alone
  whenever the plugin declares its capabilities explicitly, as the documentation requires of
  usage-only plugins. A usage-only plugin that relies on inference advertises pricing capabilities
  too; in that configuration the SDK logs the FR-013 startup warning, and the host cannot tell the
  difference.
- **SC-006**: Go and TypeScript expose the stats operation in the same release.

## Assumptions

- Capability 14 is reserved for usage stats; #506 takes 15 and moves the upper bound again when it
  lands. If #506 merges first, numbers are unaffected; only the bound update order changes.
- `__idle__` and `__cluster__` kinds are defined here for #506 to reuse but are invalid in usage-source
  output.
- The validation helper does not check unit/metric consistency or duplicate priceable identifiers;
  sources may emit custom metrics and units beyond the documented set.
- `cpu_usage`/`mem_usage` are named now so sources can adopt them later; nothing requires them.
- The TypeScript SDK mirrors the Go subject/kind/metric constants, following the usage-profile
  precedent.
- Error semantics (FR-008, FR-009) are the source's responsibility. The SDK passes status codes
  through unchanged on both transports, and the SDK's transport parity tests verify this against a
  reference implementation. The plugin conformance suite (`RunBasicConformance` and the other
  levels) is not extended for usage sources; `ValidateStatsResponse` and `ValidateGetStatsRequest`
  fill that role.
- No concrete usage source (Kubernetes, Prometheus, Datadog) and no allocation logic are in scope.
- The plan keeps the identifier names proposed in #505 (`UsageSourceProvider`,
  `ValidateStatsResponse`, and the `Subject*`, `Kind*`, and `Metric*` constants), because finfocus
  core and the kubernetes plugin are already planned against them.
- Hosts calling `Supports` on usage-only plugins is a separate SDK concern, tracked outside this
  feature (#507).

## Dependencies

- Existing resource descriptor message (reused for priceable resources).
- Plugin SDK serving, capability inference, and legacy capability mapping.
- Companion issue #506 (allocation service), which consumes this output and shares the capability
  upper bound.
