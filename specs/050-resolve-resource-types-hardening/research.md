# Research: ResolveResourceTypes Hardening

**Feature**: 050-resolve-resource-types-hardening
**Date**: 2026-09-07
**Status**: Complete

## Research Tasks

### R1: Source-Types Request Limit Values

**Decision**: `DefaultMaxSourceTypes = 200`, hard ceiling `MaxSourceTypes = 2000`.

**Rationale**: `BatchCost` uses `DefaultMaxBatchSize = 100` / `MaxBatchSize = 1000`
(`sdk/go/pluginsdk/batch.go:34-37`) because each entry there is a full
`ResourceDescriptor` for one resource. `source_types` is fundamentally different: it is
a de-duplicated list of *type strings*, not resources. Even a Terraform state with
thousands of resources typically has well under 100 distinct resource types, so the
limit can sit comfortably above `BatchCost`'s while still bounding worst-case
allocation from a malformed or hostile request.

**Alternatives Considered**:

- Reusing `DefaultMaxBatchSize`/`MaxBatchSize` verbatim — rejected; conflates two
  different cardinalities (resource count vs. distinct type count) and would be overly
  restrictive for legitimate large-state requests.
- No hard ceiling, default only — rejected; `BatchCost` establishes the precedent of a
  server-configurable default plus an unconfigurable hard ceiling, and dropping the
  ceiling would remove the DoS protection this feature exists to add.

### R2: Enforcement Point for the Size Limit

**Decision**: Validate `len(req.GetSourceTypes())` at the very top of
`Server.ResolveResourceTypes`, before any of the three dispatch tiers (custom provider
interface, `TypeRegistry`, empty fallback) run.

**Rationale**: All three tiers are equally exposed to the same oversized-allocation
risk. Enforcing once at the handler entry point — mirroring where
`ValidateBatchCostRequest` is called at `sdk.go:822` before `BatchCost` processing
begins — means a plugin implementing `ResolveResourceTypesProvider` directly does not
need to re-implement the guard itself, and `TypeRegistry.Resolve()` does not need
size-awareness at all.

**Alternatives Considered**:

- Validating inside `TypeRegistry.Resolve()` only — rejected; leaves the custom-provider
  and empty-fallback tiers unprotected.
- Validating in both the handler and `TypeRegistry.Resolve()` — rejected as redundant;
  a single well-placed check is simpler to reason about and test.

### R3: `expires_at` Field Number and Semantics

**Decision**: Add `google.protobuf.Timestamp expires_at = 2;` to
`ResolveResourceTypesResponse` (which currently defines only `mappings = 1`), with
identical semantics to the existing `expires_at` fields introduced in spec
045-caching-hint-expires-at: nil = no caching guidance (always refetch), a past
timestamp = stale, a future timestamp = valid until then. Advisory only.

**Rationale**: Field number 2 is the next available number on this message. Reusing the
exact semantics of the three existing `expires_at` fields (`ActualCostResult` field 8,
`GetProjectedCostResponse` field 13, `EstimateCostResponse` field 5) keeps the
convention uniform across every RPC that has adopted it, so callers only need to learn
the semantics once.

**Alternatives Considered**:

- A plain `int64` Unix-seconds field or a `google.protobuf.Duration` (TTL) instead of an
  absolute `Timestamp` — rejected; would break consistency with the three existing
  caching-hint fields and force callers to handle two different conventions.

### R4: `TypeRegistry` Default-TTL Design

**Decision**: Add a `TypeRegistryOption` functional-option type,
`NewTypeRegistry(opts ...TypeRegistryOption)` (variadic, so the existing zero-arg call
sites keep compiling unchanged), and `WithDefaultTTL(ttl time.Duration)`. A
`defaultTTL time.Duration` field is added to the `TypeRegistry` struct; `Resolve()`
stamps `expires_at = now + defaultTTL` on its response only when `defaultTTL > 0`.

