# Research: Caching Hint (expires_at) for Cost Results

**Feature Branch**: `045-caching-hint-expires-at`
**Date**: 2026-02-16

## Proto Field Number Availability

### Decision: Use field 8 for ActualCostResult, field 13 for GetProjectedCostResponse

**Rationale**: Sequential numbering follows proto3 conventions. `ActualCostResult` uses fields 1-7
(highest: `impact_metrics = 7`), so next available is 8. `GetProjectedCostResponse` uses fields 1-12
(highest: `confidence_level = 12`), so next available is 13. The issue proposed field 10 for both,
but the actual field structure requires different numbers.

**Alternatives considered**:

- Field 10 for both (as proposed in issue): Rejected because `ActualCostResult` only has 7 fields,
  creating a gap, and `GetProjectedCostResponse` already uses field 10 for `prediction_interval_lower`.
- Reserving fields for future use: Unnecessary complexity; proto3 supports field additions at any time.

## Proto Field Semantics (optional vs required)

### Decision: Use `google.protobuf.Timestamp` without `optional` keyword

**Rationale**: In proto3, message-type fields (like `Timestamp`) are inherently nullable. The
generated Go code produces `*timestamppb.Timestamp` regardless of the `optional` keyword. Using
plain `google.protobuf.Timestamp` is consistent with `ActualCostResult.timestamp` (field 1) and
keeps the proto definition simpler. Nil means "no caching guidance".

**Alternatives considered**:

- `optional google.protobuf.Timestamp`: Used in `DismissRecommendationRequest.expires_at` and
  `Recommendation.created_at`. However, `optional` on message types is a documentation-only marker
  with no behavioral difference. The existing `ActualCostResult.timestamp` field (also a Timestamp)
  does not use `optional`. Following the same message's existing pattern is more consistent.

## Target Message: GetProjectedCostResponse (not ProjectedCostResult)

### Decision: Add expires_at to GetProjectedCostResponse

**Rationale**: No `ProjectedCostResult` message exists in the proto. The projected cost response is
`GetProjectedCostResponse`, which is a single response (not repeated results). The `expires_at` field
applies to the entire response, unlike `ActualCostResult` where it's per-result.

**Alternatives considered**:

- Creating a new `ProjectedCostResult` wrapper: Unnecessary breaking change. The existing response
  message serves the same purpose.
- Adding to `GetProjectedCostRequest` instead: Wrong direction. Expiration is a server-side signal,
  not a client request parameter.

## SDK Helper Pattern

### Decision: Use functional options pattern for response builders, add standalone helpers

**Rationale**: The existing `ActualCostResponseOption` and `GetProjectedCostResponseOption` patterns
in `helpers.go` use functional options. New `WithExpiresAt()` options follow this established pattern.
Additionally, standalone helper functions for expiration checking (`IsExpired`, `ExpiresAt`) provide
caller-side utilities.

**Alternatives considered**:

- Builder pattern (FocusRecordBuilder style): The response-level functional options pattern is already
  established for these messages. Mixing patterns would be inconsistent.
- No SDK helpers (raw proto field access): Violates FR-006 and FR-007 which require SDK helpers for
  both setting and checking expiration.

## EstimateCostResponse Scope

### Decision: Exclude EstimateCostResponse from this feature

**Rationale**: The issue explicitly scopes to `ActualCostResult` and `ProjectedCostResult` (mapped to
`GetProjectedCostResponse`). `EstimateCost` is documented as deterministic and idempotent, making
caching hints less essential. Adding it later is a backward-compatible change if needed.

**Alternatives considered**:

- Include `EstimateCostResponse`: Would expand scope beyond the issue request. Can be a follow-up
  feature since adding a field is always backward-compatible.

## TypeScript SDK Impact

### Decision: Proto regeneration auto-exposes the field; no manual TypeScript changes needed

**Rationale**: TypeScript protobuf bindings are generated from proto definitions via buf. The new
`expires_at` field will be automatically available in generated types after `make generate`. No
manual TypeScript builder changes are required since the field is on response types, not on the
`FocusRecordBuilder`.

**Alternatives considered**:

- Adding TypeScript helper functions: Deferred. TypeScript callers can access the generated field
  directly. Helpers can be added in a follow-up if demand exists.

## Mock Plugin Configuration

### Decision: Add `ExpiresAtDuration` field to MockPlugin struct

**Rationale**: A `time.Duration` relative to "now" is simpler to configure than absolute timestamps
in test fixtures. The mock generates `expires_at = time.Now().Add(duration)` for each result. A zero
duration means "don't set expires_at" (backward compatible). This follows the existing pattern of
configurable mock fields (e.g., `BaseHourlyRate`, `Currency`).

**Alternatives considered**:

- Per-resource-type expiration map: Over-engineered for initial implementation. Can be added later.
- Fixed absolute timestamp: Brittle in tests; timestamps would need constant updating.
