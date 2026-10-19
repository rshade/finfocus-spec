# Contract: On-Demand Equivalent Response Field

## Wire contract

```proto
message GetProjectedCostResponse {
  optional double on_demand_equivalent_monthly_cost = 18;
}

message EstimateCostResponse {
  optional double on_demand_equivalent_monthly_cost = 8;
}
```

The excerpt shows only the new fields. Existing fields and RPC signatures remain intact.
Absent means unknown; present zero is a real price. A present value is finite,
non-negative, and valid only for DYNAMIC pricing. It shares the selected cost's currency
and monthly basis.

## Go SDK contract

```go
var ErrOnDemandEquivalentInvalidValue error
var ErrOnDemandEquivalentInvalidCategory error

func CheckOnDemandEquivalentConsistency(
    category pbc.FocusPricingCategory, onDemand *float64,
) []string
func WithOnDemandEquivalent(monthlyCost float64) EstimateCostResponseOption
func WithProjectedCostOnDemandEquivalent(monthlyCost float64) GetProjectedCostResponseOption
```

`ValidateEstimateCostResponse` and `ValidateGetProjectedCostResponse` return the shared
sentinels with `errors.Is` support. Options copy the value without validating or panicking;
callers validate the final response. Check the generated Go pointer for presence because the
getter returns zero for both absent and present zero.

## Conformance contract

```go
func ValidateOnDemandEquivalent(value *float64, category pbc.FocusPricingCategory) error
```

Both conformance response validators call this rule. It returns nil for absence and wraps
the same two sentinels exposed by `pluginsdk`. `MockPlugin.OnDemandEquivalentMonthlyCost`
is copied only to DYNAMIC projected and estimate responses, preserving present zero.

## Consumer contract

A host may calculate savings from the plugin's two monthly values. If the equivalent is
absent, savings are unavailable. The host handles a zero denominator and negative savings.
It does not request a STANDARD twin price or infer one from `price_options`.
