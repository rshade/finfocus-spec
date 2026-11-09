# Research: Capability-Scoped Supports Requests

There were no `NEEDS CLARIFICATION` items in Technical Context. Each decision below resolves a design choice
the spec left to planning.

## R1: Wire field

- **Decision**: Add `PluginCapability capability = 2;` to `SupportsRequest` in
  `proto/finfocus/v1/costsource.proto`. `costsource.proto` already imports `enums.proto`. The field gets an
  inline comment that `PLUGIN_CAPABILITY_UNSPECIFIED` means "any capability, answer as before this field
  existed".
- **Rationale**: The issue fixes the number and the type. Field 2 is free, because the message has only
  `resource = 1`. Reusing the enum avoids a second list that would drift.
- **Alternatives considered**: A new `SupportsQuery` enum, rejected as a duplicate list. A `repeated` list of
  capabilities, rejected because a host asks about the one operation it is about to call, and a list makes
  the answer ambiguous.

## R2: How plugins read the capability

- **Decision**: Add an exported helper, `pluginsdk.RequestedCapability(req *pbc.SupportsRequest)
  pbc.PluginCapability`, in a new file `sdk/go/pluginsdk/supports_capability.go`.
  - A value accepted by `IsValidCapability` is returned unchanged.
  - `UNSPECIFIED` returns `UNSPECIFIED` with no log.
  - Any other value (unknown, negative, or out of range) returns `UNSPECIFIED`. The first time each distinct
    value is seen, it logs one warning with the raw `int32` through the zerolog global logger. A package-level
    `sync.Map` (`LoadOrStore`) records which values have been seen.
  - A nil request returns `UNSPECIFIED`, because `GetCapability` is nil-safe.
- **Rationale**: `SupportsProvider.Supports(ctx, *SupportsRequest)` already receives the whole request, so no
  interface changes (FR-004). The normalization follows `NormalizeUsageProfile` (FR-005), but the logging is
  deduplicated. `NormalizeUsageProfile` warns on every call, which is fine for a per-request profile but
  would flood logs on Supports, which a host calls on every routing decision. On the known and unspecified
  paths the helper is an integer compare, with zero allocations (FR-006). Only unknown values touch the map.
  Converting an `int32` key to `any` allocates for values of 256 and above, so the unknown path is exempt from
  the zero-allocation goal.
- **Alternatives considered**:
  - A new `CapabilitySupportsProvider` interface with a capability parameter. Rejected: it is a second way to
    do one thing, and capability inference would have to learn it.
  - Plugins call `req.GetCapability()` directly. They still can, but they would then see raw unknown values;
    the helper is the documented forward-compatible path.

## R3: How hosts send the capability (Go client)

- **Decision**: Add a sibling method, `func (c *Client) SupportsFor(ctx, resource *pbc.ResourceDescriptor,
  capability pbc.PluginCapability) (*pbc.SupportsResponse, error)`. `Supports` becomes a one-line delegate
  with `PLUGIN_CAPABILITY_UNSPECIFIED`. `SupportsResourceType` is unchanged. The client sends the value as
  given and does not normalize it, because the host owns the enum it was built with.
- **Rationale**: The issue suggests this name. The input checks are shared (FR-008), so the two methods cannot
  drift.
- **Alternatives considered**: Variadic options on `Supports` (`opts ...SupportsOption`). Rejected: changing an
  exported method's signature breaks callers that store it as a typed func value. CLAUDE.md records the same
  reason for not adding a variadic parameter to the `Run*Conformance` runners.

## R4: Server pass-through and logging

- **Decision**: `Server.Supports` already forwards `req` unchanged to the plugin over gRPC, and
  `ConnectHandler.Supports` calls `Server.Supports`. So FR-010 needs no code, only tests on both transports.
  For FR-011, add `FieldCapability = "capability"` to `logging.go`, and add
  `Str(FieldCapability, req.GetCapability().String())` to the "Supports handler error" log.
- **Rationale**: It only touches the error path, so the success path is unchanged.
- **Alternatives considered**: Logging the capability on every Supports call. Rejected as noise on a hot
  routing RPC.

## R5: Contract validator

- **Decision**: `plugintesting.ValidateSupportsRequest` stays as it is. It accepts every capability value
  (FR-012). A new test pins that an unknown value such as `PluginCapability(999)` is accepted.
- **Rationale**: Rejecting unknown values would break newer hosts calling older plugins. That is the
  forward-compatibility failure the spec rules out.

## R6: Reference mock

- **Decision**: Add a `UnsupportedCapabilities []pbc.PluginCapability` field to `MockPlugin`. After the
  existing provider and resource checks pass, a request whose capability is in the list gets
  `Supported: false` and `Reason: "capability <NAME> is not supported for <resource_type>"`. A nil or empty
  list ignores the capability (the default, FR-013).
