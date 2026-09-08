# Data Model: Terraform Type Resolution RPC

**Feature**: 049-resolve-resource-types
**Date**: 2026-04-01

## Entities

### 1. SourceFormat (Proto Enum)

**Location**: `proto/finfocus/v1/enums.proto`
**Type**: Protobuf enum

| Value | Number | Description |
|-------|--------|-------------|
| `SOURCE_FORMAT_UNSPECIFIED` | 0 | Protobuf default sentinel. Plugins return empty mappings for this value. |
| `SOURCE_FORMAT_TERRAFORM` | 1 | Terraform / OpenTofu resource types (e.g., `aws_instance`). |
| `SOURCE_FORMAT_CLOUDFORMATION` | 2 | AWS CloudFormation logical resource types. Reserved for future use. |

**Validation Rules**:

- UNSPECIFIED is valid on the wire but semantically means "no format specified."
- Plugins MUST return empty mappings when receiving UNSPECIFIED (no error).

**Relationships**:

- Referenced by `ResolveResourceTypesRequest.source_format` (required field).
- Used as outer key in TypeRegistry's internal map.

---

### 2. ResourceTypeMapping (Proto Message)

**Location**: `proto/finfocus/v1/costsource.proto`
**Type**: Protobuf message

| Field | Type | Number | Required | Description |
|-------|------|--------|----------|-------------|
| `pulumi_token` | string | 1 | Yes | Pulumi type token (e.g., `aws:ec2/instance:Instance`). |
| `supported` | bool | 2 | Yes | Whether the plugin can price this resource type. `true` = priceable, `false` = known but not priceable. |
| `property_mappings` | map\<string, string\> | 3 | No | Optional property name overrides. Key = source property name, value = Pulumi property name. Empty in initial release. |

**Validation Rules**:

- `pulumi_token` MUST be non-empty when present in the response.
- `supported` defaults to `false` (proto3 default). Plugins SHOULD set it explicitly.
- `property_mappings` is empty by default; no validation on content in initial release.

**State Transitions**: None (stateless message).

**Relationships**:

- Contained in `ResolveResourceTypesResponse.mappings` map (value type).
- Created by `TypeRegistry.Resolve()` method.

---

### 3. ResolveResourceTypesRequest (Proto Message)

**Location**: `proto/finfocus/v1/costsource.proto`
**Type**: Protobuf message (RPC request)

| Field | Type | Number | Required | Description |
|-------|------|--------|----------|-------------|
| `source_format` | SourceFormat | 1 | Yes | The IaC tool that produced the type strings. |
| `source_types` | repeated string | 2 | Yes | List of source-format resource type strings to resolve. Example for Terraform: `aws_instance`. Example for CloudFormation: `AWS::EC2::Instance`. |

**Validation Rules**:

- `source_format` MUST NOT be UNSPECIFIED for meaningful results (but not an error).
- `source_types` MAY be empty (returns empty response, no error).
- Duplicate entries in `source_types` are deduplicated in the response (map semantics).

**Relationships**:

- Input to `CostSourceService.ResolveResourceTypes` RPC.
- Input to `TypeRegistry.Resolve()` method.

---

### 4. ResolveResourceTypesResponse (Proto Message)

**Location**: `proto/finfocus/v1/costsource.proto`
**Type**: Protobuf message (RPC response)

| Field | Type | Number | Required | Description |
|-------|------|--------|----------|-------------|
| `mappings` | map\<string, ResourceTypeMapping\> | 1 | Yes | Map from source type string to its resolved mapping. Only contains types the plugin can map. |

**Validation Rules**:

- Map keys are source type strings from the request.
- Map MUST only contain entries for types the plugin can resolve (FR-005).
- Unknown/unmappable types are **omitted** (not present with `supported=false`).
- Empty map is valid (plugin knows none of the requested types).

**Relationships**:

- Output of `CostSourceService.ResolveResourceTypes` RPC.
- Output of `TypeRegistry.Resolve()` method.

---

### 5. PluginCapability Enum Extension

**Location**: `proto/finfocus/v1/enums.proto`
**Type**: Protobuf enum value (addition to existing enum)

