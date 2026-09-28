# Contract: Plugin Information Acceptance Note

No change to `GetPluginInfoRequest` or `GetPluginInfoResponse` fields. The note uses
metadata field 5, which already exists.

## Key

| Item | Value |
| --- | --- |
| Constant | `MetadataSupportsPerRequestCredentials` |
| Key | `supports_per_request_credentials` |
| Opted-in value | `true` (the existing `ValueTrue` string) |
| Not opted in | key omitted |

This key is not produced by `CapabilitiesToLegacyMetadata`. It is not a
`PluginCapability`. `IsValidCapability` bounds stay 1 through
`PLUGIN_CAPABILITY_ALLOCATION`.

## Who writes it

`Server.GetPluginInfo` writes it after legacy capability metadata is applied.

| Plugin | Note |
| --- | --- |
| Implements `PerRequestCredentialConsumer` | key set to `true`, replacing a provider or `WithMetadata` value |
| Does not implement it | key deleted if present |

Both the configured `PluginInfo` path and the `PluginInfoProvider` path do this. A
legacy plugin that returns `Unimplemented` from `GetPluginInfo` is unchanged. It has
not opted in, because opt-in is a method on the plugin value `Serve` was given.

## What hosts should do

Read the note before attaching credentials. Do not infer opt-in from the capability
list. Attaching credentials to a plugin that omitted the note does not retarget that
process's environment credentials. This SDK will not route the call elsewhere.

## What the note does not mean

- It does not mean the host is sharing a process across tenants. The host decides that.
- It does not mean the plugin implements a new RPC.
- It does not mean CORS `AllowCredentials`.
- It does not carry a secret. The value is the word `true`.
