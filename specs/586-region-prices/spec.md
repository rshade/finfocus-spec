# Feature Specification: Per-Region Retail Prices on Cost Responses

**Feature Branch**: `586-region-prices`

**Created**: 2026-10-02

**Status**: Draft

**Input**: User description: "feat(proto): add a repeated RegionPrice for per-region retail prices. New
message RegionPrice {region, unit_price, monthly_cost, currency}; GetProjectedCostResponse
region_prices = 17; EstimateCostResponse region_prices = 7. Fields 16 and 6 are held for PriceOption
(issue 588). Advisory list, never summed into cost_per_month or cost_monthly. Validation rejects
NaN/Inf, empty region or currency on a present row, and must not require a sum. Closes #589.
Motivation: finfocus-plugin-azure-public plugin issue 47."

## Clarifications

`/speckit-clarify` was not required. The issue fixes the message shape, the field numbers, the
advisory (never summed) rule, the empty-list meaning, and the three validation rules. Remaining
choices are recorded under Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin returns prices for other regions (Priority: P1)

A plugin prices a resource in the requested region and already knows the same SKU's price in other
regions. It returns those prices as a list on the projected cost or cost estimate response. The
primary cost stays the requested region's price. A host can show a cross-region comparison from one
response instead of calling once per region and joining the results.

**Why this priority**: This is the gap the issue reports. The Azure public pricing plugin (plugin
issue 47) can order regions by price but has no field to return them in.

**Independent Test**: Build a projected cost response and a cost estimate response, each with two
region rows. Both pass response validation, and the primary monthly cost equals the requested region's
price alone.

**Acceptance Scenarios**:

1. **Given** a projected cost response with a monthly cost of 70.08 and two region rows of 65.70 and
   80.30, **When** it is validated, **Then** it passes and the monthly cost is still 70.08.
2. **Given** a cost estimate response with a monthly cost and two region rows, **When** it is
   validated, **Then** it passes, and the rows are never added to the monthly cost.
3. **Given** a region row whose price is zero (a free tier), **When** it is validated, **Then** it
   passes.

---

### User Story 2 - Existing responses stay valid (Priority: P1)

Plugins and hosts built before this change never set the list. Their responses must validate exactly
as before, and mixed versions must interoperate.

**Why this priority**: Protobuf backward compatibility is a constitutional requirement.

**Independent Test**: Validate responses without the list, and run the existing projected cost and
estimate tests unchanged.

**Acceptance Scenarios**:

1. **Given** a projected cost or cost estimate response with no region rows, **When** it is
   validated, **Then** the result is the same as before this change.
2. **Given** an older host, **When** a newer plugin sends region rows, **Then** the host ignores the
   unknown field and reads the primary cost as before.

---

### User Story 3 - Bad rows are rejected (Priority: P2)

A plugin that sends a malformed row learns about it from SDK validation and from the conformance
suite, with an error that names the row.

**Why this priority**: The list is only useful if hosts can trust each row's fields.

**Independent Test**: Validate responses with one bad row each and confirm each is rejected with the
row index and field named.

**Acceptance Scenarios**:

1. **Given** a row with a NaN or infinite unit price or monthly cost, **When** it is validated,
   **Then** it is rejected.
2. **Given** a row with an empty region or empty currency, **When** it is validated, **Then** it is
   rejected.
3. **Given** a row whose prices do not add up to anything related to the primary cost, **When** it
   is validated, **Then** it passes, because no sum rule applies.

---

### User Story 4 - Plugin authors and TypeScript hosts use the list (Priority: P2)

A Go plugin author sets the list through the SDK's response options, and the reference mock plugin
can return it. A TypeScript host reads the rows from a response.

**Why this priority**: Constitution principle XIII requires both SDKs and the test framework to stay
in sync with the protocol.

**Independent Test**: Build both responses through the Go SDK options, configure the mock plugin to
return rows, and decode a response carrying rows in the TypeScript client.

**Acceptance Scenarios**:

1. **Given** the Go SDK response options, **When** a plugin author passes two rows, **Then** the
   response carries copies of both rows.
2. **Given** the mock plugin configured with rows, **When** it answers a projected cost or estimate
   call, **Then** the response carries the rows and still passes validation.
3. **Given** a TypeScript client, **When** a response with two rows arrives, **Then** both rows are
   readable with their region, prices, and currency.

---

### Edge Cases

