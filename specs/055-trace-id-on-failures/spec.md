# Feature Specification: Trace Id on Plugin Failures

**Feature Branch**: `055-trace-id-on-failures`

**Created**: 2026-09-28

**Status**: Implemented

**Input**: User description: "Issue #193 asked for a host trace id on plugin failures.
The assessment at `.specify/assessments/trace-propagation/` stopped at
needs-clarification. The parent then chose the small slice only: a valid host
trace id is already stored for an ordinary served call, and that same id must
show up in the failure log and in the validation failure text. That behavior
shipped in pull request 538. This specification records it. It does not open a
second design."

## Overview

A person debugging a plugin can already send a trace id with a cost call. The
plugin stored a valid id for the call and then left it off the failure log and
off the validation failure the caller reads. The caller could not match the
failure to the host request without a plugin author copying the id by hand.

This slice closes that gap for an ordinary served call. A failed call writes
one failure log that names the stored trace id and the full method name. A
validation failure that did not already name an id receives that same id, and
its text gains a suffix only when an id is present. Text that has no id is
unchanged.

The web serving mode does not do this. That limit was chosen earlier and is a
non-goal here, not a defect to fix. No new header, no tracing toolkit, no
measurement of provider billing calls, and no change to the message contract.

## Clarifications

### Session 2026-09-28

- Q: Which gap is in scope, the id already stored for the call, or the host's
  full trace context on every serving mode? → A: Only the id already stored for
  an ordinary served call. It must appear in the failure log and, when the
  failure is a validation failure with no id of its own, in the text the caller
  reads.
- Q: If the host value is not a valid trace id, keep it or replace it? → A:
  Replace it. The existing acceptance rule stays. A rejected value is not
  logged and is not copied onto a validation failure.
- Q: Does the web serving mode gain the same behavior? → A: No. Leaving it out
  is a non-goal carried forward from the usage-stats and allocation work, not
  a bug for this slice.
- Q: May this slice add a legacy trace-parent header, a tracing toolkit,
  provider billing measurement, or a message-contract change? → A: No.
- Q: What if the validation failure already names a trace id? → A: Keep the
  author's id. The failure log still uses the id stored for the call.
- Q: What is the success line? → A: A host-generated trace id is present in the
  plugin failure log and in the validation failure text. That is the behavior
  that shipped.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Failed Call Names the Host Trace Id (Priority: P1)

A host sends a valid trace id with a cost call. The call fails. The person
reading the plugin log can line that failure up with the host request because
the failure log repeats the same id and names the call.

**Why this priority**: The success line is correlation. If the failure log does
not carry the host id, nothing else in the slice helps the reader.

**Independent Test**: Fail one ordinary served call that arrived with a valid
host trace id, and fail one that arrived with no header. The first log shows
the host id. The second shows a generated id, not a blank and not the missing
header.

**Acceptance Scenarios**:

1. **Given** an ordinary served call whose header is a valid trace id, **When**
   the call fails, **Then** one error-level failure log is written, its message
   is `rpc failed`, and it includes that same id and the full method name
   (the service and the method, not the short method name alone).
2. **Given** an ordinary served call with no trace id header, **When** the call
   fails and a replacement id can be generated, **Then** the failure log
   includes the replacement id and does not include an empty id.
3. **Given** an ordinary served call that succeeds, **When** the call returns,
   **Then** no `rpc failed` log is written for it.
4. **Given** a header value that is not a valid trace id, **When** the call
   fails, **Then** the failure log includes only the replacement id. The
   rejected value does not appear.

---

### User Story 2 - Validation Failure Text Carries the Id (Priority: P2)

The caller that receives a validation failure needs the same id in the failure
text, so a ticket or a client log matches the plugin log without parsing
structured fields. Failures that already name an id, and failures with no id
at all, must keep the text their readers already depend on.

**Why this priority**: The success line's second half is the validation
failure. It is useless if the log and the text disagree, and it is harmful if
every existing failure sentence changes.

**Independent Test**: Return a validation failure with an empty id, return one
that already has an id, and format one that was built outside a traced call.
Only the empty id gains the stored id. The outside failure's text has no
suffix.

**Acceptance Scenarios**:

