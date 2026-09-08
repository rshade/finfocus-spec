# Data Model: ResolveResourceTypes Hardening

**Feature**: 050-resolve-resource-types-hardening
**Date**: 2026-09-07

This feature extends entities introduced in spec 049 rather than replacing them. Only
new/changed fields and new entities are detailed below; unchanged fields from 049's
`data-model.md` are not repeated here.

## Entities

### 1. ResolveResourceTypesResponse (Proto Message — extended)

**Location**: `proto/finfocus/v1/costsource.proto`
**Type**: Protobuf message (existing, gains one field)

| Field | Type | Number | Required | Description |
|-------|------|--------|----------|-------------|
| `mappings` | map\<string, ResourceTypeMapping\> | 1 | Yes | Unchanged from spec 049. |
| `expires_at` | google.protobuf.Timestamp | 2 | No | **New.** Advisory caching hint. nil = no guidance (always refetch); past = stale; future = valid until then. Type mappings are near-static; plugins are encouraged to set a long TTL (hours to days) rather than leaving it nil. |

**Validation Rules**:

- `expires_at` is optional; absence means "no caching guidance," identical in meaning
  to the existing `expires_at` fields on `ActualCostResult`, `GetProjectedCostResponse`,
  and `EstimateCostResponse`.
- No new validation on `mappings` (unchanged from spec 049).

**Relationships**:

- Set automatically by `TypeRegistry.Resolve()` when the registry has a configured
  default TTL (see entity 3).
- Readable/writable via the new `expires_at.go` helper triple (entity 2).

---

### 2. `expires_at` SDK Helpers (Go SDK Functions — new)

