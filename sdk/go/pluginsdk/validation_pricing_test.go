package pluginsdk_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestValidateEstimateCostResponse(t *testing.T) {
	t.Run("nil_response", func(t *testing.T) {
		err := pluginsdk.ValidateEstimateCostResponse(nil)
		assert.ErrorIs(t, err, pluginsdk.ErrEstimateCostResponseNil)
	})

	t.Run("valid_response_with_zero_risk", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: 0.0,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err)
	})

	// CRITICAL: This test documents backward compatibility for legacy plugins.
	// Legacy plugins that don't set either pricing_category or spot_interruption_risk_score
	// default to UNSPECIFIED + 0.0 (proto3 defaults). This MUST remain valid.
	t.Run("valid_unspecified_category_with_zero_risk_backward_compat", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			SpotInterruptionRiskScore: 0.0,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err, "UNSPECIFIED + 0.0 must be valid for backward compatibility with legacy plugins")
	})

	// Test proto3 default behavior: when neither field is set, both default to zero values.
	// This represents a minimal valid response from legacy plugins.
	t.Run("valid_proto3_default_values", func(t *testing.T) {
		// Simulates a legacy plugin that only sets required business fields
		resp := &pbc.EstimateCostResponse{
			Currency:    "USD",
			CostMonthly: 50.0,
			// pricing_category defaults to UNSPECIFIED (0)
			// spot_interruption_risk_score defaults to 0.0
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err, "Proto3 default values (UNSPECIFIED + 0.0) must be valid")
	})

	t.Run("valid_response_with_dynamic_pricing", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			SpotInterruptionRiskScore: 0.8,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err)
	})

	t.Run("valid_boundary_values", func(t *testing.T) {
		testCases := []float64{0.0, 0.5, 1.0, 0.0001, 0.9999}
		for _, score := range testCases {
			resp := &pbc.EstimateCostResponse{
				Currency:                  "USD",
				CostMonthly:               50.0,
				PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
				SpotInterruptionRiskScore: score,
			}
			err := pluginsdk.ValidateEstimateCostResponse(resp)
			assert.NoError(t, err, "score %f should be valid", score)
		}
	})

	t.Run("invalid_nan", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: math.NaN(),
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreNaN)
	})

	t.Run("invalid_positive_inf", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: math.Inf(1),
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreNaN)
	})

	t.Run("invalid_negative_inf", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: math.Inf(-1),
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreNaN)
	})

	t.Run("invalid_negative_value", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: -0.5,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange)
	})

	t.Run("invalid_greater_than_one", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: 1.5,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange)
	})

	t.Run("invalid_unspecified_category_with_nonzero_risk", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			SpotInterruptionRiskScore: 0.5,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreInvalidCategory)
	})

	t.Run("invalid_standard_category_with_nonzero_risk", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: 0.8,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreInvalidCategory)
	})
}

