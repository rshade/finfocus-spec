package pluginsdk

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/currency"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// HoursPerMonth is the standard number of hours used for monthly cost calculations.
// This value (730) represents the average number of hours in a month (365.25 days / 12 months * 24 hours).
const HoursPerMonth = 730.0

// HoursPerDay is the number of hours in a day for time calculations.
const HoursPerDay = 24

// ResourceMatcher helps plugins determine if they support a resource.
//
// Thread Safety: ResourceMatcher is NOT safe for concurrent use. All calls to
// AddProvider and AddResourceType must complete before the plugin begins serving
// gRPC requests. Typical usage is to configure the matcher during plugin
// initialization, before calling Serve().
type ResourceMatcher struct {
	supportedProviders map[string]bool
	supportedTypes     map[string]bool
}

// NewResourceMatcher creates a ResourceMatcher with initialized empty maps for supported providers and supported resource types.
func NewResourceMatcher() *ResourceMatcher {
	return &ResourceMatcher{
		supportedProviders: make(map[string]bool),
		supportedTypes:     make(map[string]bool),
	}
}

// AddProvider adds a supported provider (e.g., "aws", "azure", "gcp").
// Empty strings are ignored.
func (rm *ResourceMatcher) AddProvider(provider string) {
	if provider == "" {
		return
	}
	rm.supportedProviders[provider] = true
}

// AddResourceType adds a supported resource type (e.g., "aws:ec2:Instance").
// Empty strings are ignored.
func (rm *ResourceMatcher) AddResourceType(resourceType string) {
	if resourceType == "" {
		return
	}
	rm.supportedTypes[resourceType] = true
}

// Supports checks if a resource is supported by this plugin.
func (rm *ResourceMatcher) Supports(resource *pbc.ResourceDescriptor) bool {
	if rm == nil || resource == nil {
		return false
	}

	if len(rm.supportedProviders) > 0 {
		if !rm.supportedProviders[resource.GetProvider()] {
			return false
		}
	}

	if len(rm.supportedTypes) > 0 {
		if !rm.supportedTypes[resource.GetResourceType()] {
			return false
		}
	}

	return true
}

// CostCalculator provides utilities for cost calculations.
type CostCalculator struct{}

// NewCostCalculator returns a new CostCalculator for performing cost conversions and creating cost responses.
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{}
}

// HourlyToMonthly converts hourly cost to monthly cost (730 hours).
func (cc *CostCalculator) HourlyToMonthly(hourlyCost float64) float64 {
	return hourlyCost * HoursPerMonth
}

// MonthlyToHourly converts monthly cost to hourly cost (730 hours).
func (cc *CostCalculator) MonthlyToHourly(monthlyCost float64) float64 {
	return monthlyCost / HoursPerMonth
}

// CreateProjectedCostResponse creates a standard projected cost response.
// unitPrice is expected to be an hourly rate; CostPerMonth is derived using 730 hours.
func (cc *CostCalculator) CreateProjectedCostResponse(
	currency string,
	unitPrice float64,
	billingDetail string,
) *pbc.GetProjectedCostResponse {
	return &pbc.GetProjectedCostResponse{
		Currency:      currency,
		UnitPrice:     unitPrice,
		CostPerMonth:  cc.HourlyToMonthly(unitPrice),
		BillingDetail: billingDetail,
	}
}

// CreateActualCostResponse creates a standard actual cost response.
//
// The FallbackHint is not set, which means it defaults to FALLBACK_HINT_UNSPECIFIED (0).
// This default value is treated as "no fallback needed" by the core system, ensuring
// backwards compatibility with existing plugins that do not explicitly set a hint.
//
// For explicit control over the fallback hint, use NewActualCostResponse with
// the WithFallbackHint option instead.
func (cc *CostCalculator) CreateActualCostResponse(
	results []*pbc.ActualCostResult,
) *pbc.GetActualCostResponse {
	return &pbc.GetActualCostResponse{
		Results: results,
	}
}

// ActualCostResponseOption is a functional option for configuring GetActualCostResponse.
// Use these options with NewActualCostResponse to create responses with explicit
// fallback hints and results.
type ActualCostResponseOption func(*pbc.GetActualCostResponse)

// WithFallbackHint sets the fallback hint on the response.
//
// Use this option to explicitly signal to the core system whether it should
// attempt to query other plugins for the requested resource:
//
//   - FALLBACK_HINT_UNSPECIFIED (0): Default. Treated as "no fallback needed".
//   - FALLBACK_HINT_NONE (1): Plugin has data; do not attempt fallback.
//   - FALLBACK_HINT_RECOMMENDED (2): Plugin has no data; core SHOULD try other plugins.
//   - FALLBACK_HINT_REQUIRED (3): Plugin cannot handle request; core MUST try fallback.
//
// Example - Plugin has data, no fallback needed:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithResults(results),
//	    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
//	)
//
// Example - Plugin has no data, recommend fallback:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED),
//	)
//
// Example - Plugin cannot handle this resource type:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_REQUIRED),
//	)
//
// Important: For actual errors (API failures, network timeouts, invalid credentials),
// return a gRPC error instead of using a fallback hint. Hints are for "no data"
// scenarios, not system failures.
func WithFallbackHint(hint pbc.FallbackHint) ActualCostResponseOption {
	return func(resp *pbc.GetActualCostResponse) {
		resp.FallbackHint = hint
	}
}

// WithNextPageToken sets the next page token on the response.
//
// Use this option to include a continuation token when there are more pages
// of results available. Pass an empty string to indicate the last page.
//
// Example - Response with more pages:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithResults(pageResults),
//	    pluginsdk.WithNextPageToken(nextToken),
//	)
func WithNextPageToken(token string) ActualCostResponseOption {
	return func(resp *pbc.GetActualCostResponse) {
		resp.NextPageToken = token
	}
}

