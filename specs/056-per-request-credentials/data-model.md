# Data Model: Opt-In Per-Request Credentials

No protobuf message changes. No database. No file. The model is the in-memory value on
one call, the note on plugin information, and the headers that carry the value between
processes.

## Credential set

A `Credentials` value is an immutable set of name/value pairs for one call.

| Field | Rule |
| --- | --- |
| Entries | 1 to 16 pairs. Zero pairs is not a credential set. |
| Name | Compared after ASCII lowercasing. Must match `^[a-z][a-z0-9_-]{0,63}$`. Length is at most 64 characters. Duplicate names after lowercasing are rejected. |
| Value | Length 1 to 4096 bytes. Every byte is in the inclusive range 0x20 through 0x7E. |
| Total | The sum of value lengths is at most 16384 bytes. |
| Ownership | `NewCredentials` copies the caller's map. Later edits to that map do not change the set. There is no method that returns the map. |

The zero `Credentials` value means "no set". It is what `ExtractCredentials` returns
when nothing was attached. It is not a successful empty attachment.

### Validation states

| State | How it is recognized | Result |
| --- | --- | --- |
| Absent | No credential header on the call, and `WithCredentials` was not used with a valid set. | `ExtractCredentials` returns the zero value and a nil error. Not a failure. |
| Present | `NewCredentials` accepted the set, and this call's context carries it. | `ExtractCredentials` returns the set and a nil error. |
| Unusable at the host | `NewCredentials` rejected the map. | `ErrEmptyCredentials` or `ErrInvalidCredentials`. Nothing is attached. |
| Unusable on the wire | A credential header arrived but broke the rules above, including two values for one name. | The context records failure without keeping the bytes. `ExtractCredentials` returns the zero value and `ErrMalformedCredentials`. |

Names and values are not part of any error string.

### Lifetime

1. The host builds a set with `NewCredentials`.
2. `WithCredentials` stores that set on a child context. A zero set stores nothing.
3. The client copies the set into outgoing headers for that RPC only.
4. The server copies incoming headers back onto the handler context.
5. The plugin reads the handler context. When the context is canceled or the RPC
   returns, the set is not stored anywhere else.

Overlapping calls do not share a set. The process environment is not a field of this
model.

## Opt-in

`PerRequestCredentialConsumer` is a marker. The method is
`ConsumesPerRequestCredentials()` and has no result. Implementing the method is the
whole declaration. `Serve` does not call the method to inject secrets.

A plugin that does not implement the marker has no acceptance note. Inbound headers do
not change its environment and do not fail the RPC by themselves.

## Acceptance note

| Item | Value |
| --- | --- |
| Map | `GetPluginInfoResponse.metadata` (existing field) |
| Key | `supports_per_request_credentials` |
| Value when opted in | `true` |
| Value when not opted in | key absent, even if the plugin wrote it |
| Capability list | unchanged; no new enumerator |

The note is derived inside `GetPluginInfo` after legacy capability notes are applied,
for both the configured `PluginInfo` path and the `PluginInfoProvider` path. Opt-in
overwrites a provider value of `false`. Non-opt-in deletes a provider value of `true`.

## Wire entry

One header per pair. Details and the canonical HTTP spelling are in
[contracts/wire-headers.md](./contracts/wire-headers.md). The trace header
`x-finfocus-trace-id` is a different key and is not read or written by this model.

## What is not an entity

- Tenant identifier. This slice does not add one. Issue #195 is not modeled.
- Process pool, route, idle deadline, instance cap. Host concepts from spec 029.
- Cloud provider enum. Names are opaque tokens.
- Refresh token store, file path, or kubeconfig. Out of scope.
