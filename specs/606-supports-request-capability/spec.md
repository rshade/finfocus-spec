# Feature Specification: Capability-Scoped Supports Requests

**Feature Branch**: `606-supports-request-capability`

**Created**: 2026-11-08

**Status**: Draft

**Input**: User description: "feat(proto): tell Supports which capability the host is asking about (issue 675).
A Supports request names only the resource, so a plugin that serves actual cost but cannot price projected cost
for that resource has no correct answer. Add an optional capability to the request, reusing the existing plugin
capability list, with 'unspecified' keeping today's meaning (any capability). Closes #675"

## Clarifications

`/speckit-clarify` was not required, because no material ambiguity remained after specification. The issue
fixes the field number, the reused value list, and the meaning of "unspecified". The three remaining choices
have repository precedent and are recorded under Assumptions and Edge Cases:

- Unknown values are read as unspecified.
- The SDK default answer is unchanged.
- Conformance does not check answer values.

## Background

A host asks each plugin "do you support this resource?" before routing a cost call to it, and caches the answer
per feature. The question carries no feature, so the plugin must give one answer for every feature. A plugin
that serves actual cost for a resource but cannot price its projected cost is forced to choose between two wrong
answers:

- "Supported" makes the host call projected cost, which fails and shows a plugin error row.
- "Not supported" makes the host stop routing actual cost to the plugin as well, hiding real data.

This was observed live with a cost-explorer plugin: a storage bucket, and compute instances with spot comparison
off, show projected-cost errors because the plugin must keep answering "supported" for actual cost.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin answers per capability (Priority: P1)

A plugin author receives the capability the host is about to call alongside the resource. For a resource it can
report actual cost for but cannot price, the plugin answers "supported" when asked about actual cost and "not
supported, with a reason" when asked about projected cost.

**Why this priority**: This is the defect in the issue. Without it, no host change can fix the error rows.

**Independent Test**: Build a plugin that reads the requested capability. Send a support question for the same
resource once with actual cost and once with projected cost, and confirm that the two answers differ as the
plugin intends.

**Acceptance Scenarios**:

1. **Given** a plugin that serves actual cost but not projected cost for a resource, **When** the host asks about
   that resource for projected cost, **Then** the plugin can answer "not supported" with a reason.
2. **Given** the same plugin, **When** the host asks about the same resource for actual cost, **Then** the plugin
   can answer "supported".
3. **Given** a request whose capability is unspecified, **When** the plugin reads the requested capability,
   **Then** it sees "unspecified" and answers for any capability, as it does today.
4. **Given** a request carrying a capability value the plugin's SDK version does not know, **When** the plugin
   reads the requested capability through the SDK helper, **Then** it sees "unspecified" and a warning is logged.

---

### User Story 2 - Host sends the capability it is about to call (Priority: P1)

A host developer using the SDK client asks a plugin about a resource for a named capability, without building the
raw request by hand. Existing host code that asks without a capability keeps compiling and sends "unspecified".

**Why this priority**: A plugin can only answer per capability once hosts can ask. The host follow-up in the core
repository depends on this.

**Independent Test**: Using the SDK client against a test plugin that records the request, ask about a resource
for projected cost and confirm the plugin received projected cost. Ask with the existing call and confirm the
plugin received "unspecified".

**Acceptance Scenarios**:

1. **Given** a host using the Go SDK client, **When** it asks about a resource for a named capability, **Then**
   the plugin receives that capability.
2. **Given** a host using the existing Go client call with no capability, **When** it asks about a resource,
   **Then** the plugin receives "unspecified" and the answer is unchanged.
3. **Given** a host using the TypeScript client, **When** it builds a support request with a capability, **Then**
   the plugin receives that capability.

---

### User Story 3 - Existing plugins keep working and pass conformance (Priority: P2)

A plugin author who never reads the new capability upgrades the SDK. The plugin compiles unchanged, answers
exactly as before, and still passes conformance. The conformance suite also exercises capability-specific
questions, so plugins that do read the capability are tested on that path.

