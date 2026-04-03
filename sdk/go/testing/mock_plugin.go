// Package testing provides test utilities for FinFocus plugin development.
// It includes mock plugin implementations, test harnesses, and conformance testing
// utilities for validating plugin behavior against the CostSource gRPC service spec.
package testing

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/internal/grpcconv"
	"github.com/rshade/finfocus-spec/sdk/go/internal/utilization"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	computeEngineResourceType  = "compute_engine"
	ec2ResourceType            = "ec2"
	blobStorageResourceType    = "blob_storage"
	cloudFunctionsResourceType = "cloud_functions"
	namespaceResourceType      = "namespace"
	cloudStorageResourceType   = "cloud_storage"
	lambdaResourceType         = "lambda"

	// Time and performance constants.
	defaultDataPoints    = 24   // 24 hours of hourly data
	defaultBaseRate      = 0.05 // Default hourly rate
	nameDelayMs          = 100  // Name RPC delay in milliseconds
	supportsDelayMs      = 200  // Supports RPC delay in milliseconds
	actualCostDelayMs    = 500  // ActualCost RPC delay in milliseconds
	projectedCostDelayMs = 300  // ProjectedCost RPC delay in milliseconds
	pricingSpecDelayMs   = 250  // PricingSpec RPC delay in milliseconds
	estimateCostDelayMs  = 150  // EstimateCost RPC delay in milliseconds
	costVariationMod     = 10   // Modulo for cost variation
	daysPerMonth         = 30   // Days per month for cost calculation

	// Cost variation constants.
	costVariationBase  = 0.8 // Base cost variation multiplier (80%)
	costVariationRange = 0.4 // Cost variation range (40% additional)

	// Rate multipliers for different resource types.
	computeRateMultiplier    = 2.0   // Compute resources (EC2, VM, Compute Engine)
	storageRateMultiplier    = 0.1   // Storage resources (S3, Blob, Cloud Storage)
	serverlessRateMultiplier = 0.001 // Serverless (Lambda, Cloud Functions)
	namespaceRateMultiplier  = 1.5   // Kubernetes namespace
	databaseRateMultiplier   = 3.0   // Database resources (SQL Database)

	// Recommendation generation constants.
	confidenceVariations       = 4     // Number of confidence score variations (0.70, 0.775, 0.85, 0.925)
	confidenceBase             = 0.7   // Base confidence score
	confidenceStep             = 0.075 // Confidence increment per variation
	savingsBase                = 50.0  // Base savings amount
	defaultRecommendationCount = 12    // 12 samples to cover all categories and action types
	defaultSpotRiskScore       = 0.3   // Default spot interruption risk score
	defaultPreemptibleRisk     = 0.35  // Default preemptible instance risk score
	savingsIncrement           = 25.0  // Savings increment per recommendation

	// Mock impact metric base values at 100% utilization.
	// These are arbitrary values for testing purposes only.
	mockCarbonPerHour    = 100.0 // gCO2e per hour at 100% utilization
	mockEnergyPerHour    = 1.0   // kWh per hour at 100% utilization
	mockWaterPerHour     = 5.0   // L per hour at 100% utilization
	defaultMockBatchSize = 100
)

// MockPlugin provides a configurable mock implementation of CostSourceServiceServer.
//
// # Thread Safety
//
// All configuration fields must be set before calling Start() on the associated TestHarness.
// Concurrent modification of fields during RPC handling will result in data races.
// Use separate MockPlugin instances for concurrent tests.
//
// Fields that must be set before Start() include: ShouldError* flags, *Delay durations,
// FallbackHint, ExpiresAtDuration, ProjectedCostExpiresAtDuration, EstimateCostExpiresAtDuration,
// MockBudgets, DryRun* fields, PricingCategory/SpotRiskScore fields, and RecommendationsConfig.
//
// The recommended pattern is:
//
//	plugin := NewMockPlugin()
//	plugin.ShouldErrorOnName = true  // Configure before Start()
//	harness := NewTestHarness(plugin)
//	harness.Start(t)
//	defer harness.Stop()
//	// Now safe to make concurrent RPC calls
type MockPlugin struct {
	pbc.UnimplementedCostSourceServiceServer

	// Configuration
	PluginName         string
	SupportedProviders []string
	SupportedResources map[string][]string // provider -> resource types

	// Behavior configuration
	ShouldErrorOnName          bool
	ShouldErrorOnSupports      bool
	ShouldErrorOnActualCost    bool
	ShouldErrorOnProjectedCost bool
	ShouldErrorOnPricingSpec   bool
	ShouldErrorOnEstimateCost  bool
	ShouldErrorOnBatchCost     bool

	// Response delays for testing timeouts
	NameDelay          time.Duration
	SupportsDelay      time.Duration
	ActualCostDelay    time.Duration
	ProjectedCostDelay time.Duration
	PricingSpecDelay   time.Duration
	EstimateCostDelay  time.Duration
	BatchCostDelay     time.Duration

	// Batch behavior configuration
	// UnsupportedBatchResourceTypes configures per-resource errors for batch processing.
	// Keys are resource_type values that should return ResourceError with
	// resource_type_unsupported=true.
	UnsupportedBatchResourceTypes map[string]bool

	// Data generation configuration
	actualCostDataPoints atomic.Int64
	BaseHourlyRate       float64
	Currency             string

	// GreenOps configuration
	SupportedMetrics []pbc.MetricKind
	OmitMetrics      []pbc.MetricKind

	// Recommendations configuration
	RecommendationsConfig RecommendationsConfig

	// Budgets configuration
	ShouldErrorOnBudgets bool
	MockBudgets          []*pbc.Budget

	// ExpiresAtDuration configures the expires_at hint for generated cost results.
	// When non-zero, each ActualCostResult will have expires_at set to
	// time.Now().Add(ExpiresAtDuration).
	// When zero (default), expires_at is not set (backward compatible).
	// Negative values produce past timestamps (immediately-stale semantics),
	// which is useful for testing cache expiration logic.
	ExpiresAtDuration time.Duration

	// ProjectedCostExpiresAtDuration configures the expires_at hint for projected
	// cost responses. Same semantics as ExpiresAtDuration: zero means unset,
	// positive means future expiration, negative means immediately stale.
	ProjectedCostExpiresAtDuration time.Duration

	// EstimateCostExpiresAtDuration configures the expires_at hint for estimate
	// cost responses. Same semantics as ExpiresAtDuration: zero means unset,
	// positive means future expiration, negative means immediately stale.
	EstimateCostExpiresAtDuration time.Duration

	// FallbackHint configuration for GetActualCost responses.
	// Thread Safety: This field must be set before the plugin begins serving
	// requests. Use SetFallbackHint() for configuration, which documents the
	// thread safety constraints.
	FallbackHint pbc.FallbackHint

	// GetPluginInfo configuration
	PluginVersion string            // Plugin version (e.g., "v1.0.0")
	SpecVersion   string            // Spec version this plugin implements (e.g., "v0.4.11")
	Metadata      map[string]string // Optional key-value metadata

	// Behavior configuration for GetPluginInfo
	ShouldErrorOnGetPluginInfo bool
	GetPluginInfoDelay         time.Duration

	// DryRun configuration
	DryRunFieldMappings      []*pbc.FieldMapping // Custom field mappings to return
	DryRunConfigValid        bool                // Whether configuration is valid
	DryRunConfigErrors       []string            // Configuration errors if invalid
	ShouldErrorOnDryRun      bool                // Whether DryRun should return an error
	DryRunDelay              time.Duration       // Delay before responding
	UnsupportedResourceTypes []string            // Resource types that return resource_type_supported=false

	// Pricing Tier configuration
	//
	// Thread Safety: These fields must be set before the plugin begins serving
	// requests. Modification during request handling is not thread-safe.
	DefaultPricingCategory           pbc.FocusPricingCategory            // Default pricing category for all responses
	DefaultSpotInterruptionRiskScore float64                             // Default spot risk score (0.0-1.0)
	PricingCategoryByResourceType    map[string]pbc.FocusPricingCategory // Per-resource-type overrides
	SpotRiskScoreByResourceType      map[string]float64                  // Per-resource-type risk score overrides
}