**Rationale**: Every existing `expires_at` setter in the codebase
(`WithActualCostResultExpiresAt`, `WithEstimateCostExpiresAt`, etc.) is a per-response,
hand-called functional option — appropriate for cost data that genuinely varies call to
call. Type mappings are the opposite: they are effectively constant for the lifetime of
a plugin process, so requiring every `Resolve()` caller (there is exactly one call site,
inside the SDK's own handler) to hand-set `expires_at` would be pure boilerplate.
Configuring the TTL once, at registry-construction time, matches how the data actually
behaves.

**Alternatives Considered**:

- Only the per-message `WithResolveResourceTypesExpiresAt` setter, no registry-level
  default — rejected; since `TypeRegistry.Resolve()` builds the response internally,
  there is no ergonomic way for a plugin author to apply a per-response option without
  bypassing the registry, defeating the "batch registration in under 10 lines" goal
  spec 049 already established.
- A package-level global default TTL — rejected; would leak configuration across
  independently-configured registries in the same process (relevant for tests).

### R5: Conformance Test Placement

**Decision**: Register `ResolveResourceTypes_EmptyPlugin` and `ResolveResourceTypes_Basic`
into `addBasicConformanceTests` in `sdk/go/testing/conformance_test.go`, immediately
after the existing `GetRecommendations_*` block, using the identical
`plugintesting.ConformanceTest{Name, Description, TestFunc}` registration shape.

**Rationale**: `GetRecommendations` is the only existing optional-capability RPC with
conformance-tier coverage, and all three of its tests live in the Basic tier (not split
across Standard/Advanced), so mirroring that placement keeps the precedent consistent.
Both new tests are written to pass whether or not a plugin implements
`ResolveResourceTypesProvider` — an empty response is a valid Basic-tier outcome, not a
failure — exactly like `GetRecommendations_EmptyPlugin`.

**Alternatives Considered**:

- Splitting `_EmptyPlugin` into Basic and a richer `_Basic` mapping-correctness
  assertion into Standard — rejected for consistency with how all three
  `GetRecommendations` tests were placed together in Basic; introducing a tier split
  here without a split for the precedent it mirrors would be an unexplained
  inconsistency.
- Leaving the existing standalone `resolve_resource_types_conformance_test.go`
  unregistered and only adding new tests elsewhere — accepted as the actual decision:
  that file validates the `Server`-level fallback path directly via `TestHarness`, a
  different (and still valuable) layer than suite-registered tier tests, so it is kept
  unmodified with a one-line cross-reference comment rather than folded in or removed.

### R6: Hit/Miss Metrics Design

**Decision**: Add `ResolveResourceTypesResolved` and `ResolveResourceTypesUnresolved`
(`*prometheus.CounterVec`, labeled `plugin_name`) to `PluginMetrics`, registered in
`NewPluginMetrics`, and populated via a new
`if method == "finfocus.v1.CostSource/ResolveResourceTypes" && err == nil { ... }`
branch inside `MetricsInterceptorWithRegistry`, mirroring the existing
`GetRecommendations` branch. The branch type-asserts both `req` and `resp` and derives
resolved (`len(resp.GetMappings())`) and unresolved
(`len(req.GetSourceTypes()) - resolved`) counts — the same arithmetic the handler
already performs for debug logging at `sdk.go:907-922`, recomputed here because
interceptors cannot reach handler-local variables.

**Rationale**: The generic per-RPC counter/histogram already covers call count and
latency for free (keyed by `info.FullMethod`); it captures nothing about *how well* the
plugin's mapping data covers what callers actually ask for. A resolved/unresolved pair
gives operators a direct signal for "this plugin's mapping table needs expansion."

**Alternatives Considered**:

- A single counter with a `result` label (`"resolved"`/`"unresolved"`) instead of two
  separate counters — considered viable, but two explicitly-named counters match the
  existing `RecommendationsTotal`/`RecommendationsPerResponse` pattern's use of distinct
  metric names per concept, and are simpler to graph independently in Prometheus/Grafana
  without a label-based `sum by`.
- Recomputing the hit/miss split inside the handler and exporting it via a package-level
  variable read by the interceptor — rejected; needlessly couples the handler and
  interceptor when the interceptor already has direct access to both `req` and `resp`.

**Test-coverage note**: The `GetRecommendations` metrics this mirrors
(`RecommendationsTotal`/`RecommendationsPerResponse`) have no dedicated test coverage in
`metrics_test.go` today. This feature does not repeat that gap — see FR-009.

### R7: Batch Property-Override Registration API

**Decision**: Add a `TypeMapping{PulumiToken string; PropertyMappings map[string]string}`
struct and `RegisterMappingsWithProperties(format pbc.SourceFormat, mappings map[string]TypeMapping)`
to `TypeRegistry`, always setting `Supported: true` (matching `RegisterMappings`'s
existing convention).

**Rationale**: The existing `RegisterMappings(format, map[string]string)` cannot express
property overrides because its value type is a bare `string` (the Pulumi token only). A
new value type is required to carry both the token and the optional overrides per entry
without changing `RegisterMappings`'s existing signature. Keeping `Supported` hardcoded
to `true` (as `RegisterMappings` already does) preserves the batch API's "common case"
simplicity; a caller needing `supported=false` on a per-entry basis still has the
existing singular `RegisterMappingWithProperties`.

**Alternatives Considered**:

- Adding a `supported bool` field to `TypeMapping` — rejected; the vast majority of
  batch registrations are for priceable types (the same reasoning `RegisterMappings`
  already used to justify hardcoding `true`), and adding the field would be unused
  boilerplate for nearly every caller.
- Overloading `RegisterMappings` to accept `map[string]any` — rejected; loses
  compile-time type safety and is a worse developer experience than a new, clearly-named
  method.

### R8: JSON Mapping-File Loader Schema

**Decision**: One `source_format` per file. Schema:
`{"source_format": "TERRAFORM", "mappings": {"<type>": {"pulumi_token": "...", "supported": true, "property_mappings": {...}}}}`.
New methods `TypeRegistry.LoadMappingsFromJSON(io.Reader) error` and
`LoadMappingsFromFile(path string) error` (the latter delegating to the former).
`source_format` is matched case-insensitively against the `SourceFormat` enum names
(`TERRAFORM`, `CLOUDFORMATION`); an unrecognized or missing value is a hard error, not a
silent no-op. Entries merge into any existing registry mappings for that format,
last-write-wins on key collision.

**Rationale**: A flat, single-format-per-file schema keeps the JSON structure trivial —
a plugin supporting both Terraform and CloudFormation simply loads two files, each
calling `LoadMappingsFromJSON` on the same registry. The field names (`pulumi_token`,
`supported`, `property_mappings`) mirror the `ResourceTypeMapping` proto message
directly, so the file format is conceptually a JSON serialization of the same entity
plugin authors already understand from `RegisterMappingWithProperties`. This ships zero
provider-specific *data* — only a deserializer — so it does not reopen spec 049's
constitution-based rejection of a hardcoded shared mapping table (Constitution II & IV);
mapping data still lives entirely in plugin- or community-maintained files outside this
repository.

**Alternatives Considered**:

- A multi-format-per-file schema (`{"formats": [{...}, {...}]}`) — considered, but
  rejected in favor of the simpler one-per-file schema; a plugin needing multiple
  formats loses nothing by calling `LoadMappingsFromJSON` twice, and the simpler schema
  is easier to hand-write, diff, and validate.
- YAML instead of JSON — rejected; the Go standard library has no YAML support, and
  adding a YAML dependency for this purpose is unwarranted given `encoding/json` is
  sufficient and dependency-free.

### R9: TypeScript SDK Scope for This Feature

**Decision**: The TypeScript SDK mirror is limited to the request-size limit
(`DEFAULT_MAX_SOURCE_TYPES`/`MAX_SOURCE_TYPES` in `utils/batch.ts`, and the matching
pre-flight check in `cost-source.ts`'s `resolveResourceTypes()`). The `expires_at` field
requires no hand-written TypeScript code — it reaches the generated `costsource_pb.ts`
bindings automatically via `make generate`/`buf generate`, exactly as it did for the
three existing `expires_at` fields per spec 045's own precedent (no bespoke TS helper
functions were built there either). The `TypeRegistry`, conformance/metrics, and JSON
loader items are Go-SDK-only (server-side and testing-framework concerns with no
TypeScript equivalent).

**Rationale**: Constitution XIII requires SDK synchronization specifically for RPC
methods and wire-visible behavior — the request-size limit is client-visible behavior
(a client-side rejection before the RPC call, matching `batchCost()`'s existing
pattern) and therefore in scope. `expires_at` is a passive field read via generated
bindings, not a method requiring a hand-written wrapper. `TypeRegistry` and the JSON
loader are server-side (plugin-author-facing) SDK concerns with no client-side
counterpart, matching how spec 049 itself scoped `TypeRegistry` as Go-only.

**Alternatives Considered**:

- Hand-writing a TypeScript `isResolveResourceTypesExpired()` helper for symmetry with
  the Go SDK — rejected; no such helper exists for any of the three prior `expires_at`
  fields in the TypeScript SDK, so adding one here would be an inconsistent one-off
  rather than filling an established gap.

### R10: CloudFormation Documentation Fix Location

**Decision**: Add the CloudFormation example (`AWS::EC2::Instance`) to
`specs/049-resolve-resource-types/data-model.md`'s `source_types` field row (entity 3,
`ResolveResourceTypesRequest`), where the Terraform example already lives. No change to
`spec.md` in either 049 or this feature.

**Rationale**: The asymmetry is confined to that one table cell — 049's `spec.md`
Assumptions and Out-of-Scope sections already treat Terraform and CloudFormation
identically and correctly. Editing the 049 data-model.md directly (rather than
duplicating the entity description in this feature's own data-model.md) avoids two
documents describing the same field with different examples.

**Alternatives Considered**:

- Duplicating the field description into this feature's own `data-model.md` instead of
  editing 049's — rejected; would create two sources of truth for the same field's
  documented examples.
