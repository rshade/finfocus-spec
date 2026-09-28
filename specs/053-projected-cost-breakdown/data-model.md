# Data Model: Projected Cost Breakdown

**Feature**: 053-projected-cost-breakdown | **Date**: 2026-09-27

## Entity: GetProjectedCostResponse (extended)

This entity already exists. It gains one field. All existing fields (1 to 14) keep their numbers and
meaning.

| Field | Number | Type | Change |
|-------|--------|------|--------|
| `cost_per_month` | 3 | `double` | Unchanged. When a breakdown is present, it is the value the breakdown's sum is checked against |
| `currency` | 2 | `string` | Unchanged. It is the currency for every breakdown value |
| `dry_run_result` | 7 | `DryRunResponse` | Unchanged. When it is set, the breakdown must be empty |
| `metadata` | 14 | `map<string,string>` | Unchanged. The breakdown uses the same entry limit |
| **`cost_breakdown`** | **15** | **`map<string, double>`** | **New** |

## Entity: Cost Breakdown (`cost_breakdown`)

An unordered map from component name to monthly cost. When the map is non-empty, its values partition
`cost_per_month`.

### States

| State | Condition | Meaning |
|-------|-----------|---------|
| Absent | `len == 0` (proto3 cannot tell unset from empty) | No breakdown available. Valid with any total |
| Present | `1 <= len <= 32` | The total is broken into named components |

The field changes only when a plugin builds a response. Responses are immutable once returned.

### Component name (map key)

| Rule | Constraint | Error |
|------|------------|-------|
| Length | 1 to 64 bytes | `ErrCostBreakdownInvalidKey` |
| First byte | `a` to `z` | `ErrCostBreakdownInvalidKey` |
| Remaining bytes | `a-z`, `0-9`, `_` | `ErrCostBreakdownInvalidKey` |
| Uniqueness | Guaranteed by the map type | n/a |

Recommended names (documented, not validated): `compute`, `storage`, `root_volume`, `network`,
`license`, `request`, `data_transfer`.

### Component value (map value)

| Rule | Constraint | Error |
|------|------------|-------|
| Finite | Not NaN, not ±Inf | `ErrCostBreakdownInvalidValue` |
| Sign | `>= 0` (discounts already applied) | `ErrCostBreakdownInvalidValue` |
| Unit | Monthly cost in the response `currency`, on the same basis as `cost_per_month` (730 h) | Documented only |

### Whole-map rules

| Rule | Constraint | Error |
|------|------------|-------|
| Entry count | `<= 32` | `ErrCostBreakdownTooManyEntries` |
| Sum | `\|Σ values − cost_per_month\| <= max(0.01, 0.001 × cost_per_month)` | `ErrCostBreakdownSumMismatch` |
| Dry run | Must be empty when `dry_run_result != nil` | `ErrCostBreakdownWithDryRun` |

### Validation order (in `ValidateGetProjectedCostResponse`)

1. The existing `cost_per_month` NaN, Inf and negative checks run first, so the tolerance is computed
   from a finite, non-negative total.
2. The existing prediction interval, confidence, spot risk, `expires_at` and metadata checks follow,
   unchanged.
3. **New** `validateCostBreakdown(breakdown, costPerMonth, isDryRun)`:
   1. If the map is empty, return nil (zero allocations).
   2. If it is a dry-run response, return `ErrCostBreakdownWithDryRun`.
   3. If there are more than 32 entries, return `ErrCostBreakdownTooManyEntries`.
   4. For each entry, validate the key, then the value, and add the value to the sum.
   5. Check the sum against the tolerance.

## Entity: MockPlugin (testing, extended)

| Field | Type | Meaning |
|-------|------|---------|
| **`ProjectedCostBreakdown`** | `map[string]float64` | Component weights. When non-nil, the values are scaled so they sum to the computed `cost_per_month`. Weights must be finite and non-negative with a positive sum, or `GetProjectedCost` returns `FailedPrecondition`. When nil, the response has no breakdown |

## Relationships

- `BatchCostResponse` → `CostData.projected_cost` (a `GetProjectedCostResponse`) carries the breakdown
  unchanged. No batch-specific rules apply.
- `EstimateCostResponse`: no change in this feature. A future breakdown there must reuse the rules above
  (spec Assumptions).
