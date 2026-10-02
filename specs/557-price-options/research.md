# Research: Alternative Retail Price Options

**Feature**: 557-price-options | **Date**: 2026-10-01

The Technical Context had no NEEDS CLARIFICATION items. Each decision below resolves a design
question the spec left to planning, or confirms a spec default against the code.

## R1. Message placement and field numbers

- **Decision**: Define `PriceOption` in `proto/finfocus/v1/costsource.proto`, next to
  `GetProjectedCostResponse`. Use `price_options = 16` on `GetProjectedCostResponse` and
  `price_options = 6` on `EstimateCostResponse`. The `PriceOption` fields are numbered 1-7 as in the
  issue.
- **Rationale**: Both parents are in `costsource.proto`. 16 and 6 are the next free numbers (15 is
  `cost_breakdown`, 5 is `expires_at`). `FocusPricingCategory` is already imported from
  `enums.proto`.
- **Alternatives considered**: A separate `pricing.proto`. Rejected: one small message does not
  justify a new file and a new import in generated code.

## R2. Holding fields 17 and 7 for the per-region list

- **Decision**: Hold them with a comment on each message ("Field 17 is held for a per-region price
  list. Do not use it for anything else."). Do not write `reserved 17;`.
- **Rationale**: `reserved` means "never use again". The later feature would have to delete the
  reservation, and `buf breaking` flags deleting a reserved range. `focus.proto` already holds
  fields 69-80 by comment for the same reason.
- **Alternatives considered**: `reserved 17;` now, then remove it later with a `buf` ignore.
  Rejected: it adds a breaking-change exception for no benefit.

## R3. Updating the EstimateCostResponse future-breakdown comment

- **Decision**: Keep the comment saying a future breakdown must follow `cost_breakdown` rules. Add
  that fields 6 (`price_options`) and 7 (held) are taken, so a breakdown uses 8 or later, and that a
  breakdown and `price_options` are different things.
- **Rationale**: FR-007. The issue chose 6 for this feature specifically so the breakdown would not
  take it.

## R4. What the validator checks (and does not)

- **Decision**: For each entry, in order: not nil; `unit_price`, `monthly_cost`, and `upfront_cost`
  finite and `>= 0`; `savings_fraction` finite. No check on `category`, `model`, or `term`. No check
  that `savings_fraction` matches the formula, that it is `<= 1`, or that any value relates to the
  primary cost.
- **Rationale**:
  - FR-009/010/012 and constitution Principle III. Recomputing the fraction would need the primary
    unit price, which `EstimateCostResponse` does not have, and float equality would need a
    tolerance policy.
  - With non-negative prices and a positive primary, the formula cannot exceed 1. A plugin that
    reports more than 1 has a calculation bug, but the SDK does not check plugin arithmetic.
  - An unknown `category` value is accepted, as the parent `pricing_category` is (forward
    compatibility).
- **Alternatives considered**: Reject `savings_fraction > 1`. Rejected for this release because it
  partly recomputes plugin math. It can be added later as a stricter rule, since tightening a rule
  only affects plugins that are already wrong.

## R5. Zero-allocation validation

- **Decision**: `validatePriceOptions(opts []*pbc.PriceOption, isDryRun bool) error` loops over the
  slice by index. Each call site guards with `if opts := resp.GetPriceOptions(); len(opts) > 0`.
  Errors use `fmt.Errorf("%w: price_options[%d].unit_price %v must be finite and non-negative",
  ...)`, so they allocate only on failure.
- **Rationale**: 053 measured that a non-inlined call alone costs more than 10% on the ~6 ns
  `_Valid` benchmark; the guard keeps the empty path at its current cost. A slice index loop does
  not allocate. Benchmarks: `BenchmarkValidateGetProjectedCostResponse_WithPriceOptions` (4
  entries) and `BenchmarkValidateEstimateCostResponse_WithPriceOptions`, plus an `AllocsPerRun`
  test asserting 0.

## R6. Sentinel errors and error shape

- **Decision**: Three sentinels in `validation.go`:
  - `ErrPriceOptionNil`: "price_options entry is nil"
  - `ErrPriceOptionInvalidValue`: "price_options value is invalid"
  - `ErrPriceOptionsWithDryRun`: "price_options must be empty for dry-run responses"

  The messages name the index and field (FR-011). Projected cost errors are wrapped with the
  `GetProjectedCostResponse:` prefix, as cost breakdown errors are. Estimate errors use an
  `EstimateCostResponse:` prefix.
- **Rationale**: This matches the `ErrCostBreakdown*` pattern. Callers can use `errors.Is`, and the
  message pinpoints the entry.
- **Alternatives considered**: One sentinel per field. Rejected: four extra exported symbols with no
  caller need. The field name is in the message.

## R7. Option helpers and copy semantics

- **Decision**: Add `WithProjectedCostPriceOptions(options ...*pbc.PriceOption)
  GetProjectedCostResponseOption` and `WithEstimatePriceOptions(options ...*pbc.PriceOption)
  EstimateCostResponseOption`. Each deep-copies the entries with `proto.CloneOf`. No arguments clear
  the field. The options do not validate.
- **Rationale**: `WithProjectedCostBreakdown` copies its map, so caller changes made after the call
  do not leak into the response. `proto.CloneOf` is already used in `batch.go` and the supplemental
  mocks. The options do not validate, matching the existing helpers.
  `NewEstimateCostResponse` options are named `WithEstimate*` / unprefixed, so the estimate option
  is `WithEstimatePriceOptions`.
- **Alternatives considered**: A `NewPriceOption(...)` constructor with functional options.
  Rejected: seven plain fields read clearly as a struct literal.

## R8. Capability and discovery

- **Decision**: No capability enum, no legacy metadata key. Hosts detect support by a non-empty
  list.
- **Rationale**: This is the same decision as `cost_breakdown` (053) and `metadata` (#427). The
  field rides on existing RPCs, and an older plugin simply sends an empty list.

## R9. TypeScript

- **Decision**: Regenerate with `make generate` (it also regenerates TS bindings, per 052). Add a
  `priceOptions` round-trip to `test/mocks/handlers.ts` and `test/integration.test.ts`, as 053 did
  for `costBreakdown`. No TS builder or validator.
- **Rationale**: Principle XIII requires the TS bindings and an integration test. The TS client has
  no response builder or validator for these responses today (spec Assumptions).

## R10. Mock plugin and documentation

- **Decision**:
  - Mock: add `ProjectedCostPriceOptions []*pbc.PriceOption` and
    `EstimateCostPriceOptions []*pbc.PriceOption` to `MockPlugin`. Each response gets a deep copy.
    Dry-run projected responses never carry options. Entries are returned as configured (no
    scaling, no validation).
  - Docs: a "Price Option Helpers (price_options)" section in `sdk/go/pluginsdk/README.md` (with a
    TOC entry and a rules table), a mock field paragraph in `sdk/go/testing/README.md`, and a
    bullet in `PLUGIN_DEVELOPER_GUIDE.md` next to the `cost_breakdown` bullet.
- **Rationale**: The breakdown mock scales weights because of its sum rule. Price options have no
  derived values, so returning them as configured is simpler, and it lets host tests feed invalid
  options on purpose. The documentation set mirrors 053 (Principle VII).
- **Alternatives considered**: Mock computes `savings_fraction` from its own unit price. Rejected:
  it hides the plugin's responsibility and adds math to test code.
