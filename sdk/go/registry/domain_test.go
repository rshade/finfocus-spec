package registry_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/registry"
)

func TestDiscoverySource(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{"filesystem", true},
		{"registry", true},
		{"url", true},
		{"git", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidDiscoverySource(test.source)
		if result != test.expected {
			t.Errorf("registry.IsValidDiscoverySource(%q) = %v, expected %v", test.source, result, test.expected)
		}
	}
}

func TestPluginStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"available", true},
		{"installed", true},
		{"active", true},
		{"inactive", true},
		{"error", true},
		{"updating", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidPluginStatus(test.status)
		if result != test.expected {
			t.Errorf("registry.IsValidPluginStatus(%q) = %v, expected %v", test.status, result, test.expected)
		}
	}
}

func TestSecurityLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"untrusted", true},
		{"community", true},
		{"verified", true},
		{"official", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidSecurityLevel(test.level)
		if result != test.expected {
			t.Errorf("registry.IsValidSecurityLevel(%q) = %v, expected %v", test.level, result, test.expected)
		}
	}
}

func TestInstallationMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"binary", true},
		{"container", true},
		{"script", true},
		{"package", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidInstallationMethod(test.method)
		if result != test.expected {
			t.Errorf("registry.IsValidInstallationMethod(%q) = %v, expected %v", test.method, result, test.expected)
		}
	}
}

func TestPluginCapability(t *testing.T) {
	tests := []struct {
		capability string
		expected   bool
	}{
		{"cost_retrieval", true},
		{"cost_projection", true},
		{"pricing_specs", true},
		{"historical_data", true},
		{"real_time_data", true},
		{"batch_processing", true},
		{"rate_limiting", true},
		{"caching", true},
		{"encryption", true},
		{"compression", true},
		{"filtering", true},
		{"aggregation", true},
		{"multi_tenancy", true},
		{"audit_logging", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidPluginCapability(test.capability)
		if result != test.expected {
			t.Errorf("registry.IsValidPluginCapability(%q) = %v, expected %v", test.capability, result, test.expected)
		}
	}
}

func TestSystemPermission(t *testing.T) {
	tests := []struct {
		permission string
		expected   bool
	}{
		{"network_access", true},
		{"filesystem_read", true},
		{"filesystem_write", true},
		{"environment_read", true},
		{"process_spawn", true},
		{"system_info", true},
		{"temp_files", true},
		{"config_read", true},
		{"metrics_collect", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidSystemPermission(test.permission)
		if result != test.expected {
			t.Errorf("registry.IsValidSystemPermission(%q) = %v, expected %v", test.permission, result, test.expected)
		}
	}
}

func TestAuthMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"none", true},
		{"api_key", true},
		{"jwt", true},
		{"oauth2", true},
		{"mtls", true},
		{"basic_auth", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := registry.IsValidAuthMethod(test.method)
		if result != test.expected {
			t.Errorf("registry.IsValidAuthMethod(%q) = %v, expected %v", test.method, result, test.expected)
		}
	}
}

func TestProvider(t *testing.T) {
	tests := []struct {
		provider string
		expected bool
	}{
		{"aws", true},
		{"azure", true},
		{"gcp", true},
		{"kubernetes", true},
		{"custom", true},
		{"invalid", false},
		{"", false},
		{"AWS", false},    // Case sensitive
		{"amazon", false}, // Not valid
		{"AZURE", false},  // Case sensitive
		{"k8s", false},    // Must be full name
	}

	for _, test := range tests {
		result := registry.IsValidProvider(test.provider)
		if result != test.expected {
			t.Errorf("registry.IsValidProvider(%q) = %v, expected %v", test.provider, result, test.expected)
		}
	}
}

func TestProviderString(t *testing.T) {
	tests := []struct {
		provider registry.Provider
		expected string
	}{
		{registry.ProviderAWS, "aws"},
		{registry.ProviderAzure, "azure"},
		{registry.ProviderGCP, "gcp"},
		{registry.ProviderKubernetes, "kubernetes"},
		{registry.ProviderCustom, "custom"},
	}

	for _, test := range tests {
		result := test.provider.String()
		if result != test.expected {
			t.Errorf("registry.Provider(%q).String() = %q, expected %q", test.provider, result, test.expected)
		}
	}
}

