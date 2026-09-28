package testing_test

import (
	"math"
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pktesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// assertSummaryFields asserts scalar fields of RecommendationSummary using Get*() methods.
func assertSummaryFields(t *testing.T, result, expected *pbc.RecommendationSummary) {
	t.Helper()

	if result.GetTotalRecommendations() != expected.GetTotalRecommendations() {
		t.Errorf("TotalRecommendations = %d, want %d",
			result.GetTotalRecommendations(), expected.GetTotalRecommendations())
	}

	if result.GetProjectionPeriod() != expected.GetProjectionPeriod() {
		t.Errorf(
			"ProjectionPeriod = %s, want %s",
			result.GetProjectionPeriod(),
			expected.GetProjectionPeriod(),
		)
	}

	if result.GetTotalEstimatedSavings() != expected.GetTotalEstimatedSavings() {
		t.Errorf(
			"TotalEstimatedSavings = %f, want %f",
			result.GetTotalEstimatedSavings(),
			expected.GetTotalEstimatedSavings(),
		)
	}

	if result.GetCurrency() != expected.GetCurrency() {
		t.Errorf("Currency = %s, want %s", result.GetCurrency(), expected.GetCurrency())
	}
}

// assertMapFieldsInt32 asserts map[string]int32 fields.
func assertMapFieldsInt32(t *testing.T, name string, result, expected map[string]int32) {
	t.Helper()

	if len(result) != len(expected) {
		t.Errorf("%s length = %d, want %d", name, len(result), len(expected))
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("%s[%s] = %d, want %d", name, k, result[k], v)
		}
	}
}

// assertMapFieldsFloat64 asserts map[string]float64 fields.
func assertMapFieldsFloat64(t *testing.T, name string, result, expected map[string]float64) {
	t.Helper()

	if len(result) != len(expected) {
		t.Errorf("%s length = %d, want %d", name, len(result), len(expected))
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("%s[%s] = %f, want %f", name, k, result[k], v)
		}
	}
}

func mockImpact(currency string, savings float64) *pbc.Recommendation {
	return &pbc.Recommendation{
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
		Impact: &pbc.RecommendationImpact{
			EstimatedSavings: savings,
			Currency:         currency,
		},
	}
}

func mockSummary(total float64, currency string) *pbc.RecommendationSummary {
	return &pbc.RecommendationSummary{
		TotalRecommendations: 2,
		CountByCategory: map[string]int32{
			"RECOMMENDATION_CATEGORY_COST": 2,
		},
		SavingsByCategory: map[string]float64{
			"RECOMMENDATION_CATEGORY_COST": total,
		},
		CountByActionType: map[string]int32{
			"RECOMMENDATION_ACTION_TYPE_MODIFY": 2,
		},
		SavingsByActionType: map[string]float64{
			"RECOMMENDATION_ACTION_TYPE_MODIFY": total,
		},
		ProjectionPeriod:      "monthly",
		TotalEstimatedSavings: total,
		Currency:              currency,
	}
}

