# Feature Specification: ResolveResourceTypes Hardening

**Feature Branch**: `050-resolve-resource-types-hardening`
**Created**: 2026-09-07
**Status**: Draft
**Input**: User description: "Harden the ResolveResourceTypes RPC shipped in spec 049-resolve-resource-types. Six follow-up items: (1) batch API + docs for property_mappings, (2) request size limit on source_types, (3) CloudFormation example in data-model docs, (4) expires_at caching hint on the response, (5) conformance/MockPlugin coverage plus hit/miss metrics, (6) a JSON mapping-file loader for TypeRegistry. All additive/backward-compatible; only item 4 touches the proto."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Core Is Protected From Oversized Type-Resolution Requests (Priority: P1)

A plugin receives a `ResolveResourceTypes` request where `source_types` contains an
unbounded or excessively large number of entries (malformed client, buggy core
integration, or a hostile caller). Today the plugin SDK and the TypeScript client both
process the list with no upper bound, unlike the sibling `BatchCost` RPC, which already
enforces a configurable maximum. The SDK must reject an oversized request before doing
any resolution work, regardless of which of the three response paths (custom provider,
`TypeRegistry`, or the empty fallback) would have served it.

**Why this priority**: This is the only genuine security/robustness gap of the six —
every other item is ergonomics, observability, or documentation. Left unfixed, a
malformed or malicious request can force unbounded allocation work in the plugin
process.

**Independent Test**: Send a `ResolveResourceTypesRequest` with more entries than the
configured maximum and verify the plugin returns an `InvalidArgument` error without
invoking a configured provider or registry lookup; send one at or under the limit and
verify normal resolution still occurs.

**Acceptance Scenarios**:

1. **Given** a plugin configured with the default limit, **When** a request contains
   more `source_types` entries than the limit, **Then** the plugin returns an
   `InvalidArgument` error identifying the limit and the request size.
2. **Given** the same oversized request, **When** the plugin has a custom
   `ResolveResourceTypesProvider` implementation configured, **Then** the provider is
   never invoked — the limit is enforced before any dispatch path.
3. **Given** a request at or under the configured limit, **When** processed, **Then**
   resolution proceeds exactly as before this change (no behavior change for
   well-formed requests).
4. **Given** a TypeScript client, **When** `resolveResourceTypes()` is called with an
   oversized request, **Then** the client rejects it locally before making the RPC call,
   matching the existing `batchCost()` client-side validation pattern.

---

### User Story 2 - Core Can Cache Type Resolutions Instead of Re-Querying Every Time (Priority: P2)

The core queries a plugin's `ResolveResourceTypes` RPC every time it ingests a
Terraform state file. Because IaC-type-to-Pulumi-token mappings almost never change
between calls (they change only when a plugin's own registry or version changes), the
core has no signal today for how long a response can safely be reused, unlike the
existing `GetActualCost`, `GetProjectedCost`, and `EstimateCost` responses, which all
carry an advisory `expires_at` caching hint. Plugin authors using the declarative
`TypeRegistry` should be able to set one default caching duration for all their
resolutions rather than stamping it by hand on every response.

**Why this priority**: Directly reduces redundant RPC traffic for a resource that is
unusually cacheable, but it is an efficiency improvement, not a correctness or safety
fix, so it ranks below the request-size limit.

**Independent Test**: Configure a `TypeRegistry` with a default TTL, call `Resolve`,
and verify the returned response's `expires_at` is set to approximately now-plus-TTL;
verify a registry without a configured TTL still returns a response with no
`expires_at` set (unchanged behavior).

**Acceptance Scenarios**:

1. **Given** a `ResolveResourceTypesResponse`, **When** the plugin does not set
   `expires_at`, **Then** the field is absent/nil, signaling the caller must not assume
   any cache lifetime (matches existing caching-hint convention).
2. **Given** a `TypeRegistry` configured with a default TTL, **When** `Resolve` is
   called, **Then** the returned response's `expires_at` reflects that TTL from the
   call time.
3. **Given** a `ResolveResourceTypesResponse` with a past `expires_at`, **When** a
   caller checks its validity, **Then** it is treated as stale, consistent with how
   existing cost RPCs treat a past `expires_at`.

---

### User Story 3 - Plugin Authors Can Verify Type-Resolution Support and Operators Can Observe It (Priority: P2)

A plugin developer wants to verify their plugin's `ResolveResourceTypes` support (or
lack thereof) using the SDK's standard conformance suite, the same way they already
verify `GetRecommendations` support. An operator running plugins in production wants to
see, via existing Prometheus metrics, how often requested resource types are actually
being resolved versus falling through to heuristic conversion, to judge whether a
plugin's mapping data needs expansion.

