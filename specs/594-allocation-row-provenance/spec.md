# Feature Specification: Allocation Row Provenance

**Feature Branch**: `594-allocation-row-provenance`

**Created**: 2026-10-01

**Status**: Draft

**Input**: User description: "#578 - AllocationRow carries no lineage or method provenance for FOCUS 1.3
allocation columns"

## Overview

An allocator divides priced infrastructure across workloads and returns rows of allocated cost. A host that
turns those rows into FOCUS cost rows must fill the FOCUS 1.3 split-cost columns: the allocation method, a
description of the method, and the resource the cost came from. Today an allocation row carries none of
these, so the host has to guess the method and the source resource.

This feature lets an allocator state, per row, which method produced the row, how the split worked, and
which priced resource the cost came from. The three values are optional, so every existing allocator and
host keeps working unchanged. The conservation rules for allocated cost are untouched.

**Scope boundary**: the issue also floats an optional lineage chain as an alternative to a source resource
id. This feature does not add it. The source resource id covers the three FOCUS 1.3 columns the issue names;
the lineage chain is deferred and its wire slot is held by comment. The request side (period, partial
selection signal) belongs to a separate issue (#579) and is not touched here.

## Clarifications

### Session 2026-10-01

- Q: Which of the issue's two options (source resource id, or a lineage chain)? -> A: Source resource id
  only. It mirrors FOCUS 1.3 `AllocatedResourceId` exactly, adds no recursive message to a row that can be
  repeated thousands of times, and the lineage chain can still be added later without breaking anything.
- Q: Must method details require a method id? -> A: No. FOCUS 1.3 conditions only the method id on the
  resource id; details are recommended and stay free-form. This matches the existing FOCUS record rule.
- Q: What does the source resource id refer to? -> A: An opaque string that the allocator chooses. By
  convention it is the `resource.id` of the priced resource whose cost the row draws on (a node id for a
  workload or idle row). The SDK does not cross-check it against the request, because an allocator may
  report a coarser or finer source than the host priced.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host Maps Allocation Rows to FOCUS Columns (Priority: P1)

A host receives allocation rows and builds FOCUS cost rows. Each row names its allocation method, an
optional description of the split, and the resource the cost came from. The host copies the values into the
matching FOCUS columns without inferring anything.

**Why this priority**: This is the whole point of the issue. Without it hosts guess.

**Independent Test**: Serve the reference allocator over each transport, send a fixed cluster, and verify
every returned row carries the method id and source resource id, and that values survive the round trip.

**Acceptance Scenarios**:

1. **Given** an allocator that fills provenance, **When** the host calls Allocate, **Then** each row carries
   the method id, optional details, and source resource id exactly as the allocator set them.
2. **Given** an allocator that sets none of the provenance values, **When** the host calls Allocate,
   **Then** rows are valid and identical in meaning to rows from before this feature.

### User Story 2 - Allocator Author Gets Consistency Checks (Priority: P1)

A plugin author runs the response validator and the allocator conformance suite. A row that names a method
without naming a source resource is rejected with an error that names the row and the field, matching the
FOCUS 1.3 rule. Rows with no provenance, or with a resource id alone, pass.

**Why this priority**: Without a check, inconsistent rows reach FOCUS output.

**Independent Test**: Validate responses with a method-only row, a resource-only row, a full row, and an
empty row; only the first is rejected.

**Acceptance Scenarios**:

1. **Given** a row with a method id and no source resource id, **When** the response is validated,
   **Then** validation fails with an invalid-argument error naming the row index and the field.
2. **Given** a row with a source resource id and no method id, **When** validated, **Then** it passes.
3. **Given** a conformance run over an allocator that emits an inconsistent row, **When** the suite runs,
   **Then** a specific scenario fails and names the problem.

### User Story 3 - Reference Allocator and Clients Show the Pattern (Priority: P2)

The reference allocator fills the provenance on every row it emits, and the TypeScript client exposes the
new fields, so plugin authors have a working example in both languages.

**Why this priority**: Parity and a worked example; the contract is usable without it.

**Independent Test**: The reference allocator's output validates, and the TypeScript client round-trips a
row with provenance.

**Acceptance Scenarios**:

1. **Given** the reference allocator, **When** it allocates a cluster, **Then** every row carries a method
   id and a source resource id and the response passes validation and conservation.
2. **Given** the TypeScript client, **When** a row with provenance is decoded, **Then** the three values are
   available on the row.

### Edge Cases

- Method id empty, details set, resource id set or empty: allowed (details are free-form).
- Whitespace-only values are treated as set (opaque strings); the SDK does not trim.
- Idle and cluster rows may carry provenance like workload rows; the rule applies to every row equally.
- An older allocator that never sets the values, and an older host that ignores them, both keep working.
- Conservation sums, currency resolution, and idle-row rules are unchanged by the presence of provenance.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: An allocation row MUST be able to carry an optional allocation method id, optional method
  details, and an optional source resource id.
- **FR-002**: Rows that set none of the three values MUST remain valid, and the change MUST be additive
  with no compatibility break against the released contract.
- **FR-003**: Response validation MUST reject a row whose method id is set while its source resource id is
  empty, with an invalid-argument error that names the row index and the field.
- **FR-004**: Response validation MUST accept a row with a source resource id and no method id, and a row
  with details and no method id.
- **FR-005**: Provenance MUST NOT affect conservation, row total rules, or currency rules.
- **FR-006**: The allocator conformance suite MUST check the method/source rule and MUST report a
  dedicated, named failure for a violation.
- **FR-007**: The reference allocator MUST fill method id and source resource id on every row it emits.
- **FR-008**: The TypeScript client MUST expose the three values on allocation rows.
- **FR-009**: The documented field text MUST state the FOCUS 1.3 column each value maps to, the
  method-needs-source rule, and that the lineage chain is a later addition.
- **FR-010**: This change MUST NOT add or change any request-side or response-level field.

### Key Entities

- **Allocation row**: one unit of allocated cost, extended with method id, method details, source resource
  id.
- **Allocation method**: an allocator-defined identifier for how a cost was split (opaque string).
- **Source resource**: the priced resource a row's cost came from, identified by an opaque string.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A host maps 100% of rows from the reference allocator to the three FOCUS columns with no
  inference.
- **SC-002**: Every response from an allocator that predates this feature still validates unchanged.
- **SC-003**: 100% of rows with a method id and no source resource id are rejected by the validator and
  flagged by the conformance suite.
- **SC-004**: The compatibility check against the released contract reports no breaking change.
- **SC-005**: Validating a valid response does not add per-row memory allocations beyond today's baseline.

## Assumptions

- Provenance strings are opaque and not length-limited by the SDK, consistent with the FOCUS record fields.
- Method details are free-form text; FOCUS 1.4's JSON object form for details is not enforced, as with the
  existing FOCUS record.
- Issue #579 (period and partial-selection signal) is developed independently and touches different
  messages; any overlap in shared files is resolved at merge time.