func TestCalculateMockSummary(t *testing.T) {
	tests := []struct {
		name             string
		recommendations  []*pbc.Recommendation
		projectionPeriod string
		expected         *pbc.RecommendationSummary
	}{
		{
			name:             "empty recommendations",
			recommendations:  []*pbc.Recommendation{},
			projectionPeriod: "monthly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations:  0,
				CountByCategory:       make(map[string]int32),
				SavingsByCategory:     make(map[string]float64),
				CountByActionType:     make(map[string]int32),
				SavingsByActionType:   make(map[string]float64),
				ProjectionPeriod:      "monthly",
				TotalEstimatedSavings: 0,
				Currency:              "",
			},
		},
		{
			name: "single recommendation with savings",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 100.0,
						Currency:         "USD",
					},
				},
			},
			projectionPeriod: "monthly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations: 1,
				CountByCategory: map[string]int32{
					"RECOMMENDATION_CATEGORY_COST": 1,
				},
				SavingsByCategory: map[string]float64{
					"RECOMMENDATION_CATEGORY_COST": 100.0,
				},
				CountByActionType: map[string]int32{
					"RECOMMENDATION_ACTION_TYPE_MODIFY": 1,
				},
				SavingsByActionType: map[string]float64{
					"RECOMMENDATION_ACTION_TYPE_MODIFY": 100.0,
				},
				ProjectionPeriod:      "monthly",
				TotalEstimatedSavings: 100.0,
				Currency:              "USD",
			},
		},
		{
			name: "multiple recommendations with consistent currency",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 50.0,
						Currency:         "USD",
					},
				},
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 75.0,
						Currency:         "USD",
					},
				},
			},
			projectionPeriod: "yearly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations: 2,
				CountByCategory: map[string]int32{
					"RECOMMENDATION_CATEGORY_COST":        1,
					"RECOMMENDATION_CATEGORY_PERFORMANCE": 1,
				},
				SavingsByCategory: map[string]float64{
					"RECOMMENDATION_CATEGORY_COST":        50.0,
					"RECOMMENDATION_CATEGORY_PERFORMANCE": 75.0,
				},
				CountByActionType: map[string]int32{
					"RECOMMENDATION_ACTION_TYPE_MODIFY":    1,
					"RECOMMENDATION_ACTION_TYPE_RIGHTSIZE": 1,
				},
				SavingsByActionType: map[string]float64{
					"RECOMMENDATION_ACTION_TYPE_MODIFY":    50.0,
					"RECOMMENDATION_ACTION_TYPE_RIGHTSIZE": 75.0,
				},
				ProjectionPeriod:      "yearly",
				TotalEstimatedSavings: 125.0,
				Currency:              "USD",
			},
		},
		{
			name: "mixed currencies clears currency field",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 50.0,
						Currency:         "USD",
					},
				},
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 75.0,
						Currency:         "EUR",
					},
				},
			},
			projectionPeriod: "monthly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations: 2,
				CountByCategory: map[string]int32{
					"RECOMMENDATION_CATEGORY_COST":        1,
					"RECOMMENDATION_CATEGORY_PERFORMANCE": 1,
				},
				SavingsByCategory: map[string]float64{
					"RECOMMENDATION_CATEGORY_COST":        50.0,
					"RECOMMENDATION_CATEGORY_PERFORMANCE": 75.0,
				},
				CountByActionType: map[string]int32{
					"RECOMMENDATION_ACTION_TYPE_MODIFY":    1,
					"RECOMMENDATION_ACTION_TYPE_RIGHTSIZE": 1,
				},
				SavingsByActionType: map[string]float64{
					"RECOMMENDATION_ACTION_TYPE_MODIFY":    50.0,
					"RECOMMENDATION_ACTION_TYPE_RIGHTSIZE": 75.0,
				},
				ProjectionPeriod:      "monthly",
				TotalEstimatedSavings: 0,
				Currency:              "", // withheld: unlike currencies are not added
			},
		},
		{
			name: "mixed category withheld and single-currency category kept",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 100.0,
						Currency:         "USD",
					},
				},
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 40.0,
						Currency:         "EUR",
					},
				},
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_PERFORMANCE,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 25.0,
						Currency:         "USD",
					},
				},
			},
			projectionPeriod: "monthly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations: 3,
				CountByCategory: map[string]int32{
					"RECOMMENDATION_CATEGORY_COST":        2,
					"RECOMMENDATION_CATEGORY_PERFORMANCE": 1,
				},
				SavingsByCategory: map[string]float64{
					"RECOMMENDATION_CATEGORY_COST":        0,
					"RECOMMENDATION_CATEGORY_PERFORMANCE": 25.0,
				},
				CountByActionType: map[string]int32{
					"RECOMMENDATION_ACTION_TYPE_RIGHTSIZE": 2,
					"RECOMMENDATION_ACTION_TYPE_TERMINATE": 1,
				},
				SavingsByActionType: map[string]float64{
					"RECOMMENDATION_ACTION_TYPE_RIGHTSIZE": 0,
					"RECOMMENDATION_ACTION_TYPE_TERMINATE": 25.0,
				},
				ProjectionPeriod:      "monthly",
				TotalEstimatedSavings: 0,
				Currency:              "",
			},
		},
		{
			name: "empty currency folds into USD",
			recommendations: []*pbc.Recommendation{
				mockImpact("USD", 100),
				mockImpact("", 50),
			},
			projectionPeriod: "monthly",
			expected:         mockSummary(150, "USD"),
		},
		{
			name: "XXX counts as its own currency",
			recommendations: []*pbc.Recommendation{
				mockImpact("XXX", 100),
				mockImpact("USD", 50),
			},
			projectionPeriod: "monthly",
			expected:         mockSummary(0, ""),
		},
		{
			name: "only empty currencies keep the sum",
			recommendations: []*pbc.Recommendation{
				mockImpact("", 100),
				mockImpact("", 50),
			},
			projectionPeriod: "monthly",
			expected:         mockSummary(150, ""),
		},
		{
			name: "letter case is a distinct currency",
			recommendations: []*pbc.Recommendation{
				mockImpact("usd", 100),
				mockImpact("USD", 50),
			},
			projectionPeriod: "monthly",
			expected:         mockSummary(0, ""),
		},
		{
			name: "missing impact still counts beside a priced row",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
				},
				mockImpact("USD", 10),
			},
			projectionPeriod: "monthly",
			expected:         mockSummary(10, "USD"),
		},
		{
			name: "recommendation without impact",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
					Impact:     nil,
				},
			},
			projectionPeriod: "monthly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations: 1,
				CountByCategory: map[string]int32{
					"RECOMMENDATION_CATEGORY_COST": 1,
				},
				SavingsByCategory: make(map[string]float64), // Empty when no impact
				CountByActionType: map[string]int32{
					"RECOMMENDATION_ACTION_TYPE_MODIFY": 1,
				},
				SavingsByActionType:   make(map[string]float64), // Empty when no impact
				ProjectionPeriod:      "monthly",
				TotalEstimatedSavings: 0,
				Currency:              "",
			},
		},
		{
			name: "multiple recommendations same category and action type",
			recommendations: []*pbc.Recommendation{
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 25.0,
						Currency:         "USD",
					},
				},
				{
					Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
					ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_MODIFY,
					Impact: &pbc.RecommendationImpact{
						EstimatedSavings: 35.0,
						Currency:         "USD",
					},
				},
			},
			projectionPeriod: "monthly",
			expected: &pbc.RecommendationSummary{
				TotalRecommendations: 2,
				CountByCategory: map[string]int32{
					"RECOMMENDATION_CATEGORY_COST": 2,
				},
				SavingsByCategory: map[string]float64{
					"RECOMMENDATION_CATEGORY_COST": 60.0,
				},
				CountByActionType: map[string]int32{
					"RECOMMENDATION_ACTION_TYPE_MODIFY": 2,
				},
				SavingsByActionType: map[string]float64{
					"RECOMMENDATION_ACTION_TYPE_MODIFY": 60.0,
				},
				ProjectionPeriod:      "monthly",
				TotalEstimatedSavings: 60.0,
				Currency:              "USD",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pktesting.CalculateMockSummary(tt.recommendations, tt.projectionPeriod)

			assertSummaryFields(t, result, tt.expected)

			assertMapFieldsInt32(
				t,
				"CountByCategory",
				result.GetCountByCategory(),
				tt.expected.GetCountByCategory(),
			)
			assertMapFieldsFloat64(
				t,
				"SavingsByCategory",
				result.GetSavingsByCategory(),
				tt.expected.GetSavingsByCategory(),
			)
			assertMapFieldsInt32(
				t,
				"CountByActionType",
				result.GetCountByActionType(),
				tt.expected.GetCountByActionType(),
			)
			assertMapFieldsFloat64(
				t,
				"SavingsByActionType",
				result.GetSavingsByActionType(),
				tt.expected.GetSavingsByActionType(),
			)
		})
	}
}

