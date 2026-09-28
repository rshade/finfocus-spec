# Tasks: Trace Id on Plugin Failures

**Input**: Design documents from `/specs/055-trace-id-on-failures/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/failure-trace.md

**Tests**: The scenarios are locked by the tests that shipped in pull request 538.
This list does not ask for a new suite. Each test task is the existing test.

**Organization**: Tasks are grouped by user story. US1 and US3 both touch
`sdk/go/pluginsdk/logging.go`, so those edits are sequential even though the
stories are independently testable.

**Shipped**: Compared with `a5e6790` on main (pull request 538). Every task
below is already satisfied. No production change remains.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- `sdk/go/pluginsdk/logging.go` and `sdk/go/pluginsdk/logging_test.go`
- `sdk/go/pluginsdk/validation_error.go` and `sdk/go/pluginsdk/validation_error_test.go`
- `sdk/go/pluginsdk/sdk.go` for `serveGRPC` only
- `sdk/go/pluginsdk/README.md` and `sdk/go/CLAUDE.md`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Keep the slice inside the existing module

- [x] T001 Confirm `go.mod` stays on Go 1.27.1 and that this slice adds no
  module, including no OpenTelemetry import (FR-011, research "No new header,
  toolkit, provider spans, or protobuf")

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Guard the rules this slice must not move

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T002 [P] Leave `ValidateTraceID` in `sdk/go/pricing/observability_validate.go`
  accepting only 32 lowercase hex characters and rejecting
  `00000000000000000000000000000000`. Do not keep an invalid host value.
  First metadata value only (FR-001, FR-002, data-model "Trace id" table)
- [x] T003 [P] Leave `GenerateTraceID` in `sdk/go/pluginsdk/traceid.go` as the
  replacement id. When generation fails, the call continues with an empty
  stored id and does not fail the RPC (FR-003, data-model "Empty stored id")
- [x] T004 [P] Leave `serveConnect` in `sdk/go/pluginsdk/sdk.go` without
  `TracingUnaryServerInterceptor`. Web serving mode writes no `rpc failed`
  log and stamps no validation id (FR-010)

**Checkpoint**: Acceptance, generation, and Connect are unchanged. Story work
can start.

---

## Phase 3: User Story 1 - Failed Call Names the Host Trace Id (Priority: P1) 🎯 MVP

**Goal**: A failed ordinary served call writes one error-level log, message
`rpc failed`, with the stored trace id and the full method name. Success
writes none. A rejected header never appears.

**Independent Test**:

```bash
go test ./sdk/go/pluginsdk/ -count=1 -run 'TestTracingUnaryServerInterceptor_StampsValidationErrorAndLog|TestTracingUnaryServerInterceptor_ValidTraceIDs|TestTracingUnaryServerInterceptor_InvalidTraceIDs|TestTracingUnaryServerInterceptor_MissingMetadata'
```

### Tests for User Story 1

- [x] T005 [US1] Keep `TestTracingUnaryServerInterceptor_StampsValidationErrorAndLog`
  in `sdk/go/pluginsdk/logging_test.go`: a valid host id and `GetActualCost`
  are in the log when the handler returns a validation failure with an empty
  `TraceID` (FR-004, FR-006)
- [x] T006 [US1] Keep `TestTracingUnaryServerInterceptor_ValidTraceIDs`,
  `TestTracingUnaryServerInterceptor_InvalidTraceIDs`,
  `TestTracingUnaryServerInterceptor_MissingMetadata`, and
  `TestTracingUnaryServerInterceptor_MultipleTraceIDs` in
  `sdk/go/pluginsdk/logging_test.go` so a valid id is stored, a bad or missing
  id is replaced, and only the first header value is a candidate (FR-001, FR-002)

### Implementation for User Story 1

- [x] T007 [US1] In `sdk/go/pluginsdk/logging.go`
  `TracingUnaryServerInterceptorWithLogger`, after the handler returns, log
  only when `err != nil` and the stored id is non-empty. Fields: `FieldTraceID`
  is the stored id (not a rejected header and not a handler-set id),
  `FieldOperation` is `info.FullMethod` or empty when info is nil or the
  method is empty, plus the error, message `rpc failed`. A nil error writes
  no failure log (FR-004, FR-005, contract "Failure log")
- [x] T008 [US1] In `sdk/go/pluginsdk/sdk.go` `serveGRPC`, pass
  `TracingUnaryServerInterceptorWithLogger(server.logger)` as the first
  interceptor. `TracingUnaryServerInterceptor()` in `sdk/go/pluginsdk/logging.go`
  uses a discarded logger and still reaches the stamp (data-model "Failure log")

**Checkpoint**: User Story 1 is the merged failure log. It is testable without
User Story 2's suffix and without User Story 3's helper.

---

## Phase 4: User Story 2 - Validation Failure Text Carries the Id (Priority: P2)

**Goal**: An empty `TraceID` becomes the stored id before the log. A set
`TraceID` stays. `Error()` gains a space and `trace_id=<id>` only when the field is set.
The sentence otherwise matches the pre-change text.

**Independent Test**:

```bash
go test ./sdk/go/pluginsdk/ -count=1 -run 'TestValidationError_TraceID|TestTracingUnaryServerInterceptor_KeepsHandlerTraceID|TestTracingUnaryServerInterceptor_StampsValidationErrorAndLog'
```

### Tests for User Story 2

- [x] T009 [P] [US2] Keep `TestValidationError_TraceID` in
  `sdk/go/pluginsdk/validation_error_test.go`: an empty `TraceID` produces no
  `trace_id=`; a set id appends a space and `trace_id=<id>` after
  `{FieldName}: {Constraint} (actual: {ActualValue}, expected: {ExpectedValue})`
  (FR-008, SC-003)
- [x] T010 [P] [US2] Keep `TestTracingUnaryServerInterceptor_KeepsHandlerTraceID`
  in `sdk/go/pluginsdk/logging_test.go`: a handler-set id is unchanged
  (FR-007)

### Implementation for User Story 2

- [x] T011 [US2] In `sdk/go/pluginsdk/validation_error.go`, keep `TraceID` on
  `ValidationError`. `Error()` returns the sentence above when `TraceID` is
  empty, and that sentence plus one space plus `trace_id=` and the field when
  it is set. Do not change `Unwrap` (FR-008, contract "Validation failure text")
- [x] T012 [US2] In `sdk/go/pluginsdk/logging.go` `stampValidationTrace`, use
  `errors.As` for the first `*ValidationError`. If `TraceID == ""` and the
  stored id is non-empty, set the field. If the field is already set, leave
  it. If the error is not a validation failure, do not change its text. Return
  the handler's error value, not a wrapper. Run this before the failure log
  (FR-006, FR-007, data-model stamp steps 1 through 5)

**Checkpoint**: User Stories 1 and 2 match the success line. Author logs are
still User Story 3.

---

## Phase 5: User Story 3 - Authors Attach the Id Once (Priority: P3)

**Goal**: `WithTrace` copies the stored id onto a child logger once. A context
with no id leaves the logger unchanged. The logging guide says not to set
`FieldTraceID` again.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -count=1 -run TestWithTrace`

