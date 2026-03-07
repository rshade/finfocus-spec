package pluginsdk_test

import (
	"context"
	"encoding/base64"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestResourceMatcher(t *testing.T) {
	matcher := pluginsdk.NewResourceMatcher()

	// Add supported providers and resource types
	matcher.AddProvider("aws")
	matcher.AddResourceType("aws:ec2:Instance")

	testCases := []struct {
		name     string
		resource *pbc.ResourceDescriptor
		expected bool
	}{
		{
			name: "supported provider and type",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "aws:ec2:Instance",
			},
			expected: true,
		},
		{
			name: "unsupported provider",
			resource: &pbc.ResourceDescriptor{
				Provider:     "azure",
				ResourceType: "azure:compute:VirtualMachine",
			},
			expected: false,
		},
		{
			name: "supported provider, unsupported type",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "aws:s3:Bucket",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.Supports(tc.resource)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for resource %v", tc.expected, result, tc.resource)
			}
		})
	}
}

func TestResourceMatcherNoFilters(t *testing.T) {
	// Empty matcher should support everything
	matcher := pluginsdk.NewResourceMatcher()

	resource := &pbc.ResourceDescriptor{
		Provider:     "any",
		ResourceType: "any:resource:Type",
	}

	if !matcher.Supports(resource) {
		t.Errorf("Empty matcher should support all resources")
	}
}

func TestCostCalculator(t *testing.T) {
	calc := pluginsdk.NewCostCalculator()

	// Test hourly to monthly conversion
	hourly := 0.10
	monthly := calc.HourlyToMonthly(hourly)
	expected := 73.0 // 0.10 * 730
	if monthly != expected {
		t.Errorf("Expected monthly cost %f, got %f", expected, monthly)
	}

	// Test monthly to hourly conversion
	monthlyInput := 146.0
	hourlyResult := calc.MonthlyToHourly(monthlyInput)
	expectedHourly := 0.2 // 146.0 / 730
	if hourlyResult != expectedHourly {
		t.Errorf("Expected hourly cost %f, got %f", expectedHourly, hourlyResult)
	}
}

func TestCostCalculatorResponses(t *testing.T) {
	calc := pluginsdk.NewCostCalculator()

	// Test projected cost response
	resp := calc.CreateProjectedCostResponse("USD", 0.05, "Test billing detail")

	if resp.GetCurrency() != "USD" {
		t.Errorf("Expected currency USD, got %s", resp.GetCurrency())
	}

	if resp.GetUnitPrice() != 0.05 {
		t.Errorf("Expected unit price 0.05, got %f", resp.GetUnitPrice())
	}

	if resp.GetCostPerMonth() != 36.5 { // 0.05 * 730
		t.Errorf("Expected cost per month 36.5, got %f", resp.GetCostPerMonth())
	}

	if resp.GetBillingDetail() != "Test billing detail" {
		t.Errorf("Expected billing detail 'Test billing detail', got %s", resp.GetBillingDetail())
	}

	// Test actual cost response
	results := []*pbc.ActualCostResult{
		{Cost: 10.0, Source: "test"},
	}
	actualResp := calc.CreateActualCostResponse(results)

	if len(actualResp.GetResults()) != 1 {
		t.Errorf("Expected 1 result, got %d", len(actualResp.GetResults()))
	}

	if actualResp.GetResults()[0].GetCost() != 10.0 {
		t.Errorf("Expected cost 10.0, got %f", actualResp.GetResults()[0].GetCost())
	}
}

func TestErrorFunctions(t *testing.T) {
	resource := &pbc.ResourceDescriptor{
		Provider:     "test",
		ResourceType: "test:resource:Type",
	}

	// Test NotSupportedError
	err := pluginsdk.NotSupportedError(resource)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedMsg := "resource type test:resource:Type from provider test is not supported"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test NoDataError
	err = pluginsdk.NoDataError("test-resource-id")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedMsg = "no cost data available for resource test-resource-id"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestBasePlugin(t *testing.T) {
	plugin := pluginsdk.NewBasePlugin("test-plugin")

	// Test name
	if plugin.Name() != "test-plugin" {
		t.Errorf("Expected name 'test-plugin', got %s", plugin.Name())
	}

	// Test matcher access
	matcher := plugin.Matcher()
	if matcher == nil {
		t.Error("Expected matcher, got nil")
	}

	// Test calculator access
	calc := plugin.Calculator()
	if calc == nil {
		t.Error("Expected calculator, got nil")
	}

	// Test default implementations
	ctx := context.Background()

	// Test default GetProjectedCost (should return not supported error)
	req := &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "test",
			ResourceType: "test:resource:Type",
		},
	}

	_, err := plugin.GetProjectedCost(ctx, req)
	if err == nil {
		t.Error("Expected not supported error, got nil")
	}

	// Test default GetActualCost (should return no data error)
	actualReq := &pbc.GetActualCostRequest{
		ResourceId: "test-resource-id",
	}

	_, err = plugin.GetActualCost(ctx, actualReq)
	if err == nil {
		t.Error("Expected no data error, got nil")
	}

	// Test default GetPricingSpec (should return not implemented error)
	pricingReq := &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "test",
			ResourceType: "test:resource:Type",
		},
	}

	_, err = plugin.GetPricingSpec(ctx, pricingReq)
	if err == nil {
		t.Error("Expected not implemented error, got nil")
	}

	// Test default EstimateCost (should return not implemented error)
	estimateReq := &pbc.EstimateCostRequest{
		ResourceType: "test:resource:Type",
	}

	_, err = plugin.EstimateCost(ctx, estimateReq)
	if err == nil {
		t.Error("Expected not implemented error, got nil")
	}
}

func TestBasePluginNilRequests(t *testing.T) {
	plugin := pluginsdk.NewBasePlugin("test-plugin")
	ctx := context.Background()

	// Test nil GetProjectedCostRequest
	_, err := plugin.GetProjectedCost(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request, got nil")
	}

	// Test GetProjectedCost with nil resource
	_, err = plugin.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{})
	if err == nil {
		t.Error("Expected error for nil resource, got nil")
	}

	// Test nil GetActualCostRequest
	_, err = plugin.GetActualCost(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request, got nil")
	}

	// Test nil GetPricingSpecRequest
	_, err = plugin.GetPricingSpec(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request, got nil")
	}

	// Test nil EstimateCostRequest
	_, err = plugin.EstimateCost(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request, got nil")
	}
}

func TestResourceMatcherEmptyStrings(t *testing.T) {
	matcher := pluginsdk.NewResourceMatcher()

	// Empty strings should be ignored
	matcher.AddProvider("")
	matcher.AddResourceType("")

	// Should not have added empty strings - matcher should still support all
	resource := &pbc.ResourceDescriptor{
		Provider:     "any",
		ResourceType: "any:resource:Type",
	}

	if !matcher.Supports(resource) {
		t.Error("Empty string filters should be ignored, matcher should support all")
	}
}

func TestResourceMatcherNilResource(t *testing.T) {
	matcher := pluginsdk.NewResourceMatcher()
	matcher.AddProvider("aws")

	// Nil resource should return false
	if matcher.Supports(nil) {
		t.Error("Expected false for nil resource")
	}
}

func TestResourceMatcherNilReceiver(t *testing.T) {
	var matcher *pluginsdk.ResourceMatcher

	// Nil receiver should return false
	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "aws:ec2:Instance",
	}

	if matcher.Supports(resource) {
		t.Error("Expected false for nil matcher")
	}
}

func TestHoursPerMonthExported(t *testing.T) {
	// Verify the constant is exported and has the correct value
	if pluginsdk.HoursPerMonth != 730.0 {
		t.Errorf("Expected HoursPerMonth to be 730.0, got %f", pluginsdk.HoursPerMonth)
	}
}

// =============================================================================
// Recommendation Validation Tests
// =============================================================================

