# Research: Trace Id on Plugin Failures

The assessment at `.specify/assessments/trace-propagation/` stopped at
needs-clarification and recommended no concept. The parent then closed those
questions. Each decision below is that closure, checked against the code that
merged in pull request 538 (`a5e6790`). None of them is reopened here.

## Decision: Specify the shipped failure record, not a new tracer

- **Decision**: The design is the behavior on main. `serveGRPC` already stores
  a valid host trace id. On failure it logs that id and stamps an empty
  `ValidationError.TraceID`. This spec records that slice. It does not rebuild
  `TracingUnaryServerInterceptor` and it does not implement issue #193's
  original "new interceptor plus trace-parent" wording.
- **Rationale**: The interceptor has been the gRPC default since 2025-11-24.
  The metadata key is `x-finfocus-trace-id` (`TraceIDMetadataKey`), formerly
  `x-pulumicost-trace-id`, never `x-pulumicost-trace-parent`. Re-implementing
  it would churn a path spec 008 already locked. The gap the parent accepted
  was visibility: the id was on the context and not on the failure log or the
  validation error. That is what `a5e6790` changed.
- **Alternatives considered**: Do nothing and close the issue as already
  shipped (assessment option B). Rejected by the parent: the success line is
  the id in the plugin log and in the validation error, and those two were
  missing. Carry the host's full trace context on gRPC and Connect (option C).
  Rejected: span id, a second header, and Connect were not chosen, and a
  tracing toolkit fights constitution IV.

## Decision: Keep spec 008's replace-on-invalid rule

- **Decision**: `TracingUnaryServerInterceptorWithLogger` still reads the first
  `x-finfocus-trace-id` value. If it is empty or `pricing.ValidateTraceID`
  returns an error, the call stores `GenerateTraceID()` instead. Valid means
  `^[0-9a-f]{32}$` and not 32 zeros. Uppercase hex is invalid. If generation
  fails, the stored id is empty and the call continues.
- **Rationale**: Spec 008 replaced bad ids so logs cannot be injected with
  control characters or oversized values. The issue text can be read as "keep
  whatever the host sent." That reading contradicts spec 008. The parent kept
  the replace rule. A rejected value must not appear in the failure log or on
  the validation error, or the injection bound is gone.
- **Alternatives considered**: Keep any non-empty host value (rejected: fights
  spec 008 and the log-injection goal). Accept W3C `traceparent` in this slice
  (rejected: different header, not what the interceptor reads, and the parent
  forbade a new header). Treat generation failure as a failed RPC (rejected:
  the existing comment and code proceed with an empty id so a randomness
  failure does not take down the call).

## Decision: Log every failed gRPC call; stamp only an empty validation id

- **Decision**: After the handler returns, if `err != nil` and the stored id is
  non-empty, the interceptor logs at error level with `trace_id` set to the
  stored id, `operation` set to `info.FullMethod` (empty when `info` is nil or
  the method is empty), the error, and message `rpc failed`. Separately,
  `stampValidationTrace` sets `TraceID` only on the first `*ValidationError`
  in the chain whose `TraceID` is empty. Other error types are logged and not
  rewritten. A nil error writes nothing.
- **Rationale**: The success line asks for the id on the plugin log and in the
  SDK validation error. Logging only validation failures would hide transport
  and handler failures that are just as hard to correlate. Stamping every
  error type would change error text spec 047 stabilized for non-validation
  failures. The stamp runs before the log so a filled id is what `Error()`
  contributes to the log's error string.
- **Alternatives considered**: Log only `*ValidationError` (rejected: the
  success line's log half is "a failed call," and the code logs any non-nil
  error). Append the id to every error string (rejected: FR-008 limits the
  suffix to validation failures, and wrapping would add a prefix). Skip the
  log when the handler already logged (rejected: the interceptor cannot see
  that, and one structured failure line is the correlation point).

## Decision: Mutate an empty TraceID; never overwrite the handler

- **Decision**: `stampValidationTrace` uses `errors.As` and assigns
  `ve.TraceID = traceID` only when that field is empty. The returned error
  value is the handler's value, not a wrapper. A handler-set id is kept even
  when it differs from the stored call id. The failure log still uses the
  stored call id.
- **Rationale**: `Error()` reads the field. Filling it in place makes the
  suffix appear without changing the concrete type, `Unwrap`, or `errors.Is`.
  Overwriting a handler id would punish a plugin that already correlated the
  failure with a different id. The log stays on the call's id so an operator
  can still join the host request when the handler picked another one.
- **Alternatives considered**: Wrap with `fmt.Errorf` (rejected: adds text
  FR-008 does not allow, and hides the type unless `%w` is used carefully).
  Always force the stored id (rejected: the parent said an empty field is the
  only one that gets stamped). Clone the error before mutating (rejected: not
  what shipped; the handler allocated the value, and a clone could drop
  unexported `err` or change identity). The unexported cause field is left
  alone so `Unwrap` stays stable.

## Decision: Suffix the text only when the field is set

- **Decision**: `(*ValidationError).Error()` stays
  `{FieldName}: {Constraint} (actual: {ActualValue}, expected: {ExpectedValue})`.
  When `TraceID != ""`, it appends one space and `trace_id=<id>`. When the
  field is empty, the string is identical to the pre-538 sentence, including
  for failures built outside a traced call.
- **Rationale**: Callers and tests match that sentence. A unconditional suffix
  would change every validation failure, including unit tests that never enter
  the interceptor. Gating on the field keeps spec 047's text for the common
  case and still satisfies "included in the SDK validation error response"
  once the interceptor has stamped an id.
