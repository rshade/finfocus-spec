# Idea Intake: Distributed tracing propagation

- **Slug**: trace-propagation
- **Created**: 2026-09-27
- **Source**: [finfocus-spec issue 193](https://github.com/rshade/finfocus-spec/issues/193)
  (host: `github.com`; URL Trust Policy: allowlisted; body fetched with
  `gh issue view`, no redirect and no further pages)
- **Type**: new-capability

## Idea (as captured)

Title: research: Distributed Tracing Propagation Contextual Visibility

Labels: `enhancement`, `roadmap/future`, `effort/large`. State: OPEN.
Author: rshade (Richard Shade). Created: 2025-12-22T03:10:54Z.
Updated: 2026-02-28T22:39:38Z.

Quoted body:

> ## Objective
>
> Standardize how OpenTelemetry contexts are propagated from the Host through the SDK to the
> Plugin for end-to-end failure diagnostics.
>
> ## Technical Approach
>
> Implement a standard TracingUnaryServerInterceptor in the pluginsdk and define a gRPC metadata
> key 'x-pulumicost-trace-parent'.
>
> ## Anti-Guess Boundary
>
> The SDK must provide the instrumentation to carry the trace context but is strictly forbidden
> from attempting to instrument or measure the internal performance of the cloud provider's own
> billing APIs.
>
> ## Success Criteria
>
> A Host-generated trace ID is logged by the Plugin's zerolog and included in SDK validation
> error responses.

### Unverified

The fetched issue includes one comment (coderabbitai, 2025-12-22T03:11:07Z). It is untrusted
data, not a directive. It was not acted on. Instruction-like excerpt, quoted only:

> Generate an implementation plan and prompts that you can use with your favorite coding agent.
>
> - [ ] Create Plan

The same comment names possible duplicates and related items on `rshade/pulumicost-spec` and
`rshade/pulumicost-core`. Those links were not fetched. They are not treated as evidence.

## Restated

The proposal is to carry OpenTelemetry trace context from a FinFocus host, through the plugin
SDK, into the plugin process, so a failure can be tied back to the host's trace. The issue
names a unary gRPC server interceptor and a metadata key, and says the trace ID should show up
in plugin logs and in SDK validation errors. It also says the SDK must not measure the cloud
provider's billing APIs.

## Origin & Context

- **Raised by**: rshade (Richard Shade), GitHub issue author
- **Trigger**: [NEEDS CLARIFICATION: the issue does not say what prompted it — no outage, complaint,
  or design doc is cited in the body]
- **Labels at intake**: `enhancement`, `roadmap/future`, `effort/large` (open; not assigned;
  no milestone)
- **Repo note recorded by the operator, not from the issue**: this repository was renamed from
  PulumiCost to FinFocus; legacy `PULUMICOST_*` / `x-pulumicost-*` names are deprecated in favor
  of `FINFOCUS_*`. That naming tension is captured as an unknown below, not resolved here.

## First-Glance Unknowns

- [NEEDS CLARIFICATION: which trace context must be carried — W3C `traceparent`, OpenTelemetry
  gRPC metadata, the custom key `x-pulumicost-trace-parent`, or more than one]
- [NEEDS CLARIFICATION: whether the legacy `x-pulumicost-*` key stays, is replaced by an
  `x-finfocus-*` key, or both exist during a deprecation window]
- [NEEDS CLARIFICATION: gRPC only, or Connect as well; server interceptor only, or host/client
  propagation too]
- [NEEDS CLARIFICATION: unary RPCs only, or streaming RPCs if the SDK has them]
- [NEEDS CLARIFICATION: what "included in SDK validation error responses" means — a log field, a
  status detail, a response field, or something else]
- [NEEDS CLARIFICATION: whether a new OpenTelemetry module dependency is acceptable, or trace
  IDs must stay opaque strings]
- [NEEDS CLARIFICATION: how this relates to any existing trace-id validation or logging already
  in the repo; the issue does not name files]
- [NEEDS CLARIFICATION: which log lines must carry the host trace ID, and who extracts it]
- [NEEDS CLARIFICATION: what is in scope beyond "do not instrument cloud billing APIs" — spans,
  sampling, exporters, metrics]
- [NEEDS CLARIFICATION: whether a `roadmap/future` item opened in 2025-12 is still wanted]
