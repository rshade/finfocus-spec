# Research: Usage Source Service (GetStats)

**Feature**: 051-usage-source-getstats | **Date**: 2026-09-25

The spec had no open `NEEDS CLARIFICATION` items after the 2026-09-25 clarification session. This
document records the design decisions the plan depends on, most of them forced by the existing
codebase.

## R1. Proto placement and shape

- **Decision**: New file `proto/finfocus/v1/usage.proto`, package `finfocus.v1`, same `go_package`
  (`…/sdk/go/proto/finfocus/v1;pbc`). It imports `finfocus/v1/costsource.proto` for
  `ResourceDescriptor`. The message shapes are those proposed in #505, with inline comments
  documenting subject keys, metrics, units, priceable tagging, and error codes. Add
  `PLUGIN_CAPABILITY_USAGE_STATS = 14` after value 13 in `enums.proto`.
- **Rationale**: This follows the existing precedent of several services in one package
  (`CostSourceService`, `ObservabilityService`, `PluginRegistryService`). A separate file keeps
  `costsource.proto` focused and generates separate `usage*.go` / `usage_pb.ts` files. That matches
  the #505 acceptance criteria. A new file plus a new enum value passes `buf breaking` (FILE).
- **Alternatives**: Adding the RPC to `CostSourceService` was rejected: usage sources have no prices
  and would stub every cost RPC. A new proto package (`finfocus.usage.v1`) was rejected: it needs a
  second Go package and `ResourceDescriptor` would cross packages for no benefit.
- **buf lint check**: The names follow DEFAULT rules: `…Service` suffix, `GetStatsRequest` /
  `GetStatsResponse`, and `STATS_MODE_UNSPECIFIED = 0` with the enum-prefixed values. The
  `google/protobuf/timestamp.proto` import is already available.

## R2. Serving the service from `pluginsdk.Serve`

- **Decision**: Check `config.Plugin.(UsageSourceProvider)` once in `Serve` and pass the result to
  both serve functions. Then:
  - **gRPC mode**: `pbc.RegisterUsageSourceServiceServer(grpcServer, &usageSourceGRPCServer{p})`,
    where the adapter embeds `pbc.UnimplementedUsageSourceServiceServer` and delegates `GetStats`.
  - **Connect mode**: `pbcconnect.NewUsageSourceServiceHandler(&usageSourceConnectHandler{p},
    handlerOpts...)` mounted on the same mux. `pbcconnect.UsageSourceServiceName` is appended to the
    `grpchealth.NewStaticChecker` arguments.
  - Neither registration happens when the assertion fails, so existing behavior is unchanged
    (US2-2).
- **Rationale**: The generated server interface requires `mustEmbedUnimplementedUsageSourceServiceServer()`,
  which plugin structs won't have. An adapter keeps plugin authors to "implement one method"
  (SC-001) and mirrors how `ConnectHandler` wraps `Server`. The existing `Server` type implements
  `CostSourceServiceServer`, so a `GetStats` method on it would be unrelated to that service.
- **Interceptors and tracing**: In gRPC mode, `grpc.ChainUnaryInterceptor` is server-wide, so the
  tracing interceptor and `ServeConfig.UnaryInterceptors` apply to `GetStats` automatically. A test
  with a counting interceptor proves it. In Connect mode, the new handler receives the same
  `handlerOpts` as the cost handler. That slice is empty today (`sdk.go` `serveConnect`), so neither
  service gets tracing or `UnaryInterceptors` over Connect. FR-010 promises parity with the cost
  service, not tracing over Connect. Adding Connect interceptors would change existing RPC behavior
  and is out of scope; `docs/usage-source.md` states the limitation.
- **Health**: The gRPC-mode server registers no health service today (only reflection). "Included
  in health checks" (FR-010) therefore applies to the Connect-mode `grpchealth` checker, which is
  the only health surface. Adding a health service to gRPC mode is out of scope.
- **Alternatives**: `Server` implementing `pbc.UsageSourceServiceServer` directly was rejected:
  `NewTestHarness(server)` and other callers would silently gain a second service. Registering the
  service unconditionally with an `Unimplemented` body was rejected: health would report
  `SERVING` for a service the plugin does not offer.

## R3. Capability inference, bounds, and the usage-only warning

