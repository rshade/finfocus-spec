# Concept: Optional chain of custody on actual cost

- **Slug**: cost-allocation-lineage
- **Created**: 2026-09-27
- **Recommended option**: Option B — pass-through chain on actual cost

## Options

### Option A — Leave ancestry to account fields and tags

- **Sketch**: No contract change. Plugins keep reporting the billing account, the sub-account,
  the resource, and tags. Hosts that need an org, folder, or management group keep joining
  that hierarchy themselves. Chargeback by cost-center tag stays as documented.
- **Appetite**: small (no build; the `effort/large` label does not apply)
- **Trade-offs**: Wins zero new surface, no second account model to drift from FOCUS columns,
  and no collision with the caching hint already stored on actual-cost results. Sacrifices
  ordered parentage. Two hosts can reconstruct different parents and the spec cannot say what
  the plugin reported. Risk: the gap stays accepted forever because nothing is blocked.
- **Rabbit holes**: None in this repo. The risk is unofficial conventions growing inside tag
  keys or the projected-cost metadata map, which this option does not police.

### Option B — Pass-through chain on actual cost only

- **Sketch**: A plugin may attach, to each actual cost it returns, the ordered chain of
  custody it already has, from the leaf toward the root. Each step has a class (organization,
  organizational unit, billing account, sub-account, resource group, resource, or custom), an
  id, a display name, and optional notes. A missing chain is normal. A chain that stops early
  is normal. The host stores and walks what it was given. It does not invent a parent, does
  not call a cloud API to check ids, and does not fail the cost because the chain omits or
  disagrees with `billing_account_id` or `sub_account_id`. Those FOCUS fields stay canonical.
  Projections, estimates, and the resource descriptor on the request do not gain a chain in
  this option. No new RPC. The December 2025 sketch's field number on the actual-cost result
  is already the caching hint, so any real change uses a new number and leaves that hint
  alone. Generated Go and TypeScript bindings follow the proto. A small helper that only
  assembles caller-supplied steps is optional; it is not the contract.
- **Appetite**: medium (weeks). The issue's `effort/large` label matches Option C better than
  this slice. Uncertainty: medium, because a helper, conformance fixtures, and docs can spill
  toward large if every cost RPC is pulled in.
- **Trade-offs**: Wins the stated goals (ordered partial parentage, no inference, old clients
  unchanged) at the size of an optional field plus tests. Sacrifices request-side lineage and
  projected-cost lineage, so a host that only estimates still has no chain unless it already
  knew one. Risks: payload growth on pages of up to 1000 results; confusion with
  `AllocatorService`; GCP project and Kubernetes namespace already mapped to sub-account in
  this repo's FOCUS docs, so the same id may appear as two classes. A nested parent also
  cannot honestly promise unlimited depth.
- **Rabbit holes**: Choosing nested parents versus a flat ordered list. Reusing field 8.
  Conformance that treats disagreement with the billing account as failure. A builder whose
  notes stick to the wrong step (the pasted snippet walks to the current root). A free-form
  name for custom classes. Extending the same chain onto projections, estimates, batch
  wrappers, and the request descriptor "while we are here." Treating the chain as audit
  evidence. A walk helper and a stored depth, which the issue asks about and which this
  option does not need for the contract.

### Option C — Chain on the request and the result, plus builder and consistency checks

- **Sketch**: Implement the issue sketch as written. The host can send a chain on the
  resource it asks about, and the plugin can return a chain on the actual cost. A fluent
  builder starts at the resource and links parents upward. Conformance covers a four-level
  round trip and may also check that the chain agrees with `billing_account_id`. Depth and a
  walk helper are in or out depending on the open questions.
- **Appetite**: large (months if consistency rules, both directions, and the helper surface
  are included). Matches the `effort/large` label. The sketched result field number cannot be
  used; that part of the sketch is already stale.
- **Trade-offs**: Wins the closest match to the filed text, including hosts that want to push
  hierarchy in rather than only read it back. Sacrifices a small contract: two directions,
  a larger test matrix, and a direct clash with the same issue's rule that partial chains are
  valid if consistency becomes a hard check. Risk of breaking the caching hint if the old
  field number is followed literally.
- **Rabbit holes**: All of Option B's, plus request/response duplication, echoing the chain
  on batch resource descriptors, and what a plugin should do when the host's chain and the
  billing export disagree. The pasted builder cannot annotate a middle step after a higher
  parent is added.

## Recommendation

Option B, if this problem is specified at all. It is the smallest change that meets the
goals: the host can read a partial ordered chain, nothing is inferred, and costs without a
chain stay valid. Success is a four-level actual-cost round trip plus a negative case where
a short chain and a different billing account are both accepted. Option A remains the right
call while the work is icebox and no consumer exists; it does not meet the goals, it declines
them. Option C spends the large appetite on a second direction and on a consistency check the
problem statement treats as out of scope.

## Out of Scope (for the recommended option)

- Host drill-down, chargeback UI, and compliance attestation.
- Allocation math and `AllocatorService` behavior.
- Replacing or rewriting FOCUS account, resource, or tag fields.
- Inferring parents, calling provider APIs to validate ids, or rejecting a cost when the
  chain and the account ids disagree.
- A required complete chain, a stored depth, and a walk helper as part of the contract.
- Lineage on projected cost, estimates, or the request resource descriptor.
- A new RPC or a new plugin capability.
- Reusing the actual-cost field number that already carries the caching hint.
- A promise of unlimited nesting depth.

## Assumptions to Validate

- The consumer, when one exists, needs the chain on actual cost and not on projections or on
  the request. If hosts must send hierarchy into a pricing plugin, Option B is the wrong
  slice.
- Partial chains and disagreement with `billing_account_id` are success, not errors. This
  restates the anti-guess boundary and rejects open question 3 as a hard check.
- Display name plus notes are enough for a custom level. If not, the chain needs a level name
  before plugins can tell two custom steps apart.
- Plugin authors can classify a GCP project and a Kubernetes namespace without colliding with
  the repo's existing sub-account mapping, or the class list must say they are sub-accounts.
- Maintainers want this specified before a host or plugin is committed to use it. Research
  does not show that commitment. `roadmap/future` points the other way.
- A nested encoding is allowed to impose a practical depth ceiling. "No max depth" in the
  issue is not a guarantee this repo can make.