// WithTotalCount sets the total count of matching records on the response.
//
// Use this option to report the total number of records across all pages.
// A value of 0 indicates the total is unknown or expensive to compute.
//
// Example:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithResults(pageResults),
//	    pluginsdk.WithNextPageToken(nextToken),
//	    pluginsdk.WithTotalCount(int32(len(allResults))),
//	)
func WithTotalCount(count int32) ActualCostResponseOption {
	return func(resp *pbc.GetActualCostResponse) {
		resp.TotalCount = count
	}
}

// WithResults sets the cost results on the response.
//
// Use this option to include actual cost data in the response.
// If the results slice is non-empty, you typically want to use
// FALLBACK_HINT_NONE to indicate the plugin has authoritative data.
//
// Zero-cost results vs no data:
//   - Empty results [] with FALLBACK_HINT_RECOMMENDED = "no billing data found"
//   - Results with cost: 0.00 and FALLBACK_HINT_NONE = "resource is free tier"
func WithResults(results []*pbc.ActualCostResult) ActualCostResponseOption {
	return func(resp *pbc.GetActualCostResponse) {
		resp.Results = results
	}
}

// NewActualCostResponse creates a GetActualCostResponse using functional options.
//
// This is the preferred way to create responses when you need to explicitly
// set the fallback hint. The response starts with default values (no results,
// FALLBACK_HINT_UNSPECIFIED) and options are applied in order.
//
// Example - Plugin found cost data:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithResults(results),
//	    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
//	)
//
// Example - Plugin has no data, recommend trying other plugins:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED),
//	)
func NewActualCostResponse(opts ...ActualCostResponseOption) *pbc.GetActualCostResponse {
	resp := &pbc.GetActualCostResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}

// ValidateActualCostResponse validates a GetActualCostResponse for structural correctness.
//
// Validation Rules:
//   - Response is not nil
//   - All results have non-negative costs
//   - All results have non-empty source identifiers
//   - No nil results in the results slice
//
// Validation stops at the first error encountered. To find all validation errors
// in a response, you would need to implement your own multi-error collection.
//
// Semantic Consistency (NOT validated):
// This function performs structural validation only. The following combinations
// are structurally valid but semantically unusual:
//   - results present + FALLBACK_HINT_RECOMMENDED (plugin has data but suggests fallback)
//   - results present + FALLBACK_HINT_REQUIRED (plugin has data but requires fallback)
//
// The core system treats data presence as authoritative, so these combinations
// will not trigger fallback behavior. Use them only when the plugin has partial
// data and wants to signal that other plugins should also be queried.
//
// Example:
//
//	resp := pluginsdk.NewActualCostResponse(
//	    pluginsdk.WithResults(results),
//	    pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
//	)
//	if err := pluginsdk.ValidateActualCostResponse(resp); err != nil {
//	    return nil, status.Errorf(codes.Internal, "invalid response: %v", err)
//	}
func ValidateActualCostResponse(resp *pbc.GetActualCostResponse) error {
	if resp == nil {
		return errors.New("response cannot be nil")
	}

	for i, result := range resp.GetResults() {
		if result == nil {
			return fmt.Errorf("results[%d] cannot be nil", i)
		}
		if result.GetCost() < 0 {
			return fmt.Errorf("results[%d].cost cannot be negative: %f", i, result.GetCost())
		}
		if result.GetSource() == "" {
			return fmt.Errorf("results[%d].source cannot be empty", i)
		}
		if ts := result.GetExpiresAt(); ts != nil {
			if err := ts.CheckValid(); err != nil {
				return fmt.Errorf("results[%d].expires_at is invalid: %w", i, err)
			}
		}
	}

	return nil
}

// =============================================================================
// ResourceDescriptor Helper Functions
// =============================================================================

// ResourceDescriptorOption is a functional option for configuring ResourceDescriptor.
// Use these options with NewResourceDescriptor to create descriptors with ID and ARN fields.
type ResourceDescriptorOption func(*pbc.ResourceDescriptor)

// WithID sets the client correlation identifier on the ResourceDescriptor.
//
// The id field is used for request/response correlation in batch operations.
// When provided, plugins MUST include this ID in any recommendations or responses
// related to this resource, enabling clients to match responses to their original
// requests.
//
// The ID is treated as an opaque string - plugins MUST NOT validate or
// transform this value. Common formats include Pulumi URNs, UUIDs, or
// application-specific identifiers.
//
// Example - Setting a Pulumi URN as correlation ID:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "aws", "ec2",
//	    pluginsdk.WithID("urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web"),
//	)
//
// Example - Using with GetRecommendationsRequest:
//
//	req := &pbc.GetRecommendationsRequest{
//	    TargetResources: []*pbc.ResourceDescriptor{
//	        pluginsdk.NewResourceDescriptor(
//	            "aws", "ec2",
//	            pluginsdk.WithID("batch-001"),
//	            pluginsdk.WithARN("arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"),
//	        ),
//	    },
//	}
func WithID(id string) ResourceDescriptorOption {
	return func(desc *pbc.ResourceDescriptor) {
		desc.Id = id
	}
}

// WithARN sets the canonical cloud resource identifier on the ResourceDescriptor.
//
// The arn field is used for exact resource matching instead of fuzzy
// type/sku/region/tags matching. When provided, plugins SHOULD use this
// for precise resource lookup.
//
// This field uses "arn" as the name for consistency with GetActualCostRequest,
// but accepts canonical identifiers from any cloud provider:
//
//   - AWS ARN: arn:aws:ec2:us-east-1:123456789012:instance/i-abc123
//   - Azure Resource ID: /subscriptions/{sub}/resourceGroups/{rg}/providers/...
//   - GCP Full Resource Name: //compute.googleapis.com/projects/{project}/zones/{zone}/instances/{name}
//   - Kubernetes: {cluster}/{namespace}/{kind}/{name} or UID
//   - Cloudflare: {zone-id}/{resource-type}/{resource-id}
//
// Example - AWS resource with exact ARN matching:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "aws", "ec2",
//	    pluginsdk.WithARN("arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"),
//	)
//
// Example - Azure resource:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "azure", "virtualMachines",
//	    pluginsdk.WithARN("/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"),
//	)
func WithARN(arn string) ResourceDescriptorOption {
	return func(desc *pbc.ResourceDescriptor) {
		desc.Arn = arn
	}
}

