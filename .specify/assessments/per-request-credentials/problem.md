# Problem Definition: Unmeasured cost of one plugin process per tenant

- **Slug**: per-request-credentials
- **Created**: 2026-09-27
- **Inputs used**: intake.md | research.md

## Problem Statement

A multi-tenant host that follows spec 029 keeps a separate plugin process for each
organization, with that organization's cloud credentials only in that process. That
isolation has a resource cost and a startup delay for a tenant whose process is not
already running. Nobody has published, in this repository or on issue #220, whether that
cost is large enough to matter. Until a host measures it, it is not known whether the
current model is hurting anyone.

## Affected Users & Stakeholders

- **Users**: Operators of a multi-tenant host (spec 029's example is Pulumi Insights) —
  they would feel memory pressure, instance-cap pressure, or cold-start delay if the
  per-organization process model is too heavy. Whether any operator feels that today is
  [NEEDS CLARIFICATION: no host telemetry is in this repo or on issue #220].
- **Users**: People waiting on a cost call after a tenant's plugin was idle — they would
  feel launch delay. Spec 029's target is under 5 seconds. The actual wait is
  [NEEDS CLARIFICATION: not measured here].
- **Stakeholders**: Richard Shade (`rshade`), issue author and roadmap owner — decides
  whether this discovery item stays, is measured, or is dropped. No other commenter has
  weighed in.
- **Stakeholders**: Plugin authors — not blocked by today's ambient-credential model.
  They become affected only if a shared process later requires each plugin to handle
  secrets explicitly. That burden is a cost of a future change, not a pain felt now.
- **Stakeholders**: Maintainers of this spec repository — own the protocol and SDK.
  Constitution principle IV and spec 029 put the process launcher outside this repo, so
  they cannot remove the overhead by themselves.

## Goals

- Learn, from a host that actually runs many tenants, whether per-process memory, CPU,
  instance count, and cold start miss the limits spec 029 already set.
- Keep organization-to-organization credential isolation at least as strong as separate
  processes until those measurements show the current model fails.
- Spend no large protocol change on an optimization whose benefit is still an unsourced
  claim.

## Non-Goals

- Changing cost math, pricing, or any backend store. Issue #220's boundary excludes that,
  and constitution principle III already does.
- Moving the multi-tenant orchestrator into this repository. Spec 029 assigns it to the
  host.
- Treating per-organization processes as a mistake before measurements exist. Spec 029
  made that model the default on purpose.
- Deciding issue #195 (authorization middleware). It is a related open question, not this
  problem.
- Writing a credential channel, metadata layout, or plugin interface. Those are solution
  sketches, and they are out of this problem statement.

## Success Metrics

- Resident memory and CPU per plugin process, and the number of concurrent processes, on
  a real multi-tenant host (baseline: unknown; issue #220's ~50-100MB per process is
  unsourced and was not found elsewhere in the repo).
- Cold-start latency for a tenant with no running plugin (baseline: unknown; spec 029
  SC-014 requires under 5 seconds and is a target, not an observation).
- How often the idle timeout (spec 029 default 5 minutes) already exits processes for
  infrequent tenants, and whether the per-node instance cap (FR-026) is ever hit
  (baseline: unknown; this repo does not run the orchestrator).
- Qualitative, and secondary: plugin authors can keep using ambient cloud credentials
  without a new secret-handling contract (baseline: that is what spec 029 FR-016 and
  FR-021 require today).

The problem is worth a protocol change only if the first three signals miss those
existing targets. Meeting the targets means the current model is the success case.

## Cost of Inaction

If nothing else is built, hosts that follow spec 029 keep one process per active
organization, start it on first use, and stop it after the idle timeout. Isolation stays
the P1 property spec 029 wrote down. This repository changes nothing, which matches the
fact that the launcher does not live here.

The unpaid cost is whatever extra memory and startup delay that model has. That cost is
not quantified. There is no incident, comment, or benchmark showing that the 100-instance
or 5-second targets are already missed. Waiting avoids a large, security-sensitive
protocol change. It also means that if a host is quietly over those targets, this repo
will not notice until someone measures it.

## Open Questions

- [NEEDS CLARIFICATION: What memory, CPU, and process counts does a real host see, and do
  they miss spec 029's 100-instance target?]
- [NEEDS CLARIFICATION: What is measured cold-start latency versus the 5-second target,
  and how often does idle timeout avoid a start?]
- [NEEDS CLARIFICATION: Has any host hit its per-node plugin instance cap?]
- [NEEDS CLARIFICATION: Which host, if any, still wants a shared plugin pool? Pulumi
  Insights is an example in spec 029, not a committed consumer of issue #220.]
- [NEEDS CLARIFICATION: Should issue #195's identity-metadata work be settled before any
  shared-process credential question, or are they independent?]
- [NEEDS CLARIFICATION: Do plugin authors follow FR-021 (credentials in the environment)
  or the startup guide's advice to avoid secrets in environment variables?]