func TestValidateGetProjectedCostResponse(t *testing.T) {
	t.Run("nil_response", func(t *testing.T) {
		err := pluginsdk.ValidateGetProjectedCostResponse(nil)
		assert.ErrorIs(t, err, pluginsdk.ErrGetProjectedCostResponseNil)
	})

	t.Run("valid_response_with_zero_risk", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: 0.0,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err)
	})

	// CRITICAL: Backward compatibility test for legacy plugins.
	t.Run("valid_unspecified_category_with_zero_risk_backward_compat", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			SpotInterruptionRiskScore: 0.0,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err, "UNSPECIFIED + 0.0 must be valid for backward compatibility")
	})

	// Test proto3 default behavior.
	t.Run("valid_proto3_default_values", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:    0.05,
			Currency:     "USD",
			CostPerMonth: 36.50,
			// pricing_category defaults to UNSPECIFIED (0)
			// spot_interruption_risk_score defaults to 0.0
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err, "Proto3 default values must be valid")
	})

	t.Run("valid_response_with_dynamic_pricing", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			SpotInterruptionRiskScore: 0.8,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err)
	})

	t.Run("invalid_nan", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			SpotInterruptionRiskScore: math.NaN(),
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreNaN)
	})

	t.Run("invalid_out_of_range", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			SpotInterruptionRiskScore: 2.0,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange)
	})

	t.Run("invalid_negative_inf", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			SpotInterruptionRiskScore: math.Inf(-1),
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreNaN)
	})

	t.Run("invalid_negative_value", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			SpotInterruptionRiskScore: -0.5,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange)
	})

	t.Run("invalid_unspecified_category_with_nonzero_risk", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			SpotInterruptionRiskScore: 0.5,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreInvalidCategory)
	})

	t.Run("invalid_standard_category_with_nonzero_risk", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:                 0.05,
			Currency:                  "USD",
			CostPerMonth:              36.50,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: 0.8,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreInvalidCategory)
	})

	// Zero-width interval validation tests
	t.Run("invalid_zero_width_interval_with_nonzero_cost", func(t *testing.T) {
		lower := 0.0
		upper := 0.0
		confidence := 0.95
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            100.0, // Non-zero cost doesn't match bounds
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
			ConfidenceLevel:         &confidence,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "zero-width prediction interval")
		assert.Contains(t, err.Error(), "requires cost_per_month to equal bounds")
	})

	t.Run("valid_zero_width_interval_with_zero_cost", func(t *testing.T) {
		lower := 0.0
		upper := 0.0
		confidence := 0.95
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.0,
			Currency:                "USD",
			CostPerMonth:            0.0, // Zero cost matches zero-width bounds
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
			ConfidenceLevel:         &confidence,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err, "zero-width interval with matching cost should be valid")
	})

	t.Run("valid_zero_width_interval_with_matching_nonzero_cost", func(t *testing.T) {
		lower := 42.0
		upper := 42.0
		confidence := 0.95
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            42.0, // Cost equals bounds - valid
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
			ConfidenceLevel:         &confidence,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err, "zero-width interval [42, 42] with cost=42 should be valid")
	})

	t.Run("invalid_zero_width_interval_cost_mismatch", func(t *testing.T) {
		lower := 42.0
		upper := 42.0
		confidence := 0.95
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0, // Cost doesn't match bounds
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
			ConfidenceLevel:         &confidence,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "zero-width prediction interval")
		assert.Contains(t, err.Error(), "[42")
		assert.Contains(t, err.Error(), "requires cost_per_month to equal bounds")
		assert.Contains(t, err.Error(), "50")
	})

	// NaN/Inf tests for prediction interval bounds
	t.Run("invalid_prediction_interval_lower_nan", func(t *testing.T) {
		lower := math.NaN()
		upper := 100.0
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0,
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prediction_interval_lower")
		assert.Contains(t, err.Error(), "NaN")
	})

	t.Run("invalid_prediction_interval_upper_nan", func(t *testing.T) {
		lower := 10.0
		upper := math.NaN()
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0,
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prediction_interval_upper")
		assert.Contains(t, err.Error(), "NaN")
	})

	t.Run("invalid_prediction_interval_lower_positive_inf", func(t *testing.T) {
		lower := math.Inf(1)
		upper := 100.0
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0,
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prediction_interval_lower")
		assert.Contains(t, err.Error(), "Inf")
	})

	t.Run("invalid_prediction_interval_upper_positive_inf", func(t *testing.T) {
		lower := 10.0
		upper := math.Inf(1)
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0,
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prediction_interval_upper")
		assert.Contains(t, err.Error(), "Inf")
	})

	t.Run("invalid_prediction_interval_lower_negative_inf", func(t *testing.T) {
		lower := math.Inf(-1)
		upper := 100.0
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0,
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prediction_interval_lower")
		assert.Contains(t, err.Error(), "Inf")
	})

	t.Run("invalid_prediction_interval_upper_negative_inf", func(t *testing.T) {
		lower := 10.0
		upper := math.Inf(-1)
		resp := &pbc.GetProjectedCostResponse{
			UnitPrice:               0.05,
			Currency:                "USD",
			CostPerMonth:            50.0,
			PredictionIntervalLower: &lower,
			PredictionIntervalUpper: &upper,
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prediction_interval_upper")
		assert.Contains(t, err.Error(), "Inf")
	})

	t.Run("invalid_expires_at_nanos", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			CostPerMonth: 36.50,
			ExpiresAt:    &timestamppb.Timestamp{Seconds: 0, Nanos: -1},
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expires_at is invalid")
	})

	t.Run("valid_expires_at", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			CostPerMonth: 36.50,
			ExpiresAt:    timestamppb.Now(),
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		assert.NoError(t, err)
	})

	// Backward compatibility: responses from plugins that never set cost_breakdown.
	t.Run("valid_no_cost_breakdown_backward_compat", func(t *testing.T) {
		tests := []struct {
			name string
			resp *pbc.GetProjectedCostResponse
		}{
			{"nonzero_total", &pbc.GetProjectedCostResponse{Currency: "USD", CostPerMonth: 8.0}},
			{"zero_total", &pbc.GetProjectedCostResponse{Currency: "USD", CostPerMonth: 0}},
			{"dry_run", &pbc.GetProjectedCostResponse{DryRunResult: &pbc.DryRunResponse{}}},
			{"empty_non_nil_map", &pbc.GetProjectedCostResponse{
				Currency:      "USD",
				CostPerMonth:  8.0,
				CostBreakdown: map[string]float64{},
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(tt.resp))
			})
		}
	})
}

func TestCheckSpotRiskConsistency(t *testing.T) {
	t.Run("consistent_dynamic_with_risk", func(t *testing.T) {
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			0.8,
		)
		assert.Empty(t, warnings)
	})

	t.Run("consistent_standard_with_zero_risk", func(t *testing.T) {
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			0.0,
		)
		assert.Empty(t, warnings)
	})

	// CRITICAL: UNSPECIFIED + 0.0 should produce no warnings (backward compat).
	// This is the most common case for legacy plugins.
	t.Run("consistent_unspecified_with_zero_risk_backward_compat", func(t *testing.T) {
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			0.0,
		)
		assert.Empty(t, warnings, "UNSPECIFIED + 0.0 should have no warnings (legacy plugin case)")
	})

	t.Run("inconsistent_standard_with_nonzero_risk", func(t *testing.T) {
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			0.5,
		)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "spot_interruption_risk_score > 0.0")
		assert.Contains(t, warnings[0], "not DYNAMIC")
	})

	t.Run("inconsistent_committed_with_nonzero_risk", func(t *testing.T) {
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
			0.3,
		)
		assert.Len(t, warnings, 1)
	})

	t.Run("dynamic_with_zero_risk_warns_about_missing_data", func(t *testing.T) {
		// DYNAMIC pricing with zero risk score triggers an advisory warning
		// because it may indicate missing risk data
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			0.0,
		)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "pricing_category is DYNAMIC")
		assert.Contains(t, warnings[0], "spot_interruption_risk_score is 0.0")
	})

	t.Run("unspecified_category_with_risk", func(t *testing.T) {
		warnings := pluginsdk.CheckSpotRiskConsistency(
			pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_UNSPECIFIED,
			0.5,
		)
		assert.Len(t, warnings, 1)
	})
}