// WithSKU sets the provider-specific SKU or instance size on the ResourceDescriptor.
//
// Example:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "aws", "ec2",
//	    pluginsdk.WithSKU("t3.micro"),
//	)
func WithSKU(sku string) ResourceDescriptorOption {
	return func(desc *pbc.ResourceDescriptor) {
		desc.Sku = sku
	}
}

// WithRegion sets the deployment region on the ResourceDescriptor.
//
// Example:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "aws", "ec2",
//	    pluginsdk.WithRegion("us-east-1"),
//	)
func WithRegion(region string) ResourceDescriptorOption {
	return func(desc *pbc.ResourceDescriptor) {
		desc.Region = region
	}
}

// WithTags sets the resource tags on the ResourceDescriptor.
//
// Example:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "aws", "ec2",
//	    pluginsdk.WithTags(map[string]string{"env": "prod", "team": "platform"}),
//	)
func WithTags(tags map[string]string) ResourceDescriptorOption {
	return func(desc *pbc.ResourceDescriptor) {
		desc.Tags = tags
	}
}

// NewResourceDescriptor creates a new ResourceDescriptor with the given provider
// and resource type, plus any additional options.
//
// Provider and resource_type are required fields per the proto specification.
// Use functional options to set optional fields like id, arn, sku, region, and tags.
//
// Example - Basic descriptor:
//
//	desc := pluginsdk.NewResourceDescriptor("aws", "ec2")
//
// Example - Full descriptor with ID and ARN for batch correlation:
//
//	desc := pluginsdk.NewResourceDescriptor(
//	    "aws", "ec2",
//	    pluginsdk.WithID("urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web"),
//	    pluginsdk.WithARN("arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"),
//	    pluginsdk.WithSKU("t3.micro"),
//	    pluginsdk.WithRegion("us-east-1"),
//	    pluginsdk.WithTags(map[string]string{"env": "prod"}),
//	)
func NewResourceDescriptor(provider, resourceType string, opts ...ResourceDescriptorOption) *pbc.ResourceDescriptor {
	desc := &pbc.ResourceDescriptor{
		Provider:     provider,
		ResourceType: resourceType,
	}
	for _, opt := range opts {
		opt(desc)
	}
	return desc
}

// NotSupportedError returns an error indicating the specified resource type and provider are not supported.
// The formatted message includes the resource's ResourceType and Provider.
func NotSupportedError(resource *pbc.ResourceDescriptor) error {
	return fmt.Errorf(
		"resource type %s from provider %s is not supported",
		resource.GetResourceType(),
		resource.GetProvider(),
	)
}

// NoDataError returns a standard error when no cost data is available.
func NoDataError(resourceID string) error {
	return fmt.Errorf("no cost data available for resource %s", resourceID)
}

// BasePlugin provides common functionality for plugin implementations.
type BasePlugin struct {
	name    string
	matcher *ResourceMatcher
	calc    *CostCalculator
}

// NewBasePlugin creates a new BasePlugin with the given name and initializes its
// ResourceMatcher and CostCalculator for shared plugin functionality.
func NewBasePlugin(name string) *BasePlugin {
	return &BasePlugin{
		name:    name,
		matcher: NewResourceMatcher(),
		calc:    NewCostCalculator(),
	}
}

// Name returns the plugin name.
func (bp *BasePlugin) Name() string {
	return bp.name
}

// Matcher returns the resource matcher.
func (bp *BasePlugin) Matcher() *ResourceMatcher {
	return bp.matcher
}

// Calculator returns the cost calculator.
func (bp *BasePlugin) Calculator() *CostCalculator {
	return bp.calc
}

// GetProjectedCost provides a default implementation that returns not supported.
func (bp *BasePlugin) GetProjectedCost(
	_ context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if req == nil {
		return nil, errors.New("GetProjectedCostRequest cannot be nil")
	}
	resource := req.GetResource()
	if resource == nil {
		return nil, errors.New("resource cannot be nil")
	}
	return nil, NotSupportedError(resource)
}

// GetActualCost provides a default implementation that returns no data.
func (bp *BasePlugin) GetActualCost(
	_ context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if req == nil {
		return nil, errors.New("GetActualCostRequest cannot be nil")
	}
	return nil, NoDataError(req.GetResourceId())
}

// GetPricingSpec provides a default implementation that returns not implemented.
// Override this method in your plugin to return pricing specifications.
func (bp *BasePlugin) GetPricingSpec(
	_ context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	if req == nil {
		return nil, errors.New("GetPricingSpecRequest cannot be nil")
	}
	return nil, errors.New("GetPricingSpec not implemented")
}

// EstimateCost provides a default implementation that returns not implemented.
// Override this method in your plugin to return cost estimates.
func (bp *BasePlugin) EstimateCost(
	_ context.Context,
	req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	if req == nil {
		return nil, errors.New("EstimateCostRequest cannot be nil")
	}
	return nil, errors.New("EstimateCost not implemented")
}

// =============================================================================
// GetRecommendations Validation Helpers
// =============================================================================

