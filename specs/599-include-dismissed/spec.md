# Feature Specification: Include Dismissed Recommendations

**Feature Branch**: `599-include-dismissed`

**Created**: 2026-10-04

**Status**: Implemented

**Input**: User description: "Add include_dismissed to GetRecommendationsRequest so a plugin that stores
dismissals can return those recommendations when a host asks. The default stays false.
excluded_recommendation_ids still omits those IDs. Tracked by rshade/finfocus#545."

## Clarifications

`/speckit-clarify` was not required. Issue #545 in rshade/finfocus names the field and the CLI flag.
The interaction with `excluded_recommendation_ids` is resolved in Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A host asks a plugin for dismissed recommendations (Priority: P1)

An operator runs an audit and wants the recommendations a plugin has itself dismissed or snoozed, not
only the copy the host saved locally. The host sets `include_dismissed` on `GetRecommendations`. A
plugin that stores dismissals returns those recommendations. A host that leaves the field unset, and
every existing client, still receives the list with plugin-side dismissals omitted.

**Why this priority**: This is the field rshade/finfocus#545 asks for. Without it, `--include-dismissed`
can only merge the host's local snapshots.

**Independent Test**: Configure a mock plugin with three recommendations and mark one dismissed. A
request with `include_dismissed` false omits it. The same request with `include_dismissed` true
returns it.

**Acceptance Scenarios**:

1. **Given** a plugin that has dismissed recommendation `rec-2`, **When** a host calls
   `GetRecommendations` without `include_dismissed`, **Then** the response does not contain `rec-2`.
2. **Given** the same plugin, **When** the host sets `include_dismissed` true, **Then** the response
   contains `rec-2` along with the recommendations that were never dismissed.
3. **Given** a plugin that does not store dismissals, **When** a host sets `include_dismissed`,
   **Then** the response is unchanged from a request that leaves the field unset.

---

### User Story 2 - An explicit exclusion still wins (Priority: P1)

A host keeps its own dismissal list and sends those IDs in `excluded_recommendation_ids`. That list
is how today's hosts hide recommendations from plugins that do not store dismissals. Turning on
`include_dismissed` must not bring an excluded ID back.

**Why this priority**: If the new field overrode the exclusion list, existing hosts that send both
by accident would show recommendations the operator dismissed.

**Independent Test**: Send `include_dismissed` true and `excluded_recommendation_ids: ["rec-2"]` to a
plugin that also has `rec-2` dismissed. The response does not contain `rec-2`.

**Acceptance Scenarios**:

1. **Given** `include_dismissed` is true and `rec-2` is in `excluded_recommendation_ids`, **When** the
   plugin handles the request, **Then** `rec-2` is absent whether or not the plugin had dismissed it.
2. **Given** `include_dismissed` is false and `rec-9` is only in `excluded_recommendation_ids`,
   **When** the plugin handles the request, **Then** `rec-9` is absent and the plugin's own dismissed
   IDs are also absent.

---

### User Story 3 - Plugin authors have one helper and a current description (Priority: P2)

A plugin author who stores dismissals needs one SDK function that applies both rules, and the
developer guide needs to describe the field next to `excluded_recommendation_ids`. Generated Go and
TypeScript clients expose the field. An older plugin that ignores unknown fields keeps working.

**Why this priority**: The field does nothing for real plugins until authors can implement it the
same way, and the wire change has to stay additive.

**Independent Test**: Call the SDK helper with a dismissed ID and an excluded ID under both values
of `include_dismissed`, and confirm the generated Go getter and the TypeScript field round-trip.

**Acceptance Scenarios**:

1. **Given** the published developer guide, **When** an author reads `GetRecommendations`, **Then**
   they find `include_dismissed`, the default, and the rule that exclusion IDs still apply.
2. **Given** a Go or TypeScript client generated from this spec, **When** a caller sets the field
   true, **Then** the request reports true and the field number is 8.
3. **Given** a plugin built before this field existed, **When** a new host sends it, **Then** the
   plugin still accepts the request and the default behavior is unchanged.

### Edge Cases

- An empty dismissal list and an empty exclusion list leave the recommendation slice unchanged.
- A recommendation with an empty ID is not removed when the dismissal list contains an empty string.
- `include_dismissed` does not change pagination: filtering happens before the page is cut.
- Snoozed recommendations are treated like dismissed ones. The field does not distinguish them.
- The response has no new status field. A returned recommendation is not labeled dismissed on the wire.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `GetRecommendationsRequest` MUST add `bool include_dismissed = 8`. The default is false.
- **FR-002**: A plugin that stores dismissed or snoozed recommendation IDs MUST omit those IDs when
  `include_dismissed` is false, and MUST include them when `include_dismissed` is true.
- **FR-003**: `excluded_recommendation_ids` MUST still omit its IDs when `include_dismissed` is true.
- **FR-004**: A plugin that does not store dismissal state MUST ignore `include_dismissed`.
- **FR-005**: The Go SDK MUST provide one helper that applies FR-002 and FR-003. The mock plugin
  MUST apply the same rules, including before pagination.
- **FR-006**: Generated Go and TypeScript bindings MUST expose the field. Authors MUST NOT hand-edit
  generated code.
- **FR-007**: The plugin developer guide MUST document the field, the default, and FR-003.
- **FR-008**: The change MUST be wire-compatible. `buf breaking` against `main` MUST be clean.
  Existing field numbers MUST NOT change.

### Key Entities

- **GetRecommendationsRequest.include_dismissed**: Host instruction to a plugin that stores
  dismissals. False omits them. True includes them. It does not cancel `excluded_recommendation_ids`.
- **Dismissed ID**: An identifier the plugin itself decided to hide. The spec does not choose where
  the plugin stores it.
- **Excluded ID**: An identifier the host listed on this request. It is always omitted.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A mock with one dismissed recommendation out of three returns 2 when the field is false
  and 3 when the field is true.
- **SC-002**: The same mock returns 2 when the field is true and that recommendation is also excluded.
- **SC-003**: `buf breaking` against `main` reports no breaking changes, and `buf lint` is clean.
- **SC-004**: A Go test reads field number 8, and a TypeScript test round-trips `includeDismissed`.

## Assumptions

- Field number 8 is free. `usage_profile` is 7. No reserved number sits between them.
- Snoozed and dismissed recommendations use the same plugin-side omit list. The request does not
  carry a separate snooze flag.
- `excluded_recommendation_ids` wins over `include_dismissed`. Hosts that want a plugin-dismissed
  recommendation to appear must leave its ID off the exclusion list.
- Plugins outside this repository are not edited here. A plugin with no dismissal store already
  meets FR-004. A plugin that stores dismissals uses the helper when it upgrades.
- The host's local dismissal merge stays in rshade/finfocus. This spec does not add a dismissed
  status to `Recommendation`.
- `SpecVersion` stays `v0.7.4` until release-please tags the release. The field is additive on the
  current version.