// NewMockPlugin creates a preconfigured MockPlugin populated with sensible defaults for testing.
// The default configuration includes supported providers and resource types, a base hourly rate and currency, initialized batch settings, sample recommendations, default plugin/spec versions, valid dry-run configuration, and default pricing category and spot interruption risk overrides.
func NewMockPlugin() *MockPlugin {
	p := &MockPlugin{
		PluginName:         "mock-test-plugin",
		SupportedProviders: []string{"aws", "azure", "gcp", "kubernetes"},
		SupportedResources: map[string][]string{
			"aws":        {ec2ResourceType, "s3", lambdaResourceType, "rds"},
			"azure":      {"vm", blobStorageResourceType, "sql_database", "compute"},
			"gcp":        {computeEngineResourceType, cloudStorageResourceType, cloudFunctionsResourceType, "compute"},
			"kubernetes": {namespaceResourceType, "pod", "service"},
		},
		BaseHourlyRate:                defaultBaseRate,
		Currency:                      "USD",
		UnsupportedBatchResourceTypes: make(map[string]bool),
		// Pre-populate with sample recommendations for filtering tests
		RecommendationsConfig: RecommendationsConfig{
			Recommendations: GenerateSampleRecommendations(defaultRecommendationCount),
		},
		// GetPluginInfo defaults
		PluginVersion: "v1.0.0",
		SpecVersion:   "v0.4.11",
		// DryRun defaults - configuration valid by default
		DryRunConfigValid: true,
		// Pricing Tier defaults
		DefaultPricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		DefaultSpotInterruptionRiskScore: 0.0,
		PricingCategoryByResourceType: map[string]pbc.FocusPricingCategory{
			"spot":        pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			"preemptible": pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
		},
		SpotRiskScoreByResourceType: map[string]float64{
			"spot":        defaultSpotRiskScore,
			"preemptible": defaultPreemptibleRisk,
		},
	}
	p.actualCostDataPoints.Store(defaultDataPoints)
	return p
}

// ConfigurableErrorMockPlugin creates a mock plugin that can be configured to return errors.
func ConfigurableErrorMockPlugin() *MockPlugin {
	plugin := NewMockPlugin()
	plugin.PluginName = "error-test-plugin"
	return plugin
}

// SlowMockPlugin creates a mock plugin with artificial delays for timeout testing.
func SlowMockPlugin() *MockPlugin {
	plugin := NewMockPlugin()
	plugin.PluginName = "slow-test-plugin"
	plugin.NameDelay = nameDelayMs * time.Millisecond
	plugin.SupportsDelay = supportsDelayMs * time.Millisecond
	plugin.ActualCostDelay = actualCostDelayMs * time.Millisecond
	plugin.ProjectedCostDelay = projectedCostDelayMs * time.Millisecond
	plugin.PricingSpecDelay = pricingSpecDelayMs * time.Millisecond
	plugin.EstimateCostDelay = estimateCostDelayMs * time.Millisecond
	return plugin
}

// SetActualCostDataPoints sets the number of data points to generate for GetActualCost responses.
//
// Thread Safety: This method uses atomic operations and is safe for concurrent use.
func (m *MockPlugin) SetActualCostDataPoints(n int) {
	if n < 0 {
		n = 0
	}
	m.actualCostDataPoints.Store(int64(n))
}

// SetFallbackHint sets the fallback hint to be returned in GetActualCost responses.
//
// Thread Safety: This method is NOT safe for concurrent use. All calls to
// SetFallbackHint must complete before the plugin begins serving requests.
// Typical usage is to configure the mock during test setup, before calling
// harness.Start() or benchmark.ResetTimer().
func (m *MockPlugin) SetFallbackHint(hint pbc.FallbackHint) {
	m.FallbackHint = hint
}

// SetPricingCategory sets the default pricing category for all responses.
//
// Thread Safety: This method is NOT safe for concurrent use. All calls to
// SetPricingCategory must complete before the plugin begins serving requests.
//
// Example:
//
//	plugin := NewMockPlugin()
//	plugin.SetPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC)
func (m *MockPlugin) SetPricingCategory(category pbc.FocusPricingCategory) {
	m.DefaultPricingCategory = category
}

// SetSpotRiskScore sets the default spot interruption risk score for all responses.
// The score must be between 0.0 and 1.0 and must not be NaN or Inf.
//
// Thread Safety: This method is NOT safe for concurrent use. All calls to
// SetSpotRiskScore must complete before the plugin begins serving requests.
//
// Example:
//
//	plugin := NewMockPlugin()
//	plugin.SetSpotRiskScore(0.8)  // 80% interruption risk
//
// Panics if score is NaN, Inf, or not in the range [0.0, 1.0].
func (m *MockPlugin) SetSpotRiskScore(score float64) {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		panic(fmt.Sprintf("invalid spot risk score: cannot be NaN or Inf, got %f", score))
	}
	if score < 0.0 || score > 1.0 {
		panic(fmt.Sprintf("invalid spot risk score: must be between 0.0 and 1.0, got %f", score))
	}
	m.DefaultSpotInterruptionRiskScore = score
}

// SetPricingCategoryForResourceType sets a resource-type-specific pricing category override.
//
// Thread Safety: This method is NOT safe for concurrent use. All calls to
// SetPricingCategoryForResourceType must complete before the plugin begins serving requests.
//
// Example:
//
//	plugin := NewMockPlugin()
//	plugin.SetPricingCategoryForResourceType("spot", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC)
//	plugin.SetPricingCategoryForResourceType("reserved", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED)
func (m *MockPlugin) SetPricingCategoryForResourceType(resourceType string, category pbc.FocusPricingCategory) {
	if m.PricingCategoryByResourceType == nil {
		m.PricingCategoryByResourceType = make(map[string]pbc.FocusPricingCategory)
	}
	m.PricingCategoryByResourceType[resourceType] = category
}

// SetSpotRiskScoreForResourceType sets a resource-type-specific spot risk score override.
// The score must be between 0.0 and 1.0 and must not be NaN or Inf.
//
// Thread Safety: This method is NOT safe for concurrent use. All calls to
// SetSpotRiskScoreForResourceType must complete before the plugin begins serving requests.
//
// Example:
//
//	plugin := NewMockPlugin()
//	plugin.SetSpotRiskScoreForResourceType("spot", 0.8)  // 80% interruption risk for spot instances
//	plugin.SetSpotRiskScoreForResourceType("preemptible", 0.6)  // 60% for preemptible instances
//
// Panics if score is NaN, Inf, or not in the range [0.0, 1.0].
func (m *MockPlugin) SetSpotRiskScoreForResourceType(resourceType string, score float64) {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		panic(fmt.Sprintf("invalid spot risk score: cannot be NaN or Inf, got %f", score))
	}
	if score < 0.0 || score > 1.0 {
		panic(fmt.Sprintf("invalid spot risk score: must be between 0.0 and 1.0, got %f", score))
	}
	if m.SpotRiskScoreByResourceType == nil {
		m.SpotRiskScoreByResourceType = make(map[string]float64)
	}
	m.SpotRiskScoreByResourceType[resourceType] = score
}

// Name returns the plugin name.
func (m *MockPlugin) Name(_ context.Context, _ *pbc.NameRequest) (*pbc.NameResponse, error) {
	if m.NameDelay > 0 {
		time.Sleep(m.NameDelay)
	}

	if m.ShouldErrorOnName {
		return nil, status.Error(codes.Internal, "mock error: name operation failed")
	}

	return &pbc.NameResponse{
		Name: m.PluginName,
	}, nil
}

// GetPluginInfo returns metadata about the plugin including name, version, and spec version.
// This RPC enables compatibility verification by reporting which spec version the plugin implements.
func (m *MockPlugin) GetPluginInfo(
	_ context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	if m.GetPluginInfoDelay > 0 {
		time.Sleep(m.GetPluginInfoDelay)
	}

	if m.ShouldErrorOnGetPluginInfo {
		return nil, status.Error(codes.Internal, "mock error: get plugin info operation failed")
	}

	return &pbc.GetPluginInfoResponse{
		Name:        m.PluginName,
		Version:     m.PluginVersion,
		SpecVersion: m.SpecVersion,
		Providers:   m.SupportedProviders,
		Metadata:    m.Metadata,
	}, nil
}

// DryRun returns field mapping information for introspection without actual cost retrieval.
// This implements the dry-run capability allowing hosts to query plugin field support.
func (m *MockPlugin) DryRun(
	_ context.Context,
	req *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
	if m.DryRunDelay > 0 {
		time.Sleep(m.DryRunDelay)
	}

	if m.ShouldErrorOnDryRun {
		return nil, status.Error(codes.Internal, "mock error: dry run operation failed")
	}

	resource := req.GetResource()
	if resource == nil {
		return &pbc.DryRunResponse{
			ResourceTypeSupported: false,
			ConfigurationValid:    false,
			ConfigurationErrors:   []string{"resource descriptor is required"},
		}, nil
	}

	// Check if resource type is in the unsupported list
	resourceType := resource.GetResourceType()
	for _, unsupported := range m.UnsupportedResourceTypes {
		if resourceType == unsupported {
			return &pbc.DryRunResponse{
				ResourceTypeSupported: false,
				ConfigurationValid:    m.DryRunConfigValid,
				ConfigurationErrors:   m.DryRunConfigErrors,
			}, nil
		}
	}

	// Check if provider/resource type is supported
	provider := resource.GetProvider()
	providerSupported := false
	for _, supportedProvider := range m.SupportedProviders {
		if provider == supportedProvider {
			providerSupported = true
			break
		}
	}

	if !providerSupported {
		return &pbc.DryRunResponse{
			ResourceTypeSupported: false,
			ConfigurationValid:    m.DryRunConfigValid,
			ConfigurationErrors:   m.DryRunConfigErrors,
		}, nil
	}

	// Check if resource type is supported for this provider
	supportedResources, exists := m.SupportedResources[provider]
	if !exists {
		return &pbc.DryRunResponse{
			ResourceTypeSupported: false,
			ConfigurationValid:    m.DryRunConfigValid,
			ConfigurationErrors:   m.DryRunConfigErrors,
		}, nil
	}

	resourceSupported := false
	for _, sr := range supportedResources {
		if resourceType == sr {
			resourceSupported = true
			break
		}
	}

	if !resourceSupported {
		return &pbc.DryRunResponse{
			ResourceTypeSupported: false,
			ConfigurationValid:    m.DryRunConfigValid,
			ConfigurationErrors:   m.DryRunConfigErrors,
		}, nil
	}

	// Return configured field mappings or generate default ones
	fieldMappings := m.DryRunFieldMappings
	if len(fieldMappings) == 0 {
		// Generate default field mappings for all FOCUS fields
		fieldMappings = generateDefaultFieldMappings()
	}

	return &pbc.DryRunResponse{
		FieldMappings:         fieldMappings,
		ResourceTypeSupported: true,
		ConfigurationValid:    m.DryRunConfigValid,
		ConfigurationErrors:   m.DryRunConfigErrors,
	}, nil
}