| Value | Number | Description |
|-------|--------|-------------|
| `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES` | 13 | Plugin implements ResolveResourceTypes RPC. |

**Relationships**:

- Reported by `GetPluginInfoResponse.capabilities`.
- Auto-detected when plugin implements `ResolveResourceTypesProvider` interface.
- Mapped to legacy metadata `"supports_resolve_resource_types": "true"`.

---

### 6. TypeRegistry (Go SDK Helper)

**Location**: `sdk/go/pluginsdk/type_registry.go`
**Type**: Go struct (SDK helper, not proto)

| Field | Type | Visibility | Description |
|-------|------|------------|-------------|
| `mappings` | `map[pbc.SourceFormat]map[string]*pbc.ResourceTypeMapping` | Private | Two-level map: format -> source type -> mapping. |

**Methods**:

| Method | Signature | Description |
|--------|-----------|-------------|
| `NewTypeRegistry` | `func NewTypeRegistry() *TypeRegistry` | Constructor. |
| `RegisterMapping` | `func (r *TypeRegistry) RegisterMapping(format pbc.SourceFormat, sourceType string, pulumiToken string, supported bool)` | Register a single mapping. |
| `RegisterMappings` | `func (r *TypeRegistry) RegisterMappings(format pbc.SourceFormat, mappings map[string]string)` | Batch register (source type -> pulumi token) with `supported=true`. |
| `Resolve` | `func (r *TypeRegistry) Resolve(req *pbc.ResolveResourceTypesRequest) *pbc.ResolveResourceTypesResponse` | Resolve a batch of source types. Returns only known mappings. |
| `Len` | `func (r *TypeRegistry) Len() int` | Total number of registered mappings across all formats. |

**Validation Rules**:

- `RegisterMapping` / `RegisterMappings` MUST only be called during initialization (before
  gRPC server starts). Calling after server start is a programming error (no runtime guard;
  documented contract).
- `Resolve` is thread-safe for concurrent reads.
- Empty `source_types` in request returns empty response.
- `SOURCE_FORMAT_UNSPECIFIED` returns empty response.

**Relationships**:

- Referenced by `ServeConfig.TypeRegistry` (optional field).
- Used by the server handler when plugin does not implement `ResolveResourceTypesProvider`.
- Infers `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES` when non-nil in `ServeConfig`.

---

### 7. ResolveResourceTypesProvider (Go SDK Interface)

**Location**: `sdk/go/pluginsdk/sdk.go`
**Type**: Go interface (optional provider)

```go
type ResolveResourceTypesProvider interface {
    ResolveResourceTypes(ctx context.Context, req *pbc.ResolveResourceTypesRequest) (
        *pbc.ResolveResourceTypesResponse, error)
}
```

**Relationships**:

- Detected by `inferCapabilities()` for auto-discovery.
- Takes precedence over `TypeRegistry` when both are configured.
- Follows the pattern of `RecommendationsProvider`, `BudgetsProvider`, `DismissProvider`.

## Entity Relationship Diagram

```text
CostSourceService
  └── ResolveResourceTypes(ResolveResourceTypesRequest) -> ResolveResourceTypesResponse
        │
        ├── ResolveResourceTypesRequest
        │     ├── source_format: SourceFormat (enum)
        │     └── source_types: []string
        │
        └── ResolveResourceTypesResponse
              └── mappings: map<string, ResourceTypeMapping>
                    └── ResourceTypeMapping
                          ├── pulumi_token: string
                          ├── supported: bool
                          └── property_mappings: map<string, string>

Server Handler Resolution Order:
  1. Plugin implements ResolveResourceTypesProvider? -> Delegate to plugin
  2. ServeConfig.TypeRegistry != nil? -> Delegate to TypeRegistry.Resolve()
  3. Neither? -> Return empty ResolveResourceTypesResponse{}

Capability Auto-Discovery:
  inferCapabilities(plugin)
    ├── plugin.(ResolveResourceTypesProvider)? -> PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES
    └── ServeConfig.TypeRegistry != nil? -> PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES
```