**Why this priority**: Closes an inconsistency with how every other optional
capability in this SDK is validated and observed, improving trust in the ecosystem, but
it does not change wire behavior for any existing caller.

**Independent Test**: Run the SDK's Basic conformance tier against a plugin that does
not implement type resolution and verify it passes (empty-response contract); run it
against the bundled mock plugin (which will implement the interface) and verify
returned mappings are well-formed. Separately, drive a `ResolveResourceTypes` call
through the metrics-instrumented server and verify the resolved/unresolved counters
change by the expected amounts.

**Acceptance Scenarios**:

1. **Given** a plugin that does not implement `ResolveResourceTypesProvider`, **When**
   Basic conformance tests run, **Then** the ResolveResourceTypes tests pass (empty
   response is a valid outcome, not a failure).
2. **Given** the SDK's mock plugin, configured with known type mappings, **When** Basic
   conformance tests run, **Then** the ResolveResourceTypes tests verify every returned
   mapping has a non-empty Pulumi token and that unknown types are correctly omitted.
3. **Given** a metrics-instrumented plugin server, **When** a `ResolveResourceTypes`
   call resolves 3 of 5 requested types, **Then** the resolved counter increases by 3
   and the unresolved counter increases by 2, and no counter changes on an error
   response.

---

### User Story 4 - Plugin Authors Register Property Overrides in Bulk (Priority: P3)

A plugin developer has dozens of Terraform-to-Pulumi type mappings, a handful of which
also need non-mechanical property name overrides (the existing per-entry
`RegisterMappingWithProperties` method only registers one mapping at a time). They want
a batch method comparable to the existing bulk `RegisterMappings`, and they want the
SDK documentation to actually describe how property overrides work and when to use
them — today the README only documents the plain (no-override) batch path.

**Why this priority**: Pure ergonomics/documentation improvement for an already-shipped
storage mechanism (`property_mappings` is stored and round-trips correctly today); no
behavior is currently broken.

**Independent Test**: Register several mappings via the new batch method, some with
property overrides and some without, then call `Resolve` and verify each mapping's
overrides (or absence thereof) are returned correctly.

**Acceptance Scenarios**:

1. **Given** a batch of mappings with mixed property-override presence, **When**
   registered via the new batch method and resolved, **Then** each returned mapping's
   `property_mappings` matches exactly what was registered for it (empty where none was
   given).
2. **Given** the SDK README, **When** a plugin author looks for guidance on property
   overrides, **Then** they find a documented example distinguishing a mechanical
   rename (needs no override) from a non-mechanical one (needs an override).

---

### User Story 5 - Plugin Communities Maintain Mapping Data as Files, Not Hardcoded Registrations (Priority: P3)

A plugin author (or a separate community-maintained mapping project) wants to keep a
large Terraform-to-Pulumi mapping table as a versioned data file that can be updated,
reviewed, and shipped independently of a Go code change, instead of hand-writing
hundreds of `RegisterMapping` calls. finfocus-spec itself does not — and per this
project's own architecture, should not — ship or maintain any provider-specific mapping
data; it provides the loading mechanism only.

**Why this priority**: Meaningful ergonomics win for plugin authors with large mapping
sets, but no existing consumer is blocked without it, and it introduces no wire-level
change.

**Independent Test**: Write a small JSON mapping file, load it into a `TypeRegistry`,
and verify the resulting registry resolves the same way a hand-coded registration
would; verify malformed input (bad JSON, unknown source format, missing required field)
fails with a clear error rather than silently registering nothing.

**Acceptance Scenarios**:

1. **Given** a well-formed JSON mapping file for one source format, **When** loaded
   into a `TypeRegistry`, **Then** every entry is resolvable exactly as if it had been
   registered via `RegisterMapping`/`RegisterMappingWithProperties` calls.
2. **Given** a JSON file with an unrecognized `source_format` value, **When** loading is
   attempted, **Then** the load fails with a descriptive error and no entries are
   registered.
3. **Given** a registry that already has mappings for a format, **When** a file is
   loaded for the same format with an overlapping key, **Then** the file's entry wins
   (last-write-wins merge).

---

### User Story 6 - CloudFormation-Supporting Plugin Authors Have a Concrete Example (Priority: P4)

A future plugin author implementing `SOURCE_FORMAT_CLOUDFORMATION` support looks at the
data model documentation for `source_types` and finds a concrete Terraform example
(`aws_instance`) but no equivalent CloudFormation example, even though the field
description already refers to "CloudFormation logical resource types."

