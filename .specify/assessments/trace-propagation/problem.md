# Problem Definition: Correlating plugin failures to the host request

- **Slug**: trace-propagation
- **Created**: 2026-09-27
- **Inputs used**: intake.md, research.md

## Problem Statement

When a FinFocus plugin fails or rejects input, the person debugging that failure
often cannot tie the plugin's log line or the SDK's validation error back to the
host request that caused it. On the gRPC path a valid host trace id can already
sit on the handler context, but nothing forces it into the log or the error. On
the Connect path the incoming id is dropped. An id that is not already in the
SDK's 32-hex form is replaced, so the value the host generated is not the value
later seen. The issue that raised this does not cite an outage; the pain is
diagnostic, and how often it happens in production is unknown.

## Affected Users & Stakeholders

- **Users**: Plugin authors who log with zerolog and return SDK validation
  errors. They can read a trace id from context only if they remember to log it.
  Validation errors they return do not carry that id. — [source: research.md
  prior art on logging and `ValidationError`]
- **Users**: Operators of a FinFocus host who need one id across the host and the
  plugin when a cost, usage, or allocation call fails. What header the host
  sends today is not in this repository. — [source: research.md gaps]
- **Stakeholders**: finfocus-spec maintainers (issue author rshade). They decide
  whether a `roadmap/future` item is still in scope. — [source: intake.md]
- **Stakeholders**: finfocus-core (the host application). Constitution IV puts
  host behavior there, not in this repo. That repo was not inspected. — [source:
  research.md constitution note]

## Goals

- A failure handled for a host request can be matched to that request by the
  same trace id in the plugin log and in the SDK validation error, without the
  plugin author inventing a second identifier.
- The id in those two places is the host's id when the host sent a value the SDK
  already accepts as a trace id.
- Correlation does not depend on measuring or timing the cloud provider's billing
  APIs. That boundary is an intake constraint, not a metric.

## Non-Goals

- Timing, tracing, or otherwise measuring the cloud provider's own billing APIs.
  — [source: intake anti-guess boundary]
- Building the host application, sampling, or a trace exporter. Those are outside
  this repository's role. — [source: constitution IV, as cited in research.md]
- Re-deciding pricing math, billing modes, or FOCUS field validation. Trace
  correlation is not a cost-calculation problem. — [source: constitution III,
  implied by the idea's diagnostic scope; not a new claim about pricing bugs]
- Treating "add the interceptor" as the problem. That mechanism is already on
  the gRPC server. The unsolved part is end-to-end visibility when logs, errors,
  or the other transport do not show the host id. — [source: research.md]

## Success Metrics

- For a gRPC call whose incoming trace id is valid under the current 32-hex
  rule, the plugin zerolog line for that call contains that same id.
  Baseline: not automatic. The logger does not add the field; only handlers that
  copy it do. No count of plugins that copy it. — [source: research.md]
- For that same call, an SDK validation error produced while handling it carries
  the same id in a stable, machine-readable place (not only inside a free-text
  sentence that callers must scrape). Baseline: `ValidationError` has no trace
  field, and its `Error()` string has none. — [source: research.md]
- For that same call, the id is not replaced by a newly generated one.
  Baseline: true on gRPC when the header is `x-finfocus-trace-id` and the value
  is valid 32-hex; false on Connect; false when the value fails validation
  (it is replaced). — [source: research.md]
- Qualitative: a debugger does not need a second, process-wide env var to
  recover the per-request id. Baseline: `FINFOCUS_TRACE_ID` and the per-request
  context are separate and are not joined. — [source: research.md] (qualitative)

No production rate, latency budget, or incident count is in the research. Do not
treat the issue's `effort/large` label as a measured size.

## Cost of Inaction

gRPC handlers that already copy the context id into zerolog keep working, and
valid ids on `x-finfocus-trace-id` keep arriving on context. Gaps that stay:
Connect calls are uncorrelated; plugins that forget the log field stay
uncorrelated; validation errors stay uncorrelated; a host id that is not 32-hex
is discarded and a new one is logged instead, so the host and the plugin disagree.
There is no cited incident that says this is happening now. The cost is ongoing
manual logging discipline and a diagnostic hole whose frequency is unknown. Later
specs already chose to leave Connect tracing alone, so inaction is also the
current documented behavior for that transport.

## Open Questions

- [NEEDS CLARIFICATION: is the hurt "logs and validation errors omit an id the
  context already has", or "the host's real trace context never arrives" because
  the header or the id format does not match?]
- [NEEDS CLARIFICATION: what does the host send today?]
- [NEEDS CLARIFICATION: are Connect failures in scope, given later specs left
  that transport without tracing on purpose?]
- [NEEDS CLARIFICATION: if the host id is not 32-hex, is success "keep the host
  value" or "replace it", which is what the SDK does now?]
- [NEEDS CLARIFICATION: where must the id appear on a validation error — the
  Go error value, a proto field, or a log line only?]
- [NEEDS CLARIFICATION: is a direct OpenTelemetry dependency in bounds?]
- [NEEDS CLARIFICATION: must the TypeScript client participate, or only the Go
  plugin process?]
- [NEEDS CLARIFICATION: is this still wanted, given `roadmap/future` and no human
  discussion since 2025-12-22?]