func TestSetSpotRiskScore_ValidValues(t *testing.T) {
	tests := []struct {
		name  string
		score float64
	}{
		{"zero_risk", 0.0},
		{"low_risk", 0.25},
		{"medium_risk", 0.5},
		{"high_risk", 0.75},
		{"max_risk", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := pktesting.NewMockPlugin()
			// Should not panic for valid values
			plugin.SetSpotRiskScore(tt.score)
			if plugin.DefaultSpotInterruptionRiskScore != tt.score {
				t.Errorf("DefaultSpotInterruptionRiskScore = %f, want %f",
					plugin.DefaultSpotInterruptionRiskScore, tt.score)
			}
		})
	}
}

func TestSetSpotRiskScore_InvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		score float64
	}{
		{"negative", -0.1},
		{"below_zero", -1.0},
		{"above_one", 1.1},
		{"too_high", 2.0},
		{"nan", math.NaN()},
		{"positive_inf", math.Inf(1)},
		{"negative_inf", math.Inf(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := pktesting.NewMockPlugin()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("SetSpotRiskScore(%f) did not panic", tt.score)
				}
			}()
			plugin.SetSpotRiskScore(tt.score)
		})
	}
}

func TestSetSpotRiskScoreForResourceType_ValidValues(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		score        float64
	}{
		{"spot_zero", "spot", 0.0},
		{"spot_low", "spot", 0.3},
		{"preemptible_medium", "preemptible", 0.5},
		{"reserved_high", "reserved", 0.8},
		{"on_demand_max", "on-demand", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := pktesting.NewMockPlugin()
			// Should not panic for valid values
			plugin.SetSpotRiskScoreForResourceType(tt.resourceType, tt.score)
			if plugin.SpotRiskScoreByResourceType[tt.resourceType] != tt.score {
				t.Errorf("SpotRiskScoreByResourceType[%s] = %f, want %f",
					tt.resourceType, plugin.SpotRiskScoreByResourceType[tt.resourceType], tt.score)
			}
		})
	}
}