// ValidateRecommendation validates a recommendation has all required fields.
// Returns an error if any required field is missing or invalid.
func ValidateRecommendation(rec *pbc.Recommendation) error {
	if rec == nil {
		return errors.New("recommendation cannot be nil")
	}
	if rec.GetId() == "" {
		return errors.New("recommendation.id is required")
	}
	if rec.GetCategory() == pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED {
		return errors.New("recommendation.category must be specified")
	}
	if rec.GetActionType() == pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_UNSPECIFIED {
		return errors.New("recommendation.action_type must be specified")
	}
	if rec.GetResource() == nil {
		return errors.New("recommendation.resource is required")
	}
	if err := ValidateResourceRecommendationInfo(rec.GetResource()); err != nil {
		return fmt.Errorf("recommendation.resource: %w", err)
	}
	if rec.GetImpact() == nil {
		return errors.New("recommendation.impact is required")
	}
	if err := ValidateRecommendationImpact(rec.GetImpact()); err != nil {
		return fmt.Errorf("recommendation.impact: %w", err)
	}
	if rec.ConfidenceScore != nil {
		if err := ValidateConfidenceScore(rec.ConfidenceScore); err != nil {
			return fmt.Errorf("recommendation.confidence_score: %w", err)
		}
	}
	return nil
}

// ValidateResourceRecommendationInfo validates resource information fields.
func ValidateResourceRecommendationInfo(res *pbc.ResourceRecommendationInfo) error {
	if res == nil {
		return errors.New("resource info cannot be nil")
	}
	if res.GetId() == "" {
		return errors.New("resource.id is required")
	}
	if res.GetProvider() == "" {
		return errors.New("resource.provider is required")
	}
	return nil
}

// ValidateRecommendationImpact validates impact information fields.
// Validates currency against ISO 4217 standard.
func ValidateRecommendationImpact(impact *pbc.RecommendationImpact) error {
	if impact == nil {
		return errors.New("impact cannot be nil")
	}
	if impact.GetCurrency() == "" {
		return errors.New("impact.currency is required")
	}
	if !currency.IsValid(impact.GetCurrency()) {
		return fmt.Errorf("impact.currency %q is not a valid ISO 4217 currency code", impact.GetCurrency())
	}
	if impact.GetEstimatedSavings() < 0 {
		return errors.New("impact.estimated_savings cannot be negative")
	}
	return nil
}

// ValidateConfidenceScore validates that a confidence score is within the valid range.
// Confidence scores must be between 0.0 and 1.0 inclusive.
func ValidateConfidenceScore(score *float64) error {
	if score == nil {
		return nil // nil is valid (confidence not available)
	}
	if *score < 0.0 || *score > 1.0 {
		return fmt.Errorf("confidence_score must be between 0.0 and 1.0, got %f", *score)
	}
	return nil
}

// ValidateRecommendationSummary validates summary information fields.
func ValidateRecommendationSummary(summary *pbc.RecommendationSummary) error {
	if summary == nil {
		return nil // Summary is optional
	}
	if summary.GetCurrency() != "" && !currency.IsValid(summary.GetCurrency()) {
		return fmt.Errorf("summary.currency %q is not a valid ISO 4217 currency code", summary.GetCurrency())
	}
	return nil
}

// =============================================================================
// GetRecommendations Filter Helpers
// =============================================================================

// ApplyRecommendationFilter filters recommendations based on the provided filter criteria.
// ApplyRecommendationFilter returns recommendations that match ALL specified filter criteria.
// Empty filter values are ignored (match all).
//
// Supported filter fields:
//   - Core filters (1-7): provider, region, resource_type, category, action_type, sku, tags
//   - P0 filters (8-10): priority, min_estimated_savings, source
//   - P1 filters (11-13): account_id (sort_by/sort_order handled by SortRecommendations)
//   - P2 filters (14-16): min_confidence_score, max_age_days, resource_id
func ApplyRecommendationFilter(
	recommendations []*pbc.Recommendation,
	filter *pbc.RecommendationFilter,
) []*pbc.Recommendation {
	if filter == nil {
		return recommendations
	}

	result := make([]*pbc.Recommendation, 0, len(recommendations))
	for _, rec := range recommendations {
		if matchesFilter(rec, filter) {
			result = append(result, rec)
		}
	}
	return result
}

