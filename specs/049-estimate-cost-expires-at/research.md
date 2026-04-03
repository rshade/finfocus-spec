# Research: EstimateCost expires_at Cache-Hint Parity

**Feature Branch**: `049-estimate-cost-expires-at`
**Date**: 2026-04-02

## Proto Field Number Availability

### Decision: Use field 5 for EstimateCostResponse

**Rationale**: `EstimateCostResponse` currently uses fields 1-4 (currency = 1, cost_monthly = 2,
pricing_category = 3, spot_interruption_risk_score = 4). Field 5 is the next sequential number.
This was confirmed in the spec (FR-001) and matches the assumption in the feature specification.

**Alternatives considered**:

- Higher field number (e.g., 10) to leave room for future fields: Unnecessary. Proto3 supports
  adding fields at any time, and sequential numbering is the established convention in this
  codebase (045 used field 8 for ActualCostResult and field 13 for GetProjectedCostResponse,
  both the next available).

## Proto Field Semantics

### Decision: Use `google.protobuf.Timestamp` without `optional` keyword

**Rationale**: Consistent with the 045 implementation. In proto3, message-type fields (like
`Timestamp`) are inherently nullable, generating `*timestamppb.Timestamp` in Go. The existing
`expires_at` fields on `ActualCostResult` (field 8) and `GetProjectedCostResponse` (field 13)
do not use the `optional` keyword. Following the same pattern ensures consistency.

**Alternatives considered**:

- `optional google.protobuf.Timestamp`: While used on some other fields in the proto
  (e.g., `DismissRecommendationRequest.expires_at`), the `optional` keyword on message types is
  a documentation-only marker with no behavioral difference in proto3. The 045 precedent does
  not use it for `expires_at`.

## EstimateCost Caching Rationale

### Decision: Add expires_at despite deterministic nature of estimates

**Rationale**: While spec 045 excluded `EstimateCostResponse` because estimates are
"deterministic and idempotent," real-world usage reveals scenarios where estimate caching is
valuable:

- **Fixed pricing**: Plugins returning stable pricing can set a far-future `expires_at` to signal
  callers that re-fetching is unnecessary (e.g., fixed-rate storage at $0.023/GB/month).
- **Volatile/spot pricing**: Plugins returning spot or dynamic pricing may set a short TTL or
  leave `expires_at` unset to indicate frequent re-validation is needed.
- **Rate-limited upstream APIs**: Plugins fetching pricing data from rate-limited APIs (e.g.,
  AWS Pricing API) benefit from signaling cache validity to avoid redundant upstream calls.
- **Uniform caller code path**: Callers consuming all three cost RPCs can use a single caching
  policy when all responses expose `expires_at`, eliminating special-case logic for estimates.

**Alternatives considered**:

- Keep `EstimateCostResponse` without `expires_at`: Rejected. Forces callers to special-case
  estimate caching, breaking the uniform pattern that spec 045 established for the other two
  response types.

## SDK Helper Pattern

### Decision: Extend existing expires_at.go with symmetric helpers

**Rationale**: The existing `expires_at.go` file contains helpers for `ActualCostResult` and
`GetProjectedCostResponse`. Adding `EstimateCostResponse` helpers to the same file maintains
the single-file grouping for all expiration-related logic. The naming convention follows the
established pattern:

- Reader: `EstimateCostExpiresAt(resp) (time.Time, bool)`
- Checker: `IsEstimateCostExpired(resp, now) bool`
- Option: `WithEstimateCostExpiresAt(t time.Time) EstimateCostResponseOption`

Note: `EstimateCostResponseOption` already exists in `helpers.go` (line 1223). The new
`WithEstimateCostExpiresAt` option function uses this existing type. No new `Apply*` function
is needed because `EstimateCostResponse` uses the builder pattern (`NewEstimateCostResponse`)
rather than a separate applier (consistent with `GetProjectedCostResponse`).

**Alternatives considered**:

- Separate file for estimate expiration: Rejected. The 045 pattern groups all expiration helpers
  in one file. Splitting would fragment related logic.
- Generic `CostExpiresAt` interface: Over-engineered. The three response types have different
  proto types with no shared interface. Type-specific helpers are clearer.

## Mock Plugin Configuration

### Decision: Add `EstimateCostExpiresAtDuration` field to MockPlugin

**Rationale**: Follows the exact pattern from 045. `ExpiresAtDuration` (actual cost) and
`ProjectedCostExpiresAtDuration` (projected cost) already exist. Adding
`EstimateCostExpiresAtDuration` completes the set with identical semantics:

- Zero duration = nil (backward compatible, no `expires_at` set)
- Positive duration = future timestamp (`time.Now().Add(duration)`)
- Negative duration = past timestamp (for testing stale data)

**Alternatives considered**:

- Reuse `ExpiresAtDuration` for all three types: Confusing. Each cost type may have different
  caching characteristics in tests. Separate fields provide fine-grained control.

## TypeScript SDK Impact

### Decision: Proto regeneration auto-exposes the field; no manual TypeScript changes needed

**Rationale**: Identical to 045. TypeScript protobuf bindings are generated from proto
definitions via buf. The new `expires_at` field will be automatically available in generated
types after `make generate`. No manual TypeScript builder changes are required.

**Alternatives considered**:

- Adding TypeScript helper functions: Deferred. TypeScript callers can access the generated field
  directly. Helpers can be added in a follow-up if demand exists.

## BatchGetCost Propagation

### Decision: No additional work needed for batch responses

**Rationale**: The `BatchGetCostResponse` wraps `ResourceCostResult`, which uses a oneof
containing `EstimateCostResponse`. Since `expires_at` is added directly to
`EstimateCostResponse`, batch callers automatically gain access to the field through the
existing wrapper. No additional proto or SDK changes are needed.

**Alternatives considered**:

- Adding batch-level `expires_at` aggregation: Out of scope. Per-result expiration is the
  established pattern from 045 (where `ActualCostResult.expires_at` is per-result within
  the repeated results).