**Why this priority**: Backward compatibility is constitutional, and conformance is how authors find out whether
they are compatible.

**Independent Test**: Run the conformance suite against the default mock plugin, which ignores the capability,
and against a plugin that answers per capability. Both pass.

**Acceptance Scenarios**:

1. **Given** a plugin that ignores the capability, **When** conformance runs, **Then** it passes with no new
   failures.
2. **Given** any plugin, **When** conformance sends an unspecified request, **Then** the check confirms a valid
   response, the same as today's support check.
3. **Given** a plugin that declares capabilities, **When** conformance sends a capability-specific request for
   each declared resource-scoped capability (FR-014), **Then** each answer must be a valid response and not an
   error. Either a "supported" or a "not supported" answer passes. A plugin that declares only capabilities
   outside the resource-scoped set gets only the unspecified request.
4. **Given** a plugin that answers a capability-specific request with an error, **When** conformance runs,
   **Then** the check fails and names the capability.

---

### User Story 4 - Plugin authors and hosts know the rules (Priority: P3)

The plugin guide and SDK documentation explain how to answer per capability, and that an answer to an unspecified
request covers every capability. Release notes describe the addition.

**Why this priority**: The contract is only useful if both sides read it the same way, but the code paths stand on
their own without it.

**Independent Test**: A reader of the plugin guide can find the rule for unspecified requests, an example that
answers per capability, and the host call that sends a capability.

**Acceptance Scenarios**:

1. **Given** the plugin developer guide, **When** an author looks up the support check, **Then** it explains
   reading the requested capability and answering per capability.
2. **Given** the guide, **When** a host developer reads it, **Then** it states that an answer to an unspecified
   request covers every capability.

### Edge Cases

- **Unspecified capability**: The request means "any capability". The plugin answers as it did before the field
  existed. Old hosts never send the field, so every request from them is unspecified.
- **Unknown capability value**: A newer host may send a value an older plugin's SDK does not define, including a
  negative or out-of-range number. The SDK helper reports "unspecified" and logs a warning, so routing keeps
  working instead of failing.
- **Capability the plugin does not declare**: The plugin may answer "not supported". The SDK does not answer on
  its behalf; the plugin's own support logic decides.
- **Capability that is not about a resource**: Values such as dry run or budgets carry no resource meaning. The
  request is still valid, and the plugin decides. The docs list the resource-scoped capabilities hosts are
  expected to send.
- **Plugin without its own support logic**: The SDK default answer ("not supported" with the existing reason) is
  unchanged for every capability.
- **Old plugin, new host**: An old plugin ignores the unknown field on the wire and answers as before.
- **Consistency**: If a plugin answers "supported" for some specific capability of a resource, it should also
  answer "supported" to the unspecified question for that resource. This is documented guidance, not enforced.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The support request MUST carry an optional capability. It reuses the existing plugin capability
  list, so no new list of values is created.
- **FR-002**: An unspecified capability MUST mean "any capability", with exactly the behavior that existed before
  the field.
- **FR-003**: The change MUST be additive. No field, value, method, or exported signature is removed or
  renamed, old hosts and old plugins interoperate unchanged, and the breaking-change check passes. This is a
  minor version change.
- **FR-004**: The Go SDK MUST give plugin support handlers a way to read the requested capability without new
  method signatures. Existing support handlers MUST compile and behave the same. The SDK's default answer for
  plugins without support logic, the provider and region registry rejection, and the mapping of handler errors
  to status codes are all unchanged.
- **FR-005**: The SDK's read helper MUST report "unspecified" for any value other than "unspecified" that the
  SDK version does not define, including negative values. Such a value MUST be logged as a warning at most once
  per distinct value per process, so a host that asks on every routing decision cannot flood plugin logs.
- **FR-006**: Reading the requested capability MUST NOT allocate memory for a known value or for "unspecified".
  Unknown values are exempt: they are rare, and deduplicating their warning needs a lookup keyed by the value.
