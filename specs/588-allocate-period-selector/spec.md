# Feature Specification: Allocation Period and Partial-Selection Scope

**Feature Branch**: `588-allocate-period-selector`

**Created**: 2026-10-02

**Status**: Draft

**Input**: User description: "AllocateRequest carries no period and gives no signal for partial
selections. Add the period the costs cover to the request and echo it in the response. Add a
partial-selection signal so an allocator knows idle cost is not comparable. Additive only; validation,
the reference allocator, conformance, and the TypeScript client updated. Closes #579."

## Clarifications

### Session 2026-10-02

- Q: Which partial-selection contract should the request carry? → A: A workload selector with the same
  meaning as the usage request's selector. Allocation invariants stay unchanged. A non-empty selector
  means idle and cluster rows include capacity that unselected workloads use. The allocator adds a
  warning, and hosts should omit or label those rows.
- Q: How is the period represented? → A: Start and end timestamps with the usage request's rules (both
  or neither; start not after end), echoed on the response.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host labels allocated costs with their period (Priority: P1)

A host gathers historical usage for a window, prices the cluster, and asks an allocator to divide the
cost. It passes the window on the allocation request. The allocator echoes the window on its response,
so the host can label every row with the period its cost covers.

**Why this priority**: Today the response says its costs are "for the normalized period" without saying
which one, so allocation rows are unlabeled (issue 579).

**Independent Test**: Send an allocation request with a window to the reference allocator. The
response carries the same window and passes response validation.

**Acceptance Scenarios**:

1. **Given** a request with a start and end, **When** the reference allocator answers, **Then** the
   response carries the identical start and end.
2. **Given** a request with no window (a run-rate allocation), **When** the allocator answers, **Then**
   the response carries no window and validates as before.
3. **Given** a request with only a start, only an end, or a start after the end, **When** it is
   validated, **Then** it is rejected as an invalid argument naming the problem.

---

### User Story 2 - Allocator knows the selection is partial (Priority: P1)

A host narrows workloads, for example to one namespace, but node capacity stays cluster-wide. It passes
the same selector it used for usage on the allocation request. The allocator still satisfies every
invariant (conservation and one idle row per priced node), and it adds a warning that idle and cluster
rows include capacity used by workloads outside the selection. The host omits or labels those rows.

**Why this priority**: Without a signal, idle cost in a namespace-filtered view is silently inflated
by other namespaces' usage (issue 579 and the consumer's design rule).

**Independent Test**: Send a request with a namespace selector to the reference allocator. The response
passes every invariant check and carries the partial-selection warning. The same request without a
selector carries no such warning.

**Acceptance Scenarios**:

1. **Given** a request with a non-empty selector, **When** the reference allocator answers, **Then**
   conservation holds, every priced node has exactly one idle row, and the warnings include the
   partial-selection notice.
2. **Given** a request with no selector, **When** the allocator answers, **Then** no partial-selection
   warning appears.

---

### User Story 3 - Hosts and older allocators interoperate (Priority: P1)

A new host sends a window to an allocator built before this change. That allocator ignores the new
fields and returns no window. The host's response validation must still accept the response.

**Why this priority**: A host that rejected every older allocator response would break deployed
plugins on the host's first SDK upgrade, even though the wire change is additive.

**Independent Test**: Validate a response with no window against a request that has one. It passes.
Validate a response whose window differs from the request's. It fails.

**Acceptance Scenarios**:

1. **Given** a request with a window and a response without one, **When** the response is validated,
   **Then** it passes.
2. **Given** a response whose window differs from the request's, **When** it is validated, **Then** it
   fails, naming the mismatched field.
3. **Given** a response with a window when the request had none, **When** it is validated, **Then** it
   fails.

---

### User Story 4 - Allocator authors prove the contract (Priority: P2)

An allocator author runs the conformance suite. It now checks that the allocator echoes the window and
keeps every invariant under a selector.

**Why this priority**: Conformance is where new allocators learn to echo the window, which the
host-side validator cannot demand without breaking older allocators.

**Independent Test**: Run the suite against the reference allocator (passes) and against an allocator
that drops the window (fails the echo scenario).

**Acceptance Scenarios**:

1. **Given** the reference allocator, **When** conformance runs, **Then** the period and selector
   scenarios pass.