// matchesFilter checks if a recommendation matches all filter criteria.
//
//nolint:gocognit,gocyclo,cyclop // filter matching logic for 16 fields requires complexity
func matchesFilter(rec *pbc.Recommendation, filter *pbc.RecommendationFilter) bool {
	// Core filters (1-7)
	// Filter by provider
	if filter.GetProvider() != "" {
		if rec.GetResource() == nil || rec.GetResource().GetProvider() != filter.GetProvider() {
			return false
		}
	}

	// Filter by region
	if filter.GetRegion() != "" {
		if rec.GetResource() == nil || rec.GetResource().GetRegion() != filter.GetRegion() {
			return false
		}
	}

	// Filter by resource type
	if filter.GetResourceType() != "" {
		if rec.GetResource() == nil || rec.GetResource().GetResourceType() != filter.GetResourceType() {
			return false
		}
	}

	// Filter by category
	if filter.GetCategory() != pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED {
		if rec.GetCategory() != filter.GetCategory() {
			return false
		}
	}

	// Filter by action type
	if filter.GetActionType() != pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_UNSPECIFIED {
		if rec.GetActionType() != filter.GetActionType() {
			return false
		}
	}

	// Filter by SKU (field 6)
	if filter.GetSku() != "" {
		if rec.GetResource() == nil || rec.GetResource().GetSku() != filter.GetSku() {
			return false
		}
	}

	// Filter by tags (field 7) - all specified tags must match
	if len(filter.GetTags()) > 0 {
		if rec.GetResource() == nil {
			return false
		}
		recTags := rec.GetResource().GetTags()
		for key, value := range filter.GetTags() {
			if recTags[key] != value {
				return false
			}
		}
	}

	// P0 filters (8-10)
	// Filter by priority (field 8)
	if filter.GetPriority() != pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_UNSPECIFIED {
		if rec.GetPriority() != filter.GetPriority() {
			return false
		}
	}

	// Filter by min_estimated_savings (field 9)
	if filter.GetMinEstimatedSavings() > 0 {
		if rec.GetImpact() == nil || rec.GetImpact().GetEstimatedSavings() < filter.GetMinEstimatedSavings() {
			return false
		}
	}

	// Filter by source (field 10)
	if filter.GetSource() != "" {
		if rec.GetSource() != filter.GetSource() {
			return false
		}
	}

	// P1 filters (11-13) - sort_by/sort_order handled by SortRecommendations
	// Filter by account_id (field 11) - stored in resource tags or metadata
	// Note: account_id is typically stored in recommendation metadata
	if filter.GetAccountId() != "" {
		// Check metadata for account_id
		if rec.GetMetadata()["account_id"] != filter.GetAccountId() {
			return false
		}
	}

	// P2 filters (14-16)
	// Filter by min_confidence_score (field 14)
	if filter.GetMinConfidenceScore() > 0 {
		if rec.ConfidenceScore != nil {
			if *rec.ConfidenceScore < filter.GetMinConfidenceScore() { //nolint:protogetter // direct access needed to distinguish nil from 0
				return false
			}
		}
		// nil confidence passes filter (unknown != low confidence).
		// We explicitly check for nil to avoid GetConfidenceScore() defaulting to 0,
		// which would incorrectly filter out recommendations with unknown confidence.
	}

	// Filter by max_age_days (field 15)
	if filter.GetMaxAgeDays() > 0 && rec.GetCreatedAt() != nil {
		maxAge := time.Duration(filter.GetMaxAgeDays()) * HoursPerDay * time.Hour
		cutoff := time.Now().Add(-maxAge)
		if rec.GetCreatedAt().AsTime().Before(cutoff) {
			return false
		}
	}

	// Filter by resource_id (field 16)
	if filter.GetResourceId() != "" {
		if rec.GetResource() == nil || rec.GetResource().GetId() != filter.GetResourceId() {
			return false
		}
	}

	return true
}

// ExcludeRecommendationsByIDs removes recommendations with IDs in the exclusion list.
// Use this to filter out dismissed recommendations from GetRecommendations results.
func ExcludeRecommendationsByIDs(
	recommendations []*pbc.Recommendation,
	excludedIDs []string,
) []*pbc.Recommendation {
	if len(excludedIDs) == 0 {
		return recommendations
	}

	// Build lookup set for O(1) checks
	excluded := make(map[string]bool, len(excludedIDs))
	for _, id := range excludedIDs {
		excluded[id] = true
	}

	result := make([]*pbc.Recommendation, 0, len(recommendations))
	for _, rec := range recommendations {
		if rec == nil {
			continue
		}
		if !excluded[rec.GetId()] {
			result = append(result, rec)
		}
	}
	return result
}

// SortRecommendations sorts recommendations based on the specified sort criteria.
// If sort_by is UNSPECIFIED, recommendations are returned in their original order.
// Default sort order is DESC for ESTIMATED_SAVINGS and PRIORITY, ASC for others.
func SortRecommendations(
	recommendations []*pbc.Recommendation,
	sortBy pbc.RecommendationSortBy,
	sortOrder pbc.SortOrder,
) []*pbc.Recommendation {
	if len(recommendations) == 0 || sortBy == pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_UNSPECIFIED {
		return recommendations
	}

	// Make a copy to avoid modifying the original slice
	sorted := make([]*pbc.Recommendation, len(recommendations))
	copy(sorted, recommendations)

	// Determine effective sort order (default varies by sort field)
	ascending := determineSortOrder(sortBy, sortOrder)

	sort.SliceStable(sorted, func(i, j int) bool {
		if ascending {
			return compareRecommendations(sorted[i], sorted[j], sortBy)
		}
		// Swap arguments for descending order to maintain strict weak ordering.
		// Using !less would break the sort contract for equal elements.
		return compareRecommendations(sorted[j], sorted[i], sortBy)
	})

	return sorted
}

// determineSortOrder returns true for ascending, false for descending.
func determineSortOrder(sortBy pbc.RecommendationSortBy, sortOrder pbc.SortOrder) bool {
	if sortOrder == pbc.SortOrder_SORT_ORDER_ASC {
		return true
	}
	if sortOrder == pbc.SortOrder_SORT_ORDER_DESC {
		return false
	}
	// Default: DESC for savings/priority, ASC for created_at/confidence
	switch sortBy {
	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
		pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_PRIORITY:
		return false
	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_CREATED_AT,
		pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_CONFIDENCE,
		pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_UNSPECIFIED:
		return true
	default:
		return true
	}
}

// compareRecommendations returns true if rec1 < rec2 for the given sort field.
func compareRecommendations(rec1, rec2 *pbc.Recommendation, sortBy pbc.RecommendationSortBy) bool {
	switch sortBy {
	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS:
		savingsI := float64(0)
		savingsJ := float64(0)
		if rec1.GetImpact() != nil {
			savingsI = rec1.GetImpact().GetEstimatedSavings()
		}
		if rec2.GetImpact() != nil {
			savingsJ = rec2.GetImpact().GetEstimatedSavings()
		}
		return savingsI < savingsJ

	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_PRIORITY:
		return rec1.GetPriority() < rec2.GetPriority()

	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_CREATED_AT:
		timeI := time.Time{}
		timeJ := time.Time{}
		if rec1.GetCreatedAt() != nil {
			timeI = rec1.GetCreatedAt().AsTime()
		}
		if rec2.GetCreatedAt() != nil {
			timeJ = rec2.GetCreatedAt().AsTime()
		}
		return timeI.Before(timeJ)

	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_CONFIDENCE:
		// Treat nil as -1 to group unknown confidence at one end
		conf1 := float64(-1)
		conf2 := float64(-1)
		if rec1.ConfidenceScore != nil {
			conf1 = *rec1.ConfidenceScore //nolint:protogetter // direct access needed to distinguish nil from 0
		}
		if rec2.ConfidenceScore != nil {
			conf2 = *rec2.ConfidenceScore //nolint:protogetter // direct access needed to distinguish nil from 0
		}
		return conf1 < conf2

	case pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_UNSPECIFIED:
		return false

	default:
		return false
	}
}

