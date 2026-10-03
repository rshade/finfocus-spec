# Data Model: Plugin Manifest Writer and Validator Agree

No `.proto` message changes. The entities below already exist in `proto/finfocus/v1/registry.proto`;
this feature fixes their serialized form and validation.

## PluginManifest (canonical serialized form)

| Section | Field | Serialized form | Rule |
|---------|-------|-----------------|------|
| `metadata` | `name`, `version`, `description`, `author`, ... | snake_case keys | unchanged |
| `metadata` | `created_at`, `updated_at` | RFC 3339 string, UTC | omitted when unset |
| `specification` | `supported_providers` | array of cloud strings | unchanged |
| `specification` | `supported_resources` | object keyed by cloud, keys sorted | see ProviderResources |
| `specification` | `capabilities` | array of strings | existing 14 strings or a protocol capability name |
| `specification.service_definition` | `methods` | array of RPC names | any `CostSourceService` RPC, unique, no count cap |
| `specification.service_definition` | `port` | JSON number | unchanged |
| `security` | `security_level` | `untrusted`, `community`, `verified`, `official` | omitted when unspecified |
| `installation` | `installation_method` | `binary`, `container`, `script`, `package` | required by the validator |

Encoding: two-space indent, keys sorted at every level, no HTML escaping, trailing newline. YAML carries
the same keys and values in the same order.

Zero values are omitted (proto3 implicit presence), so an unset section does not appear.

## ProviderResources (`supported_resources.<cloud>`)

| Field | Required | Rule |
|-------|----------|------|
| key | yes | one of `aws`, `azure`, `gcp`, `kubernetes`, `custom` |
| `resource_types` | yes | non-empty, unique, each 1 to 256 characters |
| `billing_modes` | no | unique, each in the schema's billing mode list |
| `regions` | no | unique, each 2 to 30 characters |
| other keys | no | rejected |

## Capability names

| Source | Example | Count |
|--------|---------|-------|
| Existing manifest strings | `cost_retrieval`, `caching` | 14 |
| `PluginCapability` (value name minus `PLUGIN_CAPABILITY_`, lowercase) | `dry_run`, `estimate_cost` | 18 (all non-unspecified values) |

`pricing_spec` (protocol) and `pricing_specs` (existing) are distinct accepted values.

## Method names

Every RPC of `finfocus.v1.CostSourceService`, by its proto method name (`Name`, `Supports`,
`GetActualCost`, `GetProjectedCost`, `GetPricingSpec`, `EstimateCost`, `GetRecommendations`,
`DismissRecommendation`, `GetBudgets`, `GetPluginInfo`, `DryRun`, `BatchCost`, `ResolveResourceTypes`).
The list is read from the service descriptor, not maintained by hand.

## Accepted input forms (loader)

| Aspect | Canonical | Also accepted |
|--------|-----------|---------------|
| Keys | `spec_version` | `specVersion` (old JSON), `specversion` (old YAML) |
| Enums | `binary` | `INSTALLATION_METHOD_BINARY`, `1` |
| Timestamps | `"2025-01-01T12:00:00Z"` | `{seconds: ..., nanos: ...}` (old YAML) |