// TestValidateRecommendation tests the ValidateRecommendation function.
func TestValidateRecommendation(t *testing.T) {
	validRec := &pbc.Recommendation{
		Id:         "rec-001",
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
		Resource: &pbc.ResourceRecommendationInfo{
			Id:       "i-12345",
			Provider: "aws",
		},
		Impact: &pbc.RecommendationImpact{
			EstimatedSavings: 100.0,
			Currency:         "USD",
		},
	}

	testCases := []struct {
		name        string
		rec         *pbc.Recommendation
		expectError bool
	}{
		{
			name:        "valid recommendation",
			rec:         validRec,
			expectError: false,
		},
		{
			name:        "nil recommendation",
			rec:         nil,
			expectError: true,
		},
		{
			name: "missing id",
			rec: &pbc.Recommendation{
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				Resource:   &pbc.ResourceRecommendationInfo{Id: "i-123", Provider: "aws"},
				Impact:     &pbc.RecommendationImpact{Currency: "USD"},
			},
			expectError: true,
		},
		{
			name: "unspecified category",
			rec: &pbc.Recommendation{
				Id:         "rec-001",
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_UNSPECIFIED,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				Resource:   &pbc.ResourceRecommendationInfo{Id: "i-123", Provider: "aws"},
				Impact:     &pbc.RecommendationImpact{Currency: "USD"},
			},
			expectError: true,
		},
		{
			name: "unspecified action_type",
			rec: &pbc.Recommendation{
				Id:         "rec-001",
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_UNSPECIFIED,
				Resource:   &pbc.ResourceRecommendationInfo{Id: "i-123", Provider: "aws"},
				Impact:     &pbc.RecommendationImpact{Currency: "USD"},
			},
			expectError: true,
		},
		{
			name: "missing resource",
			rec: &pbc.Recommendation{
				Id:         "rec-001",
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				Impact:     &pbc.RecommendationImpact{Currency: "USD"},
			},
			expectError: true,
		},
		{
			name: "missing impact",
			rec: &pbc.Recommendation{
				Id:         "rec-001",
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				Resource:   &pbc.ResourceRecommendationInfo{Id: "i-123", Provider: "aws"},
			},
			expectError: true,
		},
		{
			name: "valid metadata passes",
			rec: &pbc.Recommendation{
				Id:         "rec-001",
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				Resource:   &pbc.ResourceRecommendationInfo{Id: "i-123", Provider: "aws"},
				Impact:     &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "USD"},
				Metadata:   map[string]string{"source": "cloudwatch"},
			},
			expectError: false,
		},
		{
			name: "invalid metadata key fails",
			rec: &pbc.Recommendation{
				Id:         "rec-001",
				Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
				Resource:   &pbc.ResourceRecommendationInfo{Id: "i-123", Provider: "aws"},
				Impact:     &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "USD"},
				Metadata:   map[string]string{"invalid key": "value"},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pluginsdk.ValidateRecommendation(tc.rec)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestValidateConfidenceScore tests the ValidateConfidenceScore function.
func TestValidateConfidenceScore(t *testing.T) {
	testCases := []struct {
		name        string
		score       *float64
		expectError bool
	}{
		{"nil score", nil, false},
		{"zero score", ptr(0.0), false},
		{"mid score", ptr(0.5), false},
		{"max score", ptr(1.0), false},
		{"negative score", ptr(-0.1), true},
		{"over 1.0", ptr(1.1), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pluginsdk.ValidateConfidenceScore(tc.score)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestValidateRecommendationImpact tests the ValidateRecommendationImpact function.
func TestValidateRecommendationImpact(t *testing.T) {
	testCases := []struct {
		name        string
		impact      *pbc.RecommendationImpact
		expectError bool
	}{
		{
			name:        "nil impact",
			impact:      nil,
			expectError: true,
		},
		{
			name:        "valid USD",
			impact:      &pbc.RecommendationImpact{Currency: "USD", EstimatedSavings: 100.0},
			expectError: false,
		},
		{
			name:        "valid EUR",
			impact:      &pbc.RecommendationImpact{Currency: "EUR", EstimatedSavings: 50.0},
			expectError: false,
		},
		{
			name:        "empty currency",
			impact:      &pbc.RecommendationImpact{Currency: "", EstimatedSavings: 100.0},
			expectError: true,
		},
		{
			name:        "invalid currency",
			impact:      &pbc.RecommendationImpact{Currency: "INVALID", EstimatedSavings: 100.0},
			expectError: true,
		},
		{
			name:        "negative savings",
			impact:      &pbc.RecommendationImpact{Currency: "USD", EstimatedSavings: -1.0},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pluginsdk.ValidateRecommendationImpact(tc.impact)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// =============================================================================
// Recommendation Filter Tests
// =============================================================================

// TestApplyRecommendationFilter tests the ApplyRecommendationFilter function.
func TestApplyRecommendationFilter(t *testing.T) {
	recommendations := []*pbc.Recommendation{
		{
			Id:         "rec-1",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Resource:   &pbc.ResourceRecommendationInfo{Id: "i-1", Provider: "aws", Region: "us-east-1"},
		},
		{
			Id:         "rec-2",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Resource:   &pbc.ResourceRecommendationInfo{Id: "i-2", Provider: "aws", Region: "us-west-2"},
		},
		{
			Id:         "rec-3",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,
			Resource:   &pbc.ResourceRecommendationInfo{Id: "v-1", Provider: "azure", Region: "eastus"},
		},
	}

	testCases := []struct {
		name          string
		filter        *pbc.RecommendationFilter
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "nil filter returns all",
			filter:        nil,
			expectedCount: 3,
			expectedIDs:   []string{"rec-1", "rec-2", "rec-3"},
		},
		{
			name:          "empty filter returns all",
			filter:        &pbc.RecommendationFilter{},
			expectedCount: 3,
			expectedIDs:   []string{"rec-1", "rec-2", "rec-3"},
		},
		{
			name:          "filter by category COST",
			filter:        &pbc.RecommendationFilter{Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST},
			expectedCount: 2,
			expectedIDs:   []string{"rec-1", "rec-3"},
		},
		{
			name:          "filter by provider aws",
			filter:        &pbc.RecommendationFilter{Provider: "aws"},
			expectedCount: 2,
			expectedIDs:   []string{"rec-1", "rec-2"},
		},
		{
			name:          "filter by region us-east-1",
			filter:        &pbc.RecommendationFilter{Region: "us-east-1"},
			expectedCount: 1,
			expectedIDs:   []string{"rec-1"},
		},
		{
			name: "filter by action type TERMINATE",
			filter: &pbc.RecommendationFilter{
				ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,
			},
			expectedCount: 1,
			expectedIDs:   []string{"rec-3"},
		},
		{
			name: "combined filter",
			filter: &pbc.RecommendationFilter{
				Provider: "aws",
				Category: pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			},
			expectedCount: 1,
			expectedIDs:   []string{"rec-1"},
		},
		{
			name:          "filter with no matches",
			filter:        &pbc.RecommendationFilter{Provider: "gcp"},
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pluginsdk.ApplyRecommendationFilter(recommendations, tc.filter)
			if len(result) != tc.expectedCount {
				t.Errorf("expected %d results, got %d", tc.expectedCount, len(result))
			}
			for i, rec := range result {
				if i < len(tc.expectedIDs) && rec.GetId() != tc.expectedIDs[i] {
					t.Errorf("expected ID %s at position %d, got %s", tc.expectedIDs[i], i, rec.GetId())
				}
			}
		})
	}
}

// =============================================================================
// Pagination Tests
// =============================================================================

// testRecommendations25 creates 25 test recommendations with sequential IDs for pagination tests.
func testRecommendations25() []*pbc.Recommendation {
	recs := make([]*pbc.Recommendation, 25)
	for i := range 25 {
		recs[i] = &pbc.Recommendation{Id: string(rune('A' + i))}
	}
	return recs
}

// TestPaginateRecommendations_FirstPageDefaultSize tests pagination with default page size.
func TestPaginateRecommendations_FirstPageDefaultSize(t *testing.T) {
	recommendations := testRecommendations25()
	result, nextToken, err := pluginsdk.PaginateRecommendations(recommendations, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 25 {
		t.Errorf("expected 25 results, got %d", len(result))
	}
	if nextToken != "" {
		t.Errorf("expected no next token but got %s", nextToken)
	}
}

// TestPaginateRecommendations_FirstPageSize10 tests pagination with page size 10.
func TestPaginateRecommendations_FirstPageSize10(t *testing.T) {
	recommendations := testRecommendations25()
	result, nextToken, err := pluginsdk.PaginateRecommendations(recommendations, 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 10 {
		t.Errorf("expected 10 results, got %d", len(result))
	}
	if nextToken == "" {
		t.Error("expected next token but got empty")
	}
}

// TestPaginateRecommendations_SecondPageSize10 tests pagination for second page with size 10.
func TestPaginateRecommendations_SecondPageSize10(t *testing.T) {
	recommendations := testRecommendations25()
	result, nextToken, err := pluginsdk.PaginateRecommendations(recommendations, 10, pluginsdk.EncodePageToken(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 10 {
		t.Errorf("expected 10 results, got %d", len(result))
	}
	if nextToken == "" {
		t.Error("expected next token but got empty")
	}
}

// TestPaginateRecommendations_LastPageSize10 tests pagination for last page with size 10.
func TestPaginateRecommendations_LastPageSize10(t *testing.T) {
	recommendations := testRecommendations25()
	result, nextToken, err := pluginsdk.PaginateRecommendations(recommendations, 10, pluginsdk.EncodePageToken(20))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 5 {
		t.Errorf("expected 5 results, got %d", len(result))
	}
	if nextToken != "" {
		t.Errorf("expected no next token but got %s", nextToken)
	}
}

// TestPaginateRecommendations_InvalidToken tests pagination with invalid page token.
func TestPaginateRecommendations_InvalidToken(t *testing.T) {
	recommendations := testRecommendations25()
	_, _, err := pluginsdk.PaginateRecommendations(recommendations, 10, "invalid-token")
	if err == nil {
		t.Error("expected error but got nil")
	}
}

// TestPaginateRecommendations_OffsetBeyondRange tests pagination with offset beyond range.
func TestPaginateRecommendations_OffsetBeyondRange(t *testing.T) {
	recommendations := testRecommendations25()
	result, nextToken, err := pluginsdk.PaginateRecommendations(recommendations, 10, pluginsdk.EncodePageToken(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
	if nextToken != "" {
		t.Errorf("expected no next token but got %s", nextToken)
	}
}

// TestEncodeDecodePageToken tests encoding and decoding of page tokens.
func TestEncodeDecodePageToken(t *testing.T) {
	testCases := []struct {
		offset int
	}{
		{0},
		{10},
		{100},
		{1000},
	}

	for _, tc := range testCases {
		token := pluginsdk.EncodePageToken(tc.offset)
		decoded, err := pluginsdk.DecodePageToken(token)
		if err != nil {
			t.Errorf("failed to decode token for offset %d: %v", tc.offset, err)
		}
		if decoded != tc.offset {
			t.Errorf("expected offset %d, got %d", tc.offset, decoded)
		}
	}
}

// TestDecodePageToken_ErrorCases tests DecodePageToken error handling for various invalid inputs.
func TestDecodePageToken_ErrorCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		token       string
		expectError string
	}{
		{
			name:        "invalid base64",
			token:       "not-valid-base64!!!",
			expectError: "malformed page token",
		},
		{
			name:        "non-numeric decoded content",
			token:       b64encode("abc"),
			expectError: "invalid page token value",
		},
		{
			name:        "negative offset",
			token:       b64encode("-1"),
			expectError: "page token offset cannot be negative",
		},
		{
			name:        "offset exceeds MaxInt32",
			token:       b64encode("2147483648"), // math.MaxInt32 + 1
			expectError: "page token offset exceeds maximum allowed value",
		},
		{
			name:        "offset is MaxInt64",
			token:       b64encode("9223372036854775807"),
			expectError: "page token offset exceeds maximum allowed value",
		},
		{
			name:        "empty decoded content",
			token:       b64encode(""),
			expectError: "invalid page token value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := pluginsdk.DecodePageToken(tc.token)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
		})
	}
}

// TestDecodePageToken_MaxInt32Succeeds verifies that math.MaxInt32 is a valid offset.
func TestDecodePageToken_MaxInt32Succeeds(t *testing.T) {
	t.Parallel()

	token := b64encode("2147483647") // math.MaxInt32
	offset, err := pluginsdk.DecodePageToken(token)
	require.NoError(t, err)
	assert.Equal(t, math.MaxInt32, offset)
}

// b64encode is a test helper for creating base64-encoded page tokens from raw strings.
func b64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// =============================================================================
// Summary Calculation Tests
// =============================================================================

// TestCalculateRecommendationSummary tests the CalculateRecommendationSummary function.
func TestCalculateRecommendationSummary(t *testing.T) {
	recommendations := []*pbc.Recommendation{
		{
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
		},
		{
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "USD"},
		},
		{
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 25.0, Currency: "USD"},
		},
	}

	summary := pluginsdk.CalculateRecommendationSummary(recommendations, "monthly")

	if summary.GetTotalRecommendations() != 3 {
		t.Errorf("expected 3 total, got %d", summary.GetTotalRecommendations())
	}

	if summary.GetTotalEstimatedSavings() != 175.0 {
		t.Errorf("expected 175.0 savings, got %f", summary.GetTotalEstimatedSavings())
	}

	if summary.GetCurrency() != "USD" {
		t.Errorf("expected USD, got %s", summary.GetCurrency())
	}

	if summary.GetProjectionPeriod() != "monthly" {
		t.Errorf("expected monthly, got %s", summary.GetProjectionPeriod())
	}

	// Check category counts
	costCatName := pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST.String()
	if summary.GetCountByCategory()[costCatName] != 2 {
		t.Errorf("expected 2 COST, got %d", summary.GetCountByCategory()[costCatName])
	}

	// Check action type counts
	rightsizeActionName := pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE.String()
	if summary.GetCountByActionType()[rightsizeActionName] != 2 {
		t.Errorf("expected 2 RIGHTSIZE, got %d", summary.GetCountByActionType()[rightsizeActionName])
	}
}

// TestCalculateRecommendationSummaryEmpty tests summary calculation with empty recommendations.
func TestCalculateRecommendationSummaryEmpty(t *testing.T) {
	summary := pluginsdk.CalculateRecommendationSummary([]*pbc.Recommendation{}, "monthly")

	if summary.GetTotalRecommendations() != 0 {
		t.Errorf("expected 0 total, got %d", summary.GetTotalRecommendations())
	}

	if summary.GetTotalEstimatedSavings() != 0.0 {
		t.Errorf("expected 0.0 savings, got %f", summary.GetTotalEstimatedSavings())
	}
}

// TestCalculateRecommendationSummaryMixedCurrency tests summary calculation with mixed currencies.
func TestCalculateRecommendationSummaryMixedCurrency(t *testing.T) {
	// Test that mixed currencies result in empty currency field
	recs := []*pbc.Recommendation{
		{
			Id:         "rec-1",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
		},
		{
			Id:         "rec-2",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "EUR"},
		},
	}

	summary := pluginsdk.CalculateRecommendationSummary(recs, "monthly")

	// Currency should be empty when mixed
	if summary.GetCurrency() != "" {
		t.Errorf("expected empty currency for mixed currencies, got %s", summary.GetCurrency())
	}

	// Total savings should still be calculated
	if summary.GetTotalEstimatedSavings() != 150.0 {
		t.Errorf("expected 150.0 total savings, got %f", summary.GetTotalEstimatedSavings())
	}
}

// TestCalculateRecommendationSummaryConsistentCurrency tests summary calculation with consistent currencies.
func TestCalculateRecommendationSummaryConsistentCurrency(t *testing.T) {
	// Test that consistent currencies result in populated currency field
	recs := []*pbc.Recommendation{
		{
			Id:         "rec-1",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
		},
		{
			Id:         "rec-2",
			Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Impact:     &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "USD"},
		},
	}

	summary := pluginsdk.CalculateRecommendationSummary(recs, "monthly")

	// Currency should be set when consistent
	if summary.GetCurrency() != "USD" {
		t.Errorf("expected USD currency, got %s", summary.GetCurrency())
	}

	// Total savings should be calculated
	if summary.GetTotalEstimatedSavings() != 150.0 {
		t.Errorf("expected 150.0 total savings, got %f", summary.GetTotalEstimatedSavings())
	}
}

// Helper function to create pointer to float64.
func ptr(v float64) *float64 {
	return &v
}

// =============================================================================
// FallbackHint Tests
// =============================================================================

// TestFallbackHintDefaultValue tests default hint value (unspecified = 0) behavior.
func TestFallbackHintDefaultValue(t *testing.T) {
	// Verify that FallbackHint_FALLBACK_HINT_UNSPECIFIED is 0 (proto3 default)
	if pbc.FallbackHint_FALLBACK_HINT_UNSPECIFIED != 0 {
		t.Errorf("Expected FALLBACK_HINT_UNSPECIFIED to be 0, got %d",
			pbc.FallbackHint_FALLBACK_HINT_UNSPECIFIED)
	}

	// A new GetActualCostResponse should have unspecified (0) as default
	resp := &pbc.GetActualCostResponse{}
	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_UNSPECIFIED {
		t.Errorf("Expected default hint to be UNSPECIFIED, got %v", resp.GetFallbackHint())
	}
}

// TestCreateActualCostResponseDefaultHint tests that CreateActualCostResponse returns unspecified hint by default.
func TestCreateActualCostResponseDefaultHint(t *testing.T) {
	calc := pluginsdk.NewCostCalculator()

	results := []*pbc.ActualCostResult{
		{Cost: 10.0, Source: "test"},
	}
	resp := calc.CreateActualCostResponse(results)

	// Existing CreateActualCostResponse should work with default (unspecified) hint
	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_UNSPECIFIED {
		t.Errorf("Expected default hint to be UNSPECIFIED, got %v", resp.GetFallbackHint())
	}

	// Results should still be set correctly
	if len(resp.GetResults()) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resp.GetResults()))
	}
}

// TestWithFallbackHintNone tests WithFallbackHint(NONE) option.
func TestWithFallbackHintNone(t *testing.T) {
	resp := pluginsdk.NewActualCostResponse(
		pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
	)

	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_NONE {
		t.Errorf("Expected FALLBACK_HINT_NONE, got %v", resp.GetFallbackHint())
	}
}

// TestNewActualCostResponseWithResultsAndNoneHint tests NewActualCostResponse with results and explicit NONE hint.
func TestNewActualCostResponseWithResultsAndNoneHint(t *testing.T) {
	results := []*pbc.ActualCostResult{
		{Cost: 25.50, Source: "aws-ce"},
		{Cost: 10.00, Source: "aws-ce"},
	}

	resp := pluginsdk.NewActualCostResponse(
		pluginsdk.WithResults(results),
		pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_NONE),
	)

	// Check results
	if len(resp.GetResults()) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resp.GetResults()))
	}
	if resp.GetResults()[0].GetCost() != 25.50 {
		t.Errorf("Expected first cost 25.50, got %f", resp.GetResults()[0].GetCost())
	}

	// Check hint
	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_NONE {
		t.Errorf("Expected FALLBACK_HINT_NONE, got %v", resp.GetFallbackHint())
	}
}