- **Empty list**: means the plugin supplied no other regions. It is not an error.
- **Zero price**: a real price (for example a free tier). It passes. A region with no price is
  omitted instead of sent as zero.
- **Negative price**: rejected, consistent with the primary monthly cost rule.
- **Currency differs from the primary currency**: allowed. Rows are not converted.
- **Currency not a valid ISO 4217 code**: rejected.
- **Requested region also listed**: allowed. The list is advisory, and its rows are never compared
  with the primary fields.
- **Same region listed twice**: not rejected by validation. Plugins should list each region once.
- **Missing row (null element)**: rejected.
- **Dry run projected cost response**: carries no region rows, consistent with the cost breakdown rule.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The protocol MUST define a region price record with a region name, a unit price, a
  monthly cost, and a currency.
- **FR-002**: The projected cost response MUST carry an optional list of region prices as field 17,
  and the cost estimate response MUST carry one as field 7. This feature MUST NOT use fields 16 and 6:
  they carry the alternative-price list `price_options` (issue 588, merged in PR 599).
- **FR-003**: The list MUST be advisory. It MUST NOT be summed into, compared with, or required to
  match the primary monthly cost of either response.
- **FR-004**: An empty list MUST mean the plugin supplied no other regions, and a response without
  the list MUST validate exactly as before.
- **FR-005**: Each row's unit price and monthly cost MUST use the same basis as the parent response's
  primary price fields.
- **FR-006**: Validation MUST reject a row whose unit price or monthly cost is NaN, infinite, or
  negative. A zero price MUST pass.
- **FR-007**: Validation MUST reject a row with an empty region, or with a currency that is empty or
  not a valid ISO 4217 code. Rows MUST NOT be converted to the parent currency.
- **FR-008**: Validation MUST reject a null row. Errors MUST name the row index and, for a present row,
  the field.
- **FR-009**: A dry run projected cost response MUST NOT carry region rows.
- **FR-010**: The field documentation MUST state that the list is advisory and is never summed into
  the primary cost, that an empty list means no other regions were supplied, and that a region with no
  price is omitted rather than sent as zero.
- **FR-011**: The Go SDK MUST offer response options that set the list on both responses. The options
  MUST copy the caller's rows, so later caller edits do not change the response. The SDK response
  validators MUST apply FR-006 to FR-009.
- **FR-012**: The conformance suite's response validators MUST apply the same row rules, so a plugin
  under test that sends bad rows fails.
- **FR-013**: The reference mock plugin MUST be able to return a configured list on both responses.
- **FR-014**: The generated TypeScript types MUST expose the list, and a TypeScript client test MUST
  read rows from a response.
- **FR-015**: Plugin-author documentation MUST describe the list, its advisory nature, and the row
  rules.
- **FR-016**: Validating a response MUST stay at zero allocations with and without rows, and the
  no-rows path MUST add no more than one length check: no extra non-inlined call.

### Key Entities

- **Region price**: One region's price for the same resource and basis as the parent response.
  Region name, unit price, monthly cost, currency.
- **Projected cost response**: Gains an optional list of region prices. The primary price fields
  stay the requested region's price.
- **Cost estimate response**: Gains the same optional list. Its primary monthly cost stays the
  requested configuration's price.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A response carrying two region rows passes validation in 100% of test cases while its
  primary monthly cost equals the requested region's price alone.
- **SC-002**: 100% of existing projected cost and estimate tests pass unchanged.
- **SC-003**: Each of the six bad-row kinds (NaN, infinity, negative, empty region, empty or invalid
  currency, null row) is rejected, and every rejection names the row index and field.
- **SC-004**: Validating a response stays at zero allocations with and without region rows. On the
  no-rows path, the only added work is one length check, confirmed by an A/B benchmark against the
  main branch whose numbers are recorded in the plan research.
- **SC-005**: The protocol compatibility check against the main branch reports no breaking change.

## Assumptions

- Negative prices are rejected, matching the existing rule for the primary monthly cost. The issue
  does not name this rule, but a negative retail price is not meaningful.
- Each row's currency must be a valid ISO 4217 code, because the issue names ISO 4217 for that field.
- Duplicate regions are not rejected. The issue does not ask for it, and rejecting them would make
  validation allocate or scale quadratically.
- The dry-run exclusion follows the existing cost breakdown rule for projected cost responses.
- The batch cost path keeps whatever responses plugins return. It needs no change, because it carries
  whole projected cost and estimate responses.
- Requests do not change. Callers cannot yet ask for specific regions. That would be a separate issue.