// TestSpotRiskScoreEdgeCases tests float precision edge cases that may occur
// from floating-point arithmetic operations.
func TestSpotRiskScoreEdgeCases(t *testing.T) {
	t.Run("negative_zero", func(t *testing.T) {
		// IEEE 754 negative zero (-0.0) should be treated as zero
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: math.Copysign(0, -1), // -0.0
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err, "negative zero should be treated as zero")
	})

	t.Run("very_small_positive_value", func(t *testing.T) {
		// Values smaller than epsilon should be treated as zero
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: 1e-10, // smaller than epsilon (1e-9)
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err, "very small values should be treated as zero")
	})

	t.Run("subnormal_number", func(t *testing.T) {
		// Subnormal numbers (smallest representable positive floats) should be treated as zero
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: math.SmallestNonzeroFloat64, // ~5e-324
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err, "subnormal numbers should be treated as zero")
	})

	t.Run("float_precision_near_one", func(t *testing.T) {
		// Value very close to 1.0 due to floating-point arithmetic
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			SpotInterruptionRiskScore: 0.9999999999999999, // Very close to 1.0
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.NoError(t, err, "values very close to 1.0 should be valid")
	})

	t.Run("float_precision_just_over_one_invalid", func(t *testing.T) {
		// 1.0 + small epsilon should now be INVALID - probability cannot exceed 100%
		// Upper bound is strict 1.0 (epsilon tolerance only applies to lower bound)
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			SpotInterruptionRiskScore: 1.0 + 1e-10, // Just over 1.0 - should be invalid
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange, "values over 1.0 should be invalid")
	})

	t.Run("clearly_over_one", func(t *testing.T) {
		// 1.0 + large epsilon should fail
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			SpotInterruptionRiskScore: 1.0 + 1e-8, // Clearly over 1.0
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange)
	})

	t.Run("max_float64", func(t *testing.T) {
		// Maximum float64 value should be rejected
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: math.MaxFloat64,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		assert.ErrorIs(t, err, pluginsdk.ErrSpotRiskScoreOutOfRange)
	})
}

// TestWithSpotRiskPanics tests that WithSpotRisk panics for invalid values.
func TestWithSpotRiskPanics(t *testing.T) {
	t.Run("panics_on_nan", func(t *testing.T) {
		assert.PanicsWithValue(t,
			"WithSpotRisk: invalid score (NaN/Inf): NaN",
			func() { pluginsdk.WithSpotRisk(math.NaN()) },
		)
	})

	t.Run("panics_on_positive_inf", func(t *testing.T) {
		assert.PanicsWithValue(t,
			"WithSpotRisk: invalid score (NaN/Inf): +Inf",
			func() { pluginsdk.WithSpotRisk(math.Inf(1)) },
		)
	})

	t.Run("panics_on_negative_inf", func(t *testing.T) {
		assert.PanicsWithValue(t,
			"WithSpotRisk: invalid score (NaN/Inf): -Inf",
			func() { pluginsdk.WithSpotRisk(math.Inf(-1)) },
		)
	})

	t.Run("panics_on_negative_value", func(t *testing.T) {
		assert.Panics(t, func() { pluginsdk.WithSpotRisk(-0.5) })
	})

	t.Run("panics_on_greater_than_one", func(t *testing.T) {
		assert.Panics(t, func() { pluginsdk.WithSpotRisk(1.5) })
	})

	t.Run("does_not_panic_on_valid_values", func(t *testing.T) {
		validValues := []float64{0.0, 0.5, 1.0, 0.0001, 0.9999}
		for _, score := range validValues {
			assert.NotPanics(t, func() { pluginsdk.WithSpotRisk(score) },
				"score %f should not panic", score)
		}
	})
}

