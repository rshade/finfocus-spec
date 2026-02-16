# Data Model: Caching Hint (expires_at) for Cost Results

**Feature Branch**: `045-caching-hint-expires-at`
**Date**: 2026-02-16

## Entity Changes

### ActualCostResult (modified)

| Field | Type | Number | Status | Description |
|-------|------|--------|--------|-------------|
| timestamp | google.protobuf.Timestamp | 1 | existing | Point-in-time or bucket start |
| cost | double | 2 | existing | Total cost for the period |
| usage_amount | double | 3 | existing | Usage amount |
| usage_unit | string | 4 | existing | Unit of usage |
| source | string | 5 | existing | Data source identifier |
| focus_record | FocusCostRecord | 6 | existing | FOCUS 1.2 format record |
| impact_metrics | repeated ImpactMetric | 7 | existing | Sustainability metrics |
| **expires_at** | **google.protobuf.Timestamp** | **8** | **new** | **When this cost data expires** |

**Validation rules**:

- Nil/zero: No caching guidance (caller should always re-fetch)
- Past timestamp: Data is immediately stale
- Future timestamp: Data is valid until this time
- No upper bound enforcement (callers may apply their own max TTL)

**State transitions**: N/A (stateless field, no lifecycle)

### GetProjectedCostResponse (modified)

| Field | Type | Number | Status | Description |
|-------|------|--------|--------|-------------|
| unit_price | double | 1 | existing | Price per unit |
| currency | string | 2 | existing | Pricing currency |
| cost_per_month | double | 3 | existing | Monthly cost estimate |
| billing_detail | string | 4 | existing | Billing context |
| impact_metrics | repeated ImpactMetric | 5 | existing | Sustainability metrics |
| growth_type | GrowthType | 6 | existing | Growth model hint |
| dry_run_result | DryRunResponse | 7 | existing | DryRun field mappings |
| pricing_category | FocusPricingCategory | 8 | existing | Pricing model category |
| spot_interruption_risk_score | double | 9 | existing | Spot risk score |
| prediction_interval_lower | optional double | 10 | existing | Prediction lower bound |
| prediction_interval_upper | optional double | 11 | existing | Prediction upper bound |
| confidence_level | optional double | 12 | existing | Confidence level |
| **expires_at** | **google.protobuf.Timestamp** | **13** | **new** | **When this projection expires** |

**Validation rules**: Same as ActualCostResult.expires_at above.

## Relationships

- `ActualCostResult.expires_at` is per-result (within repeated `results` in `GetActualCostResponse`)
- `GetProjectedCostResponse.expires_at` is per-response (single response, single expiration)
- Both use `google.protobuf.Timestamp` (already imported in the proto file)
- No cross-entity dependencies or foreign key relationships

## SDK Helper Types

### Response Option Functions (Go)

| Function | Applies To | Description |
|----------|-----------|-------------|
| `WithProjectedCostExpiresAt(t time.Time)` | GetProjectedCostResponse | Sets expires_at on projected response |

Note: `ActualCostResult.ExpiresAt` is set via direct proto field access (no wrapper needed).

### Expiration Check Helpers (Go)

| Function | Input | Output | Description |
|----------|-------|--------|-------------|
| `IsActualCostExpired(result, now)` | *ActualCostResult, time.Time | bool | True if result has expired |
| `ActualCostExpiresAt(result)` | *ActualCostResult | (time.Time, bool) | Returns expiration time if set |
| `IsProjectedCostExpired(resp, now)` | *GetProjectedCostResponse, time.Time | bool | True if response has expired |
| `ProjectedCostExpiresAt(resp)` | *GetProjectedCostResponse | (time.Time, bool) | Returns expiration time if set |

## Backward Compatibility

- New fields use the next available field numbers (8 and 13)
- No existing field numbers are modified or reserved
- No existing field types are changed
- Nil/zero default means "no caching guidance" which preserves existing behavior
- Existing plugins that don't set the field continue to work identically
