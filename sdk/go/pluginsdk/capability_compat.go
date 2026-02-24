package pluginsdk

import (
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// capabilityTrue is the string value used for enabled capabilities in legacy metadata.
const capabilityTrue = "true"

// legacyCapabilityNames maps PluginCapability enums to their legacy string names.
// This map is the single source of truth for enum-to-legacy conversions.
//
// Exhaustive Nolint Rationale:
// This map intentionally excludes PLUGIN_CAPABILITY_UNSPECIFIED (value 0) because
// it is the protobuf default sentinel value, not a real capability. All other
// PluginCapability values (1-11) MUST be included in this map.
//
// When adding new capabilities to the proto definition:
// 1. Add a corresponding entry to this map with a "supports_" prefix
// 2. Update IsValidCapability bounds in plugin_info.go (maxValidCapability constant)
// 3. Add capability inference in inferCapabilities() if auto-detectable
//
//nolint:exhaustive,gochecknoglobals // UNSPECIFIED intentionally excluded (protobuf default)
var legacyCapabilityNames = map[pbc.PluginCapability]string{
	pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS:         "supports_recommendations",
	pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN:                 "supports_dry_run",
	pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS:                 "supports_budgets",
	pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS: "supports_dismiss_recommendations",
	pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS:         "supports_projected_costs",
	pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS:            "supports_actual_costs",
	pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC:            "supports_pricing_spec",
	pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST:           "supports_estimate_cost",
	pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON:                  "supports_carbon",
	pbc.PluginCapability_PLUGIN_CAPABILITY_ENERGY:                  "supports_energy",
	pbc.PluginCapability_PLUGIN_CAPABILITY_WATER:                   "supports_water",
}

// CapabilityToLegacyName converts a PluginCapability enum to its legacy string name.
// Returns empty string if the capability has no legacy equivalent.
func CapabilityToLegacyName(capability pbc.PluginCapability) string {
	return legacyCapabilityNames[capability]
}

// CapabilitiesToLegacyMetadata converts capability enums to legacy metadata format.
// Returns a map with "supports_xyz": "true" entries for each capability.
// Capabilities without legacy equivalents are silently skipped.
//
// Deprecated: Use [CapabilitiesToLegacyMetadataWithWarnings] instead to get
// visibility into unmapped capabilities. This function will be removed in v1.0.
func CapabilitiesToLegacyMetadata(capabilities []pbc.PluginCapability) map[string]string {
	metadata, _ := CapabilitiesToLegacyMetadataWithWarnings(capabilities)
	return metadata
}

// CapabilityConversionWarning represents a warning about unmapped capabilities.
type CapabilityConversionWarning struct {
	Capability pbc.PluginCapability
	Reason     string
}

// CapabilitiesToLegacyMetadataWithWarnings converts capabilities and reports unmapped ones.
// Use this when you need to log warnings about capabilities that cannot be represented
// in the legacy format.
func CapabilitiesToLegacyMetadataWithWarnings(
	capabilities []pbc.PluginCapability,
) (map[string]string, []CapabilityConversionWarning) {
	if len(capabilities) == 0 {
		return nil, nil
	}
	metadata := make(map[string]string, len(capabilities))
	var warnings []CapabilityConversionWarning
	for _, capability := range capabilities {
		if name := CapabilityToLegacyName(capability); name != "" {
			metadata[name] = capabilityTrue
		} else if capability != pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED {
			// Capability has no legacy mapping - this indicates either:
			// 1. An invalid enum value (outside valid range)
			// 2. A new capability not yet added to legacyCapabilityNames
			reason := "capability has no legacy metadata mapping"
			if !IsValidCapability(capability) {
				reason = "invalid capability enum value (outside valid range)"
			}
			warnings = append(warnings, CapabilityConversionWarning{
				Capability: capability,
				Reason:     reason,
			})
		}
	}
	return metadata, warnings
}
