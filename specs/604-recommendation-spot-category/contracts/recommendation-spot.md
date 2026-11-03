# Contract: Recommendation Spot Fields

## Proto (additive)

```proto
message Recommendation {
  // ... fields 1-19 unchanged ...

  // pricing_category is the pricing model of the recommended target state.
  // DYNAMIC means the recommendation moves the resource to spot capacity.
  // UNSPECIFIED (default) means the recommendation says nothing about it.
  FocusPricingCategory pricing_category = 20;

  // spot_interruption_risk_score is the interruption probability, 0.0 to 1.0,
  // of the recommended spot target. 0.0 means unknown, exactly as on
  // GetProjectedCostResponse. Non-zero only when pricing_category is DYNAMIC.
  // Validation: pluginsdk.ValidateRecommendation.
  double spot_interruption_risk_score = 21;
}
```

The final comment block expands on the tolerance, zero-means-unknown rule, and the host guidance that risk
only annotates a recommendation and never reorders a list.

## Go surface

| Symbol | Package | Purpose |
|--------|---------|---------|
| `ValidateRecommendationSpot(category, score)` | `sdk/go/testing` | Shared rule, 0 allocs/op on valid input |
| `ErrRecommendationSpotInvalidValue` | `sdk/go/testing`, aliased in `pluginsdk` | Bad number |
| `ErrRecommendationSpotInvalidCategory` | `sdk/go/testing`, aliased in `pluginsdk` | Non-zero risk without DYNAMIC |
| `NewSpotRecommendation(id, risk)` | `sdk/go/testing` | Valid DYNAMIC recommendation for tests |
| `ValidateRecommendation` | `pluginsdk` | Now applies the rule (error prefix `recommendation.spot_interruption_risk_score`) |
| `RPCCorrectness_GetRecommendationsSpot` | conformance, Standard | Applies the rule to returned recommendations |

## Compatibility

- Wire: fields 20 and 21 are new; `buf breaking` must pass against `main`.
- Behaviour: a recommendation with the default pair, or any pair with risk 0.0, validates as before.
- Hosts that ignore the fields keep working; the scorer request validator is unchanged.
