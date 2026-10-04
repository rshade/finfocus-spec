# Research: Keep Handler Status Codes in the Plugin SDK Server

## R1. Classifying a handler error

- **Decision**: one unexported helper in `sdk/go/pluginsdk/sdk.go`:
  `handlerStatus(err error, internalMsg string) (clientErr error, useDefault bool)`.
  `status.FromError(err)`; not ok → `Internal` with `internalMsg`; `Unimplemented` → `useDefault`;
  `OK` or `Unknown` → `Internal` with `internalMsg`; any other code → `st.Err()`.
- **Rationale**: `status.FromError` already unwraps `%w` chains and custom `GRPCStatus()` types, so
  both pass through (spec edge cases). Returning `st.Err()` rather than the raw error gives the
  client the status's own message and keeps details, and `toConnectError` converts it for Connect
  with no change (FR-006). `Unknown` is what grpc-go reports for errors it could not classify, so it
  is treated like a plain error and the no-leak rule holds (FR-002).
- **Alternatives considered**: pass `Unimplemented` through (rejected by the maintainer: a
  stub-embedding plugin would lose SDK defaults for five RPCs); an allow-list of codes (more code,
  and a plugin's `Aborted` or `ResourceExhausted` is just as deliberate).

## R2. Not-a-provider paths as functions

- **Decision**: extract each RPC's existing not-a-provider branch into a small method
  (`supportsDefault`, `recommendationsDefault`, `budgetsUnsupported`, `dismissUnsupported`,
  `resolveDefault`, `pluginInfoDefault`, `dryRunUnsupported`) and call it from both the `!ok` branch
  and the `useDefault` branch. `BatchCost` already has `batchCostFallback`.
- **Rationale**: FR-003 says "exactly as if the plugin did not implement the interface"; sharing the
  code makes that true by construction (SC-002).

## R3. Logging

- **Decision**: the existing error-level log stays for pass-through and internal errors. A fallback
  logs one debug line ("handler returned Unimplemented; using SDK default").

## R4. Supports and Core routing

- `supportsDefault` returns `Supported: false` with `DefaultSupportsNotImplementedReason`.
  rshade/finfocus#1512 (Core v0.4.2) fails open on that reason, so stub-embedding plugins that used
  to get `Internal` (also fail-open) stay routed.

## R5. Capability inference

- Out of scope by maintainer decision. Recorded as a PR follow-up.
