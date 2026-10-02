# Feature Specification: Caller-Supplied Billing Account ID on Actual Cost Requests

**Feature Branch**: `585-actual-cost-billing-account-id`

**Created**: 2026-10-01

**Status**: Draft

**Input**: User description: "feat(proto): add billing_account_id to GetActualCostRequest. Add optional
`string billing_account_id = 9;` on GetActualCostRequest. Empty means caller did not supply an id;
plugins MUST NOT invent a value and may leave ActualCostResult.focus_record unset while still returning
cost. Non-empty is the id the plugin passes into FocusCostRecord.billing_account_id; it is not a filter
and not summed into cost. Do not put the id in tags. Do not add billing_account_name; leave field 10 free
for it. Closes #590. Motivation: rshade/finfocus-plugin-azure-public issue #46."

## Clarifications

`/speckit-clarify` was not required. The issue fixes the field number, the reserved field, the empty
versus populated meaning, and caller precedence over plugin-discovered ids.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host supplies the billing account for a FOCUS record (Priority: P1)

A host asks a cost source plugin for the actual cost of one resource and knows which billing
account that resource is billed to. The host puts that account id on the actual cost request. The
plugin copies the id into the FOCUS record it attaches to each result. The record passes the SDK's
FOCUS validation, so the host gets a FOCUS row without the plugin guessing an account.

