# Feature Specification: Opt-In Per-Request Credentials

**Feature Branch**: `056-per-request-credentials`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "Opt-in per-request cloud credentials so one plugin process can
serve more than one tenant. The default remains one process per organization with
credentials in the environment, as spec 029 requires. A plugin opts in by implementing
one exported interface. The host attaches credentials to a single call. The plugin reads
them with an extract helper and otherwise keeps using ambient environment credentials.
Missing per-request credentials is not an error. Credential values are never written to
logs, error strings, or metrics. This repository does not build the process pool, router,
or instance cap. Those stay in the host. Do not add a protobuf field or a new
PluginCapability enum. Advertise support through the existing PluginInfo metadata map
when that can be done without a proto change; otherwise the interface is the signal, and
the plan says why. Cover gRPC metadata. Cover Connect only if it can carry the same
headers without a new interceptor design; if not, the plan records Connect as unchanged
and why. No new PULUMICOST_* names."

## Overview

A host that serves many tenants can already give each organization its own plugin
process and put that organization's cloud credentials in the process environment. That
remains the default. This slice adds a second, optional way to pass credentials: the
host attaches them to one call, and only a plugin that has opted in reads them.

The point of the slice is to make that optional path safe to build on. It does not
decide which tenants share a process. It does not reduce memory by itself. A host that
never attaches credentials, and a plugin that does not opt in, see no change.

**Relationship to spec 029**: spec 029 requires one process per organization, environment
credentials, and no credentials in request metadata for that default. This slice does
not relax those rules for plugins that have not opted in. Issue #220 is the deferred
follow-on spec 029 named. The assessment under
`.specify/assessments/per-request-credentials/` said not to specify this until a host
measured memory and cold start. The maintainer overruled that wait. The default model
is unchanged either way.

## Clarifications

### Session 2026-09-28

- Q: Should this wait until a host measures memory and cold start? → A: No. The
  maintainer ordered the opt-in slice now. The assessment's wait remains a rejected
  alternative in the plan. This slice does not claim those measurements and does not
  change the default.
- Q: Where do the credential values live? → A: On that one call only. They are not
  written to the process environment, a cache, a file, or any other store.
- Q: What if a call has no per-request credentials? → A: That is not an error. An
  opted-in plugin keeps using the process environment credentials.
- Q: How does a host tell that a plugin accepts this? → A: From the existing plugin
  information notes. No new message field and no new capability entry.
- Q: Does this repository launch, route, or cap plugin processes? → A: No. Those stay
  with the host.
- Q: What if the attached material is unusable? → A: The call can report a fixed
  failure that does not contain the material. That failure is different from "nothing
  was attached."
- Q: Are caller identity and cloud secrets the same problem? → A: No. Open issue #195
  is caller identity. This slice is only named cloud credential values for one call.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Attach credentials to one call (Priority: P1)

A host is talking to a plugin that has opted in. The host attaches one tenant's named
credential values to a single call. The plugin reads those values for that call and no
other. The next call can carry a different tenant's values, or none.

**Why this priority**: Without a per-call attachment that does not leak into the next
call, a shared process cannot tell tenants apart. Everything else in the slice depends
on this boundary.

**Independent Test**: On one opted-in plugin process, make one call with a known
credential value and a second call with no per-request credentials. The first call
observes the value. The second call does not observe it and does not fail because the
value is missing.

**Acceptance Scenarios**:

1. **Given** an opted-in plugin and a call with one named credential value, **When** the
   plugin reads the call, **Then** it receives that name and value.
2. **Given** the same plugin process, **When** the next call has no per-request
   credentials, **Then** the read reports that nothing was attached and the call is not
   failed for that reason.
3. **Given** two calls in sequence with different values, **When** each call is read,
   **Then** each call sees only its own value.
4. **Given** a host that attached a value, **When** the caller then changes the original
   collection, **Then** the call still has the value that was attached.

---

### User Story 2 - Leave the default tenant process unchanged (Priority: P1)

An operator keeps running one plugin process per organization and passes cloud
credentials through the process environment. Plugins that have not opted in keep working
that way, including when a caller happens to attach per-request values.

**Why this priority**: spec 029 made process isolation the security default. This slice
is not allowed to replace it.

**Independent Test**: Run a plugin that has not opted in. Its plugin information does
not advertise per-request credentials. Calls with and without attached values still
follow the environment-credential behavior the plugin already had. The process
environment is not modified by the attachment.

**Acceptance Scenarios**:

1. **Given** a plugin that has not opted in, **When** plugin information is read,
   **Then** it does not advertise per-request credentials.
2. **Given** that plugin, **When** a call arrives with or without attached values,
   **Then** the attachment is not written into the process environment and the call is
   not failed for lack of per-request credentials.
3. **Given** an opted-in plugin and a call with no attached values, **When** the plugin
   handles the call, **Then** missing per-request credentials are not an error and the
   plugin is expected to keep using its environment credentials.