func TestSetSpotRiskScoreForResourceType_InvalidValues(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		score        float64
	}{
		{"negative", "spot", -0.1},
		{"below_zero", "preemptible", -1.0},
		{"above_one", "spot", 1.1},
		{"too_high", "reserved", 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := pktesting.NewMockPlugin()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("SetSpotRiskScoreForResourceType(%s, %f) did not panic",
						tt.resourceType, tt.score)
				}
			}()
			plugin.SetSpotRiskScoreForResourceType(tt.resourceType, tt.score)
		})
	}
}

// TestGetProjectedCostPricingOverrides verifies that pricing overrides configured with
// simple resource keys (e.g., "ec2") are correctly applied when GetProjectedCost is called
// with full Pulumi-style resource descriptors (e.g., "aws:ec2/instance:Instance").
func TestGetProjectedCostPricingOverrides(t *testing.T) {
	tests := []struct {
		name                  string
		simpleResourceKey     string
		fullResourceType      string
		provider              string
		pricingCategory       pbc.FocusPricingCategory
		spotRiskScore         float64
		expectPricingCategory pbc.FocusPricingCategory
		expectSpotRiskScore   float64
	}{
		{
			name:                  "ec2_pricing_override",
			simpleResourceKey:     "ec2",
			fullResourceType:      "aws:ec2/instance:Instance",
			provider:              "aws",
			pricingCategory:       pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			spotRiskScore:         0.0, // T043: non-zero spot risk only valid with DYNAMIC pricing
			expectPricingCategory: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			expectSpotRiskScore:   0.0,
		},
		{
			name:                  "spot_pricing_override",
			simpleResourceKey:     "spot",
			fullResourceType:      "aws:spot/spotInstance:SpotInstance",
			provider:              "aws",
			pricingCategory:       pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			spotRiskScore:         0.85,
			expectPricingCategory: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			expectSpotRiskScore:   0.85,
		},
		{
			name:                  "s3_pricing_override",
			simpleResourceKey:     "s3",
			fullResourceType:      "aws:s3/bucket:Bucket",
			provider:              "aws",
			pricingCategory:       pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			spotRiskScore:         0.0,
			expectPricingCategory: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			expectSpotRiskScore:   0.0,
		},
		{
			name:                  "lambda_pricing_override",
			simpleResourceKey:     "lambda",
			fullResourceType:      "aws:lambda/function:Function",
			provider:              "aws",
			pricingCategory:       pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			spotRiskScore:         0.0, // T043: non-zero spot risk only valid with DYNAMIC pricing
			expectPricingCategory: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			expectSpotRiskScore:   0.0,
		},
		{
			name:                  "vm_azure_pricing_override",
			simpleResourceKey:     "vm",
			fullResourceType:      "azure:vm/virtualMachine:VirtualMachine",
			provider:              "azure",
			pricingCategory:       pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			spotRiskScore:         0.0, // T043: non-zero spot risk only valid with DYNAMIC pricing
			expectPricingCategory: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			expectSpotRiskScore:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock plugin and configure pricing overrides
			plugin := pktesting.NewMockPlugin()
			plugin.SetPricingCategoryForResourceType(tt.simpleResourceKey, tt.pricingCategory)
			plugin.SetSpotRiskScoreForResourceType(tt.simpleResourceKey, tt.spotRiskScore)

			// Create and start test harness
			harness := pktesting.NewTestHarness(plugin)
			harness.Start(t)
			defer harness.Stop()

			client := harness.Client()

			// Create resource descriptor with full Pulumi-style resource type
			resource := &pbc.ResourceDescriptor{
				Provider:     tt.provider,
				ResourceType: tt.fullResourceType,
				Sku:          "test-sku",
				Region:       "us-east-1",
			}

			// Call GetProjectedCost
			resp, err := client.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
				Resource: resource,
			})
			if err != nil {
				t.Fatalf("GetProjectedCost() failed: %v", err)
			}

			// Verify pricing category override was applied
			if resp.GetPricingCategory() != tt.expectPricingCategory {
				t.Errorf("PricingCategory = %v, want %v",
					resp.GetPricingCategory(), tt.expectPricingCategory)
			}

			// Verify spot risk score override was applied
			if resp.GetSpotInterruptionRiskScore() != tt.expectSpotRiskScore {
				t.Errorf("SpotInterruptionRiskScore = %f, want %f",
					resp.GetSpotInterruptionRiskScore(), tt.expectSpotRiskScore)
			}
		})
	}
}

