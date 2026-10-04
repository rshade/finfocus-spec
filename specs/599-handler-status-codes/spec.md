# Feature Specification: Keep Handler Status Codes in the Plugin SDK Server

**Feature Branch**: `599-handler-status-codes`

**Created**: 2026-10-04

**Status**: Draft

**Input**: User description: "fix(pluginsdk): keep handler gRPC status codes instead of rewrapping
them as Internal. pluginsdk.Server turns every error a plugin handler returns into codes.Internal
for DryRun, Supports, GetRecommendations, GetBudgets, DismissRecommendation, BatchCost (custom
handler), and ResolveResourceTypes. Plugins that embed the generated stub report Internal for RPCs
they do not implement, and deliberate codes (InvalidArgument, NotFound, Unavailable, Unimplemented)
are lost. Keep Internal and the generic message for plain errors; pass real statuses through; one
helper; log as today; document the rule in the pluginsdk README. Closes #626"

## Clarifications

### Session 2026-10-04

- Q: When a provider handler returns `Unimplemented` (always true for the embedded generated stub),
  should the SDK pass it through or fall back? → A: Fall back. The SDK takes the same path it takes
  for a plugin that does not implement the provider interface, so a plugin that embeds the stub
  behaves on the wire exactly like one that does not.
- Q: Stub-embedding plugins also advertise capabilities they do not implement. In scope? → A: No.
  Capability inference is a separate change, listed as a follow-up in the pull request.

### Issue claims verified against the branch base (d819094 plus PR 627)

- **Confirmed**: the seven rewrap sites (`sdk.go` lines 711, 812, 892, 948, 999, 1041, 1104). A
  handler's `InvalidArgument` or `NotFound` reaches the client as `Internal`; a plain error's detail
  is not exposed.
- **Wider than reported**: the issue names two RPCs that a stub-embedding plugin breaks
  (`GetBudgets`, `DismissRecommendation`). Measured with a plugin that embeds
  `UnimplementedCostSourceServiceServer` and is served by `NewServer`, **seven** RPCs return
  `Internal`: `Supports`, `GetRecommendations`, `GetBudgets`, `DismissRecommendation`,
  `ResolveResourceTypes`, `BatchCost`, and `GetPluginInfo`. The stub's methods satisfy each provider
  interface, so the SDK calls the stub instead of its own default.
- **Missing site**: `GetPluginInfo` also rewraps a provider error (`"plugin failed to retrieve
  metadata"`, line 530). For a stub-embedding plugin this hides both a configured
  `ServeConfig.PluginInfo` and the legacy `Unimplemented` answer hosts fall back on.
- **Out of scope, recorded**: a stub-embedding plugin advertises 9 capabilities where a plain one
  advertises 4.
- **Dependency checked**: the `Supports` fallback returns `Supported: false` with
  `DefaultSupportsNotImplementedReason`. Core fails open on that response since rshade/finfocus#1512
  (Core v0.4.2), so routing does not drop these plugins.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A handler's deliberate status reaches the host (Priority: P1)

A plugin handler rejects a request on purpose, for example with `InvalidArgument` for a bad
descriptor or `NotFound` for an unknown resource. The host receives that code and the handler's
message, so it can tell a bad request from a plugin fault and route a `NotFound` to another plugin.

**Why this priority**: This is the defect the issue reports; hosts cannot classify errors from
eight RPCs today.

**Independent Test**: For each wrapped RPC, a handler returning `status.Error(codes.InvalidArgument,
"m")` yields `InvalidArgument` with message `m` at the client, over gRPC and over Connect.

**Acceptance Scenarios**:

1. **Given** a handler that returns a gRPC status other than `Unknown` and `Unimplemented`,
   **When** the host calls the RPC, **Then** it receives that status code and message unchanged.
2. **Given** a handler that returns a plain Go error, a context error, or an `Unknown` status,
   **When** the host calls the RPC, **Then** it receives `Internal` with the same generic message as
   today, and the error's text is not in the response.
3. **Given** a handler error that is not `Unimplemented`, **When** the server handles it, **Then**
   the original error is logged server-side as today.

---

### User Story 2 - Embedding the generated stub is safe (Priority: P1)

A plugin author embeds `UnimplementedCostSourceServiceServer`, the standard forward-compatibility
pattern, and implements only the cost RPCs. Every optional RPC answers exactly as it would for a
plugin that does not embed the stub.

**Why this priority**: Embedding the stub is the documented pattern, and today it turns seven RPCs
into server faults, fails the SDK's own `GetBudgets` conformance check, and breaks `GetPluginInfo`.

