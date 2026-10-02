# Proto Contract: price_options

File: `proto/finfocus/v1/costsource.proto`. All changes are additive. `buf breaking` against `main`
must pass.

## New message: `PriceOption`

Place it directly after `GetProjectedCostResponse`.

```protobuf
// PriceOption is one alternative retail price for the resource on the parent
// response, such as an on-demand, reservation, or savings-plan rate.
//
// A PriceOption is advisory. It is never summed into, and never replaces, the
// parent's selected price (GetProjectedCostResponse.cost_per_month or
// EstimateCostResponse.cost_monthly). It is not a component of that price:
// use cost_breakdown for components.
//
// Example (Azure D2s v5, East US, selected price is Consumption at 0.096/hour):
//   { category: COMMITTED, model: "Reservation", term: "1 Year",
//     unit_price: 0.0573, monthly_cost: 41.83, upfront_cost: 502.00,
//     savings_fraction: 0.403125 }
//
// Validation (pluginsdk.ValidateGetProjectedCostResponse and
// pluginsdk.ValidateEstimateCostResponse):
//   - unit_price, monthly_cost, upfront_cost: finite and >= 0
//   - savings_fraction: finite; may be negative
//   - category, model, term: not checked
message PriceOption {
  // category is the FOCUS pricing category of this option: STANDARD for
  // on-demand, COMMITTED for reservations and savings plans, DYNAMIC for spot.
  FocusPricingCategory category = 1;

  // model is the provider's pricing model name, such as "Consumption",
  // "Reservation", or "SavingsPlan". It is free text; consumers must not
  // reject unknown names.
  string model = 2;

  // term is empty when the model has no term. Otherwise it is the provider's
  // term string, such as "1 Year" or "3 Years".
  string term = 3;

  // unit_price is the price per unit on the same basis (unit and currency) as
  // the parent response. For a reservation published as a term total, this is
  // the hourly equivalent and the total goes in upfront_cost.
  double unit_price = 4;

  // monthly_cost is this option's monthly cost. It is never added to the
  // parent's monthly cost.
  double monthly_cost = 5;

  // upfront_cost is the amount charged up front for the term. Use 0 when there
  // is no upfront charge.
  double upfront_cost = 6;

  // savings_fraction is (primary - this option) / primary, with no rounding,
  // where primary is the parent's selected price:
  //   - GetProjectedCostResponse: parent unit_price vs. this unit_price
  //   - EstimateCostResponse: parent cost_monthly vs. this monthly_cost
  // It is negative when this option costs more, and 0 when the primary is 0.
  // The plugin computes it; the SDK does not recompute or check the formula.
  double savings_fraction = 7;
}
```

## `GetProjectedCostResponse`: field 16

Append after `cost_breakdown = 15`:

```protobuf
  // price_options lists alternative retail prices for the same resource, such
  // as on-demand, reservation, and savings-plan rates.
  //
  // This list is advisory:
  //   - It is never summed into, and never replaces, cost_per_month.
  //   - It is not checked against cost_per_month or cost_breakdown.
  //   - An empty list means the plugin did not supply alternatives.
  //   - It may include an entry for the selected price (savings_fraction 0).
  //     Consumers must not count that entry as a second charge.
  //   - It must be empty on dry-run responses.
  //
  // Field 17 is held for a later per-region price list. Do not use it for
  // anything else.
  //
  // Backward compatibility: empty when not populated; older consumers skip it.
  repeated PriceOption price_options = 16;
```

## `EstimateCostResponse`: field 6 and comment update

Append after `expires_at = 5`:

```protobuf
  // price_options lists alternative retail prices for the estimated resource.
  // Same semantics as GetProjectedCostResponse.price_options: advisory, never
  // summed into or replacing cost_monthly, and empty when the plugin did not
  // supply alternatives. savings_fraction compares monthly costs, because this
  // message has no unit price.
  //
  // Field 7 is held for a later per-region price list. Do not use it for
  // anything else.
  repeated PriceOption price_options = 6;
```

Replace the message's leading comment paragraph about a future breakdown with:

```protobuf
// Future versions may add an optional cost breakdown (e.g., compute vs storage).
// It MUST follow the rules of GetProjectedCostResponse.cost_breakdown
// (key format, non-negative values, sum tolerance) for consistency, and it
// MUST use field 8 or later: field 6 is price_options and field 7 is held.
// A cost breakdown splits cost_monthly into components; price_options lists
// other prices. They are different fields.
```

## Generated code

- Go: `pbc.PriceOption` with `GetCategory`, `GetModel`, `GetTerm`, `GetUnitPrice`,
  `GetMonthlyCost`, `GetUpfrontCost`, `GetSavingsFraction`. `GetPriceOptions() []*PriceOption` on
  both parents.
- TypeScript: `PriceOption` type and `PriceOptionSchema`. `priceOptions: PriceOption[]` on both
  parents.