- **Decision**:
  - `inferCapabilities` appends `PLUGIN_CAPABILITY_USAGE_STATS` when
    `plugin.(UsageSourceProvider)`, and `optionalCapabilities` goes 6 → 7.
  - `legacyCapabilityNames` gets `"supports_usage_stats"`.
  - `maxValidCapability` becomes `PLUGIN_CAPABILITY_USAGE_STATS` (14), with a comment noting that
    #506 moves it to 15.
  - New `warnUsageSourceCapabilities(logger, plugin, info)`, called from `Serve` after the server is
    constructed, logs one `Warn` when all of these hold:
    1. the plugin implements `UsageSourceProvider`,
    2. `info == nil || len(info.Capabilities) == 0`, and
    3. the plugin does **not** implement `PluginInfoProvider`.

    The message says usage-only plugins should set `PluginInfo.Capabilities` explicitly.
- **Rationale**: This matches clarification Q1: inference unchanged, document + warn. A plugin that
  implements `PluginInfoProvider` controls its capabilities dynamically, so warning it would be a
  false positive. The warning goes through `server.logger`, which is `ServeConfig.Logger` or the
  default, so tests can capture it with a buffer-backed zerolog logger.
- **Alternatives**: Suppressing the base pricing capabilities when a plugin implements
  `UsageSourceProvider` was rejected by the clarification: a plugin may do both. Failing startup
  was rejected as a breaking behavior for a legitimate dual-role plugin.

## R4. Where the vocabulary constants live (import-cycle constraint)

- **Decision**: Exported constants live in `sdk/go/pluginsdk/subjects.go` (names per #505, plus
  `Unit*` constants for the four documented units). `sdk/go/testing/usage_source.go` keeps an
  **unexported** copy of the known subject keys, the `label.` prefix, and the valid row kinds.
  `pluginsdk/subjects_test.go` asserts that the two sets are identical by calling the exported
  `plugintesting.KnownSubjectKeys()` accessor, which returns a copy.
- **Rationale**: `sdk/go/pluginsdk/conformance.go` imports `sdk/go/testing` (non-test code), so
  `testing` cannot import `pluginsdk`. The drift test keeps the one duplication honest.
- **Alternatives**:
  - A new leaf package (`sdk/go/usage`) with pluginsdk re-declaring `const SubjectCluster =
    usage.SubjectCluster`: this adds a third public location for the same names, more surface than
    a private list plus one test.
  - Moving the validator into `pluginsdk`: #505 fixes it in `sdk/go/testing`, and core is planned
    against that.

## R5. `ValidateStatsResponse` rule set and ordering

- **Decision**: The function returns the **first** violation as an `error` wrapping the sentinel
  `ErrInvalidStatsResponse`. The message names the offending row or priceable index and the key or
  value. Checks run in this order:
  1. `nil` response.
  2. Mode is `UNSPECIFIED`.
  3. For each row, in index order:
     - `kind` is missing;
     - `kind` is not `workload`/`node` (this also catches `__idle__`/`__cluster__`);
     - `kind` is `node` but the `node` subject key is missing or empty;
     - a subject key is not known and not `label.<non-empty>`;
     - the amount is negative, NaN, or ±Inf;
     - the canonical (subject, metric) pair was already seen.
  4. For each priceable entry, in index order:
     - `nil` entry, or empty `id`;
     - `tags["kind"] == "node"` and `id` is not in the set of row `node` subject values.

  Accepted:
  - nodes with capacity rows and no priceable entry;
  - priceable `kind=cluster` with no matching `node` subject;
  - priceable entries with any other or missing `kind` tag (not checked);
  - custom metrics and units;
  - an empty response with a set mode.
- **Rationale**: First-error returns match every existing `Validate*Response` helper in `harness.go`
  and give table tests one exact assertion per case (SC-003). Non-finite amounts are treated as
  invalid under the same "non-negative amount" requirement: NaN fails `>= 0` checks silently and
  would poison allocation. An empty-suffix `label.` key is not a `label.<key>`, so it is rejected
  as unknown.
- **Duplicate key**: For each row, sort the subject keys and build `k=v\x00…\x00metric`, then
  track the result in a `map[string]struct{}`. This allocates, which is acceptable for a test
  helper. It is O(rows × keys log keys).
- **Alternatives**: `errors.Join` of all violations was rejected: it breaks with repo convention
  and makes assertions brittle. Checking unit/metric consistency was rejected: the spec assumptions
  exclude it.

## R6. In-memory harness for usage sources

