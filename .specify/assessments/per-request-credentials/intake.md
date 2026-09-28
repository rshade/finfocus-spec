# Idea Intake: Per-Request Credential Passing

- **Slug**: per-request-credentials
- **Created**: 2026-09-27
- **Source**: <https://github.com/rshade/finfocus-spec/issues/220> (host: github.com, policy: allowlisted)
- **Type**: exploration

## Idea (as captured)

GitHub issue #220, opened 2025-12-30 by Richard Shade (`rshade`), still open, no comments.
Labels: `enhancement`, `roadmap/future`, `effort/large`. Title: "discovery: Per-Request Credential
Passing for Multi-Tenant Optimization". Fetched with `gh issue view` (no query or fragment). Sanitized
URL: <https://github.com/rshade/finfocus-spec/issues/220>.

Quoted body (wording preserved; no secrets were present):

> ## Rationale
>
> Enable optional per-request credential passing to allow a single plugin instance to serve
> multiple tenants. This is an optimization over the default per-tenant plugin instance model
> that could reduce resource usage in scenarios where:
>
> - Many tenants have infrequent requests
> - Memory/CPU resources are constrained
> - Cold start latency is problematic
>
> ## Context
>
> The primary multi-tenant model (spec 029-grpc-web-support) uses **per-org plugin instances**
> where each tenant gets a dedicated plugin process with credentials passed via environment
> variables. This provides strong isolation but has resource overhead.
>
> Per-request credentials would allow a shared plugin pool where credentials are passed in
> request metadata/headers, enabling one plugin to serve multiple tenants.
>
> ## Tradeoffs Analysis
>
> ### Per-Request Credentials (This Issue)
>
> **Pros:**
>
> - Lower resource usage (fewer processes)
> - No cold start latency for infrequent tenants
> - Simpler deployment (one plugin per type)
>
> **Cons:**
>
> - Plugin code must handle credentials explicitly (developer burden)
> - Cloud SDKs expect ambient credentials (compatibility issues)
> - Larger blast radius if plugin is compromised
> - Cache isolation complexity
> - Every plugin author must implement securely
>
> ### Per-Org Instances (Current Default)
>
> **Pros:**
>
> - Process isolation (security)
> - Cloud SDK compatible (ambient credentials)
> - Simple plugin code
> - Clear compliance boundaries
>
> **Cons:**
>
> - Resource overhead (~50-100MB per process)
> - Cold start latency for new tenants
>
> ## Proposed Approach
>
> If implemented, this would be an **opt-in alternative** to the default per-org model:
>
> 1. SDK provides `WithCredentials(creds)` option on client requests
> 2. Plugin SDK provides `ExtractCredentials(ctx)` helper
> 3. Plugins opt-in to credential handling via interface implementation
> 4. Orchestrator routes to shared pool when plugin supports per-request credentials
>
> ## Decision Required
>
> Before implementing, we should validate whether per-request credentials are actually needed:
>
> - Monitor resource usage with per-org instances in production
> - Measure cold start latency impact
> - Evaluate if the complexity is justified
>
> ## Boundary Check
>
> Infrastructure enhancement for transport/DX; no backend state or math.
>
> ## Related
>
> - Spec: 029-grpc-web-support (defines per-org as default)
> - Issue: #189 (parent gRPC-Web support issue)

The issue states it is not ready to implement until resource use and cold-start latency are measured.
The proposed shape is recorded only as captured text, not as a chosen design.

## Restated

The idea is an optional way for one plugin process to serve more than one tenant by accepting
credentials on each request, instead of the stated default of one process per organization with
credentials in the environment. The ticket asks for a decision on whether that option is needed
before any implementation.

## Origin & Context

- **Raised by**: Richard Shade (`rshade`), as GitHub issue #220 on 2025-12-30. Last updated
  2026-02-28. No assignees, milestone, or comments.
- **Trigger**: Stated resource overhead and cold-start cost of a per-organization plugin process.
  The body points at spec `029-grpc-web-support` as the source of the per-org default and at issue
  #189 as the parent gRPC-Web issue. Whether those still say that is not settled in this note.
- **Related issue observed at intake**: #189 ("discovery: gRPC-Web support for Browser Diagnostics")
  is closed (closed 2025-12-30). Its body is about Connect/gRPC-Web transport for browser clients,
  not per-request tenant credentials. Sanitized URL:
  <https://github.com/rshade/finfocus-spec/issues/189> (host: github.com, policy: allowlisted).

## First-Glance Unknowns

- [NEEDS CLARIFICATION: Whether production per-org process memory and cold-start latency have been
  measured, and what numbers would justify a second credential model.]
- [NEEDS CLARIFICATION: Whether spec 029-grpc-web-support still defines per-org plugin processes and
  environment-variable credentials as the default, or whether that claim has drifted.]
- [NEEDS CLARIFICATION: Where the orchestrator that would route tenants to a shared plugin pool
  lives, and whether it is in this repository.]
- [NEEDS CLARIFICATION: Which current plugin and host credential paths exist (environment, metadata,
  request fields), and which of the issue's cited SDK names still match the tree.]
- [NEEDS CLARIFICATION: Who must keep using ambient cloud-SDK credentials, and whether an opt-in
  interface is enough for those plugins.]
- [NEEDS CLARIFICATION: What isolation, cache, and compliance properties the per-org process is
  required to preserve if a shared process is ever allowed.]
- [NEEDS CLARIFICATION: Whether closing #189 changed the multi-tenant hosting model this idea
  depends on.]
