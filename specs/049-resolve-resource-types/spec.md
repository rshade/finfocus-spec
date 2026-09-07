# Feature Specification: Terraform Type Resolution RPC

**Feature Branch**: `049-resolve-resource-types`
**Created**: 2026-04-01
**Status**: Draft
**Input**: User description: "Add ResolveResourceTypes RPC, SourceFormat enum, TypeRegistry helper, and capability inference for mapping Terraform/CloudFormation types to Pulumi tokens"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Core Resolves Terraform Types via Plugin (Priority: P1)

The finfocus core receives a Terraform state file containing resource types like `aws_instance`
and `aws_s3_bucket`. Before pricing these resources, the core needs to translate them into Pulumi
type tokens (e.g., `aws:ec2/instance:Instance`). The core sends a batch of Terraform type strings
to the appropriate plugin and receives back a mapping of resolved types. Types the plugin cannot
map are omitted from the response, signaling the core to fall back to raw Terraform types with
mechanical property name conversion (`snake_to_camel`).

**Why this priority**: This is the primary use case that unblocks Terraform state ingestion in the
core. Without this RPC, the core cannot ask plugins for authoritative type mappings and must rely
entirely on heuristic conversion, which fails for non-trivial resource types.

**Independent Test**: Can be fully tested by sending a ResolveResourceTypes request with known
Terraform types and verifying the response contains correct Pulumi tokens. Delivers the core
contract that enables Terraform state ingestion.

**Acceptance Scenarios**:

1. **Given** a plugin that knows AWS Terraform mappings, **When** the core sends
   `["aws_instance", "aws_s3_bucket"]` with source format TERRAFORM, **Then** the response
   contains mappings for both types with correct Pulumi tokens and `supported=true`.
2. **Given** a plugin that knows some but not all types, **When** the core sends
   `["aws_instance", "unknown_resource_xyz"]`, **Then** the response contains a mapping only for
   `aws_instance`; `unknown_resource_xyz` is omitted (not returned with `supported=false`).
3. **Given** a plugin that does not implement type resolution, **When** the core calls
   ResolveResourceTypes, **Then** the response is empty (not an error), and the core falls back to
   raw Terraform types.

---

### User Story 2 - Plugin Developer Registers Type Mappings Declaratively (Priority: P2)

A plugin developer building a new cost source plugin for AWS needs to register hundreds of
Terraform-to-Pulumi type mappings. Instead of implementing raw RPC handling logic, they use the
SDK's TypeRegistry helper to declare all mappings at initialization time. The SDK automatically
handles incoming ResolveResourceTypes requests using the registry data.

**Why this priority**: Reduces plugin development effort from implementing a full RPC handler to
a single declarative registration call. This ergonomic improvement directly affects adoption
of the new capability.

**Independent Test**: Can be tested by creating a TypeRegistry, registering batch mappings, and
verifying that `Resolve()` returns correct results for known types and omits unknown types.

**Acceptance Scenarios**:

1. **Given** a TypeRegistry with 3 Terraform mappings registered, **When** `Resolve` is called
   with those 3 source types plus 1 unknown type, **Then** the response contains exactly 3
   mappings (the unknown type is omitted).
2. **Given** a TypeRegistry with Terraform mappings only, **When** `Resolve` is called with
   source format CLOUDFORMATION, **Then** the response is empty (no cross-format leakage).
3. **Given** a plugin using TypeRegistry (not implementing the provider interface directly),
   **When** the server receives a ResolveResourceTypes request, **Then** it delegates to the
   registry and returns correct mappings.

---

### User Story 3 - Capability Auto-Detection for Type Resolution (Priority: P3)

A plugin developer implements the `ResolveResourceTypesProvider` interface on their plugin struct.
When the host queries `GetPluginInfo`, the `RESOLVE_RESOURCE_TYPES` capability is automatically
reported without any manual configuration. The core uses this capability to decide whether to
call `ResolveResourceTypes` or fall back to heuristic type conversion.