**Independent Test**: Serve one plugin that embeds the stub and one that does not, with the same
`ServeConfig`; call each of the seven RPCs on both; the codes and responses match.

**Acceptance Scenarios**:

1. **Given** a stub-embedding plugin, **When** the host calls `GetBudgets` or
   `DismissRecommendation`, **Then** it receives `Unimplemented`, and the plugin passes the
   `RPCCorrectness_GetBudgetsRPC` conformance check.
2. **Given** a stub-embedding plugin, **When** the host calls `Supports`, `GetRecommendations`,
   `ResolveResourceTypes`, or `BatchCost`, **Then** it receives the SDK default: the default
   `Supports` response, an empty recommendation list, the `TypeRegistry` result or an empty
   response, and the SDK's per-resource batch.
3. **Given** a stub-embedding plugin with `ServeConfig.PluginInfo` set, **When** the host calls
   `GetPluginInfo`, **Then** it receives the configured info; without it, `Unimplemented`.

---

### User Story 3 - Authors know the rule (Priority: P2)

A plugin author reads the plugin SDK README and learns which errors pass through, which become
`Internal`, and that returning `Unimplemented` means "use the SDK default".

**Independent Test**: The README section states all three rules and its example compiles.

### Edge Cases

- A status error wrapped with `fmt.Errorf("...: %w", err)` is still a real status; it passes
  through with the code the status carries.
- A custom error type that implements `GRPCStatus()` (as the SDK's own `InvalidArgument` errors do)
  passes through.
- A custom error whose `GRPCStatus()` reports `OK` (`status.Error(codes.OK, ...)` itself is nil) has
  no usable status: `Internal` with the generic message.
- `DryRun`: its handler interface (`HandleDryRun`) is not a stub method, but a handler that returns
  `Unimplemented` gets the same not-a-provider answer (`Unimplemented`, "plugin does not support
  DryRun").
- Nil-response and invalid-response guards (`plugin returned a nil response`, incomplete plugin info)
  are SDK checks, not handler errors; they stay `Internal`.
- RPCs that already return handler errors unchanged (cost RPCs, optional services) are not touched.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: For the eight wrapped RPCs (`DryRun`, `Supports`, `GetRecommendations`, `GetBudgets`,
  `DismissRecommendation`, custom-handler `BatchCost`, `ResolveResourceTypes`, provider
  `GetPluginInfo`), a handler error that carries a gRPC status with a code other than `OK`,
  `Unknown`, and `Unimplemented` MUST reach the client with that code and message.
- **FR-002**: A handler error that carries no gRPC status, or a status with code `OK` or `Unknown`,
  MUST reach the client as `Internal` with the message each RPC uses today; the error's text MUST
  NOT appear in the response.
- **FR-003**: A handler error with code `Unimplemented` MUST be answered by that RPC's
  not-a-provider path, exactly as if the plugin did not implement the provider interface.
- **FR-004**: The classification MUST live in one helper used by all eight RPCs.
- **FR-005**: Every pass-through and internal handler error MUST still be logged server-side at
  error level, as today. A fallback (FR-003) MUST be logged at debug level only, so a stub-embedding
  plugin does not log an error on every call.
- **FR-006**: The gRPC and Connect transports MUST report the same codes and messages.
- **FR-007**: The plugin SDK README MUST document the three rules (FR-001 to FR-003) with an
  example that compiles, and say that embedding the generated stub is safe.
- **FR-008**: Changes MUST NOT add provider-specific content; tests use neutral names.

### Key Entities

- **Handler error class**: pass-through (real status), fallback (`Unimplemented`), or internal
  (anything else).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 8 of 8 wrapped RPCs return a handler's `InvalidArgument` unchanged, over both
  transports.
- **SC-002**: A stub-embedding plugin and a plain plugin with the same configuration return the same
  code and response from all 7 optional RPCs it embeds the stub for.
- **SC-003**: 0 existing tests that expect `Internal` for a plain handler error change.
- **SC-004**: A stub-embedding plugin passes the Standard conformance level's
  `RPCCorrectness_GetBudgetsRPC` check.

## Assumptions

- Changing a code that hosts used to see as `Internal` into the handler's own code is the intended
  behavior change; it ships as a `fix`. Hosts that treated every non-OK result as a fault keep
  working; hosts that classify codes get correct answers.
- Context errors (`context.Canceled`, `context.DeadlineExceeded`) returned by a handler are plain
  errors and stay `Internal`, as today; mapping them is out of scope.
- Capability inference for stub-embedding plugins is out of scope (follow-up).
- Core v0.4.2 or later fails open on the default `Supports` response (rshade/finfocus#1512).
- No proto, generated code, TypeScript, or conformance-suite change is needed.