**Why this priority**: This is the gap the issue reports. Today a plugin that does not discover the
billing account itself cannot attach a valid FOCUS record to an actual cost result. The Azure public
pricing plugin (plugin issue #46) leaves the record out for that reason.

**Independent Test**: Send an actual cost request with a non-empty billing account id to the
reference (mock) plugin. Every attached FOCUS record carries that id and passes FOCUS record
validation.

**Acceptance Scenarios**:

1. **Given** a plugin that attaches FOCUS records, **When** a host sends an actual cost request with
   billing account id `"ba-123"`, **Then** each returned FOCUS record has billing account id
   `"ba-123"` and passes FOCUS record validation.
2. **Given** the same request, **When** the plugin computes cost, **Then** the returned cost values
   are identical to those for a request without a billing account id. The id is not a filter and
   does not change cost.

---

### User Story 2 - Host does not know the billing account (Priority: P1)

A host sends an actual cost request without a billing account id. This is the state of every
existing host. The plugin must not invent an account id. It returns the cost, and it may leave the
FOCUS record unset if it cannot build a valid record without the id.

**Why this priority**: Backward compatibility is a constitutional requirement. Every existing caller
is in this state, and nothing it sees may change.

**Independent Test**: Send an actual cost request with no billing account id to the reference
plugin, and send the same request through an older client. Both succeed, both return the same costs
as before this change, and no FOCUS record carries a made-up account id.

**Acceptance Scenarios**:

1. **Given** a request with an empty billing account id, **When** the reference plugin answers,
   **Then** it returns the cost results and does not attach a FOCUS record.
2. **Given** a client built before this change, **When** it calls a plugin built after it, **Then**
   the call succeeds and the plugin sees an empty billing account id.
3. **Given** a client built after this change sending a non-empty id, **When** it calls a plugin
   built before it, **Then** the call succeeds and the plugin ignores the unknown field.

---

### User Story 3 - Plugin author checks the contract (Priority: P2)

A plugin author runs the SDK conformance suite against their plugin. The suite sends an actual cost
request with a billing account id and checks that any FOCUS record the plugin attaches carries that
id.

**Why this priority**: The conformance suite is how plugin authors learn the rules. Without a check,
the field is a comment that nobody enforces.

**Independent Test**: Run the conformance check against a plugin that echoes the id (passes), one
that writes a different id (fails, naming the field), and one that attaches no FOCUS record (passes).

**Acceptance Scenarios**:

1. **Given** a plugin that copies the request id into its FOCUS records, **When** the conformance
   check runs, **Then** it passes.
2. **Given** a plugin whose FOCUS records carry a different billing account id, **When** the check
   runs, **Then** it fails and names the billing account id field.
3. **Given** a plugin that attaches no FOCUS records, **When** the check runs, **Then** it passes.

---

### User Story 4 - TypeScript host sets the id (Priority: P2)

A TypeScript host sets the billing account id on an actual cost request, including through the
paginated actual cost iterator. The id reaches the plugin on every page.

**Why this priority**: The constitution requires the TypeScript SDK to stay in sync with the proto.

**Independent Test**: Use the TypeScript actual cost iterator with a billing account id against a
mocked transport, and confirm every page request carries the id.

**Acceptance Scenarios**:

1. **Given** a TypeScript actual cost request with billing account id `"ba-123"`, **When** the
   iterator fetches several pages, **Then** every page request carries `"ba-123"`.

---

### Edge Cases

- **Empty string**: Means "not supplied." It is not an error and it is never a valid account id.
  The plugin must not substitute a default, a placeholder, or an id derived from other request
  fields.
- **Whitespace-only value**: Treated as a value the caller supplied. The SDK does not trim it. Any
  FOCUS record built from it fails FOCUS validation only if FOCUS validation already rejects it.
- **Plugin has its own account data**: When the caller supplies a non-empty id, it is the id the
  plugin puts on the FOCUS record. A plugin does not override it with a value it discovered.
- **Dry run**: The id has no effect on a dry run request. Dry run returns field mappings, not
  records.
- **Pagination**: The id is part of the query a page token continues. A caller sends the same id on
  every page of one query.
- **Tags**: The id is never read from or written to the request `tags` map.
- **Billing account name**: Not carried on this request. A FOCUS record that needs a name gets it
  from the plugin's own data or leaves it empty, as today.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The actual cost request MUST carry an optional billing account id, as a new field
  numbered 9. No existing field changes number, type, or meaning.
- **FR-002**: An empty billing account id MUST mean "the caller did not supply one." It MUST be
  valid, and a request that omits it MUST behave exactly as it did before this change.
- **FR-003**: When the billing account id is empty, a plugin MUST NOT invent one. It MAY leave the
  FOCUS record unset on its results and MUST still return the cost.
- **FR-004**: When the billing account id is non-empty and a plugin attaches a FOCUS record to a
  result, that record's billing account id MUST equal the request's id.
- **FR-005**: The billing account id MUST NOT act as a filter and MUST NOT change any cost value.
- **FR-006**: The billing account id MUST NOT be carried in the request `tags` map.
- **FR-007**: The field's documentation MUST state FR-002, FR-003, FR-004, and FR-005, including
  the words that plugins must not invent an id when the field is empty.
- **FR-008**: Field 10 of the actual cost request MUST stay unused, set aside for a later billing
  account name.
- **FR-009**: The generated Go and TypeScript request types MUST expose the billing account id.
- **FR-010**: The reference (mock) plugin MUST attach a FOCUS record built from the request's id
  when it is non-empty, and MUST NOT attach one when it is empty.
- **FR-011**: The SDK conformance suite MUST check FR-004 against a plugin under test, at the same
  conformance level as the existing actual cost correctness check (Standard). It MUST pass a plugin
  that attaches no FOCUS record, and a plugin that reports no data available. FR-003 stays normative
  text only, because a test cannot tell an invented id from a real one.
- **FR-012**: The TypeScript actual cost iterator MUST carry the billing account id on every page
  request.
- **FR-013**: Plugin-author documentation for actual cost MUST describe the field, the empty versus
  populated contract, and the rule that the id is not a filter.

### Key Entities

- **Actual cost request**: The host's query for historical cost of one resource over a time range.
  Gains an optional billing account id that the host supplies.
- **Actual cost result**: One cost data point. May carry a FOCUS record.
- **FOCUS cost record**: A FOCUS-format row. Its billing account id is required for the record to
  pass validation. With this change, the request's id is where that value comes from when the
  plugin does not know it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin with no billing account knowledge of its own can return a FOCUS record that
  passes validation for 100% of actual cost requests that carry a billing account id.
- **SC-002**: 100% of existing actual cost tests and conformance checks pass unchanged, which shows
  that callers who omit the id are unaffected.
- **SC-003**: Cost values for the same query are identical with and without a billing account id
  in 100% of reference-plugin test cases.
- **SC-004**: The conformance check tells a plugin that echoes the id apart from one that does not,
  with no false failures for plugins that attach no FOCUS record.
- **SC-005**: The protocol compatibility check against the current main branch reports no breaking
  change.

## Assumptions

- The caller is the authority for the billing account id on a request. A plugin passes it through
  and does not validate it against the provider's account format.
- One request covers one resource, so one billing account id per request is the right grain. A host
  that prices resources from several accounts sends a different id per request.
- The SDK does not trim, normalize, or format-check the id. FOCUS record validation stays the only
  place that judges a billing account id.
- Projected cost, cost estimate, and pricing spec requests stay out of scope. They do not attach
  FOCUS records today.
- The batch cost request is out of scope. It has no billing account field, so actual cost results
  from the batch path keep omitting FOCUS records.
- A billing account name field is out of scope. Field 10 is held for it.
- No new plugin capability flag is needed. The field is optional, and a plugin that ignores it is
  still compliant as long as it does not invent an id.
