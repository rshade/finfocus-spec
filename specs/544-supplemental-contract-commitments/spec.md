# Feature Specification: Supplemental Dataset Service (Contract Commitments)

**Feature Branch**: `544-supplemental-contract-commitments`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "Stage A of the FOCUS supplemental dataset delivery decision (issue
544, decision recorded in `.specify/assessments/focus-1-4-support/decision.md`, verdict go, option
2 staged). Add a new optional service with one operation that returns the existing FOCUS 1.3
Contract Commitment records, paged and filtered by an optional time window, discovered through a
new automatically inferred capability, with SDK validation, a reference producer, conformance,
a TypeScript client, and docs."

## Overview

FOCUS defines *supplemental datasets* that sit beside the Cost and Usage rows and join to them by
key. FOCUS 1.3 added the Contract Commitment dataset, and finfocus-spec already has a
`ContractCommitment` message and builder for it. No operation sends that message, so plugins
cannot deliver commitments to hosts and the conformance suite can only exercise the builder. The
FOCUS 1.4 work (issue 542 adds 17 commitment columns, issue 543 adds Billing Period and Invoice
Detail) would grow that untested schema further.

The recorded decision chose a new optional service for supplemental datasets, built in stages.
This feature is stage A: the service with a single operation that returns contract commitments.
It reuses the proven optional-service shape of the usage source (spec 051) and allocator
(spec 052) features: an optional provider interface, registration by the SDK's serve entry point
only when implemented, an automatically inferred capability, both transports, and health
reporting.

**Relationship to constitution principle III**: the service passes through commitment records the
source already holds. It adds no commitment, discount, or amortization math.

## Clarifications

### Session 2026-09-28

No human was available; answers were derived from the decision record, issue 544, the FOCUS 1.4
research, and the 044, 051, and 052 precedents.

- Q: Do FOCUS 1.4 Correction Handling (Replacement, Delta, Ledger) and Delivery Handling
  (Overwrite, Append) belong on the response? → A: No. FOCUS attaches them to a dataset *export*
  (the file delivery), not to rows, and the FOCUS 1.3 Contract Commitment dataset has no such
  column. This operation returns the source's current view of the commitments in the window each
  time it is called, which is Replacement / Overwrite semantics: a host replaces what it holds for
  that window. A handling field can be added later without breaking anyone if a producer needs
  Delta or Ledger delivery.
- Q: What does the request filter on: a billing-period window only, or also commitment status or
  ID? → A: A time window only. FOCUS 1.3 commitments have no status column (lifecycle status
  arrives with issue 542), and an ID lookup has no named consumer. Both are additive later.
- Q: Is the time window required? → A: No. Leaving both bounds unset returns every commitment the
  source holds. Setting exactly one bound, or an end that is not after the start, is
  invalid-argument, as in `GetStats`.
- Q: Which commitments match a window? → A: Those whose period overlaps it. The period is the
  commitment period when either commitment-period bound is set, otherwise the contract period. An
  unset bound is open-ended, and a commitment with no period bounds at all always matches. Periods
  and windows are half-open (`[start, end)`).
- Q: Does a request with no page size and no page token return everything, as `GetActualCost`
  does for legacy hosts? → A: No. The operation is new, so there are no legacy hosts to protect.
  A page size of 0 means the default of 50, values above 1000 are clamped to 1000, and negative
  values are invalid-argument. Every response is bounded.
- Q: Does the SDK warn at startup when a commitment provider relies on inferred capabilities, as
  it does for usage-only and allocation-only plugins? → A: No. Commitment data comes from billing
  sources that are normally also cost sources, so the inferred pricing capabilities are usually
  true. The docs tell commitment-only plugins to set capabilities explicitly.
- Q: Must the total count be exact, or may a source report 0 when the total is expensive, as
  `GetActualCost` allows? → A: Exact. A source holds its commitments as a bounded list, so the
  total is always the number of commitments matching the window across all pages, and 0 only when
  none match. Hosts and the conformance suite can then check a full walk against it.
- Q: Is the reference producer added to the existing mock plugin? → A: No. It is a separate mock
  type. Adding the operation to the existing mock plugin would change the capabilities it infers
  and the services it serves for every existing user of that mock.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host Retrieves Contract Commitments from a Plugin (Priority: P1)

