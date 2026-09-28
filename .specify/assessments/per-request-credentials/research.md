# Idea Research: Per-Request Credential Passing

- **Slug**: per-request-credentials
- **Created**: 2026-09-27
- **Evidence confidence (overall)**: low

Code and spec citations below are high confidence. The claim that a second credential
model is needed is not. No production measurement of process memory or cold-start
latency is in this repository or on issue #220. Overall confidence stays low because
that missing measurement is what the idea itself says must decide the work.

## Users & Demand

- The only request in this repo is GitHub issue #220, opened 2025-12-30 by Richard Shade
  (`rshade`). It has no comments, assignees, or milestone. Labels are `enhancement`,
  `roadmap/future`, and `effort/large`. The body asks to measure per-org resource use and
  cold-start latency before implementing. — [source: <https://github.com/rshade/finfocus-spec/issues/220>]
  (confidence: high that this want was stated; low that anyone else has asked)
- `ROADMAP.md` still lists "Per-Request Credential Passing" (#220) under "Proposed for
  Discussion (Discovery)", effort large, as a multi-tenant optimization. It is not under
  Active Research. Nearby Active Research item #195 is authorization middleware, a
  different identity problem. — [source: `ROADMAP.md`] (confidence: high)
- No support tickets, usage counts, or host telemetry are stored here. Treating "many
  tenants, infrequent requests" as observed behavior would be a guess. — [ASSUMPTION]
  (confidence: high that this repo has no such data; low if read as a claim about
  production)

## Prior Art

- Spec `029-grpc-web-support` (created 2025-12-29, header status still `Draft`) makes
  per-organization plugin processes the default. User story 3 is P1: separate processes
  so one organization's credentials cannot leak to another. FR-016 says plugins MUST use
  ambient credentials from environment variables. FR-020 through FR-027 assign launch,
  routing, idle timeout (default 5 minutes), restart, and instance caps to an
  orchestrator. FR-021 says that orchestrator MUST pass tenant credentials via environment
  variables at launch, not via request headers. FR-028 says the SDK MUST include tenant
  context in request metadata for audit, and explicitly not credentials. Out of scope:
  "Per-request credential passing (see future enhancement issue #220)". Assumptions say
  the orchestrator is a web-platform concern (example: Pulumi Insights), not
  `finfocus-core`, and not this spec repo. — [source: `specs/029-grpc-web-support/spec.md`]
  (confidence: high)
- The same spec's research note chose environment variables because each instance is a
  process and cloud SDKs pick up ambient variables. It says `Serve` does not need special
  credential handling if it does not cache credentials globally. —
  [source: `specs/029-grpc-web-support/research.md`, section 5] (confidence: high)
- `specs/029-grpc-web-support/tasks.md` marks the feature complete for connect-go,
  CORS, the Go client, and a credential-cache audit (T027: "No global caching").
  Multi-tenant deployment documentation (T028) is still unchecked and "Deferred for
  future iteration". The spec header remains `Draft` while the task list says complete.
  — [source: `specs/029-grpc-web-support/tasks.md`, `spec.md` header] (confidence: high)
- Issue #189 ("discovery: gRPC-Web support for Browser Diagnostics") is closed
  (2025-12-30). Its body is Connect/gRPC-Web for browser clients. It does not describe
  per-request tenant credentials. #220 calls #189 the parent; #189 does not list #220.
  The per-org default was written into spec 029, which names #189 as its related issue,
  and 029 then deferred #220. — [source: <https://github.com/rshade/finfocus-spec/issues/189>,
  `specs/029-grpc-web-support/spec.md`] (confidence: high)
- Issue #195 ("research: Authorization Middleware (OIDC/IAM)") is open, with the same
  three labels as #220. It asks to carry OIDC or IAM identity in gRPC metadata and says
  the SDK must not verify tokens or manage refresh. That is caller identity for a plugin,
  not a pool of tenants sharing one process, and it is unfinished. Building #220 beside
  it risks two metadata stories for "how a plugin learns who the caller is." —
  [source: <https://github.com/rshade/finfocus-spec/issues/195>, `ROADMAP.md`]
  (confidence: high that both are open; medium that they would collide)
- Searched `sdk/go/pluginsdk` for `WithCredentials` and `ExtractCredentials`: no matches.
  Those names exist only in the issue text. They were not moved; they were never added.
  — [source: repository search] (confidence: high)
- `sdk/go/pluginsdk/env.go` reads port, log level, log format, log file, trace id, and
  test mode (`FINFOCUS_*`, with deprecated `PULUMICOST_*` fallbacks). It has no cloud
  credential variable. — [source: `sdk/go/pluginsdk/env.go`] (confidence: high)
- Incoming gRPC metadata handling in the SDK is the tracing interceptor. It reads
  `x-finfocus-trace-id` (`TraceIDMetadataKey`) and stores a trace id on the context. No
  tenant-id or credential metadata key was found under `sdk/go/pluginsdk`. —
  [source: `sdk/go/pluginsdk/logging.go`] (confidence: high)
- `ClientConfig` in `sdk/go/pluginsdk/client.go` covers base URL, protocol, HTTP client,
  timeout, and Connect options. It has no cloud-credential option. `Serve` in
  `sdk/go/pluginsdk/sdk.go` binds `127.0.0.1` unless the caller injects a listener, and
  `ServeConfig` has no credential field. — [source: `client.go`, `sdk.go`]
  (confidence: high)
- `PluginCapability` in `proto/finfocus/v1/enums.proto` runs from unspecified through
  `PLUGIN_CAPABILITY_ALLOCATION = 15`. None names credential passing or a shared pool.
  Request protos do not carry a credential message. `ERROR_CODE_INVALID_CREDENTIALS` is
  an error code only. Usage RPC comments mention unauthenticated calls when the plugin
  has no usable credentials; they do not define a credential channel. —
  [source: `proto/finfocus/v1/enums.proto`, `costsource.proto`, `usage.proto`]
  (confidence: high)
- `WebConfig.WithAllowCredentials` allows CORS cookies and authorization headers. It is
  not a cloud-credential API. The name is easy to confuse with the issue's
  `WithCredentials`. — [source: `sdk/go/pluginsdk/options.go`] (confidence: high)
- `PLUGIN_DEVELOPER_GUIDE.md` tells plugin authors to load API keys, OAuth tokens, or
  service-account credentials from environment variables or config files.
  `docs/PLUGIN_STARTUP_PROTOCOL.md` says to avoid passing secrets via environment
  variables when possible. That note conflicts with spec 029 FR-021 and is not
  implemented as a credential helper. — [source: those two docs] (confidence: high that
  both sentences exist; medium on which operators follow)
- Constitution principle IV: this repo defines the specification and plugin SDK, not the
  end-user application. Principle III: the spec does not calculate cost. #220's boundary
  ("transport/DX only; no backend state or math") fits III. A shared-pool router would
  not fit IV, and spec 029 already places that router outside this repo. —
  [source: `.specify/memory/constitution.md`, spec 029 assumptions] (confidence: high)

## Market & Context

- The documented way to cope today is one process per organization, started lazily,
  stopped after an idle timeout, and capped per node. Those rules are requirements on a
  host orchestrator. This repository does not contain that orchestrator. —
  [source: spec 029 FR-020–FR-027 and assumptions] (confidence: high)
- Doing nothing in this repo leaves that host-side model in place. Cold start is bounded
  on paper by SC-014 (under 5 seconds for the first request to a new tenant). That figure
  is a success criterion, not a measured result. — [source: spec 029 SC-014]
  (confidence: high that it is a target; low as a description of production)
- Issue #220 lists lower resource use, no cold start for rare tenants, and one process
  per plugin type as the gains of a shared pool. The same issue lists the costs: explicit
  credential handling, cloud SDKs that expect ambient credentials, a larger blast radius,
  cache isolation, and every plugin author implementing secret handling. —
  [source: issue #220] (confidence: high that these tradeoffs were asserted)

## Data & Constraints

- The "~50-100MB per process" figure appears in issue #220 and was not found in specs,
  benchmarks, or docs. It is unsourced. — [source: issue #220; repo search found no
  matching measurement] (confidence: low)
- Spec 029 SC-013 says an orchestrator can manage at least 100 concurrent plugin
  instances. SC-014 sets launch latency under 5 seconds. FR-024 sets a 5-minute default
  idle timeout. None of these are observations from a running host. —
  [source: spec 029] (confidence: high as written requirements)
- No benchmark in this repo measures plugin-process resident memory or process start
  time. — [source: repo search] (confidence: high that the bench is absent)
- Plugins bind loopback by default (`docs/PLUGIN_STARTUP_PROTOCOL.md`, `listenOnLoopback`
  in `sdk.go`). A shared process would still be local unless a host overrides the
  listener, but process isolation between tenants would be gone. —
  [source: those files] (confidence: high)
- Logging guidance warns that default log-file mode is world-readable and that attribute
  values must not be logged, because they can carry secrets. Any future credential on
  the request path would have to stay out of logs, traces, and metrics. Spec 029 SC-016
  already requires that. This note does not describe how. —
  [source: `sdk/go/pluginsdk/logging.go`, spec 029 SC-016] (confidence: high)

## Evidence Against the Idea

- The issue's own gate is unmet. It says not to implement until resource use and cold
  start are measured, and those measurements are not in the issue or the repo. —
  [source: issue #220] (confidence: high)
- Spec 029 already chose the opposite default, for security and for cloud-SDK ambient
  credentials, and pointed #220 at a future enhancement. The isolation story is a P1
  user story, not a nice-to-have. — [source: spec 029] (confidence: high)
- SDK helpers named in the issue would not, by themselves, reduce process count. Routing
  to a shared pool is orchestrator work, and spec 029 says that component is outside
  this repository. — [source: spec 029 assumptions, constitution IV] (confidence: high)
- There is no second requester. Roadmap placement is "proposed for discussion," with
  `effort/large`. — [source: issue labels, `ROADMAP.md`] (confidence: high)
- Issue #195 already proposes metadata-carried identity without the SDK managing secrets,
  and it is not decided. A per-request secret channel could fight that boundary. —
  [source: issue #195] (confidence: medium)
- The stated memory savings rest on an unsourced per-process size. If real processes are
  small, or idle timeout already reaps rare tenants, the large security and DX cost is
  not justified. — [ASSUMPTION about production size; the absence of a measurement is
  cited above] (confidence: low on the savings, high on the gap)

## Gaps & Open Questions

- [NEEDS CLARIFICATION: What resident memory, CPU, and concurrent process count does a
  real multi-tenant host see per plugin, and does that miss spec 029's 100-instance
  target?]
- [NEEDS CLARIFICATION: What is measured cold-start latency versus the 5-second target,
  and how often does the idle timeout already avoid a start?]
- [NEEDS CLARIFICATION: Has any host hit its per-node plugin cap (FR-026)? This repo
  cannot answer; the orchestrator is not here.]
- [NEEDS CLARIFICATION: Is #195 meant to cover how plugins learn caller identity, leaving
  #220 only for raw cloud secrets in a shared process, or are the two issues the same
  unfinished idea?]
- [NEEDS CLARIFICATION: Which host, if any, still plans a shared plugin pool? Spec 029
  names Pulumi Insights as an example, not as a committed consumer of #220.]
- [NEEDS CLARIFICATION: The developer guide says avoid secrets in environment variables,
  while FR-021 requires them. Which rule do plugin authors follow today?]

## Sources

- <https://github.com/rshade/finfocus-spec/issues/220> (host: github.com, policy: allowlisted)
- <https://github.com/rshade/finfocus-spec/issues/189> (host: github.com, policy: allowlisted)
- <https://github.com/rshade/finfocus-spec/issues/195> (host: github.com, policy: allowlisted)
- `ROADMAP.md` (repository file)
- `specs/029-grpc-web-support/spec.md` (repository file)
- `specs/029-grpc-web-support/research.md` (repository file)
- `specs/029-grpc-web-support/tasks.md` (repository file)
- `.specify/memory/constitution.md` (repository file)
- `sdk/go/pluginsdk/env.go`, `logging.go`, `client.go`, `sdk.go`, `options.go`
  (repository files)
- `proto/finfocus/v1/enums.proto`, `costsource.proto`, `usage.proto` (repository files)
- `PLUGIN_DEVELOPER_GUIDE.md`, `docs/PLUGIN_STARTUP_PROTOCOL.md` (repository files)
