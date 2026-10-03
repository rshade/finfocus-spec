# Contract: Go SDK Surface

## `sdk/go/pluginsdk`

```go
// Changed behavior: writes the canonical manifest form (snake_case keys, schema enum strings,
// sorted keys, trailing newline). JSON for .json, YAML for .yaml/.yml. Signature unchanged.
func SaveManifest(path string, m *pbc.PluginManifest) error

// Changed behavior: reads the canonical form and the previous writer formats. Signature unchanged.
// Does not validate; callers validate with registry.ValidatePluginManifest.
func LoadManifest(path string) (*pbc.PluginManifest, error)

// New: canonical JSON bytes, identical to what SaveManifest writes for a .json path.
func MarshalManifestJSON(m *pbc.PluginManifest) ([]byte, error)

// New: canonical YAML bytes, identical to what SaveManifest writes for a .yaml path.
func MarshalManifestYAML(m *pbc.PluginManifest) ([]byte, error)
```

Guarantee: for any manifest `m`, `registry.ValidatePluginManifest(MarshalManifestJSON(m))` gives the same
verdict as the JSON schema, and `LoadManifest(SaveManifest(m))` equals `m` (`proto.Equal`).

## `sdk/go/registry`

```go
// New: longest accepted supported_resources.<cloud>.resource_types entry.
const MaxResourceTypeLength = 256

// New: the 38 billing mode strings the manifest schema accepts in supported_resources, in schema order.
// Distinct from the pricing package's BillingMode enum.
func AllManifestBillingModes() []string

// New: reports whether mode is in AllManifestBillingModes. Zero allocations.
func IsValidManifestBillingMode(mode string) bool

// New: manifest string for a protocol capability ("dry_run" for PLUGIN_CAPABILITY_DRY_RUN).
// Returns "" for UNSPECIFIED and for values not in the enum.
func ManifestCapabilityName(c pbc.PluginCapability) string

// New: reports whether name is an RPC of finfocus.v1.CostSourceService. Zero allocations.
func IsValidServiceMethod(name string) bool

// New: the CostSourceService RPC names, in proto declaration order.
func AllServiceMethods() []string

// Changed: also accepts every protocol capability name. Zero allocations.
func IsValidPluginCapability(capability string) bool

// Changed: returns the 14 existing values followed by the protocol capability names in enum order.
func AllPluginCapabilities() []PluginCapability

// Changed: also validates supported_resources and capabilities; methods accept every
// CostSourceService RPC. Existing error messages for other fields are unchanged.
func ValidatePluginManifest(manifestJSON []byte) error
```

Error message format for new checks (prefix is the JSON path):

```text
specification.supported_resources: 'azure-native' is not a valid provider, must be one of: aws, azure, gcp, kubernetes, custom
specification.supported_resources.azure.resource_types is required
specification.supported_resources.azure.resource_types[2] must be 1 to 256 characters
specification.supported_resources.azure.billing_modes[0]: 'hourly' is not a valid billing mode
specification.supported_resources.azure.regions contains duplicate region 'eastus'
specification.capabilities[1]: 'teleport' is not a valid capability
```
