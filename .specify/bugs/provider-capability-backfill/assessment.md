# Bug Assessment: PluginInfoProvider responses advertise no inferred capabilities

- **Slug**: provider-capability-backfill
- **Created**: 2026-09-25
- **Source**: <https://github.com/rshade/finfocus-spec/issues/504>
  - Host: `github.com`
  - URL policy: allowlisted (read with `gh issue view 504`)
- **Verdict**: valid
- **Severity**: high

## Report (summarized)

Issue #504, "fix(pluginsdk): backfill inferred capabilities when a PluginInfoProvider omits
them" (labels: `bug`, `effort/small`).

When a plugin implements `PluginInfoProvider`, `Server.GetPluginInfo` returns the provider's
response verbatim. The SDK never merges in the capabilities it inferred from the plugin's
interfaces (`globalCapabilities`). It also never adds the legacy `supports_*` metadata keys. A
plugin that copies the README's "Dynamic Metadata" example therefore advertises no
capabilities, and FinFocus Core never calls RPCs that the plugin implements.

`finfocus-plugin-aws-public` hit this bug. `resolve_resource_types` was never advertised, so
Core v0.3.7 never called `ResolveResourceTypes`, and `--terraform-state` returned $0. The
plugin worked around it in commit `480ceb7`.

There is a second, related gap. `ServeConfig.TypeRegistry` makes `ResolveResourceTypes` answer
correctly, but the SDK never advertises `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES` for it.

The issue sets the merge rule: fill-if-empty, not union, for the capability enum. The legacy
keys are always backfilled, and the provider's value wins when both set the same key.

## Symptom

**Observed**: a plugin that implements `PluginInfoProvider` and leaves `Capabilities` empty gets
a `GetPluginInfo` response with no capabilities and no `supports_*` keys. A plugin that relies
only on `ServeConfig.TypeRegistry` never advertises `RESOLVE_RESOURCE_TYPES`, on any path.

**Expected**: both situations advertise the inferred capabilities and the matching legacy keys,
the same way the `ServeConfig.PluginInfo` path and `Supports` already do.

## Reproduction

1. Define a plugin that implements `Plugin`, `RecommendationsProvider`, and `PluginInfoProvider`.
   Its `GetPluginInfo` returns a name, a version, and `SpecVersion`, and no `Capabilities`.
2. Call `NewServer(plugin).GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})`.
3. The response has no `Capabilities` and no `supports_recommendations` key.
   `server.GetGlobalCapabilities()` does contain `RECOMMENDATIONS`.
4. TypeRegistry gap: run `Serve` with `TypeRegistry` set, using a plugin that does not implement
   `ResolveResourceTypesProvider` and a `PluginInfo` without `Capabilities`.
   `GetPluginInfo` and `Supports` both leave out `RESOLVE_RESOURCE_TYPES`, although
   `ResolveResourceTypes` returns mappings.

## Suspected Code Paths

All paths are under `sdk/go/pluginsdk/`. The line numbers match the issue, which was checked
against HEAD `e0f5afe`.

- `sdk.go:415-436` `GetPluginInfo`: when the plugin implements `PluginInfoProvider`, the server
  delegates to the provider first.
- `sdk.go:439-482` `handleProviderPluginInfo` (the defect):
  - It never reads `s.globalCapabilities`.
  - It never calls `CapabilitiesToLegacyMetadataWithWarnings`.
  - It assigns `resp.Metadata = meta` (`:478`), which changes the message the plugin owns. The
    code builds a new map, but it stores that map on the provider's struct, so a shared or
    cached response is still modified, and concurrent calls race.
- `sdk.go:484-535` `handleConfiguredPluginInfo`: the correct sibling, and the model for the fix.
  It uses the explicit capabilities if set, otherwise `globalCapabilities`, then fills in the
  legacy keys and `max_batch_size`.
- `sdk.go:624-653` `Supports`: fill-if-empty from `globalCapabilities`, then regenerates the
  legacy map from the enum.
- `sdk.go:324-353` `NewServerWithOptions`: computes `globalCapabilities`; an explicit
  `PluginInfo.Capabilities` wins.
- `sdk.go:1148-1151` `Serve`: the server is built first, and `typeRegistry` is assigned only
  after `globalCapabilities` has already been computed.
- `sdk.go:887-897` `ResolveResourceTypes`: falls back to `s.typeRegistry`, which is the RPC
  that works but is never advertised.
- `plugin_info.go:235-280` `inferCapabilities`: checks interfaces only, so it cannot see
  `TypeRegistry`.
- `sdk.go:205-217` (`PluginInfoProvider` doc comment) and `sdk.go:397-414` (`GetPluginInfo`
  priority comment): neither states the capability rule.
- `README.md:551-569` and `README.md:599-614`: the two `PluginInfoProvider` examples set no
  `Capabilities`, which leads plugins straight into the bug.

## Root Cause Hypothesis

`handleProviderPluginInfo` was written only as a validation pass-through. Capability
inheritance was added later, to `handleConfiguredPluginInfo` and `Supports`, and was never
extended to the provider path. As a result, one server can advertise different capabilities in
`GetPluginInfo` and in `Supports`. Separately, `Serve` assigns `TypeRegistry` after capability
inference has run, and inference only checks interfaces, so the registry can never add
`RESOLVE_RESOURCE_TYPES`. **Confidence: high.** Both gaps are visible in the code, and the
downstream plugin's workaround confirms them.

## Proposed Remediation

**Preferred**:

