# Data Model: Trace Id on Plugin Failures

No new stored type and no protobuf change. This page is the life of one trace
id from the host header to the failure log and the validation failure. Names
match [spec.md](./spec.md). Code names are the ones in [plan.md](./plan.md).

## Trace id

| Rule | Value |
|------|-------|
| Header | `x-finfocus-trace-id` (`TraceIDMetadataKey`). First metadata value only. |
| Valid | 32 characters, `[0-9a-f]`, and not `00000000000000000000000000000000`. |
| Invalid | Anything else, including uppercase hex, the wrong length, and all zeros. |
| Missing | No metadata, or the header absent or empty. Treated like invalid. |
| Stored | The valid host value, or `GenerateTraceID()` when the candidate is missing or invalid and generation succeeds. |
| Empty stored id | Generation failed. The call still runs. Context may carry an empty string. |
| Context | `ContextWithTraceID` / `TraceIDFromContext`. Not a protobuf field. |

A rejected candidate is dropped before the handler runs. It is not stored, so
it cannot appear in the failure log or on a validation failure.

## Validation failure

`ValidationError` fields that this slice cares about:

| Field | Rule |
|-------|------|
| FieldName, Constraint, ActualValue, ExpectedValue | Unchanged. They form the sentence in the contract. |
| TraceID | Empty unless a handler set it or the interceptor stamped it. Not a proto field. |
| cause (`err`, unexported) | Unchanged. `Unwrap` still returns it. |

Stamp rule, applied to the handler's error value, not to a copy:

1. If the error is nil, or the stored trace id is empty, do nothing.
2. Walk the chain with `errors.As` for `*ValidationError`.
3. If none is found, do nothing to the text.
4. If the first match has `TraceID == ""`, set it to the stored trace id.
5. If that field is already set, leave it, even when it differs from the
   stored id.

The id on the validation failure and the id in the failure log are the same
only in case 4. In case 5 they may differ. That is intentional.

Text rule for `Error()`:

- `TraceID == ""`: `{FieldName}: {Constraint} (actual: {ActualValue}, expected: {ExpectedValue})`
- otherwise: that sentence, then one space, then `trace_id=` and the field value

No other character is added. An empty field must not produce a trailing space
or a bare `trace_id=`.

## Failure log

One zerolog event, written only when the handler returned a non-nil error and
the stored trace id is non-empty.

| Field | Constant | Rule |
|-------|----------|------|
| message | — | `rpc failed` |
| level | error | Dropped if the logger's level is above error. The stamp still happened. |
| trace_id | `FieldTraceID` | The stored call id. Never the rejected header. Never the handler's id when they differ. |
| operation | `FieldOperation` | `UnaryServerInfo.FullMethod`, or empty when info is nil or the method is empty. |
| error | zerolog's error field | `err.Error()` after the stamp, so a stamped validation failure includes the suffix. |

The interceptor adds those fields to the logger it was given. Fields that
logger already had (timestamp, plugin name, plugin version) stay. Success
writes none of this event.

`serveGRPC` passes `server.logger`: `ServeConfig.Logger` when the host set
one, otherwise `newDefaultLogger()` at info level, which still emits error
events. `TracingUnaryServerInterceptor()` passes `zerolog.Nop()`: the stamp
still runs, and the event is discarded.

## Author log record

`WithTrace` does not write a failure log. It returns a logger:

| Context | Result |
|---------|--------|
| Stored id non-empty | Child logger with one `trace_id` field. Parent unchanged. |
| No id, or empty id | The same logger. No `trace_id` field is added. |

A later `Str(FieldTraceID, ...)` on that child appends a second `trace_id`
key. The model has one id per record. The second write is outside the model.
The logging guide forbids it.

## State transitions

```text
header candidate
  ├─ valid ───────────────────────────── store host id
  ├─ missing or invalid, generate ok ─── store generated id
  └─ missing or invalid, generate fails ─ store empty id
        │
        handler returns
        ├─ nil error ──────────────────── no stamp, no failure log
        ├─ error and stored id empty ──── no stamp, no failure log
        └─ error and stored id set
             ├─ validation failure, empty TraceID ─ stamp, then failure log
             ├─ validation failure, TraceID set ─── keep field, failure log
             └─ any other error ─────────────────── no text change, failure log
```

The web serving mode (`serveConnect`) does not enter this machine. A
validation failure created there, or in a unit test that never enters the
interceptor, keeps an empty `TraceID` unless the test set one.

## Relationships

- One ordinary served call stores one trace id.
- A failed call with a stored id has one failure log.
- A validation failure has at most one `TraceID`.
- The failure log's `trace_id` refers to the call, not to the validation
  failure's field. They match when the field was empty and got stamped.
- Optional gRPC services on the same server share this interceptor. Connect
  handlers for those services do not.

## What does not change

- `pricing.ValidateTraceID` and `GenerateTraceID`.
- Which header is read, and that only the first value counts.
- The validation failure sentence when `TraceID` is empty.
- `serveConnect`, Connect handler options, and `ServeConfig.UnaryInterceptors`
  on that path (they already do not apply there).
- Protobuf messages, including any correlation or error-detail field.
- Provider billing clients. They are not given this id by this slice.
- The legacy `PULUMICOST_TRACE_ID` / `FINFOCUS_TRACE_ID` environment fallback.
  That channel is not the header.
