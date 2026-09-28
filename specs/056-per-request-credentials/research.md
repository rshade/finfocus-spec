# Research: Opt-In Per-Request Credentials

## Decision: Build the opt-in SDK slice now

- **Decision**: Implement the opt-in attach and read path in this repository now. Keep
  one process per organization, with environment credentials, as the default required
  by spec 029. Do not wait for a host measurement inside this change.
- **Rationale**: The maintainer chose to build the opt-in SDK slice now. The assessment
  at `.specify/assessments/per-request-credentials/` (decision.md, 2026-09-27) returned
  needs-clarification. It said not to specify option C until a host reported resident
  memory, cold start, and whether spec 029's 100-instance and 5-second targets were
  missed. Those numbers are still not in the repository or on issue #220, and the
  ~50-100MB figure is still unsourced. This slice does not treat that gap as solved.
  It also does not block the SDK contract the maintainer ordered. The contract does not
  by itself share a process across tenants.
- **Alternatives considered**: Option A, change nothing and keep only the
  per-organization process (rejected as the sole outcome: it remains the default, but
  it does not add the ordered slice). Option B, measure on a host and only then reopen
  the question (rejected: that is the assessment recommendation the maintainer
  overruled; this repo still cannot run that host). Option C as a full shared pool,
  including routing and the instance cap, inside this repo (rejected: constitution
  principle IV and spec 029 put the orchestrator in the host; issue #220's effort label
  was for that larger design, not for this slice).

## Decision: Context on the call, never the process environment

- **Decision**: `WithCredentials` stores a validated `Credentials` value on the call
  context. `ExtractCredentials` reads that value. The SDK does not call `os.Setenv`,
  does not read cloud environment variables, and does not remember the value after the
  context is done.
- **Rationale**: Spec 029 FR-016 and FR-021 make ambient environment credentials the
  default because cloud SDKs read them from the process. A process-global write would
  race when two calls overlap and would leak one tenant into the next. The user
  required the value to live on that call's context only. Missing credentials stay a
  normal `ExtractCredentials` result (`Credentials` zero value and a nil error), so a
  plugin keeps using whatever it already read from the environment.
- **Alternatives considered**: Set the environment for the duration of the handler
  (rejected: races, leaks, and violates the context-only rule). A package-level "current
  credentials" holder (rejected: that is a second store and breaks overlapping calls).
  A cache keyed by tenant (rejected: out of scope, and it would retain secrets). Putting
  the secret in the protobuf request (rejected: proto change, and it would be logged by
  anything that dumps messages).

## Decision: One marker interface, and a metadata note derived from it

- **Decision**: Plugins opt in by implementing `PerRequestCredentialConsumer` with
  `ConsumesPerRequestCredentials()`. `Serve` sets
  `metadata["supports_per_request_credentials"] = "true"` when the plugin implements
  that interface, and deletes that key when it does not. The plugin's hand-written note
  does not win in either direction. No `PluginCapability` value is added.
- **Rationale**: Hosts cannot type-assert a plugin in another process. `GetPluginInfo`
  already returns an open metadata map (`costsource.proto`, field 5), so the note does
  not need a proto change. Principle XII wants interface discovery. The interface is
  that discovery. The note is the host-visible copy, in the same spirit as
  `supports_dry_run`, but it is not a capability enumerator. `maxValidCapability` stays
  `PLUGIN_CAPABILITY_ALLOCATION` (15). `inferCapabilities` does not grow.
- **Alternatives considered**: A new `PLUGIN_CAPABILITY_PER_REQUEST_CREDENTIALS`
  (rejected: proto and generated SDK change, and the maintainer forbade a new
  enumerator). Interface only, with no metadata note (rejected: a remote host cannot
  see the interface; the user said to use the metadata map when that needs no proto
  change, and it does not). Trust a hand-set `WithMetadata` value (rejected: the note
  would lie when the plugin did not implement the method, or would hide a plugin that
  did). A method that returns `bool` (rejected: the return could disagree with the type
  assertion; presence of the method is the opt-in).

## Decision: Header prefix `x-finfocus-credential-`, text values, fixed limits

- **Decision**: Each entry is one header. The gRPC metadata key and the HTTP header name
  are `x-finfocus-credential-` plus the lowercase name. Names match
  `^[a-z][a-z0-9_-]{0,63}$` after ASCII lowercasing. Values are 1 to 4096 bytes, each
  byte in the inclusive range 0x20 through 0x7E. At most 16 entries. The sum of value
  lengths is at most 16384. Duplicate names, repeated header values, an empty set, and
  anything outside those rules are unusable. Validation errors are the fixed sentinels
  in the credentials contract and do not include the name or the value.