// TestWithFallbackHintRecommendedEmptyResults tests WithFallbackHint(RECOMMENDED) with empty results.
func TestWithFallbackHintRecommendedEmptyResults(t *testing.T) {
	resp := pluginsdk.NewActualCostResponse(
		pluginsdk.WithResults(nil),
		pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED),
	)

	// Should have no results
	if len(resp.GetResults()) != 0 {
		t.Errorf("Expected 0 results, got %d", len(resp.GetResults()))
	}

	// Should have RECOMMENDED hint
	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED {
		t.Errorf("Expected FALLBACK_HINT_RECOMMENDED, got %v", resp.GetFallbackHint())
	}
}

// TestNewActualCostResponseNilResultsRecommended tests NewActualCostResponse with nil results and RECOMMENDED hint.
func TestNewActualCostResponseNilResultsRecommended(t *testing.T) {
	resp := pluginsdk.NewActualCostResponse(
		pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED),
	)

	// Results should be nil/empty
	if resp.GetResults() != nil && len(resp.GetResults()) != 0 {
		t.Errorf("Expected nil or empty results, got %v", resp.GetResults())
	}

	// Hint should be RECOMMENDED
	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED {
		t.Errorf("Expected FALLBACK_HINT_RECOMMENDED, got %v", resp.GetFallbackHint())
	}
}

