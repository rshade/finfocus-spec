# Strategic Roadmap: finfocus-spec

## Vision

To provide the definitive, high-performance gRPC specification and Go SDK for cloud cost observability,
centered around the FinOps Foundation's FOCUS standard.

---

## Immediate Focus

*No active items. Next candidate: [#564](https://github.com/rshade/finfocus-spec/issues/564).*

---

## Near-Term Vision

### Post-0.7.0 Review Follow-ups

- [ ] **Pin Release and Publish Actions to SHAs**
  ([#564](https://github.com/rshade/finfocus-spec/issues/564)) [S] -
  Actions holding a PAT or package-write token use mutable tags.

---

## Future Vision (Long-Term)

### Active Research

- [ ] **Authorization Middleware (OIDC/IAM)**
  ([#195](https://github.com/rshade/finfocus-spec/issues/195)) (spike, timebox 1d) -
  Standardizing how plugins receive and validate identity without violating "Stateless" boundaries.

### Completed Research

- [x] **Standardized Cost Allocation Lineage Metadata**
  ([#191](https://github.com/rshade/finfocus-spec/issues/191)) -
  Added `LineageNode` chain to `ActualCostResult` and `ResourceDescriptor` with
  pass-through semantics, plus a `LineageBuilder` SDK helper. Sep 2026.
- [x] **Streaming Actual Cost (Streaming RPCs)**
  ([#197](https://github.com/rshade/finfocus-spec/issues/197)) -
  Superseded by pagination approach ([#353](https://github.com/rshade/finfocus-spec/issues/353)). Jan 2026.
- [x] **Standardized Recommendation Reasoning Metadata**
  ([#192](https://github.com/rshade/finfocus-spec/issues/192)) - Closed Jan 2026.
- [x] **Cross-Language SDKs (Python/TS)**
  ([#196](https://github.com/rshade/finfocus-spec/issues/196)) -
  TypeScript SDK delivered; Python SDK remains future work.
- [x] **Standardized Recommendation Reasoning** ([#188](https://github.com/rshade/finfocus-spec/issues/188)) -
  Closed Jan 2026.
- [x] **Validation Bypass Protocol** ([#216](https://github.com/rshade/finfocus-spec/issues/216)) -
  Added `BypassReason` and `OverrideMetadata` to `ValidationResult` for governance auditing.
- [x] **Currency Library Evaluation**
  ([#358](https://github.com/rshade/finfocus-spec/issues/358)) -
  Evaluated external currency libraries (x/text/currency, bojanz/currency). Feb 2026.
- [x] **Batch RPC for Multi-Resource Queries**
  ([#221](https://github.com/rshade/finfocus-spec/issues/221)) -
  Implemented dedicated BatchCost RPC for multi-resource cost queries. Feb 2026.

### Proposed for Discussion (Discovery)

- [ ] **Signed Page Tokens for v2.0**
  ([#370](https://github.com/rshade/finfocus-spec/issues/370)) (spike, timebox 2h) -
  Research HMAC-signed tokens to prevent token manipulation in pagination.

---

## Completed Milestones

### Q4 2026

- [x] #611 `registry`: Manifest schema and SDK writer accept valid manifests. Closed 2026-10-03. [M]
- [x] #609 `docs`: Document how Pulumi inputs reach plugin tags. Closed 2026-10-03. [S]
- [x] #612 `proto`: Nine missing FOCUS 1.3 ServiceCategory values (not planned). Closed 2026-10-03. [S]
- [x] #574 `proto`: Scorer sessions let duplicate groups span batches. Closed 2026-10-02. [L]
- [x] #580 `proto`: ScorerInfo request id and model lists for multi-call scorers. Closed 2026-10-02. [S]
- [x] #579 `proto`: AllocateRequest period and selector with echoed window. Closed 2026-10-02. [M]
- [x] #578 `proto`: AllocationRow method and resource provenance fields. Closed 2026-10-02. [M]
- [x] #576 `proto`: Scorer omitted_fields names host-cleared paths. Closed 2026-10-02. [S]
- [x] #573 `pluginsdk`: Advertise scorer batch limit and signals in plugin info. Closed 2026-10-02. [M]
- [x] #589 `proto`: RegionPrice list for per-region retail prices. Closed 2026-10-02. [M]
- [x] #588 `proto`: PriceOption list for alternative retail prices. Closed 2026-10-02. [M]
- [x] #590 `proto`: Add billing_account_id to GetActualCostRequest. Closed 2026-10-02. [M]
- [x] #575 `docs`: Per-request credentials for GetStats, Allocate, and scoring. Closed 2026-10-02. [S]
- [x] #581 `docs`: Score cache validity and versioning guidance. Closed 2026-10-02. [S]
- [x] #577 `docs`: Clarify identifier_mode scope in the scoring contract. Closed 2026-10-02. [S]
- [x] #582 `deps`: Require grpc v1.83.2 for GO-2026-6443. Closed 2026-10-02. [S]
- [x] #565 `ts-sdk`: REST gateway hides upstream errors and stops draining bodies. Closed 2026-10-02. [M]
- [x] #561 `testing`: Share bufconn harness and duplicate-key scan. Closed 2026-10-02. [M]
- [x] #566 `docs`: Fail-closed ExtractCredentials and credential header filtering. Closed 2026-10-02. [S]
- [x] #562 `pluginsdk`: isJSONObject rejects invalid applicability JSON. Closed 2026-10-02. [S]
- [x] #563 `pluginsdk`: stampValidationTrace no longer mutates handler errors. Closed 2026-10-02. [S]

### Q3 2026

- [x] #584 `testing`: Deterministic ScoreRecommendations response validation. Closed 2026-09-30. [S]
- [x] #556 `proto`: RecommendationScorerService.ScoreRecommendations for scorer plugins. Closed 2026-09-29. [L]
- [x] #540 `proto`: Support FOCUS 1.4 billing, invoice, and commitment datasets. Closed 2026-09-29. [L]
- [x] #543 `proto`: FOCUS 1.4 Billing Period and Invoice Detail datasets. Closed 2026-09-29. [L]
- [x] #542 `proto`: FOCUS 1.4 Contract Commitment columns. Closed 2026-09-29. [M]
- [x] #544 `proto`: Decide delivery path for supplemental datasets. Closed 2026-09-28. [S]
- [x] #541 `pluginsdk`: FOCUS 1.4 Cost and Usage columns. Closed 2026-09-28. [S]
- [x] #433 `proto`: Add cost_breakdown map to GetProjectedCostResponse. Closed 2026-09-28. [M]
- [x] #220 `pluginsdk`: Opt-in per-request credential passing. Closed 2026-09-28. [L]
- [x] #193 `pluginsdk`: Log host trace id on validation failures. Closed 2026-09-28. [L]
- [x] #190 `pluginsdk`: Multi-currency segregation pattern. Closed 2026-09-28. [L]
- [x] #514 `ts-sdk`: Remove ignoreDeprecations before TypeScript 7. Closed 2026-09-28. [S]
- [x] #513 `ts-sdk`: Make middleware and framework-plugins packages compile. Closed 2026-09-28. [M]
- [x] #506 `proto`: AllocatorService.Allocate for cost allocation plugins. Closed 2026-09-28. [M]
- [x] #403 `pluginsdk`: Resolve concurrency TODO in BatchCost worker-pool test. Closed 2026-09-28. [M]
- [x] #402 `pluginsdk`: Benchmark descriptorClone overhead in batch results. Closed 2026-09-28. [M]
- [x] #504 `pluginsdk`: Backfill inferred capabilities for PluginInfoProvider. Closed 2026-09-26. [S]
- [x] #507 `pluginsdk`: Fix Supports always failing with default registry. Closed 2026-09-26. [S]
- [x] #505 `proto`: UsageSourceService.GetStats for workload usage plugins. Closed 2026-09-25. [M]
- [x] #494 `pluginsdk`: Adopt ax-go v0.6.0 CLI in Serve for plugin binaries. Closed 2026-09-08. [L]

### Q2 2026

- [x] #434 `proto`: Add expires_at to EstimateCostResponse. Closed 2026-04-03. [M]

### Q1 2026

- [x] #204 Extract ResourceDescriptor test helper
- [x] #210 Integrate ValidationError with validation

#### Protocol & Modeling

- [x] #381 Metadata map for confidence signals
- [x] #363 Enforce token opacity in proto comments
- [x] #365 Warn on TotalCount change mid-iteration
- [x] #377 Plugin contract for GetProjectedCost
- [x] #353 Pagination for GetActualCost RPC
- [x] #241, #250 Forecasting primitives
- [x] #199 FOCUS 1.3 migration
- [x] #315 Cost anomaly detection support
- [x] #314 Prediction interval fields
- [x] #217 Pricing tier intelligence
- [x] #218 Usage profiles
- [x] #380 Caching hint (expires_at)
- [x] #221 Batch cost RPC
- [x] #407 BatchCost pagination continuation docs

#### SDK & Tooling

- [x] #344 StreamResult.ValidateOutputJSON helper
- [x] #396 Document DryRunHandler.HandleDryRun breaking change
- [x] #397 Remove proto.Clone on BatchCostRequest
- [x] #398 Bound goroutines in batchCostFallback
- [x] #401 ResourceDescriptor field validation limits
- [x] #400 Improve ValidateBatchCostRequest return type
- [x] #399 Deduplicate grpcCodeToInt32
- [x] #394 Avoid full proto.Clone in BatchCost normalization
- [x] #395 PluginInfoProvider and BatchCostHandler test coverage
- [x] #404 Replace magic constant defaultInternalErrCode
- [x] #405 Extract helper from MockPlugin.BatchCost
- [x] #406 Improve ResourceError.code proto comment
- [x] #242 Plugin capability discovery
- [x] #248 Plugin capability dry run mode
- [x] #201 Contextual FinOps validation
- [x] #213 Advanced SDK patterns
- [x] #243 SDK documentation overhaul
- [x] #225 DismissRecommendation consistency
- [x] #113, #142 Standardized benchmark suite
- [x] #187 JSON-LD / Schema.org serialization
- [x] #257, #227, #234, #230, #232, #203, #246, #224, #212, #205 v0.4.14 SDK polish release
- [x] #244 GetPluginInfo performance test
- [x] #245 User-friendly error messages
- [x] #293 TypeScript SDK
- [x] #304 TypeScript SDK Connect-ES v2 migration
- [x] #311 TypeScript SDK publishing infrastructure
- [x] #313 TypeScript SDK publishing enhancements
- [x] #287 Plugin capability enum
- [x] #297, #296 Capability discovery robustness
- [x] #294, #295, #299, #300, #301 Capability discovery bug fixes
- [x] #283 Legacy environment variable compatibility
- [x] #282 Migration documentation
- [x] #272 PulumiCost to FinFocus rename
- [x] #284 Log file security warning
- [x] #298 Unclear exhaustive nolint directive
- [x] #336 Float validation harmonization
- [x] #334 TypeScript SDK documentation
- [x] #340 Bypass metadata security
- [x] #346, #347, #348 Documentation drift audit
- [x] #364, #367, #368, #369 Pagination hardening
- [x] #366 Iterator concurrency safety documentation
- [x] #371 Expand pagination SDK documentation
- [x] #345 CapabilitiesToLegacyMetadataWithWarnings default
- [x] #350 Upgrade golangci-lint

### Pre-2026

#### Protocol & Modeling

- [x] #176 GreenOps standardization
- [x] #173, #171, #166 Recommendation enhancements
- [x] #100, #99 FOCUS 1.2 integration
- [x] #125, #123, #90 RPC expansion: EstimateCost, GetBudgets, GetRecommendations
- [x] #63, #33 Zero-allocation enums

#### SDK & Tooling

- [x] #109 Plugin conformance suite
- [x] #151, #148, #145, #139 SDK foundation
- [x] #181, #143, #126 Orchestration support
- [x] #189, #223 Multi-protocol support (gRPC-Web/Connect)

---

## Boundary Safeguards (The "Hard No's")

- **No Orchestration Logic**: Multi-plugin aggregation and "Muxing" logic will stay in `finfocus-core`.
- **No Persistence**: The SDK will not provide built-in database support; it remains stateless.
- **No Native Math Engines**: The SDK will not perform financial amortization;
  it will only define the fields to store the results of such calculations.