// BatchCost returns per-resource batch cost results with partial failure semantics.
func (m *MockPlugin) BatchCost(
	ctx context.Context,
	req *pbc.BatchCostRequest,
) (*pbc.BatchCostResponse, error) {
	if m.BatchCostDelay > 0 {
		time.Sleep(m.BatchCostDelay)
	}

	if m.ShouldErrorOnBatchCost {
		return nil, status.Error(codes.Internal, "mock error: batch cost operation failed")
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "batch request is required")
	}

	resources := req.GetResources()
	if len(resources) == 0 {
		return &pbc.BatchCostResponse{
			Results:      []*pbc.ResourceCostResult{},
			MaxBatchSize: defaultMockBatchSize,
		}, nil
	}

	if int64(len(resources)) > int64(defaultMockBatchSize) {
		return nil, status.Errorf(codes.InvalidArgument,
			"batch size %d exceeds max_batch_size %d", len(resources), defaultMockBatchSize)
	}

	queryType := req.GetQueryType()
	if queryType == pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED {
		queryType = pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE
	}

	if queryType == pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL {
		start := req.GetStart()
		end := req.GetEnd()
		if start == nil || end == nil {
			return nil, status.Error(codes.InvalidArgument, "start and end are required for ACTUAL query type")
		}
		if !start.AsTime().Before(end.AsTime()) {
			return nil, status.Error(codes.InvalidArgument, "start must be before end for ACTUAL query type")
		}
	}

	results := make([]*pbc.ResourceCostResult, len(resources))
	for index, resource := range resources {
		results[index] = m.batchSingleResource(ctx, req, resource, queryType)
	}

	return &pbc.BatchCostResponse{
		Results:      results,
		MaxBatchSize: defaultMockBatchSize,
	}, nil
}

// batchSingleResource processes a single resource for BatchCost and returns a ResourceCostResult.
// It handles nil resource validation, unsupported resource types, dry-run mode, and query type routing.
func (m *MockPlugin) batchSingleResource(
	ctx context.Context,
	req *pbc.BatchCostRequest,
	resource *pbc.ResourceDescriptor,
	queryType pbc.CostQueryType,
) *pbc.ResourceCostResult {
	if err := ctx.Err(); err != nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(err),
			},
		}
	}

	if resource == nil {
		return &pbc.ResourceCostResult{
			Resource: nil,
			Result: &pbc.ResourceCostResult_Error{
				Error: &pbc.ResourceError{
					Code:    grpcconv.CodeToInt32(codes.InvalidArgument),
					Message: "resource descriptor is required",
				},
			},
		}
	}

	if m.UnsupportedBatchResourceTypes[resource.GetResourceType()] {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: &pbc.ResourceError{
					Code: grpcconv.CodeToInt32(codes.Unimplemented),
					Message: fmt.Sprintf(
						"resource type %q is not supported",
						resource.GetResourceType(),
					),
					ResourceTypeUnsupported: true,
				},
			},
		}
	}

	if req.GetDryRun() {
		dryRunResp, err := m.DryRun(ctx, &pbc.DryRunRequest{Resource: descriptorClone(resource)})
		if err != nil {
			return &pbc.ResourceCostResult{
				Resource: resource,
				Result: &pbc.ResourceCostResult_Error{
					Error: batchResourceErrorFromErr(err),
				},
			}
		}

		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_CostData{
				CostData: &pbc.CostData{
					Data: &pbc.CostData_DryRunResult{
						DryRunResult: dryRunResp,
					},
				},
			},
		}
	}

	switch queryType {
	case pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL:
		return m.batchActualCost(ctx, req, resource)
	case pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED:
		return m.batchProjectedCost(ctx, resource)
	case pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED:
		// NormalizeCostQueryType maps UNSPECIFIED -> ESTIMATE before this function is called;
		// UNSPECIFIED is listed here for exhaustive switch compliance.
		return m.batchEstimateCost(ctx, resource)
	default:
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: &pbc.ResourceError{
					Code:    grpcconv.CodeToInt32(codes.InvalidArgument),
					Message: "invalid query type",
				},
			},
		}
	}
}

// batchActualCost delegates to GetActualCost and wraps the response as a ResourceCostResult.
func (m *MockPlugin) batchActualCost(
	ctx context.Context,
	req *pbc.BatchCostRequest,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	actualResp, err := m.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: defaultBatchResourceID(resource),
		Start:      req.GetStart(),
		End:        req.GetEnd(),
		Tags:       copyStringMap(resource.GetTags()),
		Arn:        resource.GetArn(),
	})
	if err != nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(err),
			},
		}
	}
	if actualResp == nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(
					errors.New("GetActualCost returned nil response")),
			},
		}
	}

	return &pbc.ResourceCostResult{
		Resource: resource,
		Result: &pbc.ResourceCostResult_CostData{
			CostData: &pbc.CostData{
				Data: &pbc.CostData_ActualCost{
					ActualCost: &pbc.ActualCostData{
						Results:       actualResp.GetResults(),
						FallbackHint:  actualResp.GetFallbackHint(),
						NextPageToken: actualResp.GetNextPageToken(),
						TotalCount:    actualResp.GetTotalCount(),
					},
				},
			},
		},
	}
}

// batchProjectedCost delegates to GetProjectedCost and wraps the response as a ResourceCostResult.
func (m *MockPlugin) batchProjectedCost(
	ctx context.Context,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	projectedResp, err := m.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: descriptorClone(resource),
	})
	if err != nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(err),
			},
		}
	}
	if projectedResp == nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(
					errors.New("GetProjectedCost returned nil response")),
			},
		}
	}

	return &pbc.ResourceCostResult{
		Resource: resource,
		Result: &pbc.ResourceCostResult_CostData{
			CostData: &pbc.CostData{
				Data: &pbc.CostData_ProjectedCost{
					ProjectedCost: projectedResp,
				},
			},
		},
	}
}

// batchEstimateCost delegates to EstimateCost and wraps the response as a ResourceCostResult.
func (m *MockPlugin) batchEstimateCost(
	ctx context.Context,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	estimateResp, err := m.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: batchEstimateResourceType(resource),
	})
	if err != nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(err),
			},
		}
	}
	if estimateResp == nil {
		return &pbc.ResourceCostResult{
			Resource: resource,
			Result: &pbc.ResourceCostResult_Error{
				Error: batchResourceErrorFromErr(
					errors.New("EstimateCost returned nil response")),
			},
		}
	}

	return &pbc.ResourceCostResult{
		Resource: resource,
		Result: &pbc.ResourceCostResult_CostData{
			CostData: &pbc.CostData{
				Data: &pbc.CostData_Estimate{
					Estimate: estimateResp,
				},
			},
		},
	}
}

// batchEstimateResourceType builds a canonical estimate resource type string for a ResourceDescriptor
// in the form "provider:resourceType/resource:Resource". If the provider is empty it uses "custom".
// If the resource type is empty it returns an empty string.
func batchEstimateResourceType(resource *pbc.ResourceDescriptor) string {
	provider := strings.ToLower(resource.GetProvider())
	if provider == "" {
		provider = "custom"
	}

	resourceType := resource.GetResourceType()
	if resourceType == "" {
		return ""
	}

	return fmt.Sprintf("%s:%s/%s:Resource", provider, resourceType, resourceType)
}

// defaultBatchResourceID returns the first available identifier for the given resource descriptor.
// It prefers Id, then Arn, then ResourceType, and falls back to "unknown-resource" when none are present.
func defaultBatchResourceID(resource *pbc.ResourceDescriptor) string {
	if resource.GetId() != "" {
		return resource.GetId()
	}
	if resource.GetArn() != "" {
		return resource.GetArn()
	}
	if resource.GetResourceType() != "" {
		return resource.GetResourceType()
	}
	return "unknown-resource"
}