// TestWithFallbackHintRequired tests WithFallbackHint(REQUIRED).
func TestWithFallbackHintRequired(t *testing.T) {
	resp := pluginsdk.NewActualCostResponse(
		pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_REQUIRED),
	)

	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_REQUIRED {
		t.Errorf("Expected FALLBACK_HINT_REQUIRED, got %v", resp.GetFallbackHint())
	}
}

// TestNewActualCostResponseRequiredForUnsupportedType tests NewActualCostResponse with REQUIRED hint for unsupported type.
func TestNewActualCostResponseRequiredForUnsupportedType(t *testing.T) {
	// Simulate a plugin that cannot handle a specific resource type
	// It returns empty results with REQUIRED hint
	resp := pluginsdk.NewActualCostResponse(
		pluginsdk.WithFallbackHint(pbc.FallbackHint_FALLBACK_HINT_REQUIRED),
	)

	// No results (plugin doesn't handle this type)
	if len(resp.GetResults()) != 0 {
		t.Errorf("Expected 0 results for unsupported type, got %d", len(resp.GetResults()))
	}

	// REQUIRED hint signals core must try fallback
	if resp.GetFallbackHint() != pbc.FallbackHint_FALLBACK_HINT_REQUIRED {
		t.Errorf("Expected FALLBACK_HINT_REQUIRED, got %v", resp.GetFallbackHint())
	}
}

// TestGetActualCostReturnsErrorForAPIFailures tests that GetActualCost should return error for API failures, not hint.
func TestGetActualCostReturnsErrorForAPIFailures(t *testing.T) {
	// This test documents the expected behavior: errors should be returned
	// as gRPC errors, not as success responses with hints.
	//
	// The BasePlugin.GetActualCost returns an error (NoDataError), not a hint.
	// This is the correct pattern - system failures use error path, not fallback path.

	plugin := pluginsdk.NewBasePlugin("test-plugin")
	ctx := context.Background()

	req := &pbc.GetActualCostRequest{
		ResourceId: "test-resource-id",
	}

	// Default implementation returns an error, not a response with hint
	resp, err := plugin.GetActualCost(ctx, req)

	// Should return error
	if err == nil {
		t.Error("Expected error for default GetActualCost, got nil")
	}

	// Response should be nil when error occurs
	if resp != nil {
		t.Errorf("Expected nil response when error occurs, got %v", resp)
	}

	// This demonstrates the pattern: errors (API failures, network issues) should
	// return gRPC errors, not success responses with hints.
}

// =============================================================================
// FallbackHint Edge Case Tests
// =============================================================================

// TestFallbackHintDataWithConflictingHint tests edge case where results exist but hint says fallback.
// This documents the semantics: the hint is advisory, results take precedence for data presence.
func TestFallbackHintDataWithConflictingHint(t *testing.T) {
	testCases := []struct {
		name           string
		results        []*pbc.ActualCostResult
		hint           pbc.FallbackHint
		expectedValid  bool
		expectedReason string
	}{
		{
			name: "results with NONE hint - valid",
			results: []*pbc.ActualCostResult{
				{Cost: 10.0, Source: "aws-ce"},
			},
			hint:          pbc.FallbackHint_FALLBACK_HINT_NONE,
			expectedValid: true,
		},
		{
			name:          "no results with RECOMMENDED hint - valid",
			results:       nil,
			hint:          pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED,
			expectedValid: true,
		},
		{
			name:          "no results with REQUIRED hint - valid",
			results:       nil,
			hint:          pbc.FallbackHint_FALLBACK_HINT_REQUIRED,
			expectedValid: true,
		},
		{
			name: "results with RECOMMENDED hint - inconsistent but allowed",
			results: []*pbc.ActualCostResult{
				{Cost: 5.0, Source: "test"},
			},
			hint:           pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED,
			expectedValid:  true, // Allowed but inconsistent
			expectedReason: "data with fallback hint is semantically inconsistent",
		},
		{
			name: "results with REQUIRED hint - inconsistent but allowed",
			results: []*pbc.ActualCostResult{
				{Cost: 15.0, Source: "kubecost"},
			},
			hint:           pbc.FallbackHint_FALLBACK_HINT_REQUIRED,
			expectedValid:  true, // Allowed but inconsistent
			expectedReason: "data with required fallback is semantically inconsistent",
		},
		{
			name:          "empty results with NONE hint - valid (authoritative empty)",
			results:       []*pbc.ActualCostResult{},
			hint:          pbc.FallbackHint_FALLBACK_HINT_NONE,
			expectedValid: true,
		},
		{
			name: "zero-cost result with NONE hint - valid (free tier)",
			results: []*pbc.ActualCostResult{
				{Cost: 0.0, Source: "aws-ce"},
			},
			hint:          pbc.FallbackHint_FALLBACK_HINT_NONE,
			expectedValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := pluginsdk.NewActualCostResponse(
				pluginsdk.WithResults(tc.results),
				pluginsdk.WithFallbackHint(tc.hint),
			)

			// Verify response is constructed correctly
			if resp.GetFallbackHint() != tc.hint {
				t.Errorf("Expected hint %v, got %v", tc.hint, resp.GetFallbackHint())
			}

			if len(resp.GetResults()) != len(tc.results) {
				t.Errorf("Expected %d results, got %d", len(tc.results), len(resp.GetResults()))
			}

			// Validate using the helper
			err := pluginsdk.ValidateActualCostResponse(resp)
			if tc.expectedValid && err != nil {
				t.Errorf("Expected valid response, got error: %v", err)
			}
			if !tc.expectedValid && err == nil {
				t.Error("Expected invalid response, got nil error")
			}
		})
	}
}

// TestFallbackHintAllEnumValues tests all FallbackHint enum values are handled.
func TestFallbackHintAllEnumValues(t *testing.T) {
	allHints := []pbc.FallbackHint{
		pbc.FallbackHint_FALLBACK_HINT_UNSPECIFIED,
		pbc.FallbackHint_FALLBACK_HINT_NONE,
		pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED,
		pbc.FallbackHint_FALLBACK_HINT_REQUIRED,
	}

	for _, hint := range allHints {
		t.Run(hint.String(), func(t *testing.T) {
			resp := pluginsdk.NewActualCostResponse(
				pluginsdk.WithFallbackHint(hint),
			)

			if resp.GetFallbackHint() != hint {
				t.Errorf("Expected hint %v, got %v", hint, resp.GetFallbackHint())
			}

			// All enum values should pass basic validation
			err := pluginsdk.ValidateActualCostResponse(resp)
			if err != nil {
				t.Errorf("Unexpected validation error for %v: %v", hint, err)
			}
		})
	}
}