- **Alternatives considered**: A structured gRPC status detail or a proto
  `correlation_id` (rejected: message-contract change, constitution I and VI,
  and the parent forbade protobuf edits). Put the id in a new `Error()`
  sentence even when empty, as `trace_id=` (rejected: SC-003 requires zero
  occurrences when the field is empty). Log the id only and leave `Error()`
  untouched (rejected: the success line includes the validation error text).

## Decision: WithTrace copies the id once per request

- **Decision**: `WithTrace(ctx, logger)` returns `logger` unchanged when
  `TraceIDFromContext` is empty. Otherwise it returns a child logger with
  `FieldTraceID` (`trace_id`) set from that context. Callers do that once.
  They must not also call `Str(FieldTraceID, ...)` on the child. Zerolog
  writes a second key; it does not replace the first.
- **Rationale**: Author logs for successful calls and intermediate steps are
  outside the failure interceptor. One helper avoids each plugin formatting
  the id by hand. A second `Str` looks like two ids. The README example was
  updated to call `WithTrace` and to drop the extra `FieldTraceID`. The
  no-id path must not invent an empty field, or success logs grow a blank id
  the acceptance rule would have rejected.
- **Alternatives considered**: A logger middleware that always injects the
  field inside `Serve` for handler code (rejected: handlers receive the
  context, not a per-request logger, and `ServeConfig.Logger` is process-wide).
  Make `WithTrace` overwrite an existing field (rejected: zerolog's `With`
  does not deduplicate, and teaching a special case hides the double-write).
  Require authors to keep setting `FieldTraceID` themselves (rejected: that is
  the miss the failure log was added to stop, and the guide would keep showing
  the duplicated pattern).

## Decision: Connect does not run the interceptor

- **Decision**: `serveConnect` stays free of `TracingUnaryServerInterceptor`.
  Only `serveGRPC` appends `TracingUnaryServerInterceptorWithLogger(server.logger)`.
  The public `TracingUnaryServerInterceptor()` delegates to that function with
  `zerolog.Nop()`, so a caller who chains the no-logger helper still stamps
  validation errors and discards the log line.
- **Rationale**: Spec 051 recorded that Connect cost handlers have no
  interceptors, and that adding them would change existing RPC behavior.
  Spec 052 registered the allocator on Connect the same way: gRPC gets the
  tracing interceptor; Connect does not. The parent said to keep that as a
  non-goal. `server.logger` is `ServeConfig.Logger` or `newDefaultLogger()`
  (info level), so an error line is visible unless the operator raises the
  level past error. Stamping does not depend on the line being emitted.
- **Alternatives considered**: Add a Connect interceptor in this spec
  (rejected: the parent called that a non-goal, not a bug). Use `Nop` inside
  `Serve` (rejected: then operators would not see `rpc failed` on the plugin
  logger, which is the success line). Log with a new logger that ignores
  `ServeConfig.Logger` (rejected: plugins already choose the writer and level
  on that logger).

## Decision: No new header, toolkit, provider spans, or protobuf

- **Decision**: Do not read or write `x-pulumicost-trace-parent`. Do not add a
  module that imports OpenTelemetry. Do not wrap provider billing HTTP
  clients. Do not edit `proto/` or regenerate bindings. Do not add a
  TypeScript client for this log line.
- **Rationale**: Constitution IV keeps the spec free of host-application
  stacks. Constitution I and XIII move only when the service definition
  changes. Provider timing is an anti-guess the assessment called out and the
  parent repeated. The legacy env fallback (`PULUMICOST_TRACE_ID` then
  `FINFOCUS_TRACE_ID`) is a different channel and is not this header.
- **Alternatives considered**: Depend on OpenTelemetry's propagator and accept
  `traceparent` (rejected: new dependency, different grammar than 32-hex, and
  not the header the host path already uses). Add an optional span id beside
  the trace id (rejected: option C, not chosen). Echo the id in a proto field
  so TypeScript sees it (rejected: wire change for a log-and-error-text
  behavior).

## Decision: No new benchmark

- **Decision**: Do not add a benchmark for the stamp or the failure log.
- **Rationale**: Constitution VIII asks for benchmarks of new core SDK logic
  because cost queries move large payloads. The success path does not log and
  does not stamp. The failure path is one type assertion walk and one log
  the plugin was already going to pay for when it logged the error itself.
  A sub-microsecond benchmark would be noisier than the CI note in this repo
  already warns about, and it would not guard FR-004.
- **Alternatives considered**: Benchmark `Error()` with and without a suffix
  (rejected: string formatting of one short sentence is not the hot path, and
  the parent said not to invent work the merged code does not contain).
  Require a zero-allocation stamp (rejected: `errors.As` and the log event
  allocate; pretending otherwise would fight zerolog).

## Decision: Docs that shipped are sufficient

- **Decision**: The plugin logging guide in `sdk/go/pluginsdk/README.md` is the
  FR-009 guide: `WithTrace` once, do not also set `FieldTraceID`, the failure
  log and the empty-`TraceID` stamp, and Connect not running the interceptor.
  `sdk/go/CLAUDE.md` repeats the boundary for maintainers. Do not edit
  historical specs, the assessment, or unrelated observability samples.
- **Rationale**: Constitution VII and XIV. The README example is what authors
  copy. A wider doc sweep was the rabbit hole assessment option B warned
  about. The parent limited this PR to the design record plus any real code
  gap. The guide already matches the code.
- **Alternatives considered**: Rewrite the observability guide's OpenTelemetry
  sample (rejected: out of scope; that sample is not this interceptor). Add a
  new top-level doc (rejected: the README section is the guide FR-009 names).