// batchResourceErrorFromErr converts an error into a pbc.ResourceError suitable for batch responses.
// Context errors are mapped via status.FromContextError to preserve Canceled/DeadlineExceeded codes.
// gRPC status errors use their code and message; other errors are reported with Internal code.
func batchResourceErrorFromErr(err error) *pbc.ResourceError {
	if err == nil {
		return &pbc.ResourceError{
			Code:    grpcconv.CodeToInt32(codes.Unknown),
			Message: "unknown error",
		}
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		ctxStatus := status.FromContextError(err)
		return &pbc.ResourceError{
			Code:    grpcconv.CodeToInt32(ctxStatus.Code()),
			Message: ctxStatus.Message(),
		}
	}

	st, ok := status.FromError(err)
	if !ok {
		return &pbc.ResourceError{
			Code:    grpcconv.CodeToInt32(codes.Internal),
			Message: err.Error(),
		}
	}

	return &pbc.ResourceError{
		Code:    grpcconv.CodeToInt32(st.Code()),
		Message: st.Message(),
	}
}

// descriptorClone returns a deep copy of the given ResourceDescriptor.
// If resource is nil, descriptorClone returns nil.
func descriptorClone(resource *pbc.ResourceDescriptor) *pbc.ResourceDescriptor {
	if resource == nil {
		return nil
	}
	// proto.Clone preserves the concrete type, so this assertion is safe.
	//nolint:errcheck // Type assertion guaranteed by proto.Clone contract.
	return proto.Clone(resource).(*pbc.ResourceDescriptor)
}

// copyStringMap returns a shallow copy of the input map, or nil if the input is empty.
// This prevents callers from mutating the original map through the returned reference.
func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// generateDefaultFieldMappings creates default field mappings for all FOCUS fields.
// generateDefaultFieldMappings returns a slice of FieldMapping entries for the canonical FOCUS fields,
// each marked as FIELD_SUPPORT_STATUS_SUPPORTED. This is used when no custom field mappings are configured.
func generateDefaultFieldMappings() []*pbc.FieldMapping {
	// FOCUS 1.2/1.3 field names matching FocusCostRecord
	fieldNames := []string{
		// Identity & Hierarchy
		"provider_name", "billing_account_id", "billing_account_name",
		"sub_account_id", "sub_account_name", "billing_account_type", "sub_account_type",
		// Billing Period
		"billing_period_start", "billing_period_end", "billing_currency",
		// Charge Period
		"charge_period_start", "charge_period_end",
		// Charge Details
		"charge_category", "charge_class", "charge_description", "charge_frequency",
		// Pricing Details
		"pricing_category", "pricing_quantity", "pricing_unit", "list_unit_price",
		"pricing_currency", "pricing_currency_contracted_unit_price",
		"pricing_currency_effective_cost", "pricing_currency_list_unit_price",
		// Service & Product
		"service_category", "service_name", "service_subcategory", "publisher",
		// Resource Details
		"resource_id", "resource_name", "resource_type",
		// SKU Details
		"sku_id", "sku_price_id", "sku_meter", "sku_price_details",
		// Location
		"region_id", "region_name", "availability_zone",
		// Financial Amounts
		"billed_cost", "list_cost", "effective_cost", "contracted_cost", "contracted_unit_price",
		// Consumption/Usage
		"consumed_quantity", "consumed_unit",
		// Commitment Discounts
		"commitment_discount_category", "commitment_discount_id", "commitment_discount_name",
		"commitment_discount_quantity", "commitment_discount_status", "commitment_discount_type",
		"commitment_discount_unit",
		// Capacity Reservation
		"capacity_reservation_id", "capacity_reservation_status",
		// Invoice Details
		"invoice_id", "invoice_issuer",
		// Metadata & Extension
		"tags", "extended_columns",
		// FOCUS 1.3 Provider Identification
		"service_provider_name", "host_provider_name",
		// FOCUS 1.3 Split Cost Allocation
		"allocated_method_id", "allocated_method_details", "allocated_resource_id",
		"allocated_resource_name", "allocated_tags",
		// FOCUS 1.3 Contract Commitment Link
		"contract_applied",
	}

	mappings := make([]*pbc.FieldMapping, len(fieldNames))
	for i, name := range fieldNames {
		mappings[i] = &pbc.FieldMapping{
			FieldName:     name,
			SupportStatus: pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED,
		}
	}
	return mappings
}

// Supports checks if a resource type is supported by this mock plugin.
func (m *MockPlugin) Supports(_ context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	if m.SupportsDelay > 0 {
		time.Sleep(m.SupportsDelay)
	}

	if m.ShouldErrorOnSupports {
		return nil, status.Error(codes.InvalidArgument, "mock error: supports operation failed")
	}

	resource := req.GetResource()
	if resource == nil {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    "resource descriptor is required",
		}, nil
	}

	provider := resource.GetProvider()
	resourceType := resource.GetResourceType()

	// Check if provider is supported
	providerSupported := false
	for _, supportedProvider := range m.SupportedProviders {
		if provider == supportedProvider {
			providerSupported = true
			break
		}
	}

	if !providerSupported {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("provider %s is not supported", provider),
		}, nil
	}

	// Check if resource type is supported for this provider
	supportedResources, exists := m.SupportedResources[provider]
	if !exists {
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("no resource types configured for provider %s", provider),
		}, nil
	}

	for _, supportedResource := range supportedResources {
		if resourceType == supportedResource {
			return &pbc.SupportsResponse{
				Supported:        true,
				Reason:           "",
				SupportedMetrics: m.SupportedMetrics,
				Capabilities: map[string]bool{
					"dry_run": true, // T027: MockPlugin supports DryRun capability
				},
			}, nil
		}
	}

	return &pbc.SupportsResponse{
		Supported: false,
		Reason:    fmt.Sprintf("resource type %s is not supported for provider %s", resourceType, provider),
	}, nil
}

