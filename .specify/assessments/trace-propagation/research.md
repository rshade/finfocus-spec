# Idea Research: Distributed tracing propagation

- **Slug**: trace-propagation
- **Created**: 2026-09-27
- **Evidence confidence (overall)**: medium

Internal code and spec citations below are high confidence. What the host actually
sends, and the current W3C / OpenTelemetry wire format, were not fetched (see
Sources). Those gaps keep the overall confidence at medium. This note does not
decide whether to build anything.

## Users & Demand

- One stated request: GitHub issue rshade/finfocus-spec#193, opened 2025-12-22 by
  rshade, titled "research: Distributed Tracing Propagation Contextual Visibility".
  Labels: `enhancement`, `roadmap/future`, `effort/large`. No assignee, no milestone.
  `updatedAt` is 2026-02-28; the timeline was not fetched, so the reason for that
  update is unknown. — [source: `gh issue view 193`] (confidence: high)
- The only comment is a CodeRabbit plan-mode bot note (2025-12-22). It is not a
  user asking for the feature. It names older `pulumicost-spec` issues and PRs.
  Those linked pages were not fetched. — [source: issue comment on #193]
  (confidence: high that the comment exists; low that the linked items still
  describe this gap)
- No support tickets, interviews, or usage metrics are in this repository. Demand
  strength beyond the maintainer's own roadmap label is unknown. — [source: repo
  search; absence is not proof of no demand] (confidence: medium)