**Location**: `sdk/go/pluginsdk/expires_at.go`
**Type**: Go functions (extends the existing per-message helper pattern)

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsResolveResourceTypesExpired` | `func(resp *pbc.ResolveResourceTypesResponse, now time.Time) bool` | Returns true if `expires_at` is set and before `now`. False if `resp` or `expires_at` is nil. |
| `ResolveResourceTypesExpiresAt` | `func(resp *pbc.ResolveResourceTypesResponse) (time.Time, bool)` | Returns the expiration time and `true`, or zero value and `false` if unset. |
| `WithResolveResourceTypesExpiresAt` | `func(expiresAt time.Time) ResolveResourceTypesResponseOption` | Functional option setting `expires_at`. A zero `time.Time` clears the field back to nil. |

**New supporting type**: `ResolveResourceTypesResponseOption func(*pbc.ResolveResourceTypesResponse)`,
plus `NewResolveResourceTypesResponse(opts ...ResolveResourceTypesResponseOption) *pbc.ResolveResourceTypesResponse`
(added to `type_registry.go` since no constructor family exists yet for this response
type).

---

### 3. TypeRegistry (Go SDK Helper — extended)

**Location**: `sdk/go/pluginsdk/type_registry.go`
**Type**: Go struct (existing, gains a field, an option type, and two methods)

| Field | Type | Visibility | Description |
|-------|------|------------|-------------|
| `mappings` | `map[pbc.SourceFormat]map[string]*pbc.ResourceTypeMapping` | Private | Unchanged from spec 049. |
| `defaultTTL` | `time.Duration` | Private | **New.** When > 0, `Resolve()` stamps `expires_at = now + defaultTTL` on its response. Zero/unset preserves spec 049's original behavior (no `expires_at`). |

**New type**: `TypeRegistryOption func(*TypeRegistry)`

**New/changed methods**:

| Method | Signature | Description |
|--------|-----------|--------------|
| `NewTypeRegistry` | `func NewTypeRegistry(opts ...TypeRegistryOption) *TypeRegistry` | **Changed from spec 049** (was zero-arg) to variadic — existing zero-arg call sites remain valid (backward compatible). |
| `WithDefaultTTL` | `func WithDefaultTTL(ttl time.Duration) TypeRegistryOption` | **New.** Configures the default TTL applied to every `Resolve()` response. |
| `RegisterMappingsWithProperties` | `func (r *TypeRegistry) RegisterMappingsWithProperties(format pbc.SourceFormat, mappings map[string]TypeMapping)` | **New.** Batch-registers mappings that include property overrides, always `Supported: true` (mirrors `RegisterMappings`'s existing convention). |
| `LoadMappingsFromJSON` | `func (r *TypeRegistry) LoadMappingsFromJSON(reader io.Reader) error` | **New.** Deserializes a JSON mapping file (one source format) and registers its entries, merging with any existing mappings for that format (last-write-wins). |
| `LoadMappingsFromFile` | `func (r *TypeRegistry) LoadMappingsFromFile(path string) error` | **New.** Opens `path` and delegates to `LoadMappingsFromJSON`. |
| `Resolve` | *(signature unchanged)* | **Behavior extended**: stamps `expires_at` on the returned response when `defaultTTL > 0`, otherwise unchanged from spec 049. |

**Validation Rules** (new, in addition to spec 049's existing rules):

- `LoadMappingsFromJSON`/`LoadMappingsFromFile` MUST fail with a descriptive error (not
  silently register nothing) for: malformed JSON, an unrecognized/missing
  `source_format` string, or any entry missing `pulumi_token`.
- `RegisterMappingsWithProperties` follows the same "call during initialization only"
  contract as all other `Register*`/`Load*` methods (documented, not runtime-enforced).

---

### 4. TypeMapping (Go SDK Struct — new)

**Location**: `sdk/go/pluginsdk/type_registry.go`
**Type**: Go struct (batch-registration value type)

| Field | Type | Description |
|-------|------|-------------|
| `PulumiToken` | `string` | The target Pulumi type token. |
| `PropertyMappings` | `map[string]string` | Optional property-name overrides (source property → Pulumi property); nil/empty is valid. |

**Relationships**: Used as the map value type in `RegisterMappingsWithProperties(format, map[string]TypeMapping)`.

---

### 5. Mapping File (JSON Document — new, external to the wire protocol)

**Location**: Consumed by `sdk/go/pluginsdk/type_registry_loader.go`; the file itself
lives in a plugin repository or community-maintained data repository, **not** in
finfocus-spec.
**Type**: JSON document (Go-package-internal convention, not a protobuf contract)

```json
{
  "source_format": "TERRAFORM",
  "mappings": {
    "aws_instance": {
      "pulumi_token": "aws:ec2/instance:Instance",
      "supported": true,
      "property_mappings": {
        "instance_type": "instanceType"
      }
    }
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source_format` | string | Yes | Case-insensitive match against `SourceFormat` enum names (`TERRAFORM`, `CLOUDFORMATION`). One format per file. |
| `mappings` | object | Yes | Keyed by source type string; value mirrors `ResourceTypeMapping`. |
| `mappings.*.pulumi_token` | string | Yes | Non-empty Pulumi type token. |
| `mappings.*.supported` | bool | No | Defaults to JSON `false` if omitted (matches proto3 default). |
| `mappings.*.property_mappings` | object | No | Optional; omitted or empty is valid. |

**Validation Rules**:

- Unrecognized `source_format` → hard error, no entries registered.
- Any entry missing `pulumi_token` → hard error, no entries from that file registered.
- finfocus-spec ships zero example/default mapping-file *content* of its own — this
  schema is a mechanism definition only (Constitution II & IV; see research.md R8).

---

### 6. Source-Types Request Limit (Go SDK Constants + Validator — new)

**Location**: `sdk/go/pluginsdk/batch.go`
**Type**: Go constants and a validation function (mirrors the existing `BatchCost` limit pattern)

| Name | Value | Description |
|------|-------|-------------|
| `DefaultMaxSourceTypes` | `200` | Default cap applied when a server is not explicitly configured with a limit. |
| `MaxSourceTypes` | `2000` | Hard ceiling; a configured value above this is clamped down. |

| Function | Signature | Description |
|----------|-----------|--------------|
| `ValidateResolveResourceTypesRequest` | `func(req *pbc.ResolveResourceTypesRequest, maxSourceTypes int32) error` | Internally applies `if maxSourceTypes <= 0 { maxSourceTypes = DefaultMaxSourceTypes }` (mirrors `ValidateBatchCostRequest`'s identical defensive fallback), then returns `status.Error(codes.InvalidArgument, ...)` when `len(req.GetSourceTypes())` exceeds the resolved limit. Returns nil for a nil request, empty `source_types`, or `SOURCE_FORMAT_UNSPECIFIED` (those remain empty-response cases, not validation errors). |
| `resolveSourceTypesLimit` | `func(configured int) int32` | Clamps a configured value into `[1, MaxSourceTypes]`, defaulting to `DefaultMaxSourceTypes` when `configured <= 0`. |

**Relationships**:

- `Server` (in `sdk.go`) gains a `maxSourceTypes int32` field, set from
  `ServeConfig.MaxSourceTypes int` via `resolveSourceTypesLimit` during `Serve()`.
- `Server.ResolveResourceTypes` calls `ValidateResolveResourceTypesRequest` before
  dispatching to any of the three response tiers.

---

### 7. ResolveResourceTypesConfig (Go SDK Mock Fixture — new)

**Location**: `sdk/go/testing/mock_plugin.go`
**Type**: Go struct (test fixture configuration, mirrors `RecommendationsConfig`)

| Field | Type | Description |
|-------|------|-------------|
| `Mappings` | `map[string]*pbc.ResourceTypeMapping` | Seeded mappings the mock returns. |
| `ShouldError` | `bool` | When true, `ResolveResourceTypes` returns an error instead of a response. |
| `ErrorMessage` | `string` | Error text used when `ShouldError` is true. |

`MockPlugin` implements `ResolveResourceTypesProvider` using this configuration, set via
`SetResolveResourceTypesConfig(config ResolveResourceTypesConfig)`. `NewMockPlugin()`
seeds one default entry (`aws_instance` → `aws:ec2/instance:Instance`) so conformance
tests have non-empty data without per-test setup.

---

### 8. Resolved/Unresolved Metrics (Prometheus Counters — new)

**Location**: `sdk/go/pluginsdk/metrics.go`
**Type**: `*prometheus.CounterVec` fields on `PluginMetrics`

| Field | Labels | Description |
|-------|--------|--------------|
| `ResolveResourceTypesResolved` | `plugin_name` | Incremented by the count of `source_types` entries present in the response `mappings` for each successful call. |
| `ResolveResourceTypesUnresolved` | `plugin_name` | Incremented by `len(source_types) - resolved` for each successful call. Not incremented on error responses. |

**Relationships**: Populated in `MetricsInterceptorWithRegistry`, alongside (not
replacing) the existing generic `RequestsTotal`/`RequestDuration` metrics that already
cover every RPC including this one.

## Entity Relationship Diagram

```text
CostSourceService
  └── ResolveResourceTypes(ResolveResourceTypesRequest) -> ResolveResourceTypesResponse
        │
        ├── [NEW] size validated: len(source_types) <= maxSourceTypes, else InvalidArgument
        │         (checked before any of the 3 dispatch tiers below)
        │
        └── ResolveResourceTypesResponse
              ├── mappings: map<string, ResourceTypeMapping>   (unchanged)
              └── expires_at: google.protobuf.Timestamp        [NEW, field 2]

TypeRegistry (extended)
  ├── mappings: map[SourceFormat]map[string]*ResourceTypeMapping   (unchanged)
  ├── defaultTTL: time.Duration                                    [NEW]
  ├── RegisterMapping / RegisterMappingWithProperties / RegisterMappings   (unchanged)
  ├── RegisterMappingsWithProperties(format, map[string]TypeMapping)      [NEW]
  ├── LoadMappingsFromJSON(io.Reader) / LoadMappingsFromFile(path)        [NEW]
  └── Resolve() -> stamps expires_at when defaultTTL > 0                 [EXTENDED]

Mapping File (JSON, external — plugin/community repos only)
  {source_format, mappings: {type: {pulumi_token, supported, property_mappings}}}
        │
        └── LoadMappingsFromJSON/File ──> TypeRegistry entries (identical to RegisterMapping*)

Metrics (per plugin_name)
  ├── RequestsTotal / RequestDuration          (generic, unchanged, covers this RPC already)
  └── ResolveResourceTypesResolved/Unresolved  [NEW, populated from req/resp diff in interceptor]

Conformance (Basic tier)
  ├── ResolveResourceTypes_EmptyPlugin  [NEW] — passes whether or not interface is implemented
  └── ResolveResourceTypes_Basic        [NEW] — validates well-formed mappings via MockPlugin
```