A FinFocus host connected to a billing plugin asks for the contract commitments that were active
during a time window (or all of them), pages through the results, and receives FOCUS Contract
Commitment records that it can join to cost rows by commitment ID.

**Why this priority**: This is the delivery path issue 544 exists to create. Without it the
Contract Commitment dataset cannot leave a plugin.

**Independent Test**: Serve the reference producer through the SDK with a known set of
commitments, call the operation over each transport with and without a window and with small page
sizes, and verify the records, page tokens, and totals.

**Acceptance Scenarios**:

1. **Given** a plugin holding three valid commitments, **When** the host requests commitments with
   no window, **Then** it receives all three records and an empty next page token.
2. **Given** a plugin holding commitments for 2024 and 2025, **When** the host requests the window
   January to February 2025, **Then** only commitments whose period overlaps that window are
   returned.
3. **Given** five commitments and a page size of 2, **When** the host follows next page tokens
   until one is empty, **Then** it receives all five records exactly once, in a stable order, over
   three pages, and each response reports a total of five.
4. **Given** a plugin served by the SDK, **When** the host makes the same call over gRPC and over
   Connect, **Then** both return the same records, and failed calls return the same error code and
   message.

---

### User Story 2 - Host Discovers Which Plugins Serve Commitments (Priority: P1)

A host inspects a plugin's reported capabilities and health, and only calls the commitment
operation on plugins that advertise it.

**Why this priority**: Hosts must be able to route without trial calls, and plugins that do not
implement the feature must look exactly as they did before.

**Independent Test**: Start one plugin that implements the provider and one that does not; compare
their capabilities, legacy metadata flags, registered services, and health results.

**Acceptance Scenarios**:

1. **Given** a plugin implementing the commitment provider without explicit capabilities,
   **When** the host reads the plugin's info, **Then** the contract-commitments capability is
   present and the legacy flag `supports_contract_commitments` is `"true"`.
2. **Given** a plugin that does not implement the provider, **When** the host reads its info or
   calls the operation, **Then** the capability is absent and the call fails with unimplemented,
   exactly as before this feature.
3. **Given** a commitment plugin in Connect mode, **When** the host asks the health service about
   the supplemental dataset service, **Then** it is reported as serving; for a plugin without the
   provider it is unknown.
4. **Given** a plugin that sets its capabilities explicitly, **When** the host reads its info,
   **Then** exactly the explicit list is reported, whether or not the provider is implemented.

---

### User Story 3 - Plugin Developer Serves and Validates Commitments (Priority: P1)

A plugin author implements one method, uses the SDK's helpers to validate the request, filter by
window, and paginate, and serves the result through the standard entry point.

**Why this priority**: The first real producer (for example an Azure reservations plugin) needs a
short path from its data to a correct response.

**Independent Test**: Write a minimal plugin that implements the method with the SDK helpers,
serve it, and call it with valid and invalid requests.

**Acceptance Scenarios**:

1. **Given** a request with only a start time, an end before or equal to the start, or a negative
   page size, **When** the plugin validates it with the SDK, **Then** validation fails with
   invalid-argument naming the problem, and the plugin can return that error unchanged.
2. **Given** a commitment missing its ID, contract ID, or currency, with an unspecified category,
   an invalid currency code, a negative or non-finite amount, or an end before its start,
   **When** it is validated, **Then** validation fails with invalid-argument naming the field,
   using the same rules the builder applies.
3. **Given** a list of commitments and a page token the SDK did not issue, **When** the plugin
   paginates, **Then** it gets an invalid-argument error rather than a silent first page.

---

### User Story 4 - Plugin Developer Proves a Commitment Source Is Correct (Priority: P2)

A plugin author runs a single conformance entry point against their implementation and learns
whether it validates input, pages correctly, and emits valid, window-consistent records.

**Why this priority**: Conformance turns the dataset from "schema-only" into tested protocol, which
is the problem the decision set out to solve.

**Independent Test**: Run the conformance suite against the reference producer (passes) and against
deliberately broken sources (each fails the scenario that targets its defect).

