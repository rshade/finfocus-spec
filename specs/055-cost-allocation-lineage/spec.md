# Feature Specification: Standardized Cost Allocation Lineage Metadata

**Feature Branch**: `055-cost-allocation-lineage`

**Created**: 2026-09-28

**Status**: Draft

**Input**: User description: "GitHub issue #191, assessed in
`.specify/assessments/cost-allocation-lineage/` (verdict: needs-clarification, blocking
questions since resolved by the maintainer). Attach an optional, ordered chain-of-custody
lineage to cost data as pure pass-through metadata: plugins report it, hosts store and walk
it, nothing is inferred or validated."

## Overview

The contract today is flat. A cost record names a billing account, a sub-account, a resource,
and unordered tags. A plugin that knows the full ancestry of a resource (organization,
organizational unit, billing account, sub-account, resource group, resource) has no way to say
so. Two hosts that need that ancestry each rebuild it themselves, and the spec cannot say what
the plugin actually reported.

This slice adds an optional lineage chain to cost data. The chain starts at the resource and
links upward, one node per level, toward the organization. Each node carries a classification,
a provider-assigned id, a display name, and free-form attributes. A plugin attaches the chain
it already knows; a host stores it and walks it in order. A missing chain is normal. A chain
that stops early is normal. The chain sits beside the existing account fields and never
replaces them.

Lineage is pass-through metadata. Nobody in this contract infers a parent that was not
reported, checks an id against a cloud provider, or rejects a cost because the chain omits or
disagrees with the billing account fields. Those flat account fields stay canonical for
two-level attribution.

**Relationship to constitution principle III ("The Spec Consumes, It Does Not Calculate")**:
a lineage chain is reported data, not computation. This slice moves a hierarchy the plugin
already had; it does not derive one. Allocation math stays with `AllocatorService`, which owns
the words "cost allocation" for dividing costs across workloads — this slice carries ancestry,
not splits.

**Origin**: GitHub issue #191 (`research: Standardized Cost Allocation Lineage Metadata`),
assessed in `.specify/assessments/cost-allocation-lineage/`. The assessment verdict was
needs-clarification; its blocking questions are resolved by the maintainer decisions recorded
under Clarifications.

## Clarifications

### Session 2026-09-28

- Q: Which messages carry the chain — the actual-cost result only, or the resource descriptor
  too? → A: Both. The resource descriptor gains `lineage` at field 11 (still free, matching
  the issue sketch). The actual-cost result gains `lineage` at field 9, not the field 8 the
  issue sketch named: field 8 has carried the `expires_at` caching hint since spec 045, and
  reusing it would break the wire contract. Projections, estimates, and batch wrappers do not
  get lineage.
- Q: Is a chain that omits or disagrees with `billing_account_id` / `sub_account_id` valid?
  → A: Yes, always. Lineage is pure pass-through. There is no consistency check against the
  FOCUS account fields, no warning, and no conformance failure on disagreement. The FOCUS
  account fields remain canonical. Partial chains are valid. This resolves the assessment's
  third blocking question and rejects the issue's open question 3 as a hard check.
- Q: Is the Go SDK helper in scope? → A: Yes. A fluent builder ships in the plugin SDK
  (`NewLineageBuilder` / `WithParent` / `WithMetadata` / `Build`). It only assembles
  caller-supplied nodes: it does not infer parents, does not validate ids, and accepts partial
  chains of any depth the caller builds. The issue's sketch walked the chain on every call and
  had ambiguous metadata placement; the shipped builder tracks the leaf and the current top so
  metadata lands on the most recently added node.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin Author Reports the Hierarchy With a Cost (Priority: P1)

A plugin author knows the full ancestry of a resource — for example an organization, a payer
account, a member account, and the resource itself. When the plugin returns a cost for that
resource, it attaches the chain it knows. Each level is classified (organization,
organizational unit, billing account, sub-account, resource group, resource, or a custom
level), named, and given its provider id, with optional attributes such as an owner email or
cost center on any level. The plugin may also attach the same kind of chain when describing a
resource in a request.

**Why this priority**: If plugins cannot report the chain, nothing else in the slice exists.
Every other story consumes what this one produces.

**Independent Test**: Build a four-level chain (resource, sub-account, billing account,
organization) with per-level attributes, attach it to a cost result, serialize the result, and
read it back. The chain walks in the exact order built, with every id, name, classification,
and attribute intact.

**Acceptance Scenarios**:

1. **Given** a plugin that knows a four-level ancestry, **When** it returns a cost with that
   lineage attached, **Then** a reader walking the chain from the resource reaches the
   sub-account, the billing account, and the organization in that order, each with its
   reported id, name, and classification.
2. **Given** a plugin that knows only the resource and its sub-account, **When** it returns a
   cost with that partial chain, **Then** the chain is accepted as-is and ends where the
   plugin stopped.