- **Rationale**: gRPC metadata keys are lowercase. HTTP canonicalizes them to
  `X-Finfocus-Credential-<Name>`. One prefix keeps the trace header
  `x-finfocus-trace-id` untouched. Printable ASCII is the gRPC metadata value rule, so
  a `-bin` base64 channel is unnecessary for the tokens this slice allows. The caps sit
  well under `DefaultMaxHeaderBytes` (1 MiB) and under a normal gRPC header budget, so
  `Serve` does not raise limits. A service-account file or a kubeconfig does not fit on
  purpose.
- **Alternatives considered**: One JSON blob in a single header (rejected: a parse
  error tends to quote the blob, which is the secret). Provider-specific headers such
  as an AWS access-key field (rejected: principle II, provider-agnostic names). A `-bin`
  suffix and base64 (rejected: a second encoding, larger headers, and still not a file
  channel). New `FINFOCUS_*` or `PULUMICOST_*` environment variables for the secrets
  (rejected: secrets are not environment configuration; the user forbade new
  `PULUMICOST_*` names). Echoing the bad name in the error (rejected: a caller can put
  the secret in the name).

## Decision: gRPC metadata via the existing interceptor chain; Connect via headers it already carries

- **Decision**: On `serveGRPC`, scan credential metadata inside
  `TracingUnaryServerInterceptorWithLogger` after it has already called
  `metadata.FromIncomingContext`. The scan function does not log. On `serveConnect`,
  wrap the mux with HTTP middleware, inside CORS and the payload limit, that copies
  the same headers onto `r.Context()`.
  `pluginsdk.Client` writes those headers on the `connect.Request` it already builds,
  for Connect, gRPC, and gRPC-Web. Export `CredentialUnaryClientInterceptor` for a host
  that uses `grpc.ClientConn` instead of `pluginsdk.Client`. Do not call
  `connect.WithInterceptors`. Do not add credential names to `DefaultAllowedHeaders`.
- **Rationale**: connect-go v1.21.0 exposes application headers on
  `(*Request).Header()` and, in both `protocol_connect.go` and `protocol_grpc.go`, uses
  the HTTP request context as the handler context. `ServeConfig.UnaryInterceptors` are
  gRPC interceptors; Connect mode does not run them (already stated on the tracing
  interceptor and in `allocator_serve_test.go`). Middleware next to
  `payloadLimitMiddleware` is the existing HTTP extension point, not a new interceptor
  type. Default CORS stays closed to these headers so a browser origin cannot start
  sending cloud secrets without an operator opting in through `WithAllowedHeaders`.
  Native gRPC has no HTTP middleware. A dedicated interceptor would call
  `FromIncomingContext` a second time. That function copies every header (measured:
  the copy itself allocates, and the copy grows with unrelated keys). The tracing
  interceptor already holds the copy, so the scan runs there and does not add a log
  field. The error log remains trace id, method, and the error value.
- **Alternatives considered**: Leave Connect unchanged (rejected: the same headers
  already fit; the user said to skip Connect only if a new interceptor design would be
  required, and it is not). A Connect interceptor stack mirroring gRPC (rejected: that
  is the new design the user said not to add). Document the header and make every host
  format it by hand, with no client helper (rejected: `pluginsdk.Client` is the host
  client in this module, and a missed header would silently fall back to ambient
  credentials). Put the copy inside every `ConnectHandler` method (rejected: usage and
  allocator handlers would need the same edit; middleware covers the mux once).

## Decision: Attach with context, not a field on `Client` and not a new method argument

- **Decision**: Hosts call `WithCredentials(ctx, creds)` and pass that context into the
  existing client methods. `Client` has no credential field. Method signatures do not
  gain a variadic option.
- **Rationale**: A field on the long-lived client would stick to the next tenant. A
  variadic option would touch every RPC method for a behavior the context already
  scopes, which is how trace IDs travel today. Issue #220's "option on client requests"
  is this per-call context value, not client construction.
- **Alternatives considered**: `ClientConfig` credential field (rejected: cross-tenant
  leak on a reused client). Variadic `...CallOption` on every method (rejected: wide
  signature churn for no extra isolation). A dedicated RPC (rejected: proto change and
  a second store on the server).

