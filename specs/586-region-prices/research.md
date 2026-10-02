# Research: Per-Region Retail Prices on Cost Responses

## R1: Message shape and field numbers

- **Decision**: New top-level `message RegionPrice { string region = 1; double unit_price = 2; double
  monthly_cost = 3; string currency = 4; }` in `costsource.proto`. `repeated RegionPrice
  region_prices = 17` on `GetProjectedCostResponse` and `= 7` on `EstimateCostResponse`. Fields 16
  and 6 are left free by comment for `PriceOption` (issue 588), not by `reserved`.
- **Update**: Issue 588 merged first (PR 599) and took fields 16 and 6 for `price_options`. This
  branch was rebased onto it, and both lists now sit on each response. The "left free" comments
  were dropped.
- **Rationale**: The issue fixes the shape and the numbers. A `reserved` statement would forbid the
  tags that issue 588 is going to use.
- **Alternatives considered**: `map<string, double>` keyed by region (it cannot carry the unit price
  and currency per row); reusing `PricingSpec` (far larger than one price row).

## R2: Where the row rules live

- **Decision**: `sdk/go/testing/region_price.go` holds `ValidateRegionPrices(rows) error` and two
  sentinels, `ErrInvalidRegionPrice` and `ErrRegionPricesWithDryRun`. `pluginsdk` re-exports both
  sentinels and calls the same function from `ValidateGetProjectedCostResponse` and
  `ValidateEstimateCostResponse`. The harness validators `ValidateProjectedCostResponse` and
  `ValidateEstimateCostResponse` call it too.
- **Rationale**: `testing` cannot import `pluginsdk`, but `pluginsdk` imports `testing` (the 052
  allocator pattern). One rule set then covers both plugin self-validation and the conformance
  suite (FR-011, FR-012). By contrast, spec 053's `cost_breakdown` lives only in `pluginsdk`, so
  conformance cannot check it.
- **Alternatives considered**: Rules only in `pluginsdk` (repeats 053's conformance gap).

## R3: Row rules

- **Decision**: Fail fast in row order. Reject a nil row; an empty `region`; a `unit_price` or
  `monthly_cost` that is NaN, infinite, or negative; and a `currency` that is empty or not
  `currency.IsValid`. Errors wrap `ErrInvalidRegionPrice` and name `region_prices[i].<field>`. No
  sum rule, no comparison with the parent, and no duplicate check.
- **Rationale**: These are the issue's three rules plus two the spec adds under Assumptions:
  non-negative (matches the parent's `cost_per_month` rule) and ISO 4217 (the issue names the code
  set). `currency.IsValid` is a map lookup with 0 allocations. A duplicate check would need a map or
  a quadratic scan, and the issue does not ask for one.
- **Alternatives considered**: A 3-character currency check like the harness parent check (weaker
  than the ISO rule the issue names); collecting all row errors (other validators here fail fast).

## R4: Zero-allocation path

- **Decision**: Guard each call with `if rows := resp.GetRegionPrices(); len(rows) > 0` at the call
  site, as `cost_breakdown` does. A/B benchmark `BenchmarkValidateGetProjectedCostResponse_Valid` and
  `BenchmarkValidateEstimateCostResponse_Valid` against a `main` worktree with prebuilt test binaries.
  Add `_WithRegionPrices` benchmarks that report 0 allocs/op.
- **Rationale**: CLAUDE.md (053) records that a non-inlined call alone costs more than 10% on the
  about-6 ns `_Valid` benchmark, and that sequential runs drift by 20% or more on a loaded machine.
- **Measured** (10 interleaved rounds of prebuilt binaries, median of
  `BenchmarkValidateGetProjectedCostResponse_Valid`): main 5.65 ns; branch with the region check compiled
  out 6.08 ns; branch 6.30 ns. About 0.45 ns comes from the changed binary layout (the new proto field and
  new code). The guard itself costs about 0.2 ns. Moving the error path into
  `validateProjectedRegionPrices` did not change the median, but it keeps the inline cost to one length
  check. `ValidateEstimateCostResponse_Valid` stayed within noise (2.50 ns versus 2.54 ns). Every
  variant is 0 allocs/op, and the four-row benchmarks are about 34 ns and 29 ns, both at 0 allocs/op.

## R5: Dry run

- **Decision**: A projected cost response that carries `dry_run_result` and any region row fails with
  `ErrRegionPricesWithDryRun`. `EstimateCostResponse` has no dry-run result, so the rule does not
  apply there.
- **Rationale**: Mirrors `ErrCostBreakdownWithDryRun`. A dry run reports field support, not prices.

## R6: SDK options and the mock

- **Decision**: `pluginsdk.WithProjectedCostRegionPrices(rows ...*pbc.RegionPrice)` and
  `pluginsdk.WithEstimateCostRegionPrices(rows ...*pbc.RegionPrice)`. Each clones every row with
  `proto.Clone` so later caller edits do not leak in, sets nil for no rows, and does not validate
  (the same as `WithProjectedCostBreakdown`). `MockPlugin.RegionPrices []*pbc.RegionPrice` is cloned
  onto non-dry-run projected cost responses and onto estimate responses.
- **Rationale**: Follows the 053 option and mock-field pattern. Cloning matches `maps.Clone` in 053.

## R7: TypeScript

- **Decision**: No hand-written source change. Regenerated types gain `regionPrices: RegionPrice[]`.
  Extend the msw `GetProjectedCost` handler and the client integration test to read two rows.
- **Rationale**: Mirrors 053, which tested `costBreakdown` the same way.

## R8: Agent context

- **Decision**: Add the CLAUDE.md entries by hand. Do not run `update-agent-context.sh`.
- **Rationale**: The script drops the first line of wrapped "Recent Changes" entries.
- **After rebasing onto PR 599** (10 interleaved rounds against the new main, medians): projected
  6.01 ns to 6.33 ns, estimate 2.35 ns to 2.67 ns, all 0 allocs/op. The region guard sits next to
  the `price_options` guard, and both keep their error wrapping out of line.
