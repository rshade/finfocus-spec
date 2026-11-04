# Data Model: Plugin SDK Decline Log Levels

No stored data. One pure classification.

## Classification

| Input | Class | Level |
| --- | --- | --- |
| `InvalidArgument`, `Unimplemented`, `NotFound`, `FailedPrecondition` | decline | debug |
| `Internal`, `Unknown`, `Unavailable`, `DataLoss` | fault | error |
| any other non-OK code | fault | error |
| error without a status | fault (reads as `Unknown`) | error |

`OK` never reaches the classifier because it runs only when an error is non-nil.

## Shape

- `declined(code codes.Code) bool`: true for the four decline codes. Switch on the code; no
  allocation.
- `handlerLogLevel(err error) zerolog.Level`: `DebugLevel` when `declined(status.Code(err))`,
  else `ErrorLevel`.
- Call sites use `s.logger.WithLevel(handlerLogLevel(x))` in place of `s.logger.Error()`, keeping
  their fields and messages.