// GetActualCost returns mock historical cost data.
func (m *MockPlugin) GetActualCost(
	_ context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if m.ActualCostDelay > 0 {
		time.Sleep(m.ActualCostDelay)
	}

	if m.ShouldErrorOnActualCost {
		return nil, status.Error(codes.NotFound, "mock error: actual cost data not available")
	}

	// T045: Check dry_run flag - return DryRunResponse if true
	// Note: GetActualCostRequest uses resource_id instead of ResourceDescriptor.
	// For dry-run mode, we return default field mappings since we can't determine
	// resource type from resource_id (format is plugin-specific).
	if req.GetDryRun() {
		fieldMappings := m.DryRunFieldMappings
		if len(fieldMappings) == 0 {
			fieldMappings = generateDefaultFieldMappings()
		}

		return &pbc.GetActualCostResponse{
			DryRunResult: &pbc.DryRunResponse{
				FieldMappings:         fieldMappings,
				ResourceTypeSupported: true,
				ConfigurationValid:    m.DryRunConfigValid,
				ConfigurationErrors:   m.DryRunConfigErrors,
			},
		}, nil
	}

	start := req.GetStart()
	end := req.GetEnd()

	if start == nil || end == nil {
		return nil, status.Error(codes.InvalidArgument, "start and end timestamps are required")
	}

	startTime := start.AsTime()
	endTime := end.AsTime()

	if endTime.Before(startTime) {
		return nil, status.Error(codes.InvalidArgument, "end time must be after start time")
	}

	// Generate mock cost data points
	duration := endTime.Sub(startTime)
	dataPoints := int(math.Min(float64(m.actualCostDataPoints.Load()), duration.Hours()+1))

	results := make([]*pbc.ActualCostResult, 0, dataPoints)

	for i := range dataPoints {
		timestamp := startTime.Add(time.Duration(i) * time.Hour)
		if timestamp.After(endTime) {
			break
		}

		// Generate some variation in cost data
		costVariation := costVariationBase + (costVariationRange * float64(i%costVariationMod) / costVariationMod) // Varies between 0.8x and 1.2x base rate
		cost := m.BaseHourlyRate * costVariation
		usageAmount := 1.0 * costVariation // Assume 1 unit of usage

		result := &pbc.ActualCostResult{
			Timestamp:   timestamppb.New(timestamp),
			Cost:        cost,
			UsageAmount: usageAmount,
			UsageUnit:   "hour",
			Source:      m.PluginName,
		}

		// Set expires_at caching hint if configured
		if m.ExpiresAtDuration != 0 {
			result.ExpiresAt = timestamppb.New(time.Now().Add(m.ExpiresAtDuration))
		}

		results = append(results, result)
	}

	// Apply pagination if requested
	page, nextToken, totalCount, err := paginateMockActualCosts(results, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pbc.GetActualCostResponse{
		Results:       page,
		FallbackHint:  m.FallbackHint,
		NextPageToken: nextToken,
		TotalCount:    totalCount,
	}, nil
}

// shouldOmitMetric checks if a metric kind should be omitted from responses.
func (m *MockPlugin) shouldOmitMetric(kind pbc.MetricKind) bool {
	for _, omit := range m.OmitMetrics {
		if kind == omit {
			return true
		}
	}
	return false
}

// buildImpactMetrics creates impact metrics based on configured supported metrics and utilization.
func (m *MockPlugin) buildImpactMetrics(utilization float64) []*pbc.ImpactMetric {
	var metrics []*pbc.ImpactMetric
	for _, kind := range m.SupportedMetrics {
		if m.shouldOmitMetric(kind) {
			continue
		}

		val, unit := getImpactMetricValue(kind, utilization)
		if val == 0 && unit == "" {
			continue // Skip unspecified or unknown metric kinds
		}

		// Always add carbon even if 0 for testing purposes.
		if val > 0 || kind == pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT {
			metrics = append(metrics, &pbc.ImpactMetric{
				Kind:  kind,
				Value: val,
				Unit:  unit,
			})
		}
	}
	return metrics
}

// getImpactMetricValue returns the value and unit for a given metric kind based on utilization.
func getImpactMetricValue(kind pbc.MetricKind, utilization float64) (float64, string) {
	switch kind {
	case pbc.MetricKind_METRIC_KIND_CARBON_FOOTPRINT:
		return mockCarbonPerHour * utilization, "gCO2e"
	case pbc.MetricKind_METRIC_KIND_ENERGY_CONSUMPTION:
		return mockEnergyPerHour * utilization, "kWh"
	case pbc.MetricKind_METRIC_KIND_WATER_USAGE:
		return mockWaterPerHour * utilization, "L"
	case pbc.MetricKind_METRIC_KIND_UNSPECIFIED:
		return 0, "" // Not a valid sustainability metric
	default:
		return 0, ""
	}
}

// resolvePricing returns the pricing category and spot risk score for a resource type.
// It applies resource-type-specific overrides from PricingCategoryByResourceType and
// SpotRiskScoreByResourceType, falling back to defaults if no override exists.
func (m *MockPlugin) resolvePricing(resourceType string) (pbc.FocusPricingCategory, float64) {
	pricingCategory := m.DefaultPricingCategory
	spotRiskScore := m.DefaultSpotInterruptionRiskScore

	// Check for resource-type-specific overrides
	if categoryOverride, exists := m.PricingCategoryByResourceType[resourceType]; exists {
		pricingCategory = categoryOverride
	}
	if riskOverride, exists := m.SpotRiskScoreByResourceType[resourceType]; exists {
		spotRiskScore = riskOverride
	}

	return pricingCategory, spotRiskScore
}

// getSimpleResourceKey extracts the simple resource key (e.g., "ec2", "spot") from a ResourceDescriptor.
// This matches resource types against SupportedResources to find the canonical simple key.
// It handles Pulumi-style resource descriptors like "aws:ec2/instance:Instance" by extracting
// the module name and matching against supported resources.
func (m *MockPlugin) getSimpleResourceKey(resource *pbc.ResourceDescriptor) string {
	if resource == nil {
		return ""
	}

	resourceType := resource.GetResourceType()
	if resourceType == "" {
		return ""
	}

	provider, module, resourceName, err := parseResourceType(resourceType)
	if err != nil {
		// Fall back to raw resource type for backwards compatibility
		return resourceType
	}

	// Check SupportedResources for provider
	supportedResources, exists := m.SupportedResources[provider]
	if !exists {
		return module // Fall back to module name
	}

	// Match against supported resources (same logic as EstimateCost)
	for _, supported := range supportedResources {
		if module == supported || resourceName == supported || module+"_"+resourceName == supported {
			return supported
		}
	}

	return module // Fall back to module name
}

// GetProjectedCost returns mock projected cost data.
func (m *MockPlugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if m.ProjectedCostDelay > 0 {
		time.Sleep(m.ProjectedCostDelay)
	}

	if m.ShouldErrorOnProjectedCost {
		return nil, status.Error(codes.Unavailable, "mock error: projected cost service unavailable")
	}

	// T046: Check dry_run flag - return DryRunResponse if true
	if req.GetDryRun() {
		dryRunResp, err := m.DryRun(ctx, &pbc.DryRunRequest{
			Resource: req.GetResource(),
		})
		if err != nil {
			return nil, err
		}

		return &pbc.GetProjectedCostResponse{
			DryRunResult: dryRunResp,
		}, nil
	}

	resource := req.GetResource()
	if resource == nil {
		return nil, status.Error(codes.InvalidArgument, "resource descriptor is required")
	}

	// Calculate cost based on resource type (keep in sync with GetPricingSpec).
	simpleResourceType := m.getSimpleResourceKey(resource)
	multiplier := getRateMultiplier(simpleResourceType)

	// Incorporate utilization into impact modeling (for GreenOps metrics).
	util := utilization.Get(req)

	unitPrice := m.BaseHourlyRate * multiplier
	costPerMonth := unitPrice * HoursPerDay * daysPerMonth

	billingDetail := fmt.Sprintf("mock-%s-rate (util:%.2f)", strings.ToLower(resource.GetProvider()), util)

	// Determine pricing category and spot risk score
	pricingCategory, spotRiskScore := m.resolvePricing(simpleResourceType)

	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:                 unitPrice,
		Currency:                  m.Currency,
		CostPerMonth:              costPerMonth,
		BillingDetail:             billingDetail,
		PricingCategory:           pricingCategory,
		SpotInterruptionRiskScore: spotRiskScore,
	}

	// Add impact metrics if configured
	if len(m.SupportedMetrics) > 0 {
		resp.ImpactMetrics = m.buildImpactMetrics(util)
	}

	// Set expires_at caching hint if configured
	if m.ProjectedCostExpiresAtDuration != 0 {
		resp.ExpiresAt = timestamppb.New(time.Now().Add(m.ProjectedCostExpiresAtDuration))
	}

	return resp, nil
}

// getBillingModeAndUnit returns billing mode and unit for a resource type.
func getBillingModeAndUnit(resourceType string) (string, string) {
	switch resourceType {
	case "s3", blobStorageResourceType, cloudStorageResourceType:
		return "per_gb_month", "GB-month"
	case lambdaResourceType, cloudFunctionsResourceType:
		return "per_invocation", "request"
	case namespaceResourceType:
		return "per_cpu_hour", "hour"
	case "sql_database":
		return "per_dtu", "DTU"
	default:
		return "per_hour", "hour"
	}
}

// isKnownResourceType returns true if the resource type is known/supported.
func isKnownResourceType(resourceType string) bool {
	knownTypes := []string{
		ec2ResourceType, "s3", lambdaResourceType, "rds",
		"vm", blobStorageResourceType, "sql_database",
		computeEngineResourceType, cloudStorageResourceType, cloudFunctionsResourceType,
		namespaceResourceType, "pod", "service",
	}
	for _, known := range knownTypes {
		if resourceType == known {
			return true
		}
	}
	return false
}

// getRateMultiplier returns the rate multiplier for a resource type.
func getRateMultiplier(resourceType string) float64 {
	switch resourceType {
	case ec2ResourceType, "vm", computeEngineResourceType, "compute":
		return computeRateMultiplier
	case "s3", blobStorageResourceType, cloudStorageResourceType:
		return storageRateMultiplier
	case lambdaResourceType, cloudFunctionsResourceType:
		return serverlessRateMultiplier
	case namespaceResourceType:
		return namespaceRateMultiplier
	case "sql_database":
		return databaseRateMultiplier
	default:
		return 1.0
	}
}

// getMetricHints returns metric hints for a resource type.
func getMetricHints(resourceType string) []*pbc.UsageMetricHint {
	switch resourceType {
	case ec2ResourceType, "vm", computeEngineResourceType, "compute":
		return []*pbc.UsageMetricHint{
			{Metric: "vcpu_hours", Unit: "hour"},
			{Metric: "memory_gb_hours", Unit: "hour"},
		}
	case "s3", blobStorageResourceType, cloudStorageResourceType:
		return []*pbc.UsageMetricHint{
			{Metric: "storage_gb", Unit: "GB"},
			{Metric: "requests", Unit: "count"},
		}
	case lambdaResourceType, cloudFunctionsResourceType:
		return []*pbc.UsageMetricHint{
			{Metric: "invocations", Unit: "count"},
			{Metric: "duration_ms", Unit: "millisecond"},
		}
	case namespaceResourceType:
		return []*pbc.UsageMetricHint{
			{Metric: "cpu_cores", Unit: "hour"},
			{Metric: "memory_gb", Unit: "hour"},
		}
	default:
		return []*pbc.UsageMetricHint{}
	}
}

