# Feature Specification: Caching Hint (expires_at) for Cost Results

**Feature Branch**: `045-caching-hint-expires-at`
**Created**: 2026-02-16
**Status**: Draft
**Input**: GitHub Issue #380 - feat(proto): Add caching hint (expires_at) to cost results

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Plugin Signals Data Freshness Duration (Priority: P1)

A plugin developer integrates with a rate-limited billing API (e.g., CloudZero) that refreshes
cost data every 6 hours. After fetching actual cost data for a resource, the plugin sets an
`expires_at` timestamp on each cost result to indicate how long the data remains valid. This
prevents the caller from re-requesting data unnecessarily and respects the upstream API's
rate limits.

**Why this priority**: This is the core value proposition. Without the ability to signal data
freshness, callers have no way to avoid redundant API calls, leading to rate limiting and
degraded performance.

**Independent Test**: Can be fully tested by having a mock plugin return cost results with
`expires_at` set to a future timestamp, then verifying the caller can read and interpret
the field correctly.

**Acceptance Scenarios**:

1. **Given** a plugin that fetches costs from a rate-limited API, **When** it returns
   `ActualCostResult` records, **Then** each record includes an `expires_at` timestamp
   indicating when the data should be considered stale.
2. **Given** a plugin returns cost results without setting `expires_at`, **When** the caller
   reads the results, **Then** `expires_at` is the zero-value timestamp (absent/unset),
   indicating no caching guidance is available.
3. **Given** a plugin sets `expires_at` to a timestamp in the past, **When** the caller
   processes the result, **Then** the caller treats the data as immediately stale
   (equivalent to no caching hint).

---

### User Story 2 - CLI Uses Caching Hints for Local Cache Management (Priority: P2)

A CLI tool or host application maintains a local persistent cache (e.g., BoltDB). When
processing cost results from plugins, it reads the `expires_at` field to determine how
long to cache each result. If the cached entry has not expired, the CLI skips querying
the plugin for that resource on subsequent scans.

**Why this priority**: This delivers the primary performance benefit to end users by
reducing scan times and unnecessary plugin invocations. Depends on P1 being implemented
first.

**Independent Test**: Can be tested by simulating cached results with future `expires_at`
values and verifying the system skips re-fetching, then simulating expired entries and
verifying re-fetch occurs.

**Acceptance Scenarios**:

1. **Given** a cost result with `expires_at` set 6 hours in the future, **When** the CLI
   performs a scan within that 6-hour window, **Then** the CLI uses the cached result
   and does not invoke the plugin for that resource.
2. **Given** a cost result with `expires_at` that has passed, **When** the CLI performs
   a scan, **Then** the CLI queries the plugin to refresh the data.
3. **Given** a cost result with no `expires_at` set (zero-value), **When** the CLI performs
   a scan, **Then** the CLI always queries the plugin (no caching behavior).

---

### User Story 3 - Plugin Signals Pricing Cycle Boundary (Priority: P3)

A plugin providing projected cost data knows that current pricing is valid until the end
of a billing cycle (e.g., month-end). The plugin sets `expires_at` on the projected cost
response to the billing cycle boundary, allowing callers to understand when the projection
might change.

**Why this priority**: This is an advanced use case that builds on the same field. Projected
cost data changes less frequently but benefits from the same signaling mechanism.

**Independent Test**: Can be tested by having a mock plugin set `expires_at` on projected
cost responses to month-end, then verifying the caller reads the value correctly.

**Acceptance Scenarios**:

1. **Given** a plugin that knows pricing changes at month-end, **When** it returns a
   projected cost response, **Then** `expires_at` is set to the end of the current
   billing cycle.
2. **Given** a projected cost response with `expires_at` in the future, **When** a caller
   checks data validity, **Then** the caller can determine the data is still current
   without re-querying.

---

### Edge Cases

- What happens when a plugin sets `expires_at` to an extremely far future date (e.g., year 2100)?
  The caller should accept it but may apply its own maximum TTL policy.
- What happens when `expires_at` is set on a dry-run response? The field should be ignored since
  dry-run responses contain no cost data to cache.
- What happens when clock skew exists between plugin and caller? Callers should use their own
  system clock for comparison and tolerate reasonable skew (minutes, not hours).
- What happens when a paginated response has different `expires_at` values per result? Each
  result has independent freshness; the caller should respect per-result expiration.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The system MUST provide an optional timestamp field on actual cost results to
  indicate when the data expires and should be refreshed.
- **FR-002**: The system MUST provide an optional timestamp field on projected cost responses
  to indicate when the projection expires and should be refreshed.
- **FR-003**: When the expiration field is not set (zero-value/absent), the system MUST treat
  this as "no caching guidance available" and the caller MUST NOT assume any TTL.
- **FR-004**: The expiration field MUST accept any valid timestamp value, including past
  timestamps (indicating immediately stale data).
- **FR-005**: The system MUST maintain full backward compatibility. Existing plugins that do
  not set the field MUST continue to work without modification.
- **FR-006**: The system MUST provide SDK helper functions for plugins to easily set the
  expiration timestamp on cost results.
- **FR-007**: The system MUST provide SDK helper functions for callers to check whether a
  cost result has expired relative to a given reference time.
- **FR-008**: The mock plugin in the testing framework MUST support configuring expiration
  timestamps for test scenarios.
- **FR-009**: The conformance test suite MUST validate that the expiration field round-trips
  correctly through the transport layer.

### Key Entities

- **ActualCostResult**: An individual cost data point returned by GetActualCost. Gains an
  optional expiration timestamp.
- **GetProjectedCostResponse**: The response message for projected cost queries. Gains an
  optional expiration timestamp.
- **Expiration Timestamp**: A point-in-time value indicating when the associated cost data
  should be considered stale and eligible for refresh.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Plugins can set an expiration hint on 100% of cost result types (actual and
  projected) without additional round-trips or separate calls.
- **SC-002**: Callers can determine data freshness by reading a single field per result,
  with no additional computation or external lookups required.
- **SC-003**: Existing plugins and callers that do not use the new field continue to operate
  identically to before the change (zero breaking changes).
- **SC-004**: The testing framework supports expiration-aware test scenarios, enabling plugin
  developers to validate caching hint behavior in their conformance tests.
- **SC-005**: The feature adds no measurable overhead to existing calls when the field
  is not populated (zero-cost when unused).

## Assumptions

- The `ProjectedCostResult` message referenced in the issue does not exist in the current proto.
  The equivalent message is `GetProjectedCostResponse`. The expiration field will be added to
  `GetProjectedCostResponse` instead.
- Field numbers in proto messages: `ActualCostResult` currently uses fields 1-7, so the next
  available field number is 8. The issue proposed field 10, but the actual field number will
  be determined during implementation planning to follow proto conventions.
- The expiration timestamp uses `google.protobuf.Timestamp` which is already imported in the
  proto file and used throughout the codebase.
- Clock synchronization between plugin and caller is assumed to be reasonable (within minutes).
  The feature does not include clock skew compensation.
- The CLI/host caching implementation is out of scope for this feature. This feature only adds
  the signaling mechanism; how callers use it is their responsibility.
- TypeScript SDK parity will be maintained by exposing the new field through the existing
  generated types.
