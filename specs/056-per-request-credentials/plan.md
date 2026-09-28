# Implementation Plan: Opt-In Per-Request Credentials

**Branch**: `056-per-request-credentials` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/056-per-request-credentials/spec.md`

## Summary

Add an opt-in credential channel in `sdk/go/pluginsdk` so a host can attach named cloud
credential values to one call and an opted-in plugin can read them from that call's
context. The default stays spec 029: one plugin process per organization, credentials in
the process environment. Nothing in this repository launches, routes, or caps processes.

The assessment at `.specify/assessments/per-request-credentials/` returned
needs-clarification and told us not to specify until a host measured memory and cold
start. That alternative is rejected. The maintainer chose to build this opt-in SDK slice
now. The measurements are still absent and are not a success criterion here.

No protobuf field, no generated code, no `PluginCapability` value, no new RPC, no
TypeScript client, no `PULUMICOST_*` name, no cache, and no process-environment write.
Support is advertised by setting the existing `GetPluginInfoResponse.metadata` key
`supports_per_request_credentials` to `true` when the plugin implements
`PerRequestCredentialConsumer`. gRPC metadata carries the values on the native server.
Connect, gRPC-Web, and gRPC-over-connect carry the same HTTP headers through the request
context connect-go already uses. That does not add a Connect interceptor.

## Technical Context

**Language/Version**: Go 1.27.1 (go.mod)

**Primary Dependencies**: Existing `google.golang.org/grpc` metadata, `connectrpc.com/connect`
v1.21.0 request headers, `net/http`, `github.com/rs/zerolog`. No new module requirements.

**Storage**: N/A. The value lives on the call context only. No cache, file, or second store.

**Testing**: `go test` for `sdk/go/pluginsdk`. External test package `pluginsdk_test`.
Fixture secret `test-secret-value`. Benchmarks with `-benchmem` for the absent-header path.

**Target Platform**: Library used by FinFocus plugin processes and by hosts that call them.

**Project Type**: Go SDK library inside a protocol repository.

**Performance Goals**: A call with no credential headers does no extra allocation in the
new server path beyond the metadata lookup the process already pays for. Attaching a
set allocates one copied map. No per-call lock and no global map.

**Constraints**:

- Do not edit `proto/`, `sdk/go/proto/`, or `sdk/typescript/`. Do not run `make generate`.
- Do not add a `PluginCapability` enumerator, and do not bump `maxValidCapability`.
- Do not add a pool, router, idle reaper, or instance cap.
- Do not call `os.Setenv` or read cloud credentials from the environment in this SDK.
- Do not log, format, or label metric series with credential names or values. Errors are
  fixed strings: `ErrEmptyCredentials`, `ErrInvalidCredentials`, `ErrMalformedCredentials`.
- Header budget stays inside today's servers. Connect allows `DefaultMaxHeaderBytes`
  (1 MiB). This slice caps a set at 16 names, 64-character names, 4096-character values,
  and 16384 value bytes together, which fits without raising either limit.
- No new `PULUMICOST_*` identifier. The header prefix is `x-finfocus-credential-`.
- `WithAllowCredentials` stays the CORS flag it already is. It is not this API.

**Scale/Scope**: One new source file plus serve, client, plugin-info, README, and
developer-guide touch points. Two transports, one metadata key, one marker interface.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Source: `.specify/memory/constitution.md` (v1.5.0). The sentences below are the gates
this slice can hit. A gate fails only when this plan breaks that sentence.

- **I. gRPC Proto Specification-First Development.** Constitution: "Every change to the
  protocol MUST begin with protobuf specification updates before implementation" and
  "SDK code is generated from proto definitions; manual edits to generated code are
  FORBIDDEN." **Pass.** This slice does not change the protocol. `GetPluginInfoResponse`
  already has `map<string, string> metadata = 5`. The acceptance note is one key in that
  map. Credential bytes ride on gRPC metadata and HTTP headers, which are not protobuf
  fields. No file under `proto/` or `sdk/go/proto/` is edited, so there is nothing to
  generate.
- **II. Multi-Provider gRPC Consistency.** Constitution: "ResourceDescriptor message
  fields MUST be provider-agnostic." **Pass.** Names are caller-chosen tokens
  (`access_key`, `session_token`, and so on). The SDK does not define an AWS, Azure,
  GCP, or Kubernetes credential schema and does not add provider fields.
- **III. The Spec Consumes, It Does Not Calculate.** Constitution: "The specification
  and its implementing plugins are NOT responsible for complex pricing logic."
  **Pass.** No cost math. The slice moves opaque strings for one call.
- **IV. Strict Separation of Concerns.** Constitution: "finfocus-spec defines
  interfaces; finfocus-core contains application logic" and "The SDK is for plugin
  creators, not end-users." **Pass.** The host keeps the orchestrator that spec 029
  already assigned outside this repository. This plan refuses a pool, a router, and an
  instance cap. See research.md, "Do not build the host pool."
- **V. Test-First Protocol.** Constitution: "TDD is mandatory for all gRPC
  specification changes" and tests "MUST fail against current proto/implementation"
  before the proto edit. **Pass, with the proto clause not triggered.** There is no
  proto edit. Tasks still write failing tests before the helpers, the serve wiring, and
  the client header copy. That is the test-first rule this repository can apply to an
  SDK-only change.
- **VI. Protobuf Backward Compatibility.** Constitution: "Breaking changes to protobuf
  definitions are strictly controlled" and "buf breaking change detection MUST pass."
  **Pass.** No wire schema change. Existing plugins that do not implement the marker
  gain no metadata key and no new RPC behavior. `buf` breaking is not in play because
  proto bytes do not change.
- **VII. Comprehensive Documentation & Identity Transition.** Constitution:
  "Documentation MUST be updated in the same PR as feature implementation" and
  "Root README.md, docs/, and SDK README.md MUST stay in sync." **Pass.** Update
  `sdk/go/pluginsdk/README.md` with a section that distinguishes this API from CORS
  `AllowCredentials`. Add one short paragraph beside the existing environment-credential
  bullets in `PLUGIN_DEVELOPER_GUIDE.md`. Do not rewrite spec 029 and do not edit the
  root README unless a sentence there says per-request credentials cannot exist. No new
  `PULUMICOST_*` name, which keeps the FinFocus identity rule already used by `env.go`.
- **VIII. Performance as a gRPC Requirement.** Constitution: "Common operations
  (validation, enum lookups) should aim for zero-allocation" and "Benchmarks are
  required for all new core SDK logic." **Pass.** The absent-header path must not
  allocate. `credentials_benchmark_test.go` covers that path and `NewCredentials` for a
  one-entry set. The hit path may allocate the copied map. No global lock.
- **IX. Observability & Validation.** Constitution: "Logging and Metrics are Separate"
  and metrics "should be implemented as optional, distinct components." **Pass.** No
  new log line and no new metric. The credential interceptor is not the tracing
  interceptor, so a trace failure log cannot dump metadata. Metric labels stay
  `grpc_method`, `grpc_code`, and `plugin_name`. Tests assert `test-secret-value` is
  absent from the log buffer, from `Error()` strings, and from gathered metric labels.
- **X. Follow Established Patterns.** Constitution: "New contributions MUST adhere to
  existing, documented patterns" and "Significant changes MUST be proposed via a design
  spec in specs/." **Pass.** This directory is that spec. The marker interface matches
  `UsageSourceProvider` and `AllocatorProvider` discovery. The metadata note matches
  `supports_dry_run` style keys without joining the capability enum. Context storage
  matches `ContextWithTraceID` / `TraceIDFromContext`, with a private key type.
- **XI. Mandatory Copyright Headers.** Constitution: "Every source file (Go, Proto,
  Script, Schema) MUST include the standard Apache 2.0 copyright header." **Pass.**
  New Go files use the header already on `sdk/go/pluginsdk/usage_source.go`.
- **XII. Automatic Capability Declaration.** Constitution: "Plugin capabilities MUST be
  automatically discovered through interface implementation" and "Legacy string-based
  capability metadata MUST be maintained in GetPluginInfo responses." **Pass.** Opt-in
  is interface discovery: `PerRequestCredentialConsumer`. The host-visible legacy note
  is written by `Serve` from that interface, not from a hand-edited map. This is not a
  new `PluginCapability`. Principle XII's enum list is the closed proto set of RPC
  capabilities (`PROJECTED_COSTS` through `ALLOCATION`). A credential transport does
  not add an RPC, so adding enumerator 16 would be a proto change principle I and the
  maintainer both forbid. The interface plus the existing metadata map is the discovery
  principle XII describes, without pretending this is a service capability.
- **XIII. Multi-Language SDK Synchronization.** Constitution: "All language SDKs (Go,
  TypeScript) MUST be kept in sync when gRPC service definitions change." **Pass.**
  Service definitions do not change. The TypeScript client is generated from those
  definitions. No TypeScript edit. A hand-written browser helper that attaches cloud
  secrets is rejected in research.md.
- **XIV. Documentation Integrity.** Constitution: "All exported functions, types, and
  methods in Go SDK packages MUST have godoc comments" and "Code examples in README
  files MUST compile." **Pass.** Every new export gets a godoc comment. The README
  sample is the same sequence as an `Example` in `example_test.go`, which `go test`
  compiles. Spec status is Draft until tasks exist, which this workflow adds in the
  same change.

## Project Structure

### Documentation (this feature)

```text
specs/056-per-request-credentials/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── credentials-api.md
│   ├── plugin-info-metadata.md
│   └── wire-headers.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
sdk/go/pluginsdk/
├── credentials.go                 # types, validation, context, header codec
├── credentials_test.go            # redaction, limits, context, codec
├── credentials_benchmark_test.go  # absent path and one-entry attach
├── credentials_serve_test.go      # native gRPC, Connect, plugin info, metrics
├── sdk.go                         # Serve chains the gRPC copy; Connect middleware; metadata note
├── client.go                      # copy context credentials onto connect.Request headers
├── example_test.go                # compiled README sample
└── README.md                      # new section

