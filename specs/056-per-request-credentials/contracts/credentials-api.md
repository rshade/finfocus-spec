# Contract: Per-Request Credential API

Package: `sdk/go/pluginsdk`. No new protobuf type. No new RPC.

## Opt-in

```text
type PerRequestCredentialConsumer interface {
    ConsumesPerRequestCredentials()
}
```

Implementing the method opts the plugin in. The method has no result and `Serve` does
not call it to deliver secrets. The plugin reads secrets with `ExtractCredentials`.

## Value

```text
func NewCredentials(entries map[string]string) (Credentials, error)
func (c Credentials) Get(name string) (string, bool)
func (c Credentials) Len() int
func (c Credentials) Names() []string
func (c Credentials) String() string
func (c Credentials) GoString() string
```

`NewCredentials` copies `entries`. Rules are in [data-model.md](../data-model.md).

| Condition | Error |
| --- | --- |
| `entries` is nil or empty | `ErrEmptyCredentials` = "per-request credentials require at least one value" |
| Any other rule broken | `ErrInvalidCredentials` = "per-request credentials are not valid" |

`Get` lowercases `name` before lookup. `Names` returns lowercase names in ascending
order. `String` and `GoString` may include the count and must not include a name or a
value. Example shape: `per-request credentials (1 redacted)`. The zero value's string
is `per-request credentials (none)`.

There is no `Entries`, `Map`, or `Values` method.

## Context

```text
func WithCredentials(ctx context.Context, creds Credentials) context.Context
func ExtractCredentials(ctx context.Context) (Credentials, error)
func CredentialUnaryClientInterceptor() grpc.UnaryClientInterceptor
```

`WithCredentials` requires a non-nil context, same as `context.WithValue`. A zero
`Credentials` leaves the context without a set. A valid set is stored on a child
context. The caller's map is not retained beyond the copy `NewCredentials` already made.

`ExtractCredentials`:

| Context | Result |
| --- | --- |
| Nil context, or no set and no wire failure | zero `Credentials`, nil error |
| Valid set | that set, nil error |
| Wire failure recorded on the context | zero `Credentials`, `ErrMalformedCredentials` |

`ErrMalformedCredentials` = "per-request credentials are malformed".

A non-nil error means the caller attempted to send material and it was unusable. An
opted-in plugin must fail that call and must not use process-environment credentials
for it. A nil error and a zero value means nothing was attached, which is not a
failure.

`CredentialUnaryClientInterceptor` copies a valid set from the outgoing call context
into gRPC metadata. It does not log. Hosts that use `pluginsdk.Client` do not need it;
that client writes the same headers itself. Hosts that use `grpc.ClientConn` add this
interceptor and still call `WithCredentials` on the call context.

`pluginsdk.Client` has no credential field. Existing methods keep their signatures.
Passing a context from `WithCredentials` attaches the set to that RPC only.

## Server wiring

Native server (`Web.Enabled` false): the tracing interceptor scans the metadata copy
it already loaded and fills the handler context. The scan does not log and it does
not fail the RPC. It does not call `metadata.FromIncomingContext` a second time.

Web-compatible server (`Web.Enabled` true): HTTP middleware fills `r.Context()` from
the request headers before Connect or gRPC-on-connect handlers run. No
`connect.WithInterceptors` option is added. `ServeConfig.UnaryInterceptors` stay gRPC
only.

Both paths call the same codec. Health checks and other RPCs may carry the headers.
Only a handler that calls `ExtractCredentials` observes them.

## Redaction

SDK logs on this path: none. SDK metrics added: none. Any `error` value defined above
is safe to log because its text is fixed. Plugins must not log the result of `Get`.