1. **Given** a failed ordinary served call with a stored trace id and a
   validation failure whose id is empty, **When** the failure is returned,
   **Then** the failure's id becomes the stored id, and the text ends with
   a space and `trace_id=<that id>`.
2. **Given** the same call and a validation failure that already has its own
   id, **When** the failure is returned, **Then** the author's id is unchanged.
   The failure log still names the id stored for the call.
3. **Given** a validation failure with no trace id, **When** its text is read,
   **Then** the text is the field, the constraint, the actual value, and the
   expected value, with no `trace_id=` suffix.
4. **Given** a failed call whose error is not a validation failure, **When**
   the failure is returned, **Then** the failure log still names the stored
   id, and the error text gains no trace id suffix.

---

### User Story 3 - Authors Attach the Id Once (Priority: P3)

Plugin authors still write their own lines during a call. They need one way to
put the stored trace id on those lines. Putting it on twice makes two copies
and the reader cannot tell which one to trust.

**Why this priority**: The automatic failure log covers failures. Authors still
log successes and intermediate steps. A duplicated field would undo the
correlation this slice adds.

**Independent Test**: Attach the stored id to a logger that does not yet have
one, and attach nothing when the call has no id. A second attachment of the
same field is what the logging guide tells the author not to do.

**Acceptance Scenarios**:

1. **Given** a call whose stored trace id is present, **When** an author
   attaches it once to a log record, **Then** that record contains the id.
2. **Given** a call with no stored trace id, **When** an author asks to attach
   it, **Then** the record is unchanged and does not gain an empty id.
3. **Given** the plugin logging guide, **When** an author reads how to log a
   traced call, **Then** the guide says to attach the id once and not set the
   trace id field again on that same record.

---

### Edge Cases

- Several values for the trace id header: only the first is a candidate
  (FR-001). If that first value is invalid, it is replaced (FR-002). Later
  values are ignored.
- All zeros, uppercase hexadecimal, the wrong length, and any non-hexadecimal
  character are invalid. They are replaced, not kept.
- If a replacement id cannot be generated, the call continues with no trace id.
  No failure log is written to carry an id, and a validation failure does not
  gain a suffix.
- If the full method name is unavailable, the failure log still includes the
  stored trace id and uses an empty method name.
- A validation failure nested inside another failure receives the id on the
  validation failure itself. Outer text shows the suffix only when it includes
  the validation failure's own text.
- A logger that is configured to drop error-level records still returns the
  stamped validation failure. This slice does not force the line out.
- The entry point that has no plugin logger still copies the id onto an empty
  validation failure. It does not emit a failure log.
- Two ordinary served calls at the same time keep their own ids.
- Attaching the trace id to a record that already has that field writes a
  second copy. Authors attach it once.
- The web serving mode never writes this failure log and never copies an id
  onto a validation failure, including when the same plugin binary serves both
  modes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A valid host trace id on an ordinary served call MUST be stored
  for that call. Valid means 32 lowercase hexadecimal characters and not all
  zeros. When several header values arrive, only the first is a candidate.
  This slice MUST NOT change that acceptance rule.
- **FR-002**: A missing id, an empty id, or an id that is not valid under
  FR-001 MUST be replaced with a newly generated valid id when generation
  succeeds. The rejected value MUST NOT be stored, logged, or copied onto a
  validation failure.
- **FR-003**: When an id cannot be generated, the call MUST continue with no
  trace id. It MUST NOT fail because generation failed.
- **FR-004**: When an ordinary served call fails and a trace id was stored, the
  plugin MUST write one error-level failure log. That log MUST include the
  stored trace id, the full method name (service and method), and the failure.
  Its message MUST be `rpc failed`.
- **FR-005**: A successful ordinary served call MUST NOT write that failure
  log.
- **FR-006**: When the failure is a validation failure whose trace id is empty,
  the stored trace id MUST be copied onto that failure before the failure log
  is written, so the log and the failure name the same id.
- **FR-007**: When the validation failure already has a trace id, that value
  MUST be kept. The failure log MUST still use the id stored for the call,
  even when the two ids differ.
- **FR-008**: With no trace id set, validation failure text MUST remain
  `{field}: {constraint} (actual: {actual}, expected: {expected})`. With a
  trace id set, the text MUST be that sentence followed by one space and
  `trace_id=<id>`, and MUST NOT change any other part of the sentence.
