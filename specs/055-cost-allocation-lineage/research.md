# Research: Standardized Cost Allocation Lineage Metadata

This feature came out of the assessment at `.specify/assessments/cost-allocation-lineage/`
(intake, research, problem, concept, decision; verdict: needs-clarification on 2026-09-27).
That directory holds the cited evidence: field-number tables, prior art, payload constraints,
and evidence against the idea. This page records only the decisions, each traceable to the
assessment or to the maintainer's resolution of its blocking questions.

## Decision: Attach lineage on both the descriptor and the actual-cost result

- **Decision**: `ResourceDescriptor.lineage = 11` and `ActualCostResult.lineage = 9`. Nothing
  on `GetProjectedCostResponse`, `EstimateCostResponse`, batch wrappers, or `FocusCostRecord`.
- **Rationale**: Maintainer resolution of the assessment's second blocking question. The
  assessment's concept recommended result-only (Option B) because the issue's objective is
  plugins passing metadata upstream; the maintainer chose both attachment points, matching the
  issue sketch. Field 11 was verified free by the assessment's field table. The issue's field
  8 on the result is stale: `expires_at` took it in spec 045, and constitution VI forbids
  reusing it.
- **Alternatives considered**: Result-only (rejected by the maintainer). Both plus projections
  and estimates (rejected: Option C appetite; no consumer for estimated lineage).

## Decision: Pure pass-through, no consistency check

- **Decision**: A chain that omits or disagrees with `billing_account_id` / `sub_account_id`
  is valid. No warning, no conformance failure. FOCUS account fields stay canonical. Partial
  chains are valid. No parent inference, no id validation against provider APIs.
- **Rationale**: Maintainer resolution of the assessment's third blocking question. The
  issue's own anti-guess boundary says partial chains are valid and the SDK must not invent
  relationships; its open question 3 (fail conformance on disagreement) contradicts that and
  is rejected as a hard check.
- **Alternatives considered**: Warn on disagreement (rejected: a warning is still a judgment
  the contract has no basis to make). Hard consistency check (rejected: conflicts with partial
  chains and the anti-guess boundary).

## Decision: Singly-linked nested nodes, no depth field, no walk helper

- **Decision**: `LineageNode` carries `type`, `id`, `name`, `parent`, and
  `map<string, string> metadata`. The parent link runs leaf to root. No stored `depth`; no
  `WalkLineage` in the contract.
- **Rationale**: Position in the chain is the depth; a counter can lie and must be maintained.
  The assessment lists both as rabbit holes the recommended option does not need. A nested
  encoding cannot honestly promise unlimited depth, so the comments bound it instead of
  adding machinery.
- **Alternatives considered**: Flat ordered list of nodes (rejected: order becomes a convention
  instead of structure; nested links make the walk unambiguous). Stored depth and walk helper
  (rejected: contract surface without a consumer).

## Decision: Ship the Go builder, with leaf-plus-top tracking

- **Decision**: `pluginsdk.LineageBuilder` with `NewLineageBuilder(resourceID, resourceName)`,
  `WithParent(nodeType, id, name)`, `WithMetadata(key, value)`, `Build()`. The builder keeps a
  pointer to the leaf (returned by `Build`) and to the current top (target of `WithMetadata`
  and attachment point for `WithParent`).
- **Rationale**: Maintainer resolution: the SDK helper is in scope. The issue's sketch tracked
  only one pointer and walked the chain on every call — O(n²) — and its metadata semantics
  were ambiguous (the assessment notes metadata lands on the current root and a middle node
  cannot be tagged after a higher parent is linked). Tracking both pointers makes each call
  O(1) and gives metadata an unambiguous target: the most recently added node.
- **Alternatives considered**: No builder (rejected by the maintainer). The sketch as written
  (rejected: quadratic walks and ambiguous metadata placement). `WithMetadata` taking a level
  index or id (rejected: invites callers to address levels the pass-through model says the
  builder should not interpret).

## Decision: Comments carry the anti-guess boundary

- **Decision**: The pass-through rules (no inference, no provider-API validation, partial
  chains valid, disagreement not an error, recursion-limit caveat) live in the proto comments
  on `LineageNode`, both lineage fields, and the enum, plus the builder's godoc.
- **Rationale**: Constitution VII: the proto is the documentation of record. A host that reads
  only the field comment gets the whole rule set.
- **Alternatives considered**: A separate doc page (rejected for this slice: would duplicate
  the field comments and drift; no existing lineage doc exists to update).

## Decision: No conformance fixture, no new capability

- **Decision**: Tests are unit and round-trip tests in `pluginsdk`. No conformance-harness
  fixture and no `PLUGIN_CAPABILITY_*` entry.
- **Rationale**: Lineage is optional pass-through payload; there is no behavior for a
  capability to advertise and nothing for conformance to check beyond what the round-trip
  tests already lock (the consistency check that conformance might have run was explicitly
  rejected).
- **Alternatives considered**: A conformance case asserting round-trip fidelity (rejected for
  this slice: the protojson tests in `pluginsdk` cover it without growing the harness).
