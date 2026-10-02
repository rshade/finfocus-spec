# Feature Specification: Multi-Call Scorer Identity and Per-Item Error Semantics

**Feature Branch**: `589-scorer-multi-call`

**Created**: 2026-10-02

**Status**: Draft

**Input**: User description: "ScorerInfo scalars and per-item error semantics do not fit multi-call
scorers. Make provider_request_id repeated (keep the scalar, deprecate it) and allow model to list every
model used. Document that hosts SHOULD surface the per-item error message, and that
resource_type_unsupported MUST NOT be set by scorers. Validators, mock, conformance, and TypeScript
aligned. Additive only. Closes #580."

## Clarifications

`/speckit-clarify` was not required. The issue chooses the direction for both fields and for the error
flag. The remaining choices (list order, the relationship between the primary model and the list, and
the cache key) are recorded under Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scorer reports every backend request id (Priority: P1)

A scorer fans one scoring call out to several backend calls. It reports every backend request id as a
list, so a host can attach all of them to a support ticket instead of parsing one joined string.

**Why this priority**: The JEV scorer currently joins several ids into one string, which hosts cannot
split reliably (issue 580).

**Independent Test**: Validate a response whose scorer identity lists three request ids. It passes, and
each id is readable on its own.

**Acceptance Scenarios**:

1. **Given** a scorer response listing three request ids, **When** it is validated, **Then** it passes.
2. **Given** a list that contains an empty id, **When** it is validated, **Then** it is rejected,
   naming the entry.
3. **Given** a scorer that sets only the old single id, **When** its response is validated, **Then** it
   passes as before.

---

### User Story 2 - Scorer reports every model it used (Priority: P1)

A scorer that consults several models lists all of them. The primary model stays in the existing single
field and is the first entry in the list, so hosts that read only the single field still see the main
model.

**Why this priority**: Score caches key on the model. A scorer that used several models must report all
of them, or a cached score can outlive a change to a secondary model.

**Independent Test**: Validate a response whose list starts with the primary model. It passes. A list
whose first entry differs from the primary model is rejected.

**Acceptance Scenarios**:

1. **Given** a primary model "jev-1.13.0" and a list ["jev-1.13.0", "embed-2"], **When** the response
   is validated, **Then** it passes.
2. **Given** a primary model that does not match the list's first entry, **When** it is validated,
   **Then** it is rejected.
3. **Given** a list with an empty entry, **When** it is validated, **Then** it is rejected.

---

### User Story 3 - Per-item errors mean the same thing to every host (Priority: P1)

A scorer that cannot score one recommendation returns a per-item error with a code and a message. The
"resource type unsupported" flag that the shared error record carries has no meaning for scoring, so
scorers never set it. Hosts show the message, not only the code.

**Why this priority**: A host today keeps only the code and drops the message. And an error flag with
no scoring meaning invites inconsistent use.

**Independent Test**: Validate a response whose per-item error sets the unsupported-type flag. It is
rejected. A per-item error with a code and message passes.

**Acceptance Scenarios**:

1. **Given** a per-item error with a non-OK code, a message, and the flag unset, **When** it is
   validated, **Then** it passes.
2. **Given** a per-item error with the flag set, **When** it is validated, **Then** it is rejected,
   naming the result index.

---

### User Story 4 - Plugin authors and TypeScript hosts use the fields (Priority: P2)

The reference mock scorer can report several request ids and models, the conformance suite rejects a
scorer that breaks the new rules, and a TypeScript host reads the lists.

**Why this priority**: Constitution principle XIII requires SDK and test framework parity.

**Independent Test**: Configure the mock with two ids and two models, run conformance against it and
against scorers that break each rule, and decode a response with both lists in TypeScript.

**Acceptance Scenarios**:

1. **Given** the mock configured with two request ids and two models, **When** it scores, **Then** the
   response carries both lists, the single id equals the first id, and it passes conformance.
2. **Given** a scorer that sets the unsupported-type flag or a mismatched primary model, **When**
   conformance runs, **Then** it fails.
3. **Given** a TypeScript client, **When** a response with both lists arrives, **Then** both are
   readable.

---

### Edge Cases

- **Both the single id and the list set**: allowed. Hosts read the list. Scorers should set the single
  id to the list's first entry for older hosts. Validation does not require it.
- **Only the single id set**: allowed (an older or single-call scorer).
- **List set without a primary model**: rejected, because the primary model must equal the first entry.
- **Primary model set without a list**: allowed, as before.
- **Duplicate models or ids in a list**: allowed. A scorer may call the same model twice.
- **Per-item error with an empty message**: allowed by validation. Hosts show the code.
- **Model list order**: the primary model first, then the others in the order the scorer used them.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The scorer identity MUST gain a list of backend request ids (field 5) and a list of models
  (field 6).
- **FR-002**: The single backend request id (field 4) MUST be marked deprecated. It stays readable and
  valid until at least the next MAJOR version (constitution VI). Hosts MUST read the list first and fall
  back to the single id when the list is empty.
- **FR-003**: Scorers that set the request id list SHOULD also set the single id to the list's first
  entry.
- **FR-004**: Validation MUST reject an empty entry in either list, naming the list and index.
- **FR-005**: When the model list is non-empty, the primary model MUST equal its first entry, and
  validation MUST reject a mismatch.
- **FR-006**: Scorers MUST NOT set the "resource type unsupported" flag on a per-item error, and
  validation MUST reject a response that does, naming the result index.
- **FR-007**: Documentation MUST state that hosts SHOULD surface a per-item error's message as well as
  its code.
- **FR-008**: The score cache guidance MUST add the model list to the cache key, and MUST describe the
  request id list as a log field, not a key part, as the single id is today.
- **FR-009**: The reference mock scorer MUST be able to report configured request ids and models,
  following FR-003 and FR-005.
- **FR-010**: The scorer conformance suite MUST fail a scorer that breaks FR-004, FR-005, or FR-006.
- **FR-011**: The generated TypeScript types MUST expose both lists, and a TypeScript client test MUST
  read them.
- **FR-012**: Existing responses that set neither list and never set the flag MUST validate exactly as
  before.

### Key Entities

- **Scorer identity**: Name, primary model, calibration, deprecated single request id, plus the new
  request id list and model list.
- **Per-item error**: The shared error record. For scoring, only its code and message carry meaning.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A response that lists three request ids and two models (primary first) passes validation
  in 100% of test cases.
- **SC-002**: Each rule breach (empty id, empty model, mismatched primary model, unsupported-type flag)
  is rejected by validation and by conformance, and each rejection names the field and index.
- **SC-003**: 100% of existing scoring tests and conformance scenarios pass unchanged.
- **SC-004**: The protocol compatibility check against the main branch reports no breaking change.

## Assumptions

- The primary model is the first entry of the model list. That keeps the single field meaningful to
  older hosts and gives the list a defined order.
- Requiring the single id to equal the first listed id is a SHOULD, not a validation rule. Older scorers
  that never set the list must keep passing, and scorers that set only the list are not wrong.
- The unsupported-type flag is rejected rather than ignored. It has no scoring meaning, and a scorer that
  sets it is mis-mapping an error. The scoring-specific error message that the issue mentions as an
  alternative is not added.
- The request and response fields that open PRs 602 to 604 use (request 4 and 5, response 5) are not
  touched. Only the scorer identity changes.
