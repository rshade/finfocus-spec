# Concept: Whether a shared plugin process is worth a second credential model

- **Slug**: per-request-credentials
- **Created**: 2026-09-27
- **Recommended option**: none

## Options

### Option A — Keep one process per organization

- **Sketch**: Hosts keep launching a plugin process per tenant and putting that tenant's
  cloud credentials in the process environment. Rare tenants pay a start when their
  process is gone, and idle processes exit. This repository adds no new credential path.
  Callers and plugin authors see no change.
- **Appetite**: small
- **Trade-offs**: Wins the isolation and ambient cloud-SDK behavior spec 029 already
  chose. Sacrifices nothing that has been measured. The risk is that a host is already
  over the 100-instance or 5-second targets and this option does not look. Unknown:
  actual process size and start time.
- **Rabbit holes**: Rewriting spec 029's orchestrator rules, or building a process
  manager inside this spec repo "so the default is real." The launcher is a host
  concern. Document-only churn in the SDK would not change tenant density.

### Option B — Measure on a host, then reopen the question

- **Sketch**: Before any protocol work, the host that runs many tenants records
  per-process memory and CPU, how many processes stay up, cold-start time against the
  5-second target, and whether the idle timeout and instance cap already cope. This
  repository does not gain an API. The question stays a discovery item until those
  numbers miss the targets spec 029 wrote down.
- **Appetite**: small for the measurement, and uncertain because the work sits in a host
  this repo does not contain. Appetite inside finfocus-spec is effectively none.
- **Trade-offs**: Wins a real baseline and avoids a large secret-handling design. Does
  not itself reduce memory or cold start. If no host will measure, the question stays
  open indefinitely. Risk: someone treats a benchmark harness in this repo as a
  substitute for production shape.
- **Rabbit holes**: Inventing an in-repo orchestrator or load generator so the spec
  project can "prove" process cost. A lab number still would not be the host's number.
  Expanding the measurement into a credentials design before the numbers exist.

### Option C — Opt-in shared process for plugins that accept per-call credentials

- **Sketch**: The default stays one process per organization. A plugin that opts in could
  be pooled, and the host would attach that tenant's cloud credentials to each call so
  infrequent tenants do not each need a process. Plugins that do not opt in stay on
  option A. The host, not this SDK alone, would decide which pool receives the call.
- **Appetite**: large. The issue is labeled `effort/large`. A multi-month budget is a
  fair ceiling, not an estimate. Uncertainty is high: the router is outside this repo,
  and cloud SDKs expect ambient credentials.
- **Trade-offs**: Wins fewer processes and no per-tenant cold start, if a host uses the
  pool and if plugins actually opt in. Sacrifices process isolation, simple plugin code,
  and the ambient credential chain spec 029 required. Named risks, unmitigated here:
  a larger blast radius if the shared process is compromised, caches that mix tenants,
  and every participating author handling secrets. Overlaps the open identity-metadata
  question in issue #195.
- **Rabbit holes**: A concrete on-the-wire credential layout; secret lifetime and
  redaction; cache keys; how gRPC and Connect both carry the material; credential
  rotation without a restart; a capability flag that hosts must honor; doing the pool
  router inside this repo despite constitution principle IV. Any of those blows a
  discovery item into a security project. This concept does not pick a layout.

## Recommendation

Do not proceed. No option should be specified now.

Option A is the standing design, not a new bet. Option B is the only step that matches
the issue's own gate, and it is a host measurement, not a finfocus-spec change. Option C
misses the goals: isolation must hold until measurements show the current model fails,
and there is no measured miss. Specifying C would spend a large appetite on an unsourced
memory claim and on a router this repository does not own.

Success for this assessment is a recorded "not yet," not a credential feature. Reopen
only if a host shows it misses spec 029's instance or latency targets and idle reaping
does not fix that.

## Out of Scope (for the recommended option)

- Any credential channel, client option, plugin helper, or capability flag.
- Replacing per-organization processes as the default.
- Orchestrator lifecycle, pool routing, and instance caps (host-owned under spec 029).
- Cost math, pricing, and backend state.
- Settling issue #195's authorization-middleware research.
- Closing, relabeling, or commenting on GitHub issues from this assessment.

## Assumptions to Validate

- Spec 029's per-organization process, environment credentials, idle timeout, and
  instance cap are still the host's intended model, not a draft that was abandoned when
  the spec header was left as `Draft`.
- A host exists that can measure process size and cold start. If none will, option B
  never produces a number and option C stays unjustified.
- Issue #195 stays a separate identity question. If it is meant to be the only metadata
  story for caller context, a later shared-process design has to start there instead of
  from raw cloud secrets.
- Plugin authors still rely on ambient cloud SDK credentials. If they already take
  explicit secrets per call, the compatibility objection is weaker than spec 029 claims.
- The ~50-100MB figure is not evidence. Any later recommendation that builds option C
  has to replace it with a host measurement.