- **FR-007**: The Go SDK client MUST let hosts ask about a resource for a named capability. The existing support
  call and the resource-type convenience call MUST keep their signatures and send "unspecified". The
  resource-type convenience call deliberately gets no capability form: a host that cares about the capability
  builds a descriptor.
- **FR-008**: The client MUST apply the same input checks to a capability-specific request that it applies today
  (resource present, resource type present).
- **FR-009**: The TypeScript SDK MUST expose the new field, and its client MUST pass it through. The TypeScript
  SDK is host-side only (clients and gateways, with no plugin server), so it gets no read helper.
- **FR-010**: The SDK server MUST pass the capability to the plugin's support logic unchanged, over both
  transports.
- **FR-011**: When a support handler fails, the server log MUST include the requested capability.
- **FR-012**: The support request contract validator MUST accept every capability value, including unknown ones,
  so newer hosts are not rejected.
- **FR-013**: The reference mock plugin MUST be configurable to answer "not supported" for chosen capabilities.
  The capability is considered only after the existing provider and resource checks pass, and the reason names
  the capability and the resource type. Its default MUST ignore the capability, as today.
- **FR-014**: Conformance MUST include a Standard-level check. It sends an unspecified request, then a
  capability-specific request for each resource-scoped capability the plugin declares. When a plugin declares
  none, or capability discovery is unimplemented (legacy), it sends projected cost and actual cost. Every answer
  MUST be a valid response, not an error, and a failure MUST name the capability. Any other discovery error
  fails the check. The resource-scoped set is projected cost, actual cost, carbon, recommendations, energy,
  water, pricing spec, estimate cost, batch cost, and historical carbon. The other capabilities describe
  plugin-level or dataset services, and the check does not ask about them.
- **FR-015**: A plugin that ignores the capability MUST pass every conformance level it passed before.
- **FR-016**: The plugin developer guide, the SDK readme, the testing readme, and the TypeScript client readme
  MUST describe the field, the unspecified rule, the read helper, the host call, and the new conformance check.
  The guide's Supports section MUST also tell hosts that an old plugin answers a capability-specific request
  with its "any" answer. Every Go and TypeScript code example MUST compile.
- **FR-017**: The field MUST be documented in the protocol definition, including the rule for unspecified values.
- **FR-018**: The change's commit and pull request text MUST describe the new field and the host call, because
  the release notes are generated from it.

### Key Entities

- **Support request**: The question a host asks a plugin. It has the resource it already carried, plus the new
  capability the host is about to call.
- **Plugin capability**: The existing list of things a plugin can do (projected cost, actual cost, carbon, and
  others). It is now also used to name the operation being asked about. "Unspecified" means any.
- **Support answer**: Unchanged. "Supported" or "not supported", with a reason and the plugin's capabilities.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin can give different answers for the same resource depending on the capability asked about.
  This is shown by a test in which actual cost answers "supported" and projected cost answers "not supported".
- **SC-002**: 100% of existing tests and conformance checks pass without changes to plugins that ignore the
  capability.
- **SC-003**: The breaking-change check reports zero breaking changes.
- **SC-004**: A host can send a capability with one SDK call, and existing host calls need zero code changes.
- **SC-005**: Reading the requested capability costs zero allocations, as shown by a benchmark.
- **SC-006**: The guide's Supports section answers "what does an unspecified request mean?", and its
  per-capability example compiles.

## Assumptions

- Out of scope here, with follow-ups tracked in other repositories: the core host sending its feature as the
  capability, and the cost-explorer plugin answering "not supported" for projected cost on types it cannot price.
- The SDK does not answer per capability on a plugin's behalf, for example by refusing capabilities the plugin
  does not declare. The default answer for plugins without support logic is unchanged.
- The answer is not changed and does not echo the capability. The host already knows what it asked.
- Field number 2 is free on the support request, which today has only the resource (field 1).
- The version bump is minor and is applied by the release automation, not by hand.
- Conformance does not check answer values, because the correct answer depends on the plugin. It checks only that
  each answer is valid and not an error.