- **Rationale**: The mock needs a capability-specific answer to show SC-001 end to end, and to exercise
  the conformance check against a plugin that does answer per capability. Running the check after the
  resource checks keeps resource-level reasons unchanged.

## R7: Conformance check

- **Decision**: Add a new Standard-level check, `RPCCorrectness_SupportsCapability`, in a new file
  `sdk/go/testing/supports_capability.go`. It registers through `RPCCorrectnessTests()` next to
  `recommendationsSpotTest()`. Using `harness.SampleResource()`, it does the following:
  1. Sends an unspecified request. An error fails; otherwise the response must pass `ValidateSupportsResponse`.
  2. Discovers capabilities with `harness.discoverCapabilities()`. A discovery error fails, as in the spot
     check. The query set is the declared capabilities that appear in the private `resourceScopedCapabilities`
     list. When the plugin declares nothing (legacy), it uses projected cost and actual cost.
  3. Sends one request per capability in the set. An error fails, naming the capability. A response that
     fails validation also fails, naming the capability. Answer values are never checked.
- **Rationale**: FR-014 and FR-015. A plugin that ignores the field answers both kinds of request the same way
  and passes. Standard matches the other recent request-field checks (585, 597). The check lives in its own
  file to keep it out of `rpc_correctness.go`, where open PR 676 is editing. Only the one registration line
  touches that file.
- **Resource-scoped list**: PROJECTED_COSTS (1), ACTUAL_COSTS (2), CARBON (3), RECOMMENDATIONS (4), ENERGY (7),
  WATER (8), PRICING_SPEC (9), ESTIMATE_COST (10), BATCH_COST (12), HISTORICAL_CARBON (20). The other values
  are excluded because they describe plugin-level or dataset services (DRY_RUN, BUDGETS,
  DISMISS_RECOMMENDATIONS, RESOLVE_RESOURCE_TYPES, USAGE_STATS, ALLOCATION, CONTRACT_COMMITMENTS, INVOICE_DATA,
  RECOMMENDATION_SCORING, COST_PARSE). The list stays private to `sdk/go/testing` and is mirrored in the docs.
  It is not exported, because nothing else needs it yet.
- **Alternatives considered**:
  - Basic level. Rejected, because the issue only requires that ignoring plugins pass, and Standard is
    where request-field checks live.
  - Asserting that a specific `true` implies `true` for unspecified. Rejected: the spec makes that guidance,
    not a rule.

## R8: TypeScript SDK

- **Decision**: `make generate` regenerates the TS bindings, so `SupportsRequest` gains `capability`. The
  client's `supports(req)` already forwards the whole request, so it needs no wrapper change. Add a vitest
  (`test/supports-capability.test.ts`) showing the field reaches the server, plus a README snippet.
- **Rationale**: FR-009 and constitution XIII. Per CLAUDE.md, client tests run with `npx vitest run`, because
  `tsc --noEmit` excludes `test/`.

## R9: Documentation

- **Decision**: Update these places:
  - The proto comment.
  - `PLUGIN_DEVELOPER_GUIDE.md`, under "Granular Capability Discovery (`Supports`)" and "Supports RPC": the
    unspecified rule, per-capability answering, the read helper, the host call, and the consistency guidance.
  - `sdk/go/pluginsdk/README.md`, in the SupportsProvider section: the helper and `SupportsFor`.
  - `sdk/go/testing/README.md` and `docs/conformance.md`, if it lists checks: the new check and the mock field.
  - `sdk/typescript/README.md`, the TS SDK README (the client package has none): the snippet.
  - `CLAUDE.md`, `AGENTS.md`, and `GEMINI.md`: Active Technologies, Recent Changes, and a pattern note. These
    are edited by hand, never by `update-agent-context.sh`.
- **Rationale**: FR-016 and FR-017, plus constitutions VII and XIV. README Go examples must compile; check
  them by building them.

## R10: Performance and versioning

- **Decision**: Add `BenchmarkRequestedCapability`, covering the known path and the unspecified path, with
  0 allocs/op as the gate (SC-005). The cost-RPC hot paths are unchanged. The version bump is minor through
  release-please, and `SpecVersion` is never edited by hand. Verify with `buf breaking` against a
  `git archive origin/main` copy, which is the worktree workaround from CLAUDE.md, and also against the CI URL.
- **Rationale**: Constitutions VI and VIII.
- **Not affected**: There is no new RPC or capability value, so the manifest and registry schema drift tests
  (`TestSchemaDrift`), `maxValidCapability`, and legacy capability names need no change. `SupportsRequest` is
  not a PricingSpec message, so the JSON schema is unchanged.