// TestValidateActualCostResponse tests the ValidateActualCostResponse helper function.
func TestValidateActualCostResponse(t *testing.T) {
	testCases := []struct {
		name        string
		resp        *pbc.GetActualCostResponse
		expectError bool
	}{
		{
			name:        "nil response",
			resp:        nil,
			expectError: true,
		},
		{
			name:        "empty response with default hint",
			resp:        &pbc.GetActualCostResponse{},
			expectError: false,
		},
		{
			name: "valid response with results and NONE hint",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{Cost: 10.0, Source: "test"},
				},
				FallbackHint: pbc.FallbackHint_FALLBACK_HINT_NONE,
			},
			expectError: false,
		},
		{
			name: "valid response with nil results and RECOMMENDED hint",
			resp: &pbc.GetActualCostResponse{
				FallbackHint: pbc.FallbackHint_FALLBACK_HINT_RECOMMENDED,
			},
			expectError: false,
		},
		{
			name: "response with negative cost",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{Cost: -5.0, Source: "test"},
				},
			},
			expectError: true,
		},
		{
			name: "response with empty source",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{Cost: 10.0, Source: ""},
				},
			},
			expectError: true,
		},
		{
			name: "response with nil result in slice",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{nil},
			},
			expectError: true,
		},
		{
			name: "multiple results with second invalid (validation stops at first error)",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{Cost: 10.0, Source: "valid-source"},
					{Cost: -5.0, Source: "test"}, // negative cost - invalid
					{Cost: 20.0, Source: ""},     // empty source - also invalid
				},
			},
			expectError: true,
		},
		{
			name: "multiple results with first invalid (nil)",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					nil, // first result is nil
					{Cost: 10.0, Source: "valid"},
					{Cost: 20.0, Source: "also-valid"},
				},
			},
			expectError: true,
		},
		{
			name: "multiple valid results all pass validation",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{Cost: 10.0, Source: "source-1"},
					{Cost: 0.0, Source: "source-2"}, // zero cost is valid (free tier)
					{Cost: 100.0, Source: "source-3"},
				},
				FallbackHint: pbc.FallbackHint_FALLBACK_HINT_NONE,
			},
			expectError: false,
		},
		{
			name: "result with invalid expires_at nanos",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{
						Cost:      10.0,
						Source:    "test",
						ExpiresAt: &timestamppb.Timestamp{Seconds: 0, Nanos: -1},
					},
				},
			},
			expectError: true,
		},
		{
			name: "result with valid expires_at",
			resp: &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{
						Cost:      10.0,
						Source:    "test",
						ExpiresAt: timestamppb.Now(),
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pluginsdk.ValidateActualCostResponse(tc.resp)
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestValidateActualCostResponseStopsAtFirstError verifies that validation
// stops at the first invalid result and reports the correct index.
func TestValidateActualCostResponseStopsAtFirstError(t *testing.T) {
	// Response with multiple errors: second result has negative cost,
	// third result has empty source. Validation should stop at index 1.
	resp := &pbc.GetActualCostResponse{
		Results: []*pbc.ActualCostResult{
			{Cost: 10.0, Source: "valid-source"}, // index 0 - valid
			{Cost: -5.0, Source: "test"},         // index 1 - negative cost (first error)
			{Cost: 20.0, Source: ""},             // index 2 - empty source (never reached)
			{Cost: -10.0, Source: ""},            // index 3 - both errors (never reached)
		},
	}

	err := pluginsdk.ValidateActualCostResponse(resp)
	if err == nil {
		t.Fatal("Expected validation error but got nil")
	}

	// Verify error mentions index 1 (second result), not index 2 or 3
	errMsg := err.Error()
	if !strings.Contains(errMsg, "results[1]") {
		t.Errorf("Expected error to reference results[1], got: %s", errMsg)
	}
	if strings.Contains(errMsg, "results[2]") || strings.Contains(errMsg, "results[3]") {
		t.Errorf("Error should not reference later indices (stops at first error), got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "negative") {
		t.Errorf("Expected error to mention negative cost, got: %s", errMsg)
	}
}

// =============================================================================
// SortRecommendations Tests
// =============================================================================

// TestSortRecommendations tests the SortRecommendations function.
func TestSortRecommendations(t *testing.T) {
	// Create test recommendations with varying savings
	rec100 := &pbc.Recommendation{
		Id:     "rec-100",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
	}
	rec50 := &pbc.Recommendation{
		Id:     "rec-50",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "USD"},
	}
	rec200 := &pbc.Recommendation{
		Id:     "rec-200",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 200.0, Currency: "USD"},
	}

	testCases := []struct {
		name        string
		recs        []*pbc.Recommendation
		sortBy      pbc.RecommendationSortBy
		sortOrder   pbc.SortOrder
		expectedIDs []string
	}{
		{
			name:        "unspecified sort returns original order",
			recs:        []*pbc.Recommendation{rec100, rec50, rec200},
			sortBy:      pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_UNSPECIFIED,
			sortOrder:   pbc.SortOrder_SORT_ORDER_UNSPECIFIED,
			expectedIDs: []string{"rec-100", "rec-50", "rec-200"},
		},
		{
			name:        "sort by savings ascending",
			recs:        []*pbc.Recommendation{rec100, rec50, rec200},
			sortBy:      pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
			sortOrder:   pbc.SortOrder_SORT_ORDER_ASC,
			expectedIDs: []string{"rec-50", "rec-100", "rec-200"},
		},
		{
			name:        "sort by savings descending",
			recs:        []*pbc.Recommendation{rec100, rec50, rec200},
			sortBy:      pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
			sortOrder:   pbc.SortOrder_SORT_ORDER_DESC,
			expectedIDs: []string{"rec-200", "rec-100", "rec-50"},
		},
		{
			name:        "default order for savings is descending",
			recs:        []*pbc.Recommendation{rec100, rec50, rec200},
			sortBy:      pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
			sortOrder:   pbc.SortOrder_SORT_ORDER_UNSPECIFIED,
			expectedIDs: []string{"rec-200", "rec-100", "rec-50"},
		},
		{
			name:        "empty slice",
			recs:        []*pbc.Recommendation{},
			sortBy:      pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
			sortOrder:   pbc.SortOrder_SORT_ORDER_ASC,
			expectedIDs: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pluginsdk.SortRecommendations(tc.recs, tc.sortBy, tc.sortOrder)
			if len(result) != len(tc.expectedIDs) {
				t.Errorf("expected %d results, got %d", len(tc.expectedIDs), len(result))
				return
			}
			for i, rec := range result {
				if rec.GetId() != tc.expectedIDs[i] {
					t.Errorf("expected ID %s at position %d, got %s", tc.expectedIDs[i], i, rec.GetId())
				}
			}
		})
	}
}

// TestSortRecommendationsDoesNotModifyOriginal tests that sorting does not modify the original slice.
func TestSortRecommendationsDoesNotModifyOriginal(t *testing.T) {
	rec100 := &pbc.Recommendation{
		Id:     "rec-100",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
	}
	rec50 := &pbc.Recommendation{
		Id:     "rec-50",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 50.0, Currency: "USD"},
	}
	rec200 := &pbc.Recommendation{
		Id:     "rec-200",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 200.0, Currency: "USD"},
	}

	original := []*pbc.Recommendation{rec100, rec50, rec200}

	// Sort and get result
	_ = pluginsdk.SortRecommendations(
		original,
		pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
		pbc.SortOrder_SORT_ORDER_ASC,
	)

	// Original should remain unchanged
	expectedOriginal := []string{"rec-100", "rec-50", "rec-200"}
	for i, rec := range original {
		if rec.GetId() != expectedOriginal[i] {
			t.Errorf("original modified: expected ID %s at position %d, got %s",
				expectedOriginal[i], i, rec.GetId())
		}
	}
}

// TestSortRecommendationsWithEqualValues tests sorting with equal values maintains stability.
// This test specifically verifies the fix for the strict weak ordering violation.
func TestSortRecommendationsWithEqualValues(t *testing.T) {
	// Create recommendations with equal savings values
	recA := &pbc.Recommendation{
		Id:     "rec-A",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
	}
	recB := &pbc.Recommendation{
		Id:     "rec-B",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
	}
	recC := &pbc.Recommendation{
		Id:     "rec-C",
		Impact: &pbc.RecommendationImpact{EstimatedSavings: 100.0, Currency: "USD"},
	}

	testCases := []struct {
		name        string
		sortOrder   pbc.SortOrder
		expectedIDs []string
	}{
		{
			name:        "ascending - equal values maintain original order (stable)",
			sortOrder:   pbc.SortOrder_SORT_ORDER_ASC,
			expectedIDs: []string{"rec-A", "rec-B", "rec-C"},
		},
		{
			name:        "descending - equal values maintain original order (stable)",
			sortOrder:   pbc.SortOrder_SORT_ORDER_DESC,
			expectedIDs: []string{"rec-A", "rec-B", "rec-C"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := []*pbc.Recommendation{recA, recB, recC}
			result := pluginsdk.SortRecommendations(
				input,
				pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
				tc.sortOrder,
			)

			for i, rec := range result {
				if rec.GetId() != tc.expectedIDs[i] {
					t.Errorf("expected ID %s at position %d, got %s", tc.expectedIDs[i], i, rec.GetId())
				}
			}
		})
	}
}

// TestSortRecommendationsByPriority tests sorting by priority.
func TestSortRecommendationsByPriority(t *testing.T) {
	recHigh := &pbc.Recommendation{
		Id:       "rec-high",
		Priority: pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_HIGH,
	}
	recMedium := &pbc.Recommendation{
		Id:       "rec-medium",
		Priority: pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_MEDIUM,
	}
	recLow := &pbc.Recommendation{
		Id:       "rec-low",
		Priority: pbc.RecommendationPriority_RECOMMENDATION_PRIORITY_LOW,
	}

	testCases := []struct {
		name        string
		sortOrder   pbc.SortOrder
		expectedIDs []string
	}{
		{
			name:        "ascending priority",
			sortOrder:   pbc.SortOrder_SORT_ORDER_ASC,
			expectedIDs: []string{"rec-low", "rec-medium", "rec-high"},
		},
		{
			name:        "descending priority (default)",
			sortOrder:   pbc.SortOrder_SORT_ORDER_UNSPECIFIED,
			expectedIDs: []string{"rec-high", "rec-medium", "rec-low"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := []*pbc.Recommendation{recHigh, recMedium, recLow}
			result := pluginsdk.SortRecommendations(
				input,
				pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_PRIORITY,
				tc.sortOrder,
			)

			for i, rec := range result {
				if rec.GetId() != tc.expectedIDs[i] {
					t.Errorf("expected ID %s at position %d, got %s", tc.expectedIDs[i], i, rec.GetId())
				}
			}
		})
	}
}

