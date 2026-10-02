# Feature Specification: Alternative Retail Price Options

**Feature Branch**: `557-price-options`

**Created**: 2026-10-01

**Status**: Complete

**Input**: GitHub issue 588, "feat(proto): add a repeated PriceOption for alternative retail
prices". The plugin work it unblocks is `rshade/finfocus-plugin-azure-public` issue 45.

## Summary

`GetProjectedCost` and `EstimateCost` each return one selected price. A plugin that also knows the
on-demand, savings-plan, and reservation rates for the same resource has nowhere to report them.
`metadata` holds plugin hints as strings, and a consumer must ignore keys it does not recognize.
`cost_breakdown` must sum to the one monthly cost, and a second retail price is not part of that
cost. This feature adds an advisory, repeated list of alternative prices to both responses. The
list never changes, and is never summed into, the selected price.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin reports alternative prices on a projected cost (Priority: P1)

A pricing plugin author (for example, the Azure public pricing plugin) returns the selected
Consumption price as today. In the same `GetProjectedCost` response, it also lists the 1-year and
3-year reservation prices and the savings-plan price. Each alternative has its own category,
model name, term, unit price, monthly cost, upfront charge, and savings fraction.

**Why this priority**: This is the gap that blocks plugin issue 45. Projected cost is the main
response that hosts show to users when comparing purchase options.

**Independent Test**: Build a projected cost response with a selected price and two alternatives.
Check that the SDK validator accepts it and that `cost_per_month` still equals the selected price
alone.

**Acceptance Scenarios**:

1. **Given** a projected cost response with `cost_per_month` set to the selected Consumption price
   and two alternative prices, **When** the plugin validates the response, **Then** validation
   passes and `cost_per_month` is unchanged.
2. **Given** a reservation whose published price is a 3-year term total, **When** the plugin
   reports it, **Then** the term total is in `upfront_cost`, `unit_price` is the hourly
   equivalent, and `term` is the provider's term string, such as `3 Years`.
3. **Given** a projected cost response that has both a `cost_breakdown` and alternative prices,
   **When** the plugin validates it, **Then** only the breakdown is checked against
   `cost_per_month`. The alternative prices are not included in that sum.

---

### User Story 2 - Plugin reports alternative prices on a cost estimate (Priority: P2)

A plugin answering `EstimateCost` for a proposed resource configuration returns the selected
`cost_monthly` and also lists alternative prices in the same shape as projected cost.

**Why this priority**: The same plugin serves both RPCs, so both should be able to report
alternatives. Projected cost comes first because it is the response the plugin issue names first.

**Independent Test**: Build an estimate response with two alternatives and confirm that the
estimate validator accepts it and that `cost_monthly` is unchanged.

**Acceptance Scenarios**:

1. **Given** an estimate response with `cost_monthly` and two alternative prices, **When** the
   plugin validates it, **Then** validation passes and `cost_monthly` equals the selected price
   alone.

---

### User Story 3 - Existing plugins and hosts keep working (Priority: P1)

A plugin built against an earlier SDK version never sets the new list. A host built against an
earlier version receives responses that contain it.

**Why this priority**: Backward compatibility is a constitution requirement (Principle VI). A
regression here would break every deployed plugin.

**Independent Test**: Validate responses that omit the list. Decode a response that has the list
with message definitions that do not have the field.

**Acceptance Scenarios**:

1. **Given** a projected cost or estimate response with no alternative prices, **When** it is
   validated, **Then** it passes exactly as it did before this feature.
2. **Given** a host that predates this feature, **When** it receives a response with alternative
   prices, **Then** it decodes the response, skips the unknown field, and reads the selected price
   unchanged.

---

### User Story 4 - Validator rejects malformed alternative prices (Priority: P2)

A plugin author makes a mistake and puts NaN, infinity, or a negative price into an alternative.
The SDK validator reports which entry and which value is wrong.

**Why this priority**: Hosts sort and display these prices. A NaN or negative price would corrupt
comparisons in every consumer.

**Independent Test**: Validate responses with one bad value in an alternative and check the error
for each case.

**Acceptance Scenarios**:

1. **Given** an alternative whose `unit_price`, `monthly_cost`, `upfront_cost`, or
   `savings_fraction` is NaN or infinite, **When** the response is validated, **Then** validation
   fails with an error that names the field and the entry's position in the list.
2. **Given** an alternative whose `unit_price`, `monthly_cost`, or `upfront_cost` is negative,
   **When** the response is validated, **Then** validation fails.
3. **Given** an alternative with a negative `savings_fraction` (it costs more than the selected
   price), **When** the response is validated, **Then** validation passes.

---

### Edge Cases

- **Selected price is zero** (free tier): the savings ratio has no defined value. The plugin
  reports `savings_fraction` as 0, and the validator does not reject it.
- **Alternative matches the selected price**: the list may include an entry for the selected
  price, with `savings_fraction` 0. Hosts must not treat that entry as a second charge.
- **Option with no term** (Consumption, spot): `term` is empty.
- **No upfront charge**: `upfront_cost` is 0.
- **Unrecognized category value** from a newer plugin: accepted, the same as the parent
  `pricing_category`, for forward compatibility.
- **Dry-run response**: a projected cost response that carries a dry-run result has no cost data,
  so it must not carry alternative prices. This matches the existing `cost_breakdown` rule.
- **`savings_fraction` does not match the formula**: the validator does not recompute it. The
  plugin owns the calculation (Principle III).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The protocol MUST define a `PriceOption` message with these fields: pricing category
  (1), model name (2), term (3), unit price (4), monthly cost (5), upfront cost (6), and savings
  fraction (7). The category reuses `FocusPricingCategory`.
