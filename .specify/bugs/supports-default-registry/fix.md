# Bug Fix: Supports always fails with the default registry

- **Slug**: supports-default-registry
- **Fixed**: 2026-09-25
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

`Server.Supports` now runs the provider/region registry check only when a real
`RegistryLookup` is configured. With no registry (nil or `DefaultRegistryLookup`), the request
goes straight to the plugin's `SupportsProvider`, or to the unchanged not-implemented default.
Before, every request was rejected with `InvalidArgument`.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `sdk/go/pluginsdk/sdk.go` | modified | New `Server.hasRegistry()`; step 1 of `Supports` is gated on it; doc comments on `RegistryLookup`, `DefaultRegistryLookup`, `Supports`, `ServeConfig.Registry`, and the three constructors |
| `sdk/go/pluginsdk/supports_registry_test.go` | added test | No-registry delegation matrix; configured-registry validation; nil resource; gRPC `Serve` end to end |
| `sdk/go/pluginsdk/sdk_test.go` | removed tests | `TestSupports_NewServerUsesDefaultRegistry` and `TestNewServerWithRegistry_NilRegistryUsesDefault` asserted the bug; replaced by the matrix above |
| `sdk/go/pluginsdk/client_test.go` | modified test | `TestClient_Supports` (Connect) now uses a `SupportsProvider` plugin and asserts real answers, including `SupportsResourceType` |
| `sdk/go/pluginsdk/README.md` | modified | The `ServeConfig.Registry` comment describes the nil behavior |

## Diff Highlights

```go
// Step 1: Registry lookup - validate provider/region combination
if s.hasRegistry() && s.registry.FindPlugin(provider, region) == "" {
    return nil, status.Errorf(codes.InvalidArgument, "no plugin registered for provider %q and region %q", ...)
}
```

```go
func (s *Server) hasRegistry() bool {
    if s.registry == nil {
        return false
    }
    _, isDefault := s.registry.(*DefaultRegistryLookup)
    return !isDefault
}
```

## Tests Added or Updated

- `supports_registry_test.go::TestSupports_NoRegistryDelegatesToPlugin` runs 4 constructors × 4
  cases:
  - Constructors: `NewServer`, `NewServerWithRegistry(p, nil)`, an explicit
    `&DefaultRegistryLookup{}`, and `NewServerWithOptions(p, nil, nil, nil)`.
  - Cases: the plugin supports; the plugin declines with a reason; an empty provider and
    region, as hosts send today; no `SupportsProvider`, which gives `Supported: false` with
    `DefaultSupportsNotImplementedReason` and no error.
  - Every case also asserts that `CapabilitiesEnum` and the legacy map are still populated.
- `supports_registry_test.go::TestSupports_ConfiguredRegistryStillValidates`: with a
  configured registry, a registered request returns OK, while an unregistered region and an
  empty provider/region both return `InvalidArgument`.
- `supports_registry_test.go::TestSupports_NilResourceStillRejectedWithoutRegistry`: a nil
  resource still returns `InvalidArgument`.
- `supports_registry_test.go::TestServe_SupportsWithoutRegistryReachesPlugin`: over gRPC, a
  `Serve` with no `Registry` returns the plugin's answer.
- `client_test.go::TestClient_Supports`: over Connect, the plugin answers per region, and
  `SupportsResourceType` returns the plugin's answer instead of an error.
- Unchanged and passing: `TestSupports_InvalidProviderRegionReturnsInvalidArgument`,
  `TestSupports_NoPluginRegisteredReturnsInvalidArgument`, `TestSupports_AutoDiscovery`, and
  the other `TestSupports_*` tests that use `mockRegistry`.

## Local Verification

- Red first: before the `sdk.go` change, `go test -run 'TestSupports|TestServe_Supports|TestClient_Supports'`
  failed `TestSupports_NoRegistryDelegatesToPlugin`,
  `TestServe_SupportsWithoutRegistryReachesPlugin`, and `TestClient_Supports`. The
  configured-registry tests passed.
- `go test -race ./sdk/go/pluginsdk/`: ok
- `go test ./...`: 15 packages ok
- `golangci-lint run ./...`: 0 issues, after wrapping one line that golines flagged
- `npx markdownlint-cli2 sdk/go/pluginsdk/README.md`: 0 issues

## Deviations from Assessment

- The two `sdk_test.go` tests that asserted the bug were **removed**, not rewritten in place.
  The constructor matrix in `supports_registry_test.go` covers both `NewServer` and
  `NewServerWithRegistry(p, nil)`.
- The README had no separate section on `Supports` defaults. Only the `ServeConfig.Registry`
  comment described the old behavior, so that is the only README change. No other non-spec
  docs referred to it.

## Follow-ups

- Confirm how FinFocus Core handles `Supported: false` with
  `DefaultSupportsNotImplementedReason` (the assessment's open question) before release.
  Previously, plugins without `SupportsProvider` returned an error, which Core treats as
  "supported".
- Companion change in Core: `checkPluginSupports` should send `provider` and `region`, tracked
  in `rshade/finfocus`.