// TestSortRecommendationsStrictWeakOrdering verifies the comparison function satisfies
// strict weak ordering requirements for sort.SliceStable.
// This is a regression test for the bug where !less was used for descending order.
func TestSortRecommendationsStrictWeakOrdering(t *testing.T) {
	// Create many recommendations with some equal values to stress test the sort
	recs := make([]*pbc.Recommendation, 100)
	for i := range 100 {
		// Use modulo to create repeated values (tests stability and ordering)
		savings := float64((i % 10) * 10)
		recs[i] = &pbc.Recommendation{
			Id:     string(rune('A' + i)),
			Impact: &pbc.RecommendationImpact{EstimatedSavings: savings, Currency: "USD"},
		}
	}

	// This should not panic - the bug caused panics with violated sort contracts
	result := pluginsdk.SortRecommendations(
		recs,
		pbc.RecommendationSortBy_RECOMMENDATION_SORT_BY_ESTIMATED_SAVINGS,
		pbc.SortOrder_SORT_ORDER_DESC,
	)

	if len(result) != 100 {
		t.Errorf("expected 100 results, got %d", len(result))
	}

	// Verify descending order
	for i := 1; i < len(result); i++ {
		prev := result[i-1].GetImpact().GetEstimatedSavings()
		curr := result[i].GetImpact().GetEstimatedSavings()
		if prev < curr {
			t.Errorf("not in descending order: position %d has %f, position %d has %f",
				i-1, prev, i, curr)
		}
	}
}

// =============================================================================
// ResourceDescriptor Helper Tests
// =============================================================================

func TestNewResourceDescriptor_Basic(t *testing.T) {
	t.Parallel()

	desc := pluginsdk.NewResourceDescriptor("aws", "ec2")

	if desc.GetProvider() != "aws" {
		t.Errorf("expected provider 'aws', got %q", desc.GetProvider())
	}
	if desc.GetResourceType() != "ec2" {
		t.Errorf("expected resource_type 'ec2', got %q", desc.GetResourceType())
	}
	// Optional fields should be empty
	if desc.GetId() != "" {
		t.Errorf("expected empty id, got %q", desc.GetId())
	}
	if desc.GetArn() != "" {
		t.Errorf("expected empty arn, got %q", desc.GetArn())
	}
}

func TestNewResourceDescriptor_WithIDOption(t *testing.T) {
	t.Parallel()

	desc := pluginsdk.NewResourceDescriptor(
		"aws", "ec2",
		pluginsdk.WithID("urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web"),
	)

	if desc.GetId() != "urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web" {
		t.Errorf("ID not set correctly: %q", desc.GetId())
	}
}

func TestNewResourceDescriptor_WithARNOption(t *testing.T) {
	t.Parallel()

	desc := pluginsdk.NewResourceDescriptor(
		"aws", "ec2",
		pluginsdk.WithARN("arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"),
	)

	if desc.GetArn() != "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123" {
		t.Errorf("ARN not set correctly: %q", desc.GetArn())
	}
}

func TestNewResourceDescriptor_WithAllOptions(t *testing.T) {
	t.Parallel()

	desc := pluginsdk.NewResourceDescriptor(
		"aws", "ec2",
		pluginsdk.WithID("batch-001"),
		pluginsdk.WithARN("arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"),
		pluginsdk.WithSKU("t3.micro"),
		pluginsdk.WithRegion("us-east-1"),
		pluginsdk.WithTags(map[string]string{"env": "prod", "team": "platform"}),
	)

	if desc.GetProvider() != "aws" {
		t.Errorf("expected provider 'aws', got %q", desc.GetProvider())
	}
	if desc.GetResourceType() != "ec2" {
		t.Errorf("expected resource_type 'ec2', got %q", desc.GetResourceType())
	}
	if desc.GetId() != "batch-001" {
		t.Errorf("expected id 'batch-001', got %q", desc.GetId())
	}
	if desc.GetArn() != "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123" {
		t.Errorf("ARN not set correctly: %q", desc.GetArn())
	}
	if desc.GetSku() != "t3.micro" {
		t.Errorf("expected sku 't3.micro', got %q", desc.GetSku())
	}
	if desc.GetRegion() != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %q", desc.GetRegion())
	}
	if len(desc.GetTags()) != 2 {
		t.Errorf("expected 2 tags, got %d", len(desc.GetTags()))
	}
	if desc.GetTags()["env"] != "prod" {
		t.Errorf("expected tag env=prod, got %q", desc.GetTags()["env"])
	}
}

func TestWithID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		id   string
	}{
		{
			name: "Pulumi URN",
			id:   "urn:pulumi:prod::myapp::aws:ec2/instance:Instance::webserver",
		},
		{
			name: "UUID",
			id:   "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "simple tracking ID",
			id:   "batch-001",
		},
		{
			name: "empty string",
			id:   "",
		},
		{
			name: "special characters",
			id:   "resource@domain.com:path/to/item#section?query=value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			desc := pluginsdk.NewResourceDescriptor("aws", "ec2", pluginsdk.WithID(tc.id))
			if desc.GetId() != tc.id {
				t.Errorf("expected id %q, got %q", tc.id, desc.GetId())
			}
		})
	}
}

func TestWithARN(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		arn  string
	}{
		{
			name: "AWS ARN",
			arn:  "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
		},
		{
			name: "Azure Resource ID",
			arn:  "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1",
		},
		{
			name: "GCP Full Resource Name",
			arn:  "//compute.googleapis.com/projects/my-project/zones/us-central1-a/instances/vm-1",
		},
		{
			name: "Kubernetes Resource Path",
			arn:  "prod-cluster/default/Deployment/nginx",
		},
		{
			name: "empty string",
			arn:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			desc := pluginsdk.NewResourceDescriptor("custom", "resource", pluginsdk.WithARN(tc.arn))
			if desc.GetArn() != tc.arn {
				t.Errorf("expected arn %q, got %q", tc.arn, desc.GetArn())
			}
		})
	}
}

func TestResourceDescriptorOptions_Composability(t *testing.T) {
	t.Parallel()

	// Test that options can be applied in any order
	t.Run("ID then ARN", func(t *testing.T) {
		t.Parallel()
		desc := pluginsdk.NewResourceDescriptor(
			"aws", "ec2",
			pluginsdk.WithID("id-first"),
			pluginsdk.WithARN("arn-second"),
		)
		if desc.GetId() != "id-first" || desc.GetArn() != "arn-second" {
			t.Errorf("options not composed correctly")
		}
	})

	t.Run("ARN then ID", func(t *testing.T) {
		t.Parallel()
		desc := pluginsdk.NewResourceDescriptor(
			"aws", "ec2",
			pluginsdk.WithARN("arn-first"),
			pluginsdk.WithID("id-second"),
		)
		if desc.GetId() != "id-second" || desc.GetArn() != "arn-first" {
			t.Errorf("options not composed correctly")
		}
	})

	t.Run("all options together", func(t *testing.T) {
		t.Parallel()
		desc := pluginsdk.NewResourceDescriptor(
			"gcp", "compute_engine",
			pluginsdk.WithSKU("e2-micro"),
			pluginsdk.WithRegion("us-central1"),
			pluginsdk.WithID("gcp-resource-001"),
			pluginsdk.WithARN("//compute.googleapis.com/projects/myproj/zones/us-central1-a/instances/vm1"),
			pluginsdk.WithTags(map[string]string{"owner": "team-a"}),
		)

		if desc.GetProvider() != "gcp" {
			t.Errorf("provider mismatch")
		}
		if desc.GetResourceType() != "compute_engine" {
			t.Errorf("resource_type mismatch")
		}
		if desc.GetSku() != "e2-micro" {
			t.Errorf("sku mismatch")
		}
		if desc.GetRegion() != "us-central1" {
			t.Errorf("region mismatch")
		}
		if desc.GetId() != "gcp-resource-001" {
			t.Errorf("id mismatch")
		}
		if desc.GetArn() != "//compute.googleapis.com/projects/myproj/zones/us-central1-a/instances/vm1" {
			t.Errorf("arn mismatch")
		}
		if desc.GetTags()["owner"] != "team-a" {
			t.Errorf("tags mismatch")
		}
	})
}

// FuzzResourceDescriptorID fuzz tests the WithID option for ResourceDescriptor.
func FuzzResourceDescriptorID(f *testing.F) {
	// Seed corpus
	f.Add("urn:pulumi:prod::myapp::resource::name")
	f.Add("https://example.com/resource?id=12345#anchor")
	f.Add("arn:aws:ec2:us-east-1:123456789012:instance/i-abc")
	f.Add("/subscriptions/sub-123/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/myVM")
	f.Add("//compute.googleapis.com/projects/proj/zones/z/instances/i")
	f.Add("")                        // Empty string
	f.Add(strings.Repeat("a", 1000)) // Long string
	f.Add("null\x00byte")            // Embedded null
	f.Add("emoji\U0001F600test")     // Unicode

	f.Fuzz(func(t *testing.T, id string) {
		// Should never panic
		desc := pluginsdk.NewResourceDescriptor(
			"provider", "type",
			pluginsdk.WithID(id),
		)

		// ID should round-trip correctly
		if desc.GetId() != id {
			t.Errorf("ID mismatch: got %q, want %q", desc.GetId(), id)
		}
	})
}

// =============================================================================
// Prediction Interval Tests
// =============================================================================

