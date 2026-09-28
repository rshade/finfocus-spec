# Idea Intake: Multi-Currency Cost Segregation Pattern

- **Slug**: multi-currency-segregation
- **Created**: 2026-09-27
- **Source**: [finfocus-spec issue 190](https://github.com/rshade/finfocus-spec/issues/190)
  (host: github.com; URL Trust Policy: allowlisted; fetched via `gh issue view`, no redirect followed)
- **Type**: exploration

## Idea (as captured)

GitHub issue #190, opened 2025-12-22 by rshade (Richard Shade), still open.
Title: `discovery: Multi-Currency Cost Segregation Pattern`.
Labels: `enhancement`, `roadmap/future`, `effort/large`.
No milestone, assignee, parent, or sub-issues.
Last updated 2026-02-28.

Quoted body:

```text
## Rationale
Standardize how records with multiple currencies are handled in the SDK to prevent accidental aggregation of different units.

## Boundary Check
Validation/typing feature; does not perform currency conversion (math).
```

## Unverified

The only comment is an auto-generated CodeRabbit "Plan Mode" note (not acted on). Instruction-like text, quoted verbatim:

> Generate an implementation plan and prompts that you can use with your favorite coding agent.

The same comment lists a related issue
`https://github.com/rshade/pulumicost-spec/issues/101` and suggests assignee rshade.
That linked issue was not fetched.

## Restated

The idea is to standardize, in the FinFocus SDK, how cost records that use more than one currency
are typed and validated so amounts in different currency units are not accidentally added together.
The stated boundary is validation and typing only; currency conversion math is out of scope.

## Origin & Context

- **Raised by**: rshade (Richard Shade), GitHub issue #190
- **Trigger**: [NEEDS CLARIFICATION: the issue does not say what prompted it (incident, host
  bug, FOCUS rule, or roadmap item)]
- **When**: opened 2025-12-22; labels include `roadmap/future` and `effort/large`
- **Repo context at intake**: branch `assess/190-multi-currency-segregation` at `fc1402d` (origin/main)

## First-Glance Unknowns

- [NEEDS CLARIFICATION: what "segregation" means in the SDK (typed money, grouping key,
  reject mixed sums, or a warning only)]
- [NEEDS CLARIFICATION: which records and RPCs are in scope (actual cost, projected cost,
  FOCUS rows, allocation, pricing spec, budgets)]
- [NEEDS CLARIFICATION: whether Go, TypeScript, proto, or all three must change]
- [NEEDS CLARIFICATION: who enforces the rule (plugin author, host, or both) and what the
  failure looks like]
- [NEEDS CLARIFICATION: how this relates to existing ISO 4217 currency validation already in
  the repo]
- [NEEDS CLARIFICATION: whether unspecified, empty, or multi-currency line items inside one
  response are in scope]
- [NEEDS CLARIFICATION: whether any host already aggregates across currencies, and whether
  that behavior is a bug to close]
