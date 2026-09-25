# Bug Fix: PluginInfoProvider responses advertise no inferred capabilities

- **Slug**: provider-capability-backfill
- **Fixed**: 2026-09-25
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

`handleProviderPluginInfo` now works on a copy of the provider's response. When the provider
leaves `Capabilities` empty, the copy gets the server's inferred set. Legacy `supports_*` keys
the provider did not set are then added. `Serve` now advertises `RESOLVE_RESOURCE_TYPES` when a
`TypeRegistry` is configured and the capabilities were inferred rather than listed explicitly.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `sdk/go/pluginsdk/sdk.go` | modified | Provider path: clone, fill-if-empty, backfill legacy keys; new helpers `withLegacyCapabilityMetadata`, `logCapabilityDrift`, `setTypeRegistry`; new `capabilityDriftOnce` field; doc comments |
| `sdk/go/pluginsdk/capabilities_provider_test.go` | added test | Table-driven provider-path, immutability, drift-log, TypeRegistry, `Supports`, and end-to-end `Serve` tests |
| `sdk/go/pluginsdk/README.md` | modified | Both "Dynamic Metadata" sections state the inherit/replace rule; the TypeRegistry section notes the advertised capability |

## Diff Highlights

```go
out, ok := proto.Clone(resp).(*pbc.GetPluginInfoResponse)
...
if len(out.GetCapabilities()) == 0 {
    out.Capabilities = slices.Clone(s.globalCapabilities)
} else {
    s.logCapabilityDrift(out.GetCapabilities())
}
out.Metadata = s.withLegacyCapabilityMetadata(out.GetCapabilities(), out.GetMetadata(), false)
```

```go
server.setTypeRegistry(config.TypeRegistry,
    config.PluginInfo != nil && len(config.PluginInfo.Capabilities) > 0)
```

`handleConfiguredPluginInfo` now calls the same helper with `overwrite=true`, so its existing
behavior is unchanged: computed keys still overwrite configured ones.

## Tests Added or Updated

All in `sdk/go/pluginsdk/capabilities_provider_test.go`:

- `TestGetPluginInfo_ProviderCapabilityBackfill`:
  - empty capabilities inherit `GetGlobalCapabilities()`, including `RECOMMENDATIONS` and
    `RESOLVE_RESOURCE_TYPES`, plus the matching `supports_*` keys;
  - nil metadata is allocated for the backfilled keys;
  - an explicit list of the four base capabilities comes back unchanged, with no `BUDGETS`, no
    `supports_budgets`, and no `max_batch_size`;
  - a provider-set `supports_recommendations=false` is not overwritten;
  - non-capability metadata (`region`, `type`) is preserved;
  - `max_batch_size` is added when `BATCH_COST` is backfilled or listed explicitly.
- `TestGetPluginInfo_ProviderResponseNotMutated`: a shared package-level response is unchanged,
  checked with `proto.Equal` after 8 concurrent calls under `-race`.
- `TestGetPluginInfo_ProviderCapabilityDriftLoggedOnce`: the Debug drift log appears exactly
  once across two calls and names the omitted `BUDGETS`.
- `TestServer_SetTypeRegistryCapability`: covers four cases (no registry; registry on
  inferred capabilities; registry with explicit capabilities; resolver plugin not duplicated).
- `TestSupports_TypeRegistryCapability`: `Supports` also advertises the registry capability.
- `TestServe_TypeRegistryAdvertisesResolveResourceTypes`: end-to-end over real gRPC with an
  injected listener, for inferred and for explicit `PluginInfo.Capabilities`.

## Local Verification

- `go test -race ./sdk/go/pluginsdk/`: ok
- `go test ./...`: all packages ok
- `golangci-lint run ./...`: 0 issues
- `npx markdownlint-cli2 sdk/go/pluginsdk/README.md`: 0 issues
- The existing regression tests `TestGetPluginInfo`, `TestGetPluginInfo_AutoDiscovery`, and
  `TestGetPluginInfo_BackwardCompatibility_CapabilitiesEnumAndStringMap` pass unchanged.

## Deviations from Assessment

- The TypeRegistry logic is a `Server.setTypeRegistry` method rather than inline code in
  `Serve`, so it can be unit-tested without starting a server. `Serve` still covers it end to
  end.
- The test helper is named `baseCapabilityList()` because `baseCapabilities` is already a
  package constant in `plugin_info.go`.
- No `CHANGELOG.md` edit. release-please generates it, so the behavior change goes in the
  `fix(pluginsdk):` commit message instead.
- `max_batch_size` is still always set by the server, even when the provider supplied one,
  which matches the previous behavior. Only the `supports_*` keys defer to the provider.

## Follow-ups

- `finfocus-plugin-aws-public`: after the SDK bump, remove the hand-built legacy metadata and
  keep the explicit capability lists. Track this in that repo.