func TestDiscoverySourceString(t *testing.T) {
	tests := []struct {
		source   registry.DiscoverySource
		expected string
	}{
		{registry.DiscoverySourceFilesystem, "filesystem"},
		{registry.DiscoverySourceRegistry, "registry"},
		{registry.DiscoverySourceURL, "url"},
		{registry.DiscoverySourceGit, "git"},
	}

	for _, test := range tests {
		result := test.source.String()
		if result != test.expected {
			t.Errorf("registry.DiscoverySource(%q).String() = %q, expected %q", test.source, result, test.expected)
		}
	}
}

func TestPluginStatusString(t *testing.T) {
	tests := []struct {
		status   registry.PluginStatus
		expected string
	}{
		{registry.PluginStatusAvailable, "available"},
		{registry.PluginStatusInstalled, "installed"},
		{registry.PluginStatusActive, "active"},
		{registry.PluginStatusInactive, "inactive"},
		{registry.PluginStatusError, "error"},
		{registry.PluginStatusUpdating, "updating"},
	}

	for _, test := range tests {
		result := test.status.String()
		if result != test.expected {
			t.Errorf("registry.PluginStatus(%q).String() = %q, expected %q", test.status, result, test.expected)
		}
	}
}

func TestSecurityLevelString(t *testing.T) {
	tests := []struct {
		level    registry.SecurityLevel
		expected string
	}{
		{registry.SecurityLevelUntrusted, "untrusted"},
		{registry.SecurityLevelCommunity, "community"},
		{registry.SecurityLevelVerified, "verified"},
		{registry.SecurityLevelOfficial, "official"},
	}

	for _, test := range tests {
		result := test.level.String()
		if result != test.expected {
			t.Errorf("registry.SecurityLevel(%q).String() = %q, expected %q", test.level, result, test.expected)
		}
	}
}

func TestInstallationMethodString(t *testing.T) {
	tests := []struct {
		method   registry.InstallationMethod
		expected string
	}{
		{registry.InstallationMethodBinary, "binary"},
		{registry.InstallationMethodContainer, "container"},
		{registry.InstallationMethodScript, "script"},
		{registry.InstallationMethodPackage, "package"},
	}

	for _, test := range tests {
		result := test.method.String()
		if result != test.expected {
			t.Errorf("registry.InstallationMethod(%q).String() = %q, expected %q", test.method, result, test.expected)
		}
	}
}

func TestPluginCapabilityString(t *testing.T) {
	tests := []struct {
		capability registry.PluginCapability
		expected   string
	}{
		{registry.PluginCapabilityCostRetrieval, "cost_retrieval"},
		{registry.PluginCapabilityCostProjection, "cost_projection"},
		{registry.PluginCapabilityPricingSpecs, "pricing_specs"},
		{registry.PluginCapabilityHistoricalData, "historical_data"},
		{registry.PluginCapabilityRealTimeData, "real_time_data"},
		{registry.PluginCapabilityBatchProcessing, "batch_processing"},
		{registry.PluginCapabilityRateLimiting, "rate_limiting"},
		{registry.PluginCapabilityCaching, "caching"},
		{registry.PluginCapabilityEncryption, "encryption"},
		{registry.PluginCapabilityCompression, "compression"},
		{registry.PluginCapabilityFiltering, "filtering"},
		{registry.PluginCapabilityAggregation, "aggregation"},
		{registry.PluginCapabilityMultiTenancy, "multi_tenancy"},
		{registry.PluginCapabilityAuditLogging, "audit_logging"},
	}

	for _, test := range tests {
		result := test.capability.String()
		if result != test.expected {
			t.Errorf("registry.PluginCapability(%q).String() = %q, expected %q", test.capability, result, test.expected)
		}
	}
}

