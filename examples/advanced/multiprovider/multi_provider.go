// multi_provider.go demonstrates advanced patterns for building plugins
// that support multiple cloud providers (AWS, Azure, GCP) using the
// mapping package for property extraction.
//
// This example implements the Strategy Pattern described in
// docs/ADVANCED_PATTERNS.md and shows how to:
// - Use provider-specific extractors from the mapping package
// - Implement a multi-provider plugin architecture
// - Handle provider-specific property names cleanly
//
// Reference: sdk/go/pluginsdk/mapping/doc.go
package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk/mapping"
)

// Provider constants.
const (
	providerAWS   = "aws"
	providerAzure = "azure"
	providerGCP   = "gcp"
)

// Property key constants.
const (
	propertyKeyRegion = "region"
)

// Region name constants.
const (
	regionUSEast1      = "us-east-1"
	regionUSWest2      = "us-west-2"
	regionEUWest1      = "eu-west-1"
	regionAPNortheast1 = "ap-northeast-1"
	regionEastUS       = "eastus"
	regionWestEurope   = "westeurope"
	regionEuropeWest1  = "europe-west1"
)

// Canonical region constants.
const (
	canonicalUSEast        = "us-east"
	canonicalUSWest        = "us-west"
	canonicalEuropeWest    = "europe-west"
	canonicalAsiaNortheast = "asia-northeast"
)

// ComputeDetails holds extracted compute resource information.
type ComputeDetails struct {
	Provider       string
	SKU            string
	Region         string
	InstanceFamily string
	IsSpot         bool
	IsPreemptible  bool
}

// extractComputeDetails extracts pricing-relevant details from compute resources
// using provider-specific extractors from the mapping package.
func extractComputeDetails(provider string, props map[string]string) ComputeDetails {
	details := ComputeDetails{Provider: provider}

	switch provider {
	case providerAWS:
		details.SKU = mapping.ExtractAWSSKU(props)
		details.Region = mapping.ExtractAWSRegion(props)
		details.InstanceFamily = extractInstanceFamily(details.SKU)
		details.IsSpot = props["instanceLifecycle"] == "spot"

	case providerAzure:
		details.SKU = mapping.ExtractAzureSKU(props)
		details.Region = mapping.ExtractAzureRegion(props)
		details.InstanceFamily = extractAzureFamily(details.SKU)
		details.IsSpot = props["priority"] == "Spot"

	case providerGCP:
		details.SKU = mapping.ExtractGCPSKU(props)
		details.Region = mapping.ExtractGCPRegion(props)
		details.InstanceFamily = extractGCPFamily(details.SKU)
		details.IsPreemptible = props["scheduling.preemptible"] == "true"

	default:
		// Use generic extractors for custom providers
		details.SKU = mapping.ExtractSKU(props, "sku", "type", "size", "machineType")
		details.Region = mapping.ExtractRegion(props, "region", "location", "zone")
	}

	return details
}

// extractInstanceFamily extracts AWS instance family from instance type.
// Example: t3.micro -> t3, m5.large -> m5.
func extractInstanceFamily(instanceType string) string {
	if idx := strings.IndexByte(instanceType, '.'); idx >= 0 {
		return instanceType[:idx]
	}
	return instanceType
}

// azureStandardPrefix is the prefix for standard Azure VM sizes.
const azureStandardPrefix = "Standard_"

// extractAzureFamily extracts Azure VM family from VM size.
// Example: Standard_D2s_v3 -> D, Standard_E8_v4 -> E.
func extractAzureFamily(vmSize string) string {
	if vmSize == "" {
		return ""
	}

	// Remove Standard_ prefix if present
	familyPart := vmSize
	if strings.HasPrefix(vmSize, azureStandardPrefix) {
		familyPart = strings.TrimPrefix(vmSize, azureStandardPrefix)
	}

	if familyPart == "" {
		return vmSize
	}

	// Find first digit to extract family letter(s)
	for i, c := range familyPart {
		if c >= '0' && c <= '9' {
			return familyPart[:i]
		}
	}

	return familyPart
}