**Why this priority**: Follows the established auto-discovery pattern used by DryRun,
Recommendations, Budgets, and other optional capabilities. Consistency reduces cognitive load for
plugin developers who are already familiar with the pattern.

**Independent Test**: Can be tested by creating a plugin that implements the interface and
verifying `GetPluginInfo` includes the new capability enum value.

**Acceptance Scenarios**:

1. **Given** a plugin that implements `ResolveResourceTypesProvider`, **When** `GetPluginInfo` is
   called, **Then** the capabilities list includes `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES`.
2. **Given** a plugin that does NOT implement `ResolveResourceTypesProvider`, **When**
   `GetPluginInfo` is called, **Then** the capabilities list does NOT include the new capability.
3. **Given** a plugin using TypeRegistry (without direct interface), **When** the server is
   constructed with the registry, **Then** the capability is still correctly reported.

---

### User Story 4 - Future-Proof Property Mapping Overrides (Priority: P4)

Some Terraform-to-Pulumi property name translations are not purely mechanical `snake_to_camel`
conversions. The type mapping response includes an optional property mappings field that plugins
can populate in the future to override the default conversion for specific properties. This field
is intentionally unused in the initial implementation but designed in to avoid a future proto
change.

**Why this priority**: Low priority because no plugin populates this field initially. Its value
is in avoiding a future breaking proto change when edge cases inevitably arise (e.g., a Terraform
property name that diverges from the Pulumi equivalent non-mechanically).

**Independent Test**: Can be tested by verifying the field exists in the proto definition and that
the TypeRegistry/response correctly serializes and deserializes it when populated.

**Acceptance Scenarios**:

1. **Given** a ResourceTypeMapping with property_mappings populated, **When** serialized and
   deserialized, **Then** the property mappings are preserved correctly.
2. **Given** the initial SDK implementation, **When** TypeRegistry creates mappings, **Then**
   property_mappings is empty by default.

---

### Edge Cases

- What happens when a plugin receives an empty `source_types` list? The response should be empty
  with no error.
- What happens when a plugin receives `SOURCE_FORMAT_UNSPECIFIED`? The plugin should return empty
  mappings (no match for unspecified format).
- What happens when multiple plugins can resolve the same Terraform type? The core is responsible
  for choosing which plugin to query (out of scope for the spec; the spec defines only the
  plugin-side contract).
- What happens when the TypeRegistry is used concurrently? It must be thread-safe for concurrent
  reads after initial registration at startup.
- What happens when a plugin knows a type mapping but cannot price that resource type? It returns
  the mapping with `supported=false`, allowing the core to use the Pulumi token for display
  purposes even without pricing.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The protocol MUST define a `ResolveResourceTypes` RPC on the `CostSourceService`
  that accepts a list of source-format type strings and returns their Pulumi token mappings.
- **FR-002**: The protocol MUST define a `SourceFormat` enum to identify the IaC tool that
  produced the resource type strings (UNSPECIFIED, TERRAFORM, CLOUDFORMATION).
- **FR-003**: The protocol MUST define a `ResourceTypeMapping` message containing the Pulumi
  token, a supported flag, and optional property name overrides.
- **FR-004**: The protocol MUST add `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES` (value 13) to the
  `PluginCapability` enum.
- **FR-005**: Types the plugin cannot map MUST be omitted from the response (not returned with
  `supported=false` or as an error).
- **FR-006**: Plugins that do not implement type resolution MUST return an empty response (not an
  Unimplemented error), enabling graceful fallback in the core.
- **FR-007**: The SDK MUST provide a `ResolveResourceTypesProvider` optional interface following
  the existing pattern (RecommendationsProvider, BudgetsProvider, DismissProvider).
- **FR-008**: The SDK MUST auto-detect the `RESOLVE_RESOURCE_TYPES` capability when a plugin
  implements the `ResolveResourceTypesProvider` interface.
