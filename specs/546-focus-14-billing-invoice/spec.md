# Feature Specification: FOCUS 1.4 Billing Period and Invoice Detail

**Feature Branch**: `546-focus-14-billing-invoice`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "feat(proto): FOCUS 1.4 Billing Period and Invoice Detail datasets. Closes #543"

## User Scenarios & Testing

### User Story 1 - Describe a billing period (Priority: P1)

A plugin author with provider billing data can record one billing cycle: when it starts and
ends, whether it is open or closed, who issued the invoices, and when the record was created
and last updated.

**Why this priority**: Cost and invoice rows join to this cycle. Without it, invoice
reconciliation has no period to attach to.

**Independent Test**: Build one open period and confirm it is accepted. Confirm a period with
a missing issuer, an unspecified status, an end that is not after the start, or a last-updated
time before creation is rejected.

**Acceptance Scenarios**:

1. **Given** a period with all six columns set and an end after the start, **When** it is
   validated, **Then** it is accepted.
2. **Given** a period whose status is unset, **When** it is validated, **Then** validation
   fails and names the status.
3. **Given** a period whose last update is earlier than its creation, **When** it is
   validated, **Then** validation fails.

---

### User Story 2 - Describe an invoice line (Priority: P1)

A plugin author can record one invoice line with the columns FOCUS 1.4 requires, including a
charge category that is not a refund, a billing currency, and optional settlement-currency
columns that stay in step with each other.

**Why this priority**: This is the invoice row hosts reconcile against cost rows.

**Independent Test**: Build a usage line in USD and confirm it is accepted. Confirm a refund
category, an unspecified category, and a settlement currency without its cost are rejected.
Confirm a billed cost of zero is distinct from a missing settlement cost.

**Acceptance Scenarios**:

1. **Given** an invoice line with every column that does not allow nulls, **When** it is
   validated, **Then** it is accepted.
2. **Given** a line whose charge category is refund or unset, **When** it is validated,
   **Then** validation fails.
3. **Given** a settlement currency without a settlement cost, or a settlement cost without a
   currency, **When** it is validated, **Then** validation fails.
4. **Given** a settlement cost of zero, **When** the line is read back, **Then** the zero is
   present and is not treated as an omitted cost.

---

### User Story 3 - Read the columns from another SDK (Priority: P2)

A TypeScript client and a JSON-LD reader can see the same billing period and invoice line a
Go plugin built, including an empty grain versus a grain that is set.

**Why this priority**: The constitution requires every new record to reach each SDK in the
same change.

**Independent Test**: Build a line in TypeScript and confirm a later change to the builder
does not alter the record already returned. Confirm JSON-LD writes a zero settlement cost and
omits an unset one.

**Acceptance Scenarios**:

1. **Given** a built invoice line, **When** the builder is changed afterward, **Then** the
   built line is unchanged.
2. **Given** a line with a zero settlement cost, **When** it is serialized, **Then** the cost
   key is present and equal to zero.

---

### Edge Cases

- A billing period end equal to its start is rejected. The end is an exclusive bound.
- Invoice description, grain, issue date, and payment due date may be empty. Empty means null.
  A description is recommended, and an empty one is still accepted.
- Purchase order number may be empty. A single line cannot know whether the issuer accepts
  purchase orders.
- Grain keys are the FOCUS names (ContractId, RegionId, ResourceId, ResourceType, ServiceName,
  SkuId, SkuMeter, SkuPriceId, SubAccountId) or a custom key that starts with `x_`.
- Custom monetary columns use an `x_` key and a decimal value. An empty set is valid.
- Billed cost may be zero or negative. It must be a finite number.
- When a settlement cost is non-zero and a payment-currency invoice line id is set, that id
  matches this line's invoice detail id.
- A billing currency or settlement currency that is set must be an ISO 4217 code.

## Requirements

### Functional Requirements

- **FR-001**: A billing period MUST carry start, exclusive end, status, invoice issuer, created
  time, and last-updated time. None of those may be null.
- **FR-002**: Billing period status MUST be Open or Closed. Last updated MUST be greater than
  or equal to created. End MUST be after start.
- **FR-003**: An invoice line MUST carry the eighteen FOCUS columns that are always present.
  Description, grain, issue date, and payment due date MAY be null. The other fourteen MUST
  NOT be null.
- **FR-004**: Charge category MUST be Usage, Purchase, Tax, Credit, or Adjustment. Refund and
  an unset category MUST be rejected.
- **FR-005**: Invoice issue status MUST be Open, Issued, or Voided.
- **FR-006**: Settlement currency and settlement cost MUST be both set or both omitted. Zero
  MUST remain distinct from omitted. A set currency MUST be ISO 4217.
- **FR-007**: Grain keys and custom monetary columns MUST follow the FOCUS key rules above.
- **FR-008**: The same records MUST be available to Go plugins, TypeScript clients, conformance
  tests, JSON-LD output, and the column reference.
- **FR-009**: Simple field setters MUST allocate nothing. Known-value checks for the two new
  statuses MUST allocate nothing.

### Key Entities

- **Billing period**: One invoice issuer's billing cycle and whether that cycle is open or closed.
- **Invoice detail**: One line on an invoice, in the billing currency, with optional settlement
  currency and optional lineage to a different aggregation level.

## Success Criteria

- **SC-001**: A plugin author can build a billing period and an invoice line that each satisfy
  every column that does not allow nulls.
- **SC-002**: A refund category and a half-set settlement currency are rejected, and the
  rejection names the column.
- **SC-003**: Adding the records does not remove or renumber any existing field.

## Assumptions

- Clarify was not run as a separate question round. The published FOCUS 1.4 column table is
  the source for which columns are mandatory. That table has eighteen always-present invoice
  columns (four of them nullable) and four conditional columns. Issue 543 said seventeen and
  five. The issue's field list still contains every published column, so nothing is dropped.
- Payment currency and payment-currency billed cost share one condition (the issuer can bill
  and settle in different currencies), so a single record requires both or neither. The
  payment-currency invoice detail id has a separate condition and may be absent even when the
  pair is set.
- Purchase order number is not required on a single record. The dataset requires the column
  only when the issuer accepts purchase orders, which a per-record check cannot know.
- Empty description is accepted. FOCUS says it should not be null, not that it must not be.
- Invoice detail grain is a string map, matching the Tags decision. An empty map means null.
- Payment-currency billed cost uses a presence bit so zero is distinct from absent.
- No delivery RPC is added. How plugins serve these records is a later phase.
- Cross-row rules are out of scope: invoice sums, rounding tolerance, joins to cost rows,
  uniqueness of a line id inside an invoice, and one-way status changes.
- The spec directory is 546 because 545 is the highest existing specs prefix.

## Out of Scope

- An RPC or capability that returns these records.
- Dataset-level attributes such as correction handling and delivery handling.
- Checking that a reference invoice id points at a real earlier invoice.
- Checking that an invoice line's created time is not after a closed billing period's last update.
