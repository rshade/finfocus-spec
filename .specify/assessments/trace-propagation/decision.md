# Decision: Correlating plugin failures to the host request

- **Slug**: trace-propagation
- **Decided**: 2026-09-27
- **Verdict**: needs-clarification
- **Artifacts reviewed**: intake.md | research.md | problem.md | concept.md, plus
  `.specify/memory/constitution.md` (strategic fit only; not an assessment write)

## Scorecard

| Criterion | Rating | Justification |
|-----------|--------|---------------|
| Problem validity | adequate | Logs and validation errors do not automatically carry the host trace id, and Connect drops the header. That gap is in the code. No incident, ticket, or usage count shows it hurting anyone now. The issue's "build an interceptor" wording is not a current problem. |
| Evidence strength | adequate | SDK behavior, the header rename, spec 005/008, and the Connect deferral in specs 051/052 are cited from this repo. Demand is one maintainer issue plus a bot comment. What the host sends, and the W3C header shape, are unknown because those sources were not fetched. Enough to reject the stale approach; not enough to pick a replacement. |
| Value vs. inaction | weak | Doing nothing keeps a working gRPC context id and a documented manual log field. The unmeasured cost is plugins that forget the field, validation errors with no id, and Connect. Option A might be worth days later. It does not beat inaction on the evidence in hand. |
| Feasibility / appetite | adequate | Option A is a small, located change. Option C's size is honestly unknown and easy to inflate into a tracer, an exporter, and a Connect redesign. The `effort/large` label describes the issue text, not a measured residual. |
| Strategic fit | adequate | Maintainer observability matches constitution IX (zerolog for events, metrics separate). A direct OpenTelemetry stack fits poorly with IV (minimal dependencies) and with the issue's ban on measuring provider billing APIs. XIII would pull TypeScript in only if the contract changes. |
| Risk posture | weak | Proceeding now either rebuilds an interceptor that already shipped, or opens header format, spec 008's replace-on-invalid rule, spec 047's error text, and Connect scope that later specs closed. Those risks are named and not mitigated, because the problem frame was not chosen. |

## Verdict & Rationale

**Needs clarification.** Problem validity and evidence are adequate, which would
allow a go only with a recommended concept. concept.md recommends none. The
approach in issue #193 (a new `TracingUnaryServerInterceptor` and
`x-pulumicost-trace-parent`) is already superseded: the interceptor has shipped
since 2025-11-24, and the key is `x-finfocus-trace-id`, formerly
`x-pulumicost-trace-id`, never `trace-parent`. Value versus inaction is weak,
and risk posture is weak, so this is not a go. It is also not a kill: a smaller
gap (the id on the context is not forced into zerolog or into `ValidationError`,
and Connect never sees it) is real, and a human could still want Option A. That
choice changes the problem being specified, so it has to be made before specify.
Unknowns called out here are the host's actual header, production frequency, and
whether `roadmap/future` still means not now. They are not treated as evidence.

## If needs-clarification

- **Blocking questions**:
  - [NEEDS CLARIFICATION: which problem is in scope — (1) the gRPC context id is
    not automatically present in the plugin zerolog line and in SDK validation
    errors, or (2) the host's full trace context never arrives (other header,
    span id, Connect)? Option A solves only (1). Option C is (2). Option B is
    "neither, the named work already shipped."]
  - [NEEDS CLARIFICATION: what header and value does the host send today? If it
    is not `x-finfocus-trace-id` as 32 lowercase hex, Option A does not preserve
    the host id. Answering this means looking at finfocus-core or a host log,
    which this assessment did not do.]
  - [NEEDS CLARIFICATION: if the host value is not 32-hex, keep it (issue success
    text) or replace it (spec 008)?]
  - [NEEDS CLARIFICATION: is Connect in scope? Specs 051 and 052 left it out on
    purpose.]
  - [NEEDS CLARIFICATION: is this still wanted, given `roadmap/future`,
    `effort/large`, no assignee, and no human comment since 2025-12-22?]
- **Revisit stage**: define

Define must pick one problem statement. If the pick is (2), run research again
against the host repo before shaping Option C. Do not shape from the unverified
OpenTelemetry assumption in research.md.

## If go — Handoff to `/speckit-specify`

Not applicable. Verdict is needs-clarification. There is no chosen approach, and
specify must not start from this record.
