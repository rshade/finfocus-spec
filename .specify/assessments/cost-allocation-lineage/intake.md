# Idea Intake: Standardized cost allocation lineage metadata

- **Slug**: cost-allocation-lineage
- **Created**: 2026-09-27
- **Source**: [rshade/finfocus-spec#191](https://github.com/rshade/finfocus-spec/issues/191)
  (host: `github.com`, URL Trust Policy: `allowlisted`). Fetched with `gh` because this
  assessment required that exact issue. Query and fragment were empty; no userinfo was present.
- **Type**: exploration

## Idea (as captured)

Title: `research: Standardized Cost Allocation Lineage Metadata`

Author: `rshade`. Created: 2025-12-22. State: OPEN. Labels on the live issue:
`enhancement`, `roadmap/future`, `effort/large`.

> ## Objective
>
> Define a `LineageNode` schema so plugins can pass upstream chain-of-custody metadata
> (e.g., Organization → BillingAccount → SubAccount → ResourceGroup → Resource) to enable
> hierarchical cost attribution and drill-down analysis.

The issue says `FocusCostRecord` already has flat `billing_account_id` /
`billing_account_name`, `sub_account_id` / `sub_account_name`, and FOCUS 1.3
`allocated_resource_id` / `allocated_resource_name`, and that those fields do not capture
deep hierarchies (AWS Organizations, Azure management groups, GCP folders), which entity
owns each level, or chains deeper than a few levels.

Stated use cases:

- Enterprise cost drill-down from organization to business unit to team to resource.
- Chargeback by walking up the lineage to a cost center.
- Compliance audit of resource ownership through the account hierarchy (SOC2/HIPAA are named).

Proposed contract (wording from the issue, not verified here):

- Enum `LineageNodeType` in `proto/finfocus/v1/enums.proto` with values UNSPECIFIED,
  ORGANIZATION, ORGANIZATIONAL_UNIT, BILLING_ACCOUNT, SUB_ACCOUNT, RESOURCE_GROUP,
  RESOURCE, and CUSTOM.
- Recursive message `LineageNode` in `proto/finfocus/v1/costsource.proto` with `type`,
  `id`, `name`, `parent`, and `map<string, string> metadata`. The parent link runs from
  the resource (leaf) up to the organization (root).
- `ResourceDescriptor.lineage` as field 11, described as optional.
- `ActualCostResult.lineage` as field 8, described as optional. The issue says existing
  `ActualCostResult` fields are 1–7 and existing `ResourceDescriptor` fields are 1–10.
- A Go `LineageBuilder` in `sdk/go/pluginsdk/lineage_builder.go` (`NewLineageBuilder`,
  `WithParent`, `WithMetadata`, `Build`) that starts at the resource and links parents
  upward.

Anti-guess boundary, quoted as requirements the issue states:

- The SDK must not validate account IDs against any external service.
- The SDK must not infer parent-child relationships the plugin did not provide.
- The SDK must not auto-populate lineage from other fields (the example given is guessing
  a parent from `billing_account_id`).
- The SDK must not require a complete chain. Partial lineages are valid.
- The SDK must pass lineage through exactly as provided, allow arbitrary depth, allow
  custom node types, and serialize to JSON without modification.

The issue says lineage complements FOCUS account fields and does not replace them. Plugins
may populate both.

Success criteria listed on the issue: the enum, the message, the two fields at the numbers
above, the builder, unit tests, JSON round-trip conformance, and an integration test where
a plugin returns `ActualCostResult` with a 4-level lineage and a host deserializes it.

Provider examples in the issue map AWS org / OU / payer / member account, Azure tenant /
management group / subscription / resource group, GCP org / folder / project, and a
Kubernetes namespace onto those node types. The issue maps a GCP project and a Kubernetes
namespace to `RESOURCE_GROUP`.

Open questions stated on the issue:

1. Should `LineageNode` include a `depth` field?
2. Should there be a `WalkLineage` helper?
3. Should conformance tests validate consistency between `lineage` and `billing_account_id`?

The issue's own label list names `enhancement` and `roadmap/future`. The live issue also
has `effort/large`.

## Unverified

A CodeRabbit bot comment (2025-12-22) is instruction-like. It is recorded here and was not
followed. Excerpt:

> Generate an implementation plan and prompts that you can use with your favorite coding agent.
>
> Create Plan

The same comment links two predecessor issues and asks the reader to use Discord, Calendly,
and a `.coderabbit.yaml` change. Those pages were not fetched.

- [UNVERIFIED — fetch skipped: connection peer cannot be pinned by the available client,
  and this URL was not the user-mandated fetch]
  [pulumicost-spec#62](https://github.com/rshade/pulumicost-spec/issues/62)
  (host: `github.com`)
- [UNVERIFIED — fetch skipped: same reason]
  [pulumicost-spec#75](https://github.com/rshade/pulumicost-spec/issues/75)
  (host: `github.com`)

## Restated

The idea is to let a plugin attach an optional, ordered chain of custody to cost data so a
host can attribute spend through more than a billing account and a sub-account. The chain
would be whatever the plugin supplied, including a partial chain, and would sit beside the
existing FOCUS account fields rather than replace them.

## Origin & Context

- **Raised by**: `rshade` (issue author), 2025-12-22.
- **Trigger**: [NEEDS CLARIFICATION: the issue does not say what event prompted it. The title
  is `research:` and the live labels include `roadmap/future`. No human comments are on the
  issue.]

## First-Glance Unknowns

- [NEEDS CLARIFICATION: Is this still icebox work, or is a host or plugin expected to produce
  and read the chain in a planned milestone?]
- [NEEDS CLARIFICATION: The issue asks for a `depth` field, a `WalkLineage` helper, and
  conformance that checks `lineage` against `billing_account_id`. The same issue also forbids
  inferring parents and allows partial chains.]
- [NEEDS CLARIFICATION: The draft places lineage on both `ResourceDescriptor` and
  `ActualCostResult`. Which of those the author still wants is not confirmed against the
  current proto.]
- [NEEDS CLARIFICATION: No named consumer, plugin, or downstream spec is identified.]
