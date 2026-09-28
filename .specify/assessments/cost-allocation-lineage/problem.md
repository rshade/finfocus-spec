# Problem Definition: Ordered cost ancestry beyond the account pair

- **Slug**: cost-allocation-lineage
- **Created**: 2026-09-27
- **Inputs used**: intake.md | research.md

## Problem Statement

Plugin authors who already know how a resource sits inside a provider hierarchy, and host
authors who need to attribute spend through that hierarchy, cannot exchange an ordered chain
of custody in the FinFocus contract. Today's contract stops at a billing account, a
sub-account, a resource id, and unordered tags. Anything above the sub-account (organization,
organizational unit, folder, management group) or between those levels has to be reconstructed
outside the contract, where the plugin and the host can disagree, or it is dropped. The pain
shows up when someone needs parentage, not when they only need a cost-center tag.

## Affected Users & Stakeholders

- **Users**: Plugin authors who observe provider hierarchy while reporting cost. They cannot
  hand the host an ordered chain without overloading tags or another free-form map.
  — [source: research.md prior art on flat FOCUS columns and `ResourceDescriptor.tags`]
- **Users**: Host authors who would group or audit spend by that chain. The drill-down and
  chargeback stories in issue 191 describe them. No host in this repo is blocked on it today.
  — [source: intake.md use cases; research.md demand section]
- **Users**: FinOps practitioners who want chargeback or an ownership trace. Named in the
  issue only. No interview or ticket in this repo shows the workaround failing.
  — [source: issue 191 via intake.md] [NEEDS CLARIFICATION: no observed practitioner]
- **Stakeholders**: `rshade`, as issue author and spec maintainer, decides whether icebox
  research becomes a contract change. Decision power is the maintainer's; no other sponsor is
  named. — [source: intake.md origin]

## Goals

- A host can recover the ordered parentage a plugin actually reported for a cost, including a
  chain that stops before the organization.
- No parent is invented when the plugin did not supply it, and account identifiers are not
  checked against a cloud API as part of carrying the chain.
- Existing two-level account identity and tag-based attribution keep working unchanged when
  no chain is present.
- Deeper ancestry stays optional context beside the canonical account fields, not a second
  source of truth that plugins are required to keep in lockstep.

## Non-Goals

- Building the drill-down, chargeback, or audit user interface. That belongs to a host.
- Computing allocated amounts, splitting a bill across workloads, or checking that money is
  conserved. That is a different problem, already covered by the allocator contract.
- Replacing billing-account, sub-account, resource-id, or tag fields.
- Requiring a complete chain from resource to organization.
- Proving SOC2 or HIPAA controls. The issue names them and does not state a control.
- Validating that provider ids exist, or guessing a parent from an account field.
- A new remote procedure. Nothing in the intake says the host needs a new call, only a place
  to carry metadata it does not have today.

## Success Metrics

- A host can read back a four-level chain a plugin reported on an actual cost, in the same
  order, with the same ids, names, and per-level notes. Baseline: not representable. Only the
  account pair, resource id, and unordered tags exist. — [source: research.md field inventory]
  (qualitative until a fixture exists)
- Responses that omit the chain stay valid and identical in meaning to today's responses.
  Baseline: every current plugin omits it, because the contract has nowhere to put it.
  — [source: research.md "no lineage field"] (measurable later by existing conformance staying
  green)
- When a reported chain does not mention the billing account, the cost is still accepted.
  Baseline: there is no chain to disagree with. The issue both allows partial chains and asks
  whether disagreement should fail a test. — [source: intake.md anti-guess boundary vs open
  question 3] (measurable as a negative test once a contract exists)
- [NEEDS CLARIFICATION: no usage baseline. There is no count of plugins, hosts, or cost rows
  that would populate a chain.]

## Cost of Inaction

Two-level FOCUS attribution and tag-based cost centers keep working. Spec 052 can still divide
priced infrastructure across workloads. No in-repo feature is stuck. Deeper parentage stays
out of band: tags, a projected-cost metadata map, or a join the host does on its own. Two
implementations can then disagree about who owns a resource, and this spec cannot say which
chain was reported. The issue has waited on `roadmap/future` since 2025-12-22 without a named
consumer, so waiting longer does not unblock a current milestone. The cost of waiting is a
continued gap, not a broken contract.

## Open Questions

- [NEEDS CLARIFICATION: Is a host or plugin committed to produce and read this chain in a
  planned milestone, or does the icebox label mean the gap stays accepted?]
- [NEEDS CLARIFICATION: For which cost interactions is missing parentage actually painful:
  actual cost only, or projections and estimates too? The intake attaches the idea to both a
  request identity and an actual-cost result, which are opposite directions.]
- [NEEDS CLARIFICATION: If the chain and the canonical account ids disagree, is that success
  (partial report) or failure? The intake states both.]
- [NEEDS CLARIFICATION: Do practitioners need a stored depth and a walk helper, or only the
  ordered chain itself?]
- [NEEDS CLARIFICATION: How should a GCP project or a Kubernetes namespace be talked about,
  given this repo already treats both as a sub-account?]
- [NEEDS CLARIFICATION: For a plugin-defined level, is a generic "custom" class enough, or
  does the level need its own name?]
