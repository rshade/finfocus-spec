package pluginsdk_test

import (
	"context"
	"testing"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestNewPluginInfo(t *testing.T) {
	tests := []struct {
		name        string
		pluginName  string
		version     string
		opts        []pluginsdk.PluginInfoOption
		wantName    string
		wantVersion string
		wantSpec    string
		wantProvs   []string
		wantMeta    map[string]string
		wantCaps    []pbc.PluginCapability
	}{
		{
			name:        "basic plugin info",
			pluginName:  "test-plugin",
			version:     "v1.0.0",
			wantName:    "test-plugin",
			wantVersion: "v1.0.0",
			wantSpec:    pluginsdk.SpecVersion,
		},
		{
			name:       "with providers",
			pluginName: "aws-plugin",
			version:    "v2.0.0",
			opts: []pluginsdk.PluginInfoOption{
				pluginsdk.WithProviders("aws", "azure"),
			},
			wantName:    "aws-plugin",
			wantVersion: "v2.0.0",
			wantSpec:    pluginsdk.SpecVersion,
			wantProvs:   []string{"aws", "azure"},
		},
		{
			name:       "with metadata",
			pluginName: "meta-plugin",
			version:    "v1.0.0",
			opts: []pluginsdk.PluginInfoOption{
				pluginsdk.WithMetadata("build_date", "2024-01-15"),
				pluginsdk.WithMetadata("git_commit", "abc123"),
			},
			wantName:    "meta-plugin",
			wantVersion: "v1.0.0",
			wantSpec:    pluginsdk.SpecVersion,
			wantMeta: map[string]string{
				"build_date": "2024-01-15",
				"git_commit": "abc123",
			},
		},
		{
			name:       "with metadata map",
			pluginName: "maptest-plugin",
			version:    "v1.0.0",
			opts: []pluginsdk.PluginInfoOption{
				pluginsdk.WithMetadataMap(map[string]string{
					"key1": "value1",
					"key2": "value2",
				}),
			},
			wantName:    "maptest-plugin",
			wantVersion: "v1.0.0",
			wantSpec:    pluginsdk.SpecVersion,
			wantMeta: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:       "with custom spec version",
			pluginName: "legacy-plugin",
			version:    "v1.0.0",
			opts: []pluginsdk.PluginInfoOption{
				pluginsdk.WithSpecVersion("v0.3.0"),
			},
			wantName:    "legacy-plugin",
			wantVersion: "v1.0.0",
			wantSpec:    "v0.3.0",
		},
		{
			name:       "with all options",
			pluginName: "full-plugin",
			version:    "v3.0.0",
			opts: []pluginsdk.PluginInfoOption{
				pluginsdk.WithProviders("gcp", "kubernetes"),
				pluginsdk.WithMetadata("author", "test"),
				pluginsdk.WithSpecVersion("v0.4.0"),
			},
			wantName:    "full-plugin",
			wantVersion: "v3.0.0",
			wantSpec:    "v0.4.0",
			wantProvs:   []string{"gcp", "kubernetes"},
			wantMeta:    map[string]string{"author": "test"},
		},
		{
			name:       "with capabilities",
			pluginName: "capability-plugin",
			version:    "v1.0.0",
			opts: []pluginsdk.PluginInfoOption{
				pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS),
			},
			wantName:    "capability-plugin",
			wantVersion: "v1.0.0",
			wantSpec:    pluginsdk.SpecVersion,
			wantCaps: []pbc.PluginCapability{
				pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := pluginsdk.NewPluginInfo(tt.pluginName, tt.version, tt.opts...)
			assertPluginInfo(t, info, tt.wantName, tt.wantVersion, tt.wantSpec, tt.wantProvs, tt.wantMeta, tt.wantCaps)
		})
	}
}