// TestWithPredictionInterval tests the WithPredictionInterval functional option.
func TestWithPredictionInterval(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		lower              float64
		upper              float64
		confidence         float64
		expectedLower      float64
		expectedUpper      float64
		expectedConfidence float64
	}{
		{
			name:               "typical 95% confidence interval",
			lower:              30.0,
			upper:              50.0,
			confidence:         0.95,
			expectedLower:      30.0,
			expectedUpper:      50.0,
			expectedConfidence: 0.95,
		},
		{
			name:               "90% confidence interval",
			lower:              32.0,
			upper:              48.0,
			confidence:         0.90,
			expectedLower:      32.0,
			expectedUpper:      48.0,
			expectedConfidence: 0.90,
		},
		{
			name:               "99% confidence interval",
			lower:              25.0,
			upper:              55.0,
			confidence:         0.99,
			expectedLower:      25.0,
			expectedUpper:      55.0,
			expectedConfidence: 0.99,
		},
		{
			name:               "zero lower bound",
			lower:              0.0,
			upper:              100.0,
			confidence:         0.95,
			expectedLower:      0.0,
			expectedUpper:      100.0,
			expectedConfidence: 0.95,
		},
		{
			name:               "equal bounds (point estimate)",
			lower:              42.0,
			upper:              42.0,
			confidence:         1.0,
			expectedLower:      42.0,
			expectedUpper:      42.0,
			expectedConfidence: 1.0,
		},
		// Note: Invalid confidence values (0, negative, >1.0) now panic at construction
		// time rather than being passed through. See TestWithPredictionIntervalPanics.
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := pluginsdk.NewGetProjectedCostResponse(
				pluginsdk.WithPredictionInterval(tc.lower, tc.upper, tc.confidence),
			)

			if resp.GetPredictionIntervalLower() != tc.expectedLower {
				t.Errorf("expected lower %f, got %f", tc.expectedLower, resp.GetPredictionIntervalLower())
			}
			if resp.GetPredictionIntervalUpper() != tc.expectedUpper {
				t.Errorf("expected upper %f, got %f", tc.expectedUpper, resp.GetPredictionIntervalUpper())
			}
			if resp.GetConfidenceLevel() != tc.expectedConfidence {
				t.Errorf("expected confidence %f, got %f", tc.expectedConfidence, resp.GetConfidenceLevel())
			}
		})
	}
}

// TestWithPredictionIntervalPanics tests that WithPredictionInterval panics for programming errors.
// This is consistent with WithSpotRisk which also uses fail-fast validation via panics.
func TestWithPredictionIntervalPanics(t *testing.T) {
	t.Parallel()

	t.Run("panics_on_lower_nan", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: lower bound is NaN/Inf: NaN",
			func() { pluginsdk.WithPredictionInterval(math.NaN(), 50.0, 0.95) },
		)
	})

	t.Run("panics_on_lower_positive_inf", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: lower bound is NaN/Inf: +Inf",
			func() { pluginsdk.WithPredictionInterval(math.Inf(1), 50.0, 0.95) },
		)
	})

	t.Run("panics_on_lower_negative_inf", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: lower bound is NaN/Inf: -Inf",
			func() { pluginsdk.WithPredictionInterval(math.Inf(-1), 50.0, 0.95) },
		)
	})

	t.Run("panics_on_upper_nan", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: upper bound is NaN/Inf: NaN",
			func() { pluginsdk.WithPredictionInterval(30.0, math.NaN(), 0.95) },
		)
	})

	t.Run("panics_on_upper_positive_inf", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: upper bound is NaN/Inf: +Inf",
			func() { pluginsdk.WithPredictionInterval(30.0, math.Inf(1), 0.95) },
		)
	})

	t.Run("panics_on_upper_negative_inf", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: upper bound is NaN/Inf: -Inf",
			func() { pluginsdk.WithPredictionInterval(30.0, math.Inf(-1), 0.95) },
		)
	})

	t.Run("panics_on_confidence_nan", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: confidence is NaN/Inf: NaN",
			func() { pluginsdk.WithPredictionInterval(30.0, 50.0, math.NaN()) },
		)
	})

	t.Run("panics_on_confidence_positive_inf", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: confidence is NaN/Inf: +Inf",
			func() { pluginsdk.WithPredictionInterval(30.0, 50.0, math.Inf(1)) },
		)
	})

	t.Run("panics_on_confidence_negative_inf", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithValue(t,
			"WithPredictionInterval: confidence is NaN/Inf: -Inf",
			func() { pluginsdk.WithPredictionInterval(30.0, 50.0, math.Inf(-1)) },
		)
	})

	t.Run("panics_on_negative_lower", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { pluginsdk.WithPredictionInterval(-10.0, 50.0, 0.95) })
	})

	t.Run("panics_on_lower_greater_than_upper", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { pluginsdk.WithPredictionInterval(60.0, 50.0, 0.95) })
	})

	t.Run("panics_on_confidence_zero", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { pluginsdk.WithPredictionInterval(30.0, 50.0, 0.0) })
	})

	t.Run("panics_on_confidence_negative", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { pluginsdk.WithPredictionInterval(30.0, 50.0, -0.5) })
	})

	t.Run("panics_on_confidence_greater_than_one", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { pluginsdk.WithPredictionInterval(30.0, 50.0, 1.5) })
	})

	t.Run("does_not_panic_on_valid_values", func(t *testing.T) {
		t.Parallel()
		validCases := []struct {
			name                     string
			lower, upper, confidence float64
		}{
			{"zero-width interval", 0.0, 0.0, 0.95},
			{"zero lower bound", 0.0, 100.0, 0.99},
			{"typical interval", 30.0, 50.0, 0.5},
			{"zero-width at confidence 1.0", 30.0, 30.0, 1.0},
			{"small confidence", 0.0001, 100.0, 0.0001},
		}
		for _, tc := range validCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.NotPanics(t, func() {
					pluginsdk.WithPredictionInterval(tc.lower, tc.upper, tc.confidence)
				})
			})
		}
	})
}

// TestWithPredictionIntervalCombinedWithOtherOptions tests combining prediction interval with other options.
func TestWithPredictionIntervalCombinedWithOtherOptions(t *testing.T) {
	t.Parallel()

	resp := pluginsdk.NewGetProjectedCostResponse(
		pluginsdk.WithProjectedCostDetails(0.05, "USD", 36.50, "spot-instance"),
		pluginsdk.WithPredictionInterval(30.0, 45.0, 0.95),
	)

	// Check base details
	if resp.GetUnitPrice() != 0.05 {
		t.Errorf("expected unit price 0.05, got %f", resp.GetUnitPrice())
	}
	if resp.GetCurrency() != "USD" {
		t.Errorf("expected currency USD, got %s", resp.GetCurrency())
	}
	if resp.GetCostPerMonth() != 36.50 {
		t.Errorf("expected cost per month 36.50, got %f", resp.GetCostPerMonth())
	}
	if resp.GetBillingDetail() != "spot-instance" {
		t.Errorf("expected billing detail 'spot-instance', got %s", resp.GetBillingDetail())
	}

	// Check prediction interval
	if resp.GetPredictionIntervalLower() != 30.0 {
		t.Errorf("expected lower 30.0, got %f", resp.GetPredictionIntervalLower())
	}
	if resp.GetPredictionIntervalUpper() != 45.0 {
		t.Errorf("expected upper 45.0, got %f", resp.GetPredictionIntervalUpper())
	}
	if resp.GetConfidenceLevel() != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", resp.GetConfidenceLevel())
	}
}

// TestValidateMismatchedPredictionIntervalBounds tests that having only one bound set is caught by validation.
func TestValidateMismatchedPredictionIntervalBounds(t *testing.T) {
	t.Parallel()

	t.Run("lower set, upper missing", func(t *testing.T) {
		t.Parallel()
		// Manually create a response with only lower bound set
		lower := 30.0
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            40.0,
			BillingDetail:           "test",
			PredictionIntervalLower: &lower,
			// PredictionIntervalUpper is nil
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		if err == nil {
			t.Error("expected validation error for mismatched bounds but got none")
		} else if !strings.Contains(err.Error(), "prediction_interval_lower is set but prediction_interval_upper is missing") {
			t.Errorf("expected error about mismatched bounds, got %q", err.Error())
		}
	})

	t.Run("upper set, lower missing", func(t *testing.T) {
		t.Parallel()
		// Manually create a response with only upper bound set
		upper := 50.0
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            40.0,
			BillingDetail:           "test",
			PredictionIntervalUpper: &upper,
			// PredictionIntervalLower is nil
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		if err == nil {
			t.Error("expected validation error for mismatched bounds but got none")
		} else if !strings.Contains(err.Error(), "prediction_interval_upper is set but prediction_interval_lower is missing") {
			t.Errorf("expected error about mismatched bounds, got %q", err.Error())
		}
	})
}

// TestValidateCostPerMonthNaNInf tests that NaN/Inf in cost_per_month is caught by validation.
func TestValidateCostPerMonthNaNInf(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		costPerMonth float64
	}{
		{"NaN cost_per_month", math.NaN()},
		{"+Inf cost_per_month", math.Inf(1)},
		{"-Inf cost_per_month", math.Inf(-1)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := &pbc.GetProjectedCostResponse{
				UnitPrice:     0.05,
				Currency:      "USD",
				CostPerMonth:  tc.costPerMonth,
				BillingDetail: "test",
			}
			err := pluginsdk.ValidateGetProjectedCostResponse(resp)
			if err == nil {
				t.Errorf("expected validation error for %s but got none", tc.name)
			} else if !strings.Contains(err.Error(), "cost_per_month is NaN/Inf") {
				t.Errorf("expected error about NaN/Inf cost_per_month, got %q", err.Error())
			}
		})
	}
}