## Decision: Redact every SDK rendering

- **Decision**: `Credentials.String` and `Credentials.GoString` report only how many
  entries are present. `NewCredentials` and `ExtractCredentials` return sentinel errors
  whose text is constant. The SDK does not log on the credential path, including when
  material is malformed. Tests use `test-secret-value` and assert that log buffers,
  `error.Error()`, and Prometheus label values do not contain it. `Names` and `Len` and
  `Get` exist so a plugin can read one value without a method that returns the map.
- **Rationale**: `%v` and `%#v` call `String` and `GoString`. A map return would be
  formatted by the next `fmt` or zerolog dump. The tracing interceptor logs `err`; the
  sentinel text is safe if a plugin returns it. Metric labels are unchanged; the test
  locks that.
- **Alternatives considered**: A debug log of header names on failure (rejected: the
  name can be the secret). Returning the raw header in `ErrMalformedCredentials`
  (rejected: FR-006). Skipping the metric assertion because this slice adds no metric
  (rejected: a future edit could put the value in a label, and SC-003 asks for the
  measurement).

## Decision: Do not auto-fail a plugin that did not opt in

- **Decision**: Inbound headers are parsed for every plugin. `ExtractCredentials`
  returns `ErrMalformedCredentials` when the material is unusable, and a zero value
  with a nil error when no credential header was present. `Serve` does not abort the
  RPC before the handler. An opted-in plugin must treat a non-nil extract error as a
  failed call and must not fall back to the process environment for that call. A plugin
  that did not opt in is expected not to call `ExtractCredentials`. Its environment
  credentials stay in force, and the RPC is not failed because headers arrived.
- **Rationale**: FR-002 says non-opt-in behavior does not change. Failing the RPC from
  middleware would make a stray header break plugins that never asked for this feature.
  The SDK cannot apply the secrets for the plugin without `os.Setenv`, which is
  rejected above. Enforcement is the plugin reading the context, which is what opt-in
  means.
- **Alternatives considered**: Reject the RPC in the interceptor when the plugin did
  not opt in (rejected: behavior change for existing plugins). Reject the RPC in the
  interceptor when the plugin did opt in and the headers are malformed (rejected: the
  handler is the right place to return the sentinel, and the interceptor would have to
  special-case health checks and plugin info). Silently drop malformed headers and
  report "not attached" (rejected: the plugin would then use another tenant's ambient
  credentials on a shared process).

## Decision: Issue #195, spec 029, TypeScript, and CORS stay where they are

- **Decision**: Do not implement authorization middleware or caller identity. Do not
  edit `specs/029-grpc-web-support/`. Do not edit `sdk/typescript/`. Do not add
  `x-finfocus-credential-*` to the default CORS allow-list.
- **Rationale**: Issue #195 is OIDC or IAM identity and says the SDK must not manage
  secrets. Mixing it into this slice would create two stories for one header. Spec 029
  remains the default; this spec is the opt-in exception it deferred to #220. Rewriting
  029's FR-021 would look like the default changed. Principle XIII fires when service
  definitions change; they do not. Browser pages are the wrong place to put cloud
  secrets, and `DefaultAllowedHeaders` is what a browser is allowed to send.
- **Alternatives considered**: A sentence in spec 029's out-of-scope list pointing at
  spec 056 (rejected for this change: 029 is a completed historical spec, and the
  pointer belongs in 056). A TypeScript `withCredentials` helper (rejected: no proto
  change to mirror, and it would invite browser use). Opening default CORS for the
  prefix (rejected: turns a server-to-server header into a cross-origin browser
  header).

## Decision: Tests fail first, and the absent path is benchmarked

- **Decision**: Add `credentials_test.go` and `credentials_serve_test.go` before
  `credentials.go` is considered done. Assert the fixture `test-secret-value` never
  appears in logs, error strings, or gathered metrics. Benchmark the server parse when
  incoming metadata has no credential prefix, and expect zero allocations.
- **Rationale**: Constitution VIII requires a benchmark for new SDK logic. Constitution
  V's test-first habit still applies even though proto is untouched. The serve test is
  what proves `serveGRPC` and `serveConnect` were actually wired.
- **Alternatives considered**: Unit-test the codec only and trust `Serve` to call it
  (rejected: a missed line in `serveGRPC` or `serveConnect` would ship a silent ambient
  fallback). A benchmark of the hit path only (rejected: the hot path in production is
  the plugin that was not sent headers).
