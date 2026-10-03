// Package registry provides domain types and validation for FinFocus plugin registry management.
// It defines enums for discovery sources, plugin statuses, security levels, installation methods,
// capabilities, system permissions, authentication methods, and validation functions for plugin
// manifests and names.
package registry

import (
	"errors"
	"fmt"
	"strings"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Provider represents supported cloud providers.
type Provider string

const (
	// ProviderAWS indicates Amazon Web Services provider.
	ProviderAWS Provider = "aws"
	// ProviderAzure indicates Microsoft Azure provider.
	ProviderAzure Provider = "azure"
	// ProviderGCP indicates Google Cloud Platform provider.
	ProviderGCP Provider = "gcp"
	// ProviderKubernetes indicates Kubernetes provider.
	ProviderKubernetes Provider = "kubernetes"
	// ProviderCustom indicates custom provider implementation.
	ProviderCustom Provider = "custom"
)

// allProviders is a package-level slice containing all valid Provider values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allProviders = []Provider{
	ProviderAWS, ProviderAzure, ProviderGCP, ProviderKubernetes, ProviderCustom,
}

// AllProviders returns a slice of all supported cloud providers in the registry.
func AllProviders() []Provider {
	return allProviders
}

// String returns the provider name as a lowercase string value (e.g., "aws", "azure", "gcp").
func (p Provider) String() string {
	return string(p)
}

// getAllProviderStrings returns all valid provider names as strings.
func getAllProviderStrings() []string {
	providers := AllProviders()
	strs := make([]string, len(providers))
	for i, p := range providers {
		strs[i] = p.String()
	}
	return strs
}

// IsValidProvider checks if the given string represents a valid cloud provider supported by the registry.
// Valid providers include "aws", "azure", "gcp", "kubernetes", and "custom".
func IsValidProvider(p string) bool {
	provider := Provider(p)
	for _, validProvider := range allProviders {
		if provider == validProvider {
			return true
		}
	}
	return false
}

const (
	// MinPluginNameLength defines the minimum required length for plugin names.
	MinPluginNameLength = 2
	// MaxPluginNameLength defines the maximum allowed length for plugin names.
	MaxPluginNameLength = 50
)

// DiscoverySource represents the supported plugin discovery sources.
type DiscoverySource string

const (
	// DiscoverySourceFilesystem indicates filesystem-based plugin discovery.
	DiscoverySourceFilesystem DiscoverySource = "filesystem"
	// DiscoverySourceRegistry indicates registry-based plugin discovery.
	DiscoverySourceRegistry DiscoverySource = "registry"
	// DiscoverySourceURL indicates URL-based plugin discovery.
	DiscoverySourceURL DiscoverySource = "url"
	// DiscoverySourceGit indicates git repository-based plugin discovery.
	DiscoverySourceGit DiscoverySource = "git"
)

// allDiscoverySources is a package-level slice containing all valid DiscoverySource values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allDiscoverySources = []DiscoverySource{
	DiscoverySourceFilesystem, DiscoverySourceRegistry, DiscoverySourceURL, DiscoverySourceGit,
}

// AllDiscoverySources returns all valid discovery sources.
func AllDiscoverySources() []DiscoverySource {
	return allDiscoverySources
}

// String returns the discovery source name as a lowercase string value (e.g., "filesystem", "registry").
func (d DiscoverySource) String() string {
	return string(d)
}

// IsValidDiscoverySource checks if a discovery source is valid.
func IsValidDiscoverySource(source string) bool {
	discoverySource := DiscoverySource(source)
	for _, validSource := range allDiscoverySources {
		if discoverySource == validSource {
			return true
		}
	}
	return false
}

// PluginStatus represents the current status of a plugin.
type PluginStatus string

const (
	// PluginStatusAvailable indicates plugin is available for installation.
	PluginStatusAvailable PluginStatus = "available"
	// PluginStatusInstalled indicates plugin is currently installed.
	PluginStatusInstalled PluginStatus = "installed"
	// PluginStatusActive indicates plugin is installed and running.
	PluginStatusActive PluginStatus = "active"
	// PluginStatusInactive indicates plugin is installed but not running.
	PluginStatusInactive PluginStatus = "inactive"
	// PluginStatusError indicates plugin is in error state.
	PluginStatusError PluginStatus = "error"
	// PluginStatusUpdating indicates plugin is currently being updated.
	PluginStatusUpdating PluginStatus = "updating"
)