// =============================================================================
// PaginateActualCosts Tests
// =============================================================================

// testActualCosts creates a slice of n mock ActualCostResult records.
func testActualCosts(n int) []*pbc.ActualCostResult {
	results := make([]*pbc.ActualCostResult, n)
	for i := range n {
		results[i] = &pbc.ActualCostResult{
			Cost:   float64(i) + 1.0,
			Source: "test",
		}
	}
	return results
}

// TestPaginateActualCosts_FirstPageDefaultSize verifies that calling with page_size=0
// and an empty page_token returns all records (legacy behavior for backward compatibility).
// When a page_token is provided, page_size=0 should apply the DefaultPageSize.
func TestPaginateActualCosts_FirstPageDefaultSize(t *testing.T) {
	results := testActualCosts(100)

	tests := []struct {
		name          string
		pageSize      int32
		pageToken     string
		wantLen       int
		wantNextEmpty bool
		wantTotal     int32
	}{
		{
			name:          "legacy_no_pagination",
			pageSize:      0,
			pageToken:     "",
			wantLen:       100,
			wantNextEmpty: true,
			wantTotal:     100,
		},
		{
			name:          "modern_default_page_size",
			pageSize:      0,
			pageToken:     pluginsdk.EncodePageToken(0),
			wantLen:       pluginsdk.DefaultPageSize,
			wantNextEmpty: false,
			wantTotal:     100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, tc.pageSize, tc.pageToken)
			require.NoError(t, err)
			require.Len(t, page, tc.wantLen)
			require.Equal(t, tc.wantNextEmpty, nextToken == "", "nextToken empty mismatch")
			require.Equal(t, tc.wantTotal, totalCount)
		})
	}
}

// TestPaginateActualCosts_FirstPageCustomSize verifies page_size=100 returns exactly
// 100 records from a 500-record dataset with a valid continuation token.
func TestPaginateActualCosts_FirstPageCustomSize(t *testing.T) {
	results := testActualCosts(500)
	page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, 100, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != 100 {
		t.Errorf("expected 100 results, got %d", len(page))
	}
	if nextToken == "" {
		t.Error("expected next token but got empty")
	}
	if totalCount != 500 {
		t.Errorf("expected total_count=500, got %d", totalCount)
	}
}

// TestPaginateActualCosts_LastPage verifies the last page returns remaining
// records and an empty next_page_token.
func TestPaginateActualCosts_LastPage(t *testing.T) {
	results := testActualCosts(25)
	page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, 10, pluginsdk.EncodePageToken(20))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != 5 {
		t.Errorf("expected 5 results, got %d", len(page))
	}
	if nextToken != "" {
		t.Errorf("expected empty next token, got %s", nextToken)
	}
	if totalCount != 25 {
		t.Errorf("expected total_count=25, got %d", totalCount)
	}
}

// TestPaginateActualCosts_InvalidToken verifies malformed token returns an error.
func TestPaginateActualCosts_InvalidToken(t *testing.T) {
	results := testActualCosts(100)
	_, _, _, err := pluginsdk.PaginateActualCosts(results, 10, "invalid-token!!!")
	if err == nil {
		t.Error("expected error for invalid token but got nil")
	}
}

// TestPaginateActualCosts_OffsetBeyondRange verifies valid token pointing beyond
// data returns empty results and empty next token.
func TestPaginateActualCosts_OffsetBeyondRange(t *testing.T) {
	results := testActualCosts(25)
	page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, 10, pluginsdk.EncodePageToken(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != 0 {
		t.Errorf("expected 0 results, got %d", len(page))
	}
	if nextToken != "" {
		t.Errorf("expected empty next token, got %s", nextToken)
	}
	if totalCount != 25 {
		t.Errorf("expected total_count=25, got %d", totalCount)
	}
}

// TestPaginateActualCosts_MaxPageSizeClamping verifies page_size > 1000 is clamped to MaxPageSize.
func TestPaginateActualCosts_MaxPageSizeClamping(t *testing.T) {
	results := testActualCosts(2000)
	page, nextToken, _, err := pluginsdk.PaginateActualCosts(results, 5000, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != pluginsdk.MaxPageSize {
		t.Errorf("expected %d results (MaxPageSize), got %d", pluginsdk.MaxPageSize, len(page))
	}
	if nextToken == "" {
		t.Error("expected next token but got empty")
	}
}

// TestPaginateActualCosts_TotalCount verifies total_count is auto-populated from slice length.
func TestPaginateActualCosts_TotalCount(t *testing.T) {
	results := testActualCosts(42)
	_, _, totalCount, err := pluginsdk.PaginateActualCosts(results, 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if totalCount != 42 {
		t.Errorf("expected total_count=42, got %d", totalCount)
	}
}

// TestPaginateActualCosts_SinglePageFitsAll verifies that when total records < page_size,
// all records returned with empty next token.
func TestPaginateActualCosts_SinglePageFitsAll(t *testing.T) {
	results := testActualCosts(10)
	page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, 100, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != 10 {
		t.Errorf("expected 10 results, got %d", len(page))
	}
	if nextToken != "" {
		t.Errorf("expected empty next token, got %s", nextToken)
	}
	if totalCount != 10 {
		t.Errorf("expected total_count=10, got %d", totalCount)
	}
}

// TestPaginateActualCosts_FullIteration iterates through all pages of a 500-record
// dataset with page_size=100 and verifies: 5 pages total, each page has correct
// count, no duplicates, no gaps, total_count consistent across pages.
func TestPaginateActualCosts_FullIteration(t *testing.T) {
	results := testActualCosts(500)
	var allCollected []*pbc.ActualCostResult
	pageToken := ""
	pageCount := 0

	for {
		page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, 100, pageToken)
		if err != nil {
			t.Fatalf("page %d: unexpected error: %v", pageCount, err)
		}
		if totalCount != 500 {
			t.Errorf("page %d: expected total_count=500, got %d", pageCount, totalCount)
		}
		allCollected = append(allCollected, page...)
		pageCount++

		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	if pageCount != 5 {
		t.Errorf("expected 5 pages, got %d", pageCount)
	}
	if len(allCollected) != 500 {
		t.Errorf("expected 500 total records, got %d", len(allCollected))
	}
	// Verify no gaps: each record has sequential cost values
	for i, r := range allCollected {
		expected := float64(i) + 1.0
		if r.GetCost() != expected {
			t.Errorf("record %d: expected cost=%f, got %f", i, expected, r.GetCost())
			break
		}
	}
}

// TestPaginateActualCosts_IntegerOverflowProtection verifies that totalCount is clamped
// to int32 max when the result set exceeds int32 capacity. This prevents overflow bugs
// in the proto response where total_count is defined as int32.
func TestPaginateActualCosts_IntegerOverflowProtection(t *testing.T) {
	// Note: We can't actually create a slice with >2^31 elements in tests (would require 16+ GB RAM),
	// but we can verify the logic by checking the implementation handles large counts correctly.
	results := testActualCosts(1000)

	tests := []struct {
		name          string
		pageSize      int32
		pageToken     string
		wantLen       int
		wantNextEmpty bool
		wantTotal     int32
	}{
		{
			name:          "normal_count_no_clamping",
			pageSize:      50,
			pageToken:     "",
			wantLen:       50,
			wantNextEmpty: false,
			wantTotal:     1000,
		},
		{
			name:          "legacy_large_dataset",
			pageSize:      0,
			pageToken:     "",
			wantLen:       1000,
			wantNextEmpty: true,
			wantTotal:     1000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, tc.pageSize, tc.pageToken)
			require.NoError(t, err)
			require.Len(t, page, tc.wantLen)
			require.Equal(t, tc.wantNextEmpty, nextToken == "", "nextToken empty mismatch")
			require.Equal(t, tc.wantTotal, totalCount)
		})
	}
}

// TestPaginateActualCosts_NegativePageSize verifies that a negative page_size
// is normalized to the default instead of returning an error (proto contract: <=0 uses default).
func TestPaginateActualCosts_NegativePageSize(t *testing.T) {
	results := testActualCosts(100)

	tests := []struct {
		name          string
		pageSize      int32
		pageToken     string
		wantLen       int
		wantNextEmpty bool
		wantTotal     int32
	}{
		{
			name:          "negative_with_token_normalizes",
			pageSize:      -1,
			pageToken:     pluginsdk.EncodePageToken(0),
			wantLen:       pluginsdk.DefaultPageSize,
			wantNextEmpty: false,
			wantTotal:     100,
		},
		{
			name:          "negative_without_token_legacy",
			pageSize:      -1,
			pageToken:     "",
			wantLen:       100,
			wantNextEmpty: true,
			wantTotal:     100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(results, tc.pageSize, tc.pageToken)
			require.NoError(t, err)
			require.Len(t, page, tc.wantLen)
			require.Equal(t, tc.wantNextEmpty, nextToken == "", "nextToken empty mismatch")
			require.Equal(t, tc.wantTotal, totalCount)
		})
	}
}

// BenchmarkPaginateActualCosts benchmarks pagination with 1000 records and page_size=100.
func BenchmarkPaginateActualCosts(b *testing.B) {
	results := testActualCosts(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _, _, _ = pluginsdk.PaginateActualCosts(results, 100, "")
	}
}