func assertPluginInfo(
	t *testing.T,
	info *pluginsdk.PluginInfo,
	wantName, wantVersion, wantSpec string,
	wantProvs []string,
	wantMeta map[string]string,
	wantCaps []pbc.PluginCapability,
) {
	t.Helper()
	if info.Name != wantName {
		t.Errorf("Name = %q, want %q", info.Name, wantName)
	}
	if info.Version != wantVersion {
		t.Errorf("Version = %q, want %q", info.Version, wantVersion)
	}
	if info.SpecVersion != wantSpec {
		t.Errorf("SpecVersion = %q, want %q", info.SpecVersion, wantSpec)
	}

	// Check providers
	if len(info.Providers) != len(wantProvs) {
		t.Errorf("Providers length = %d, want %d", len(info.Providers), len(wantProvs))
	} else {
		for i, p := range info.Providers {
			if p != wantProvs[i] {
				t.Errorf("Providers[%d] = %q, want %q", i, p, wantProvs[i])
			}
		}
	}

	// Check metadata
	if len(info.Metadata) != len(wantMeta) {
		t.Errorf("Metadata length = %d, want %d", len(info.Metadata), len(wantMeta))
	} else {
		for k, v := range wantMeta {
			if info.Metadata[k] != v {
				t.Errorf("Metadata[%q] = %q, want %q", k, info.Metadata[k], v)
			}
		}
	}

	// Check capabilities
	if len(info.Capabilities) != len(wantCaps) {
		t.Errorf("Capabilities length = %d, want %d", len(info.Capabilities), len(wantCaps))
	} else {
		for i, c := range info.Capabilities {
			if c != wantCaps[i] {
				t.Errorf("Capabilities[%d] = %v, want %v", i, c, wantCaps[i])
			}
		}
	}
}

