# Decision: Per-request credentials for a shared plugin process

- **Slug**: per-request-credentials
- **Decided**: 2026-09-27
- **Verdict**: needs-clarification
- **Artifacts reviewed**: intake.md | research.md | problem.md | concept.md, plus
  `.specify/memory/constitution.md` (for strategic fit)

## Scorecard

| Criterion | Rating | Justification |
|-----------|--------|---------------|
| Problem validity | unknown | Separate processes can cost memory and a cold start, but no host measurement shows that cost is real or over spec 029's targets. |
| Evidence strength | weak | Spec 029, the SDK, and the roadmap are cited and consistent. The reason to change (process size, cold start, demand beyond the author) is assumption. |
| Value vs. inaction | unknown | Inaction keeps the isolation model spec 029 already chose. Nothing on issue #220 shows that model is failing. |
| Feasibility / appetite | weak | The only in-appetite step is "do not build." A shared pool is `effort/large`, and the router lives outside this repo. |
| Strategic fit | weak | Principle III allows a transport-only change, but principle IV and spec 029 keep orchestration in the host and forbid request-header credentials as the default. |
| Risk posture | weak | Isolation loss, ambient cloud SDKs, cache mixing, author-handled secrets, and overlap with open issue #195 are named and not mitigated. |

## Verdict & Rationale

**Needs clarification**, not a specification. Evidence strength is weak and problem
validity and value versus inaction are unknown, so a go is not allowed. Concept.md
recommends no option. Spec 029 already made one process per organization, with
environment credentials, the default, and it deferred issue #220. The SDK still has no
credential channel: environment helpers cover port, logs, and trace id, and request
metadata carries only `x-finfocus-trace-id`. The issue's own gate — measure memory and
cold start first — has no data behind it, including the unsourced ~50-100MB figure.
Killing the idea outright would pretend those measurements came back negative. They have
not been taken. Until a host reports them, this stays discovery.

If a later measurement meets spec 029's 100-instance and 5-second targets, or no host
will measure, the next verdict should be kill. If a host misses those targets and idle
reaping does not fix it, revisit shape before any specify step. Do not specify option C
on the current record.

## If needs-clarification

- **Blocking questions**:
  - [NEEDS CLARIFICATION: What resident memory, CPU, and concurrent process count does a
    real multi-tenant host see per plugin, and does that miss spec 029's 100-instance
    target?]
  - [NEEDS CLARIFICATION: What is measured cold-start latency versus the 5-second target,
    and how often does the idle timeout already avoid a start?]
  - [NEEDS CLARIFICATION: Has any host hit its per-node plugin instance cap (spec 029
    FR-026)?]
  - [NEEDS CLARIFICATION: Which host, if any, still wants a shared plugin pool? Pulumi
    Insights is an example in spec 029, not a committed consumer of #220.]
  - [NEEDS CLARIFICATION: Is open issue #195 the identity-metadata path that must be
    settled first, or is it independent of raw per-request cloud secrets?]
- **Revisit stage**: research

## If go — Handoff to `/speckit-specify`

Not applicable. The verdict is needs-clarification, and concept.md recommends no option.
