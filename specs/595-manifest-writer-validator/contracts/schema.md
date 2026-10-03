# Contract: JSON Schema Changes

## `schemas/plugin_manifest.schema.json`

| Path | Before | After |
|------|--------|-------|
| `specification.supported_resources.*.resource_types.items.maxLength` | 50 | 256 |
| `specification.supported_resources.*.resource_types.description` | "Supported resource types for this provider" | States the entry is matched against `ResourceDescriptor.resource_type`, recommends the full type token, and lists native-package types under their cloud |
| `specification.capabilities.items.enum` | 14 strings | 14 strings plus 18 protocol capability names |
| `specification.service_definition.methods.items.enum` | 5 RPCs | all 13 `CostSourceService` RPCs |
| `specification.service_definition.methods.maxItems` | 5 | removed |

## `schemas/plugin_registry.schema.json`

| Path | Before | After |
|------|--------|-------|
| `$defs.RegistryEntry.properties.capabilities.items.enum` | 14 strings (a different list: includes registry-only `tagging`, `anomaly_detection`, `forecasting`, `alerts`, `custom`) | its 14 strings plus every manifest capability it lacked (37 total) |

All changes widen what is accepted; every document valid before stays valid.

## Drift guard

A Go test reads both schema files and asserts:

- the manifest `methods` enum equals `registry.AllServiceMethods()` (set equality);
- the manifest `capabilities` enum equals `registry.AllPluginCapabilities()` (set equality), and the
  registry index `capabilities` enum contains every value of it;
- the `resource_types` `maxLength` equals `registry.MaxResourceTypeLength`;
- the `billing_modes` enum equals `registry.AllManifestBillingModes()`.