// allPluginStatuses is a package-level slice containing all valid PluginStatus values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allPluginStatuses = []PluginStatus{
	PluginStatusAvailable, PluginStatusInstalled, PluginStatusActive,
	PluginStatusInactive, PluginStatusError, PluginStatusUpdating,
}

// AllPluginStatuses returns all valid plugin statuses.
func AllPluginStatuses() []PluginStatus {
	return allPluginStatuses
}

// String returns the plugin status as a lowercase string value (e.g., "available", "installed").
func (p PluginStatus) String() string {
	return string(p)
}

// IsValidPluginStatus checks if a plugin status is valid.
func IsValidPluginStatus(status string) bool {
	pluginStatus := PluginStatus(status)
	for _, validStatus := range allPluginStatuses {
		if pluginStatus == validStatus {
			return true
		}
	}
	return false
}

// SecurityLevel represents plugin security trust levels.
type SecurityLevel string

const (
	// SecurityLevelUntrusted indicates untrusted plugin requiring explicit approval.
	SecurityLevelUntrusted SecurityLevel = "untrusted"
	// SecurityLevelCommunity indicates community-verified plugin.
	SecurityLevelCommunity SecurityLevel = "community"
	// SecurityLevelVerified indicates officially verified plugin.
	SecurityLevelVerified SecurityLevel = "verified"
	// SecurityLevelOfficial indicates official FinFocus plugin.
	SecurityLevelOfficial SecurityLevel = "official"
)

// allSecurityLevels is a package-level slice containing all valid SecurityLevel values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allSecurityLevels = []SecurityLevel{
	SecurityLevelUntrusted, SecurityLevelCommunity, SecurityLevelVerified, SecurityLevelOfficial,
}

// AllSecurityLevels returns all valid security levels.
func AllSecurityLevels() []SecurityLevel {
	return allSecurityLevels
}

// String returns the security level as a lowercase string value (e.g., "untrusted", "verified").
func (s SecurityLevel) String() string {
	return string(s)
}

// IsValidSecurityLevel checks if a security level is valid.
func IsValidSecurityLevel(level string) bool {
	securityLevel := SecurityLevel(level)
	for _, validLevel := range allSecurityLevels {
		if securityLevel == validLevel {
			return true
		}
	}
	return false
}

// InstallationMethod represents different plugin installation methods.
type InstallationMethod string

const (
	// InstallationMethodBinary indicates direct binary installation.
	InstallationMethodBinary InstallationMethod = "binary"
	// InstallationMethodContainer indicates container image deployment.
	InstallationMethodContainer InstallationMethod = "container"
	// InstallationMethodScript indicates script-based installation.
	InstallationMethodScript InstallationMethod = "script"
	// InstallationMethodPackage indicates system package manager installation.
	InstallationMethodPackage InstallationMethod = "package"
)

// allInstallationMethods is a package-level slice containing all valid InstallationMethod values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allInstallationMethods = []InstallationMethod{
	InstallationMethodBinary, InstallationMethodContainer, InstallationMethodScript, InstallationMethodPackage,
}

// AllInstallationMethods returns all valid installation methods.
func AllInstallationMethods() []InstallationMethod {
	return allInstallationMethods
}

// String returns the installation method as a lowercase string value (e.g., "binary", "container").
func (i InstallationMethod) String() string {
	return string(i)
}

// IsValidInstallationMethod checks if an installation method is valid.
func IsValidInstallationMethod(method string) bool {
	installationMethod := InstallationMethod(method)
	for _, validMethod := range allInstallationMethods {
		if installationMethod == validMethod {
			return true
		}
	}
	return false
}

// PluginCapability represents plugin capabilities.
type PluginCapability string

const (
	// PluginCapabilityCostRetrieval indicates cost data retrieval capability.
	PluginCapabilityCostRetrieval PluginCapability = "cost_retrieval"
	// PluginCapabilityCostProjection indicates cost projection capability.
	PluginCapabilityCostProjection PluginCapability = "cost_projection"
	// PluginCapabilityPricingSpecs indicates pricing specification capability.
	PluginCapabilityPricingSpecs PluginCapability = "pricing_specs"
	// PluginCapabilityHistoricalData indicates historical data support.
	PluginCapabilityHistoricalData PluginCapability = "historical_data"
	// PluginCapabilityRealTimeData indicates real-time data support.
	PluginCapabilityRealTimeData PluginCapability = "real_time_data"
	// PluginCapabilityBatchProcessing indicates batch processing support.
	PluginCapabilityBatchProcessing PluginCapability = "batch_processing"
	// PluginCapabilityRateLimiting indicates rate limiting support.
	PluginCapabilityRateLimiting PluginCapability = "rate_limiting"
	// PluginCapabilityCaching indicates caching support.
	PluginCapabilityCaching PluginCapability = "caching"
	// PluginCapabilityEncryption indicates encryption support.
	PluginCapabilityEncryption PluginCapability = "encryption"
	// PluginCapabilityCompression indicates compression support.
	PluginCapabilityCompression PluginCapability = "compression"
	// PluginCapabilityFiltering indicates filtering support.
	PluginCapabilityFiltering PluginCapability = "filtering"
	// PluginCapabilityAggregation indicates aggregation support.
	PluginCapabilityAggregation PluginCapability = "aggregation"
	// PluginCapabilityMultiTenancy indicates multi-tenancy support.
	PluginCapabilityMultiTenancy PluginCapability = "multi_tenancy"
	// PluginCapabilityAuditLogging indicates audit logging support.
	PluginCapabilityAuditLogging PluginCapability = "audit_logging"
)

