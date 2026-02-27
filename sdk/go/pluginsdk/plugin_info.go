// Package pluginsdk provides plugin information types and builders.
package pluginsdk

import (
	"errors"
	"fmt"
	"maps"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// PluginInfo holds metadata about a plugin that can be reported via GetPluginInfo RPC.
// This information enables compatibility verification and diagnostic visibility.
type PluginInfo struct {
	// Name is the unique identifier for this plugin (required).
	Name string

	// Version is the plugin version following semantic versioning (required).
	// Example: "v1.0.0", "v2.1.3"
	Version string

	// SpecVersion is the finfocus-spec version this plugin implements (required).
	// This value should match the SDK's SpecVersion constant unless the plugin
	// is designed to work with a specific older/newer spec version.
	// Example: "v0.4.11"
	SpecVersion string

	// Providers is a list of cloud providers this plugin supports.
	// Examples: ["aws"], ["azure", "gcp"], ["kubernetes"]
	// An empty list indicates the plugin is provider-agnostic.
	Providers []string

	// Metadata is optional key-value pairs for diagnostic information.
	// Examples: {"build_date": "2024-01-15", "git_commit": "abc123"}
	Metadata map[string]string

	// Capabilities declares the functional capabilities this plugin provides.
	// When set, these are returned in GetPluginInfoResponse.capabilities.
	// When nil or empty, capabilities are auto-discovered from implemented interfaces.
	// This allows plugins to override auto-discovery if needed.
	Capabilities []pbc.PluginCapability
}

// PluginInfoOption is a functional option for configuring PluginInfo.
type PluginInfoOption func(*PluginInfo)

// NewPluginInfo creates a new PluginInfo with the given name and version.
// The SpecVersion is automatically set to the SDK's current SpecVersion constant.
// The PluginInfo should be treated as immutable after passing to ServeConfig.
// Modifying the original slices/maps after construction may cause data races.
// Use functional options to customize additional fields.
//
// Example:
//
//	info := NewPluginInfo("aws-cost-plugin", "v1.0.0",
//	    WithProviders("aws"),
//	    WithMetadata("build_date", "2024-01-15"),
//	)
func NewPluginInfo(name, version string, opts ...PluginInfoOption) *PluginInfo {
	info := &PluginInfo{
		Name:        name,
		Version:     version,
		SpecVersion: SpecVersion, // Use SDK's current spec version
	}

	for _, opt := range opts {
		opt(info)
	}

	return info
}

// WithSpecVersion overrides the spec version for the plugin.
// Use this only if your plugin needs to report a different spec version than the SDK's default.
func WithSpecVersion(specVersion string) PluginInfoOption {
	return func(info *PluginInfo) {
		info.SpecVersion = specVersion
	}
}

// WithProviders sets the list of cloud providers this plugin supports.
// Multiple providers can be specified in a single call.
// A defensive copy is made to prevent aliasing issues if the caller modifies
// the slice after construction.
//
// Example:
//
//	WithProviders("aws", "azure", "gcp")
func WithProviders(providers ...string) PluginInfoOption {
	return func(info *PluginInfo) {
		// Defensive copy to prevent aliasing issues
		info.Providers = append([]string{}, providers...)
	}
}

// WithMetadata adds a key-value pair to the plugin's metadata.
// Multiple calls can be chained to add multiple metadata entries.
//
// Example:
//
//	NewPluginInfo("plugin", "v1.0.0",
//	    WithMetadata("build_date", "2024-01-15"),
//	    WithMetadata("git_commit", "abc123"),
//	)
func WithMetadata(key, value string) PluginInfoOption {
	return func(info *PluginInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]string)
		}
		info.Metadata[key] = value
	}
}

// WithMetadataMap sets the entire metadata map at once.
// This replaces any existing metadata.
// A defensive copy is made to prevent aliasing issues if the caller modifies
// the map after construction.
//
// Example:
//
//	WithMetadataMap(map[string]string{
//	    "build_date": "2024-01-15",
//	    "git_commit": "abc123",
//	})
func WithMetadataMap(metadata map[string]string) PluginInfoOption {
	return func(info *PluginInfo) {
		// Defensive copy to prevent aliasing issues
		info.Metadata = make(map[string]string, len(metadata))
		maps.Copy(info.Metadata, metadata)
	}
}

// WithCapabilities sets the plugin capabilities.
// When set, these override auto-discovery from implemented interfaces.
// A defensive copy is made to prevent aliasing issues.
//
// Example:
//
//	WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
//	                 pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS)
func WithCapabilities(capabilities ...pbc.PluginCapability) PluginInfoOption {
	return func(info *PluginInfo) {
		// Defensive copy to prevent aliasing issues
		info.Capabilities = append([]pbc.PluginCapability{}, capabilities...)
	}
}