// GetPricingSpec returns mock pricing specification.
func (m *MockPlugin) GetPricingSpec(
	_ context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	if m.PricingSpecDelay > 0 {
		time.Sleep(m.PricingSpecDelay)
	}

	if m.ShouldErrorOnPricingSpec {
		return nil, status.Error(codes.PermissionDenied, "mock error: pricing spec access denied")
	}

	resource := req.GetResource()
	if resource == nil {
		return nil, status.Error(codes.InvalidArgument, "resource descriptor is required")
	}

	// Validate required fields (FR-011)
	if resource.GetProvider() == "" {
		return nil, status.Error(codes.InvalidArgument, "provider is required")
	}
	if resource.GetResourceType() == "" {
		return nil, status.Error(codes.InvalidArgument, "resource_type is required")
	}

	resourceType := resource.GetResourceType()

	// Handle unknown resource types with not_implemented (T039)
	if !isKnownResourceType(resourceType) {
		return &pbc.GetPricingSpecResponse{
			Spec: &pbc.PricingSpec{
				Provider:     resource.GetProvider(),
				ResourceType: resourceType,
				Sku:          resource.GetSku(),
				Region:       resource.GetRegion(),
				BillingMode:  "not_implemented",
				RatePerUnit:  0,
				Currency:     m.Currency,
				Unit:         "unknown",
				Description:  fmt.Sprintf("Pricing not implemented for %s", resourceType),
				Assumptions: []string{
					fmt.Sprintf("Resource type %s is not yet supported", resourceType),
					"Pricing data is unavailable for this resource type",
				},
				Source: m.PluginName,
			},
		}, nil
	}

	billingMode, unit := getBillingModeAndUnit(resourceType)
	ratePerUnit := m.BaseHourlyRate * getRateMultiplier(resourceType)
	metricHints := getMetricHints(resourceType)

	// Generate assumptions based on resource type
	assumptions := []string{
		fmt.Sprintf(
			"Pricing based on %s %s in %s",
			resource.GetProvider(),
			resource.GetResourceType(),
			resource.GetRegion(),
		),
		"On-demand pricing without reserved capacity discounts",
		"Standard tier without additional features",
	}

	spec := &pbc.PricingSpec{
		Provider:     resource.GetProvider(),
		ResourceType: resource.GetResourceType(),
		Sku:          resource.GetSku(),
		Region:       resource.GetRegion(),
		BillingMode:  billingMode,
		RatePerUnit:  ratePerUnit,
		Currency:     m.Currency,
		Description:  fmt.Sprintf("Mock pricing for %s %s", resource.GetProvider(), resource.GetResourceType()),
		MetricHints:  metricHints,
		PluginMetadata: map[string]string{
			"mock_plugin_version": "1.0.0",
			"test_mode":           "true",
			"data_source":         "synthetic",
		},
		Source:      m.PluginName,
		Unit:        unit,
		Assumptions: assumptions,
	}

	return &pbc.GetPricingSpecResponse{
		Spec: spec,
	}, nil
}

// parseResourceType parses a resource type string in format provider:module/resource:Type.
// Returns (provider, module, resource, error). The Type component is validated but not returned.
func parseResourceType(resourceType string) (string, string, string, error) {
	const (
		expectedColonParts = 3
		expectedSlashParts = 2
	)
	// Split into provider, module/resource, and type name
	parts := strings.SplitN(resourceType, ":", expectedColonParts)
	if len(parts) != expectedColonParts || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "",
			fmt.Errorf(
				"invalid format: expected provider:module/resource:Type, got %s",
				resourceType,
			)
	}

	provider := parts[0]
	moduleResource := parts[1]
	// parts[2] (typeName) is validated in the check above but not returned

	// Split module/resource
	moduleResourceParts := strings.SplitN(moduleResource, "/", expectedSlashParts)
	if len(moduleResourceParts) != expectedSlashParts || moduleResourceParts[0] == "" || moduleResourceParts[1] == "" {
		return "", "", "",
			fmt.Errorf(
				"invalid format: expected provider:module/resource:Type, got %s",
				resourceType,
			)
	}

	module := moduleResourceParts[0]
	resource := moduleResourceParts[1]

	return provider, module, resource, nil
}

// =============================================================================
// GetRecommendations Support
// =============================================================================

// RecommendationsConfig holds configuration for mock recommendations.
type RecommendationsConfig struct {
	Recommendations []*pbc.Recommendation
	ShouldError     bool
	ErrorMessage    string
	Delay           time.Duration
}

// SetRecommendationsConfig configures the recommendations response.
//
// Thread Safety: This method is NOT safe for concurrent use. All calls to
// SetRecommendationsConfig must complete before the plugin begins serving requests.
// Typical usage is to configure the mock during test setup, before calling
// harness.Start() or benchmark.ResetTimer().
func (m *MockPlugin) SetRecommendationsConfig(config RecommendationsConfig) {
	m.RecommendationsConfig = config
}

// generateAnomalyRecommendation creates a realistic anomaly recommendation with
// appropriate negative savings (overspend) and descriptive context.
func generateAnomalyRecommendation(index int, baseInfo *pbc.ResourceRecommendationInfo,
	confidence float64, priority pbc.RecommendationPriority) *pbc.Recommendation {
	const (
		anomalyVariations     = 3
		anomalyBaseMultiplier = 50.0
		overspendModulo       = 2
	)

	isOverspend := index%overspendModulo == 0
	savings := savingsBase + float64(index)*savingsIncrement

	var description string
	if isOverspend {
		savings = -savings
		anomalyPercentage := (float64(index%anomalyVariations) + 1.0) * anomalyBaseMultiplier
		description = fmt.Sprintf(
			"Anomaly %d: Unusual spending pattern detected - %.0f%% above baseline",
			index+1, anomalyPercentage,
		)
	} else {
		description = fmt.Sprintf("Anomaly %d: Cost anomaly requiring investigation", index+1)
	}

	return &pbc.Recommendation{
		Id:          fmt.Sprintf("rec-anomaly-%d", index+1),
		Category:    pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_ANOMALY,
		ActionType:  pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_INVESTIGATE,
		Priority:    priority,
		Description: description,
		Resource:    baseInfo,
		Impact: &pbc.RecommendationImpact{
			EstimatedSavings: savings,
			Currency:         "USD",
			ProjectionPeriod: "12_months",
		},
		ConfidenceScore: &confidence,
	}
}

// GenerateSampleRecommendations creates realistic test recommendation data including anomalies.
func GenerateSampleRecommendations(count int) []*pbc.Recommendation {
	categories := []pbc.RecommendationCategory{
		pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
		pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_SECURITY,
		pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_RELIABILITY,
		pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_ANOMALY,
	}

	actionTypes := []pbc.RecommendationActionType{
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_UNSPECIFIED,         // 0
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,           // 1
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,           // 2
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_PURCHASE_COMMITMENT, // 3
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_ADJUST_REQUESTS,     // 4
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,              // 5
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_DELETE_UNUSED,       // 6
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MIGRATE,             // 7
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_CONSOLIDATE,         // 8
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_SCHEDULE,            // 9
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_REFACTOR,            // 10
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_OTHER,               // 11
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_INVESTIGATE,         // 12
	}

	priorities := []pbc.RecommendationPriority{
		pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_HIGH,
		pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM,
		pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_LOW,
	}

	providers := []string{"aws", "azure", "gcp", "kubernetes"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "asia-pacific-1"}

	recs := make([]*pbc.Recommendation, count)
	for i := range count {
		confidence := confidenceBase + float64(i%confidenceVariations)*confidenceStep
		category := categories[i%len(categories)]
		actionType := actionTypes[i%len(actionTypes)]

		// Create base resource info
		baseInfo := &pbc.ResourceRecommendationInfo{
			Id:       fmt.Sprintf("resource-%d", i+1),
			Provider: providers[i%len(providers)],
			ResourceType: fmt.Sprintf(
				"%s:compute/instance:Instance",
				providers[i%len(providers)],
			),
			Region: regions[i%len(regions)],
			Name:   fmt.Sprintf("instance-%d", i+1),
		}

		// For anomalies, use helper function to generate complete recommendation
		if category == pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_ANOMALY {
			recs[i] = generateAnomalyRecommendation(i, baseInfo, confidence, priorities[i%len(priorities)])
			continue
		}

		// Non-anomaly recommendations
		savings := savingsBase + float64(i)*savingsIncrement
		description := fmt.Sprintf("Recommendation %d: Optimize resource with cost savings", i+1)

		recs[i] = &pbc.Recommendation{
			Id:          fmt.Sprintf("rec-%d", i+1),
			Category:    category,
			ActionType:  actionType,
			Priority:    priorities[i%len(priorities)],
			Description: description,
			Resource:    baseInfo,
			Impact: &pbc.RecommendationImpact{
				EstimatedSavings: savings,
				Currency:         "USD",
				ProjectionPeriod: "12_months",
			},
			ConfidenceScore: &confidence,
		}
	}
	return recs
}

