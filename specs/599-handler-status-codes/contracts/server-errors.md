# Contract: Handler Errors in pluginsdk.Server

Applies to `DryRun`, `Supports`, `GetRecommendations`, `GetBudgets`, `DismissRecommendation`,
`BatchCost` (custom `BatchCostHandler`), `ResolveResourceTypes`, and `GetPluginInfo`
(`PluginInfoProvider`), over gRPC and Connect.

| Handler returns | Client receives |
| --- | --- |
| gRPC status, code not `OK`/`Unknown`/`Unimplemented` (incl. wrapped, or a `GRPCStatus()` type) | that code and message |
| `Unimplemented` status | the RPC's not-a-provider answer (below) |
| plain error, context error, `Unknown` status | `Internal` with today's generic message |

| RPC | Not-a-provider answer |
| --- | --- |
| DryRun | `Unimplemented` "plugin does not support DryRun" |
| Supports | `Supported: false`, `DefaultSupportsNotImplementedReason` (registry check still applies first) |
| GetRecommendations | empty list, summary with the request's projection period |
| GetBudgets | `Unimplemented` "plugin does not support GetBudgets" |
| DismissRecommendation | `Unimplemented` "plugin does not support DismissRecommendation" |
| BatchCost | SDK per-resource batch (`batchCostFallback`) |
| ResolveResourceTypes | `TypeRegistry` result when configured, else empty response |
| GetPluginInfo | configured `ServeConfig.PluginInfo`, else `Unimplemented` "GetPluginInfo not implemented" |

Unchanged: nil-response and invalid-response guards stay `Internal`; the cost RPCs and optional
services already return handler errors as is. No exported identifier is added or changed.