1. **Shared helper.** Extract the legacy-metadata merge into a helper that both
   `handleConfiguredPluginInfo` and `handleProviderPluginInfo` call. For example,
   `s.mergeLegacyCapabilityMetadata(caps, meta, overwrite bool) map[string]string`. It:
   - calls `CapabilitiesToLegacyMetadataWithWarnings`;
   - logs each warning;
   - allocates the map if it is nil;
   - adds `max_batch_size` when `BATCH_COST` is present.

   The configured path keeps its current behavior, where computed keys overwrite configured
   ones. On the provider path, keys the provider already set win. The overwrite flag, or two
   small wrappers, captures that difference.
2. **Provider path.** In `handleProviderPluginInfo`, after validation:
   - Clone the response: `out := proto.Clone(resp).(*pbc.GetPluginInfoResponse)`.
   - If `out.Capabilities` is empty, set it to `slices.Clone(s.globalCapabilities)`.
   - Backfill the legacy keys only where they are absent.
   - Add `max_batch_size` when `BATCH_COST` is in the final set.

   This removes the in-place `resp.Metadata` assignment. Optionally, log at Debug level, once
   per server through a `sync.Once`, when an explicit list leaves out inferred capabilities.
3. **TypeRegistry.** In `Serve`, after `server.typeRegistry = config.TypeRegistry`, append
   `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES` to `server.globalCapabilities` when all of these
   hold:
   - the registry is non-nil;
   - the capability is not already present;
   - `config.PluginInfo == nil || len(config.PluginInfo.Capabilities) == 0`.

   `Supports` and `handleConfiguredPluginInfo` read `globalCapabilities`, so they pick up the
   capability automatically.
4. **Docs.** Update both README examples and the two doc comments. Omitting `Capabilities`
   inherits the inferred set; setting it replaces the set, and there is no union.

**Alternatives**:

- *Union* of explicit and inferred capabilities. Rejected: a method set is not the same as
  support. The aws-public router has `GetBudgets` and `DismissRecommendation` stubs that only
  return `Unimplemented`, so a union would advertise RPCs that always fail.
- *Pass the registry into `NewServerWithOptions` as an option* and include it in inference.
  This is cleaner in principle, but it changes a public constructor signature, or needs a new
  options type. The post-construction append is smaller and stays inside `Serve`.

**Files likely to change**:

- `sdk/go/pluginsdk/sdk.go`
- `sdk/go/pluginsdk/sdk_test.go` and/or `sdk/go/pluginsdk/capabilities_test.go`
- `sdk/go/pluginsdk/README.md`

**Tests to add or update** (table-driven, reusing `mockPluginInfoPlugin` at
`sdk_test.go:1111` and `mockCapabilityPlugin` at `capabilities_test.go:16`):

- A provider with empty `Capabilities` gets `GetGlobalCapabilities()`, including
  `RECOMMENDATIONS` and `RESOLVE_RESOURCE_TYPES` for a plugin that implements both, plus the
  matching `supports_*=true` keys.
- A provider with an explicit, non-empty list gets that list back unchanged. A plugin that
  satisfies `BudgetsProvider` but declares only the four base capabilities has no `BUDGETS` and
  no `supports_budgets` key.
- A key the provider sets explicitly, such as `supports_recommendations=false`, is not
  overwritten. Other provider metadata, such as `region` and `type`, is preserved.
- `max_batch_size` is present when `BATCH_COST` arrives through the backfill.
- Immutability: the provider returns a shared package-level response; after two calls,
  `proto.Equal` against a snapshot passes. Run it under `-race`.
- `Serve` with `TypeRegistry` and a non-resolver plugin advertises `RESOLVE_RESOURCE_TYPES` in
  both `GetPluginInfo` and `Supports`, and does not when `PluginInfo.Capabilities` is explicit.
  Use an injected `ServeConfig.Listener`, per project convention.
- Regression: `TestGetPluginInfo` (`sdk_test.go:1027`), `TestGetPluginInfo_AutoDiscovery`
  (`capabilities_test.go:98`), and
  `TestGetPluginInfo_BackwardCompatibility_CapabilitiesEnumAndStringMap` (`sdk_test.go:1258`)
  still pass.

## Risks & Considerations

- **Behavior change.** Provider plugins that advertised nothing now advertise their inferred
  set. That is the intended fix, but hosts may start calling RPCs they skipped before. A plugin
  whose method stubs return `Unimplemented`, and that relied on the old silence, must now
  declare an explicit list. Call this out in the commit message.
- **An empty list cannot mean "zero capabilities".** An empty slice means "inherit". This
  matches the other two paths, and every plugin has the four base capabilities anyway.
- **Precedence differs by design.** On the provider path, provider keys win. On the configured
  path, computed keys overwrite configured ones, as they do today. The shared helper must keep
  both behaviors, or this becomes a silent behavior change for configured plugins.
- **Concurrency.** Today, `resp.Metadata = meta` writes to a message the plugin may share
  across goroutines. The clone fixes this latent data race.
- **`proto.Clone` cost** is negligible for a metadata RPC.
- **CHANGELOG.** The issue asks for a `CHANGELOG.md` entry, but this repo generates it with
  release-please (see `CLAUDE.md`). Use a `fix(pluginsdk):` commit that describes the behavior
  change, not a hand edit.
- **Web/Connect mode.** `serveConnect` receives the same `*Server`, so the `Serve` fix covers
  both transports.

## Open Questions

- Should the Debug-level "explicit list omits inferred capabilities" log ship now, or wait?
  The issue marks it optional. Recommendation: include it; it is cheap and uses `sync.Once`.
- Downstream follow-up, out of scope: once the SDK bump ships, `finfocus-plugin-aws-public` can
  remove its hand-built legacy metadata and keep its explicit capability lists.
