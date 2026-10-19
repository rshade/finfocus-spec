# CostParseService Design

Status: design for review, 2026-10-18. Not filed as an issue, not implemented.
Source: the draft proposal in `parser.md` (outside this repository) and the
decisions recorded below.

## Purpose

A plugin turns an infrastructure-as-code input into a list of resources that
the host can price. The host calls `Parse(format, input, options)` on the
plugin that owns the domain, receives resource descriptors, and prices them
through the existing cost RPCs.

Intended uses:

- `aws-public` parses CloudFormation (`cfn`).
- `kubernetes` parses Helm (`helm`) and Kustomize (`kustomize`).
- `azure-public` could parse ARM and Bicep later.

## Scope

- Terraform and Pulumi ingestion stay native to the finfocus core. A plan or
  preview mixes resources from several clouds, so no single provider plugin
  owns the parsing.
- The service covers cloud-native and Kubernetes formats only. The service
  documentation states this limit.
- Out of scope: pricing, allocation, recommendations, the core's internal
  parser interface, Terraform plan ingestion, and any plugin's format logic.

## Decisions

| # | Question | Decision |
|---|----------|----------|
| 1 | Type of `format` | Free string, advertised by the plugin |
| 2 | Input transport | Both content and path, as a `oneof` |
| 3 | Combining the two | Exactly one is set; the plugin advertises the modes it accepts |
| 4 | Type of `options` | `google.protobuf.Struct` |
| 5 | Response shape | `ParsedResource` wrapper plus a separate `ParseWarning` list |
| 6 | Service name | `CostParseService` |
| 7 | Response caching hint | Keep `expires_at`, following spec 045 |

## Wire contract

New file `proto/finfocus/v1/parse.proto`.

```protobuf
service CostParseService {
  rpc Parse(ParseRequest) returns (ParseResponse);
}

message ParseRequest {
  string format = 1;
  oneof input {
    ParseFileSet files = 2;
    string root_path = 3;
  }
  google.protobuf.Struct options = 4;
}

message ParseFileSet { repeated ParseFile files = 1; }
message ParseFile {
  string path = 1;
  bytes content = 2;
}

message ParseResponse {
  repeated ParsedResource resources = 1;
  repeated ParseWarning warnings = 2;
  google.protobuf.Timestamp expires_at = 3;
}

message ParsedResource {
  ResourceDescriptor resource = 1;
  string source_id = 2;
  repeated string unknown_properties = 3;
}

message ParseWarning {
  ParseWarningSeverity severity = 1;
  string code = 2;
  string message = 3;
  string source_id = 4;
}
```

Field numbers above are the proposal. Fields held for later use are held by
comment, not by `reserved`.

### Request rules

- `format` is lowercase, 1 to 32 bytes, matching `[a-z][a-z0-9_-]*`.
- Exactly one of `files` and `root_path` is set. Both or neither is
  `InvalidArgument`, so no field is silently ignored.
- A `ParseFile.path` is relative, has no `..` segment, and is unique within the
  set.
- `options` is encoded at most 65536 bytes, the same cap as
  `ResourceDescriptor.attributes` (596). Keys are documented per format and are
  not enforced by the proto: `cfn` takes `parameters`; `helm` takes
  `release_name`, `namespace` and `values`; `kustomize` takes none. An unknown
  key is `InvalidArgument`, as the allocator does for policy (052).

### Response rules

- `resource.resource_type` carries the Pulumi type token, so the host makes no
  `ResolveResourceTypes` call for these formats.
- Properties are normalized to the shape the plugin's own pricing code expects
  (PascalCase for CloudFormation, camelCase for Pulumi).
- `source_id` is the input's own name for the resource: the CloudFormation
  logical ID, or `Kind/name` for Helm and Kustomize. It is non-empty and unique
  within the response, so the host can map a priced result back to the input.
- `unknown_properties` lists dotted paths whose values the parser could not
  resolve, such as a `Ref` to a parameter with no default. It uses the path
  convention of `omitted_fields` (593). A path listed there has no value in
  `resource.attributes`, so "unknown" stays distinct from "absent".
- `ParseWarning.code` is a stable, non-empty string (for example
  `resource_skipped`), so hosts match on it and not on the message. A skipped
  resource is a warning with `source_id` set.
- `expires_at` is an advisory hint under the 045 convention: unset means no
  caching guidance. A plugin sets it only when its output depends on something
  that changes, such as a remote Helm dependency.

### Error codes

| Condition | Code | Detail |
|-----------|------|--------|
| Invalid `format`, both or neither input, bad path, bad `options` | `InvalidArgument` | Names the field |
| Unadvertised format | `FailedPrecondition` | `ErrorInfo` reason `UNSUPPORTED_FORMAT` |
| Input mode the plugin does not accept | `FailedPrecondition` | `ErrorInfo` reason `UNSUPPORTED_INPUT_MODE` |
| Unparseable input | `InvalidArgument` | Names the file and, where known, the line |
| Input over the advertised limit | `ResourceExhausted` | |

There is no streaming in the first version. Large inputs rely on the advertised
limit below.

## Discovery

