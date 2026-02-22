package pluginsdk_test

import (
	"math"
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
