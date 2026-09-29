# Feature Specification: FOCUS 1.4 Contract Commitment Columns

**Feature Branch**: `545-focus-14-contract-commitment`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "feat(proto): FOCUS 1.4 Contract Commitment columns and 1.3 ContractApplied gap.
Closes #542"

## User Scenarios & Testing

### User Story 1 - Report 1.4 commitment terms (Priority: P1)

A plugin author reporting commitment contracts can carry lifecycle, payment, and benefit
details on the Contract Commitment dataset, so FinOps tools can compare terms without
provider-specific extensions.

**Why this priority**: This is the column set issue 542 exists to add.

**Independent Test**: Build a commitment with the new columns and confirm a valid record is
accepted and a mandatory-column violation is rejected.

**Acceptance Scenarios**:

1. **Given** a spend commitment with the 1.4 columns set, **When** it is validated, **Then**
   it is accepted and a discount of 0 is distinct from an omitted discount.
2. **Given** a commitment missing a mandatory 1.4 column, **When** it is validated, **Then**
   validation fails and names that column.
3. **Given** a usage commitment with no billing currency, **When** it is validated, **Then**
   it is accepted. A spend commitment with no billing currency is rejected.

---

### User Story 2 - Emit a Contract Applied object (Priority: P1)

A plugin author can attach a cost row to one or more commitments using the FOCUS JSON object,
instead of only a bare commitment ID.

**Why this priority**: The bare ID was already non-conformant in FOCUS 1.3.

**Independent Test**: Format one element and read back ContractCommitmentID, applied cost,
quantity, and unit.

**Acceptance Scenarios**:

1. **Given** a commitment ID and an applied cost, **When** the object is formatted, **Then**
   the JSON has an Elements array with those four FOCUS keys.
2. **Given** an existing caller that still passes a bare ID, **When** the deprecated setter is
   used, **Then** the stored value is still that ID.

---

### Edge Cases

- Discount percentage is required for Discount and must be null for Availability.
- Full Period fulfillment requires a Discontinuous model.
- All Upfront requires a One-Time payment interval and an upfront percentage of 1.
- No Upfront requires an upfront percentage of 0. Partial Upfront requires a fraction strictly
  between 0 and 1.
- Pricing currency may be absent. When it is set on a spend commitment, the pricing-currency
  cost must be set. A usage commitment may leave that cost null.
- Description may be empty. Empty means null.
- Superseded chains and one-way lifecycle transitions are not checked on a single row.

## Requirements

### Functional Requirements

- **FR-001**: The Contract Commitment record MUST carry the FOCUS 1.4 columns for applicability,
  benefit category, created time, discount percentage, duration, fulfillment interval, last
  updated time, lifecycle status, model, offer category, payment interval, payment model,
  upfront percentage, invoice issuer, pricing currency, pricing-currency cost, and service
  provider.
- **FR-002**: The record MUST also carry the FOCUS 1.3 description column.
- **FR-003**: Discount percentage, upfront percentage, and pricing-currency cost MUST distinguish
  an omitted value from zero.
- **FR-004**: Applicability MUST be one JSON object. It is not a separate typed structure.
- **FR-005**: Each of the seven new enumerations MUST reject unknown numeric values, and each
  known-value check MUST allocate nothing.
- **FR-006**: A spend commitment MUST name a billing currency. A usage commitment MAY omit it.
  A currency that is set MUST be an ISO 4217 code.
- **FR-007**: A cost row MUST be able to store a Contract Applied JSON object with an Elements
  array. The previous bare-ID setter MUST keep working and MUST be marked deprecated.
- **FR-008**: The same columns MUST be available to Go plugins, TypeScript clients, conformance
  tests, JSON-LD output, and the column reference.

### Key Entities

- **Contract commitment**: One contractual term, including its 1.3 identity and its 1.4 lifecycle,
  payment, and benefit details.
- **Contract applied element**: One link from a cost row to a commitment, with the applied cost,
  quantity, and unit.

## Success Criteria

- **SC-001**: A plugin author can build a commitment that satisfies every 1.4 column that does
  not allow nulls, and can tell a zero discount from an omitted one.
- **SC-002**: A cost row can carry a Contract Applied object that a reader can join to a
  commitment ID without provider-specific parsing.
- **SC-003**: Adding the columns does not remove or renumber any existing field.

## Assumptions

- Clarify was not run as a separate question round. Two issue questions are decided here.
- Applicability stays a JSON string, matching the other FOCUS JSON columns. A typed message
  would diverge from that pattern.
- Billing currency stays required for spend commitments, because FOCUS says it must not be null
  in that case. Usage commitments may omit it, because FOCUS allows null. The previous rule
  required it for every category.
- 1.3 period, type, cost, and quantity rules stay as they are. This feature does not newly
  require those columns.
- Payment upfront percentage is required whenever the payment model is set, because that model
  column does not allow nulls and the percentage rule depends on it.
- Cross-row rules (Superseded chains, one-way status changes, invoice sums) are out of scope.
- No new RPC is added. The existing contract-commitment RPC carries the extended message.
- The spec directory is 545 because 544 is the highest existing specs prefix.

## Out of Scope

- Billing Period and Invoice Detail datasets.
- An additional RPC whose only job is to return commitments.
- Cross-record lifecycle checks.
