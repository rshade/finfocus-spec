---
description: "Task list for opt-in per-request credentials"
---

# Tasks: Opt-In Per-Request Credentials

**Input**: Design documents from `/specs/056-per-request-credentials/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required. Write each story's tests first and see them fail before the
implementation task for that story. The user ordered test-first. Constitution principle V
applies as an SDK test gate; there is no proto edit.

**Organization**: Phases follow spec.md user-story priority. US1, US2, and US3 are all P1.
US1 is the MVP. US2 and US3 are the safety gates that ship with it. US4 and US5 are P2.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no unfinished dependency)
- **[Story]**: User story for story phases only

## Path Conventions

- SDK: `sdk/go/pluginsdk/`
- Specs: `specs/056-per-request-credentials/`
- No `proto/`, `sdk/go/proto/`, or `sdk/typescript/` edits

## Phase 1: Setup

**Purpose**: Lock the slice to the existing module before any helper exists

- [x] T001 Confirm `go.mod` gains no require, and that `proto/`, `sdk/go/proto/`, and
  `sdk/typescript/` stay untouched for specs/056-per-request-credentials/plan.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The credential set, context, and header codec every story reads

**⚠️ CRITICAL**: No user story work starts until this phase is complete

### Tests first

- [x] T002 Add failing tests in `sdk/go/pluginsdk/credentials_test.go` for
  `NewCredentials` and redaction. Cover nil and empty maps (`ErrEmptyCredentials`,
  text `per-request credentials require at least one value`). Cover duplicates after
  ASCII lowercasing, names that fail `^[a-z][a-z0-9_-]{0,63}$`, names longer than 64
  characters, values outside bytes 0x20 through 0x7E inclusive, values longer than
  4096 bytes, more than 16 entries, and value bytes summing past 16384
  (`ErrInvalidCredentials`, text `per-request credentials are not valid`). Assert
  `String` and `GoString` for a set containing `test-secret-value` do not contain that
  fixture, and that editing the input map after `NewCredentials` does not change `Get`.

- [x] T003 Add failing tests in `sdk/go/pluginsdk/credentials_test.go` for
  `WithCredentials` and `ExtractCredentials`. A nil context and a context with no set
  return the zero value and a nil error. A zero `Credentials` stores nothing. A valid
  set round-trips through `Get`. Two child contexts from one parent do not share sets.

- [x] T004 Add failing tests in `sdk/go/pluginsdk/credentials_test.go` for the header
  codec and `CredentialUnaryClientInterceptor`. One entry `token` /
  `test-secret-value` becomes `x-finfocus-credential-token`. A second value on that
  key, or a value that fails the data-model rules, returns `ErrMalformedCredentials`
  (text `per-request credentials are malformed`) and the error string does not contain
  `test-secret-value`. `x-finfocus-trace-id` is ignored. No credential header leaves
  the context absent.

### Implementation

- [x] T005 Implement `Credentials`, `NewCredentials`, `Get`, `Len`, `Names`, `String`,
  and `GoString` in `sdk/go/pluginsdk/credentials.go` with the Apache header used by
  `sdk/go/pluginsdk/usage_source.go`. Enforce data-model.md: 1 to 16 entries, name
  pattern `^[a-z][a-z0-9_-]{0,63}$` after ASCII lowercasing, value length 1 to 4096,
  bytes 0x20 through 0x7E, total value bytes at most 16384. Sentinel errors only. Copy
  the caller's map. No method returns the map.

- [x] T006 Implement `WithCredentials` and `ExtractCredentials` in
  `sdk/go/pluginsdk/credentials.go` per contracts/credentials-api.md. Private context
  key types. Nil context on extract returns zero, nil. Non-nil context is required for
  `WithCredentials`. Do not call `os.Setenv`.

- [x] T007 Implement the unexported header codec and
  `CredentialUnaryClientInterceptor` in `sdk/go/pluginsdk/credentials.go`. Prefix
  constant `CredentialMetadataPrefix` = `x-finfocus-credential-`. gRPC keys stay
  lowercase. HTTP names use `http.CanonicalHeaderKey`. Malformed input drops the bytes
  and stores `ErrMalformedCredentials` without logging.

**Checkpoint**: Context and codec tests pass. `Serve` is not wired yet.

---

## Phase 3: User Story 1 - Attach credentials to one call (Priority: P1) 🎯 MVP

**Goal**: A host attaches one set to one native call. The next call does not see it.

**Independent Test**: On one opted-in plugin process, the first native call observes
`test-secret-value` and the second call, with no set, returns a zero value and a nil
error (spec.md US1, SC-001).

### Tests for User Story 1

- [x] T008 [US1] Add a failing serve test in `sdk/go/pluginsdk/credentials_serve_test.go`
  that starts `Serve` with `Web.Enabled` false and an injected `127.0.0.1:0` listener.
  The plugin implements `PerRequestCredentialConsumer` and reports `ExtractCredentials`
  through a channel, not through an error string. Call one with `WithCredentials` and
  `CredentialUnaryClientInterceptor`, then call again with a bare context. The first
  channel value is `test-secret-value`. The second extract is zero and nil.

### Implementation for User Story 1

- [x] T009 [US1] In `sdk/go/pluginsdk/logging.go`, scan credential metadata from the
  map `TracingUnaryServerInterceptorWithLogger` already loaded and store it on the
  handler context before the handler runs. Do not add a second interceptor:
  `metadata.FromIncomingContext` copies every header. The scan does not log and does
  not fail the RPC (contracts/wire-headers.md, plan.md performance goals).

**Checkpoint**: US1 passes on the native server without the Connect client.

---

## Phase 4: User Story 2 - Leave the default unchanged (Priority: P1)

**Goal**: A plugin that has not opted in keeps environment credentials. A missing set
is not an error.

**Independent Test**: A non-opt-in plugin does not advertise acceptance. Attaching a
set changes zero environment entries and does not fail the call (spec.md US2, SC-002,
FR-001, FR-004, FR-005).

### Tests for User Story 2

- [x] T010 [US2] Extend `sdk/go/pluginsdk/credentials_serve_test.go` with a failing
  case for a plugin that does not implement `PerRequestCredentialConsumer`. Snapshot
  `os.Environ` around a native call that carries `test-secret-value`. The snapshot
  matches, the RPC is not failed because of credentials, and `GetPluginInfo` omits
  `supports_per_request_credentials`. Add a case that an opted-in plugin with no set
  returns a nil extract error.

### Implementation for User Story 2

- [x] T011 [US2] Keep `sdk/go/pluginsdk/credentials.go` and `sdk/go/pluginsdk/sdk.go`
  free of `os.Setenv` and free of an automatic RPC abort when credential headers are
  present or absent (research.md "Do not auto-fail"). Absent stays a nil error.

**Checkpoint**: US1 and US2 both pass. The default process environment is untouched.

---

## Phase 5: User Story 3 - Keep values out of records (Priority: P1)

**Goal**: Logs, error strings, and metric labels never contain the fixture secret.

**Independent Test**: Success, an unrelated handler error, and a malformed header that
includes `test-secret-value` produce zero occurrences of that fixture in the log
buffer, `Error()`, and gathered Prometheus labels (spec.md US3, SC-003, FR-006).

### Tests for User Story 3

- [x] T012 [US3] Extend `sdk/go/pluginsdk/credentials_serve_test.go` so the native
  server uses a `zerolog` buffer and `MetricsInterceptorWithRegistry`. Assert the
  buffer, the unrelated handler error, `ErrMalformedCredentials.Error()`, and gathered
  metric label values contain zero copies of `test-secret-value`.

### Implementation for User Story 3

- [x] T013 [US3] Confirm the credential interceptor in `sdk/go/pluginsdk/credentials.go`
  and the tracing interceptor in `sdk/go/pluginsdk/logging.go` do not write credential
  header values. Do not add a metric or a log field in `sdk/go/pluginsdk/metrics.go`.

**Checkpoint**: SC-003 holds on the native path.

---

## Phase 6: User Story 4 - Advertise opt-in without a protocol change (Priority: P2)

**Goal**: Plugin information shows the acceptance note only when the plugin opted in.
The capability list does not grow.

**Independent Test**: Opt-in sets `supports_per_request_credentials` to `true`. A
hand-set `true` is removed when the plugin did not opt in. A provider value `false` is
overwritten when it did. Capability lists match a non-opt-in plugin (spec.md US4,
FR-007, FR-008, FR-013, SC-004).

### Tests for User Story 4

- [x] T014 [US4] Add failing `GetPluginInfo` cases to
  `sdk/go/pluginsdk/credentials_serve_test.go` for configured `PluginInfo` and for
  `PluginInfoProvider`: opted-in note is `true`; non-opt-in deletes a hand-set `true`;
  opted-in overwrites a provider `false`; `Capabilities` match the same plugin without
  the marker. Assert no new `PluginCapability` value.

### Implementation for User Story 4

- [x] T015 [US4] In `sdk/go/pluginsdk/sdk.go`, after `withLegacyCapabilityMetadata` on
  both `GetPluginInfo` paths, set `MetadataSupportsPerRequestCredentials`
  (`supports_per_request_credentials`) to `ValueTrue` when the plugin implements
  `PerRequestCredentialConsumer`, and delete that key otherwise
  (contracts/plugin-info-metadata.md). Do not change `inferCapabilities` or
  `maxValidCapability` in `sdk/go/pluginsdk/plugin_info.go`.

**Checkpoint**: Hosts can see opt-in without a proto change.

---

## Phase 7: User Story 5 - Same attachment on both call paths (Priority: P2)

**Goal**: The web-compatible server and `pluginsdk.Client` carry the same header as
native gRPC. No Connect interceptor is added.

**Independent Test**: A Connect client call with `WithCredentials` and a second call
without a set match US1's results on `Web.Enabled` true (spec.md US5, FR-012, SC-005).

### Tests for User Story 5

- [x] T016 [US5] Add a failing test in `sdk/go/pluginsdk/credentials_serve_test.go`
  that serves `Web.Enabled` true and calls `pluginsdk.Client` (`ProtocolConnect`) with
  `WithCredentials`. The handler observes `test-secret-value`, then a bare context
  observes nothing. Assert `DefaultAllowedHeaders` is unchanged.

### Implementation for User Story 5

- [x] T017 [US5] In `sdk/go/pluginsdk/client.go`, copy a valid context set onto every
  `connect.NewRequest` header (Name, Supports, EstimateCost, BatchCost, GetActualCost,
  GetProjectedCost, GetPricingSpec, GetRecommendations, DismissRecommendation,
  GetBudgets) via one helper. No credential field on `Client`.

- [x] T018 [US5] In `sdk/go/pluginsdk/sdk.go` `serveConnect`, wrap the mux with HTTP
  middleware that copies `X-Finfocus-Credential-*` headers onto `r.Context()` before
  handlers run. Do not add `connect.WithInterceptors`. Do not edit
  `DefaultAllowedHeaders` in `sdk/go/pluginsdk/options.go`.

**Checkpoint**: Native and web-compatible paths both satisfy SC-005.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Docs, the absent-path benchmark, and the quickstart review

- [x] T019 [P] Document the API in `sdk/go/pluginsdk/README.md`, including how it
  differs from CORS `AllowCredentials`, and add a compiled example in
  `sdk/go/pluginsdk/example_test.go` that uses a placeholder value rather than a real
  secret. Godoc every new export.

- [x] T020 [P] Add one paragraph to `PLUGIN_DEVELOPER_GUIDE.md` beside the existing
  environment-credential bullets stating that per-request credentials are opt-in, do
  not replace the process environment, and must not be logged.

- [x] T021 Add `BenchmarkCredentialAbsent` in
  `sdk/go/pluginsdk/credentials_benchmark_test.go`. The absent-header parse must be
  0 allocs/op (plan.md performance goals, constitution VIII).

- [x] T022 Run the commands in `specs/056-per-request-credentials/quickstart.md` and
  the review checks there: empty diff for `proto/`, `sdk/go/proto/`, and
  `sdk/typescript/`; no new `PULUMICOST_` identifier; no pool, router, or instance cap
  (FR-009, FR-010, FR-014, SC-006).

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies
- **Foundational (Phase 2)**: depends on Phase 1 and blocks every story
- **US1 (Phase 3)**: depends on Phase 2; no dependency on later stories
- **US2 (Phase 4)** and **US3 (Phase 5)**: depend on Phase 2 and on the native server
  wiring from US1 (T009)
- **US4 (Phase 6)**: depends on Phase 2; can follow US1. Uses the same serve harness
- **US5 (Phase 7)**: depends on Phase 2 and on the codec from T007. Independent of the
  metadata note, but the serve file is shared with T009
- **Polish (Phase 8)**: depends on US1 through US5

### User Story Dependencies

- **US1 (P1)**: first native round trip. MVP.
- **US2 (P1)**: same harness, non-opt-in and absent cases. No new type.
- **US3 (P1)**: same harness plus a log buffer and a metric registry.
- **US4 (P2)**: `GetPluginInfo` only. Does not require US5.
- **US5 (P2)**: client and Connect middleware. Does not require US4.

### Within Each User Story

- Tests are written and fail before that story's implementation tasks.
- Do not mark a test task done until the test has been seen failing for the missing
  behavior, then passing after the implementation task.

### Parallel Opportunities

- T019 and T020 touch different markdown files and can run together after the API exists.
- T002, T003, and T004 edit one test file, so they are sequential.
- T014 and T016 both edit `credentials_serve_test.go`, so they are sequential.
- No story edits `proto/` in parallel with anything else, because nothing may edit it.

---

## Parallel Example: Polish

```bash
# After US5, documentation files do not overlap:
Task: "T019 sdk/go/pluginsdk/README.md and example_test.go"
Task: "T020 PLUGIN_DEVELOPER_GUIDE.md"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Finish Phase 1 and Phase 2 (the set, the context, the codec).
2. Finish Phase 3 (native attach and read).
3. Stop and run the US1 serve test before advertising the note or the Connect path.

### Incremental Delivery

1. US1 proves one call cannot see the next call's secret.
2. US2 and US3 lock the default and the redaction gate before any doc claims the feature.
3. US4 lets a host see opt-in without a capability enum.
4. US5 repeats US1 on the web-compatible path.

### Parallel Team Strategy

One implementer should own `sdk.go`, `client.go`, and `credentials.go` in order.
Documentation tasks T019 and T020 can proceed once the exports exist.

---

## Notes

- Fixture secret is `test-secret-value` only.
- Do not add `PULUMICOST_*`, a `PluginCapability`, a Connect interceptor, or `os.Setenv`.
- Task IDs are sequential. Do not renumber when a later convergence phase appends tasks.
