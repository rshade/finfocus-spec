# Data Model: Alternative Retail Price Options

**Feature**: 557-price-options | **Date**: 2026-10-01

## PriceOption (new message)

One alternative retail price for the resource on the parent response. It stands alone: it is never
added to, and never replaces, the parent's selected price.

| # | Field | Type | Meaning | Validation |
|---|-------|------|---------|------------|
| 1 | `category` | `FocusPricingCategory` | Standard, Committed, or Dynamic | None (unknown values accepted) |
| 2 | `model` | `string` | Pricing model name, such as `Consumption`, `Reservation`, `SavingsPlan` | None |
| 3 | `term` | `string` | Empty when the model has no term; otherwise the provider's term, such as `1 Year` | None |
| 4 | `unit_price` | `double` | Price per unit, on the same basis as the parent response | Finite, `>= 0` |
| 5 | `monthly_cost` | `double` | This option's monthly cost | Finite, `>= 0` |
| 6 | `upfront_cost` | `double` | Amount charged up front for the term; 0 when none | Finite, `>= 0` |
| 7 | `savings_fraction` | `double` | `(primary - option) / primary`, unrounded; 0 when primary is 0 | Finite (negative allowed) |

**Primary basis for `savings_fraction`**:

- `GetProjectedCostResponse`: the parent `unit_price` compared to the option's `unit_price`.
- `EstimateCostResponse` (no unit price): the parent `cost_monthly` compared to the option's
  `monthly_cost`.

**Term-total reservations**: the term total goes in `upfront_cost`, and `unit_price` is the hourly
equivalent.

## Parent fields (new)

| Message | Field | Number | Default |
|---------|-------|--------|---------|
| `GetProjectedCostResponse` | `repeated PriceOption price_options` | 16 | empty (no alternatives) |
| `EstimateCostResponse` | `repeated PriceOption price_options` | 6 | empty (no alternatives) |

Held by comment, not `reserved`: `GetProjectedCostResponse` field 17 and `EstimateCostResponse`
field 7 (later per-region price list).

## Relationships

- A parent response has 0..n `PriceOption` entries.
- An entry may describe the selected price itself (`savings_fraction` 0). Hosts must not count it
  as a second charge.
- `cost_breakdown` (projected cost) splits the selected price into components. `price_options` lists
  other prices. Neither affects the other's validation.

## Validation order

`ValidateGetProjectedCostResponse` (new step 9, after `cost_breakdown`), only when
`len(price_options) > 0`:

1. Dry-run result present → `ErrPriceOptionsWithDryRun`.
2. For each entry `i`, in index order (first failure wins):
   1. Entry nil → `ErrPriceOptionNil` (`price_options[i]`).
   2. `unit_price`, then `monthly_cost`, then `upfront_cost`: NaN, ±Inf, or `< 0` →
      `ErrPriceOptionInvalidValue` (`price_options[i].<field> <value>`).
   3. `savings_fraction`: NaN or ±Inf → `ErrPriceOptionInvalidValue`.

`ValidateEstimateCostResponse` (new step 3, after spot risk): the same per-entry checks. There is no
dry-run step, because the message has no dry-run result.

Errors are wrapped with `GetProjectedCostResponse:` or `EstimateCostResponse:`. A valid list
allocates nothing.

## MockPlugin fields (new)

| Field | Type | Behavior |
|-------|------|----------|
| `ProjectedCostPriceOptions` | `[]*pbc.PriceOption` | Deep-copied into each non-dry-run `GetProjectedCost` response. Nil → no options |
| `EstimateCostPriceOptions` | `[]*pbc.PriceOption` | Deep-copied into each `EstimateCost` response. Nil → no options |

Entries are returned as configured; the mock does not validate or compute them.