func TestPluginInfoValidate(t *testing.T) {
	tests := []struct {
		name    string
		info    *pluginsdk.PluginInfo
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil plugin info",
			info:    nil,
			wantErr: true,
			errMsg:  "PluginInfo is nil",
		},
		{
			name: "valid plugin info",
			info: &pluginsdk.PluginInfo{
				Name:        "test-plugin",
				Version:     "v1.0.0",
				SpecVersion: "v0.4.11",
			},
			wantErr: false,
		},
		{
			name: "valid with providers and metadata",
			info: &pluginsdk.PluginInfo{
				Name:        "full-plugin",
				Version:     "v2.0.0",
				SpecVersion: "v0.4.11",
				Providers:   []string{"aws", "azure"},
				Metadata:    map[string]string{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			info: &pluginsdk.PluginInfo{
				Name:        "",
				Version:     "v1.0.0",
				SpecVersion: "v0.4.11",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "empty version",
			info: &pluginsdk.PluginInfo{
				Name:        "test-plugin",
				Version:     "",
				SpecVersion: "v0.4.11",
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "empty spec version",
			info: &pluginsdk.PluginInfo{
				Name:        "test-plugin",
				Version:     "v1.0.0",
				SpecVersion: "",
			},
			wantErr: true,
			errMsg:  "spec_version is required",
		},
		{
			name: "invalid spec version format",
			info: &pluginsdk.PluginInfo{
				Name:        "test-plugin",
				Version:     "v1.0.0",
				SpecVersion: "0.4.11", // Missing 'v' prefix
			},
			wantErr: true,
			errMsg:  "not a valid semantic version",
		},
		{
			name: "invalid spec version with prerelease",
			info: &pluginsdk.PluginInfo{
				Name:        "test-plugin",
				Version:     "v1.0.0",
				SpecVersion: "v0.4.11-alpha",
			},
			wantErr: true,
			errMsg:  "not a valid semantic version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestWithMetadataAppends(t *testing.T) {
	// Verify that multiple WithMetadata calls append rather than replace
	info := pluginsdk.NewPluginInfo("test", "v1.0.0",
		pluginsdk.WithMetadata("key1", "value1"),
		pluginsdk.WithMetadata("key2", "value2"),
		pluginsdk.WithMetadata("key3", "value3"),
	)

	if len(info.Metadata) != 3 {
		t.Errorf("Expected 3 metadata entries, got %d", len(info.Metadata))
	}

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range expected {
		if info.Metadata[k] != v {
			t.Errorf("Metadata[%q] = %q, want %q", k, info.Metadata[k], v)
		}
	}
}

func TestWithMetadataMapReplacesExisting(t *testing.T) {
	// Verify that WithMetadataMap replaces any existing metadata
	info := pluginsdk.NewPluginInfo("test", "v1.0.0",
		pluginsdk.WithMetadata("old_key", "old_value"),
		pluginsdk.WithMetadataMap(map[string]string{
			"new_key": "new_value",
		}),
	)

	if len(info.Metadata) != 1 {
		t.Errorf("Expected 1 metadata entry after replace, got %d", len(info.Metadata))
	}

	if info.Metadata["new_key"] != "new_value" {
		t.Errorf("Metadata[new_key] = %q, want %q", info.Metadata["new_key"], "new_value")
	}

	if _, exists := info.Metadata["old_key"]; exists {
		t.Error("old_key should not exist after WithMetadataMap")
	}
}

func BenchmarkNewPluginInfo(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = pluginsdk.NewPluginInfo("bench-plugin", "v1.0.0",
			pluginsdk.WithProviders("aws", "azure", "gcp"),
			pluginsdk.WithMetadata("key", "value"),
		)
	}
}

func BenchmarkPluginInfoValidate(b *testing.B) {
	info := pluginsdk.NewPluginInfo("bench-plugin", "v1.0.0",
		pluginsdk.WithProviders("aws"),
		pluginsdk.WithMetadata("key", "value"),
	)

	b.ResetTimer()
	for range b.N {
		_ = info.Validate()
	}
}

// Edge Case Tests

func TestPluginInfoWithEmptyProviders(t *testing.T) {
	// Test with empty but non-nil providers slice
	info := pluginsdk.NewPluginInfo("test-plugin", "v1.0.0",
		pluginsdk.WithProviders(), // Empty variadic call creates empty slice
	)

	if info.Providers == nil {
		t.Error("Expected non-nil empty providers slice")
	}
	if len(info.Providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(info.Providers))
	}

	// Should still validate successfully
	if err := info.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestPluginInfoWithExplicitNilMetadata(t *testing.T) {
	// Test with explicitly nil metadata map (not just unset)
	info := &pluginsdk.PluginInfo{
		Name:        "test-plugin",
		Version:     "v1.0.0",
		SpecVersion: pluginsdk.SpecVersion,
		Providers:   []string{"aws"},
		Metadata:    nil, // Explicitly nil
	}

	// Should validate successfully (metadata is optional)
	if err := info.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestPluginInfoWithLongName(t *testing.T) {
	// Test with a very long plugin name (boundary testing)
	// 256 characters is a reasonable max for identifiers
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}

	info := pluginsdk.NewPluginInfo(string(longName), "v1.0.0")

	if info.Name != string(longName) {
		t.Error("Long plugin name was not preserved")
	}

	// Should still validate successfully (no length limit enforced)
	if err := info.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error for long name: %v", err)
	}
}

func TestWithProvidersDefensiveCopy(t *testing.T) {
	// Verify that WithProviders creates a defensive copy
	originalProviders := []string{"aws", "azure"}
	info := pluginsdk.NewPluginInfo("test-plugin", "v1.0.0",
		pluginsdk.WithProviders(originalProviders...),
	)

	// Modify the original slice after construction
	originalProviders[0] = "modified"
	originalProviders[1] = "also-modified"

	// The info's providers should be unchanged
	if info.Providers[0] != "aws" {
		t.Errorf("Expected Providers[0] = %q, got %q (defensive copy failed)", "aws", info.Providers[0])
	}
	if info.Providers[1] != "azure" {
		t.Errorf("Expected Providers[1] = %q, got %q (defensive copy failed)", "azure", info.Providers[1])
	}
}

func TestWithMetadataMapDefensiveCopy(t *testing.T) {
	// Verify that WithMetadataMap creates a defensive copy
	originalMetadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	info := pluginsdk.NewPluginInfo("test-plugin", "v1.0.0",
		pluginsdk.WithMetadataMap(originalMetadata),
	)

	// Modify the original map after construction
	originalMetadata["key1"] = "modified"
	originalMetadata["new_key"] = "new_value"

	// The info's metadata should be unchanged
	if info.Metadata["key1"] != "value1" {
		t.Errorf("Expected Metadata[key1] = %q, got %q (defensive copy failed)", "value1", info.Metadata["key1"])
	}
	if _, exists := info.Metadata["new_key"]; exists {
		t.Error("new_key should not exist in info's metadata (defensive copy failed)")
	}
	if len(info.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d (defensive copy failed)", len(info.Metadata))
	}
}

func TestLegacyCapabilityMapCompleteness(t *testing.T) {
	// Verify all PluginCapability enum values (except UNSPECIFIED) have legacy mappings.
	// This test prevents the legacyCapabilityNames from becoming stale when new
	// capabilities are added to the proto definition.
	allCaps := pbc.PluginCapability_name
	for capValue, capName := range allCaps {
		capability := pbc.PluginCapability(capValue)
		if capability == pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED {
			continue // UNSPECIFIED is intentionally excluded from legacy mapping
		}
		if pluginsdk.CapabilityToLegacyName(capability) == "" {
			t.Errorf("Capability %s (value %d) missing from legacyCapabilityNames", capName, capValue)
		}
	}
}

// Mock Interfaces for Auto-Discovery Testing

type mockDryRunHandler struct {
	pbc.UnimplementedCostSourceServiceServer
}

func (m *mockDryRunHandler) Name() string { return "mock" }
func (m *mockDryRunHandler) GetProjectedCost(
	context.Context,
	*pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}
func (m *mockDryRunHandler) GetActualCost(
	context.Context,
	*pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}
func (m *mockDryRunHandler) GetPricingSpec(
	context.Context,
	*pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}
func (m *mockDryRunHandler) EstimateCost(
	context.Context,
	*pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{}, nil
}

func (m *mockDryRunHandler) HandleDryRun(_ context.Context, _ *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
	return &pbc.DryRunResponse{}, nil
}

type mockRecsProvider struct {
	mockDryRunHandler
}

func (m *mockRecsProvider) GetRecommendations(
	context.Context,
	*pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return &pbc.GetRecommendationsResponse{}, nil
}

// TestAutoDiscovery is implicitly tested by the fact that inferCapabilities
// is called by NewServer. We can verify it by creating a server and checking its capabilities.
func TestAutoDiscovery(t *testing.T) {
	t.Run("DryRunHandler", func(t *testing.T) {
		plugin := &mockDryRunHandler{}
		info := pluginsdk.NewPluginInfo("test", "v1.0.0")
		server := pluginsdk.NewServerWithOptions(plugin, nil, nil, info)

		// DryRunHandler should be auto-discovered
		found := false
		for _, capability := range server.GetGlobalCapabilities() {
			if capability == pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected DryRun capability to be auto-discovered")
		}
	})

	t.Run("RecommendationsProvider", func(t *testing.T) {
		plugin := &mockRecsProvider{}
		info := pluginsdk.NewPluginInfo("test", "v1.0.0")
		server := pluginsdk.NewServerWithOptions(plugin, nil, nil, info)

		// Recommendations should be auto-discovered
		found := false
		for _, capability := range server.GetGlobalCapabilities() {
			if capability == pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected Recommendations capability to be auto-discovered")
		}
	})
}

func TestWithCapabilitiesOverride(t *testing.T) {
	// Test that WithCapabilities overrides any defaults
	info := pluginsdk.NewPluginInfo("test", "v1.0.0",
		pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN),
	)

	if len(info.Capabilities) != 1 {
		t.Errorf("Expected 1 capability, got %d", len(info.Capabilities))
	}
	if info.Capabilities[0] != pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN {
		t.Errorf("Expected DryRun capability")
	}
}
