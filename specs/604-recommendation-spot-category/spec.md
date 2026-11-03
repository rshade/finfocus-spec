# Feature Specification: Spot Pricing Category and Interruption Risk on Recommendation

**Feature Branch**: `604-recommendation-spot-category`

**Created**: 2026-11-03

**Status**: Implemented

**Input**: Issue #663, "feat(proto): spot pricing category and interruption risk on Recommendation"
(Spot Market Advisor, core epic rshade/finfocus#367, core issue rshade/finfocus#1839).

## Clarifications

### Session 2026-11-03

- Q: Should the scorer request validator also reject an invalid spot category and risk pair on the
  recommendations it receives? → A: No. Only the recommendation validator enforces the rule; scorers
  receive the fields unvalidated and handle their own data.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Say a recommendation targets spot capacity (Priority: P1)

A plugin recommends moving an on-demand resource to spot capacity. It marks the recommendation with a
typed pricing category of DYNAMIC and, when it knows it, an interruption risk between 0.0 and 1.0.
A host reads those two typed values and shows the same `SPOT · risk 0.12` annotation it already shows
on cost rows. The host does not parse free-form metadata keys and carries no provider knowledge.

**Why this priority**: Without it the only option is a free-form metadata convention. That puts spot
knowledge in the host and cannot tell "risk 0" from "no risk reported". Everything downstream
(host display, plugin emission) depends on this contract.

**Independent Test**: Build a recommendation with category DYNAMIC and risk 0.12, validate it, send it
over the in-memory test harness, and read both values back unchanged.

**Acceptance Scenarios**:

1. **Given** a recommendation with category DYNAMIC and risk 0.12, **When** it is validated, **Then**
   validation succeeds and a host reads both values back unchanged.
2. **Given** a recommendation with category DYNAMIC and risk left at 0.0, **When** it is validated,
   **Then** validation succeeds and the host reads the risk as unknown (matching the cost responses).
3. **Given** a recommendation whose savings are in the existing impact data, **When** the two new
   values are added, **Then** savings still come only from the impact data and are not duplicated.

---

### User Story 2 - Reject contradictory or malformed risk (Priority: P1)

A plugin author who sets a non-zero risk on a recommendation that is not DYNAMIC, or sets a risk that
is not a finite number in [0, 1], gets a clear validation error naming the problem. The rules match the
cost responses exactly, so one mental model covers both.

**Why this priority**: The contract is only trustworthy if hosts can rely on it. Invalid combinations
must fail at the plugin, not surface as a wrong label in a host.

**Independent Test**: Validate recommendations across every category and boundary risk value and check
which are accepted and which return the matching error.

**Acceptance Scenarios**:

1. **Given** a non-zero risk with category STANDARD, COMMITTED, or UNSPECIFIED, **When** validated,
   **Then** validation fails with the invalid-category error.
2. **Given** a risk that is NaN, infinite, negative, or above 1.0 (beyond the tolerance), **When**
   validated, **Then** validation fails with the invalid-value error.
3. **Given** the default pair (category UNSPECIFIED, risk 0.0), **When** validated, **Then** validation
   succeeds, so plugins that predate the fields keep passing.
4. **Given** a risk within the 1e-9 tolerance of zero or of 1.0, **When** validated, **Then** it is
   treated as 0.0 or 1.0 exactly as on the cost responses.

---

### User Story 3 - Verify plugins through conformance (Priority: P2)

A plugin author runs the conformance suite and gets a case that checks the new rules on the
recommendations the plugin returns. A test helper lets plugin tests assert the same rules.

**Why this priority**: Conformance is how the ecosystem finds plugins that emit bad data, but the
contract is usable before the case exists.

**Independent Test**: Run the conformance suite against a mock plugin that returns valid spot
recommendations (passes) and one that returns an invalid pair (fails with the specific rule).

**Acceptance Scenarios**:

1. **Given** a plugin returning valid recommendations, **When** conformance runs, **Then** the new
   case passes, including for plugins that never set the fields.
2. **Given** a plugin returning a non-zero risk without DYNAMIC, **When** conformance runs, **Then**
   the case fails and names the rule.

---

### User Story 4 - Adopt the change with clear guidance (Priority: P2)

Maintainers of hosts and plugins find the new fields in the plugin guide and proto reference, and an
upgrade note for the new spec version tells host authors what to read and plugin authors what to emit.
The release is a minor version, and old plugins and old hosts keep working unchanged.

**Why this priority**: The core and plugin repositories land their work in a fixed order after this
release, and the host's upgrade process requires a note for every release.

**Independent Test**: Follow the upgrade note to read the fields in a sample host and to set them in a
sample plugin, with no other source needed.

**Acceptance Scenarios**:

1. **Given** the updated documentation, **When** a plugin author looks up recommendations, **Then**
   they find both fields, the DYNAMIC requirement, the zero-means-unknown rule, and the guidance that
   the action type stays MODIFY or MIGRATE.
2. **Given** an older host receiving these fields, **When** it decodes the message, **Then** it ignores
   them with no error.

---

### Edge Cases

- Category DYNAMIC with risk 0.0: valid. Risk is unknown, or truly zero (unusual).
- Category STANDARD or COMMITTED with risk 0.0: valid.
- Risk of 1.0 exactly, and 1.0 plus tolerance: valid; above that: invalid.
- Risk slightly below zero but within tolerance: treated as zero; beyond tolerance: invalid.
- A recommendation with category DYNAMIC whose action type is neither MODIFY nor MIGRATE: valid. The
  category is the signal, and the action type is not constrained by it.
- A nil recommendation: keeps the existing nil error and never reaches the new rules.
- Batches and summaries: every recommendation in a list is validated on its own, and the existing
  per-item error behaviour is unchanged.
- Category set on a recommendation that is not about pricing (for example, a rightsizing change on
  STANDARD capacity): valid; the category describes the recommended target state.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A recommendation MUST be able to carry an optional pricing category describing the
  pricing model of the recommended target state, using the same category set as the cost responses.
- **FR-002**: A recommendation MUST be able to carry an optional spot interruption risk, a number from
  0.0 to 1.0, for the recommended spot target.
- **FR-003**: An unset category (UNSPECIFIED) MUST mean the recommendation says nothing about pricing
  model, and a risk of 0.0 MUST mean unknown, identical to the cost responses.
- **FR-004**: Validation MUST reject a risk above the 1e-9 tolerance when the category is not DYNAMIC.
- **FR-005**: Validation MUST reject a risk that is NaN, infinite, below zero beyond the tolerance, or
  above 1.0 beyond the tolerance, and MUST accept 0.0 and 1.0 with the same tolerance as the cost
  responses.
- **FR-006**: Validation MUST accept the default pair (UNSPECIFIED, 0.0) and every combination with
  risk 0.0, so existing plugins stay valid.
- **FR-007**: Each failure mode MUST return its own named error that callers can match, in the style of
  the existing on-demand-equivalent errors, and the message MUST identify the offending recommendation
  field.
- **FR-008**: The change MUST NOT add a recommendation action type; savings MUST stay in the existing
  impact data; the new values MUST NOT be summed with or compared against impact figures.
- **FR-009**: The conformance suite MUST include a case that checks the new rules on returned
  recommendations and passes for plugins that never set the fields.
- **FR-010**: The mock plugin and test helper MUST let tests produce valid and invalid pairs without
  hand-building messages.
- **FR-011**: The plugin guide and proto reference MUST document both fields, and an upgrade note MUST
  exist for the new spec version.
- **FR-012**: The change MUST ship in a minor release. The spec version constant advances through the
  release automation, not by hand, and the upgrade note names the release that carries the fields.
- **FR-013**: The change MUST be wire-compatible: existing fields keep their numbers and meanings,
  fields 20 and 21 are the only additions, and the breaking-change check passes.
- **FR-014**: The TypeScript client MUST expose the new fields through its regenerated bindings, with
  any builder or helper parity the repository already keeps for recommendations.
- **FR-015**: Validation of a recommendation with neither new field set MUST not allocate or slow down
  measurably versus today.
- **FR-016**: The scorer request validator MUST NOT apply the new rules. Recommendations embedded in a
  scoring request pass through with both fields unchecked, and a scorer decides how to treat them.

### Key Entities

- **Recommendation**: A plugin's suggested change to a resource. Gains a pricing category and a spot
  interruption risk describing the recommended target state.
- **Pricing category**: The existing FOCUS pricing model set (standard, dynamic, committed,
  unspecified). DYNAMIC is the spot signal.
- **Spot interruption risk**: A probability from 0.0 to 1.0 of the recommended spot target being
  interrupted; 0.0 is unknown.
- **Recommendation impact**: The existing savings and cost figures; unchanged.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A host can show a spot label for a recommendation from the two typed values with no
  free-form key parsing; a round trip returns both values unchanged in 100% of tested cases.
- **SC-002**: Every rule in FR-004 to FR-006 has at least one passing test, including each boundary
  value (0, tolerance edges, 1.0, NaN, infinities, negatives).
- **SC-003**: All existing recommendation tests and conformance runs pass without modification, and
  plugins that never set the fields pass the new conformance case.
- **SC-004**: The breaking-change check, schema lint, and the repository's full lint and test runs
  all pass.
- **SC-005**: A host or plugin author can adopt the fields from the upgrade note and guide alone,
  with no questions needing the maintainer.
- **SC-006**: The recommendation validation benchmark with the fields unset shows no measurable
  regression and no new allocations.

## Assumptions

- Field numbers 20 and 21 are free on the recommendation message and are used exactly as the issue
  states (current highest is 19).
- The tolerance and zero-means-unknown rules are copied from the projected-cost field so both read
  identically; no new tolerance is introduced.
- The existing sentinel naming style and the existing shared check for the cost responses are reused
  where practical, so one rule serves both. The final choice is a planning decision.
- The category applies to the recommended target state, not the current state; the current state
  stays in the impact data.
- Scorer-side validation of the new fields is out of scope (FR-016); the scoring request and its
  validator keep their current behaviour.
- Host display, core dependency bump, and plugin emission happen in other repositories after this
  release and are out of scope.
- Hosts never reorder recommendations by risk; the field only annotates. This is guidance in the docs,
  not a validation rule.
- No new recommendation action type is added, and no provider-specific names enter the contract.
- The next spec version is chosen by the release automation; the upgrade note is written against the
  release that carries the fields. The host's own per-release reference note lives in the core
  repository and is out of scope here.