// extractGCPFamily extracts GCP machine family from machine type.
// Example: n1-standard-4 -> n1, e2-medium -> e2.
func extractGCPFamily(machineType string) string {
	if idx := strings.IndexByte(machineType, '-'); idx >= 0 {
		return machineType[:idx]
	}
	return machineType
}

// MultiProviderMatcher demonstrates using ResourceMatcher for multi-provider support.
type MultiProviderMatcher struct {
	matcher *pluginsdk.ResourceMatcher
}

// NewMultiProviderMatcher creates a matcher supporting AWS, Azure, and GCP.
func NewMultiProviderMatcher() *MultiProviderMatcher {
	m := &MultiProviderMatcher{
		matcher: pluginsdk.NewResourceMatcher(),
	}

	// Add supported providers
	m.matcher.AddProvider(providerAWS)
	m.matcher.AddProvider(providerAzure)
	m.matcher.AddProvider(providerGCP)

	// Add supported resource types per provider
	// AWS compute resources
	m.matcher.AddResourceType("aws:ec2/instance:Instance")
	m.matcher.AddResourceType("aws:rds/instance:Instance")
	m.matcher.AddResourceType("aws:lambda/function:Function")

	// Azure compute resources
	m.matcher.AddResourceType("azure:compute/virtualMachine:VirtualMachine")
	m.matcher.AddResourceType("azure:sql/database:Database")
	m.matcher.AddResourceType("azure:web/functionApp:FunctionApp")

	// GCP compute resources
	m.matcher.AddResourceType("gcp:compute/instance:Instance")
	m.matcher.AddResourceType("gcp:sql/databaseInstance:DatabaseInstance")
	m.matcher.AddResourceType("gcp:cloudfunctions/function:Function")

	return m
}

// demonstrateAWSExtraction shows AWS property extraction.
func demonstrateAWSExtraction(logger *slog.Logger) {
	logger.Info("AWS Property Extraction")

	// EC2 instance properties
	ec2Props := map[string]string{
		"instanceType":      "t3.medium",
		"availabilityZone":  "us-east-1a",
		"instanceId":        "i-0abc123def456789",
		"instanceLifecycle": "on-demand",
	}

	details := extractComputeDetails(providerAWS, ec2Props)
	logger.Info("EC2 Instance",
		slog.String("sku", details.SKU),
		slog.String("family", details.InstanceFamily),
		slog.String("region", details.Region),
		slog.Bool("is_spot", details.IsSpot),
	)

	// Spot instance example
	spotProps := map[string]string{
		"instanceType":      "m5.xlarge",
		propertyKeyRegion:   regionUSWest2,
		"instanceLifecycle": "spot",
	}

	spotDetails := extractComputeDetails(providerAWS, spotProps)
	logger.Info("EC2 Spot Instance",
		slog.String("sku", spotDetails.SKU),
		slog.String("family", spotDetails.InstanceFamily),
		slog.String("region", spotDetails.Region),
		slog.Bool("is_spot", spotDetails.IsSpot),
	)

	// RDS instance example
	rdsProps := map[string]string{
		"instanceClass":   "db.t3.micro",
		propertyKeyRegion: regionEUWest1,
	}

	sku := mapping.ExtractAWSSKU(rdsProps)
	region := mapping.ExtractAWSRegion(rdsProps)
	logger.Info("RDS Instance",
		slog.String("sku", sku),
		slog.String("region", region),
	)
}

