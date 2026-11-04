# Contract: SDK Log Levels for Handler Errors

Applies to the unary tracing interceptor (`rpc failed`) and the wrappers for `Supports`, `DryRun`,
`GetRecommendations`, `GetRecommendationHistory`, `GetBudgets`, `DismissRecommendation`, `BatchCost`, `ResolveResourceTypes`,
and provider `GetPluginInfo` (`<RPC> handler error`).

| Handler returns | Status the host sees | SDK log level |
| --- | --- | --- |
| `InvalidArgument` | `InvalidArgument` (unchanged) | debug |
| `NotFound` | `NotFound` (unchanged) | debug |
| `FailedPrecondition` | `FailedPrecondition` (unchanged) | debug |
| `Unimplemented` | SDK default (unchanged) | debug (fallback entry; interceptor entry only where the call still fails) |
| `Internal`, `Unavailable`, `DataLoss`, other codes | passed through (unchanged) | error |
| `Unknown`, plain error | `Internal` with generic message (unchanged) | error |

Each entry keeps `trace_id`, `operation` (interceptor), and the `error` field. A plugin picks this up
by rebuilding on the new SDK; an older build keeps logging at ERROR.