// GetRecommendations returns mock cost optimization recommendations.
func (m *MockPlugin) GetRecommendations(
	_ context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	if m.RecommendationsConfig.Delay > 0 {
		time.Sleep(m.RecommendationsConfig.Delay)
	}

	if m.RecommendationsConfig.ShouldError {
		msg := m.RecommendationsConfig.ErrorMessage
		if msg == "" {
			msg = "mock error: recommendations unavailable"
		}
		return nil, status.Error(codes.Unavailable, msg)
	}

	// Validate request contract constraints (including MaxTargetResources limit)
	if err := ValidateGetRecommendationsRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request validation failed: %v", err)
	}

	// Return configured recommendations or empty list
	recs := m.RecommendationsConfig.Recommendations
	if recs == nil {
		recs = []*pbc.Recommendation{}
	}

	// Apply target_resources filtering first (defines SCOPE).
	// When provided, only recommendations matching target resources are included.
	// When empty/nil, all recommendations are returned (backward compatible).
	if len(req.GetTargetResources()) > 0 {
		recs = filterByTargetResources(recs, req.GetTargetResources())
	}

	// Apply filter if provided (defines SELECTION CRITERIA within scope).
	// NOTE: This uses a local implementation rather than pluginsdk.ApplyRecommendationFilter
	// to avoid circular imports (pluginsdk imports testing for conformance functions).
	if req.GetFilter() != nil {
		recs = applyMockFilter(recs, req.GetFilter())
	}

	// Apply pagination if page_size is specified
	var nextToken string
	if req.GetPageSize() > 0 || req.GetPageToken() != "" {
		var paginationErr error
		recs, nextToken, paginationErr = paginateMockRecommendations(
			recs,
			req.GetPageSize(),
			req.GetPageToken(),
		)
		if paginationErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", paginationErr)
		}
	}

	// NOTE: This mock intentionally calculates summary from the paginated (current page)
	// results, not the total filtered set. This is by design for testing pagination
	// scenarios where clients need to verify per-page behavior. Production implementations
	// may calculate summary from the full filtered dataset before pagination.
	summary := CalculateMockSummary(recs, req.GetProjectionPeriod())

	return &pbc.GetRecommendationsResponse{
		Recommendations: recs,
		Summary:         summary,
		NextPageToken:   nextToken,
	}, nil
}

// GetBudgets implements the mock GetBudgets RPC method.
//
//nolint:gocognit // Budget filtering logic requires multiple conditions
func (m *MockPlugin) GetBudgets(
	_ context.Context,
	req *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	if m.ShouldErrorOnBudgets {
		return nil, status.Error(codes.Internal, "mock error")
	}

	// Return configured budgets or empty list
	budgets := m.MockBudgets
	if budgets == nil {
		budgets = []*pbc.Budget{}
	}

	// Apply basic filtering if requested
	if req.GetFilter() != nil {
		filter := req.GetFilter()
		if len(filter.GetProviders()) > 0 {
			var filtered []*pbc.Budget
			for _, budget := range budgets {
				for _, provider := range filter.GetProviders() {
					if strings.Contains(budget.GetSource(), provider) {
						filtered = append(filtered, budget)
						break
					}
				}
			}
			budgets = filtered
		}
	}

	// Calculate summary
	totalBudgets := len(budgets)
	budgetsOk := 0
	budgetsWarning := 0
	budgetsCritical := 0
	budgetsExceeded := 0

	for _, budget := range budgets {
		if budget.GetStatus() != nil {
			switch budget.GetStatus().GetHealth() {
			case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK,
				pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_UNSPECIFIED:
				budgetsOk++
			case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING:
				budgetsWarning++
			case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_CRITICAL:
				budgetsCritical++
			case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED:
				budgetsExceeded++
			}
		} else {
			budgetsOk++ // No status means assume OK
		}
	}

	return &pbc.GetBudgetsResponse{
		Budgets: budgets,
		Summary: &pbc.BudgetSummary{
			TotalBudgets:    int32(totalBudgets),    //nolint:gosec // length will not exceed int32 max
			BudgetsOk:       int32(budgetsOk),       //nolint:gosec // count will not exceed int32 max
			BudgetsWarning:  int32(budgetsWarning),  //nolint:gosec // count will not exceed int32 max
			BudgetsCritical: int32(budgetsCritical), //nolint:gosec // count will not exceed int32 max
			BudgetsExceeded: int32(budgetsExceeded), //nolint:gosec // count will not exceed int32 max
		},
	}, nil
}

// CalculateMockSummary builds a RecommendationSummary from the given recommendations.
// NOTE: This duplicates pluginsdk.CalculateRecommendationSummary logic to avoid circular
// imports (pluginsdk imports testing for conformance functions).
func CalculateMockSummary(recs []*pbc.Recommendation, projectionPeriod string) *pbc.RecommendationSummary {
	summary := &pbc.RecommendationSummary{
		TotalRecommendations: int32(len(recs)), //nolint:gosec // length will not exceed int32 max
		CountByCategory:      make(map[string]int32),
		SavingsByCategory:    make(map[string]float64),
		CountByActionType:    make(map[string]int32),
		SavingsByActionType:  make(map[string]float64),
		ProjectionPeriod:     projectionPeriod,
	}

	var totalSavings float64
	var detectedCurrency string
	var currencyMismatch bool
	for _, rec := range recs {
		catName := rec.GetCategory().String()
		actionName := rec.GetActionType().String()

		summary.CountByCategory[catName]++
		summary.CountByActionType[actionName]++

		if impact := rec.GetImpact(); impact != nil {
			savings := impact.GetEstimatedSavings()
			totalSavings += savings
			summary.SavingsByCategory[catName] += savings
			summary.SavingsByActionType[actionName] += savings
			if c := impact.GetCurrency(); c != "" {
				if detectedCurrency == "" {
					detectedCurrency = c
				} else if detectedCurrency != c {
					currencyMismatch = true
				}
			}
		}
	}
	// Clear currency if recommendations have mixed currencies (sum is ambiguous)
	if currencyMismatch {
		detectedCurrency = ""
	}
	summary.TotalEstimatedSavings = totalSavings
	summary.Currency = detectedCurrency

	return summary
}

// =============================================================================
// Target Resources Filtering (Feature 019-target-resources)
// =============================================================================

// filterByTargetResources filters recommendations to only include those matching
// at least one of the target resources. This implements the SCOPE filtering for
// stack-scoped recommendations, pre-deployment optimization, and batch analysis.
//
// Matching uses OR logic: a recommendation is included if it matches ANY target.
func filterByTargetResources(recs []*pbc.Recommendation, targets []*pbc.ResourceDescriptor) []*pbc.Recommendation {
	result := make([]*pbc.Recommendation, 0, len(recs))
	for _, rec := range recs {
		if matchesAnyTargetResource(rec, targets) {
			result = append(result, rec)
		}
	}
	return result
}

// matchesAnyTargetResource checks if a recommendation matches at least one target resource.
func matchesAnyTargetResource(rec *pbc.Recommendation, targets []*pbc.ResourceDescriptor) bool {
	resource := rec.GetResource()
	if resource == nil {
		return false
	}

	for _, target := range targets {
		if matchesResourceDescriptor(resource, target) {
			return true
		}
	}
	return false
}

// matchesResourceDescriptor checks if a ResourceRecommendationInfo matches a ResourceDescriptor.
// Matching rules (per spec FR-008):
//   - provider and resource_type must always match (required fields)
//   - sku, region, and tags are matched only when specified in the target
//   - If specified, optional fields must match exactly (strict matching)
func matchesResourceDescriptor(resource *pbc.ResourceRecommendationInfo, target *pbc.ResourceDescriptor) bool {
	if target == nil {
		return false
	}

	// Required fields must always match
	if resource.GetProvider() != target.GetProvider() {
		return false
	}
	if resource.GetResourceType() != target.GetResourceType() {
		return false
	}

	// Optional fields: only check if specified in target (strict matching)
	if target.GetSku() != "" && resource.GetSku() != target.GetSku() {
		return false
	}
	if target.GetRegion() != "" && resource.GetRegion() != target.GetRegion() {
		return false
	}

	// Tags: all specified tags must be present with exact values
	if len(target.GetTags()) > 0 {
		resourceTags := resource.GetTags()
		for key, targetValue := range target.GetTags() {
			if resourceTags[key] != targetValue {
				return false
			}
		}
	}

	return true
}

// applyMockFilter filters recommendations based on the filter criteria.
// NOTE: This duplicates pluginsdk.ApplyRecommendationFilter logic to avoid circular
// imports (pluginsdk imports testing for conformance functions).
func applyMockFilter(recs []*pbc.Recommendation, filter *pbc.RecommendationFilter) []*pbc.Recommendation {
	result := make([]*pbc.Recommendation, 0, len(recs))
	for _, rec := range recs {
		if matchesMockFilter(rec, filter) {
			result = append(result, rec)
		}
	}
	return result
}

// matchesProviderFilter checks if the recommendation matches the provider filter.
func matchesProviderFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	if filter.GetProvider() == "" {
		return true
	}
	return rec.GetResource() != nil && rec.GetResource().GetProvider() == filter.GetProvider()
}

// matchesRegionFilter checks if the recommendation matches the region filter.
func matchesRegionFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	if filter.GetRegion() == "" {
		return true
	}
	return rec.GetResource() != nil && rec.GetResource().GetRegion() == filter.GetRegion()
}