// TestGetSimpleResourceKeyExtraction verifies the helper method correctly extracts
// simple resource keys from full Pulumi-style resource descriptors.
func TestGetSimpleResourceKeyExtraction(t *testing.T) {
	plugin := pktesting.NewMockPlugin()

	// Test that setting pricing for "ec2" affects full descriptor "aws:ec2/instance:Instance"
	// Use DYNAMIC pricing to comply with T043 (spot risk only valid with DYNAMIC)
	plugin.SetPricingCategoryForResourceType("ec2", pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC)
	plugin.SetSpotRiskScoreForResourceType("ec2", 0.75)

	harness := pktesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()

	// Test with full Pulumi-style resource type
	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "aws:ec2/instance:Instance",
		Sku:          "t3.micro",
		Region:       "us-west-2",
	}

	resp, err := client.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: resource,
	})
	if err != nil {
		t.Fatalf("GetProjectedCost() failed: %v", err)
	}

	// The overrides should be applied because getSimpleResourceKey extracts "ec2"
	if resp.GetPricingCategory() != pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC {
		t.Errorf("Expected DYNAMIC pricing category, got %v", resp.GetPricingCategory())
	}

	if resp.GetSpotInterruptionRiskScore() != 0.75 {
		t.Errorf("Expected spot risk score 0.75, got %f", resp.GetSpotInterruptionRiskScore())
	}
}