**Acceptance Scenarios**:

1. **Given** the reference producer, **When** the conformance suite runs, **Then** every scenario
   passes.
2. **Given** a source that ignores the window, returns an invalid record, returns duplicates across
   pages, ignores the page size, or accepts a malformed request, **When** the suite runs, **Then**
   the scenario targeting that defect fails with a message naming it.

---

### User Story 5 - TypeScript Consumers Retrieve Commitments (Priority: P3)

A TypeScript host or tool calls the same operation through a high-level client and can iterate all
pages without handling tokens itself.

**Why this priority**: The constitution requires Go and TypeScript parity for new services, but no
TypeScript consumer is named yet.

**Independent Test**: Call the client against a mocked Connect endpoint and verify a single page, a
multi-page iteration, and error-code propagation.

**Acceptance Scenarios**:

1. **Given** a mocked endpoint returning two pages, **When** the consumer iterates, **Then** it
   receives every record from both pages in order.
2. **Given** an endpoint returning invalid-argument, **When** the consumer calls it, **Then** the
   error carries that code and message.

### Edge Cases

- A commitment with no period bounds at all matches every window.
- A commitment with only a start bound is open-ended into the future; only an end bound, open into
  the past.
- A window that ends exactly when a commitment starts does not match it (half-open ranges).
- A page token past the end of the data returns an empty page with no next token.
- A page size above 1000 is served as 1000, not rejected.
- An empty data set returns an empty list, an empty next token, and a total of 0.
- Two records with the same commitment ID in one response are invalid.
- A timestamp outside the valid protobuf range in the request is invalid-argument.
- A reference producer configured with an invalid or duplicate commitment refuses to start rather
  than serving data that fails the SDK's own validators.

## Requirements *(mandatory)*

### Functional Requirements

#### Protocol

- **FR-001**: The protocol MUST define a new optional service for FOCUS supplemental datasets in a
  new proto file, with one operation, `GetContractCommitments`, that returns the existing
  `ContractCommitment` message. The existing FOCUS proto file MUST NOT change.
- **FR-002**: The request MUST carry an optional start and end timestamp, a page size, and a page
  token. The response MUST carry the commitments, a next page token, and a total count equal to
  the number of commitments matching the window across all pages.
- **FR-003**: The protocol MUST add capability value 16, contract commitments, to the capability
  enum. All changes MUST be additive and pass breaking-change detection.
- **FR-004**: The proto comments MUST document window matching (FR-010), page size bounds
  (FR-011), opaque tokens, stable ordering across pages, snapshot semantics, and error codes.

#### SDK serving and discovery

- **FR-005**: The Go SDK MUST provide an optional provider interface with one method matching the
  operation.
- **FR-006**: When a plugin implements the provider, the serve entry point MUST register the
  service over gRPC and over Connect and report it as serving in the Connect health checker. When
  it does not, nothing about the plugin's registration, health, capabilities, or metadata changes.
- **FR-007**: Connect responses MUST preserve the gRPC status code and message of provider errors.
- **FR-008**: Capability inference MUST add the contract-commitments capability when the provider
  is implemented, and legacy metadata MUST map it to `supports_contract_commitments`. Explicit
  capabilities and plugin-supplied info MUST continue to override inference.

#### Validation and helpers

- **FR-009**: The SDK MUST validate a single commitment with the rules the builder applies
  (identity, contract ID, currency presence and ISO 4217 validity, SPEND or USAGE category, period
  ordering, non-negative amounts), extended to reject non-finite amounts. The builder MUST use the
  same validator, so the two cannot drift.
- **FR-010**: The SDK MUST validate requests: both window bounds unset or both set, valid
  timestamps, end strictly after start, and page size not negative. The SDK MUST also expose the
  window-matching rule from the Clarifications as a helper.
- **FR-011**: The SDK MUST provide pagination over a commitment list: a page size of 0 means 50,
  above 1000 means 1000, tokens are opaque offsets compatible with the existing page-token
  helpers, a malformed token is an error, and the total is the list length.