---

### User Story 3 - Keep credential values out of records (Priority: P1)

A credential value is attached to a call. The call may succeed or fail. Logs, error
text, and metrics from this SDK do not contain the value.

**Why this priority**: A value that shows up in a log or an error is a leaked secret,
which defeats the reason for keeping tenants apart.

**Independent Test**: Attach the fixture value `test-secret-value`. Succeed a call, fail
a call for an unrelated reason, and reject an unusable set that also contains that
value. No log line, error string, or metric label contains `test-secret-value`.

**Acceptance Scenarios**:

1. **Given** a call carrying `test-secret-value`, **When** the call succeeds, **Then**
   logs and metrics do not contain `test-secret-value`.
2. **Given** a call carrying `test-secret-value`, **When** the plugin returns its own
   failure that does not include the value, **Then** the SDK's recorded error text still
   does not contain `test-secret-value`.
3. **Given** unusable credential material that includes `test-secret-value`, **When**
   the material is rejected, **Then** the failure text is a fixed message and does not
   contain `test-secret-value`.

---

### User Story 4 - Advertise opt-in without a protocol change (Priority: P2)

A host must know which plugins will read per-request credentials before it attaches
them. It learns that from plugin information it can already request. The capability list
does not grow.

**Why this priority**: The host and the plugin are separate processes. The host cannot
see which behaviors the plugin declared except through plugin information. This is
secondary only because a single-process test can call the plugin directly; a real host
cannot.

**Independent Test**: Compare plugin information for an opted-in plugin and a plugin
that has not opted in. Only the opted-in plugin's notes say it accepts per-request
credentials. Both capability lists match. Forcing the note on without opting in does
not stick. Opting in without writing the note still reports acceptance.

**Acceptance Scenarios**:

1. **Given** an opted-in plugin, **When** plugin information is read, **Then** the notes
   say it accepts per-request credentials and the capability list is unchanged.
2. **Given** a plugin that has not opted in but whose notes were hand-set to claim
   acceptance, **When** plugin information is read, **Then** that claim is not present.
3. **Given** an opted-in plugin whose own notes omit or deny acceptance, **When** plugin
   information is read, **Then** the notes still say it accepts per-request credentials.

---

### User Story 5 - Use the same attachment on both call paths (Priority: P2)

Hosts reach plugins on the native call path and on the web-compatible call path. The
same named values can be attached on either path, and an opted-in plugin reads the same
values.

**Why this priority**: A helper that works on only one path would split hosts. It is
not the first story because the call boundary in story 1 can be shown on one path.

**Independent Test**: Attach the same name and value once on each path. The opted-in
plugin reads the same pair both times. A call with no attachment still reports nothing
attached on both paths.

**Acceptance Scenarios**:

1. **Given** the native call path, **When** a host attaches a name and value, **Then**
   the opted-in plugin reads that pair.
2. **Given** the web-compatible call path, **When** a host attaches the same pair,
   **Then** the opted-in plugin reads the same pair.
3. **Given** either path, **When** no values are attached, **Then** the read reports
   that nothing was attached.

### Edge Cases

- No per-request credentials on the call. This is not an error.
- An empty set of values. This is unusable, not the same as an omitted set.
- The same name twice, differing only by letter case. This is unusable.
- A name that, after ignoring letter case, is empty or is not a lowercase letter
  followed only by lowercase letters, digits, underscores, and hyphens. A value that
  is empty or contains a byte outside the printable range from space through tilde.
  This is unusable. The failure does not quote the name or the value.
- More than 16 names, a name longer than 64 characters, a value longer than 4096
  characters, or a set whose values together exceed 16384 characters. This is unusable.
- The caller changes or discards the original collection after attaching. The call
  keeps the attached copy.
- Two calls overlap on one process. Each sees only its own values.
- The plugin has not opted in. Attached values are not written to the process
  environment and are not advertised.
- The call fails for a reason other than credentials while values are attached. SDK
  records of that failure do not contain the values.
- A hand-written plugin-information note disagrees with whether the plugin opted in.
  Opt-in wins, in both directions.
- Issue #195 identity material is not accepted as these credential values and is not
  defined here.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The default MUST remain one plugin process per organization, with that
  organization's cloud credentials supplied by the process environment, as spec 029
  requires. This slice MUST NOT replace that default.
- **FR-002**: A plugin MUST opt in by declaring one exported behavior. Plugins that do
  not opt in MUST keep today's credential behavior.
- **FR-003**: A host MUST be able to attach one set of named credential values to a
  single call, and a different set or no set to the next call on the same client.
- **FR-004**: An opted-in plugin MUST be able to read the set attached to the current
  call. When no set is attached, that read MUST NOT be an error, and the plugin
  continues to use its process environment credentials.
- **FR-005**: Values attached to a call MUST NOT be visible on any other call. They
  MUST NOT be written to the process environment, a cache, a file, or any store that
  outlives the call.
