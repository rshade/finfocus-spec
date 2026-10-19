# Feature Specification: On-Demand Equivalent for Spot Results

**Feature Branch**: `main` (working tree)

**Created**: 2026-10-18

**Status**: Implemented

**Input**: Issue #646, Spot Market Advisor spec-first change.

## User Scenarios & Testing

### User Story 1 - Show spot savings from plugin prices (Priority: P1)

A plugin returning a DYNAMIC spot price can also return the same resource's on-demand monthly
price. A host can then calculate savings from two plugin-supplied monthly prices, without pricing
the resource or making a second request.

**Independent Test**: Request a DYNAMIC projected cost or estimate and observe a present on-demand
equivalent in the same currency and monthly basis.

**Acceptance Scenarios**:

1. **Given** a DYNAMIC response with a known on-demand price, **when** the host reads either RPC,
   **then** it receives both monthly prices and can distinguish the on-demand value from absence.
2. **Given** a DYNAMIC response with no known on-demand price, **when** the host reads it,
   **then** the equivalent is absent and spot savings are unavailable.
3. **Given** a real zero on-demand price, **when** the host reads it, **then** presence remains true.

### User Story 2 - Reject contradictory equivalents (Priority: P1)

Plugin authors receive a hard validation error for a present equivalent on a non-DYNAMIC result or
for a non-finite or negative equivalent. A spot price may exceed the on-demand equivalent.

**Independent Test**: Validate projected and estimate responses across each category and boundary
value, and check that plugin and conformance validators return the same sentinel.

**Acceptance Scenarios**:

1. **Given** STANDARD, COMMITTED, or UNSPECIFIED pricing with a present equivalent, **when**
   validated, **then** validation returns a category error even for a present zero.
2. **Given** NaN, infinity, or a negative equivalent, **when** validated, **then** validation
   returns a value error before any category error.
3. **Given** DYNAMIC pricing with the equivalent below the spot cost, **when** validated,
   **then** validation succeeds.

### User Story 3 - Preserve older plugins and hosts (Priority: P1)

Legacy plugins leave the equivalent unset, and older hosts ignore the new wire field. Batch
responses carry the existing projected or estimate messages without a new batch field.

**Independent Test**: Round-trip absent and present-zero responses over gRPC and nested batch
serialization; run wire breaking analysis against the previous schema.

**Acceptance Scenarios**:

1. **Given** an older response without the field, **when** decoded by new bindings, **then** the
   equivalent is absent and validation behaves as before.
2. **Given** a new response, **when** read by an older decoder, **then** the unknown field is
   ignored and all prior fields remain readable.

### Edge Cases

- Present `0.0` differs from absent and is valid only for DYNAMIC pricing.
- A negative spot saving is possible; no ordering rule rejects it.
- Dry-run responses have no pricing category, so a present equivalent fails the category rule.
- The equivalent and a STANDARD `price_options` entry are advisory and are not cross-checked.

## Requirements

### Functional Requirements

- **FR-001**: Both projected and estimate responses MUST carry an optional monthly equivalent.
- **FR-002**: The value MUST use the response currency and the primary cost's monthly basis of
  730 hours for hourly-billed resources.
- **FR-003**: Absence MUST mean unknown; present zero MUST remain distinguishable.
- **FR-004**: A present value MUST be finite, non-negative, and attached only to DYNAMIC pricing.
- **FR-005**: DYNAMIC responses without the value MUST remain valid, including legacy responses.
- **FR-006**: The equivalent MUST be advisory and never summed into the primary cost; neither the
  spot cost nor `price_options` MUST be used to reject a valid equivalent.
- **FR-007**: Plugin self-validation and conformance MUST report the same error categories.
- **FR-008**: The SDK MUST offer non-panicking response options and an advisory consistency check.
- **FR-009**: Mock responses MUST emit a configured equivalent only for DYNAMIC pricing.

### Key Entities

- **On-demand equivalent**: An optional monthly price for the same resource at STANDARD pricing.
- **Selected spot price**: The primary DYNAMIC monthly cost already returned by the plugin.

## Success Criteria

- **SC-001**: Both response types preserve absent versus present zero across a real RPC.
- **SC-002**: All listed invalid categories and values are rejected consistently in both
  validators; valid and legacy cases pass.
- **SC-003**: Proto lint and breaking checks pass; no request or batch field is added.
- **SC-004**: Existing validation happy paths remain at zero allocations.

## Out of Scope

Request-side pricing-category selection, savings math, a unit-price equivalent, changes to
`price_options` or `spot_interruption_risk_score`, and downstream plugin or core implementation.