- **FR-009**: An author MUST be able to attach the stored trace id to their own
  log record once per request. The plugin logging guide MUST say to attach it
  once and MUST say not to set the trace id field again on that same record.
  A request with no stored id MUST leave the author's record unchanged.
- **FR-010**: The web serving mode MUST NOT write this failure log and MUST NOT
  copy a trace id onto a validation failure. That exclusion is a non-goal.
- **FR-011**: This slice MUST NOT add another trace header, take a dependency
  on a tracing toolkit, measure provider billing calls, or change the message
  contract between host and plugin.

### Key Entities

- **Trace id**: The identifier for one call. A valid id is 32 lowercase
  hexadecimal characters and is not all zeros. The call stores either the
  host's valid id or a generated replacement. It never stores a rejected
  value.
- **Failure log**: The one error-level record written when an ordinary served
  call fails and a trace id was stored. It carries the stored trace id, the
  full method name, the failure, and the message `rpc failed`.
- **Validation failure**: The structured failure a caller already receives for
  a field constraint. It may carry a trace id. The id is empty when the
  failure was built outside a traced call. Its text gains a suffix only when
  the id is set.
- **Ordinary served call**: A plugin call on the path that already accepts the
  host trace id header. This is the only path this slice changes.
- **Web serving mode**: The other way to serve the same plugin. It is unchanged
  and is out of scope.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every failed ordinary served call that stored a trace id produces
  exactly one failure log. That log names that id and the full method name.
  A reader can match it to the host request without a manual copy.
- **SC-002**: When that failure is a validation failure with an empty id, the
  text the caller reads ends with the same id. When the author already set an
  id, the text keeps the author's id in 100% of those failures.
- **SC-003**: A validation failure with no trace id contains zero occurrences
  of `trace_id=`. Existing sentences are unchanged.
- **SC-004**: Successful ordinary served calls produce zero `rpc failed` logs
  from this behavior.
- **SC-005**: A rejected host value appears in zero failure logs and zero
  validation texts. A replacement id is what those records show when
  generation succeeds.
- **SC-006**: The set of accepted trace headers, the message contract, and
  web-serving-mode tracing are unchanged. The number of new tracing-toolkit
  dependencies added by this slice is 0.

## Assumptions

- The host header for this slice is the existing `x-finfocus-trace-id`. A
  valid value is already stored for an ordinary served call. This slice does
  not define a second header and does not re-decide the acceptance rule.
- "Ordinary served call" means that existing path. "Web serving mode" means
  the other path, which earlier usage-stats and allocation work left without
  this correlation on purpose.
- The failure log is the plugin's structured error record for that call. The
  id is the stored trace id. The method name is the service and the method,
  not the short method name alone. The message text is `rpc failed`.
- "Included in the validation failure" means the existing validation failure
  text, not a new field on the message contract. The suffix is
  a space plus `trace_id=<id>` and appears only when the id is set.
- Authors attach the id once. The logging guide is the plugin SDK README
  section that already shows that single attachment. A second set of the same
  field is a mistake the guide forbids, not a second id the slice should merge.
- The no-logger entry point still copies the id onto an empty validation
  failure and discards the failure log. The served plugin path uses the
  plugin logger, so operators see the line unless their level drops errors.
- Issue #193's broader wording (a new interceptor, a legacy trace-parent
  header, a full host trace context) is superseded. The parent chose the
  small slice, and that slice has shipped. This document does not reopen the
  assessment questions.
- Tests that shipped with the behavior are the lock for these scenarios. This
  design does not ask for a new suite or a new benchmark. The work is a
  failure-path record, not a new cost calculation.

## Out of Scope

- The web serving mode, including any interceptor or trace header on that
  path. Specs 051 and 052 left it out. This spec does not treat that as a bug.
- A header named `x-pulumicost-trace-parent`, or any new `x-pulumicost-*`
  trace key.
- A tracing-toolkit dependency, span ids, exporters, or sampling.
- Measuring or wrapping provider billing calls.
- Any change to the message contract, generated bindings, or a second language
  SDK.
- Replacing the existing rule that discards an invalid or missing host id.
- Rewriting historical specs, the observability guide's unrelated examples, or
  the assessment that stopped at needs-clarification.