- **FR-012**: The SDK MUST validate a response against its request: every record non-nil and valid
  (FR-009), no duplicate commitment IDs, no more records than the effective page size, every record
  matching the request window, and a non-negative total that is not less than the page length.
  The total MUST be exact (FR-002); the conformance suite checks it against a full page walk.
- **FR-013**: Every validation failure MUST be a plain error (no `rpc error:` prefix) that wraps a
  documented sentinel and carries invalid-argument, so providers can return it unchanged over both
  transports.
- **FR-014**: The commitment and request validators and the window helper MUST NOT allocate on
  valid input. The response validator MUST NOT allocate for pages of up to 64 records; above that
  it MAY use one map for the duplicate check.
- **FR-015**: Because the testing package cannot import the plugin SDK, the rules MUST live in the
  testing package, and the plugin SDK MUST expose wrappers that delegate to them.

#### Reference producer and conformance

- **FR-016**: The testing package MUST provide a reference producer configured with a list of
  commitments. It MUST reject an invalid or duplicate commitment at construction, validate each
  request, filter by window, paginate per FR-011, and never emit a response that fails FR-012.
- **FR-017**: The testing package MUST provide an in-memory harness and a single conformance entry
  point that runs, as subtests: a full walk of all pages (valid responses, no duplicates, total
  consistent), stable ordering under page size 1, window filtering, one-bound window rejection,
  inverted window rejection, negative page size rejection, and malformed page token rejection.

#### TypeScript

- **FR-018**: The TypeScript SDK MUST ship regenerated bindings and a high-level client with a
  single-page call and an iterator that follows page tokens, guarded against endless empty pages.

#### Documentation

- **FR-019**: Plugin SDK, testing, and TypeScript docs and project guidance MUST describe the
  service, provider, capability, helpers, conformance, and client with exact symbol names.

### Key Entities

- **Supplemental Dataset Service**: The optional plugin-facing service for FOCUS supplemental
  datasets. Stage A has one operation; stage B (issue 543) may add invoice operations.
- **Get Contract Commitments Request**: Optional time window plus page size and page token.
- **Get Contract Commitments Response**: A page of Contract Commitment records, the next page token,
  and the total count.
- **Contract Commitment**: The existing FOCUS 1.3 record (identity, category, type, commitment and
  contract periods, cost, quantity, unit, billing currency). Issue 542 will add columns that travel
  over this operation unchanged.
- **Contract Commitments Capability**: Capability value 16, legacy flag
  `supports_contract_commitments`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin author serves commitments by implementing exactly one method and using the
  standard entry point, with no registration code.
- **SC-002**: 100% of commitment calls to an SDK-served plugin behave identically over both
  transports in the parity tests, including error code and message.
- **SC-003**: A plugin that does not implement the provider reports the same capabilities,
  metadata, services, and health as before; the existing test suite and breaking-change detection
  pass unchanged.
- **SC-004**: The reference producer passes 100% of conformance scenarios, and each deliberately
  broken source fails at least the scenario that targets its defect.
- **SC-005**: Every validation rule has at least one failing and one passing test case.
- **SC-006**: The request and commitment validators, the window helper, and the response validator
  for pages of up to 64 records run with zero allocations per call on valid input, as measured by
  benchmarks.
- **SC-007**: Go and TypeScript expose the operation in the same release.

## Assumptions

- No producer or consumer exists yet (the decision rates evidence as weak). Stage A is bounded to
  one operation over an existing message so it costs little if adoption is slow.
- Hosts keep the same window and page size across the pages of one walk. Tokens are only valid for
  the request that produced them; the SDK's offset tokens do not encode the window.
- Sources return commitments in a stable order across calls so offset pagination is consistent.
- The default and maximum page sizes (50 and 1000) are the existing SDK pagination constants.
- The operation's time window uses the same timestamp type as the cost operations.

## Dependencies

- Existing `ContractCommitment` message and builder (FOCUS 1.3).
- Page-token helpers from spec 044.
- Optional-service serving, capability inference, legacy capability mapping, and Connect error
  conversion from specs 051 and 052.
- Out of scope: invoice operations (stage B, issue 543), the 17 new commitment columns (issue 542),
  any change to `GetActualCost`, and `buf.yaml`.
