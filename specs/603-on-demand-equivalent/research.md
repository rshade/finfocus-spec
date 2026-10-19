# Research: On-Demand Equivalent for Spot Results

## Dedicated optional field

**Decision**: Add `optional double on_demand_equivalent_monthly_cost` at field 18 of
`GetProjectedCostResponse` and field 8 of `EstimateCostResponse`.

**Rationale**: Presence distinguishes unknown from real zero. `price_options` may contain
multiple STANDARD entries, none identified as the exact counterpart of the DYNAMIC result.

**Alternatives considered**: A `price_options` convention would need free-text matching and
could not express the DYNAMIC-only rule. A second STANDARD request has no request-side category
channel and is explicitly out of scope.

## Shared validation rule

**Decision**: Put the rule and sentinels in `sdk/go/testing/on_demand_equivalent.go`, then
alias the sentinels in `pluginsdk`.

**Rationale**: This follows `region_prices` and keeps conformance and plugin validation in
agreement. A nil pointer is a valid, allocation-free legacy path. For a present value, check
NaN/Inf and negativity before category.

**Alternatives considered**: Duplicating rules risks drift. Warning instead of failing a
category mismatch could let contradictory data reach hosts.

## Advisory pricing relationship

**Decision**: Do not compare the equivalent with the selected spot cost or `price_options`.
Never sum it into the primary cost.

**Rationale**: Spot can briefly exceed on-demand, and STANDARD options may differ by OS or
tenancy. Core owns savings math, including negative results and a zero denominator.

**Alternatives considered**: SDK savings calculations or cross-checks would add a second
pricing definition and reject valid plugin output.

## Compatibility evidence

**Decision**: Test absent and present zero over bufconn and inside batch serialization,
test unknown-field behavior with a minimal old decoder, and run Buf breaking checks.

**Rationale**: Getter-only assertions cannot prove presence. An additive field must leave
legacy plugins valid and older hosts able to read all prior fields.