**Why this priority**: Documentation-only polish; there is no functional gap here — a
CloudFormation type string (e.g. `AWS::EC2::Instance`) is already handled identically
to a Terraform type string by the existing mechanism, only the field's usage examples
are asymmetric.

**Independent Test**: Read the data model's `source_types` field description and
confirm both a Terraform and a CloudFormation example string are present.

**Acceptance Scenarios**:

1. **Given** the data-model documentation, **When** the `source_types` field is
   described, **Then** it includes a concrete CloudFormation example
   (`AWS::EC2::Instance`) alongside the existing Terraform example (`aws_instance`).

---

### Edge Cases

- What happens when `MaxSourceTypes` is configured to zero or a negative value? The
  default limit applies, matching the existing `BatchCost` clamp-to-default behavior.
- What happens when a `TypeRegistry` default TTL is zero or negative? No `expires_at`
  is set (unchanged, pre-existing behavior) — this is how a plugin author opts out of
  the caching hint.
- What happens when a mapping-file entry omits `pulumi_token`? Loading fails with a
  descriptive error rather than registering an incomplete mapping.
- What happens when the request-size limit is hit but `source_types` is otherwise empty
  or the format is `UNSPECIFIED`? Those remain empty-response cases handled downstream,
  not validation errors — the limit only rejects genuinely oversized lists.
