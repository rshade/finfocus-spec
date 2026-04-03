# Data Model: EstimateCost expires_at Cache-Hint Parity

**Feature Branch**: `049-estimate-cost-expires-at`
**Date**: 2026-04-02

## Entity Changes

### EstimateCostResponse (modified)

| Field | Type | Number | Status | Description |
|-------|------|--------|--------|-------------|
| currency | string | 1 | existing | ISO 4217 currency code |
| cost_monthly | double | 2 | existing | Estimated monthly cost |
| pricing_category | FocusPricingCategory | 3 | existing | Pricing model category |
| spot_interruption_risk_score | double | 4 | existing | Spot instance interruption probability |
| **expires_at** | **google.protobuf.Timestamp** | **5** | **new** | **When this estimate expires** |

**Validation rules**:

- Nil/zero: No caching guidance (caller should always re-fetch)
- Past timestamp: Data is immediately stale
- Future timestamp: Data is valid until this time
- No upper bound enforcement (callers may apply their own max TTL)

**State transitions**: N/A (stateless field, no lifecycle)

## Relationships

- `EstimateCostResponse.expires_at` is per-response (single response, single expiration)
- Mirrors `GetProjectedCostResponse.expires_at` (field 13) and `ActualCostResult.expires_at`
  (field 8) added in spec 045
- Uses `google.protobuf.Timestamp` (already imported in the proto file)
- `BatchGetCostResponse` gains access through the existing `CostData.estimate` oneof wrapper
- No cross-entity dependencies or foreign key relationships

## SDK Helper Types

### Functional Option (Go)

| Function | Applies To | Description |
|----------|-----------|-------------|
| `WithEstimateCostExpiresAt(t time.Time)` | EstimateCostResponse | Sets expires_at on estimate response |

Note: Uses the existing `EstimateCostResponseOption` type defined in `helpers.go`. The option
integrates with `NewEstimateCostResponse()` builder. No separate `Apply*` function is needed
(builder pattern, consistent with `GetProjectedCostResponse`).

### Expiration Check Helpers (Go)

| Function | Input | Output | Description |
|----------|-------|--------|-------------|
| `IsEstimateCostExpired(resp, now)` | *EstimateCostResponse, time.Time | bool | True if estimate has expired |
| `EstimateCostExpiresAt(resp)` | *EstimateCostResponse | (time.Time, bool) | Returns expiration time if set |

### Mock Plugin Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `EstimateCostExpiresAtDuration` | time.Duration | 0 (nil) | Duration relative to now for expires_at |

## Backward Compatibility

- New field uses the next available field number (5)
- No existing field numbers are modified or reserved
- No existing field types are changed
- Nil/zero default means "no caching guidance" which preserves existing behavior
- Existing plugins that don't set the field continue to work identically
- `buf breaking` check will pass (additive-only change)

## Complete expires_at Parity Table

After this feature, all three cost response types have symmetric `expires_at` support:

| Response Type | Field Number | SDK Reader | SDK Checker | SDK Option | Mock Config |
|---------------|-------------|------------|-------------|------------|-------------|
| ActualCostResult | 8 | `ActualCostExpiresAt()` | `IsActualCostExpired()` | `WithActualCostResultExpiresAt()` | `ExpiresAtDuration` |
| GetProjectedCostResponse | 13 | `ProjectedCostExpiresAt()` | `IsProjectedCostExpired()` | `WithProjectedCostExpiresAt()` | `ProjectedCostExpiresAtDuration` |
| **EstimateCostResponse** | **5** | **`EstimateCostExpiresAt()`** | **`IsEstimateCostExpired()`** | **`WithEstimateCostExpiresAt()`** | **`EstimateCostExpiresAtDuration`** |