### Tests for User Story 3

- [x] T013 [US3] Keep `TestWithTrace` in `sdk/go/pluginsdk/logging_test.go`:
  a context id is logged as one `"trace_id"` value, and a background context
  adds no `trace_id` field (FR-009)

### Implementation for User Story 3

- [x] T014 [US3] In `sdk/go/pluginsdk/logging.go` `WithTrace`, return the same
  logger when `TraceIDFromContext` is empty. Otherwise return a child with
  `FieldTraceID` set once. Godoc says adding `FieldTraceID` again writes a
  second key (FR-009, data-model "Author log record")
- [x] T015 [P] [US3] In `sdk/go/pluginsdk/README.md`, under "Creating a Plugin
  Logger", show `WithTrace` once, say not to set `FieldTraceID` again, and
  state that a failed call logs the id and that an empty `TraceID` gains
  `trace_id=<id>`. State that Connect mode does not run the interceptor
  (FR-009, FR-010)

**Checkpoint**: All three stories match the merged code.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Boundaries that apply to every story

- [x] T016 [P] In `sdk/go/CLAUDE.md`, keep the note that a failed gRPC call logs
  `x-finfocus-trace-id` and stamps `*ValidationError`, that `Error()` gains
  `trace_id=<id>` only when the field is set, that `WithTrace` is the
  per-request helper, that Connect does not run the interceptor, and that
  `x-pulumicost-trace-parent` is not added (FR-010, FR-011)
- [x] T017 Confirm the slice does not edit `proto/`, does not add
  `x-pulumicost-trace-parent`, and does not instrument provider billing calls
  (FR-011, research "No new header, toolkit, provider spans, or protobuf")
- [x] T018 Run the command in `specs/055-trace-id-on-failures/quickstart.md`
  and confirm the named tests pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup. Blocks every user story.
- **User Stories (Phase 3+)**: Depend on Foundational. US2's stamp is called
  from the US1 interceptor, so the function lands with US1's file but the
  stamp rules are US2. US3 is independent except that it edits `logging.go`
  after US1 and US2.
- **Polish (Phase 6)**: Depends on the stories whose docs it describes.

### User Story Dependencies

- **User Story 1 (P1)**: After Foundational. No dependency on US2 or US3.
- **User Story 2 (P2)**: After Foundational. Uses the interceptor from US1
  when the stamp is wired, and is still testable through `Error()` alone
  (T009, T011) before that wiring.
- **User Story 3 (P3)**: After Foundational. Independent behavior. Same file
  as US1, so do not edit `logging.go` in parallel with T007 or T012.

### Within Each User Story

- The tests listed already exist and passed with the implementation.
- Do not wrap `ValidationError` to add the id.
- Do not log on success.

### Parallel Opportunities

- T002, T003, and T004 touch different files.
- T009 and T010 touch different test files.
- T015 and T016 touch different docs.
- T007, T012, and T014 all edit `sdk/go/pluginsdk/logging.go` and are not
  parallel.

---

## Parallel Example: User Story 2

```bash
# Different files, already present on main:
# sdk/go/pluginsdk/validation_error_test.go
# sdk/go/pluginsdk/logging_test.go
go test ./sdk/go/pluginsdk/ -count=1 -run 'TestValidationError_TraceID|TestTracingUnaryServerInterceptor_KeepsHandlerTraceID'
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 and Phase 2 are already true on main.
2. User Story 1 is the failure log. It is the correlation a reader needs.
3. Stop there only if the validation text were deferred. It was not. It
   shipped in the same change.

### Incremental Delivery

User Story 1, then the validation suffix, then `WithTrace` and the guide.
Pull request 538 delivered all three. This task list records that order. It
does not schedule a second code change.

### Parallel Team Strategy

Not used. One interceptor file holds US1 and US2, and the work is already
merged.

---

## Notes

- [P] tasks are different files.
- Every box is checked because `sdk/go/pluginsdk/logging.go`,
  `validation_error.go`, `sdk.go`, the two test files, `README.md`, and
  `sdk/go/CLAUDE.md` already match the contract.
- Comparison against that merged code found no unmet task. Do not edit those
  files again for this spec.
- No benchmark task: research rejects one. Adding it would be a new gap, not
  a missed requirement.
