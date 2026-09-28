# Idea Research: Standardized cost allocation lineage metadata

- **Slug**: cost-allocation-lineage
- **Created**: 2026-09-27
- **Evidence confidence (overall)**: medium

The contract gap is documented in this repo. Demand is a single maintainer research issue.
Claims below are tagged `cited` or `assumption`.

## Users & Demand

- The only in-repo request is GitHub issue 191, opened by `rshade` on 2025-12-22 and still
  open. The title starts with `research:`. Live labels are `enhancement`, `roadmap/future`,
  and `effort/large`. The body describes FinOps drill-down, chargeback, and SOC2/HIPAA
  ownership traces. No human has commented. The only comment is a CodeRabbit bot plan prompt
  from the same minute the issue was opened. The issue was last updated 2026-02-28.
  — [source: rshade/finfocus-spec#191, fetched 2026-09-27] (confidence: high, cited)
- No support tickets, usage metrics, plugin names, or host milestones are attached to the
  issue. A named consumer is absent. — [source: issue 191 body and comment list]
  (confidence: high, cited)
- Two predecessor links in the bot comment,
  [pulumicost-spec#62](https://github.com/rshade/pulumicost-spec/issues/62) and
  [pulumicost-spec#75](https://github.com/rshade/pulumicost-spec/issues/75), were not fetched.
  They are not evidence for or against demand. — [UNVERIFIED — fetch skipped] (confidence: n/a)
- People who would feel the gap, if the stated use cases are real, are plugin authors who
  already know a provider hierarchy and host authors who would group spend by that hierarchy.
  That persona split is the issue's framing, not an observation of those users.
  — [source: issue 191 use-case section] (confidence: medium, cited as a statement of want)
- [ASSUMPTION] Enterprise finance teams walk org, folder, and management-group trees outside
  this protocol today (cloud consoles, CUR-plus-Organizations joins, tag policies). No
  interview or ticket in this repo shows that workaround failing. (confidence: low)

## Prior Art

- `FocusCostRecord` is a flat identity model, not a tree. Account columns in
  `proto/finfocus/v1/focus.proto` are `billing_account_id` (field 2), `billing_account_name`
  (field 3), `sub_account_id` (field 24), `sub_account_name` (field 25),
  `billing_account_type` (field 42), and `sub_account_type` (field 43). Resource identity is
  `resource_id` (field 12) and `resource_name` (field 13). Unordered tags are field 22.
  — [source: `proto/finfocus/v1/focus.proto`] (confidence: high, cited)
- FOCUS 1.3 allocation columns on the same message are a split-cost target, not an account
  ancestry: `allocated_method_id` (61), `allocated_method_details` (62),
  `allocated_resource_id` (63), `allocated_resource_name` (64), `allocated_tags` (65). The
  comment on `allocated_resource_id` says it is the resource receiving the allocated cost.
  — [source: `proto/finfocus/v1/focus.proto` around the FOCUS 1.3 allocation block]
  (confidence: high, cited)
- Repo FOCUS docs map sub-account to one level: AWS account, Azure subscription, GCP project.
  They map a Kubernetes namespace to `SubAccountId`, not to a separate group node. Team and
  cost-center attribution is documented as tags (`team`, `cost_center`, `environment`).
  — [source: `docs/focus-columns.md` identity tables and "Cost Allocation by Team"]
  (confidence: high, cited)
- The issue's provider table maps a GCP project and a Kubernetes namespace to
  `RESOURCE_GROUP`. That overlaps the repo's sub-account mapping for those same identifiers.
  — [source: issue 191 table vs `docs/focus-columns.md`] (confidence: high, cited)
- `AllocatorService.Allocate` (issue 506, spec `052-allocator-allocate`) already uses the
  words "cost allocation" for a different job: dividing a priced node or control plane across
  workloads. It does not carry org ancestry. `PLUGIN_CAPABILITY_ALLOCATION` is 15.
  — [source: `proto/finfocus/v1/allocation.proto`, `proto/finfocus/v1/enums.proto`]
  (confidence: high, cited)
- A search of this worktree found no `LineageNode`, no `lineage` field, and no
  `lineage_builder.go`. The draft would be new surface, not a revival of an in-tree type.
  — [source: repository search on 2026-09-27] (confidence: high, cited)
- `ResourceDescriptor.tags` is already documented as "additional resource matching and cost
  allocation." — [source: `proto/finfocus/v1/costsource.proto` field `tags` = 5]
  (confidence: high, cited)
- Issue 191 was opened before `expires_at` landed. Spec `045-caching-hint-expires-at` is dated
  2026-02-16 and adds `ActualCostResult.expires_at`. The draft's "fields 1–7, lineage = 8"
  matched the message at filing time and does not match it now.
  — [source: `specs/045-caching-hint-expires-at/spec.md`; current message below]
  (confidence: high, cited)

### Field numbers checked against `fc1402d`

`ResourceDescriptor` (`proto/finfocus/v1/costsource.proto`) ends at field 10:

| Field | Name | Issue 191 claim |
| ----- | ---- | --------------- |
| 1 | `provider` | grouped as "existing 1–10" |
| 2 | `resource_type` | same |
| 3 | `sku` | same |
| 4 | `region` | same |
| 5 | `tags` | same |
| 6 | `utilization_percentage` | same |
| 7 | `id` | same |
| 8 | `arn` | same |
| 9 | `growth_type` | same |
| 10 | `growth_rate` | same |
| 11 | free | draft assigns `lineage` here; still unused |

`ActualCostResult` in the same file:

| Field | Name | Issue 191 claim |
| ----- | ---- | --------------- |
| 1 | `timestamp` | "existing 1–7" |
| 2 | `cost` | same |
| 3 | `usage_amount` | same |
| 4 | `usage_unit` | same |
| 5 | `source` | same |
| 6 | `focus_record` | same |
| 7 | `impact_metrics` | same |
| 8 | `expires_at` (`google.protobuf.Timestamp`) | draft assigns `lineage` here; **taken** |

The next unused `ActualCostResult` number is 9. Reusing 8 would change the wire type of
`expires_at`. Constitution principle VI forbids that kind of break without a major version,
and removed numbers must be `reserved`. — [source: `.specify/memory/constitution.md`
principle VI; the message above] (confidence: high, cited)

`GetProjectedCostResponse` has no lineage field. Its last fields are `expires_at` = 13 and
`metadata` = 14 (a bounded `map<string, string>`). `EstimateCostResponse` is a separate
message. The draft does not mention either response. — [source: `costsource.proto`]
(confidence: high, cited)

Direction of the two proposed attachments differs:

- `ResourceDescriptor` is host-to-plugin input (`GetProjectedCostRequest.resource`,
  `BatchCostRequest.resources`) and is echoed on `ResourceCostResult.resource`.
- `ActualCostResult` is plugin-to-host output (`GetActualCostResponse.results` and
  `ActualCostData.results` inside batch actuals).

The issue's objective says plugins pass metadata upstream. Only the result attachment matches
that direction. The descriptor attachment would be context the host already has.
— [source: issue 191 objective vs the request and response messages] (confidence: high, cited)

`ActualCostResult.focus_record` (field 6) embeds `FocusCostRecord`. A lineage field on the
result sits beside the FOCUS record. A lineage field on `FocusCostRecord` is not what the
issue proposes. Hosts that persist only the FOCUS record would drop a sibling field. That is
an observation, not a recommendation. — [source: `ActualCostResult` field 6] (confidence: high,
cited)

## Market & Context

- Doing nothing leaves the documented model: two account levels plus tags, and FOCUS 1.3
  split-allocation columns when a cost is assigned to another resource. Drill-down past the
  sub-account stays outside the contract (tags, `GetProjectedCostResponse.metadata`, or a
  system this spec does not define). — [source: `docs/focus-columns.md`; `focus.proto`;
  projected-cost `metadata`] (confidence: high, cited)
- The constitution puts end-user drill-down in core, not in this repo. Principle IV: the spec
  defines interfaces; core holds application logic. The CFO click-path in the issue is a host
  experience. Principle III: the spec consumes calculated data and does not become a pricing
  engine. A pass-through chain is closer to III than `Allocate` was, because it does not
  compute amounts. — [source: `.specify/memory/constitution.md` principles III and IV]
  (confidence: high, cited)
- [ASSUMPTION] Cloud billing exports commonly stop at payer and usage account, with org-unit
  or folder ancestry joined from a second API. That is why a plugin might have a chain the
  FOCUS columns cannot store. Not re-verified against provider docs in this assessment.
  (confidence: medium)
- The external FOCUS specification was not re-fetched. Column behavior above is taken from
  this repo's proto and `docs/focus-columns.md`, which cite
  `https://focus.finops.org/focus-specification/v1-2/` and `v1-3/`.
  — [UNVERIFIED — fetch skipped: host not on safe list: `focus.finops.org`] (confidence: n/a)

## Data & Constraints

- Actual-cost pages are capped at 1000 results (`MaxPageSize` in `sdk/go/testing/contract.go`).
  A chain hung off every result multiplies payload by depth and by metadata map size. No
  measurement of that cost exists. — [source: `sdk/go/testing/contract.go`] (confidence: high
  for the cap; low for the performance impact)
- The in-memory test harness uses a 1 MiB buffer (`sdk/go/testing/harness.go`). A conformance
  fixture with a deep chain is unlikely to care; a page of wide metadata maps might.
  — [source: `harness.go` `bufSize`] (confidence: medium, cited limit, assumption on impact)
- The draft requires "arbitrary depth" and "no max depth." [ASSUMPTION] Protobuf decoders
  impose some nesting limit, so an unbounded recursive `parent` cannot be a literal guarantee.
  The numeric limit was not confirmed in this repo's module cache. (confidence: low)
- Adding a field with a new number is wire-compatible. Reusing field 8 is not.
  — [source: constitution VI; current `ActualCostResult`] (confidence: high, cited)
- Principle XIII requires every SDK to pick up proto changes. It specifically requires
  high-level client wrappers when a new RPC is added. The draft adds no RPC, so the cited
  obligation is regeneration of Go and TypeScript bindings, not a new client method. A fluent
  builder would be extra Go API and, if treated as public SDK surface, would need docs and
  tests under principles V, VIII, and XIV. — [source: constitution V, VIII, XIII, XIV]
  (confidence: high, cited)
- The draft's `WithMetadata` walks to the current root. Metadata therefore lands on the latest
  parent, or on the resource if no parent has been added. An intermediate node cannot be tagged
  after a higher parent is linked. That is a property of the pasted snippet, not of the
  problem. — [source: issue 191 Go snippet] (confidence: high, cited)
- Open question 3 (fail conformance when lineage and `billing_account_id` disagree) conflicts
  with the same issue's rule that partial chains are valid and that the SDK must not invent
  or require a relationship to `billing_account_id`. — [source: issue 191 "Anti-Guess
  Boundary" and "Open Questions"] (confidence: high, cited)

## Evidence Against the Idea

- Demand is one icebox research issue with no consumer. Nothing in-tree is blocked the way
  `GetStats` blocked `Allocate`. `roadmap/future` is the author's own sequencing label.
  — [source: issue 191 labels and comment list; contrast
  `.specify/assessments/allocator-allocate/decision.md`] (confidence: high, cited)
- A workable attribution path already ships: billing account, sub-account, resource id, tags,
  and FOCUS 1.3 allocated-resource columns. The missing piece is ordered parentage above the
  sub-account, not the ability to name a cost center. — [source: `focus.proto`,
  `docs/focus-columns.md`] (confidence: high, cited)
- Shipping the draft unchanged would collide with `ActualCostResult.expires_at` (field 8).
  — [source: `costsource.proto`] (confidence: high, cited)
- Putting one ancestry enum on both a request descriptor and a cost result mixes two
  directions and overlaps FOCUS sub-account for GCP projects and Kubernetes namespaces.
  — [source: issue 191 table; `docs/focus-columns.md`] (confidence: high, cited)
- "Cost allocation" is now `AllocatorService`. A second feature under the same words will be
  confused with workload split math. — [source: `allocation.proto`] (confidence: medium, cited)
- The compliance use case names SOC2 and HIPAA but does not say which control a lineage field
  would satisfy. [ASSUMPTION] An optional, plugin-asserted chain with no external ID check
  would not by itself be audit evidence. (confidence: low)

## Gaps & Open Questions

- [NEEDS CLARIFICATION: Will any named host or plugin populate and read this chain in a
  planned milestone, or does `roadmap/future` mean it stays icebox?]
- [NEEDS CLARIFICATION: Which messages should carry a chain: actual-cost results, projected
  and estimate responses, the request `ResourceDescriptor`, `FocusCostRecord`, or some subset?
  The draft names only the descriptor (field 11, still free) and `ActualCostResult` (field 8,
  not free).]
- [NEEDS CLARIFICATION: Should a mismatch between a chain and `billing_account_id` /
  `sub_account_id` be ignored, warned, or rejected? The anti-guess section and open question 3
  disagree.]
- [NEEDS CLARIFICATION: Is a stored `depth` required, or is position in the chain enough? The
  issue asks and does not answer.]
- [NEEDS CLARIFICATION: Is a walk helper part of the contract's success, or only a convenience?]
- [NEEDS CLARIFICATION: For `CUSTOM`, is the enum value enough, or does the level need its own
  name? The draft has no name field distinct from the display `name`.]
- [NEEDS CLARIFICATION: How should GCP projects and Kubernetes namespaces be classified, given
  this repo already maps both to sub-account?]
- Predecessor issues 62 and 75 were not read. [UNVERIFIED — fetch skipped: the client cannot
  pin the connected peer, and those URLs were not the mandated fetch.]

## Sources

- [rshade/finfocus-spec#191](https://github.com/rshade/finfocus-spec/issues/191)
  (host: `github.com`, policy: `allowlisted`; fetched with user-mandated `gh`)
- `proto/finfocus/v1/costsource.proto` (local)
- `proto/finfocus/v1/focus.proto` (local)
- `proto/finfocus/v1/allocation.proto` (local)
- `proto/finfocus/v1/enums.proto` (local)
- `docs/focus-columns.md` (local)
- `specs/045-caching-hint-expires-at/spec.md` (local)
- `specs/052-allocator-allocate/spec.md` (local)
- `sdk/go/testing/contract.go` (local)
- `sdk/go/testing/harness.go` (local)
- `.specify/memory/constitution.md` (local)
- `.specify/assessments/allocator-allocate/decision.md` (local; prior assessment, not a user
  study)
- [pulumicost-spec#62](https://github.com/rshade/pulumicost-spec/issues/62)
  (host: `github.com`, policy: `auto-refused: connection peer cannot be pinned`)
- [pulumicost-spec#75](https://github.com/rshade/pulumicost-spec/issues/75)
  (host: `github.com`, policy: `auto-refused: connection peer cannot be pinned`)
- `https://focus.finops.org/focus-specification/v1-2/` and `v1-3/`
  (host: `focus.finops.org`, policy: `auto-refused: host not on safe list`; cited only as links
  already published in `docs/focus-columns.md`)