- **Decision**: A new `UsageSourceHarness` in `sdk/go/testing`, with
  `NewUsageSourceHarness(impl UsageStatsServer)`, `Start(t)`, `Stop()`, and
  `Client() pbc.UsageSourceServiceClient`. `UsageStatsServer` is a local one-method interface
  (`GetStats`), which the harness wraps in an adapter embedding
  `pbc.UnimplementedUsageSourceServiceServer`.
- **Rationale**: A plugin struct implementing `pluginsdk.UsageSourceProvider` satisfies the local
  interface structurally, with no import of `pluginsdk` (R4) and no `mustEmbed…` burden. It mirrors
  `TestHarness` (bufconn, same `bufSize`, same dial pattern) and keeps `NewTestHarness`'s signature
  untouched (SC-004).
- **Alternatives**: Extending `TestHarness` to also register usage when `impl` implements it was
  rejected: `impl` is typed `CostSourceServiceServer`, so usage-only plugins could not be passed at
  all.

## R7. Transport parity test (SC-002, US1-6)

- **Decision**: In `pluginsdk/usage_source_test.go`, call `Serve` twice with an injected
  `net.Listener` (per the project pattern): once with `Web.Enabled=false` and a
  `grpc.NewClient`-based client, once with `Web.Enabled=true` and a
  `pbcconnect.NewUsageSourceServiceClient` over HTTP. Compare the responses with `proto.Equal`.
  For each error scenario in R8, compare the gRPC `status.Code`/message with the Connect
  `connect.CodeOf`/`Message` and assert both match the code the source returned. A unit test for
  `toConnectError` covers `nil`, an existing `*connect.Error`, each gRPC code 1–16, and a plain
  error.
  Connect mode also checks `grpc.health.v1.Health/Check` for `finfocus.v1.UsageSourceService`,
  which should return `SERVING`.
- **Rationale**: This exercises the real registration paths rather than the harness. The injected
  listener avoids port races (see CLAUDE.md `pluginsdk.Serve` guidance).

## R8. Error semantics ownership and Connect code mapping

- **Decision**: Status codes returned by `GetStats` reach the client unchanged on both transports.
  - **gRPC mode**: gRPC `status` errors pass through as-is.
  - **Connect mode**: the adapter converts the error with a new unexported helper,
    `toConnectError(err error) error`, in `sdk/go/pluginsdk/connect_errors.go`:
    - `nil` → `nil`;
    - an error that is already a `*connect.Error` → unchanged;
    - an error carrying a gRPC status (`status.FromError` succeeds and the code is not `OK`) →
      `connect.NewError(connect.Code(st.Code()), errors.New(st.Message()))`, preserving the message
      (the numeric codes of `codes.Code` and `connect.Code` are identical for 1–16);
    - any other error → unchanged (connect-go then reports `CodeUnknown`, matching gRPC's
      behavior for a plain error).
  - A reference usage source in the tests returns `InvalidArgument` (partial window, inverted
    window, historical on a run-rate-only source), `PermissionDenied` with a message like
    `cannot list pods`, and `Unauthenticated`. The R7 parity test asserts that each **code and
    message** is identical over gRPC and Connect.
- **Rationale**: This is verified, not assumed. connect-go v1.19.1 does not read gRPC `status`
  errors: `wrapIfUncoded` (`error.go:278`) wraps anything that is not a `*connect.Error` as
  `CodeUnknown`. Without the helper, every `PermissionDenied` or `InvalidArgument` from a usage
  source would reach Connect clients as `Unknown`, violating FR-008/009 and SC-002.
- **Scope boundary (superseded 2026-09-25)**: The user chose to fix `ConnectHandler` in this
  feature instead of a follow-up; all its methods now use `toConnectError` (T045). Original note:
  the existing `ConnectHandler` (`connect.go`) returns errors raw and has the
  same defect for every `CostSourceService` RPC. Fixing it changes the observable behavior of
  existing RPCs, so it is tracked as a separate bug; 051 only introduces the helper and uses it in
  the usage adapter. `toConnectError` is written so the follow-up fix is a one-line change per
  `ConnectHandler` method.
- **Alternatives**: Returning `connect.NewError(connect.CodeInternal, …)` for every failure was
  rejected: it discards the code the source chose. Converting inside the plugin was rejected:
  authors would have to know which transport serves them.

## R9. TypeScript surface

