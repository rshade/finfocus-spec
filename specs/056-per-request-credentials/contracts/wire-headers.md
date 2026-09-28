# Contract: Credential Headers

One header per credential entry. The trace header is not part of this contract.

## Names

| Transport | Header |
| --- | --- |
| gRPC metadata | `x-finfocus-credential-<name>` (lowercase) |
| HTTP, including Connect and gRPC-Web | `X-Finfocus-Credential-<Name>` via `http.CanonicalHeaderKey` |

`<name>` is the lowercase token from [data-model.md](../data-model.md). The prefix
constant is `CredentialMetadataPrefix` = `x-finfocus-credential-`.

`x-finfocus-trace-id` is not read or written here. Headers that do not start with the
prefix are ignored.

## Values

The header value is the credential string itself. It is not base64, not JSON, and not
percent-encoded. Characters outside 0x20 through 0x7E cannot be sent. Callers that
have binary material must not use this slice to carry it.

A single header key with two values is malformed. The whole set is rejected. Partial
pairs are not delivered.

## Who writes and reads

| Side | Behavior |
| --- | --- |
| `pluginsdk.Client` | For each existing RPC method, copy a valid context set onto `connect.Request` headers before the call. No set means no credential headers. |
| `CredentialUnaryClientInterceptor` | Same copy, into outgoing gRPC metadata, for `grpc.ClientConn` callers. |
| Native server | The tracing interceptor's already-copied metadata is scanned onto the handler context. No second copy and no log line. |
| Connect HTTP middleware | Copy incoming HTTP headers onto the request context. |

Malformed input stores `ErrMalformedCredentials` on the context and drops the bytes.
It does not write a log line.

## Limits

16 headers, name up to 64 characters, value up to 4096 bytes, 16384 value bytes in
total. `Serve` does not change `DefaultMaxHeaderBytes` or the gRPC max header list
size.

## CORS

`DefaultAllowedHeaders` does not include this prefix. A browser origin cannot send
these headers under the default CORS policy. An operator who deliberately widens
`AllowedHeaders` is outside this slice. Hosts that call plugins server-to-server do
not need that change.

## Example

A set with one entry `token` = `test-secret-value` is the single header:

```text
x-finfocus-credential-token: test-secret-value
```

Logs, errors, and metrics still must not contain `test-secret-value`. The example
above is the on-the-wire shape used by tests, not a log format.