// ValidateRecommendationFilter validates filter field values.
// Returns an error if any filter value is invalid.
func ValidateRecommendationFilter(filter *pbc.RecommendationFilter) error {
	if filter == nil {
		return nil
	}

	// Validate min_estimated_savings
	if filter.GetMinEstimatedSavings() < 0 {
		return errors.New("min_estimated_savings cannot be negative")
	}

	// Validate min_confidence_score (must be between 0.0 and 1.0)
	minConf := filter.GetMinConfidenceScore()
	if minConf < 0 || minConf > 1 {
		return fmt.Errorf(
			"min_confidence_score must be between 0.0 and 1.0, got %f",
			minConf,
		)
	}

	// Validate max_age_days
	if filter.GetMaxAgeDays() < 0 {
		return errors.New("max_age_days cannot be negative")
	}

	return nil
}

// =============================================================================
// GetRecommendations Pagination Helpers
// =============================================================================

// DefaultPageSize is the default number of recommendations per page.
const DefaultPageSize = 50

// MaxPageSize is the maximum allowed page size.
const MaxPageSize = 1000

// PaginateRecommendations applies pagination to a slice of recommendations.
// PaginateRecommendations returns the page of recommendations and the next page token (empty if last page).
func PaginateRecommendations(
	recommendations []*pbc.Recommendation,
	pageSize int32,
	pageToken string,
) ([]*pbc.Recommendation, string, error) {
	// Determine effective page size
	effectivePageSize := int(pageSize)
	if effectivePageSize <= 0 {
		effectivePageSize = DefaultPageSize
	}
	if effectivePageSize > MaxPageSize {
		log.Warn().
			Int("requested_page_size", int(pageSize)).
			Int("max_page_size", MaxPageSize).
			Msg("page_size exceeded maximum; clamped to MaxPageSize")
		effectivePageSize = MaxPageSize
	}

	// Decode offset from page token
	offset := 0
	if pageToken != "" {
		var err error
		offset, err = DecodePageToken(pageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page_token: %w", err)
		}
	}

	// Handle out-of-bounds offset
	total := len(recommendations)
	if offset >= total {
		return []*pbc.Recommendation{}, "", nil
	}

	// Calculate page boundaries
	end := offset + effectivePageSize
	if end > total {
		end = total
	}

	// Extract page
	page := recommendations[offset:end]

	// Generate next page token
	nextToken := ""
	if end < total {
		nextToken = EncodePageToken(end)
	}

	return page, nextToken, nil
}

// EncodePageToken creates an opaque page token from an offset.
func EncodePageToken(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// DecodePageToken decodes an opaque page token to an offset.
func DecodePageToken(token string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, errors.New("malformed page token")
	}
	offset, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0, errors.New("invalid page token value")
	}
	if offset < 0 {
		return 0, errors.New("page token offset cannot be negative")
	}
	if offset > math.MaxInt32 {
		return 0, errors.New("page token offset exceeds maximum allowed value")
	}
	return offset, nil
}

// =============================================================================
// GetActualCost Pagination Helpers
// =============================================================================

// PaginateActualCosts applies pagination to a slice of actual cost results.
// Returns the page of results, next page token (empty if last page), total count, and any error.
//
// Page size is clamped to [DefaultPageSize, MaxPageSize]. The total count is
// automatically set to len(results). Page tokens are base64-encoded offsets
// compatible with EncodePageToken/DecodePageToken.
//
// Example usage in a plugin's GetActualCost handler:
//
//	allResults, err := p.fetchCostData(ctx, req)
//	if err != nil {
//	    return nil, err
//	}
//	page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(
//	    allResults, req.PageSize, req.PageToken,
//	)
//	if err != nil {
//	    return nil, status.Errorf(codes.InvalidArgument, "%v", err)
//	}
//	return &pbc.GetActualCostResponse{
//	    Results:       page,
//	    NextPageToken: nextToken,
//	    TotalCount:    totalCount,
//	}, nil
func PaginateActualCosts(
	results []*pbc.ActualCostResult,
	pageSize int32,
	pageToken string,
) ([]*pbc.ActualCostResult, string, int32, error) {
	total := len(results)

	// Normalize negative page sizes to 0 (proto contract: <=0 means use default)
	if pageSize < 0 {
		pageSize = 0
	}

	// Handle legacy hosts: if no pagination params are provided, return all results
	// This maintains backward compatibility with hosts that don't use pagination
	if pageSize == 0 && pageToken == "" {
		// Clamp totalCount to int32 max to avoid overflow
		totalCount := int32(total)
		if total > math.MaxInt32 {
			log.Warn().
				Int("total", total).
				Int32("clamped_to", math.MaxInt32).
				Msg("total_count clamped to int32 max; actual count exceeds representable range")
			totalCount = math.MaxInt32
		}
		return results, "", totalCount, nil
	}

	// Determine effective page size
	effectivePageSize := int(pageSize)
	if effectivePageSize <= 0 {
		effectivePageSize = DefaultPageSize
	}
	if effectivePageSize > MaxPageSize {
		log.Warn().
			Int("requested_page_size", int(pageSize)).
			Int("max_page_size", MaxPageSize).
			Msg("page_size exceeded maximum; clamped to MaxPageSize")
		effectivePageSize = MaxPageSize
	}

	// Decode offset from page token
	offset := 0
	if pageToken != "" {
		var err error
		offset, err = DecodePageToken(pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page_token: %w", err)
		}
	}

	// Clamp totalCount to int32 max to avoid overflow
	totalCount := int32(total)
	if total > math.MaxInt32 {
		log.Warn().
			Int("total", total).
			Int32("clamped_to", math.MaxInt32).
			Msg("total_count clamped to int32 max; actual count exceeds representable range")
		totalCount = math.MaxInt32
	}

	// Handle out-of-bounds offset
	if offset >= total {
		return []*pbc.ActualCostResult{}, "", totalCount, nil
	}

	// Calculate page boundaries
	end := offset + effectivePageSize
	if end > total {
		end = total
	}

	// Extract page
	page := results[offset:end]

	// Generate next page token
	nextToken := ""
	if end < total {
		nextToken = EncodePageToken(end)
	}

	return page, nextToken, totalCount, nil
}