- **Decision**:
  - Regenerate so that `src/generated/finfocus/v1/usage_pb.ts` exports `UsageSourceService`,
    `GetStatsRequestSchema`, `StatsMode`, and related symbols. Export it from `index.ts`.
  - Add `src/clients/usage-source.ts` with a `UsageSourceClient` class shaped like `RegistryClient`
    (`ClientConfig` with `baseUrl` and optional `transport`), exposing
    `getStats(request: GetStatsRequest): Promise<GetStatsResponse>`.
  - Add `src/utils/usage-subjects.ts` mirroring the Go constants as `const` string exports.
  - Test with vitest + msw against `…/finfocus.v1.UsageSourceService/GetStats`.
- **Rationale**: Follows the auxiliary-client and `usage-profile.ts` precedents. There is no
  client-side validation, because no TS consumer exists yet (P3).

## R10. Documentation placement

- **Decision**:
  - `docs/usage-source.md` is the canonical semantics page (FR-006): subject keys, kinds, metrics,
    units per mode, the selector's reserved `namespace` key, priceable tagging with an AWS, Azure,
    and GCP node plus an EKS control plane example, unpriceable nodes, duplicate-row aggregation,
    and error codes.
  - The `sdk/go/pluginsdk/README.md` capability table gets a `UsageSourceProvider` row, plus a
    "Usage-only plugins" subsection (FR-019). Its example embeds `*pluginsdk.BasePlugin` (which
    supplies the four required `Plugin` cost methods), implements `GetStats`, and sets
    `WithCapabilities(USAGE_STATS)` explicitly. It states plainly that `GetStats` is the only
    method an author writes, because `ServeConfig.Plugin` is typed `Plugin` and the base methods
    come from `BasePlugin`.
  - `PLUGIN_DEVELOPER_GUIDE.md` gets a usage-source section.
  - `sdk/go/testing/README.md` documents the helpers and harness.
  - The root `README.md` lists `UsageSourceService` next to `CostSourceService` (project-structure
    tree, the "gRPC Service" bullet, and the "gRPC Service Interface" section) and links to
    `docs/usage-source.md`.
  - `sdk/typescript/README.md` gets a `UsageSourceClient` section under "Core API" and a line in
    "Features".
  - The root CLAUDE.md Active Technologies list gets an entry.
- **Rationale**: Constitution VII requires the root README, `docs/`, and the SDK READMEs to stay in
  sync in the same PR, and XIV requires documented names to match exported symbols. The capability
  table referenced by #505 lives in the pluginsdk README (~line 695), not the root README, but the
  root README's service list would otherwise describe only `CostSourceService`.

## R11. Request validation helper

- **Decision**: Add `ValidateGetStatsRequest(req *pbc.GetStatsRequest) error` to
  `sdk/go/testing/contract.go`, next to the other `Validate*Request` functions, with tests in
  `contract_test.go`. Rules, first violation wins:
  1. `nil` request → `ErrNilRequest`.
  2. Neither `start` nor `end` set → valid (run-rate).
  3. Only `start` set → `NewContractError("end", nil, ErrNilEndTime)`; only `end` set →
     `NewContractError("start", nil, ErrNilStartTime)`.
  4. `start` after `end` → `NewContractError("time_range", …, ErrInvertedStatsWindow)`, a new
     sentinel `errors.New("start time must not be after end time")`.

  `scope`, `selector`, and `metrics` are not checked: all are optional, and unknown metrics produce
  warnings rather than errors (FR-002). The test reference source (`referenceUsageSource`) calls the
  helper and converts its error to `codes.InvalidArgument`.
- **Rationale**: Constitution I requires validation for every protobuf message. Every existing
  request type has a validator in `contract.go`, and 050 added
  `ValidateResolveResourceTypesRequest`. Putting it in `contract.go` rather than the new
  `usage_source.go` lets Phase 2 fixtures use it before the US4 response validator exists.
- **Alternatives**: Reusing `ValidateTimeRange` was rejected. It requires both timestamps, a
  1-hour minimum, a 365-day maximum, and a start that is not in the future. The spec sets none of
  those rules for usage windows, and a historical usage window may be shorter than an hour.
  Validating in the SDK adapter before dispatch was rejected: FR-008 makes window handling the
  source's responsibility, and a source that supports only run-rate must still return its own
  `InvalidArgument` for a well-formed historical window.
