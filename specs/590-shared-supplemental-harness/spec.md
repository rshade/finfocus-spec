# Feature Specification: Shared Conformance Harness and Duplicate-Key Check

**Feature Branch**: `590-shared-supplemental-harness`

**Created**: 2026-10-01

**Status**: Draft

**Input**: User description: "Supplemental dataset harnesses and duplicate-key checks are
copy-pasted in sdk/go/testing (issue #561). Extract one generic duplicate-key helper backing all
three checks, and one shared in-memory harness helper that the exported harness types are built
on. Exported names stay source-compatible or are deprecated; zero-allocation guarantees hold; no
validation rule or error message changes. Closes #561"

## Context

The `sdk/go/testing` package gives plugin authors and SDK maintainers in-memory harnesses
that serve a plugin over an in-process connection, plus response validators for the paged
supplemental datasets. Two kinds of code are repeated there:

- **Six harnesses** (cost source, allocator, scorer, usage source, contract commitments,
  invoice dataset) each repeat the same start, stop, and client logic. Only the service they
  register and the client they return differ. Adding a dataset means adding another copy.
- **Three duplicate-key checks** (contract commitment id, billing period identity, invoice
  detail id) each repeat the same strategy: compare pairs on small pages (no allocation),
  and use a lookup table on large pages. They differ only in the key and the error text.

This is a maintainability change. Observable behavior stays the same.

## Clarifications

`/speckit-clarify` was not required. The issue's acceptance criteria fix the scope. The two
open choices (all six harnesses or only two, and one merged harness type or distinct types)
are resolved under Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - One duplicate-key strategy for every paged dataset (Priority: P1)

A maintainer who adds a paged supplemental dataset writes one key function and one error
message. They do not copy a third (or fourth) pairwise-then-lookup loop.

**Why this priority**: The duplicate checks sit on the zero-allocation validation path, so
an inconsistent copy is a correctness and performance risk. The issue names this as a
half-finished extraction, because the paging and response validation helpers next to it are
already shared.

**Independent Test**: Exercise the shared helper directly with pages below, at, and above
the pairwise limit. Confirm that each existing response validator still reports the same
first duplicate with the same message, and still allocates nothing on valid pages of up to
64 records.

**Acceptance Scenarios**:

1. **Given** a valid page of 64 or fewer records, **When** any of the three response
   validators runs, **Then** it returns no error and performs zero allocations.
2. **Given** a page that holds a duplicate key, **When** the page is at or below the
   pairwise limit and also when it is above it, **Then** the validator reports the same
   later index, the same earliest matching index, and the same message text as before
   this change.
3. **Given** a page with several duplicate keys, **When** it is validated, **Then** the
   reported pair is the duplicate whose later index is lowest, matched against that key's
   earliest occurrence, on both strategies.
4. **Given** billing periods that share an issuer and start but differ in end, **When**
   they are validated, **Then** they are still reported as duplicates (a composite key
   works with the shared helper).

---

### User Story 2 - One harness implementation behind every exported harness (Priority: P2)

A plugin author keeps using `NewContractCommitmentHarness`, `NewInvoiceDatasetHarness`,
`NewTestHarness`, and the others exactly as before. A maintainer who adds a service writes
only a registration function and a client constructor, and gets start, stop, and client
behavior for free.

**Why this priority**: This removes the most duplicated code, but it is test-support code
with no runtime path. Source compatibility for existing callers is the main constraint.

**Independent Test**: Run the existing harness-based tests and conformance suites unchanged.
Add a focused test that starts a harness, makes a call, and stops it twice, which checks the
shared lifecycle once.

**Acceptance Scenarios**:

1. **Given** existing caller code that uses any of the six exported harness types and their
   constructors and `Start`, `Stop`, and `Client` methods, **When** it is compiled against
   the new package, **Then** it compiles without changes and behaves the same way.
2. **Given** a started harness, **When** `Stop` is called more than once, **Then** the
   second call does not panic.
3. **Given** a harness whose `Start` has not been called, **When** `Stop` is called, **Then**
   the server stops and nothing panics.
4. **Given** an invoice dataset harness, **When** a client calls `GetContractCommitments`,
   **Then** it still receives `Unimplemented` (and the reverse for the commitment harness
   and invoice RPCs).

---

### Edge Cases

- A page of exactly 64 records uses the pairwise path, and 65 uses the lookup path. Both
  must report identical results for the same duplicate layout.
- Empty and single-record pages report no duplicate.
- A nil record in a page is rejected by the per-record check before the duplicate check
  runs. The duplicate helper is never the first line of defense against nil.
- Records whose key is the empty string are compared like any other key (unchanged rule).
- The cost-source harness also exposes an internal second dial path used by spec
  validation. It must keep working against the same in-memory listener.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The package MUST provide one internal duplicate-finding helper, parameterized
  by record type and key type. It returns the later index and the first index of the
  earliest duplicate, or reports that there is none.
- **FR-002**: The helper MUST compare pairs for pages at or below the existing pairwise
  limit (64), and use a lookup table above it. Both paths MUST report the same pair.
- **FR-003**: The contract commitment, billing period, and invoice detail duplicate checks
  MUST all delegate to the helper and keep their current error text, error wrapping, and
  error code: `ErrInvalidContractCommitmentsResponse`, `ErrInvalidBillingPeriodsResponse`,
  and `ErrInvalidInvoiceDetailsResponse` respectively, each with `InvalidArgument`.
- **FR-004**: Response validators MUST remain allocation-free on valid pages of up to 64
  records. The existing allocation-free tests MUST pass unchanged.
- **FR-005**: The package MUST provide one internal in-memory harness implementation that
  owns the listener, server, client connection, and start, stop, and client lifecycle. It
  is parameterized by the service registration and the client type.
- **FR-006**: All six exported harness types MUST be built on the shared implementation, and
  their names, constructors, and the `Start`, `Stop`, and `Client` method signatures MUST
  stay source-compatible. No exported identifier is removed or renamed, so no deprecation
  notices are needed.
- **FR-007**: The deprecated-dial lint suppression for the in-memory connection MUST appear
  in one place for the shared harness, rather than once per harness. The cost-source
  harness's second dial path (used by spec validation) MUST reuse that same dial.
- **FR-008**: Validation rules, error messages, conformance scenarios, and the `pluginsdk`
  re-exports MUST NOT change.

### Key Entities

- **Duplicate key**: The value that must be unique within one page. It is the contract
  commitment id, the (invoice issuer name, billing period start seconds, nanos) composite,
  or the invoice detail id.
- **Harness**: An in-memory server plus client pair for one gRPC service. It is defined by
  how it registers the service and how it builds a client.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The three duplicate checks share one strategy implementation: there is
  exactly one pairwise loop and one lookup-table loop for duplicate keys in the package.
- **SC-002**: The six harnesses share one lifecycle implementation: there is exactly one
  in-memory dial (and one lint suppression for it) used by harness `Start`.
- **SC-003**: 100% of existing tests in `sdk/go/testing` and `sdk/go/pluginsdk` pass without
  modification to their assertions, including the allocation-free tests (0 allocations
  per run).
- **SC-004**: Adding a new paged dataset's duplicate check needs only a key function and an
  error constructor (under 10 lines). Adding a new harness needs only a registration and a
  client constructor.
- **SC-005**: Linting reports zero findings, matching the repository baseline.

## Assumptions

- All six harnesses are in scope, not only the two supplemental ones. The issue's proposed
  direction names a generic harness that the cost-source harness also uses, and stopping
  at two would leave four copies. The allocator and scorer harness files are also touched by
  in-flight claims (#579, #580). This change is limited to the harness block of those files,
  so a rebase conflict, if any, is local.
- The exported harness types keep their distinct names, rather than collapsing into one
  `SupplementalHarness`. That needs no deprecation and keeps the narrow server interfaces
  (`ContractCommitmentServer`, `InvoiceDatasetServer`) as each constructor's parameter.
- The pairwise limit stays 64 and stays defined where it is today.
- No proto, TypeScript, or documentation behavior changes, so SDK parity work is not needed.
  The package README is updated only if it describes harness internals.