func TestSystemPermissionString(t *testing.T) {
	tests := []struct {
		permission registry.SystemPermission
		expected   string
	}{
		{registry.SystemPermissionNetworkAccess, "network_access"},
		{registry.SystemPermissionFilesystemRead, "filesystem_read"},
		{registry.SystemPermissionFilesystemWrite, "filesystem_write"},
		{registry.SystemPermissionEnvironmentRead, "environment_read"},
		{registry.SystemPermissionProcessSpawn, "process_spawn"},
		{registry.SystemPermissionSystemInfo, "system_info"},
		{registry.SystemPermissionTempFiles, "temp_files"},
		{registry.SystemPermissionConfigRead, "config_read"},
		{registry.SystemPermissionMetricsCollect, "metrics_collect"},
	}

	for _, test := range tests {
		result := test.permission.String()
		if result != test.expected {
			t.Errorf("registry.SystemPermission(%q).String() = %q, expected %q", test.permission, result, test.expected)
		}
	}
}

func TestAuthMethodString(t *testing.T) {
	tests := []struct {
		method   registry.AuthMethod
		expected string
	}{
		{registry.AuthMethodNone, "none"},
		{registry.AuthMethodAPIKey, "api_key"},
		{registry.AuthMethodJWT, "jwt"},
		{registry.AuthMethodOAuth2, "oauth2"},
		{registry.AuthMethodMTLS, "mtls"},
		{registry.AuthMethodBasicAuth, "basic_auth"},
	}

	for _, test := range tests {
		result := test.method.String()
		if result != test.expected {
			t.Errorf("registry.AuthMethod(%q).String() = %q, expected %q", test.method, result, test.expected)
		}
	}
}

func TestValidatePluginName(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{"valid-plugin", false},
		{"plugin123", false},
		{"ab", false},
		{"plugin-name-123", false},
		{"", true},
		{"a", true},              // single character not allowed (min length 2)
		{"x", true},              // single character not allowed (min length 2)
		{"invalid_plugin", true}, // underscore not allowed
		{"Invalid-Plugin", true}, // uppercase not allowed
		{"-invalid", true},       // cannot start with dash
		{"invalid-", true},       // cannot end with dash
		{"plugin--name", false},  // double dash allowed
		{"plugin name", true},    // space not allowed
		{"plugin.name", true},    // dot not allowed
		{"this-is-a-very-long-plugin-name-that-exceeds-fifty-characters", true}, // too long
	}

	for _, test := range tests {
		err := registry.ValidatePluginName(test.name)
		hasError := err != nil
		if hasError != test.expectError {
			if test.expectError {
				t.Errorf("registry.ValidatePluginName(%q) expected error, but got none", test.name)
			} else {
				t.Errorf("registry.ValidatePluginName(%q) unexpected error: %v", test.name, err)
			}
		}
	}
}

func TestAllFunctions(t *testing.T) {
	// Test that all enumeration functions return expected counts
	tests := []struct {
		name     string
		count    int
		function func() int
	}{
		{"registry.AllProviders", 5, func() int { return len(registry.AllProviders()) }},
		{"registry.AllDiscoverySources", 4, func() int { return len(registry.AllDiscoverySources()) }},
		{"registry.AllPluginStatuses", 6, func() int { return len(registry.AllPluginStatuses()) }},
		{"registry.AllSecurityLevels", 4, func() int { return len(registry.AllSecurityLevels()) }},
		{"registry.AllInstallationMethods", 4, func() int { return len(registry.AllInstallationMethods()) }},
		{"registry.AllPluginCapabilities", 32, func() int { return len(registry.AllPluginCapabilities()) }},
		{"registry.AllSystemPermissions", 9, func() int { return len(registry.AllSystemPermissions()) }},
		{"registry.AllAuthMethods", 6, func() int { return len(registry.AllAuthMethods()) }},
	}

	for _, test := range tests {
		count := test.function()
		if count != test.count {
			t.Errorf("%s returned %d items, expected %d", test.name, count, test.count)
		}
	}
}

// Benchmark tests for enum validation performance (T006-T013)
// These benchmarks measure the current implementation before optimization