3. **Given** a plugin with a level that fits no standard classification, **When** it reports
   that level as a custom node, **Then** the node round-trips with its id, name, and
   attributes like any other level.
4. **Given** a plugin whose chain names a different billing account than the flat account
   fields, or names none, **When** the cost is returned, **Then** the cost and its chain are
   still accepted unchanged; disagreement is not an error.
5. **Given** a resource description that carries a lineage chain, **When** the description
   crosses the contract, **Then** the chain survives with the same fidelity as on a cost
   result.

---

### User Story 2 - Host Reads the Chain Back in Order (Priority: P2)

A host author receives costs with lineage attached. The host walks the chain from the resource
upward, grouping or drilling by any reported level — per organizational unit, per billing
account, per custom cost-center level — using exactly the order and values the plugin
supplied. The host never needs to guess: a missing parent means the chain ended, not that a
parent should be invented.

**Why this priority**: Reading is why reporting exists, but a host can only read once a plugin
can write. The host-side rule set (no inference, no validation, partial is valid) is what
makes the data safe to consume.

**Independent Test**: Hand a host a serialized cost carrying a four-level chain and a cost
carrying a partial chain. The host walks both in order, groups by a middle level, and treats
the end of the partial chain as the top of what was reported.

**Acceptance Scenarios**:

1. **Given** costs carrying lineage from several resources under two organizational units,
   **When** the host groups by the organizational-unit level, **Then** each cost lands under
   the unit its chain reported, using the chain's own ids.
2. **Given** a cost whose chain stops at the sub-account, **When** the host walks it, **Then**
   the host sees exactly two levels and treats the chain as complete at that point.
3. **Given** a cost whose lineage disagrees with its flat billing account field, **When** the
   host processes it, **Then** both are delivered as reported; the flat account fields remain
   the canonical two-level attribution.
4. **Given** a chain carrying attributes on several levels, **When** the host reads it,
   **Then** each attribute is attached to the level the plugin put it on, not shifted to a
   neighbor.

---

### User Story 3 - Hosts That Ignore Lineage Are Unaffected (Priority: P3)

A host that does not care about hierarchy keeps working exactly as before. Costs without a
chain are unchanged. Costs with a chain are still valid for a host that never looks at it, and
plugins that never report one see no new requirement.

**Why this priority**: Backward compatibility is a property the slice must not break rather
than new value, but it gates adoption: no existing plugin or host may be forced to change.

**Independent Test**: Exchange costs with no lineage between an unmodified plugin and an
unmodified host, and deliver a lineage-carrying cost to a host that ignores the field. Both
flows behave exactly as they did before the slice.

**Acceptance Scenarios**:

1. **Given** a cost with no lineage, **When** it is serialized and read, **Then** every
   previously existing field is byte-for-byte what it would have been before this slice.
2. **Given** a cost with lineage read by a host built before this slice, **When** the host
   processes it, **Then** the cost's pre-existing fields are read normally and the unknown
   field is ignored without error.
3. **Given** a plugin that reports no lineage, **When** it is conformance-tested, **Then** no
   new requirement or failure appears.

---

### Edge Cases

- A chain with only the resource node is valid. So is a chain that skips classifications
  (resource straight to organization) or repeats one.
- A node of the custom classification carries its meaning in its name and attributes; nothing
  distinguishes two custom levels except what the plugin reported.
- A GCP project or Kubernetes namespace may be reported as a resource-group level by one
  plugin and mapped to the flat sub-account field by another. Both are accepted; the slice
  does not arbitrate classification.
- Very deep chains are the caller's choice to build, but nested encoding has practical
  recursion limits, so implementations keep chains to a reasonable organizational depth
  (typically under ten levels) rather than promising unlimited depth.
- Actual-cost pages can carry up to a thousand results; a chain on every result multiplies
  payload by depth and attribute count. Reporters keep chains and attributes to what
  consumers need.
- An empty id or name on a node is tolerated as reported data; the classification unset value
  marks an incomplete implementation, not a new level kind.
- Lineage is plugin-asserted metadata. It is not audit evidence by itself, and nothing in this
  slice makes it one.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The contract MUST define a lineage node with a classification, a provider-assigned
  id, a display name, an optional link to the parent level above it, and optional free-form
  attributes. Nodes form a singly-linked chain from the resource (leaf) up toward the
  organization (root); a node with no parent link is the top of the reported chain.
- **FR-002**: The contract MUST define a node classification covering: organization,
  organizational unit, billing account, sub-account, resource group, resource, and a
  plugin-defined custom level, plus an unset default.
- **FR-003**: An actual-cost result MUST be able to carry one optional lineage chain. The
  existing caching-hint field on that message MUST NOT be disturbed, and the lineage field
  MUST use a field number that has never carried another meaning.
- **FR-004**: A resource descriptor MUST be able to carry one optional lineage chain at the
  field number the issue reserved (11, still free).