- **FR-006**: This SDK MUST NOT write credential values into logs, error strings, or
  metrics. A rejection of unusable material MUST use a fixed message that contains
  neither the name nor the value.
- **FR-007**: Hosts MUST learn acceptance from the existing plugin-information notes.
  This slice MUST NOT add a message field or a capability-list entry.
- **FR-008**: The acceptance note MUST be present only when the plugin opted in. A
  hand-written note MUST NOT advertise acceptance the plugin did not declare, and an
  opted-in plugin MUST NOT be reported as not accepting.
- **FR-009**: This repository MUST NOT add a process pool, a request router, or a
  plugin-instance cap. Those remain host responsibilities under spec 029.
- **FR-010**: This slice MUST NOT introduce a new name with the legacy PulumiCost
  prefix.
- **FR-011**: An omitted set MUST be treated as "not attached." An empty set, a
  duplicate name after ignoring letter case, a name that is not a lowercase letter
  followed only by lowercase letters, digits, underscores, and hyphens, a value that
  is empty or contains any byte outside the inclusive range from space (0x20) through
  tilde (0x7E), more than 16 names, a name longer than 64 characters, a value longer
  than 4096 characters, or values that together exceed 16384 characters MUST be
  rejected as unusable and MUST NOT be treated as a successful attachment. The
  rejection MUST NOT quote the name or the value.
- **FR-012**: The same attached names and values MUST be readable on the native call
  path and on the web-compatible call path.
- **FR-013**: Plugin information for an opted-in plugin MUST keep the same capability
  list it would have had without this opt-in.
- **FR-014**: Open issue #195 MUST remain out of this slice. Caller identity is not a
  credential value and is not given a new channel here.

### Key Entities

- **Credential set**: The named secret values attached to one call. It has at least one
  name when it is usable. Names are matched without regard to letter case. It is not a
  file, a token cache, or a process-wide setting.
- **Opt-in**: The single declaration a plugin makes when it will read a credential set
  from the call. Declaring it does not by itself change the process environment.
- **Acceptance note**: The existing plugin-information note that tells a host the plugin
  opted in. It is not a new capability entry.
- **Unusable material**: A set the host attempted to provide that breaks the rules for a
  credential set. It is not the same as leaving the set off the call.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On one opted-in plugin process, a call that carries a credential value
  shows that value, and the immediately following call that carries none does not show
  it and is not failed for a missing set. Both results hold for 2 of 2 calls in that
  sequence.
- **SC-002**: A plugin that has not opted in advertises acceptance on 0 of its plugin
  information reads, and attaching values changes 0 process-environment entries.
- **SC-003**: In every exercised success, unrelated failure, and rejection that uses the
  fixture value `test-secret-value`, 0 log lines, 0 error strings, and 0 metric labels
  contain that value.
- **SC-004**: An opted-in plugin's capability list is the same length and the same
  entries as the same plugin without the opt-in, and its plugin information includes
  the acceptance note on 1 of 1 reads.
- **SC-005**: One native call and one web-compatible call carrying the same name and
  value both yield that pair. One call with no set on each path reports that nothing
  was attached.
- **SC-006**: This repository's change set contains no process pool, request router, or
  instance-cap implementation. A reviewer can confirm that by the absence of those
  behaviors in the slice.

## Assumptions

- The maintainer's order to build the slice now overrides the assessment's
  needs-clarification verdict. The plan records that verdict as a rejected alternative.
  The unsourced process-size figure on issue #220 is not a success criterion.
- Spec 029's idle timeout, instance cap, and per-organization launch rules stay host
  obligations. This repository still does not contain that host.
- A credential value is short text the plugin can hand to a cloud client for that call.
  Large documents and binary blobs are out of scope. Limits are fixed in the plan.
- Plugins that can only discover cloud credentials from the process environment do not
  become multi-tenant by opting in. Opt-in means the plugin reads the call. This SDK
  does not rewrite the process environment on the plugin's behalf.
- Hosts attach values only to plugins that advertise acceptance. This SDK does not
  route the call.
- Issue #195 stays a separate identity question.
- No new third-party library is required.

## Scope Boundaries

### In Scope

- Attaching named credential values to one call and reading them back on that call.
- Opt-in declaration and an acceptance note on existing plugin information.
- The same attachment on the native call path and the web-compatible call path.
- Rejection of unusable material without echoing it.
- Tests that prove values are not retained and are not written to logs, errors, or
  metrics.

### Out of Scope

- Replacing one process per organization as the default.
- A process pool, a router, an idle timeout, or an instance cap.
- A new message field or a new capability-list entry.
- Caller identity, token verification, and issue #195.
- Loading, refreshing, or caching cloud credentials inside this SDK.
- Writing credential values into the process environment for the duration of a call.
- A browser-origin policy change so web pages can send cloud secrets.
- Any new legacy PulumiCost-prefixed name.