func BenchmarkIsValidProvider(b *testing.B) {
	testCases := []string{"aws", "invalid", "gcp", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidProvider(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidDiscoverySource(b *testing.B) {
	testCases := []string{"filesystem", "invalid", "registry", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidDiscoverySource(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidPluginStatus(b *testing.B) {
	testCases := []string{"active", "invalid", "installed", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidPluginStatus(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidSecurityLevel(b *testing.B) {
	testCases := []string{"verified", "invalid", "official", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidSecurityLevel(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidInstallationMethod(b *testing.B) {
	testCases := []string{"binary", "invalid", "container", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidInstallationMethod(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidPluginCapability(b *testing.B) {
	testCases := []string{"cost_retrieval", "invalid", "caching", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidPluginCapability(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidSystemPermission(b *testing.B) {
	testCases := []string{"network_access", "invalid", "filesystem_read", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidSystemPermission(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidAuthMethod(b *testing.B) {
	testCases := []string{"api_key", "invalid", "jwt", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidAuthMethod(testCases[i%len(testCases)])
	}
}

// Edge case tests (T014).
func TestValidationEdgeCases(b *testing.T) {
	b.Run("EmptyString", func(t *testing.T) {
		if registry.IsValidProvider("") {
			t.Error("Empty string should be invalid")
		}
		if registry.IsValidDiscoverySource("") {
			t.Error("Empty string should be invalid")
		}
	})

	b.Run("CaseMismatch", func(t *testing.T) {
		if registry.IsValidProvider("AWS") {
			t.Error("Uppercase 'AWS' should be invalid (case-sensitive)")
		}
		if registry.IsValidPluginStatus("ACTIVE") {
			t.Error("Uppercase 'ACTIVE' should be invalid (case-sensitive)")
		}
	})

	b.Run("InvalidValues", func(t *testing.T) {
		if registry.IsValidProvider("invalid-provider") {
			t.Error("'invalid-provider' should be invalid")
		}
		if registry.IsValidPluginCapability("nonexistent") {
			t.Error("'nonexistent' should be invalid")
		}
	})
}

// Map-based comparison benchmarks (T045-T046).
// These benchmarks compare optimized slice-based validation against map-based approach.

// BenchmarkIsValidProvider_MapBased compares map-based validation for Provider enum (5 values).
func BenchmarkIsValidProvider_MapBased(b *testing.B) {
	// Map-based validation for comparison
	validProviders := map[registry.Provider]struct{}{
		registry.ProviderAWS:        {},
		registry.ProviderAzure:      {},
		registry.ProviderGCP:        {},
		registry.ProviderKubernetes: {},
		registry.ProviderCustom:     {},
	}

	testCases := []string{"aws", "invalid", "gcp", ""}
	b.ResetTimer()
	for i := range b.N {
		provider := registry.Provider(testCases[i%len(testCases)])
		_ = validProviders[provider]
	}
}

// BenchmarkIsValidPluginCapability_MapBased compares map-based validation for PluginCapability enum (14 values).
func BenchmarkIsValidPluginCapability_MapBased(b *testing.B) {
	// Map-based validation for comparison
	validCapabilities := map[registry.PluginCapability]struct{}{
		registry.PluginCapabilityCostRetrieval:   {},
		registry.PluginCapabilityCostProjection:  {},
		registry.PluginCapabilityPricingSpecs:    {},
		registry.PluginCapabilityHistoricalData:  {},
		registry.PluginCapabilityRealTimeData:    {},
		registry.PluginCapabilityBatchProcessing: {},
		registry.PluginCapabilityRateLimiting:    {},
		registry.PluginCapabilityCaching:         {},
		registry.PluginCapabilityEncryption:      {},
		registry.PluginCapabilityCompression:     {},
		registry.PluginCapabilityFiltering:       {},
		registry.PluginCapabilityAggregation:     {},
		registry.PluginCapabilityMultiTenancy:    {},
		registry.PluginCapabilityAuditLogging:    {},
	}

	testCases := []string{"cost_retrieval", "invalid", "caching", ""}
	b.ResetTimer()
	for i := range b.N {
		capability := registry.PluginCapability(testCases[i%len(testCases)])
		_ = validCapabilities[capability]
	}
}

// Scalability benchmarks (T047).
// These benchmarks test validation performance across different enum sizes.

// BenchmarkValidation_4Values tests validation performance for 4-value enums (DiscoverySource, SecurityLevel, InstallationMethod).
func BenchmarkValidation_4Values(b *testing.B) {
	testCases := []string{"filesystem", "invalid", "registry", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidDiscoverySource(testCases[i%len(testCases)])
	}
}

// BenchmarkValidation_5Values tests validation performance for 5-value enums (Provider).
func BenchmarkValidation_5Values(b *testing.B) {
	testCases := []string{"aws", "invalid", "gcp", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidProvider(testCases[i%len(testCases)])
	}
}

// BenchmarkValidation_6Values tests validation performance for 6-value enums (PluginStatus, AuthMethod).
func BenchmarkValidation_6Values(b *testing.B) {
	testCases := []string{"active", "invalid", "installed", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidPluginStatus(testCases[i%len(testCases)])
	}
}

// BenchmarkValidation_9Values tests validation performance for 9-value enums (SystemPermission).
func BenchmarkValidation_9Values(b *testing.B) {
	testCases := []string{"network_access", "invalid", "filesystem_read", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidSystemPermission(testCases[i%len(testCases)])
	}
}

// BenchmarkValidation_14Values tests validation performance for 14-value enums (PluginCapability).
func BenchmarkValidation_14Values(b *testing.B) {
	testCases := []string{"cost_retrieval", "invalid", "caching", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidPluginCapability(testCases[i%len(testCases)])
	}
}

func TestServiceMethods(t *testing.T) {
	want := []string{
		"Name", "Supports", "GetActualCost", "GetProjectedCost", "GetPricingSpec", "EstimateCost",
		"GetRecommendations", "DismissRecommendation", "GetBudgets", "GetPluginInfo", "DryRun",
		"BatchCost", "ResolveResourceTypes",
	}
	require.Equal(t, want, registry.AllServiceMethods())
	for _, method := range want {
		require.True(t, registry.IsValidServiceMethod(method), method)
	}
	for _, method := range []string{"Allocate", "GetStats", "HealthCheck", "", "name"} {
		require.False(t, registry.IsValidServiceMethod(method), method)
	}
}

func TestManifestCapabilityName(t *testing.T) {
	require.Equal(t, "dry_run", registry.ManifestCapabilityName(pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN))
	require.Equal(t, "recommendation_scoring",
		registry.ManifestCapabilityName(pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATION_SCORING))
	require.Empty(t, registry.ManifestCapabilityName(pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED))
	require.Empty(t, registry.ManifestCapabilityName(pbc.PluginCapability(999)))

	values := pbc.PluginCapability(0).Descriptor().Values()
	for i := 1; i < values.Len(); i++ {
		c := pbc.PluginCapability(values.Get(i).Number())
		name := registry.ManifestCapabilityName(c)
		require.NotEmpty(t, name, c.String())
		require.True(t, registry.IsValidPluginCapability(name), name)
	}
	for _, capability := range []string{"cost_retrieval", "caching", "pricing_specs", "pricing_spec"} {
		require.True(t, registry.IsValidPluginCapability(capability), capability)
	}
	for _, capability := range []string{"teleport", "DRY_RUN", "PLUGIN_CAPABILITY_DRY_RUN", "unspecified", ""} {
		require.False(t, registry.IsValidPluginCapability(capability), capability)
	}
}

func TestManifestBillingModes(t *testing.T) {
	require.Len(t, registry.AllManifestBillingModes(), 38)
	require.True(t, registry.IsValidManifestBillingMode("per_hour"))
	require.True(t, registry.IsValidManifestBillingMode("per_query"))
	require.False(t, registry.IsValidManifestBillingMode("hourly"))
}

func BenchmarkIsValidServiceMethod(b *testing.B) {
	testCases := []string{"GetProjectedCost", "invalid", "ResolveResourceTypes", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidServiceMethod(testCases[i%len(testCases)])
	}
}

func BenchmarkIsValidProtoPluginCapability(b *testing.B) {
	testCases := []string{"recommendation_scoring", "invalid", "dry_run", ""}
	b.ResetTimer()
	for i := range b.N {
		_ = registry.IsValidPluginCapability(testCases[i%len(testCases)])
	}
}