2. **Given** an allocator that does not echo the window, **When** conformance runs, **Then** the period
   scenario fails.

---

### User Story 5 - TypeScript hosts send the new fields (Priority: P2)

A TypeScript host sets the window and selector on an allocation request and reads the echoed window.

**Why this priority**: Constitution principle XIII requires SDK parity.

**Independent Test**: The TypeScript allocator client sends a request with a window and selector and
reads the echoed window from a mocked response.

**Acceptance Scenarios**:

1. **Given** a TypeScript allocation request with a window and selector, **When** it is sent, **Then**
   both reach the server and the echoed window is readable.

---

### Edge Cases

- **Start equals end**: valid (an empty window), matching the usage request rule.
- **Selector with only the reserved "namespace" key**: a partial selection.
- **Empty selector map**: not a partial selection; no warning.
- **Window with run-rate usage**: allowed. Allocation uses ratios, so the window is a label and does
  not change any cost.
- **Window on a request with no usage and no priced resources**: echoed. The response still has no rows.
- **Mismatched window in a response**: a hard validation error, because the host would mislabel costs.
- **Selector values**: not validated. They are opaque to the allocator, as they are to the usage source.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The allocation request MUST carry an optional window as start and end timestamps (fields
  5 and 6), and an optional workload selector (field 7).
- **FR-002**: The allocation response MUST carry an optional echoed window as start and end (fields 5
  and 6).
- **FR-003**: Request validation MUST reject a window with only one bound, or with the start after the
  end, as an invalid argument that names the problem. A request with neither bound MUST validate as
  before.
- **FR-004**: The selector MUST have the same meaning as the usage request's selector: all entries must
  match, and the key "namespace" is reserved for the namespace. The SDK does not validate its values.
- **FR-005**: A non-empty selector MUST NOT relax any allocation invariant. Conservation and exactly one
  idle row per priced node still hold.
- **FR-006**: Documentation MUST state that, with a non-empty selector, idle and cluster rows include
  capacity used by unselected workloads, and that hosts SHOULD omit or label those rows.
- **FR-007**: An allocator SHOULD add a warning when the selector is non-empty. The reference allocator
  MUST do so.
- **FR-008**: An allocator MUST echo the request's window exactly, equal in seconds and nanoseconds.
  Response validation MUST reject an echoed window that differs from the request's, and MUST reject a
  window that the request did not have. It MUST accept a response with no window, so allocators built
  before this change stay valid.
- **FR-009**: The reference allocator MUST echo the window.
- **FR-010**: The allocator conformance suite MUST require the window echo and MUST check that every
  invariant holds under a selector.
- **FR-011**: The window MUST NOT change any allocated cost.
- **FR-012**: The generated TypeScript types MUST expose the new fields, and a TypeScript client test
  MUST send and read them.
- **FR-013**: The allocator documentation and the SDK READMEs MUST describe the window, the echo rule,
  the selector, and the host rule for partial selections.

### Key Entities

- **Allocation request**: Gains a window (start, end) and a workload selector.
- **Allocation response**: Gains the echoed window.
- **Partial selection**: A request whose selector is non-empty. Node capacity still covers all
  workloads, so idle and cluster costs are not comparable to an unfiltered view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of reference allocator responses to windowed requests carry the identical window.
- **SC-002**: Every invalid window kind (only start, only end, inverted) is rejected, and each rejection
  names the problem.
- **SC-003**: Under a selector, 100% of reference allocator responses pass conservation and the
  idle-row rule and carry the partial-selection warning.
- **SC-004**: A response without a window passes validation against a windowed request, which proves
  that older allocators stay valid.
- **SC-005**: All existing allocator tests and conformance scenarios pass unchanged, and the protocol
  compatibility check against the main branch reports no breaking change.

## Assumptions

- The window follows the usage request's rule exactly, so a host can pass that request's window
  straight through.
- The partial-selection warning text is not part of the contract. Only its presence is, and the
  reference allocator's wording is documented.
- The reference allocator does not change any computation for a selector. It only warns.
- Hosts, not allocators, decide whether to omit or label idle and cluster rows, following the
  consumer's design rule.
- Allocation lineage (issue 578) is out of scope.