- **FR-005**: Lineage MUST NOT be added to projected-cost responses, estimate responses, or
  batch wrappers in this slice.
- **FR-006**: Hosts and SDK code MUST NOT infer or synthesize a parent a plugin did not
  report, MUST NOT auto-populate lineage from other fields (such as deriving a parent from
  the billing account id), and MUST NOT validate node ids against provider APIs.
- **FR-007**: A chain that omits the flat account fields' values, or disagrees with them,
  MUST be accepted without error or warning. The flat billing-account and sub-account fields
  remain the canonical two-level attribution. No conformance check tests agreement.
- **FR-008**: Partial chains MUST be valid: a chain may stop at any level and may skip or
  repeat classifications. A complete chain MUST NOT be required anywhere.
- **FR-009**: Chains of arbitrary, plugin-defined depth MUST be expressible, including custom
  node classifications. Implementations SHOULD document that nested encoding imposes practical
  recursion limits rather than promising unlimited depth.
- **FR-010**: A lineage chain MUST serialize to JSON and back without modification: order,
  ids, names, classifications, and per-level attributes all preserved.
- **FR-011**: Costs and descriptors without lineage MUST be byte-identical to what they were
  before this slice, and a reader built before this slice MUST ignore the new field without
  error.
- **FR-012**: The plugin SDK MUST provide a fluent builder that assembles a chain from
  caller-supplied values only, starting at the resource and linking parents upward. The
  builder MUST let callers attach attributes to the most recently added node, MUST NOT infer
  parents or validate ids, and MUST return the leaf node so the chain walks upward through
  parent links.
- **FR-013**: Generated bindings for every language SDK in this repository MUST be regenerated
  from the updated contract in the same change.
- **FR-014**: This slice MUST NOT add an RPC, a plugin capability, allocation math, or any
  consistency rule between lineage and other fields.

### Key Entities

- **Lineage chain**: The ordered ancestry a plugin reports for one resource, walked from the
  resource up toward the organization. Optional everywhere it may appear.
- **Lineage node**: One level of the chain. A classification, an id, a display name, a parent
  link, and free-form attributes. The node with no parent link is the top of what was
  reported, whether or not it is an organization.
- **Node classification**: The kind of level a node represents: organization, organizational
  unit, billing account, sub-account, resource group, resource, or custom. Classifications
  describe what the plugin believed; they are not checked against provider APIs.
- **Flat account fields**: The existing billing-account and sub-account identity on cost
  records. Canonical for two-level attribution; never replaced, checked against, or derived
  from lineage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A four-level chain (resource, sub-account, billing account, organization) with
  attributes on multiple levels survives a full serialize/read round trip on a cost result
  with 100% of its levels, order, ids, names, classifications, and per-level attributes
  intact.
- **SC-002**: 100% of cost results and resource descriptors that carry no lineage are
  unchanged from before this slice; a reader that predates the slice processes a
  lineage-carrying cost with zero errors.
- **SC-003**: A chain that omits or contradicts the flat billing-account and sub-account
  values is accepted in 100% of cases, with both the chain and the flat fields delivered
  exactly as reported.
- **SC-004**: A plugin author can assemble and attach a chain using the SDK helper without
  hand-building linked nodes, and attributes land on the intended level in 100% of placement
  cases (resource level, middle level, top level).
- **SC-005**: Every language binding in the repository exposes the new chain types after a
  single regeneration command, with no hand-edited generated code.

## Assumptions

- The consumer named by the assessment does not exist yet; the maintainer accepted that risk
  when resolving the blocking questions. The slice is specified so a future host reads what a
  future plugin wrote, not for a named milestone.
- Plugin authors can classify their own levels. Where a provider concept overlaps the flat
  account fields (GCP project, Kubernetes namespace), the plugin's classification is reported
  as-is and the flat fields stay canonical.
- Display name plus free-form attributes are enough to tell two custom levels apart; no
  separate level-name field is needed.
- A nested parent link is acceptable encoding; a stored depth counter and a walk helper are
  conveniences the contract does not need. Position in the chain is the depth.
- Ten levels comfortably covers real organizational hierarchies; the tests exercise that
  depth without making it a contractual ceiling.

## Out of Scope

- Host drill-down, chargeback UI, and the CFO click-path from the issue. Those are core
  application experiences, not this repository's contract.
- Allocation math and any change to `AllocatorService`, which owns cost splitting.
- Replacing, deprecating, or rewriting the flat FOCUS account, resource, or tag fields.
- Requiring a complete chain, a stored depth field, or a walk helper as part of the contract.
- Validating node ids against provider APIs, or treating lineage as compliance audit evidence.
- Lineage on projected-cost responses, estimate responses, batch wrappers, or the FOCUS record
  itself.
- A new RPC or a new plugin capability.
- Reusing the actual-cost field number that already carries the caching hint.
- A promise of unlimited nesting depth.