- **FR-009**: The SDK MUST provide a `TypeRegistry` helper that allows declarative registration
  of source-format-to-Pulumi-token mappings.
- **FR-010**: The `TypeRegistry` MUST support batch registration of mappings for a given source
  format.
- **FR-011**: The SDK server MUST support automatic RPC handling via TypeRegistry when a plugin
  does not implement the provider interface directly.
- **FR-012**: The `TypeRegistry` MUST be thread-safe for concurrent reads after initial
  registration.
- **FR-013**: All protocol changes MUST be additive and backward-compatible (no breaking changes
  to existing RPCs or messages).

### Key Entities

- **SourceFormat**: Identifies the IaC tool that produced resource type strings. Currently
  supports Terraform/OpenTofu and CloudFormation (future). Uses UNSPECIFIED as the zero value.
- **ResourceTypeMapping**: Describes how a single source-format type maps to a Pulumi type token.
  Contains the Pulumi token string, a boolean indicating pricing support, and an optional map of
  property name overrides for non-mechanical conversions.
- **TypeRegistry**: SDK-side helper that holds source-format to Pulumi token mappings. Populated
  by plugins at initialization time. Provides a `Resolve` method that implements the RPC logic.
  Contains no provider-specific data itself.

## Assumptions

- The next available `PluginCapability` enum value is 13 (verified: current highest is
  `BATCH_COST = 12`).
- CloudFormation format is included in the enum for future extensibility but no plugin will
  implement it initially.
- The `property_mappings` field on `ResourceTypeMapping` is reserved for future use; no plugin
  populates it in the initial release.
- Provider-specific mapping data (e.g., AWS Terraform types to Pulumi tokens) lives in plugin
  repositories, not in finfocus-spec. The spec provides only the mechanism (TypeRegistry), not
  the data.
- The core is responsible for selecting which plugin to query for a given set of resource types;
  the spec defines only the plugin-side contract.

## Dependencies

- **Upstream**: Approved design in `docs/superpowers/specs/2026-03-26-terraform-state-ingestion-design.md` (Section 1) from the finfocus core repository.
- **Downstream consumers**: finfocus core (`resolveResourceTypes()` in `internal/cli/common_execution.go`), finfocus-plugin-aws-public (first plugin to implement mappings).
- **Cross-repo execution order**: (1) finfocus-spec (this feature), (2) finfocus core (parallel, uses capability detection), (3) finfocus-plugin-aws-public (requires spec release).

## Scope Boundaries

### In Scope

- Protocol definition: new RPC, messages, and enum values in finfocus-spec proto files
- Go SDK: optional interface, server handler, capability inference, TypeRegistry helper
- TypeScript SDK: CostSourceClient method and client test for ResolveResourceTypes
- Unit and integration tests for all new SDK code
- Generated proto code updates

### Out of Scope

- Provider-specific mapping data (lives in plugin repos)
- Core-side integration (separate finfocus core feature)
- CloudFormation mapping implementation (enum defined for future use only)
- Performance optimization for very large mapping tables (premature until real-world data)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All existing tests continue to pass after the changes (zero regressions).
- **SC-002**: A ResolveResourceTypes request with N known types returns exactly N mappings with
  correct Pulumi tokens in the response.
- **SC-003**: A ResolveResourceTypes request to a plugin without the capability returns an empty
  response (not an error) within the standard RPC timeout.
- **SC-004**: GetPluginInfo correctly reports the RESOLVE_RESOURCE_TYPES capability when the
  interface is implemented.
- **SC-005**: TypeRegistry resolves 1000+ registered types with zero allocation overhead per
  lookup after initialization.
- **SC-006**: Plugin developers can register all their type mappings and have the RPC
  auto-handled in fewer than 10 lines of code using TypeRegistry.
- **SC-007**: The feature requires a minor version bump only (additive, non-breaking changes).