// demonstrateAzureExtraction shows Azure property extraction.
func demonstrateAzureExtraction(logger *slog.Logger) {
	logger.Info("Azure Property Extraction")

	// Virtual Machine properties
	vmProps := map[string]string{
		"vmSize":   "Standard_D2s_v3",
		"location": regionEastUS,
		"priority": "Regular",
	}

	details := extractComputeDetails(providerAzure, vmProps)
	logger.Info("Virtual Machine",
		slog.String("sku", details.SKU),
		slog.String("family", details.InstanceFamily),
		slog.String("region", details.Region),
		slog.Bool("is_spot", details.IsSpot),
	)

	// Spot VM example
	spotVMProps := map[string]string{
		"vmSize":   "Standard_E8_v4",
		"location": regionWestEurope,
		"priority": "Spot",
	}

	spotDetails := extractComputeDetails(providerAzure, spotVMProps)
	logger.Info("Spot Virtual Machine",
		slog.String("sku", spotDetails.SKU),
		slog.String("family", spotDetails.InstanceFamily),
		slog.String("region", spotDetails.Region),
		slog.Bool("is_spot", spotDetails.IsSpot),
	)

	// Alternative property names
	altProps := map[string]string{
		"sku":             "Standard_B2s",
		propertyKeyRegion: "northeurope",
	}

	sku := mapping.ExtractAzureSKU(altProps)
	region := mapping.ExtractAzureRegion(altProps)
	logger.Info("VM with alternative property names",
		slog.String("sku", sku),
		slog.String("region", region),
	)
}

// demonstrateGCPExtraction shows GCP property extraction with zone validation.
func demonstrateGCPExtraction(logger *slog.Logger) {
	logger.Info("GCP Property Extraction")

	// Compute Engine instance
	instanceProps := map[string]string{
		"machineType":            "n1-standard-4",
		"zone":                   "us-central1-a",
		"scheduling.preemptible": "false",
	}

	details := extractComputeDetails(providerGCP, instanceProps)
	logger.Info("Compute Engine Instance",
		slog.String("sku", details.SKU),
		slog.String("family", details.InstanceFamily),
		slog.String("region", details.Region),
		slog.Bool("is_preemptible", details.IsPreemptible),
	)

	// Preemptible VM example
	preemptibleProps := map[string]string{
		"machineType":            "e2-medium",
		"zone":                   regionEuropeWest1 + "-b",
		"scheduling.preemptible": "true",
	}

	preemptibleDetails := extractComputeDetails(providerGCP, preemptibleProps)
	logger.Info("Preemptible Instance",
		slog.String("sku", preemptibleDetails.SKU),
		slog.String("family", preemptibleDetails.InstanceFamily),
		slog.String("region", preemptibleDetails.Region),
		slog.Bool("is_preemptible", preemptibleDetails.IsPreemptible),
	)

	// Region validation
	logger.Info("GCP Region Validation")
	testRegions := []string{"us-central1", regionEuropeWest1, "invalid-region", "asia-east1"}
	for _, r := range testRegions {
		isValid := mapping.IsValidGCPRegion(r)
		logger.Info("Region check",
			slog.String("region", r),
			slog.Bool("valid", isValid),
		)
	}
}

// demonstrateGenericExtraction shows generic property extraction for custom providers.
func demonstrateGenericExtraction(logger *slog.Logger) {
	logger.Info("Generic/Custom Provider Extraction")

	// Custom resource with non-standard property names
	customProps := map[string]string{
		"machineSize":    "large",
		"deploymentZone": "us-west-2a",
		"customField":    "value",
	}

	// Use generic extractors with custom key priority
	sku := mapping.ExtractSKU(customProps,
		"machineSize",  // Custom key (highest priority)
		"instanceType", // AWS-style fallback
		"vmSize",       // Azure-style fallback
		"machineType",  // GCP-style fallback
	)

	region := mapping.ExtractRegion(customProps,
		"deploymentZone", // Custom key
		"region",         // Standard fallback
		"location",       // Azure fallback
	)

	logger.Info("Custom Resource",
		slog.String("sku", sku),
		slog.String("region", region),
	)

	// Kubernetes resource example
	k8sProps := map[string]string{
		"type":            "Standard_D4s_v3", // Node pool VM size
		propertyKeyRegion: regionEastUS,
		"cluster":         "aks-prod-01",
	}

	details := extractComputeDetails("kubernetes", k8sProps)
	logger.Info("Kubernetes Node Pool",
		slog.String("sku", details.SKU),
		slog.String("region", details.Region),
	)
}

