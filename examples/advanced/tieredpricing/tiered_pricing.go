// tiered_pricing.go demonstrates tiered pricing calculation patterns for
// cloud resources like storage that have volume-based pricing tiers.
//
// This example implements the TieredPricingCalculator pattern described
// in docs/ADVANCED_PATTERNS.md and shows how to:
// - Parse pricing tiers from a PricingSpec
// - Calculate costs across multiple tiers
// - Generate detailed cost breakdowns
//
// Reference: AWS S3 Standard storage pricing as of 2024
// https://aws.amazon.com/s3/pricing/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

// PricingTier represents a single pricing tier with quantity bounds and rate.
type PricingTier struct {
	MinQuantity float64 `json:"min_quantity"`
	MaxQuantity float64 `json:"max_quantity"` // 0 means unlimited
	RatePerUnit float64 `json:"rate_per_unit"`
	Description string  `json:"description"`
}

// TierBreakdown provides details about cost in each tier.
type TierBreakdown struct {
	Tier        PricingTier
	UsageInTier float64
	TierCost    float64
}

// TieredPricingCalculator calculates costs across volume-based pricing tiers.
type TieredPricingCalculator struct {
	tiers    []PricingTier
	currency string
}

// ErrEmptyCurrency indicates the currency parameter was empty.
var ErrEmptyCurrency = errors.New("currency cannot be empty")

// ErrTiersNotSorted indicates tiers are not sorted by MinQuantity.
var ErrTiersNotSorted = errors.New("tiers must be sorted by MinQuantity in ascending order")

// NewTieredPricingCalculator creates a calculator with the given tiers.
// Returns an error if currency is empty or tiers are not sorted by MinQuantity.
func NewTieredPricingCalculator(tiers []PricingTier, currency string) (*TieredPricingCalculator, error) {
	if currency == "" {
		return nil, ErrEmptyCurrency
	}

	// Validate tiers are sorted by MinQuantity
	for i := 1; i < len(tiers); i++ {
		if tiers[i].MinQuantity <= tiers[i-1].MinQuantity {
			return nil, ErrTiersNotSorted
		}
	}

	return &TieredPricingCalculator{
		tiers:    tiers,
		currency: currency,
	}, nil
}

// MustNewTieredPricingCalculator creates a calculator or panics on invalid input.
// Use for initialization with known-valid tiers.
func MustNewTieredPricingCalculator(tiers []PricingTier, currency string) *TieredPricingCalculator {
	calc, err := NewTieredPricingCalculator(tiers, currency)
	if err != nil {
		panic(fmt.Sprintf("invalid tiered pricing calculator: %v", err))
	}
	return calc
}

// CalculateCost computes the total cost for a given usage quantity.
// Returns the total cost and a breakdown by tier.
// Negative usage values are treated as zero.
func (c *TieredPricingCalculator) CalculateCost(usage float64) (float64, []TierBreakdown) {
	// Defensive: treat negative usage as zero
	if usage < 0 {
		return 0, nil
	}

	var totalCost float64
	// Pre-allocate breakdown slice for better performance
	breakdown := make([]TierBreakdown, 0, len(c.tiers))

	for _, tier := range c.tiers {
		// Calculate usage within this tier
		tierMax := tier.MaxQuantity
		if tierMax == 0 {
			tierMax = usage + 1 // Unlimited tier
		}

		// Skip if usage hasn't reached this tier yet
		if usage <= tier.MinQuantity {
			continue
		}

		// Calculate quantity in this tier
		usageInTier := min(usage, tierMax) - tier.MinQuantity
		if usageInTier <= 0 {
			continue
		}

		tierCost := usageInTier * tier.RatePerUnit
		totalCost += tierCost
		breakdown = append(breakdown, TierBreakdown{
			Tier:        tier,
			UsageInTier: usageInTier,
			TierCost:    tierCost,
		})
	}

	return totalCost, breakdown
}

// PricingSpec represents a pricing specification with tiered pricing.
type PricingSpec struct {
	Provider     string        `json:"provider"`
	ResourceType string        `json:"resource_type"`
	BillingMode  string        `json:"billing_mode"`
	RatePerUnit  float64       `json:"rate_per_unit"`
	Currency     string        `json:"currency"`
	PricingTiers []PricingTier `json:"pricing_tiers"`
}

// demonstrateTieredPricing shows tiered pricing calculation with different usage levels.
//
//nolint:mnd // pricing data is intentionally hard-coded for demonstration purposes
func demonstrateTieredPricing(logger *slog.Logger) {
	// AWS S3 Standard storage pricing tiers (us-east-1, as of 2024)
	tiers := []PricingTier{
		{
			MinQuantity: 0,
			MaxQuantity: 50000, // First 50 TB
			RatePerUnit: 0.023,
			Description: "First 50 TB / Month",
		},
		{
			MinQuantity: 50000,
			MaxQuantity: 450000, // Next 400 TB
			RatePerUnit: 0.022,
			Description: "Next 400 TB / Month",
		},
		{
			MinQuantity: 450000,
			MaxQuantity: 0, // Unlimited (over 450 TB)
			RatePerUnit: 0.021,
			Description: "Over 450 TB / Month",
		},
	}

	calc := MustNewTieredPricingCalculator(tiers, "USD")

	// Test cases with different usage levels
	testCases := []struct {
		name    string
		usageGB float64
	}{
		{"Small (10 GB)", 10},
		{"Medium (100 GB)", 100},
		{"Large (1 TB)", 1000},
		{"First tier boundary (50 TB)", 50000},
		{"Cross tier (100 TB)", 100000},
		{"High volume (500 TB)", 500000},
	}

	logger.Info("AWS S3 Standard Storage Tiered Pricing Demo")

	for _, tc := range testCases {
		totalCost, breakdown := calc.CalculateCost(tc.usageGB)

		// Build breakdown attributes
		breakdownAttrs := make([]any, 0, len(breakdown)*3)
		for i, b := range breakdown {
			breakdownAttrs = append(breakdownAttrs,
				slog.Group(fmt.Sprintf("tier_%d", i+1),
					slog.String("description", b.Tier.Description),
					slog.Float64("usage_gb", b.UsageInTier),
					slog.Float64("cost", b.TierCost),
				),
			)
		}

		logger.Info("Cost calculation",
			slog.String("usage", tc.name),
			slog.Float64("usage_gb", tc.usageGB),
			slog.Float64("total_cost", totalCost),
			slog.Group("breakdown", breakdownAttrs...),
		)
	}
}