// matchesResourceTypeFilter checks if the recommendation matches the resource type filter.
func matchesResourceTypeFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	if filter.GetResourceType() == "" {
		return true
	}
	return rec.GetResource() != nil && rec.GetResource().GetResourceType() == filter.GetResourceType()
}

// matchesCategoryFilter checks if the recommendation matches the category filter.
func matchesCategoryFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	if filter.GetCategory() == pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED {
		return true
	}
	return rec.GetCategory() == filter.GetCategory()
}

// matchesActionTypeFilter checks if the recommendation matches the action type filter.
func matchesActionTypeFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	if filter.GetActionType() == pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_UNSPECIFIED {
		return true
	}
	return rec.GetActionType() == filter.GetActionType()
}

// matchesConfidenceScoreFilter checks if the recommendation matches the confidence score filter.
func matchesConfidenceScoreFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	minScore := filter.GetMinConfidenceScore()
	if minScore <= 0 {
		return true
	}
	recScore := rec.GetConfidenceScore()
	return recScore > 0 && recScore >= minScore
}

// matchesMockFilter checks if a recommendation matches the filter criteria.
func matchesMockFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	return matchesProviderFilter(rec, filter) &&
		matchesRegionFilter(rec, filter) &&
		matchesResourceTypeFilter(rec, filter) &&
		matchesCategoryFilter(rec, filter) &&
		matchesActionTypeFilter(rec, filter) &&
		matchesConfidenceScoreFilter(rec, filter)
}

// mockDefaultPageSize is the default page size for mock pagination.
// NOTE: This intentionally mirrors pluginsdk.DefaultPageSize (50) for consistency.
// We maintain a local constant to avoid circular imports (pluginsdk imports testing).
const mockDefaultPageSize int32 = 50

// mockMaxPageSize is the maximum page size for mock pagination.
// NOTE: This intentionally mirrors pluginsdk.MaxPageSize (1000) for consistency.
const mockMaxPageSize int32 = 1000

// paginateMockRecommendations applies pagination to recommendations.
// Returns the page of recommendations, next page token, and any error.
func paginateMockRecommendations(
	recs []*pbc.Recommendation,
	pageSize int32,
	pageToken string,
) ([]*pbc.Recommendation, string, error) {
	// Use default page size if not specified
	if pageSize <= 0 {
		pageSize = mockDefaultPageSize
	}
	if pageSize > mockMaxPageSize {
		pageSize = mockMaxPageSize
	}

	// Decode offset from page token
	var offset int
	if pageToken != "" {
		decoded, err := decodeMockPageToken(pageToken)
		if err != nil {
			return nil, "", err
		}
		offset = decoded
	}

	// Handle offset beyond range - return empty
	if offset >= len(recs) {
		return []*pbc.Recommendation{}, "", nil
	}

	// Calculate end index
	end := offset + int(pageSize)
	if end > len(recs) {
		end = len(recs)
	}

	// Get page slice
	page := recs[offset:end]

	// Generate next token if more results exist
	var nextToken string
	if end < len(recs) {
		nextToken = encodeMockPageToken(end)
	}

	return page, nextToken, nil
}

// paginateMockActualCosts applies pagination to actual cost results.
// Returns the page of results, next page token, total count, and any error.
// When page_size is 0 and page_token is empty (proto3 defaults), returns all results
// for backward compatibility with non-paginated plugins.
func paginateMockActualCosts(
	results []*pbc.ActualCostResult,
	pageSize int32,
	pageToken string,
) ([]*pbc.ActualCostResult, string, int32, error) {
	var totalCount int32
	if len(results) > math.MaxInt32 {
		totalCount = math.MaxInt32
	} else {
		totalCount = int32(len(results)) //nolint:gosec // safe after bounds check above
	}

	// Backward compatibility: when both page_size and page_token are defaults,
	// return all results (non-paginated behavior)
	if pageSize <= 0 && pageToken == "" {
		return results, "", totalCount, nil
	}

	// Use default page size if not specified
	if pageSize <= 0 {
		pageSize = mockDefaultPageSize
	}
	if pageSize > mockMaxPageSize {
		pageSize = mockMaxPageSize
	}

	// Decode offset from page token
	var offset int
	if pageToken != "" {
		decoded, err := decodeMockPageToken(pageToken)
		if err != nil {
			return nil, "", 0, err
		}
		offset = decoded
	}

	// Handle offset beyond range
	if offset >= len(results) {
		return []*pbc.ActualCostResult{}, "", totalCount, nil
	}

	// Calculate end index
	end := offset + int(pageSize)
	if end > len(results) {
		end = len(results)
	}

	// Get page slice
	page := results[offset:end]

	// Generate next token if more results exist
	var nextToken string
	if end < len(results) {
		nextToken = encodeMockPageToken(end)
	}

	return page, nextToken, totalCount, nil
}

// encodeMockPageToken encodes an offset as a base64 page token.
func encodeMockPageToken(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// decodeMockPageToken decodes a base64 page token to an offset.
func decodeMockPageToken(token string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid page token encoding: %w", err)
	}
	offset, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0, fmt.Errorf("invalid page token value: %w", err)
	}
	if offset < 0 {
		return 0, errors.New("invalid page token: negative offset")
	}
	return offset, nil
}

// EstimateCost returns mock cost estimate for a resource before deployment.
func (m *MockPlugin) EstimateCost(
	_ context.Context,
	req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	if m.EstimateCostDelay > 0 {
		time.Sleep(m.EstimateCostDelay)
	}

	if m.ShouldErrorOnEstimateCost {
		return nil, status.Error(codes.Unavailable, "mock error: pricing source unavailable")
	}

	resourceType := req.GetResourceType()

	// Validate resource type format (FR-003)
	if resourceType == "" {
		return nil, status.Error(codes.InvalidArgument, "resource_type is required")
	}

	provider, module, resource, err := parseResourceType(resourceType)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			fmt.Sprintf(
				"resource_type must follow provider:module/resource:Type format, got: %s",
				resourceType,
			),
		)
	}

	// Check if provider is supported (FR-008)
	providerSupported := false
	for _, supportedProvider := range m.SupportedProviders {
		if provider == supportedProvider {
			providerSupported = true
			break
		}
	}

	if !providerSupported {
		return nil, status.Error(
			codes.NotFound,
			fmt.Sprintf("resource type %s is not supported by this plugin", resourceType),
		)
	}

	// Check if resource type is supported for this provider
	supportedResources, exists := m.SupportedResources[provider]
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			fmt.Sprintf("resource type %s is not supported by this plugin", resourceType),
		)
	}

	// For mock purposes, try to match against module, resource, or module_resource combinations
	// This handles formats like "aws:ec2/instance:Instance" where "ec2" is the module
	simpleResourceType := module
	resourceSupported := false
	for _, supportedResource := range supportedResources {
		if module == supportedResource || resource == supportedResource || module+"_"+resource == supportedResource {
			resourceSupported = true
			simpleResourceType = supportedResource
			break
		}
	}

	if !resourceSupported {
		return nil, status.Error(
			codes.NotFound,
			fmt.Sprintf("resource type %s is not supported by this plugin", resourceType),
		)
	}

	// Calculate monthly cost based on resource type
	multiplier := getRateMultiplier(simpleResourceType)
	hourlyRate := m.BaseHourlyRate * multiplier
	monthlyHours := float64(HoursPerDay * daysPerMonth)
	monthlyCost := hourlyRate * monthlyHours

	// Determine pricing category and spot risk score
	pricingCategory, spotRiskScore := m.resolvePricing(simpleResourceType)

	resp := &pbc.EstimateCostResponse{
		Currency:                  m.Currency,
		CostMonthly:               monthlyCost,
		PricingCategory:           pricingCategory,
		SpotInterruptionRiskScore: spotRiskScore,
	}

	// Set expires_at caching hint if configured
	if m.EstimateCostExpiresAtDuration != 0 {
		resp.ExpiresAt = timestamppb.New(time.Now().Add(m.EstimateCostExpiresAtDuration))
	}

	return resp, nil
}

// =============================================================================
// FallbackHint Helper Functions for Benchmarks
// =============================================================================

// NewActualCostResponseWithHint creates a GetActualCostResponse with results and a fallback hint.
// This is a convenience function for benchmark tests.
func NewActualCostResponseWithHint(
	results []*pbc.ActualCostResult,
	hint pbc.FallbackHint,
) *pbc.GetActualCostResponse {
	return &pbc.GetActualCostResponse{
		Results:      results,
		FallbackHint: hint,
	}
}

// NOTE: ValidateActualCostResponseHelper was removed. Use pluginsdk.ValidateActualCostResponse
// instead, which is the canonical implementation for validating GetActualCostResponse messages.
// Import: "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"

// NOTE: getUtilization was removed. Use github.com/rshade/finfocus-spec/sdk/go/internal/utilization.Get()
// which is the shared implementation. The circular dependency that previously prevented this
// import has been resolved by creating the internal/utilization package.
