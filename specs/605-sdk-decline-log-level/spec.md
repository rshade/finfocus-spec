# Feature Specification: Plugin SDK Decline Log Levels

**Feature Branch**: `605-sdk-decline-log-level`

**Created**: 2026-11-03

**Status**: Implemented

**Input**: Issue #664 - "pluginsdk logs handled InvalidArgument and Unimplemented declines at ERROR"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Quiet logs for contract-allowed declines (Priority: P1)

A plugin operator runs a plugin against a project that holds resources from providers the plugin does not
serve. The plugin answers each such request with a decline the contract allows (`InvalidArgument`,
`Unimplemented`), and the host treats that answer as normal. The operator reads the plugin log and
sees no ERROR lines for those answers. Today every such call writes two ERROR lines, so a single
session produced 208 lines, most of them declines.

**Why this priority**: Operators filter on ERROR to find faults. Noise from normal answers hides
real faults and makes healthy plugins look broken. This is the whole point of the issue.

**Independent Test**: Serve a plugin whose handler returns each decline code, call the RPC, and
assert the captured log holds no ERROR entry from the SDK.

**Acceptance Scenarios**:

1. **Given** a plugin whose `Supports` handler returns `InvalidArgument`, **When** a host calls
   `Supports`, **Then** the host receives `InvalidArgument` unchanged and the SDK writes no
   ERROR line.
2. **Given** a plugin whose handler for any RPC returns `Unimplemented`, **When** a host calls that
   RPC, **Then** the host receives the same outcome as today and the SDK writes no ERROR line.
3. **Given** a plugin whose handler returns `NotFound` or `FailedPrecondition`, **When** a host
   calls the RPC, **Then** the SDK writes no ERROR line.

---

### User Story 2 - Real faults stay loud (Priority: P1)

A plugin operator still needs to learn about genuine failures. A handler that fails with
`Internal`, `Unknown`, `Unavailable`, or `DataLoss`, or with a plain error that is not a status,
keeps logging at ERROR.

**Why this priority**: Demoting too much is worse than the noise. Equal priority with Story 1.

**Independent Test**: Same harness as Story 1 with fault codes; assert exactly the ERROR
entries expected.

**Acceptance Scenarios**:

1. **Given** a handler that returns `Internal`, `Unknown`, `Unavailable`, or `DataLoss`, **When** a
   host calls the RPC, **Then** the SDK writes an ERROR line.
2. **Given** a handler that returns a plain (non-status) error, **When** a host calls the RPC,
   **Then** the SDK writes an ERROR line, as today.
3. **Given** a handler that returns a status code outside both lists (for example
   `PermissionDenied`), **When** a host calls the RPC, **Then** the SDK keeps logging at ERROR.

---

### User Story 3 - Decline still traceable at debug level (Priority: P2)

A developer debugging why a plugin declined a resource raises the log level to debug and still
finds the decline, with its trace id, operation, and error text.

**Why this priority**: Demoting must not erase the evidence.

**Independent Test**: Capture logs at debug level and assert one entry per decline with the
same fields the ERROR entry carries today.

**Acceptance Scenarios**:

1. **Given** debug logging and a declined call, **When** the call finishes, **Then** the log
   holds a decline entry with trace id, operation, and error fields.

---

### Edge Cases

- A handler wraps a status error with `fmt.Errorf("%w")`: the code is read through the wrap,
  as the status-passthrough already does.
- A handler returns a raw `ValidationError` with no status: it reads as a plain error and stays at
  ERROR, unchanged. Only errors that carry a status can be declines.
- A request with no trace id: the interceptor generates one, so declines are logged at the
  right level either way; an empty trace id after a failed generation keeps today's behavior of
  logging nothing from the interceptor.
- `Supports` handler returning `Unimplemented`: already falls back to the SDK default at debug
  level and never reaches the error line; this stays unchanged.
- A plugin built on an older SDK keeps logging at ERROR until it is rebuilt.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The SDK MUST log a handler error whose gRPC code is `InvalidArgument`,
  `Unimplemented`, `NotFound`, or `FailedPrecondition` at debug level, never at ERROR.
- **FR-002**: The SDK MUST keep logging at ERROR a handler error whose code is `Internal`,
  `Unknown`, `Unavailable`, or `DataLoss`, and a handler error that carries no status.
- **FR-003**: The SDK MUST keep logging at ERROR any other code (decision recorded in
  Assumptions).
- **FR-004**: Every handler wrapper that logs a handler error (`Supports`, `DryRun`,
  `GetRecommendations`, `GetRecommendationHistory`, `GetBudgets`, `DismissRecommendation`,
  `BatchCost`, `ResolveResourceTypes`, provider `GetPluginInfo`) MUST apply the same classification, so one
  declined call produces no ERROR line from either the wrapper or the interceptor. Their
  `Unimplemented` answers already take the debug-level default path and stay unchanged.
- **FR-005**: The classification MUST read the code through error wrapping, and MUST use a single
  shared rule so the handler and the interceptor cannot disagree.
- **FR-006**: The status and message returned to the host MUST NOT change for any code.
- **FR-007**: A demoted entry MUST keep the trace id, operation, and error fields its ERROR
  form carries.
- **FR-008**: Tests MUST cover every gRPC code in a table for the classifier, the interceptor, and each of
  the nine wrappers named in FR-004.
- **FR-009**: The release note MUST state that plugins must rebuild on the new SDK to pick up
  the change. It lives in the PR description and the commit body (release-please builds
  `CHANGELOG.md` from them) and in the SDK README logging section.
- **FR-010**: The classification MUST add no allocation (0 allocs/op) and the success path (a call
  that returns no error) MUST NOT gain any code.

### Key Entities

- **Decline code**: A gRPC code the contract allows a plugin to answer with:
  `InvalidArgument`, `Unimplemented`, `NotFound`, `FailedPrecondition`.
- **Fault code**: Any other code, plus an error without a status.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin session that sends only declines of the four decline codes yields zero ERROR
  lines from the SDK (was two per call).
- **SC-002**: Each fault code and each non-status error still yields at least one ERROR line.
- **SC-003**: Every one of the 16 non-OK gRPC codes has a table test row naming its expected level.
- **SC-004**: No wire-level difference: status code and message seen by the host are
  identical before and after.
- **SC-005**: The release note names the rebuild requirement.

## Assumptions

- **Demoted set** (recorded decision): the four codes the issue proposes. Narrower sets were
  considered; `NotFound` and `FailedPrecondition` are kept because the contract uses them for
  "no such resource" and "plugin not configured for this request" answers.
- **Level**: debug. Info would still write one line per declined resource, the noise this
  removes. Debug keeps the evidence for developers (Story 3).
- **Scope**: the issue names `Supports`, but all nine wrappers share the same duplicate-ERROR
  pattern, and the issue's `GetBudgets` example is one of them. One shared rule covers all of them.
- **Other codes stay ERROR**: `Canceled`, `DeadlineExceeded`, `PermissionDenied`,
  `Unauthenticated`, `ResourceExhausted`, `Aborted`, `AlreadyExists`, `OutOfRange` are not in
  the proposal and may signal real faults; demoting them is out of scope.
- No proto, wire, or exported-API change; the change is in log output only.
- The related symptom in the issue (code `Unknown` seen through a router) was not traced to
  the SDK and is out of scope. Core-side work is tracked in rshade/finfocus#2011.

## Clarifications

`/speckit-clarify` was not required: no `[NEEDS CLARIFICATION]` markers remain, and the one open
question in the issue (the exact demoted set) is decided under Assumptions.
