# Bug Assessment: Supports always fails with the default registry

- **Slug**: supports-default-registry
- **Created**: 2026-09-25
- **Source**: <https://github.com/rshade/finfocus-spec/issues/507>
  - Host: `github.com`
  - URL policy: allowlisted (read with `gh issue view 507`)
- **Verdict**: valid
- **Severity**: high

## Report (summarized)

Issue #507, "fix(pluginsdk): Supports always fails with the default registry, so plugin
Supports is never consulted" (labels: `bug`, `effort/small`).

`Server.Supports` checks the request's provider and region against `ServeConfig.Registry`
before it consults the plugin. The default, `DefaultRegistryLookup`, always returns `""`, so
every `Supports` call returns
`InvalidArgument("no plugin registered for provider … and region …")`. No plugin in the
ecosystem sets `ServeConfig.Registry`. As a result, a host never reaches any plugin's
`SupportsProvider` implementation.

FinFocus Core calls `Supports` with only `resource_type` set, and treats any error as
"supported" (it fails open). So every plugin is treated as supporting everything, and plugins
cannot decline resources. This blocks usage-only plugins (#505) and allocator plugins (#506).

Proposed fix: treat the default (or nil) registry as "no registry configured". Skip the
provider/region check and go straight to `SupportsProvider`, or to the existing default
response. A configured `RegistryLookup` keeps today's validation. The issue rejects changing
`DefaultRegistryLookup.FindPlugin` itself.

## Symptom

**Observed**: with no `ServeConfig.Registry`, every `Supports` call fails with `InvalidArgument`
("no plugin registered"). This happens even for an empty provider and region, which is what
hosts send today. The plugin's `Supports` method is never called.

**Expected**: with no registry configured, the server delegates to the plugin's
`SupportsProvider`. If the plugin doesn't implement it, the server returns `Supported: false`
with `DefaultSupportsNotImplementedReason` and no error. A configured `RegistryLookup` keeps
today's provider/region validation.

## Reproduction

1. Define a plugin that implements `SupportsProvider` and returns `Supported: true`.
2. Serve it with `ServeConfig{Plugin: p}` and no `Registry`, or construct it with
   `NewServer(p)`.
3. Call `Supports` with any `ResourceDescriptor`, for example
   `{Provider: "aws", Region: "us-east-1"}` or `{ResourceType: "aws:ec2/instance:Instance"}`.
4. The call fails with `codes.InvalidArgument` and the message
   `no plugin registered for provider "aws" and region "us-east-1"`. The plugin's method is not
   called.

The existing tests already encode this behavior: `TestSupports_NewServerUsesDefaultRegistry`
(`sdk_test.go:477`), `TestNewServerWithRegistry_NilRegistryUsesDefault` (`sdk_test.go:503`), and
the Connect client test at `client_test.go:182-191` ("Error expected because no plugin is
registered").

## Suspected Code Paths

All paths are under `sdk/go/pluginsdk/`. Line numbers are at HEAD `6e31a4f`; they moved about
13 lines from the issue's references because of the #504 commit.

- `sdk.go:254-262`: `DefaultRegistryLookup` always returns `""`. Its doc comment says every
  `Supports` call fails, and advises using a real implementation in production.
- `sdk.go:640-661` `Server.Supports`, step 1: returns `InvalidArgument` when
  `s.registry.FindPlugin(provider, region) == ""`. This runs before step 2
  (`SupportsProvider`), and `pluginName` is not used for anything else.
- `sdk.go:314` (`NewServer`) and `sdk.go:338-340` (`NewServerWithOptions`, when
  `registry == nil`): both install `&DefaultRegistryLookup{}`, so an unconfigured server
  always rejects.
- `sdk.go:1014-1015` `ServeConfig.Registry`: its doc says "optional", but in practice it is
  required for `Supports` to work.
- `connect.go:44-48` `ConnectHandler.Supports`: delegates to `Server.Supports`, so Connect has
  the same bug.
- `client.go:307-316` `Client.SupportsResourceType`: the SDK's own client sends only
  `ResourceType`, so it always gets an error under the default registry. This is more evidence
  that the behavior is not intended.
- `README.md:193-195`: "If nil, defaults to a no-op registry", with no mention that
  `Supports` is then always rejected.

## Root Cause Hypothesis

The registry check was written as a hard gate that assumed a real `RegistryLookup` in
production. The SDK then made the registry optional and added a no-op default, without
teaching `Supports` that the default means "no registry configured". The two-step check
therefore rejects every request before the plugin is consulted. **Confidence: high.** It is
visible in the code, and existing tests pin the buggy behavior.

## Proposed Remediation

**Preferred**: in `Server.Supports`, run the step-1 registry check only when a registry was
actually configured. Add a small predicate, for example `s.hasRegistry()`, that returns false
when `s.registry` is nil or is a `*DefaultRegistryLookup`, and gate step 1 on it. Leave the
constructors as they are, so the `registry` field and the public
`NewServerWithRegistry(p, nil)` semantics stay the same. Steps 2 onward are unchanged:
`SupportsProvider` delegation, the `DefaultSupportsNotImplementedReason` fallback, and the
auto-population of capabilities and legacy keys.

Update the docs to match:

- the `DefaultRegistryLookup` doc comment: it means "no registry; Supports delegates straight
  to the plugin";
- the `Supports` doc comment: the two-step check applies only with a configured registry;
- the `ServeConfig.Registry` comment in both `sdk.go` and `README.md`;
- the comments on `NewServer`, `NewServerWithRegistry`, and `NewServerWithOptions`.

**Alternatives**:

- *Make `DefaultRegistryLookup.FindPlugin` return the plugin's name.* The issue rejects this: it
  hides the "no registry" state instead of modelling it, and a lookup that accepts
  `FindPlugin("", "")` would surprise any caller that uses it directly.
- *Store `nil` in `s.registry` when unconfigured* and check only for nil. This is slightly
  cleaner, but it changes what `NewServer` and `NewServerWithOptions` store, and a caller that
  explicitly passes `&DefaultRegistryLookup{}` would still get the broken behavior. The type
  check covers both cases.

**Files likely to change**:

- `sdk/go/pluginsdk/sdk.go`: `Supports`, the predicate, and doc comments
- `sdk/go/pluginsdk/sdk_test.go`: rewrite `TestSupports_NewServerUsesDefaultRegistry` and
  `TestNewServerWithRegistry_NilRegistryUsesDefault` to the new behavior, and add
  table-driven cases
- `sdk/go/pluginsdk/client_test.go`: the Connect `Supports` test (`:182-191`) now expects
  success
- `sdk/go/pluginsdk/README.md`: the `ServeConfig.Registry` comment, plus a note on `Supports`
  defaults

**Tests to add or update** (table-driven where possible):

- With no registry (`NewServer`, `NewServerWithRegistry(p, nil)`, and an explicit
  `&DefaultRegistryLookup{}`), a plugin implementing `SupportsProvider` is called. Assert its
  answer is returned, both `Supported: true` and `Supported: false` with a reason.
- With no registry and no `SupportsProvider`, the response is `Supported: false` with
  `DefaultSupportsNotImplementedReason` and no error.
- An empty provider and region with only `resource_type` set, as hosts send today, does not
  error under the default registry.
- With a configured `mockRegistry`, the existing tests `TestSupports_InvalidProviderRegion…`
  and `TestSupports_NoPluginRegistered…` still return `InvalidArgument`, unchanged.
- Capabilities are still auto-populated (`CapabilitiesEnum` and the legacy map) in a response
  produced with no registry.
- Transports: an end-to-end gRPC `Serve` test with no `Registry`, and the Connect
  `client_test.go` path, both reach the plugin's `Supports`.
  `Client.SupportsResourceType` now returns the plugin's answer.
- A nil resource still returns `InvalidArgument` (unchanged).

## Risks & Considerations

- **Behavior change.** Hosts that relied on the `InvalidArgument` error now get real answers.
  Core fails open on errors today, so a plugin that returns `Supported: false` will now
  actually be skipped for those resources. That is the intended fix, but it changes routing,
  so call it out in the commit message.
- **Plugins without `SupportsProvider`** now return `Supported: false` with a reason instead of
  an error. Core should treat that as a clean "no". Worth checking against Core's
  `checkPluginSupports`, which is outside this repo: if Core treats `Supported: false` more
  strictly than an error, plugins without `SupportsProvider` could be dropped from routing.
  See Open Questions.
- **An explicit `&DefaultRegistryLookup{}` passed on purpose** to reject everything is treated
  as "no registry". That use is implausible, since it disables `Supports` entirely, and the
  updated doc comment documents it.
- **Tests encode the current behavior.** Three existing tests assert the bug; updating them is
  part of the fix, not a regression.
- There is no API or proto change and no performance impact: the fix adds one type assertion
  per call.

## Open Questions

- [NEEDS CLARIFICATION: how does FinFocus Core treat `Supported: false` with
  `DefaultSupportsNotImplementedReason` from a plugin that does not implement
  `SupportsProvider`?]
  - Today it receives an error and fails open ("supported").
  - After this fix it receives a clean `false`. If Core then skips the plugin, every plugin
    without `SupportsProvider` would stop being routed, which is a much larger change than
    intended.
  - The issue's acceptance criteria keep this response unchanged, so this repo follows the
    issue. Confirm Core's handling before release, or track it in the companion Core issue.
- Companion change, out of scope: Core's `checkPluginSupports` should also send `provider` and
  `region`, tracked in `rshade/finfocus`.