- Observed behavior in this repo is narrower than the issue text: a trace-id
  interceptor has been on by default for gRPC `Serve` since 2025-11-24, before
  the issue was opened. — [source: `git log -S 'func TracingUnaryServerInterceptor'`
  on `sdk/go/pluginsdk/logging.go`, commit `6d5b5ac`, 2025-11-24, "feat(sdk): add
  zerolog logging utilities" (#76)] (confidence: high)

## Prior Art

- `TracingUnaryServerInterceptor` already exists and is chained first in gRPC
  `Serve`. It reads incoming metadata, keeps the first value, and stores a string
  on the request context. It does not start a span, call a provider API, or write
  a zerolog event. — [source: `sdk/go/pluginsdk/logging.go` `TracingUnaryServerInterceptor`
  (about lines 258–292); `sdk/go/pluginsdk/sdk.go` `serveGRPC` (about lines 1312–1319)]
  (confidence: high)
- The metadata key is `x-finfocus-trace-id` (`TraceIDMetadataKey`), not the issue's
  `x-pulumicost-trace-parent`. A repo-wide search finds no `x-pulumicost-trace-parent`
  and no `traceparent`. The pre-rename constant was `x-pulumicost-trace-id` (trace
  id, not trace parent), in the parent of commit `eecdddb` (2026-01-11, rename
  #273). The issue was filed while the old `trace-id` key was still current, and
  it already did not match that key. — [source: `logging.go` line 25; `git show
  eecdddb^:sdk/go/pluginsdk/logging.go`; `git grep` style search] (confidence: high)
- Invalid or missing ids are replaced with a generated 32-character lowercase hex
  id (16 crypto-random bytes). Callers are not failed. An all-zero id is invalid.
  A W3C `traceparent` value (longer than 32 hex characters) would fail
  `ValidateTraceID` and be replaced, so the host id would not be the one stored.
  — [source: `logging.go` lines 279–290; `sdk/go/pluginsdk/traceid.go`;
  `sdk/go/pricing/observability_validate.go` `ValidateTraceID` lines 96–111;
  `specs/008-trace-id-validation/spec.md` FR-001 through FR-009] (confidence: high
  for the code rule; the W3C header shape is an assumption, fetch skipped)
- Spec 005 (`specs/005-zerolog/`, input "GitHub Issue #75") required the
  interceptor, `x-finfocus-trace-id` in the spec text, `TraceIDFromContext`, and
  field constant `trace_id`. Its scenario for a missing header is an empty
  context value. Spec 008 later required replacement instead of empty. Both spec
  files still say `Status: Draft` even though the commits above landed.
  — [source: `specs/005-zerolog/spec.md` User Story 2 and FR-002–FR-005;
  `specs/008-trace-id-validation/spec.md` header and FR-009] (confidence: high)
- Logging the id is a plugin-author step. `NewPluginLogger` does not attach
  `trace_id`. `LogOperation` logs only `operation` and `duration_ms`. README and
  `TestStructuredLoggingExample` show `logger.Info().Str(FieldTraceID,
  TraceIDFromContext(ctx))`. There is no zerolog hook in `pluginsdk` that does
  this. The example id `trace-abc123` is not a valid 32-hex id and is injected
  with `ContextWithTraceID`, bypassing the interceptor. — [source:
  `sdk/go/pluginsdk/logging.go` `NewPluginLogger`, `LogOperation`;
  `sdk/go/pluginsdk/README.md` "Creating a Plugin Logger" and "Trace ID
  Propagation"; `sdk/go/testing/integration_test.go` `TestStructuredLoggingExample`]
  (confidence: high)
- `pluginsdk.ValidationError` fields are field name, constraint, actual, expected,
  and an optional cause. `Error()` does not include a trace id. Proto
  `ErrorDetail` has no trace field. `ErrorDetails.correlation_id` exists on
  `LogEntry` and is not set by `pluginsdk` Go code. `TelemetryMetadata.trace_id`
  and `span_id` exist on the proto and are not populated by `pluginsdk`.
  — [source: `sdk/go/pluginsdk/validation_error.go`; `proto/finfocus/v1/costsource.proto`
  `ErrorDetail`, `TelemetryMetadata`, `LogEntry`, `ErrorDetails`; search of
  `sdk/go/pluginsdk` for `TelemetryMetadata` returned no Go matches]
  (confidence: high)
- Connect mode does not install the tracing interceptor or `UnaryInterceptors`.
  `serveConnect` passes an empty `handlerOpts`. Specs 051 and 052 record that as
  intentional parity with cost RPCs, and say adding Connect interceptors was out
  of scope. Docs repeat it. — [source: `sdk/go/pluginsdk/sdk.go` `serveConnect`
  around the empty `handlerOpts`; `specs/051-usage-source-getstats/research.md`
  "Interceptors and tracing"; `specs/051-usage-source-getstats/spec.md`
  clarification on Connect interceptors; `docs/usage-source.md` "Transport Notes";
  `docs/allocator.md` states the same pattern] (confidence: high)
- No streaming RPCs. A search for the word "stream" followed by a space in
  `proto/finfocus/v1/*.proto` hits only comments about rate-limited APIs.
  Unary-only matches the current contract. — [source: `proto/finfocus/v1/`]
  (confidence: high)
- No host/client injector in this module. Search found no
  `UnaryClientInterceptor` and no `AppendToOutgoingContext` under `sdk/go`. The
  TypeScript Node transport adds a timeout interceptor only.
  — [source: search of `sdk/go`; `sdk/typescript/packages/middleware/src/transport.ts`]
  (confidence: high)
- A second, process-wide channel exists: `GetTraceID()` reads `FINFOCUS_TRACE_ID`,
  then deprecated `PULUMICOST_TRACE_ID`. The interceptor does not call it. Nothing
  in `pluginsdk` joins the env var to the per-request context. — [source:
  `sdk/go/pluginsdk/env.go` `EnvTraceID`, `GetTraceID`; `specs/016-pluginsdk-env/spec.md`
  FR-010. Spec text still names `PULUMICOST_TRACE_ID` as canonical; the code
  prefers `FINFOCUS_TRACE_ID`] (confidence: high)
- `OBSERVABILITY_GUIDE.md` "Distributed Tracing" shows `go.opentelemetry.io/otel`
  span start inside `GetActualCost`, plus extract/inject helpers. Those symbols
  are not imported by any non-generated Go file in this repo. The guide is sample
  text, not the SDK. — [source: `OBSERVABILITY_GUIDE.md` section "Distributed
  Tracing"; `rg go.opentelemetry.io` hits `go.mod`, `go.sum`, and that guide]
  (confidence: high)
- OpenTelemetry Go modules are indirect, pulled through `github.com/rshade/ax-go`,
  not required by this module and not called by `pluginsdk`. Versions in `go.mod`
  include `go.opentelemetry.io/otel v1.46.0` and `otelgrpc v0.71.0`.
  — [source: `go.mod` require block; `go mod why` for `otelgrpc` ends at `ax-go`]
  (confidence: high)
- Constitution IV: this repo is spec and plugin SDK; application/host logic is
  `finfocus-core`, and dependencies should stay minimal. Constitution IX: zerolog
  for events, Prometheus for metrics, metrics as optional interceptors.
  Constitution VIII: zero-allocation goal and benchmarks for new core SDK logic.
  Constitution XIII: Go and TypeScript SDKs stay in sync when the contract
  changes. — [source: `.specify/memory/constitution.md` sections IV, VIII, IX,
  XIII] (confidence: high)
- The issue's anti-guess line (do not measure cloud billing APIs) matches what
  the interceptor does today (copy a string onto context). The guide's sample
  starts a span around `GetActualCost`, which is plugin work, not a provider HTTP
  client. No SDK code wraps a cloud billing client. — [source: issue #193 body;
  `logging.go` interceptor; guide sample] (confidence: high)
- CodeRabbit called pulumicost-spec#75 a possible duplicate and #94 related.
  Local specs quote those numbers as their inputs (005 ← #75, 008 ← #94) and the
  matching PRs are in git history (#76, #96). The old issue pages were not
  opened. — [source: issue #193 comment; spec front matter; git log] (confidence:
  medium)

## Market & Context

- Coping mechanisms already in tree: gRPC metadata `x-finfocus-trace-id` plus
  manual zerolog fields; process env `FINFOCUS_TRACE_ID`; proto messages that
  can carry `trace_id` if a plugin fills them; an uncompiled OpenTelemetry sketch
  in `OBSERVABILITY_GUIDE.md`. — [source: files cited above] (confidence: high)
- Cost of doing nothing, on the evidence here: a host that already sends
  `x-finfocus-trace-id` as 32 lowercase hex gets that id on the handler context
  for every gRPC unary RPC, including usage and allocator services. A plugin that
  does not add `FieldTraceID` will not log it. Connect listeners drop the header.
  Validation errors do not echo it. A host that sends only a standard traceparent
  header would not be correlated by this interceptor. Whether any host does that
  is [NEEDS CLARIFICATION]. — [source: code citations above] (confidence: high
  for SDK behavior; low for host behavior)
- External standard text was not fetched. Assumption, not evidence: W3C Trace
  Context uses a `traceparent` header (`version-trace-id-parent-id-flags`), and
  OpenTelemetry gRPC instrumentation propagates that rather than
  `x-finfocus-trace-id`. — [ASSUMPTION; fetch skipped, host `opentelemetry.io`
  not on the safe list] (confidence: low)

## Data & Constraints

- Trace id rule in force: empty is accepted by `ValidateTraceID` (optional), but
  the interceptor treats empty as "generate". Non-empty must match 32 hex digits
  and must not be all zeros. Span ids have a separate 16-hex validator and are
  not propagated. — [source: `observability_validate.go` `ValidateTraceID`,
  `ValidateSpanID`] (confidence: high)
- Replacing a bad id (spec 008 FR-009) conflicts with a literal reading of issue
  #193's success line ("a host-generated trace ID" appears in logs and validation
  errors) whenever the host value is not already 32-hex. — [source: spec 008;
  issue body] (confidence: high)
- No new direct module is required to keep today's string propagation. Adopting
  the OpenTelemetry SDK would add a direct dependency the constitution tells this
  repo to keep small, and would need a benchmark story under section VIII.
  — [source: constitution IV and VIII; `go.mod`] (confidence: medium)
- Plugin RPCs are unary only, so a stream interceptor has nothing to wrap unless
  the proto gains streaming. — [source: `proto/finfocus/v1/`] (confidence: high)
- Labels say `effort/large`. The shipped gRPC string path is not large. What
  would be large is full context propagation (span, Connect, host client, both
  SDKs, error payloads) and that size is not measured here. — [source: issue
  labels; code inventory] (confidence: medium)

## Evidence Against the Idea

- The technical approach named in the issue is largely already shipped, under a
  different header, and was shipped before the issue existed. Building
  "a TracingUnaryServerInterceptor" again would duplicate `logging.go` and spec
  005/008. — [source: git history `6d5b5ac`, `dd410cd`; current `logging.go`]
  (confidence: high)
- The proposed key `x-pulumicost-trace-parent` was never the implementation, even
  under the PulumiCost name (`x-pulumicost-trace-id`). Reintroducing a pulumicost
  header would fight the FinFocus rename and the `FINFOCUS_*` fallback pattern.
  — [source: `git show eecdddb^`; `env.go` fallback comments] (confidence: high)
- The issue is `roadmap/future`, unassigned, and has no human discussion. Later
  specs (051, 052) explicitly refused Connect-mode tracing as out of scope.
  — [source: issue labels; spec 051 research] (confidence: high)
- Auto-attaching trace ids to every validation error changes a stable error type
  and string format (`ValidationError.Error`) that spec 047 just settled. Spec
  047 does not mention trace ids. — [source: `validation_error.go`;
  `specs/047-validation-error-integration/spec.md`] (confidence: high)
- Putting OpenTelemetry spans inside cost RPCs, as the guide sketches, is the
  nearest path to violating the issue's own ban on measuring provider billing
  APIs, and it is not required to carry an id. — [source: issue anti-guess
  paragraph; `OBSERVABILITY_GUIDE.md`] (confidence: medium)
- If the real goal is host-side export and sampling, this repository is the wrong
  place for the tracer provider. Constitution IV puts application behavior in
  finfocus-core. This repo was not checked against that other repository.
  — [source: constitution IV] (confidence: medium)

## Gaps & Open Questions

- [NEEDS CLARIFICATION: is the remaining job "make the host id show up in zerolog
  and validation errors", or "propagate a full OpenTelemetry context
  (traceparent, span id, tracestate)"?]
- [NEEDS CLARIFICATION: what header and value does the host send today, if any?
  This repo does not contain the host.]
- [NEEDS CLARIFICATION: must Connect mode carry the same id, given specs 051 and
  052 left that out of scope on purpose?]
- [NEEDS CLARIFICATION: if the incoming value is not 32-hex, should the SDK keep
  the host value (issue success text) or replace it (spec 008)?]
- [NEEDS CLARIFICATION: should `ValidationError` grow a trace field, should
  `ErrorDetails.correlation_id` be filled, or is a log field enough?]
- [NEEDS CLARIFICATION: is a direct OpenTelemetry dependency acceptable?]
- [NEEDS CLARIFICATION: should the TypeScript client inject the same header
  (constitution XIII) even though the host application is not this repo?]
- [NEEDS CLARIFICATION: is `roadmap/future` still the intended priority?]

## Sources

- [finfocus-spec issue 193](https://github.com/rshade/finfocus-spec/issues/193) (host: github.com, policy:
  allowlisted; fetched with `gh issue view`, no further pages)
- Repository files and git history cited inline (local read, not a URL)
- [UNVERIFIED — fetch skipped: host not on safe list: opentelemetry.io] W3C Trace
  Context and OpenTelemetry gRPC propagator docs were not fetched. Related claims
  are marked ASSUMPTION.
- CodeRabbit links to `rshade/pulumicost-spec` issues #75, #80, #94, #85 and
  associated PRs were not fetched (no crawl of links inside the issue).