// TestWithProjectedCostSpotRiskPanics tests that WithProjectedCostSpotRisk panics for invalid values.
func TestWithProjectedCostSpotRiskPanics(t *testing.T) {
	t.Run("panics_on_nan", func(t *testing.T) {
		assert.PanicsWithValue(t,
			"WithProjectedCostSpotRisk: invalid score (NaN/Inf): NaN",
			func() { pluginsdk.WithProjectedCostSpotRisk(math.NaN()) },
		)
	})

	t.Run("panics_on_positive_inf", func(t *testing.T) {
		assert.PanicsWithValue(t,
			"WithProjectedCostSpotRisk: invalid score (NaN/Inf): +Inf",
			func() { pluginsdk.WithProjectedCostSpotRisk(math.Inf(1)) },
		)
	})

	t.Run("panics_on_negative_value", func(t *testing.T) {
		assert.Panics(t, func() { pluginsdk.WithProjectedCostSpotRisk(-0.5) })
	})

	t.Run("panics_on_greater_than_one", func(t *testing.T) {
		assert.Panics(t, func() { pluginsdk.WithProjectedCostSpotRisk(1.5) })
	})

	t.Run("does_not_panic_on_valid_values", func(t *testing.T) {
		validValues := []float64{0.0, 0.5, 1.0, 0.0001, 0.9999}
		for _, score := range validValues {
			assert.NotPanics(t, func() { pluginsdk.WithProjectedCostSpotRisk(score) },
				"score %f should not panic", score)
		}
	})
}

// TestErrorMessagesIncludeValue verifies that error messages include the actual invalid value.
func TestErrorMessagesIncludeValue(t *testing.T) {
	t.Run("nan_error_includes_value", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: math.NaN(),
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NaN")
	})

	t.Run("out_of_range_error_includes_value", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			SpotInterruptionRiskScore: 2.5,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "2.5")
	})

	t.Run("invalid_category_error_includes_details", func(t *testing.T) {
		resp := &pbc.EstimateCostResponse{
			Currency:                  "USD",
			CostMonthly:               50.0,
			PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			SpotInterruptionRiskScore: 0.8,
		}
		err := pluginsdk.ValidateEstimateCostResponse(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "0.8")
		assert.Contains(t, err.Error(), "STANDARD")
	})
}

// Benchmarks for response validation functions.

// BenchmarkValidateEstimateCostResponse_Valid benchmarks the happy path validation.
func BenchmarkValidateEstimateCostResponse_Valid(b *testing.B) {
	resp := &pbc.EstimateCostResponse{
		Currency:                  "USD",
		CostMonthly:               50.0,
		PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
		SpotInterruptionRiskScore: 0.8,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateEstimateCostResponse(resp)
	}
}

