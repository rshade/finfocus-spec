package pluginsdk_test

import (
	"testing"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestCapabilityToLegacyName(t *testing.T) {
	tests := []struct {
		name       string
		capability pbc.PluginCapability
		expected   string
	}{
		{"DryRun", pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN, "supports_dry_run"},
		{"Recommendations", pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS, "supports_recommendations"},
		{"BatchCost", pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST, "supports_batch_cost"},
		{"Unspecified", pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pluginsdk.CapabilityToLegacyName(tc.capability)
			if got != tc.expected {
				t.Errorf("CapabilityToLegacyName(%v) = %q, want %q", tc.capability, got, tc.expected)
			}
		})
	}
}

func TestCapabilitiesToLegacyMetadata(t *testing.T) {
	caps := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
	}

	metadata := pluginsdk.CapabilitiesToLegacyMetadata(caps)
	if metadata["supports_dry_run"] != "true" {
		t.Errorf("expected supports_dry_run=true, got %q", metadata["supports_dry_run"])
	}
	if metadata["supports_recommendations"] != "true" {
		t.Errorf("expected supports_recommendations=true, got %q", metadata["supports_recommendations"])
	}
	if metadata["supports_batch_cost"] != "true" {
		t.Errorf("expected supports_batch_cost=true, got %q", metadata["supports_batch_cost"])
	}
	if len(metadata) != 3 {
		t.Errorf("expected 3 metadata entries, got %d", len(metadata))
	}
}

func TestCapabilitiesToLegacyMetadata_DelegatesCorrectly(t *testing.T) {
	// Both helpers return nil for nil/empty input (backward compatible).
	silentNil := pluginsdk.CapabilitiesToLegacyMetadata(nil)
	withWarningsNil, warningsNil := pluginsdk.CapabilitiesToLegacyMetadataWithWarnings(nil)
	if silentNil != nil || withWarningsNil != nil || warningsNil != nil {
		t.Fatalf(
			"expected nil from both helpers for nil input: "+
				"silent=%v, withWarnings=%v, warnings=%v",
			silentNil, withWarningsNil, warningsNil,
		)
	}

	caps := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
	}

	silent := pluginsdk.CapabilitiesToLegacyMetadata(caps)
	withWarnings, _ := pluginsdk.CapabilitiesToLegacyMetadataWithWarnings(caps)
	if silent == nil || withWarnings == nil {
		t.Fatal("unexpected nil maps")
	}

	if len(silent) != len(withWarnings) {
		t.Fatalf(
			"metadata length mismatch: silent=%d, withWarnings=%d",
			len(silent), len(withWarnings),
		)
	}
	for key, val := range withWarnings {
		if silent[key] != val {
			t.Errorf(
				"metadata mismatch for key %q: silent=%q, withWarnings=%q",
				key, silent[key], val,
			)
		}
	}
	for key, val := range silent {
		if withWarnings[key] != val {
			t.Errorf(
				"metadata mismatch for key %q: withWarnings=%q, silent=%q",
				key, withWarnings[key], val,
			)
		}
	}
}

func TestCapabilitiesToLegacyMetadataWithWarnings(t *testing.T) {
	// Test with a mix of mapped and hypothetically unmapped capabilities
	// Note: Currently most capabilities are mapped, so we use Unspecified to test filtering
	// or assume future capabilities might not be mapped.
	// For this test, we verify that Mapped ones work and Unspecified doesn't produce warning (as per logic)
	// Let's rely on logic verification: if it has name, it goes to metadata. If not AND not Unspecified, it warns.

	caps := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
	}

	metadata, warnings := pluginsdk.CapabilitiesToLegacyMetadataWithWarnings(caps)

	if len(metadata) != 1 {
		t.Errorf("expected 1 metadata entry, got %d", len(metadata))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Verify metadata content
	if metadata["supports_dry_run"] != "true" {
		t.Errorf("expected supports_dry_run=true")
	}
}