- `PLUGIN_CAPABILITY_COST_PARSE = 19`, legacy metadata name
  `supports_cost_parse`. The next free value is 19, since 18 is
  `RECOMMENDATION_SCORING`.
- `GetPluginInfo` metadata keys, set through `pluginsdk.WithParseLimits`:
  - `parse_formats`: comma-separated format names
  - `parse_input_modes`: `content`, `path`, or both
  - `parse_max_input_bytes`: the largest total input the plugin accepts
- The metadata follows the scorer limits pattern (591). No proto field carries
  these values, and response fields stay authoritative.
- `PluginInfo.Validate` rejects a malformed set through `ParseParseLimits`.
- Capability inference adds only the base pricing capabilities, so a
  parser-only plugin uses `WithCapabilities(PLUGIN_CAPABILITY_COST_PARSE)`.
  `Serve` warns otherwise, as it does for usage-only plugins (051).
- Not changed: `SourceFormat`, `ResolveResourceTypes`, and every existing
  message.

## Go SDK

- `CostParseProvider` has one method:
  `Parse(ctx, *pbc.ParseRequest) (*pbc.ParseResponse, error)`.
- `Serve` registers `CostParseService` through unexported adapters in gRPC and
  Connect mode, and adds it to the Connect health checker, only when the plugin
  implements the provider. The provider is a new `parser` field on
  `optionalServices`, so `Serve` gains no parameter.
- Handler errors go through `toConnectError` and `handlerStatus` (600). A
  handler's `Unimplemented` takes the not-a-provider path, so a plugin that
  embeds the generated stub is not treated as a parser.
- Rules live in `sdk/go/testing/parse.go`, and `pluginsdk` delegates, because
  `testing` cannot import `pluginsdk`:
  - `ValidateParseRequest(req, limits)` checks the request rules above and the
    size caps.
  - `ValidateParseResponse(req, resp)` checks that each `resource` passes the
    contract descriptor rules, that `source_id` values are unique and non-empty,
    that `unknown_properties` paths are well formed and absent from
    `attributes`, and that warning codes are non-empty.
  - Invalid-input errors use the `invalidArgumentError` pattern from 052.
- Both validators guard the call with `len() > 0` at the call site, so the
  existing hot paths and their benchmarks do not move (053, 557).

## Mock and conformance

- `MockCostParser` applies fixed rules and is model-agnostic, like
  `MockRecommendationScorer`. It is a separate type, so `MockPlugin`
  capabilities do not change.
- `RunCostParseConformance` checks structure and error codes, never resource
  contents. Scenarios:
  - `format_rejected`: an unadvertised format is `FailedPrecondition` with
    `UNSUPPORTED_FORMAT`.
  - `input_mode_xor`: both or neither input is `InvalidArgument`.
  - `unsupported_input_mode`: a mode the plugin does not advertise is
    `FailedPrecondition`.
  - `path_traversal_rejected`: a `..` or absolute path is `InvalidArgument`.
  - `source_ids_unique`, `unknown_not_absent`, `descriptors_valid`, and
    `deterministic` (the same input gives the same response).
- Scenarios use the provider-neutral sample resource from 598, with no vendor
  templates.

## TypeScript

A `CostParseClient` in `sdk/typescript/packages/client` exposes `parse(...)`.
The REST gateway speaks proto3 JSON through `fromJson` and `toJson`, because
`bytes` content and `Timestamp` fields cannot go through `JSON.stringify`.

## Host behavior

This section is guidance and is not enforced by the SDK. It goes in
`docs/cost-parse.md`.

- **Who parses.** A pricing plugin can implement `CostParseProvider` itself, or
  a separate parser-only plugin can serve a format. The RPC is identical, so
  the host sees one capability either way.
- **Conflicts.** When two plugins advertise the same format, the host chooses in
  this order: an explicit user choice; otherwise the plugin whose
  `GetPluginInfo.providers` match the resource's likely provider; otherwise it
  fails with an "ambiguous format" error. Merging two plugins' output is not
  defined.
- **Untrusted input.** A plugin treats `files` content as untrusted. In path
  mode the host sends only a root it intends to expose, and the plugin does not
  follow symlinks out of that root. Neither the SDK nor conformance can enforce
  the symlink rule, so it is a contract rule only, like the host redaction rule
  of 596.
- **Caching.** Without `expires_at`, parsing is a pure function of the input, so
  a host may cache by input hash.

## Risks and open items

- `ParseFile.path` rules cover the request, not the plugin's own file reads.
  The path-mode symlink rule depends on plugin discipline.
- The per-format `options` keys live in documentation. A drift check between
  docs and a plugin's accepted keys is not part of this design.
- The core's internal parser interface is built against this wire contract, so
  a change here needs a matching change there.
- Adding a capability value touches `maxValidCapability`,
  `optionalCapabilities`, `legacyCapabilityNames`, the `IsValidCapability`
  bounds test, the registry schema enum, and `TestSchemaDrift` (see 051 and
  595).
- Benchmarks: A/B the existing validators against `main` with prebuilt test
  binaries, since layout shifts of about 0.5 ns show up on the 6 ns benchmarks.
