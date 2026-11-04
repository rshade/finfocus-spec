# Research: Plugin SDK Decline Log Levels

## Decision 1: Which codes are declines

- **Decision**: `InvalidArgument`, `Unimplemented`, `NotFound`, `FailedPrecondition`.
- **Rationale**: The issue proposes them; the contract uses them for answers the host treats as normal.
- **Alternatives**: Only `InvalidArgument` and `Unimplemented` (the measured noise) leaves `NotFound`
  and `FailedPrecondition` noisy on the same pattern. Adding `Canceled` or `DeadlineExceeded` could
  hide real timeouts.

## Decision 2: Demoted level

- **Decision**: debug.
- **Rationale**: Info still emits a line per declined resource, which is the noise being removed.
  `logHandlerFallback` already logs the `Unimplemented` fallback at debug.

## Decision 3: Where the code comes from

- **Decision**: Interceptor: `status.Code(err)`. Wrappers: `status.Code(clientErr)`, the error
  `handlerStatus` already produced.
- **Rationale**: grpc v1.84 `status.Code` unwraps via `errors.As`, so wrapped statuses classify
  correctly. `clientErr` is already normalised: plain, `Unknown`, and `OK` errors became `Internal`,
  so the wrapper cannot misclassify them. A raw `ValidationError` has no `GRPCStatus`, reads as
  `Unknown`, and stays at ERROR as today.
- **Alternatives**: Re-running the `errors.As` walk in a second place risks the two sites disagreeing.

## Decision 4: Scope of call sites

- **Decision**: All nine `handlerStatus` wrappers plus the interceptor.
- **Rationale**: Each logs ERROR after `handlerStatus` returns a real status, so each doubles with
  the interceptor. Their `Unimplemented` already takes the debug fallback, so only the other three
  decline codes change there.

## Decision 5: Interceptor condition

- **Decision**: Keep the existing `traceID != ""` guard; change only the level.
- **Rationale**: Unrelated to this issue; changing it alters log volume in the other direction.

## Decision 6: Log message

- **Decision**: Keep the messages `rpc failed` and `<RPC> handler error` for faults. A decline logs
  the same message at debug, so existing search strings still match at that level.