// BenchmarkValidateGetProjectedCostResponse_Valid benchmarks the happy path for the
// more complex GetProjectedCostResponse validation which includes prediction interval
// and confidence level checks.
func BenchmarkValidateGetProjectedCostResponse_Valid(b *testing.B) {
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:                 0.05,
		Currency:                  "USD",
		CostPerMonth:              36.50,
		PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		SpotInterruptionRiskScore: 0.0,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateGetProjectedCostResponse_WithPredictionInterval benchmarks validation
// with all optional fields set (prediction interval + confidence level).
func BenchmarkValidateGetProjectedCostResponse_WithPredictionInterval(b *testing.B) {
	lower := 30.0
	upper := 45.0
	confidence := 0.95
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:               0.05,
		Currency:                "USD",
		CostPerMonth:            36.50,
		PredictionIntervalLower: &lower,
		PredictionIntervalUpper: &upper,
		ConfidenceLevel:         &confidence,
		PricingCategory:         pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// TestValidateMetadataMap tests metadata map validation via GetProjectedCostResponse.
func TestValidateMetadataMap(t *testing.T) {
	// base builds a valid response with the given metadata.
	base := func(m map[string]string) *pbc.GetProjectedCostResponse {
		return &pbc.GetProjectedCostResponse{
			CostPerMonth: 36.50,
			Metadata:     m,
		}
	}

	tests := []struct {
		name    string
		meta    map[string]string
		wantErr error // nil means expect no error
	}{
		{
			name:    "nil_map",
			meta:    nil,
			wantErr: nil,
		},
		{
			name:    "empty_map",
			meta:    map[string]string{},
			wantErr: nil,
		},
		{
			name:    "valid_single_entry",
			meta:    map[string]string{"key": "value"},
			wantErr: nil,
		},
		{
			name: "exactly_32_entries",
			meta: func() map[string]string {
				m := make(map[string]string, 32)
				for i := range 32 {
					m[fmt.Sprintf("key_%02d", i)] = "value"
				}
				return m
			}(),
			wantErr: nil,
		},
		{
			name: "33_entries_too_many",
			meta: func() map[string]string {
				m := make(map[string]string, 33)
				for i := range 33 {
					m[fmt.Sprintf("key_%02d", i)] = "value"
				}
				return m
			}(),
			wantErr: pluginsdk.ErrMetadataTooManyEntries,
		},
		{
			name:    "empty_key",
			meta:    map[string]string{"": "value"},
			wantErr: pluginsdk.ErrMetadataEmptyKey,
		},
		{
			name: "key_exactly_64_bytes",
			meta: map[string]string{
				strings.Repeat("a", 64): "value",
			},
			wantErr: nil,
		},
		{
			name: "key_65_bytes_too_long",
			meta: map[string]string{
				strings.Repeat("a", 65): "value",
			},
			wantErr: pluginsdk.ErrMetadataKeyTooLong,
		},
		{
			name:    "key_with_space_rejected",
			meta:    map[string]string{"key name": "value"},
			wantErr: pluginsdk.ErrMetadataKeyInvalidChar,
		},
		{
			name:    "key_with_del_0x7F_rejected",
			meta:    map[string]string{"key\x7F": "value"},
			wantErr: pluginsdk.ErrMetadataKeyInvalidChar,
		},
		{
			name:    "key_with_control_char_rejected",
			meta:    map[string]string{"key\x01": "value"},
			wantErr: pluginsdk.ErrMetadataKeyInvalidChar,
		},
		{
			name:    "key_with_multibyte_utf8_rejected",
			meta:    map[string]string{"clé": "value"},
			wantErr: pluginsdk.ErrMetadataKeyInvalidChar,
		},
		{
			name:    "key_with_tab_rejected",
			meta:    map[string]string{"key\t": "value"},
			wantErr: pluginsdk.ErrMetadataKeyInvalidChar,
		},
		{
			name: "value_exactly_1024_bytes",
			meta: map[string]string{
				"key": strings.Repeat("v", 1024),
			},
			wantErr: nil,
		},
		{
			name: "value_1025_bytes_too_long",
			meta: map[string]string{
				"key": strings.Repeat("v", 1025),
			},
			wantErr: pluginsdk.ErrMetadataValueTooLong,
		},
		{
			name:    "value_with_invalid_utf8",
			meta:    map[string]string{"key": "value\xff\xfe"},
			wantErr: pluginsdk.ErrMetadataValueNotUTF8,
		},
		{
			name:    "value_with_nul_byte_rejected",
			meta:    map[string]string{"key": "value\x00rest"},
			wantErr: pluginsdk.ErrMetadataValueControlChar,
		},
		{
			name:    "valid_printable_ascii_key",
			meta:    map[string]string{"key.sub-key_123": "value"},
			wantErr: nil,
		},
		{
			name:    "key_lowest_valid_char_0x21",
			meta:    map[string]string{"!": "value"},
			wantErr: nil,
		},
		{
			name:    "key_highest_valid_char_0x7E",
			meta:    map[string]string{"~": "value"},
			wantErr: nil,
		},
		{
			name:    "value_with_tab_rejected",
			meta:    map[string]string{"key": "value\twith tab"},
			wantErr: pluginsdk.ErrMetadataValueControlChar,
		},
		{
			name:    "value_with_newline_rejected",
			meta:    map[string]string{"key": "line1\nline2"},
			wantErr: pluginsdk.ErrMetadataValueControlChar,
		},
		{
			name:    "value_only_nul_rejected",
			meta:    map[string]string{"key": "\x00"},
			wantErr: pluginsdk.ErrMetadataValueControlChar,
		},
		{
			name: "32_entries_max_value_length",
			meta: func() map[string]string {
				m := make(map[string]string, 32)
				for i := range 32 {
					m[fmt.Sprintf("key_%02d", i)] = strings.Repeat("v", 1024)
				}
				return m
			}(),
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := pluginsdk.ValidateGetProjectedCostResponse(base(tc.meta))
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

// BenchmarkValidateGetProjectedCostResponse_WithMetadata32 benchmarks validation
// with a full 32-entry metadata map to characterize iteration cost.
func BenchmarkValidateGetProjectedCostResponse_WithMetadata32(b *testing.B) {
	m := make(map[string]string, 32)
	for i := range 32 {
		m[fmt.Sprintf("key_%02d", i)] = "value"
	}
	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth: 36.50,
		Metadata:     m,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateGetProjectedCostResponse_WithMetadata32_LongKeys benchmarks
// validation with 32 entries using realistic namespaced keys (32-40 bytes).
func BenchmarkValidateGetProjectedCostResponse_WithMetadata32_LongKeys(b *testing.B) {
	m := make(map[string]string, 32)
	for i := range 32 {
		m[fmt.Sprintf("provider.subsystem.metric_name_%02d", i)] = strings.Repeat("v", 128)
	}
	resp := &pbc.GetProjectedCostResponse{
		CostPerMonth: 36.50,
		Metadata:     m,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateGetProjectedCostResponse_Invalid_NaN benchmarks the error path
// for NaN detection which uses math.IsNaN.
func BenchmarkValidateGetProjectedCostResponse_Invalid_NaN(b *testing.B) {
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:                 0.05,
		Currency:                  "USD",
		CostPerMonth:              math.NaN(),
		PricingCategory:           pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
		SpotInterruptionRiskScore: 0.0,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// costBreakdownResponse builds a projected cost response with the given total
// and breakdown for cost_breakdown validation tests and benchmarks.
func costBreakdownResponse(total float64, breakdown map[string]float64) *pbc.GetProjectedCostResponse {
	return &pbc.GetProjectedCostResponse{
		Currency:      "USD",
		CostPerMonth:  total,
		CostBreakdown: breakdown,
	}
}

// uniformCostBreakdown returns n entries of value 1.0 whose keys are padded to keyLen bytes.
func uniformCostBreakdown(n, keyLen int) map[string]float64 {
	m := make(map[string]float64, n)
	for i := range n {
		key := fmt.Sprintf("c%02d", i)
		m[key+strings.Repeat("x", keyLen-len(key))] = 1.0
	}
	return m
}

// TestValidateCostBreakdown tests cost_breakdown validation via GetProjectedCostResponse.
func TestValidateCostBreakdown(t *testing.T) {
	type testCase struct {
		name         string
		resp         *pbc.GetProjectedCostResponse
		wantErr      error // nil means expect no error
		wantContains []string
	}

	tests := []testCase{
		{
			name: "ec2_example",
			resp: costBreakdownResponse(8.392, map[string]float64{"compute": 7.592, "root_volume": 0.80}),
		},
		{
			name: "single_component",
			resp: costBreakdownResponse(8.0, map[string]float64{"storage": 8.0}),
		},
		{
			name: "rounding_within_absolute_tolerance",
			resp: costBreakdownResponse(10.00, map[string]float64{"compute": 6.004, "storage": 4.001}),
		},
		{
			name: "large_total_within_relative_tolerance",
			resp: costBreakdownResponse(50000, map[string]float64{"compute": 30040, "storage": 20000}),
		},
		{
			name: "zero_total_zero_component",
			resp: costBreakdownResponse(0, map[string]float64{"compute": 0}),
		},
		{
			name: "zero_total_within_absolute_tolerance",
			resp: costBreakdownResponse(0, map[string]float64{"compute": 0.005}),
		},
		{
			name: "exactly_32_entries",
			resp: costBreakdownResponse(32, uniformCostBreakdown(32, 8)),
		},
		{
			name: "64_byte_key",
			resp: costBreakdownResponse(1, map[string]float64{"a" + strings.Repeat("b", 63): 1}),
		},
		{
			name: "digits_and_underscores",
			resp: costBreakdownResponse(1, map[string]float64{"data_transfer_2": 1}),
		},
		{
			name:         "sum_mismatch",
			resp:         costBreakdownResponse(8.392, map[string]float64{"compute": 8.0, "root_volume": 1.0}),
			wantErr:      pluginsdk.ErrCostBreakdownSumMismatch,
			wantContains: []string{"GetProjectedCostResponse", "sum 9", "cost_per_month 8.392"},
		},
		{
			name:    "sum_mismatch_zero_total",
			resp:    costBreakdownResponse(0, map[string]float64{"compute": 0.5}),
			wantErr: pluginsdk.ErrCostBreakdownSumMismatch,
		},
		{
			name:    "sum_just_outside_absolute_tolerance",
			resp:    costBreakdownResponse(10.00, map[string]float64{"compute": 6.01, "storage": 4.01}),
			wantErr: pluginsdk.ErrCostBreakdownSumMismatch,
		},
		{
			name:         "negative_value",
			resp:         costBreakdownResponse(0, map[string]float64{"compute": -1}),
			wantErr:      pluginsdk.ErrCostBreakdownInvalidValue,
			wantContains: []string{`"compute"`},
		},
		{
			name:         "nan_value",
			resp:         costBreakdownResponse(8, map[string]float64{"compute": math.NaN()}),
			wantErr:      pluginsdk.ErrCostBreakdownInvalidValue,
			wantContains: []string{`"compute"`},
		},
		{
			name:         "inf_value",
			resp:         costBreakdownResponse(8, map[string]float64{"compute": math.Inf(1)}),
			wantErr:      pluginsdk.ErrCostBreakdownInvalidValue,
			wantContains: []string{`"compute"`},
		},
		{
			name:    "too_many_entries",
			resp:    costBreakdownResponse(33, uniformCostBreakdown(33, 8)),
			wantErr: pluginsdk.ErrCostBreakdownTooManyEntries,
		},
		{
			name: "dry_run_with_breakdown",
			resp: &pbc.GetProjectedCostResponse{
				DryRunResult:  &pbc.DryRunResponse{},
				CostBreakdown: map[string]float64{"compute": 0},
			},
			wantErr: pluginsdk.ErrCostBreakdownWithDryRun,
		},
	}

	for _, key := range []string{
		"RootVolume", "root-volume", "1st", "_x", "", "root volume",
		"a" + strings.Repeat("b", 64), "café",
	} {
		tests = append(tests, testCase{
			name:    fmt.Sprintf("invalid_key_%q", key),
			resp:    costBreakdownResponse(1, map[string]float64{key: 1}),
			wantErr: pluginsdk.ErrCostBreakdownInvalidKey,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pluginsdk.ValidateGetProjectedCostResponse(tt.resp)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			for _, s := range tt.wantContains {
				assert.Contains(t, err.Error(), s)
			}
		})
	}
}

// TestValidateGetProjectedCostResponse_CostBreakdownZeroAlloc guards SC-004:
// validating a full, valid breakdown must not allocate.
func TestValidateGetProjectedCostResponse_CostBreakdownZeroAlloc(t *testing.T) {
	resp := costBreakdownResponse(32, uniformCostBreakdown(32, 64))
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))

	allocs := testing.AllocsPerRun(100, func() {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	})
	assert.Zero(t, allocs)
}

// BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown benchmarks the
// two-component EC2 example.
func BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown(b *testing.B) {
	resp := costBreakdownResponse(8.392, map[string]float64{"compute": 7.592, "root_volume": 0.80})
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown32 benchmarks the
// worst valid case: 32 entries with 64-byte keys.
func BenchmarkValidateGetProjectedCostResponse_WithCostBreakdown32(b *testing.B) {
	resp := costBreakdownResponse(32, uniformCostBreakdown(32, 64))
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateGetProjectedCostResponse_Invalid_CostBreakdownSum benchmarks
// the sum mismatch error path.
func BenchmarkValidateGetProjectedCostResponse_Invalid_CostBreakdownSum(b *testing.B) {
	resp := costBreakdownResponse(8.392, map[string]float64{"compute": 8.0, "root_volume": 1.0})
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// samplePriceOptions returns a 1-year reservation and a 3-year savings plan
// priced against a 0.096/hour Consumption rate (70.08 per month).
func samplePriceOptions() []*pbc.PriceOption {
	return []*pbc.PriceOption{reservationOption(), savingsPlanOption()}
}

// priceOptionsProjected builds a projected cost response for the 0.096/hour
// Consumption price with the given options.
func priceOptionsProjected(options []*pbc.PriceOption) *pbc.GetProjectedCostResponse {
	return &pbc.GetProjectedCostResponse{
		UnitPrice:    0.096,
		Currency:     "USD",
		CostPerMonth: 70.08,
		PriceOptions: options,
	}
}

// priceOptionsEstimate builds an estimate response for the same Consumption
// price with the given options.
func priceOptionsEstimate(options []*pbc.PriceOption) *pbc.EstimateCostResponse {
	return &pbc.EstimateCostResponse{
		Currency:     "USD",
		CostMonthly:  70.08,
		PriceOptions: options,
	}
}

// TestValidateGetProjectedCostResponse_PriceOptions verifies that alternative
// prices are accepted without being summed into cost_per_month, and that they
// are rejected on dry-run responses.
func TestValidateGetProjectedCostResponse_PriceOptions(t *testing.T) {
	t.Run("two_options", func(t *testing.T) {
		resp := priceOptionsProjected(samplePriceOptions())
		require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
		assert.InDelta(t, 70.08, resp.GetCostPerMonth(), 0)
	})

	t.Run("with_cost_breakdown", func(t *testing.T) {
		resp := priceOptionsProjected(samplePriceOptions())
		resp.CostBreakdown = map[string]float64{"compute": 60.08, "os_disk": 10.00}
		require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
	})

	t.Run("dry_run", func(t *testing.T) {
		resp := &pbc.GetProjectedCostResponse{
			DryRunResult: &pbc.DryRunResponse{},
			PriceOptions: samplePriceOptions(),
		}
		err := pluginsdk.ValidateGetProjectedCostResponse(resp)
		require.ErrorIs(t, err, pluginsdk.ErrPriceOptionsWithDryRun)
		assert.Contains(t, err.Error(), "GetProjectedCostResponse")
	})
}

// TestValidateEstimateCostResponse_PriceOptions verifies that alternative
// prices are accepted without being summed into cost_monthly.
func TestValidateEstimateCostResponse_PriceOptions(t *testing.T) {
	options := samplePriceOptions()
	options[0].SavingsFraction = (70.08 - 41.83) / 70.08
	options[1].SavingsFraction = (70.08 - 44.68) / 70.08

	resp := priceOptionsEstimate(options)
	require.NoError(t, pluginsdk.ValidateEstimateCostResponse(resp))
	assert.InDelta(t, 70.08, resp.GetCostMonthly(), 0)
}

// TestValidatePriceOptions_Omitted verifies that a nil or empty list leaves
// both validators' results unchanged, for valid and invalid base responses.
func TestValidatePriceOptions_Omitted(t *testing.T) {
	errString := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}

	for _, options := range [][]*pbc.PriceOption{nil, {}} {
		valid := priceOptionsProjected(options)
		require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(valid))
		require.NoError(t, pluginsdk.ValidateEstimateCostResponse(priceOptionsEstimate(options)))

		negative := priceOptionsProjected(options)
		negative.CostPerMonth = -1
		baseline := priceOptionsProjected(nil)
		baseline.CostPerMonth = -1
		assert.Equal(t,
			errString(pluginsdk.ValidateGetProjectedCostResponse(baseline)),
			errString(pluginsdk.ValidateGetProjectedCostResponse(negative)))

		badRisk := priceOptionsEstimate(options)
		badRisk.SpotInterruptionRiskScore = 0.5
		err := pluginsdk.ValidateEstimateCostResponse(badRisk)
		require.Error(t, err)
		assert.NotErrorIs(t, err, pluginsdk.ErrPriceOptionInvalidValue)
	}
}

// setPriceOptionField returns a mutation that sets one numeric field of a price option.
func setPriceOptionField(field string, v float64) func(*pbc.PriceOption) {
	return func(o *pbc.PriceOption) {
		switch field {
		case "unit_price":
			o.UnitPrice = v
		case "monthly_cost":
			o.MonthlyCost = v
		case "upfront_cost":
			o.UpfrontCost = v
		case "savings_fraction":
			o.SavingsFraction = v
		default:
			panic("unknown price option field " + field)
		}
	}
}

type priceOptionRuleCase struct {
	name        string
	mutate      func(*pbc.PriceOption)
	nilEntry    bool
	zeroPrimary bool
	wantErr     error
	wantText    string
}

// priceOptionRuleCases lists the accepted edge values and every rejected value
// for the entry at index 1.
func priceOptionRuleCases() []priceOptionRuleCase {
	tests := []priceOptionRuleCase{
		{name: "nil_entry", nilEntry: true, wantErr: pluginsdk.ErrPriceOptionNil, wantText: "price_options[1]"},
		{name: "negative_savings_fraction", mutate: setPriceOptionField("savings_fraction", -0.25)},
		{name: "zero_primary", mutate: setPriceOptionField("savings_fraction", 0), zeroPrimary: true},
		{name: "unknown_category", mutate: func(o *pbc.PriceOption) { o.Category = pbc.FocusPricingCategory(99) }},
		{name: "empty_model_and_term", mutate: func(o *pbc.PriceOption) { o.Model, o.Term = "", "" }},
		{name: "zero_upfront_cost", mutate: setPriceOptionField("upfront_cost", 0)},
	}
	for _, field := range []string{"unit_price", "monthly_cost", "upfront_cost"} {
		tests = append(tests, priceOptionRuleCase{
			name:   field + "_negative_zero",
			mutate: setPriceOptionField(field, math.Copysign(0, -1)),
		})
		for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -0.01} {
			tests = append(tests, priceOptionRuleCase{
				name:     fmt.Sprintf("%s_%v", field, v),
				mutate:   setPriceOptionField(field, v),
				wantErr:  pluginsdk.ErrPriceOptionInvalidValue,
				wantText: "price_options[1]." + field,
			})
		}
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		tests = append(tests, priceOptionRuleCase{
			name:     fmt.Sprintf("savings_fraction_%v", v),
			mutate:   setPriceOptionField("savings_fraction", v),
			wantErr:  pluginsdk.ErrPriceOptionInvalidValue,
			wantText: "price_options[1].savings_fraction",
		})
	}
	return tests
}

// TestValidatePriceOptions_Rules runs each per-entry rule through both
// validators, with the entry under test at index 1.
func TestValidatePriceOptions_Rules(t *testing.T) {
	for _, tt := range priceOptionRuleCases() {
		t.Run(tt.name, func(t *testing.T) {
			options := samplePriceOptions()
			if tt.nilEntry {
				options[1] = nil
			} else {
				tt.mutate(options[1])
			}
			projected := priceOptionsProjected(options)
			estimate := priceOptionsEstimate(options)
			if tt.zeroPrimary {
				projected.UnitPrice, projected.CostPerMonth, estimate.CostMonthly = 0, 0, 0
			}

			for name, err := range map[string]error{
				"projected": pluginsdk.ValidateGetProjectedCostResponse(projected),
				"estimate":  pluginsdk.ValidateEstimateCostResponse(estimate),
			} {
				if tt.wantErr == nil {
					require.NoError(t, err, name)
					continue
				}
				require.ErrorIs(t, err, tt.wantErr, name)
				assert.Contains(t, err.Error(), tt.wantText, name)
			}
		})
	}

	t.Run("first_failure_wins", func(t *testing.T) {
		options := samplePriceOptions()
		options[0].UnitPrice = math.NaN()
		options[1] = nil
		err := pluginsdk.ValidateGetProjectedCostResponse(priceOptionsProjected(options))
		require.ErrorIs(t, err, pluginsdk.ErrPriceOptionInvalidValue)
		assert.Contains(t, err.Error(), "price_options[0].unit_price")
	})
}

// fourPriceOptions returns four valid options for allocation tests and benchmarks.
func fourPriceOptions() []*pbc.PriceOption {
	options := samplePriceOptions()
	return append(options,
		&pbc.PriceOption{
			Category:    pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			Model:       "Consumption",
			UnitPrice:   0.096,
			MonthlyCost: 70.08,
		},
		&pbc.PriceOption{
			Category:        pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_DYNAMIC,
			Model:           "Spot",
			UnitPrice:       0.0192,
			MonthlyCost:     14.016,
			SavingsFraction: 0.8,
		},
	)
}

// TestValidatePriceOptions_ZeroAlloc guards FR-014: validating four valid
// options must not allocate in either validator.
func TestValidatePriceOptions_ZeroAlloc(t *testing.T) {
	projected := priceOptionsProjected(fourPriceOptions())
	estimate := priceOptionsEstimate(fourPriceOptions())
	require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(projected))
	require.NoError(t, pluginsdk.ValidateEstimateCostResponse(estimate))

	assert.Zero(t, testing.AllocsPerRun(100, func() {
		_ = pluginsdk.ValidateGetProjectedCostResponse(projected)
	}))
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		_ = pluginsdk.ValidateEstimateCostResponse(estimate)
	}))
}

// BenchmarkValidateGetProjectedCostResponse_WithPriceOptions benchmarks a
// projected cost response with four valid options.
func BenchmarkValidateGetProjectedCostResponse_WithPriceOptions(b *testing.B) {
	resp := priceOptionsProjected(fourPriceOptions())
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateGetProjectedCostResponse(resp)
	}
}

// BenchmarkValidateEstimateCostResponse_WithPriceOptions benchmarks an
// estimate response with four valid options.
func BenchmarkValidateEstimateCostResponse_WithPriceOptions(b *testing.B) {
	resp := priceOptionsEstimate(fourPriceOptions())
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.ValidateEstimateCostResponse(resp)
	}
}