PLUGIN_DEVELOPER_GUIDE.md          # one paragraph beside environment credentials
```

**Structure Decision**: All runtime code stays in `sdk/go/pluginsdk`. No new package, so
`pluginsdk` does not gain an import cycle with `sdk/go/testing`. Tests that need a
running server follow `allocator_serve_test.go`: `ServeConfig.Listener` on
`127.0.0.1:0`, not a fixed port. Native mode is `Web.Enabled == false` (`serveGRPC`).
The credential scan uses the metadata copy the tracing interceptor already made.
Web-compatible mode is `Web.Enabled == true` (`serveConnect`). HTTP middleware copies
credential headers onto the request context. Neither path adds a Connect interceptor.

## Complexity Tracking

No constitution gate fails. The choices that look like exceptions are recorded here so
a later edit does not "simplify" them into a proto change or a process-global secret.

| Choice | Why it is not a violation | Simpler alternative rejected because |
| --- | --- | --- |
| Metadata key instead of `PluginCapability` | Principle XII is satisfied by the marker interface plus the legacy metadata map that `GetPluginInfo` already returns. Principle I forbids a proto edit this slice does not need. | A new enumerator would change `enums.proto`, force `make generate`, and touch TypeScript. The maintainer forbade that. |
| Connect without `connect.WithInterceptors` | connect-go already puts HTTP headers on `connect.Request.Header()` and passes `http.Request.Context()` into handlers (`protocol_connect.go`, `protocol_grpc.go` in module `connectrpc.com/connect@v1.21.0`). Copying headers in the existing HTTP middleware uses that context. | A Connect interceptor stack would be a second design next to `ServeConfig.UnaryInterceptors`, which `allocator_serve_test.go` already notes do not run in Connect mode. |
| Scan inside the tracing interceptor | `metadata.FromIncomingContext` copies every header. The tracing interceptor already did that. The scan uses that copy and does not log. The error log stays trace id, method, and `err`. | A second interceptor that calls `FromIncomingContext` again was measured to allocate on every call, including calls with no credentials. |

## Post-Design Re-check

Research, the data model, and the three contracts do not add a protobuf field, a
capability enumerator, a generated file, a TypeScript client, a pool, a router, an
instance cap, a cache, a file, a `PULUMICOST_*` name, or a Connect interceptor. The
acceptance note is a key in metadata that already exists. Both call paths carry
`x-finfocus-credential-<name>`. Gates above still pass.

Rejected again if a later stage proposes them: editing spec 029's default, implementing
issue #195, calling `os.Setenv` around the handler, raising header limits, allowing
credential headers through the default CORS allow-list, and logging header names that
failed validation (the name might be the secret).
