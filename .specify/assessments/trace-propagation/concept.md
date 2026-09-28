# Concept: Correlating plugin failures to the host request

- **Slug**: trace-propagation
- **Created**: 2026-09-27
- **Recommended option**: none

No option clears the bar to specify. The approach named in issue #193 is already
in the gRPC server under a different header. The leftover gap is real but small,
and choosing it versus a full trace context is an unanswered product decision.
Option B is the default if that decision stays unanswered. Option A is the only
build that might be worth a later spec.

## Options

### Option A — Show the id the handler already has (smallest thing that could work)

- **Sketch**: On the gRPC path, a valid host trace id is already on the handler
  context. Plugin authors would stop having to copy it by hand: the log line for
  the call and the SDK validation error for that call would both carry that same
  id. Connect behavior, the header name, and the 32-hex acceptance rule stay as
  they are. No new tracing system. Authors who already log the field should not
  see a second, conflicting id.
- **Appetite**: small (days). The insertion points are known (logger, validation
  error, existing context helper). Uncertainty: a benchmark expectation under
  constitution VIII could stretch this if the change is treated as new core SDK
  logic. Not weeks unless the error contract change grows.
- **Trade-offs**: Wins the two concrete success metrics for gRPC calls that
  already send a valid `x-finfocus-trace-id`. Sacrifices Connect, span id, and
  any host that does not send that header or that exact format. Risks a visible
  change to validation error text that spec 047 just stabilized, and double
  fields if authors keep adding the id themselves. Does not meet the issue text
  if the host's real id is a trace-parent value the current rule replaces.
- **Rabbit holes**: Changing proto error messages "while we are here"; turning
  the logger into a general context-propagation framework; "fixing" the example
  that logs a non-32-hex id by relaxing validation (that fights spec 008).

### Option B — Do nothing; treat the named approach as already shipped

- **Sketch**: Leave propagation as it is. Authors keep copying the context id
  into zerolog, as the README and the structured-logging example already show.
  Validation errors stay without a trace id. Connect stays without the
  interceptor, which specs 051 and 052 already accepted. The issue's
  `x-pulumicost-trace-parent` key is not added. Optional doc hygiene only: the
  observability guide's OpenTelemetry sample is not what the SDK does, and
  leaving it unmarked invites the wrong build. That hygiene is not a feature.
- **Appetite**: small if limited to a doc correction; zero if the issue is
  closed as superseded with no doc edit. Not the issue's `effort/large` label.
- **Trade-offs**: Wins by not churning a stable error type or reopening Connect
  tracing. Sacrifices the issue's success line: a host id is not automatically
  in the log or in the validation error. The cost of that sacrifice is unmeasured
  (no incident, no count of plugins that forget the field). Risk: the guide and
  the issue keep describing a system this repo does not implement.
- **Rabbit holes**: A "docs only" pass that becomes an OpenTelemetry design; or
  closing the issue and losing the residual log-and-error gap with no successor
  note.

### Option C — Carry the host's full trace context on both transports

- **Sketch**: Whatever context the host actually uses would survive into the
  plugin on gRPC and on Connect, and would show up in the plugin log and in
  validation errors. That includes span identity if the host sends it, not only
  a 32-hex string. The plugin still would not time or trace the cloud provider's
  billing APIs. The host application itself stays in finfocus-core; this repo
  would only define what the plugin side must accept and echo. A matching
  TypeScript client behavior is in play only because constitution XIII ties the
  SDKs together when the contract changes.
- **Appetite**: uncertain, and not honestly "small". The issue says
  `effort/large`. Research shows the gRPC string path already exists, so the
  delta is the other transport, a second id (span), header compatibility, and
  possibly a direct OpenTelemetry dependency. That is medium (weeks) if it stays
  a string contract, and large (months) if it becomes a real tracer, exporter,
  and sampling story. Do not treat the label as a measurement.
- **Trade-offs**: Only this option can satisfy "OpenTelemetry contexts" and
  Connect together. It spends the most, fights the later decision to leave
  Connect interceptors alone, and pressures constitution IV (minimal
  dependencies) and VIII (benchmarks). The W3C header shape was not fetched, so
  the sketch may not match what libraries do today. Risk of violating the
  anti-guess boundary if spans get wrapped around provider calls.
- **Rabbit holes**: Adopting the OpenTelemetry SDK; sampling and exporters;
  changing spec 008's "replace invalid ids" rule; a pulumicost-named header
  "for compatibility"; instrumenting provider HTTP clients; doing the host's
  tracer setup inside this repo.

## Recommendation

Do not proceed. None of the options should be handed to specify yet.

Option B is the standing behavior and is coherent: the interceptor the issue
asks to implement has been on by default since 2025-11-24, the key was never
`x-pulumicost-trace-parent`, and later specs refused Connect tracing on purpose.
Option A is a real, small improvement to the success metrics, but only if the
problem is "the context id is invisible in logs and errors", not "the host's
trace context never arrives". That fork is still open, and there is no cited
incident that says A beats B. Option C is the issue title, and it is the option
most likely to blow the constitution and the anti-guess boundary. Recommending it
would invent a host contract this assessment did not see.

## Out of Scope (for the recommended option)

Recommended option is none, so nothing new is in scope. These stay excluded even
if a later pass picks A or C:

- Measuring or instrumenting the cloud provider's billing APIs.
- Building the host process, a trace exporter, or sampling. That is finfocus-core
  or an operations concern, not this spec repo.
- Re-implementing the existing gRPC trace-id interceptor as if it were missing.
- Reintroducing a `x-pulumicost-*` metadata key. The current key is
  `x-finfocus-trace-id`; the legacy env fallback is `PULUMICOST_TRACE_ID` to
  `FINFOCUS_TRACE_ID`, which is a different channel.
- Pricing math, FOCUS validation rules, and billing modes.

## Assumptions to Validate

- The host either already sends `x-finfocus-trace-id` as 32 lowercase hex, or it
  does not. Option A is only a solution in the first case. Not checked outside
  this repo.
- "Included in SDK validation error responses" means the Go `ValidationError`
  (and not a proto `ErrorDetail` or `correlation_id`). The issue does not say.
- Plugins that already set `trace_id` on the logger are common enough that an
  automatic field must not duplicate or override them. No census exists.
- A direct OpenTelemetry dependency is not wanted for Option A, and may not be
  wanted at all. Constitution IV suggests that; nobody has confirmed it for this
  issue.
- `roadmap/future` still means "not now". The issue is open and unlabeled as
  cancelled. Updated 2026-02-28 for an unknown reason.
- Spec 008's replace-on-invalid rule remains the rule. The issue's success line
  can be read as "always keep the host value", which contradicts it.