// demonstrateMultiProviderMatcher shows using ResourceMatcher for multi-provider support.
func demonstrateMultiProviderMatcher(logger *slog.Logger) {
	logger.Info("Multi-Provider ResourceMatcher")

	matcher := NewMultiProviderMatcher()

	// Test resources
	testResources := []struct {
		provider     string
		resourceType string
	}{
		{"aws", "aws:ec2/instance:Instance"},
		{"aws", "aws:s3/bucket:Bucket"},
		{"azure", "azure:compute/virtualMachine:VirtualMachine"},
		{"azure", "azure:storage/account:Account"},
		{"gcp", "gcp:compute/instance:Instance"},
		{"gcp", "gcp:bigtable/instance:Instance"},
		{"custom", "custom:resource/type:Type"},
	}

	for _, tr := range testResources {
		desc := pluginsdk.NewResourceDescriptor(tr.provider, tr.resourceType)
		supported := matcher.matcher.Supports(desc)
		logger.Info("Resource support check",
			slog.String("provider", tr.provider),
			slog.String("resource_type", tr.resourceType),
			slog.Bool("supported", supported),
		)
	}
}

// demonstrateRegionNormalization shows cross-provider region mapping.
func demonstrateRegionNormalization(logger *slog.Logger) {
	logger.Info("Cross-Provider Region Normalization")

	// Mapping of provider-specific regions to canonical names
	regionMappings := map[string]map[string]string{
		providerAWS: {
			regionUSEast1:      canonicalUSEast,
			regionUSWest2:      canonicalUSWest,
			regionEUWest1:      canonicalEuropeWest,
			regionAPNortheast1: canonicalAsiaNortheast,
		},
		providerAzure: {
			regionEastUS:     canonicalUSEast,
			"westus2":        canonicalUSWest,
			regionWestEurope: canonicalEuropeWest,
			"japaneast":      canonicalAsiaNortheast,
		},
		providerGCP: {
			"us-east1":        canonicalUSEast,
			"us-west1":        canonicalUSWest,
			regionEuropeWest1: canonicalEuropeWest,
			"asia-northeast1": canonicalAsiaNortheast,
		},
	}

	// Demonstrate normalization
	testCases := []struct {
		provider string
		region   string
	}{
		{providerAWS, regionUSEast1},
		{providerAzure, regionEastUS},
		{providerGCP, "us-east1"},
		{providerAWS, regionEUWest1},
		{providerAzure, regionWestEurope},
		{providerGCP, regionEuropeWest1},
	}

	for _, tc := range testCases {
		canonical := "unknown"
		if providerMap, ok := regionMappings[tc.provider]; ok {
			if c, found := providerMap[tc.region]; found {
				canonical = c
			}
		}
		logger.Info("Region normalization",
			slog.String("provider", tc.provider),
			slog.String("region", tc.region),
			slog.String("canonical", canonical),
		)
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Check for quiet mode (for testing)
	quiet := len(os.Args) > 1 && os.Args[1] == "--quiet"
	if quiet {
		logger.Info("Multi-provider mapping examples executed successfully")
		return
	}

	demonstrateAWSExtraction(logger)
	demonstrateAzureExtraction(logger)
	demonstrateGCPExtraction(logger)
	demonstrateGenericExtraction(logger)
	demonstrateMultiProviderMatcher(logger)
	demonstrateRegionNormalization(logger)

	logger.Info("Key Takeaways",
		slog.String("point_1", "Use provider-specific extractors for accurate property mapping"),
		slog.String("point_2", "The mapping package handles property name variations automatically"),
		slog.String("point_3", "Generic extractors allow custom key priority for non-standard resources"),
		slog.String("point_4", "ResourceMatcher simplifies multi-provider support checks"),
		slog.String("point_5", "Region normalization enables cross-provider cost comparison"),
	)
}
