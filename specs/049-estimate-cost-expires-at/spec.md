# Feature Specification: EstimateCost expires_at Cache-Hint Parity

**Feature Branch**: `049-estimate-cost-expires-at`
**Created**: 2026-03-12
**Status**: Draft
**Input**: GitHub Issue #434 — Add optional expires_at to EstimateCostResponse for cache-hint parity

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin Author Sets Cache Hint on EstimateCost (Priority: P1)

A plugin author implementing cost estimation wants to attach a caching hint to
`EstimateCostResponse` so that callers know when the estimate becomes stale. Today,
the plugin can set `expires_at` on projected and actual cost responses, but not on
estimate cost responses, forcing plugin authors to handle estimate caching out-of-band.

**Why this priority**: This is the core missing capability. Without the proto field,
no SDK helpers or caching behavior can be built.

**Independent Test**: Can be fully tested by adding the proto field, regenerating code,
and verifying that a plugin can set and a caller can read `expires_at` on an
`EstimateCostResponse` message.

**Acceptance Scenarios**:

1. **Given** a plugin returning an `EstimateCostResponse`, **When** the plugin sets
   `expires_at` to a future timestamp, **Then** the caller receives the timestamp and
   can use it for caching decisions.
2. **Given** a plugin returning an `EstimateCostResponse`, **When** the plugin does not
   set `expires_at`, **Then** the field is nil/unset and the caller treats it as "no
   caching guidance."
3. **Given** an existing plugin compiled against an older spec version, **When** it
   returns an `EstimateCostResponse` without `expires_at`, **Then** newer callers see
   the field as nil with no deserialization errors (wire compatibility).

---

### User Story 2 - Caller Applies Uniform Caching Policy (Priority: P2)

A host/caller consuming cost data from multiple RPCs (actual, projected, estimate)
wants to apply a single L2 caching policy across all cost endpoints using `expires_at`
as the cache-hint signal. Today, the caller must special-case `EstimateCost` because
the response lacks `expires_at`.

**Why this priority**: Enables the uniform caching pattern that motivates the feature,
but depends on the proto field from P1 being available.

**Independent Test**: Can be tested by verifying that SDK helper functions
(`EstimateCostExpiresAt`, `IsEstimateCostExpired`) return correct results for set,
unset, and past timestamps.

**Acceptance Scenarios**:

1. **Given** an `EstimateCostResponse` with `expires_at` set to 1 hour from now,
   **When** the caller invokes `EstimateCostExpiresAt()`, **Then** it returns the
   timestamp and `true`.
2. **Given** an `EstimateCostResponse` with `expires_at` set to 1 hour ago, **When**
   the caller invokes `IsEstimateCostExpired(now)`, **Then** it returns `true`.
3. **Given** an `EstimateCostResponse` with `expires_at` unset, **When** the caller
   invokes `EstimateCostExpiresAt()`, **Then** it returns the zero time and `false`.
4. **Given** a nil `EstimateCostResponse`, **When** the caller invokes
   `IsEstimateCostExpired(now)`, **Then** it returns `false` (safe nil handling).

---

### User Story 3 - Plugin Author Uses Functional Option to Set Expiration (Priority: P3)

A plugin author wants a convenient, type-safe way to set `expires_at` on
`EstimateCostResponse` using the same functional option pattern available for actual
and projected cost responses.

**Why this priority**: Developer ergonomics improvement; the proto field and reader
helpers (P1, P2) are functional without this, but the option pattern provides SDK
consistency.

**Independent Test**: Can be tested by applying the functional option to a response
and verifying the field is set correctly, including the zero-time nil behavior.

**Acceptance Scenarios**:

1. **Given** an `EstimateCostResponse`, **When** the plugin applies
   `WithEstimateCostExpiresAt(time.Now().Add(1*time.Hour))`, **Then** `expires_at` is
   set to the provided timestamp.
2. **Given** an `EstimateCostResponse`, **When** the plugin applies
   `WithEstimateCostExpiresAt(time.Time{})` (zero time), **Then** `expires_at` is nil
   (no caching guidance).

---

### Edge Cases

- What happens when `expires_at` is set to a timestamp in the past? The field is valid;
  callers interpret it as "immediately stale" per the documented semantics.
- What happens when `expires_at` is set to the Unix epoch (zero timestamp)? This is a
  valid protobuf timestamp representing 1970-01-01. Callers should treat it as expired.
  The zero `time.Time{}` Go value (which is year 0001) clears the field to nil.
- How does `BatchGetCost` propagate `expires_at` from `EstimateCostResponse`? The
  `CostData.estimate` oneof already wraps `EstimateCostResponse`, so the new field is
  automatically available in batch responses.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `EstimateCostResponse` message MUST include an optional `expires_at`
  field of type `google.protobuf.Timestamp` at the next available field number (5).
- **FR-002**: The SDK MUST provide an `EstimateCostExpiresAt(resp) (time.Time, bool)`
  helper that returns the expiration time and whether the field is set.
- **FR-003**: The SDK MUST provide an `IsEstimateCostExpired(resp, now) bool` helper
  that returns true when `expires_at` is before the given reference time.
- **FR-004**: The SDK MUST provide a functional option
  `WithEstimateCostExpiresAt(time.Time)` that integrates with the existing
  `NewEstimateCostResponse()` builder following the functional option pattern in
  `expires_at.go` and `helpers.go`.
- **FR-005**: All new helpers MUST handle nil response and nil `expires_at` gracefully
  (return false/zero-time without panicking).
- **FR-006**: The `expires_at` field semantics MUST be consistent with the existing
  `expires_at` fields on `GetProjectedCostResponse` and `ActualCostResult`:
  - nil/unset means no caching guidance
  - Past timestamp means immediately stale
  - Callers MAY enforce stricter local TTL policies
- **FR-007**: The proto field comment MUST include usage guidance for estimate-specific
  caching scenarios (e.g., fixed pricing = far future, volatile/spot pricing = short
  TTL or unset).

### Key Entities

- **EstimateCostResponse**: Existing protobuf message gaining a new optional
  `expires_at` field. Represents the estimated monthly cost for a resource configuration.
- **expires_at**: Advisory cache-hint timestamp. When set, indicates the response should
  be considered fresh until this time. Nil means no caching guidance is provided.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All three cost response types (`ActualCostResult`,
  `GetProjectedCostResponse`, `EstimateCostResponse`) expose `expires_at` with
  identical semantics — callers can use a single caching code path for all cost RPCs.
- **SC-002**: Existing plugins compiled against the previous spec version continue to
  work without modification (wire compatibility verified by integration tests).
- **SC-003**: SDK helper function coverage is symmetric — each cost response type has
  the same set of `ExpiresAt`, `IsExpired`, and functional option helpers.
- **SC-004**: All new SDK helpers handle nil inputs safely with zero panics across the
  full test suite.

## Assumptions

- The next available field number on `EstimateCostResponse` is 5, based on the current
  proto definition having fields 1-4 (currency, cost_monthly, pricing_category,
  spot_interruption_risk_score).
- The functional option pattern follows the established `ActualCostResultOption` /
  `ApplyActualCostResultOptions` pattern in `sdk/go/pluginsdk/expires_at.go`.
- No new RPC or message type is needed — this is purely an additive field on an existing
  message with corresponding SDK helpers.
- The `BatchGetCost` RPC automatically gains access to this field through the existing
  `CostData.estimate` oneof wrapper.
