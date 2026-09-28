# Contract: Failure Trace Id

This is the caller-visible contract for an ordinary served call
(`serveGRPC`). The web serving mode is not a party to it. Field rules live in
[data-model.md](../data-model.md). This file is the shape callers can assert.

No protobuf file changes. No new RPC.

## Header

| Item | Contract |
|------|----------|
| Name | `x-finfocus-trace-id` |
| Cardinality | First value wins. Further values are ignored. |
| Accepted | 32 lowercase hex characters, not all zeros. |
| Rejected | Replaced with a generated id when generation works. The rejected bytes are not echoed. |
| Absent | Same as rejected. |

The interceptor does not read `x-pulumicost-trace-parent`, `traceparent`, or
any other trace header.

## Failure log

Emitted by `TracingUnaryServerInterceptorWithLogger` for a non-nil handler
error when the stored id is non-empty. Not emitted on success. Not emitted
when the stored id is empty.

The event MUST contain:

| Field | Value |
|-------|-------|
| message | `rpc failed` |
| `trace_id` | Stored call id |
| `operation` | Full method, such as `/finfocus.v1.CostSourceService/GetActualCost`, or empty |
| `error` | The handler error's text after stamping |

It MUST NOT contain the rejected header value.

`serveGRPC` constructs the interceptor with the plugin logger, so the event
uses that logger's writer and level. A logger that drops error-level events
drops this one. The stamp does not wait for the write.

`TracingUnaryServerInterceptor()` is the same function with a discarded
logger. Stamping still follows the validation section. No event is emitted.

## Validation failure text

`(*ValidationError).Error()`:

```text
{FieldName}: {Constraint} (actual: {ActualValue}, expected: {ExpectedValue})
```

When `TraceID` is non-empty, the text is that sentence plus:

```text
 trace_id={TraceID}
```

The leading character of the addition is a single space. Example with an id:

```text
effective_cost: must not exceed billed_cost (actual: 150.00, expected: <= 100.00) trace_id=abcdef1234567890abcdef1234567890
```

Example without an id (the sentence spec 047 already required):

```text
effective_cost: must not exceed billed_cost (actual: 150.00, expected: <= 100.00)
```

Stamp, performed on the handler's returned error before the failure log:

| Handler error | `TraceID` after the call | Failure log `trace_id` |
|---------------|--------------------------|------------------------|
| `*ValidationError` with empty `TraceID` | Stored call id | Stored call id |
| `*ValidationError` with `TraceID` already set | Handler's id, unchanged | Stored call id |
| `*ValidationError` wrapped so `errors.As` finds it, field empty | Stored call id on that value | Stored call id |
| Any other non-nil error | unchanged (no field) | Stored call id |
| nil | — | no failure log |

The returned error is the same value the handler returned. `Unwrap` is
unchanged. A non-validation error's text gains no suffix.

A validation failure built with `NewValidationError` or
`NewValidationErrorWithCause` starts with an empty `TraceID`. Outside the
interceptor, `Error()` has no suffix unless the caller set the field.

## Author logger

```text
logger = WithTrace(ctx, logger)
```

| Input | Output |
|-------|--------|
| Context holds a non-empty trace id | Child logger whose `trace_id` is that id. One field. |
| Context holds no id | The input logger, unchanged. |

Callers MUST NOT also set `FieldTraceID` on the child. A second set writes a
second key and leaves the contract (one id per record).

`WithTrace` does not read the header itself. It reads the id the interceptor
stored. On the web serving mode the context has no such id unless the handler
stored one itself, and `WithTrace` then has nothing to copy.

## Non-behavior

- No change to RPC request or response messages.
- No new status code. Stamping is not a new failure.
- No Connect interceptor and no Connect failure log from this contract.
- No OpenTelemetry span, exporter, or provider billing instrumentation.
- No second metadata key.
- `ServeConfig.UnaryInterceptors` still run after this interceptor on gRPC
  only. They see the stored context id. They are not required to log it.