// Validate checks that the PluginInfo has all required fields and they are valid.
// Returns an error if validation fails. Returns an error if info is nil.
//
// Error messages include the actual invalid values to aid debugging. This is safe
// because plugin names, versions, and spec versions are not sensitive data.
func (info *PluginInfo) Validate() error {
	if info == nil {
		return errors.New("plugin info validation failed: PluginInfo is nil")
	}
	if info.Name == "" {
		return errors.New("plugin info validation failed: name is required (got empty string)")
	}
	if info.Version == "" {
		return fmt.Errorf("plugin info validation failed: version is required for plugin %q (got empty string)",
			info.Name)
	}
	if info.SpecVersion == "" {
		return fmt.Errorf("plugin info validation failed: spec_version is required for plugin %q (got empty string)",
			info.Name)
	}

	// Validate spec_version is a valid SemVer
	if err := ValidateSpecVersion(info.SpecVersion); err != nil {
		return fmt.Errorf("plugin info validation failed for plugin %q: invalid spec_version %q: %w",
			info.Name, info.SpecVersion, err)
	}

	// DoS protection: reject excessive capability configurations
	// This prevents memory exhaustion from malicious plugins
	if len(info.Capabilities) > MaxConfiguredCapabilities {
		return fmt.Errorf("plugin info validation failed for plugin %q: too many capabilities (%d > %d max)",
			info.Name, len(info.Capabilities), MaxConfiguredCapabilities)
	}

	return nil
}

// Capability counts for pre-allocation during capability inference.
// These constants document the expected capability breakdown and help
// prevent the maxCapabilities constant from becoming stale.
const (
	// baseCapabilities is the number of capabilities always present
	// from the required Plugin interface: GetProjectedCost, GetActualCost,
	// GetPricingSpec, EstimateCost.
	baseCapabilities = 4

	// optionalCapabilities is the number of capabilities from optional
	// interfaces: RecommendationsProvider, BudgetsProvider, DismissProvider,
	// DryRunHandler, BatchCostHandler.
	optionalCapabilities = 5

	// maxCapabilities is the total maximum number of capabilities a plugin
	// can have. Used for pre-allocation to minimize allocations during
	// capability inference.
	maxCapabilities = baseCapabilities + optionalCapabilities

	// MaxConfiguredCapabilities is the maximum number of capabilities allowed
	// in PluginInfo.Capabilities. This limit prevents DoS attacks where malicious
	// plugins configure excessive capabilities to exhaust memory. The limit is
	// generous (64) compared to currently defined capabilities (12) to allow for
	// future growth while still providing protection.
	MaxConfiguredCapabilities = 64

	// minValidCapability is the minimum valid PluginCapability enum value.
	// PLUGIN_CAPABILITY_UNSPECIFIED (0) is the protobuf default and not a valid capability.
	minValidCapability = pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS // 1

	// maxValidCapability is the maximum valid PluginCapability enum value.
	// This should be updated when new capabilities are added to the proto definition.
	maxValidCapability = pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST // 12
)

// IsValidCapability reports whether a PluginCapability value is within the
// inclusive range of supported capability constants. PLUGIN_CAPABILITY_UNSPECIFIED
// (0) is treated as invalid because it is the protobuf default and does not
// represent a real capability.
func IsValidCapability(capability pbc.PluginCapability) bool {
	return capability >= minValidCapability && capability <= maxValidCapability
}

// inferCapabilities returns the PluginCapability values supported by the given
// plugin. If plugin is nil, nil is returned to avoid panicking on type
// assertions. The returned slice always includes the four base capabilities
// (PROJECTED_COSTS, ACTUAL_COSTS, PRICING_SPEC, ESTIMATE_COST) and appends
// additional entries when the plugin implements optional interfaces such as
// RecommendationsProvider, BudgetsProvider, DismissProvider, DryRunHandler,
// or BatchCostHandler.
func inferCapabilities(plugin Plugin) []pbc.PluginCapability {
	// Defensive nil check to prevent panic on type assertions.
	// See function documentation for rationale on handling nil plugins gracefully.
	if plugin == nil {
		return nil
	}

	// Pre-allocate for common case (4 base + 5 optional = maxCapabilities)
	// This reduces allocations from ~2-3 (slice growth) to 1 (initial make)
	capabilities := make([]pbc.PluginCapability, 0, maxCapabilities)

	// Base capabilities - always present since required by Plugin interface
	capabilities = append(capabilities,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
	)

	// Check optional interfaces using zero-allocation type assertions
	if _, ok := plugin.(RecommendationsProvider); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS)
	}
	if _, ok := plugin.(BudgetsProvider); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS)
	}
	if _, ok := plugin.(DismissProvider); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS)
	}
	if _, ok := plugin.(DryRunHandler); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN)
	}
	if _, ok := plugin.(BatchCostHandler); ok {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)
	}

	return capabilities
}