- **FR-002**: `GetProjectedCostResponse` MUST gain `repeated PriceOption price_options = 16`.
- **FR-003**: `EstimateCostResponse` MUST gain `repeated PriceOption price_options = 6`.
- **FR-004**: Field 17 on `GetProjectedCostResponse` and field 7 on `EstimateCostResponse` MUST be
  held for a later per-region price list. They are held by comment, not by a `reserved`
  statement, so that the later feature can use them without a breaking-change exception.
- **FR-005**: The field comments MUST state that the list is advisory, that an empty list means
  the plugin did not supply alternatives, and that the list is never summed into, and never
  replaces, `cost_per_month` or `cost_monthly`.
- **FR-006**: The `PriceOption` comments MUST define each field:
  - `model`: the pricing model name, such as `Consumption`, `Reservation`, or `SavingsPlan`.
  - `term`: empty when the model has no term, otherwise the provider's term string, such as
    `1 Year`.
  - `unit_price`: the price per unit on the same basis as the parent response.
  - `monthly_cost`: that option's monthly cost, never added to the parent monthly cost.
  - `upfront_cost`: the amount charged up front for the term, 0 when there is none. A term-total
    reservation price goes here, with `unit_price` set to the hourly equivalent.
  - `savings_fraction`: `(primary - option) / primary`, unrounded, negative when the option costs
    more, and 0 when the primary is 0.
- **FR-007**: The existing comment on `EstimateCostResponse` that describes a possible future cost
  breakdown MUST be updated so that the breakdown does not take field 6 or field 7.
- **FR-008**: `ValidateGetProjectedCostResponse` and `ValidateEstimateCostResponse` MUST accept a
  response that omits `price_options`, with no change in behavior from before this feature.
- **FR-009**: Both validators MUST reject any `PriceOption` whose `unit_price`, `monthly_cost`,
  `upfront_cost`, or `savings_fraction` is NaN or infinite.
- **FR-010**: Both validators MUST reject any `PriceOption` whose `unit_price`, `monthly_cost`, or
  `upfront_cost` is negative. They MUST accept a negative `savings_fraction`.
- **FR-011**: Validation errors for a `PriceOption` MUST name the field and the zero-based index of
  the entry in the list.
- **FR-012**: Neither validator may require the list's prices to sum to, or relate to,
  `cost_per_month` or `cost_monthly`. A response with alternatives and an unchanged primary cost
  MUST pass. `cost_breakdown` sum checking MUST ignore the list.
- **FR-013**: `ValidateGetProjectedCostResponse` MUST reject `price_options` on a response that
  carries a dry-run result, as it already does for `cost_breakdown`.
- **FR-014**: Validation of a response with no alternatives MUST stay at zero allocations per call,
  and validation of a response with valid alternatives MUST also be zero allocations per call.
- **FR-015**: The Go SDK MUST provide a way for plugins to attach alternative prices when they
  build a projected cost response, following the existing response option pattern. The mock
  plugin MUST be able to return configured alternative prices, so conformance and integration
  tests can exercise them.
- **FR-016**: Regenerated Go and TypeScript bindings MUST expose the repeated field and the
  `PriceOption` type (Principle XIII).
- **FR-017**: The change MUST pass the repository's breaking-change check against `main`.

### Key Entities

- **PriceOption**: One alternative retail price for the resource being priced. Attributes:
  category, model name, term, unit price, monthly cost, upfront cost, and savings fraction. It
  stands alone and does not add to the parent's cost.
- **Selected (primary) price**: The existing single price on each response (`unit_price` and
  `cost_per_month` on projected cost, `cost_monthly` on estimate). It is the basis for each
  option's `savings_fraction`, and it is unchanged by this feature.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin can report at least four prices for one resource in one response: one
  selected and three alternatives (on-demand, 1-year reservation, savings plan). The selected
  monthly cost stays the same.
- **SC-002**: 100% of existing validator tests for both responses pass unchanged.
- **SC-003**: A response with two valid alternatives passes both validators, and the primary cost
  equals the selected price alone.
- **SC-004**: Every invalid numeric case in User Story 4 (NaN, positive infinity, negative
  infinity, and negative price on each applicable field) is rejected, and the error names the field
  and the entry.
- **SC-005**: Validation adds no memory allocations for responses with or without alternatives.
- **SC-006**: An older host can decode a response with alternatives and read the selected price
  without error.

## Assumptions

- On `EstimateCostResponse`, which has no unit price, the `savings_fraction` basis is
  `cost_monthly` compared to the option's `monthly_cost`. When unit prices share the parent's
  quantity basis, the two ratios are the same.
- Negative `unit_price`, `monthly_cost`, and `upfront_cost` are rejected to match the existing
  non-negative rule on `cost_per_month`. The issue only required NaN and infinity rejection. This
  is a stricter default for a new field, so no existing plugin can break.
- `model` and `term` are free provider strings. The SDK does not enforce a list of names.
- There is no SDK limit on list length. Plugins are expected to send a handful of options per
  resource.
- The list may include an entry for the selected price.
- `savings_fraction` is plugin-computed. The SDK checks only that it is finite (Principle III).
- No new capability enum or RPC is needed. The field is additive on existing responses, and a host
  detects support by a non-empty list.
- The TypeScript SDK has no response builder for these fields today. Generated types are enough
  for TypeScript parity.
- The per-region price list (fields 17 and 7) is a separate feature and is out of scope.
- No JSON schema change is needed. `PricingSpec` is not affected.