// allPluginCapabilities is a package-level slice containing all valid PluginCapability values: the
// manifest strings above followed by the protocol capability names (see ManifestCapabilityName).
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allPluginCapabilities = append([]PluginCapability{
	PluginCapabilityCostRetrieval, PluginCapabilityCostProjection, PluginCapabilityPricingSpecs,
	PluginCapabilityHistoricalData, PluginCapabilityRealTimeData, PluginCapabilityBatchProcessing,
	PluginCapabilityRateLimiting, PluginCapabilityCaching, PluginCapabilityEncryption,
	PluginCapabilityCompression, PluginCapabilityFiltering, PluginCapabilityAggregation,
	PluginCapabilityMultiTenancy, PluginCapabilityAuditLogging,
}, protoPluginCapabilities()...)

const protoCapabilityPrefix = "PLUGIN_CAPABILITY_"

// protoPluginCapabilities returns the manifest names of every pbc.PluginCapability value except
// UNSPECIFIED, in enum-number order.
func protoPluginCapabilities() []PluginCapability {
	values := pbc.PluginCapability(0).Descriptor().Values()
	out := make([]PluginCapability, 0, values.Len())
	for i := range values.Len() {
		if name := ManifestCapabilityName(pbc.PluginCapability(values.Get(i).Number())); name != "" {
			out = append(out, PluginCapability(name))
		}
	}
	return out
}

// ManifestCapabilityName returns the manifest capability string for a protocol capability: the
// lowercase value name without the PLUGIN_CAPABILITY_ prefix, so PLUGIN_CAPABILITY_DRY_RUN becomes
// "dry_run". Plugins can build a manifest's specification.capabilities from the same values they
// report through GetPluginInfo. It returns "" for PLUGIN_CAPABILITY_UNSPECIFIED and unknown values.
func ManifestCapabilityName(c pbc.PluginCapability) string {
	if c == pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED {
		return ""
	}
	value := c.Descriptor().Values().ByNumber(c.Number())
	if value == nil {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(string(value.Name()), protoCapabilityPrefix))
}

// AllPluginCapabilities returns all valid plugin capabilities: the 14 manifest capability strings
// followed by the protocol capability names in enum-number order.
func AllPluginCapabilities() []PluginCapability {
	return allPluginCapabilities
}

// String returns the plugin capability as a lowercase string value (e.g., "cost_retrieval", "caching").
func (p PluginCapability) String() string {
	return string(p)
}

// IsValidPluginCapability checks if a plugin capability is valid. It accepts the manifest capability
// strings and the protocol capability names returned by ManifestCapabilityName.
func IsValidPluginCapability(capability string) bool {
	pluginCapability := PluginCapability(capability)
	for _, validCapability := range allPluginCapabilities {
		if pluginCapability == validCapability {
			return true
		}
	}
	return false
}

// SystemPermission represents required system permissions.
type SystemPermission string

const (
	// SystemPermissionNetworkAccess indicates outbound network connection permission.
	SystemPermissionNetworkAccess SystemPermission = "network_access"
	// SystemPermissionFilesystemRead indicates filesystem read permission.
	SystemPermissionFilesystemRead SystemPermission = "filesystem_read"
	// SystemPermissionFilesystemWrite indicates filesystem write permission.
	SystemPermissionFilesystemWrite SystemPermission = "filesystem_write"
	// SystemPermissionEnvironmentRead indicates environment variable read permission.
	SystemPermissionEnvironmentRead SystemPermission = "environment_read"
	// SystemPermissionProcessSpawn indicates process spawn permission.
	SystemPermissionProcessSpawn SystemPermission = "process_spawn"
	// SystemPermissionSystemInfo indicates system information access permission.
	SystemPermissionSystemInfo SystemPermission = "system_info"
	// SystemPermissionTempFiles indicates temporary file creation permission.
	SystemPermissionTempFiles SystemPermission = "temp_files"
	// SystemPermissionConfigRead indicates configuration file read permission.
	SystemPermissionConfigRead SystemPermission = "config_read"
	// SystemPermissionMetricsCollect indicates metrics collection permission.
	SystemPermissionMetricsCollect SystemPermission = "metrics_collect"
)

