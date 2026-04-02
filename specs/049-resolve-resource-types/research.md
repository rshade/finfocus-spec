# Research: Terraform Type Resolution RPC

**Feature**: 049-resolve-resource-types
**Date**: 2026-04-01
**Status**: Complete

## Research Tasks

### R1: Next Available PluginCapability Enum Value

**Decision**: Use value `13` for `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES`.

**Rationale**: The current highest value in `proto/finfocus/v1/enums.proto:187` is
`PLUGIN_CAPABILITY_BATCH_COST = 12`. Value 13 is the next sequential integer and follows
the established pattern of sequential assignment.

**Alternatives Considered**:

- Skipping to a higher value (e.g., 20) for "headroom" -- rejected because protobuf enum
  values are not positional; gaps provide no benefit and break the visual pattern.

### R2: Behavior When Plugin Does Not Implement the Interface

**Decision**: Return an **empty response** (not an Unimplemented error) when a plugin does
not implement `ResolveResourceTypesProvider`.

**Rationale**: This matches the `GetRecommendations` pattern in `sdk/go/pluginsdk/sdk.go:652-666`,
where the server returns an empty list when the plugin doesn't implement `RecommendationsProvider`.
The spec explicitly requires this (FR-006) to enable graceful fallback in the core -- the core
interprets an empty response as "fall back to heuristic type conversion."

**Alternatives Considered**:

- `codes.Unimplemented` error (the `GetBudgets` pattern at `sdk.go:720-724`) -- rejected because
  FR-006 explicitly requires empty response for graceful degradation. The core must not need error
  handling to decide on fallback; an empty mapping set IS the fallback signal.
- `codes.NotFound` error -- rejected; conflates "plugin doesn't support this RPC" with "no mappings
  found for these types."

### R3: SourceFormat Enum Placement

**Decision**: Define `SourceFormat` enum in `proto/finfocus/v1/enums.proto` alongside
`PluginCapability`, `UsageProfile`, `RecommendationReason`, etc.

**Rationale**: All non-message-specific enums in the project are defined in `enums.proto`.
The SourceFormat enum is a cross-cutting concern (used by the RPC request message and
potentially by future features) and fits this pattern.

**Alternatives Considered**:

- Defining it inline in `costsource.proto` near the message that uses it -- rejected because
  it would break the established convention of centralizing enums in `enums.proto`.

### R4: TypeRegistry Thread Safety Pattern

**Decision**: Use a read-only `map[SourceFormat]map[string]ResourceTypeMapping` populated at
initialization time. No mutex needed.

**Rationale**: The spec requires thread-safety for concurrent reads after initialization
(FR-012). Go maps are safe for concurrent reads when no writes occur. The registry is populated
during plugin init (before the gRPC server starts) and never modified afterward. This is the
same pattern used by the `legacyCapabilityNames` package-level map in
`sdk/go/pluginsdk/capability_compat.go:24`.

**Alternatives Considered**:

- `sync.RWMutex` around a mutable map -- rejected because it adds lock contention for a
  read-only data structure and contradicts the zero-allocation performance goal.
- `sync.Map` -- rejected because it's optimized for key-disjoint concurrent writes, which
  is not our access pattern. Standard maps are faster for read-only workloads.

### R5: TypeRegistry Data Structure for Zero-Allocation Lookup

**Decision**: Two-level map: `map[pbc.SourceFormat]map[string]*pbc.ResourceTypeMapping`.
The outer key is the SourceFormat enum, the inner key is the source type string.

**Rationale**: O(1) lookup per type string. The outer map partitions by format (preventing
cross-format leakage per User Story 2, Acceptance Scenario 2). Using proto enum as the
outer key avoids string conversion. Using `*pbc.ResourceTypeMapping` (pointer to proto message)
avoids copying structs on each lookup.

**Alternatives Considered**:

- Flat map with composite key `"TERRAFORM:aws_instance"` -- rejected because it requires
  string concatenation per lookup (allocation) and is less ergonomic for batch registration.
- Slice-based linear scan (registry package pattern) -- rejected because TypeRegistry may
  hold 1000+ entries, making O(n) scan too slow. Map O(1) is appropriate here.

### R6: Legacy Metadata Name for New Capability

**Decision**: Use `"supports_resolve_resource_types"` as the legacy metadata string.

**Rationale**: Follows the established naming convention in `legacyCapabilityNames` map
(`capability_compat.go:24-37`): `"supports_"` prefix + snake_case capability name.
Examples: `supports_dry_run`, `supports_batch_cost`, `supports_dismiss_recommendations`.

**Alternatives Considered**: None -- the naming convention is unambiguous.

### R7: Property Mappings Field Design

**Decision**: Include `map<string, string> property_mappings` on `ResourceTypeMapping` as
an optional field, empty by default.

**Rationale**: Per User Story 4, this field is intentionally unused in the initial
implementation but designed in to avoid a future proto change. The map key is the source
property name (e.g., `instance_type`), the value is the Pulumi property name (e.g.,
`instanceType`). Using `map<string, string>` is the simplest proto representation.

**Alternatives Considered**:

- Separate `PropertyMapping` message with metadata fields -- rejected as premature; the
  initial use case is purely name translation. A richer message can be added later without
  breaking the map field.
- `repeated` field of `PropertyMapping` messages -- rejected for the same reason; adds
  unnecessary message type proliferation for unused functionality.

### R8: TypeScript SDK Update Scope

**Decision**: Add `resolveResourceTypes()` async method to `CostSourceClient` class in
`sdk/typescript/packages/client/src/clients/cost-source.ts`.

**Rationale**: Constitution XIII requires all SDKs to be updated when new RPCs are added.
The existing pattern shows each RPC has a corresponding async method in the client class
(e.g., `dryRun()`, `batchCost()`). The TypeScript protobuf bindings will be auto-generated
by `buf generate`, and the client wrapper method is a thin delegate.

**Alternatives Considered**:

- Deferring TypeScript to a follow-up spec -- rejected because Constitution XIII explicitly
  requires synchronization in the same change.

### R9: Capability Constants Update

**Decision**: Update the following constants in `sdk/go/pluginsdk/plugin_info.go`:

- `optionalCapabilities`: 5 -> 6
- `maxCapabilities`: 9 -> 10
- `maxValidCapability`: BATCH_COST (12) -> RESOLVE_RESOURCE_TYPES (13)

**Rationale**: These constants document the capability breakdown and control pre-allocation
size. Adding one optional capability requires updating all three. The `MaxConfiguredCapabilities`
constant (64) does not change -- it's a DoS protection limit, not a count.

**Alternatives Considered**: None -- the constants are documentation-as-code and must stay
accurate.

### R10: TypeRegistry Integration with Server Handler

**Decision**: The `ServeConfig` struct gains an optional `TypeRegistry *TypeRegistry` field.
When set and the plugin does NOT implement `ResolveResourceTypesProvider`, the server
delegates to the registry's `Resolve()` method. When both are set, the interface takes
precedence (explicit implementation overrides declarative registry).

**Rationale**: This follows the `PluginInfo` / `PluginInfoProvider` dual-mode pattern:
static configuration via `ServeConfig` fields, with optional dynamic override via interface
implementation. Plugin developers get the simplest path (TypeRegistry) by default, with an
escape hatch for custom logic.

**Alternatives Considered**:

- TypeRegistry as a standalone middleware/interceptor -- rejected because it would require
  a different integration pattern than all other optional RPCs in the SDK.
- TypeRegistry auto-detected on the plugin struct (like capability interfaces) -- rejected
  because TypeRegistry is not an interface implementation; it's a configuration object.
  Detecting it via reflection would be fragile.