// =============================================================================
// GetRecommendations Summary Calculation
// =============================================================================

// CalculateRecommendationSummary computes aggregated summary statistics for recommendations.
func CalculateRecommendationSummary(
	recommendations []*pbc.Recommendation,
	projectionPeriod string,
) *pbc.RecommendationSummary {
	summary := &pbc.RecommendationSummary{
		TotalRecommendations: int32(len(recommendations)), //nolint:gosec // length will not exceed int32 max
		ProjectionPeriod:     projectionPeriod,
		CountByCategory:      make(map[string]int32),
		SavingsByCategory:    make(map[string]float64),
		CountByActionType:    make(map[string]int32),
		SavingsByActionType:  make(map[string]float64),
	}

	var totalSavings float64
	var detectedCurrency string
	var currencyMismatch bool

	for _, rec := range recommendations {
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
// Pricing Tier Field Builders
// =============================================================================

// EstimateCostResponseOption is a functional option for configuring EstimateCostResponse.
type EstimateCostResponseOption func(*pbc.EstimateCostResponse)

// WithPricingCategory sets the pricing_category field for EstimateCostResponse.
//
// Example:
//
//	resp := pluginsdk.NewEstimateCostResponse(
//	    pluginsdk.WithPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC),
//	)
func WithPricingCategory(category pbc.FocusPricingCategory) EstimateCostResponseOption {
	return func(resp *pbc.EstimateCostResponse) {
		resp.PricingCategory = category
	}
}

// WithSpotRisk sets the spot_interruption_risk_score field for EstimateCostResponse.
// The score must be between 0.0 and 1.0.
//
// This function performs fail-fast validation and panics for invalid values:
//   - NaN or Inf values (programming error)
//   - Values outside [0.0, 1.0] range (programming error)
//
// Semantic consistency between score and pricing_category is validated separately
// by ValidateEstimateCostResponse.
//
// Example:
//
//	resp := pluginsdk.NewEstimateCostResponse(
//	    pluginsdk.WithSpotRisk(0.8),  // 80% interruption risk
//	)
func WithSpotRisk(score float64) EstimateCostResponseOption {
	// Fail-fast validation for programming errors
	if math.IsNaN(score) || math.IsInf(score, 0) {
		panic(fmt.Sprintf("WithSpotRisk: invalid score (NaN/Inf): %v", score))
	}
	// Allow epsilon tolerance on lower bound only (handles float errors near 0)
	// Upper bound is strict 1.0 - probability cannot exceed 100%
	if score < -spotRiskEpsilon || score > 1.0 {
		panic(fmt.Sprintf("WithSpotRisk: score must be between 0.0 and 1.0, got %f", score))
	}

	return func(resp *pbc.EstimateCostResponse) {
		// Clamp score to valid range to handle minor floating-point errors
		clampedScore := score
		if clampedScore < 0.0 {
			clampedScore = 0.0
		} else if clampedScore > 1.0 {
			clampedScore = 1.0
		}
		resp.SpotInterruptionRiskScore = clampedScore
	}
}

// WithEstimateCost sets the basic cost fields for EstimateCostResponse.
//
// Example:
//
//	resp := pluginsdk.NewEstimateCostResponse(
//	    pluginsdk.WithEstimateCost("USD", 50.0),
//	)
func WithEstimateCost(currency string, costMonthly float64) EstimateCostResponseOption {
	return func(resp *pbc.EstimateCostResponse) {
		resp.Currency = currency
		resp.CostMonthly = costMonthly
	}
}

// NewEstimateCostResponse creates an EstimateCostResponse with functional options.
//
// Example:
//
//	resp := pluginsdk.NewEstimateCostResponse(
//	    pluginsdk.WithEstimateCost("USD", 50.0),
//	    pluginsdk.WithPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC),
//	    pluginsdk.WithSpotRisk(0.8),
//	)
func NewEstimateCostResponse(opts ...EstimateCostResponseOption) *pbc.EstimateCostResponse {
	resp := &pbc.EstimateCostResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}

// GetProjectedCostResponseOption is a functional option for configuring GetProjectedCostResponse.
type GetProjectedCostResponseOption func(*pbc.GetProjectedCostResponse)

// WithProjectedCostPricingCategory sets the pricing_category field for GetProjectedCostResponse.
//
// Example:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED),
//	)
func WithProjectedCostPricingCategory(category pbc.FocusPricingCategory) GetProjectedCostResponseOption {
	return func(resp *pbc.GetProjectedCostResponse) {
		resp.PricingCategory = category
	}
}

// WithProjectedCostSpotRisk sets the spot_interruption_risk_score field for GetProjectedCostResponse.
// The score must be between 0.0 and 1.0.
//
// This function performs fail-fast validation and panics for invalid values:
//   - NaN or Inf values (programming error)
//   - Values outside [0.0, 1.0] range (programming error)
//
// Semantic consistency between score and pricing_category is validated separately
// by ValidateGetProjectedCostResponse.
//
// Example:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostSpotRisk(0.3),  // 30% interruption risk
//	)
func WithProjectedCostSpotRisk(score float64) GetProjectedCostResponseOption {
	// Fail-fast validation for programming errors
	if math.IsNaN(score) || math.IsInf(score, 0) {
		panic(fmt.Sprintf("WithProjectedCostSpotRisk: invalid score (NaN/Inf): %v", score))
	}
	if score < 0.0 || score > 1.0 {
		panic(fmt.Sprintf("WithProjectedCostSpotRisk: score must be between 0.0 and 1.0, got %f", score))
	}

	return func(resp *pbc.GetProjectedCostResponse) {
		resp.SpotInterruptionRiskScore = score
	}
}

// WithProjectedCostDetails sets the basic cost details for GetProjectedCostResponse.
//
// Example:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostDetails(0.05, "USD", 36.50, "spot-instance"),
//	)
func WithProjectedCostDetails(
	unitPrice float64,
	currency string,
	costPerMonth float64,
	billingDetail string,
) GetProjectedCostResponseOption {
	return func(resp *pbc.GetProjectedCostResponse) {
		resp.UnitPrice = unitPrice
		resp.Currency = currency
		resp.CostPerMonth = costPerMonth
		resp.BillingDetail = billingDetail
	}
}

// WithPredictionInterval sets the prediction interval fields for GetProjectedCostResponse.
//
// This sets the lower and upper bounds of a prediction interval along with the
// confidence level.
//
// Fail-fast validation (panics on programming errors):
//   - NaN or Inf values for lower, upper, or confidence
//   - Negative lower bound (lower < 0)
//   - Invalid interval structure (lower > upper)
//   - Confidence out of range (confidence <= 0 || confidence > 1.0)
//
// Deferred validation (checked by ValidateGetProjectedCostResponse):
//   - cost_per_month consistency (cost must be within [lower, upper])
//   - Both bounds must be set together (not just one)
//
// IMPORTANT: Callers MUST call ValidateGetProjectedCostResponse() on the final
// response to validate cost-within-interval constraints.
//
// Example:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostDetails(0.05, "USD", 36.50, "spot-instance"),
//	    pluginsdk.WithPredictionInterval(30.0, 45.0, 0.95),  // 95% CI: [$30, $45]
//	)
//	if err := pluginsdk.ValidateGetProjectedCostResponse(resp); err != nil {
//	    return nil, status.Errorf(codes.InvalidArgument, "invalid response: %v", err)
//	}
func WithPredictionInterval(lower, upper, confidence float64) GetProjectedCostResponseOption {
	// Fail-fast validation for programming errors

	// Check for NaN/Inf values
	if math.IsNaN(lower) || math.IsInf(lower, 0) {
		panic(fmt.Sprintf("WithPredictionInterval: lower bound is NaN/Inf: %v", lower))
	}
	if math.IsNaN(upper) || math.IsInf(upper, 0) {
		panic(fmt.Sprintf("WithPredictionInterval: upper bound is NaN/Inf: %v", upper))
	}
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
		panic(fmt.Sprintf("WithPredictionInterval: confidence is NaN/Inf: %v", confidence))
	}

	// Check for negative lower bound
	if lower < 0 {
		panic(fmt.Sprintf("WithPredictionInterval: lower bound cannot be negative: %f", lower))
	}

	// Check for structural invalidity (lower > upper)
	if lower > upper {
		panic(fmt.Sprintf("WithPredictionInterval: lower bound (%f) > upper bound (%f)", lower, upper))
	}

	// Check confidence range (must be in range (0.0, 1.0])
	if confidence <= 0 || confidence > 1.0 {
		panic(fmt.Sprintf("WithPredictionInterval: confidence must be in (0.0, 1.0], got %f", confidence))
	}

	return func(resp *pbc.GetProjectedCostResponse) {
		resp.PredictionIntervalLower = &lower
		resp.PredictionIntervalUpper = &upper
		resp.ConfidenceLevel = &confidence
	}
}

// WithProjectedCostExpiresAt returns a GetProjectedCostResponseOption that sets
// the expires_at caching hint on the projected cost response.
//
// A zero time.Time results in a nil expires_at (no caching guidance).
//
// Usage:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostExpiresAt(time.Now().Add(24 * time.Hour)),
//	)
func WithProjectedCostExpiresAt(expiresAt time.Time) GetProjectedCostResponseOption {
	return func(resp *pbc.GetProjectedCostResponse) {
		if expiresAt.IsZero() {
			resp.ExpiresAt = nil
			return
		}
		resp.ExpiresAt = timestamppb.New(expiresAt)
	}
}

// NewGetProjectedCostResponse creates a GetProjectedCostResponse with functional options.
//
// Example:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostDetails(0.05, "USD", 36.50, "spot-instance"),
//	    pluginsdk.WithProjectedCostPricingCategory(pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC),
//	    pluginsdk.WithProjectedCostSpotRisk(0.8),
//	)
//
// Example with prediction interval:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostDetails(0.05, "USD", 36.50, "spot-instance"),
//	    pluginsdk.WithPredictionInterval(30.0, 45.0, 0.95),  // 95% CI
//	)
//
// Example with caching hint:
//
//	resp := pluginsdk.NewGetProjectedCostResponse(
//	    pluginsdk.WithProjectedCostDetails(0.05, "USD", 36.50, "spot-instance"),
//	    pluginsdk.WithProjectedCostExpiresAt(time.Now().Add(24 * time.Hour)),
//	)
func NewGetProjectedCostResponse(opts ...GetProjectedCostResponseOption) *pbc.GetProjectedCostResponse {
	resp := &pbc.GetProjectedCostResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}