// demonstratePricingSpecParsing shows how to parse tiered pricing from a JSON spec.
// Returns an error if parsing fails, demonstrating proper error handling patterns.
func demonstratePricingSpecParsing(logger *slog.Logger) error {
	// Example PricingSpec JSON (like examples/specs/aws-s3-tiered-pricing.json)
	specJSON := `{
		"provider": "aws",
		"resource_type": "s3",
		"billing_mode": "tiered",
		"rate_per_unit": 0.023,
		"currency": "USD",
		"pricing_tiers": [
			{"min_quantity": 0, "max_quantity": 50000, "rate_per_unit": 0.023, "description": "First 50 TB / Month"},
			{"min_quantity": 50000, "max_quantity": 450000, "rate_per_unit": 0.022, "description": "Next 400 TB / Month"},
			{"min_quantity": 450000, "max_quantity": 0, "rate_per_unit": 0.021, "description": "Over 450 TB / Month"}
		]
	}`

	var spec PricingSpec
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return fmt.Errorf("failed to parse pricing spec: %w", err)
	}

	logger.Info("Parsed PricingSpec",
		slog.String("provider", spec.Provider),
		slog.String("resource_type", spec.ResourceType),
		slog.String("billing_mode", spec.BillingMode),
		slog.Int("tier_count", len(spec.PricingTiers)),
	)

	// Create calculator from parsed spec
	calc, err := NewTieredPricingCalculator(spec.PricingTiers, spec.Currency)
	if err != nil {
		return fmt.Errorf("failed to create calculator: %w", err)
	}

	// Calculate cost for 75 TB
	usage := 75000.0 // 75 TB in GB
	totalCost, breakdown := calc.CalculateCost(usage)

	logger.Info("Cost for 75 TB storage",
		slog.Float64("total_cost", totalCost),
		slog.String("currency", spec.Currency),
		slog.Int("tiers_used", len(breakdown)),
	)

	for _, b := range breakdown {
		logger.Info("Tier breakdown",
			slog.String("tier", b.Tier.Description),
			slog.Float64("cost", b.TierCost),
		)
	}

	return nil
}

// demonstrateGCPStorageTiers shows GCP Cloud Storage tiered pricing.
//
//nolint:mnd // pricing data is intentionally hard-coded for demonstration purposes
func demonstrateGCPStorageTiers(logger *slog.Logger) {
	// GCP Cloud Storage Standard pricing tiers (us-multi-region, as of 2024)
	tiers := []PricingTier{
		{
			MinQuantity: 0,
			MaxQuantity: 50000, // First 50 TB
			RatePerUnit: 0.020,
			Description: "First 50 TB / Month",
		},
		{
			MinQuantity: 50000,
			MaxQuantity: 450000, // Next 400 TB
			RatePerUnit: 0.019,
			Description: "Next 400 TB / Month",
		},
		{
			MinQuantity: 450000,
			MaxQuantity: 0, // Over 450 TB
			RatePerUnit: 0.018,
			Description: "Over 450 TB / Month",
		},
	}

	calc := MustNewTieredPricingCalculator(tiers, "USD")

	logger.Info("GCP Cloud Storage Standard Tiered Pricing Demo")

	// Compare cost for same usage across providers
	usage := 100000.0 // 100 TB
	gcpCost, gcpBreakdown := calc.CalculateCost(usage)

	logger.Info("GCP cost for 100 TB",
		slog.Float64("total_cost", gcpCost),
		slog.Int("tiers_used", len(gcpBreakdown)),
	)

	for _, b := range gcpBreakdown {
		logger.Info("Tier breakdown",
			slog.String("tier", b.Tier.Description),
			slog.Float64("usage_gb", b.UsageInTier),
			slog.Float64("rate", b.Tier.RatePerUnit),
			slog.Float64("cost", b.TierCost),
		)
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Check for quiet mode (for testing)
	quiet := len(os.Args) > 1 && os.Args[1] == "--quiet"
	if quiet {
		logger.Info("Tiered pricing examples executed successfully")
		return
	}

	demonstrateTieredPricing(logger)
	if err := demonstratePricingSpecParsing(logger); err != nil {
		logger.Error("Pricing spec parsing demo failed", "error", err)
		os.Exit(1)
	}
	demonstrateGCPStorageTiers(logger)

	logger.Info("Key Takeaways",
		slog.String("point_1", "Tiered pricing rewards higher usage with lower per-unit rates"),
		slog.String("point_2", "Always calculate cost per tier, not using a single rate"),
		slog.String("point_3", "Parse pricing_tiers from PricingSpec for accurate calculations"),
		slog.String("point_4", "Consider cross-provider comparison for cost optimization"),
	)
}
