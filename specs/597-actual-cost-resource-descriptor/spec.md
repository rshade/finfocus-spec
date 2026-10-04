# Feature Specification: Resource Descriptor on Actual Cost Requests

**Feature Branch**: `597-actual-cost-resource-descriptor`

**Created**: 2026-10-04

**Status**: Draft

**Input**: User description: "feat(proto): add a ResourceDescriptor to GetActualCostRequest. Add
`ResourceDescriptor resource = 11;` to GetActualCostRequest (field 10 stays free for a billing account
name). Same semantics as GetProjectedCostRequest.resource including attributes and host redaction rules
(#617). Unset means host sent none; plugins fall back to tags, resource_id, arn. tags keeps its meaning
(cloud tags usable as billing filters); hosts SHOULD keep sending them. When resource is set, plugins read
pricing dimensions from it, not tags. ValidateActualCostRequest runs ValidateResourceDescriptor on
resource when set. Pagination and dry run unchanged; host sends same resource on every page. Docs,
bindings, conformance case for a plugin that ignores resource, and a mock that reads it. Closes #620"

## Clarifications

`/speckit-clarify` was not required. The issue fixes the field number, the field kept free, the
unset and set meanings, the precedence over `tags`, the validation rule, and what is out of scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - List-price plugin prices actual cost from the same inputs as projected cost (Priority: P1)

A host asks a plugin for the actual cost of one resource. The plugin derives actual cost from list
price, the same way it derives projected cost. The host sends the resource's description, including
its provider, type, SKU, region, and declared inputs such as instance count or disk size, in the same
form it uses for projected cost. The plugin reads those inputs from the description, so the actual
cost and the projected cost agree.

**Why this priority**: This is the defect the issue reports. Today a scale set with three instances
projects three instances but its actual cost prices one, because the actual cost request has no place
for declared inputs.

**Independent Test**: Send an actual cost request that carries a resource description to the
reference (mock) plugin. The plugin's results show that it read the description, and the request
passes SDK validation.

**Acceptance Scenarios**:

1. **Given** an actual cost request carrying a resource description with provider, type, SKU,
   region, and declared inputs, **When** the request is validated, **Then** it passes.
2. **Given** the same request, **When** the reference plugin answers, **Then** its results reflect
   the description it received, so a test can see the description reached the plugin.
3. **Given** a request carrying a description and cloud tags that both name a region, **When** a
   plugin follows the documented rule, **Then** it reads the region from the description and treats
   the tag as a label only. This is a plugin rule the SDK cannot enforce; it is met by the field
   comment and the plugin documentation.

---

### User Story 2 - Existing hosts and plugins keep working (Priority: P1)

A host built before this change sends no resource description. A plugin built before this change
ignores one it receives. Nothing either of them sees changes.

**Why this priority**: Backward compatibility is a constitutional rule, and every existing caller and
plugin is in this state.

**Independent Test**: Send actual cost requests without a description to the reference plugin and
through the conformance suite. Run the conformance suite against a plugin that ignores the
description. All pass, with the same results as before this change.

**Acceptance Scenarios**:

1. **Given** a request with no resource description, **When** it is validated and answered, **Then**
   it behaves exactly as before this change.
2. **Given** a plugin that ignores the description, **When** the conformance suite sends it a request
   that carries one, **Then** the plugin passes.
3. **Given** a host built after this change that sends a description, **When** it calls a plugin built
   before it, **Then** the call succeeds and the plugin ignores the unknown field.

---

### User Story 3 - Validation rejects an oversized or malformed description (Priority: P2)

A host or plugin validates an actual cost request. When the request carries a resource description,
the same limits that apply to a projected cost description apply here: the size of declared inputs,
the number and length of tags, and the field lengths.

**Why this priority**: The limits protect plugins from unbounded input. Sharing them with projected
cost means a host builds one description and it is valid on both paths.

**Independent Test**: Validate actual cost requests whose descriptions break each limit, and one with
no description. The broken ones fail with an invalid-argument error that names the problem; the one
without a description passes.

**Acceptance Scenarios**:

1. **Given** a request whose description carries declared inputs larger than the attribute size
   limit, **When** it is validated, **Then** validation fails with the attribute size error.
2. **Given** a request with no description, **When** it is validated, **Then** the description is
   not checked and the rest of the request is validated as before.
3. **Given** a request whose description breaks a tag limit, **When** it is validated, **Then**
   validation fails with the same error a projected cost description would get.

---

### User Story 4 - TypeScript host sends the description on every page (Priority: P2)

A TypeScript host sets a resource description on an actual cost request, including through the
paginated actual cost iterator. The description reaches the plugin on every page.

**Why this priority**: The constitution requires the TypeScript SDK to stay in step with the proto.

**Independent Test**: Use the TypeScript actual cost iterator with a description against a mocked
transport, and confirm every page request carries it.

**Acceptance Scenarios**:

1. **Given** a TypeScript actual cost request carrying a description, **When** the iterator fetches
   several pages, **Then** every page request carries the same description.

---

### Edge Cases

- **Description unset**: The host sent none. Plugins fall back to `tags`, `resource_id`, and `arn`,
  as today. This is never an error.
- **Description set, tags also set**: `tags` keeps its meaning, the resource's cloud tags, usable as
  billing filters. Pricing dimensions come from the description, not from `tags`. A user tag named
  `region`, `sku`, or `provider` does not override the description.
- **Description `id` differs from `resource_id`**: `resource_id` stays the required identifier of the
  actual cost request. The SDK does not compare it with the description's `id`.
- **Description present but empty**: It is validated like a projected cost description. The SDK's
  zero-allocation validator checks lengths and limits only; the contract validator also requires
  provider and resource type, as it does for every other descriptor.
- **Dry run**: Unchanged. A dry-run request may carry a description; the SDK still validates it if it
  is set.
- **Pagination**: The description is part of the query a page token continues. A host sends the same
  description on every page of one query.
- **Redacted inputs**: The host redaction rules for declared inputs (#617) apply unchanged. The SDK
  cannot detect what a host dropped.
- **Field 10**: Stays unused, held for a later billing account name.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The actual cost request MUST carry an optional resource description, as a new field
  numbered 11, of the same message type the projected cost request uses. No existing field changes
  number, type, or meaning, and field 10 MUST stay unused.
- **FR-002**: The field's documentation MUST state that it has the same meaning as the projected cost
  request's resource description, including declared inputs and the host redaction rules.
- **FR-003**: The documentation MUST state that an unset description means the host sent none, and
  that plugins then fall back to `tags`, `resource_id`, and `arn`.
- **FR-004**: The documentation MUST state that `tags` keeps its meaning (the resource's cloud tags,
  usable as billing filters), and that hosts SHOULD keep sending them so older plugins are unaffected.
- **FR-005**: The documentation MUST state that when the description is set, plugins read pricing
  dimensions from it and not from `tags`.
- **FR-006**: The documentation MUST state that a host sends the same description on every page of
  one paginated query, and that pagination and dry run are otherwise unchanged.
- **FR-007**: The SDK's actual cost request validator MUST validate the description with the same
  descriptor rules the SDK applies elsewhere when it is set, and MUST skip it when it is unset.
- **FR-008**: The conformance contract validator for actual cost requests MUST validate the
  description with its descriptor rules when it is set, and MUST skip it when it is unset.
- **FR-009**: A valid actual cost request with no description MUST stay zero-allocation in the SDK
  validator.
- **FR-010**: The generated Go and TypeScript request types MUST expose the description.
- **FR-011**: The reference (mock) plugin MUST read the description when it is set, in a way a test
  can observe, and MUST behave as before when it is unset.
- **FR-012**: The SDK conformance suite MUST include a check that sends an actual cost request
  carrying a description and passes a plugin that ignores it.
- **FR-013**: The TypeScript actual cost iterator MUST carry the description on every page request.
- **FR-014**: Plugin-author documentation (the property mapping guide, the plugin developer guide's
  actual cost section, and both SDK READMEs) MUST describe the field, the fallback, and the precedence
  over `tags`.

### Key Entities

- **Actual cost request**: The host's query for the historical cost of one resource over a time range.
  Gains an optional resource description.
- **Resource description**: The existing message that describes one resource on the projected cost,
  supports, and estimate paths: provider, type, SKU, region, tags, and declared inputs.
- **Cloud tags**: The resource's own labels. On the actual cost request they stay billing filters, not
  pricing dimensions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The actual cost request carries the same resource description message as the projected
  cost request, so 100% of the description fields a plugin can read on the projected path are
  available to it on the actual path.
- **SC-002**: 100% of existing actual cost tests and conformance checks pass unchanged, which shows
  that hosts and plugins that do not use the field are unaffected.
- **SC-003**: The actual path checks a description with the same descriptor rules the other paths
  use, so a description is valid or invalid the same way everywhere. Representative violations
  are rejected on the actual path with the same error they get elsewhere: oversized declared inputs
  and an over-long tag value on both validator layers, and a missing provider or type on the contract
  layer.
- **SC-004**: The conformance suite passes a plugin that ignores the description, with no false
  failures.
- **SC-005**: The protocol compatibility check against the current main branch reports no breaking
  change.

## Assumptions

- The host builds the description the same way for projected and actual cost. How it does so is a
  host concern (finfocus core), not part of this change.
- The SDK does not reconcile the description with `tags`, `resource_id`, or `arn`. Precedence is a
  documented rule for plugins, not something a validator can check.
- No new plugin capability flag is needed. The field is optional, and a plugin that ignores it stays
  compliant.
- The batch cost request is out of scope. It does not embed the actual cost request.
- Out of scope, per the issue: changing what `tags` means, removing the tags hosts inject today, and
  plugins that look up billed cost by `resource_id` or `arn`.
