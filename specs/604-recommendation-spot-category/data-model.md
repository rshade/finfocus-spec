# Data Model: Spot Pricing Category and Interruption Risk on Recommendation

## Recommendation (existing message, two additions)

| Field | Number | Type | Presence | Meaning |
|-------|--------|------|----------|---------|
| `pricing_category` | 20 | `FocusPricingCategory` | UNSPECIFIED (0) = silent | Pricing model of the recommended target state. DYNAMIC means spot capacity. |
| `spot_interruption_risk_score` | 21 | `double` | 0.0 = unknown | Interruption probability of the recommended spot target, 0.0 to 1.0. |

Existing fields 1 to 19 are unchanged. Savings stay in `impact` (`RecommendationImpact`). The new values
are never summed with or compared against impact figures or `confidence_score`.

## Validation rules (applied in this order)

1. `spot_interruption_risk_score` is NaN or infinite: `ErrRecommendationSpotInvalidValue`.
2. `|score| < 1e-9`: valid for every category (fast path).
3. `score < -1e-9` or `score > 1.0`: `ErrRecommendationSpotInvalidValue`.
4. `score` non-zero and category is not DYNAMIC: `ErrRecommendationSpotInvalidCategory`.

## State table

| Category | Risk 0.0 | Risk in (0, 1] | Risk NaN, Inf, < 0, > 1 |
|----------|----------|----------------|--------------------------|
| UNSPECIFIED | valid (legacy default) | invalid category | invalid value |
| STANDARD | valid | invalid category | invalid value |
| COMMITTED | valid | invalid category | invalid value |
| DYNAMIC | valid (risk unknown) | valid | invalid value |

`action_type` is unconstrained by either field.

## Relationships

- Uses the existing `FocusPricingCategory` enum (the one on cost responses and `FocusCostRecord`).
- `ScoreRecommendationsRequest.recommendations` carries the same message; the scorer validator does not
  apply these rules (FR-016).