- What happens to the existing standalone fallback-only conformance test
  (unimplemented-plugin path) once suite-registered tests are added? It remains,
  unmodified, validating a different layer (the server's dispatch fallback) than the
  new suite-registered tests (the conformance-tier contract).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The SDK MUST reject a `ResolveResourceTypesRequest` whose `source_types`
  count exceeds a configurable maximum, returning an `InvalidArgument` error, before any
  of the three response paths (custom provider, `TypeRegistry`, empty fallback) run.
- **FR-002**: The SDK MUST provide a default maximum and a hard upper-bound maximum for
  `source_types` count, configurable per server instance, following the same
  default/hard-limit pattern as `BatchCost`.
- **FR-003**: The TypeScript client MUST validate `source_types` length locally before
  issuing a `resolveResourceTypes()` call, matching its existing `batchCost()`
  client-side validation.
- **FR-004**: The protocol MUST add an optional, advisory `expires_at` timestamp field
  to `ResolveResourceTypesResponse`, using the same semantics as the existing
  `expires_at` fields on `ActualCostResult`, `GetProjectedCostResponse`, and
  `EstimateCostResponse` (nil = no guidance, past = stale, future = valid-until).
- **FR-005**: The SDK MUST provide helper functions to read and set the new
  `expires_at` field, following the existing per-message helper-function pattern used
  for the other three RPCs.
- **FR-006**: The `TypeRegistry` MUST support an optional default expiration duration,
  applied automatically to every `Resolve()` response when configured, with no change
  in behavior when left unconfigured.
- **FR-007**: The SDK's conformance testing framework MUST include tests for
  `ResolveResourceTypes` covering both the "plugin does not implement type resolution"
  and "plugin resolves known types correctly" cases, registered the same way other
  optional-capability RPCs (e.g. `GetRecommendations`) are.
- **FR-008**: The SDK's mock plugin fixture MUST support configurable
  `ResolveResourceTypes` behavior for use in conformance tests and plugin-author
  experimentation.
- **FR-009**: The SDK's Prometheus metrics MUST track, per plugin, how many requested
  `source_types` were resolved versus left unresolved across `ResolveResourceTypes`
  calls, with test coverage for the new metrics.
- **FR-010**: The `TypeRegistry` MUST support batch registration of mappings that
  include property name overrides, not only the existing single-entry registration
  method.
- **FR-011**: The SDK documentation MUST describe how to register and when to use
  property name overrides, with a concrete example distinguishing mechanical from
  non-mechanical renames.
- **FR-012**: The `TypeRegistry` MUST support loading mappings from an external
  JSON-encoded source (a byte stream and, separately, a file path), covering one IaC
  source format per load call, without finfocus-spec itself shipping or maintaining any
  provider-specific mapping data.
- **FR-013**: Loading mappings from an external JSON source MUST fail with a
  descriptive error (not a silent no-op) for malformed JSON, an unrecognized source
  format, or an entry missing its Pulumi token.
- **FR-014**: The data model documentation for `source_types` MUST include a concrete
  example for both Terraform and CloudFormation source formats.
- **FR-015**: All changes MUST be additive and backward-compatible — no existing RPC,
  message field, or SDK function signature may change in a way that breaks existing
  callers.

### Key Entities

- **Source-types size limit**: A configurable ceiling (with a sensible default and a
  hard maximum) on how many resource-type strings a single `ResolveResourceTypes`
  request may contain, enforced identically regardless of which response path would
  ultimately serve the request.
- **Caching hint**: An advisory expiration timestamp on a type-resolution response,
  following the existing cross-RPC caching-hint convention, with an optional
  registry-level default so plugin authors do not need to set it per response.
- **Mapping file**: An external JSON-encoded document describing source-type-to-Pulumi
  token mappings (with optional property overrides) for one IaC source format, loadable
  into a `TypeRegistry` at plugin initialization time as an alternative to hardcoded
  registration calls.

## Assumptions

- The next available field number on `ResolveResourceTypesResponse` is 2 (the message
  currently defines only `mappings = 1`).
- No production consumer of `ResolveResourceTypes` exists yet (per spec 049's own
  dependency notes, the finfocus core integration is separate and still pending), so
  the new request-size limit carries no realistic backward-compatibility risk.
- Provider-specific mapping *data* continues to live outside finfocus-spec, per the
  separation-of-concerns decision already made in spec 049; this feature adds only a
  generic loading *mechanism* (a JSON deserializer), not any bundled mapping content.
- A sensible default/maximum for `source_types` count sits above the equivalent
  `BatchCost` limits, because a de-duplicated type list is inherently smaller than a
  per-resource list even for very large Terraform states.

## Dependencies

- **Upstream**: Spec `049-resolve-resource-types` (merged, PR #486) — this feature
  modifies the RPC, messages, and SDK helpers it introduced.
- **Precedent patterns reused**: `BatchCost` request-size validation (`batch.go`);
  `expires_at` caching-hint convention introduced in spec `045-caching-hint-expires-at`;
  `GetRecommendations` conformance-tier and metrics precedent (spec
  `014-recommendations-rpc`).
- **Downstream consumers**: None known yet — same as spec 049, the finfocus core
  integration that will call this RPC in production is tracked separately and is out of
  scope here.

## Scope Boundaries

### In Scope

- Go SDK: request-size validation, `expires_at` field support and `TypeRegistry`
  default-TTL option, conformance tests, mock plugin support, metrics, batch
  property-override registration, JSON mapping-file loader.
- TypeScript SDK: client-side request-size validation mirroring `batchCost()`.
- Proto: one additive field (`expires_at`) on `ResolveResourceTypesResponse`.
- Documentation: README updates for property overrides, caching hints, request limits,
  and the mapping-file loader; a data-model doc clarification for CloudFormation
  examples.

### Out of Scope

- Opening a tracking issue (or any other change) in the `finfocus` core repository for
  actually consuming this RPC during Terraform-state ingestion — tracked separately.
- Any actual Terraform/CloudFormation/AWS/Azure/GCP mapping *data* — this remains a
  plugin- and community-repo responsibility.
- A property-name-override transformer/consumer — `property_mappings` remains a
  storage-only mechanism; applying overrides during resource-shape translation is
  core's responsibility, unchanged from spec 049.
- Any CloudFormation-specific implementation beyond the documentation example in User
  Story 6 — no plugin implements CloudFormation resolution yet.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All existing tests continue to pass after the changes (zero regressions).
- **SC-002**: A `ResolveResourceTypesRequest` exceeding the configured size limit is
  rejected before any provider or registry code executes, in 100% of tested cases.
- **SC-003**: A well-formed request at or under the size limit behaves identically to
  its pre-change behavior in 100% of tested cases (no regression for valid traffic).
- **SC-004**: A `TypeRegistry` configured with a default TTL stamps `expires_at` on
  every `Resolve()` response within the expected time window, with zero
  `expires_at`-related regressions for registries that do not configure one.
- **SC-005**: The Basic conformance tier passes for both a plugin without type
  resolution support and the SDK's mock plugin with type resolution configured.
- **SC-006**: Resolved/unresolved metrics counters reflect the exact resolved and
  unresolved counts for 100% of tested `ResolveResourceTypes` call patterns, including
  the error case (no increment).
- **SC-007**: A plugin author can load an externally-maintained mapping file into a
  `TypeRegistry` and have it behave identically to the equivalent hardcoded
  registration calls, with zero additional code beyond the load call itself.
- **SC-008**: The feature requires only a minor version bump (additive, non-breaking
  changes throughout).