// allSystemPermissions is a package-level slice containing all valid SystemPermission values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allSystemPermissions = []SystemPermission{
	SystemPermissionNetworkAccess, SystemPermissionFilesystemRead, SystemPermissionFilesystemWrite,
	SystemPermissionEnvironmentRead, SystemPermissionProcessSpawn, SystemPermissionSystemInfo,
	SystemPermissionTempFiles, SystemPermissionConfigRead, SystemPermissionMetricsCollect,
}

// AllSystemPermissions returns all valid system permissions.
func AllSystemPermissions() []SystemPermission {
	return allSystemPermissions
}

// String returns the system permission as a lowercase string value (e.g., "network_access", "filesystem_read").
func (s SystemPermission) String() string {
	return string(s)
}

// IsValidSystemPermission checks if a system permission is valid.
func IsValidSystemPermission(permission string) bool {
	systemPermission := SystemPermission(permission)
	for _, validPermission := range allSystemPermissions {
		if systemPermission == validPermission {
			return true
		}
	}
	return false
}

// AuthMethod represents supported authentication methods.
type AuthMethod string

const (
	// AuthMethodNone indicates no authentication required.
	AuthMethodNone AuthMethod = "none"
	// AuthMethodAPIKey indicates API key authentication.
	AuthMethodAPIKey AuthMethod = "api_key"
	// AuthMethodJWT indicates JWT token authentication.
	AuthMethodJWT AuthMethod = "jwt"
	// AuthMethodOAuth2 indicates OAuth2 authentication.
	AuthMethodOAuth2 AuthMethod = "oauth2"
	// AuthMethodMTLS indicates mutual TLS authentication.
	AuthMethodMTLS AuthMethod = "mtls"
	// AuthMethodBasicAuth indicates basic HTTP authentication.
	AuthMethodBasicAuth AuthMethod = "basic_auth"
)

// allAuthMethods is a package-level slice containing all valid AuthMethod values.
// This is allocated once at package initialization for zero-allocation validation.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allAuthMethods = []AuthMethod{
	AuthMethodNone, AuthMethodAPIKey, AuthMethodJWT, AuthMethodOAuth2, AuthMethodMTLS, AuthMethodBasicAuth,
}

// AllAuthMethods returns all valid authentication methods.
func AllAuthMethods() []AuthMethod {
	return allAuthMethods
}

// String returns the authentication method as a lowercase string value (e.g., "api_key", "jwt").
func (a AuthMethod) String() string {
	return string(a)
}

// IsValidAuthMethod checks if an authentication method is valid.
func IsValidAuthMethod(method string) bool {
	authMethod := AuthMethod(method)
	for _, validMethod := range allAuthMethods {
		if authMethod == validMethod {
			return true
		}
	}
	return false
}

// ValidatePluginName validates a plugin name according to registry rules.
func ValidatePluginName(name string) error {
	if name == "" {
		return errors.New("plugin name cannot be empty")
	}
	if len(name) < MinPluginNameLength {
		return fmt.Errorf("plugin name must be at least %d characters long", MinPluginNameLength)
	}
	if len(name) > MaxPluginNameLength {
		return fmt.Errorf("plugin name must be no more than %d characters long", MaxPluginNameLength)
	}

	// Check pattern: ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$
	if !isValidPluginNameChar(rune(name[0])) || name[0] == '-' {
		return errors.New("plugin name must start with a lowercase letter or digit")
	}
	if len(name) > 1 {
		if !isValidPluginNameChar(rune(name[len(name)-1])) || name[len(name)-1] == '-' {
			return errors.New("plugin name must end with a lowercase letter or digit")
		}
	}

	for i, r := range name {
		if !isValidPluginNameChar(r) && r != '-' {
			return fmt.Errorf("plugin name contains invalid character '%c' at position %d", r, i)
		}
	}

	return nil
}

// isValidPluginNameChar checks if a rune is valid for plugin names (a-z, 0-9).
func isValidPluginNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
