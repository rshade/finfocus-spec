# Data Model: Capability-Scoped Supports Requests

## SupportsRequest (changed)

| Field | Number | Type | Rule |
|-------|--------|------|------|
| `resource` | 1 | `ResourceDescriptor` | Unchanged. Required by the SDK server and the client. |
| `capability` | 2 | `PluginCapability` | New and optional. `UNSPECIFIED` (0, the default) means any capability. |

Validation:

- Every `capability` value is accepted on the wire and by `plugintesting.ValidateSupportsRequest`, including
  values the receiver does not define (research R5).
- `pluginsdk.RequestedCapability` maps any value outside `IsValidCapability` other than `UNSPECIFIED` to
  `UNSPECIFIED`, and logs a warning once per distinct value per process, for at most 64 distinct values
  (research R2).

Wire compatibility:

| Host | Plugin | Result |
|------|--------|--------|
| old (no field) | new | Plugin reads `UNSPECIFIED` and answers as before. |
| new, `UNSPECIFIED` | old | Identical bytes; the zero value is not serialized. |
| new, specific value | old | The old plugin skips the unknown field and answers as before (the "any" answer). |
| new, value newer than the plugin's SDK | new | The helper returns `UNSPECIFIED`, logs a warning, and the plugin answers for any capability. |

## SupportsResponse (unchanged)

The response is unchanged. It does not echo `capability`. Its existing `capabilities` and `capabilities_enum`
fields still describe what the plugin *has*, not the question asked.

## PluginCapability (unchanged, reused)

No values are added. The values hosts are expected to send (the resource-scoped set) are:

`PROJECTED_COSTS` (1), `ACTUAL_COSTS` (2), `CARBON` (3), `RECOMMENDATIONS` (4), `ENERGY` (7), `WATER` (8),
`PRICING_SPEC` (9), `ESTIMATE_COST` (10), `BATCH_COST` (12), and `HISTORICAL_CARBON` (20).

Other values are valid in a request, and the plugin decides how to answer them.

## MockPlugin (changed, test reference)

| Field | Type | Default | Rule |
|-------|------|---------|------|
| `UnsupportedCapabilities` | `[]pbc.PluginCapability` | nil | A capability in the list answers `Supported: false` after the provider and resource checks pass. nil means the capability is ignored. |

## Conformance check (new)

`RPCCorrectness_SupportsCapability` is a Standard check in `CategoryRPCCorrectness`. It sends, in order:

1. An unspecified request (the same request as `RPCCorrectness_SupportsRPC`).
2. One request per capability in the set, where the set is the declared capabilities that are in the
   resource-scoped list. A plugin that declares nothing gets `{PROJECTED_COSTS, ACTUAL_COSTS}`. A plugin that
   declares only capabilities outside the list sends nothing further, and passes on step 1.

A request fails the check when the RPC returns an error or the response fails `ValidateSupportsResponse`. The
failure names the capability.
