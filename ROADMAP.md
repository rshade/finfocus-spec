# Strategic Roadmap: finfocus-spec

## Vision

To provide the definitive, high-performance gRPC specification and Go SDK for cloud cost observability,
centered around the FinOps Foundation's FOCUS standard.

---

## Immediate Focus

### Stability & Maintenance

- [ ] **Dependency Management** ([#13](https://github.com/rshade/finfocus-spec/issues/13)) -
  Automated tracking and updating of core proto and SDK dependencies.

---

## Near-Term Vision

### Post-0.7.0 Review Follow-ups

- [ ] **isJSONObject Accepts Invalid JSON**
  ([#562](https://github.com/rshade/finfocus-spec/issues/562)) [S] -
  Contract commitment applicability check is a bracket scanner, not a JSON parser.
- [ ] **stampValidationTrace Mutates Returned Error**
  ([#563](https://github.com/rshade/finfocus-spec/issues/563)) [S] -
  Writes a trace id into a `ValidationError` the handler owns; races on shared error values.
- [ ] **REST Gateway Error Text and Body Drain**
  ([#565](https://github.com/rshade/finfocus-spec/issues/565)) [M] -
  TypeScript gateway returns upstream error text and keeps draining oversized request bodies.
- [ ] **Pin Release and Publish Actions to SHAs**
  ([#564](https://github.com/rshade/finfocus-spec/issues/564)) [S] -
  Actions holding a PAT or package-write token use mutable tags.
- [ ] **Credential Handling Docs**
  ([#566](https://github.com/rshade/finfocus-spec/issues/566)) [S] -
  Document fail-closed `ExtractCredentials` and filtering credential headers in plugin interceptors.
- [ ] **Deduplicate Supplemental Test Harnesses**
  ([#561](https://github.com/rshade/finfocus-spec/issues/561)) [M] -
  Merge copy-pasted bufconn harnesses and duplicate-key checks in `sdk/go/testing`.

---

## Future Vision (Long-Term)

### Active Research

- [ ] **Authorization Middleware (OIDC/IAM)**
  ([#195](https://github.com/rshade/finfocus-spec/issues/195)) [L] -
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
  ([#370](https://github.com/rshade/finfocus-spec/issues/370)) [L] -
  Research HMAC-signed tokens to prevent token manipulation in pagination.

---

## Completed Milestones

### Q3 2026

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

- [x] #204 `pluginsdk`: Extract ResourceDescriptor test helper. Closed 2026-03-13. [S]
- [x] #210 `pluginsdk`: Integrate ValidationError with validation. Closed 2026-03-11. [M]

#### Protocol & Modeling

- [x] **Metadata Map for Confidence Signals**
  ([#381](https://github.com/rshade/finfocus-spec/issues/381)) -
  Added metadata map to GetProjectedCostResponse for property confidence signals. Mar 2026.
- [x] **Enforce Token Opacity in Proto Comments**
  ([#363](https://github.com/rshade/finfocus-spec/issues/363)) -
  Added explicit warning that `page_token` is opaque; clients MUST NOT parse or construct tokens. Mar 2026.
- [x] **Warn on TotalCount Change Mid-Iteration**
  ([#365](https://github.com/rshade/finfocus-spec/issues/365)) -
  Log zerolog warning when `total_count` changes between pages during iteration. Mar 2026.
- [x] **Plugin Contract for GetProjectedCost**
  ([#377](https://github.com/rshade/finfocus-spec/issues/377)) -
  Documented plugin contract for sparse/old-state property handling in projected cost responses. Mar 2026.
- [x] **Pagination for GetActualCost RPC**
  ([#353](https://github.com/rshade/finfocus-spec/issues/353)) -
  Added `page_size`/`page_token` pagination to enable 10,000+ record retrieval without memory exhaustion.
- [x] **Forecasting Primitives**
  ([#241](https://github.com/rshade/finfocus-spec/issues/241),
  [#250](https://github.com/rshade/finfocus-spec/issues/250)) -
  Added `GrowthType` (Linear, Exponential) and `GrowthRate` to `CostResult` for cost projections.
- [x] **FOCUS 1.3 Migration** ([#199](https://github.com/rshade/finfocus-spec/issues/199)) -
  Audit new columns and entities in FOCUS 1.3 and update builder APIs.
- [x] **Cost Anomaly Detection Support**
  ([#315](https://github.com/rshade/finfocus-spec/issues/315)) -
  Added ANOMALY category and INVESTIGATE action for cost anomaly detection in recommendations.
- [x] **Prediction Interval Fields**
  ([#314](https://github.com/rshade/finfocus-spec/issues/314)) -
  Added confidence intervals to GetProjectedCostResponse for uncertainty quantification.
- [x] **Pricing Tier Intelligence** ([#217](https://github.com/rshade/finfocus-spec/issues/217)) -
  Added `PricingTier` enum and `SpotRisk` enum for interruption probability.
- [x] **Usage Profiles** ([#218](https://github.com/rshade/finfocus-spec/issues/218)) -
  Added `UsageProfile` context for environment-aware recommendations.
- [x] **Caching Hint (expires_at)**
  ([#380](https://github.com/rshade/finfocus-spec/issues/380)) -
  Added `expires_at` caching hint to cost results for cache freshness signaling.
- [x] **Batch Cost RPC**
  ([#221](https://github.com/rshade/finfocus-spec/issues/221)) -
  Dedicated BatchCost RPC for efficient multi-resource cost queries.
- [x] **BatchCost Pagination Continuation Docs**
  ([#407](https://github.com/rshade/finfocus-spec/issues/407)) -
  Documented pagination continuation workflow for BatchCost results.

#### SDK & Tooling

- [x] **Add StreamResult.ValidateOutputJSON() Helper**
  ([#344](https://github.com/rshade/finfocus-spec/issues/344)) -
  Helper method for JSON-LD output validation. Mar 2026.
- [x] **Document DryRunHandler.HandleDryRun Breaking Change**
  ([#396](https://github.com/rshade/finfocus-spec/issues/396)) -
  Documented DryRunHandler.HandleDryRun interface change for plugin authors. Mar 2026.
- [x] **Remove Unnecessary proto.Clone on BatchCostRequest**
  ([#397](https://github.com/rshade/finfocus-spec/issues/397)) -
  Removed unnecessary proto.Clone in Server.BatchCost for performance. Mar 2026.
- [x] **Fix Unbounded Goroutine Creation in batchCostFallback**
  ([#398](https://github.com/rshade/finfocus-spec/issues/398)) -
  Added goroutine pool bounds to prevent unbounded concurrency. Mar 2026.
- [x] **Add ResourceDescriptor Field Validation Limits**
  ([#401](https://github.com/rshade/finfocus-spec/issues/401)) -
  Added field validation limits to batch request validation. Mar 2026.
- [x] **Improve ValidateBatchCostRequest Return Type**
  ([#400](https://github.com/rshade/finfocus-spec/issues/400)) -
  Improved return type to avoid (nil, nil) sentinel pattern. Mar 2026.
- [x] **Deduplicate grpcCodeToInt32**
  ([#399](https://github.com/rshade/finfocus-spec/issues/399)) -
  Eliminated during Batch RPC hardening; function no longer exists in either package. Mar 2026.
- [x] **Avoid Full proto.Clone in BatchCost Normalization**
  ([#394](https://github.com/rshade/finfocus-spec/issues/394)) -
  Avoided full proto.Clone in BatchCost request normalization for performance. Mar 2026.
- [x] **Add PluginInfoProvider + BatchCostHandler Test Coverage**
  ([#395](https://github.com/rshade/finfocus-spec/issues/395)) -
  Added test coverage for PluginInfoProvider + BatchCostHandler combination. Mar 2026.
- [x] **Replace Magic Constant defaultInternalErrCode**
  ([#404](https://github.com/rshade/finfocus-spec/issues/404)) -
  Replaced magic constant with `codes.Internal` from gRPC. Mar 2026.
- [x] **Extract Helper from MockPlugin.BatchCost**
  ([#405](https://github.com/rshade/finfocus-spec/issues/405)) -
  Extracted per-resource helper to reduce cognitive complexity. Mar 2026.
- [x] **Improve ResourceError.code Proto Comment**
  ([#406](https://github.com/rshade/finfocus-spec/issues/406)) -
  Added full gRPC status code reference to proto comment. Mar 2026.
- [x] **Plugin Capability Discovery** ([#242](https://github.com/rshade/finfocus-spec/issues/242)) -
  Implemented `GetPluginInfo` RPC for spec version compatibility and capability advertisement.
- [x] **Plugin Capability Dry Run Mode** ([#248](https://github.com/rshade/finfocus-spec/issues/248)) -
  Implemented `DryRun` for plugin field mapping discovery.
- [x] **Contextual FinOps Validation** ([#201](https://github.com/rshade/finfocus-spec/issues/201)) -
  Extended `pluginsdk/validation` to include contextual checks.
- [x] **Advanced SDK Patterns** ([#213](https://github.com/rshade/finfocus-spec/issues/213)) -
  Implementation examples for complex tiered pricing and multi-provider mapping.
- [x] **SDK Documentation Overhaul** ([#243](https://github.com/rshade/finfocus-spec/issues/243)) -
  Comprehensive Godoc, thread safety, rate limiting, and performance documentation.
- [x] **DismissRecommendation Consistency** ([#225](https://github.com/rshade/finfocus-spec/issues/225)) -
  Made DismissRecommendation follow the same pattern as other RPC methods.
- [x] **Standardized Benchmark Suite**
  ([#113](https://github.com/rshade/finfocus-spec/pull/113),
  [#142](https://github.com/rshade/finfocus-spec/pull/142)) -
  Formalized "Time to First Byte" and memory allocation benchmarks for plugins.
- [x] **JSON-LD / Schema.org Serialization** ([#187](https://github.com/rshade/finfocus-spec/issues/187),
  [#252](https://github.com/rshade/finfocus-spec/pull/252)) -
  Added `jsonld` package for FOCUS cost data serialization with Schema.org compatibility.
- [x] **v0.4.14 SDK Polish Release** ([#257](https://github.com/rshade/finfocus-spec/issues/257)) -
  Significant improvements to SDK developer experience, testing, and Connect protocol support.
  Includes:
  - **Connect Protocol**: Test coverage ([#227](https://github.com/rshade/finfocus-spec/issues/227)),
    CORS validation ([#234](https://github.com/rshade/finfocus-spec/issues/234)).
  - **Developer Experience**: Custom HealthChecker ([#230](https://github.com/rshade/finfocus-spec/issues/230)),
    Context helpers ([#232](https://github.com/rshade/finfocus-spec/issues/232)), ARN validation ([#203](https://github.com/rshade/finfocus-spec/issues/203)),
    Migration guide ([#246](https://github.com/rshade/finfocus-spec/issues/246)).
  - **Quality**: CI Benchmarks ([#224](https://github.com/rshade/finfocus-spec/issues/224)),
    Extreme value tests ([#212](https://github.com/rshade/finfocus-spec/issues/212)), Fuzzing ([#205](https://github.com/rshade/finfocus-spec/issues/205)).
- [x] **GetPluginInfo Performance Test** ([#244](https://github.com/rshade/finfocus-spec/issues/244)) -
  Added standalone conformance test for GetPluginInfo response time (<100ms assertion).
- [x] **User-Friendly Error Messages** ([#245](https://github.com/rshade/finfocus-spec/issues/245)) -
  Improved GetPluginInfo error messages for end-users.
- [x] **TypeScript SDK** ([#293](https://github.com/rshade/finfocus-spec/issues/293),
  [#302](https://github.com/rshade/finfocus-spec/pull/302)) -
  Initial TypeScript client SDK for browser and Node.js environments.
- [x] **TypeScript SDK Connect-ES v2 Migration** ([#304](https://github.com/rshade/finfocus-spec/issues/304)) -
  Updated TypeScript SDK to use Connect-ES v2 for improved browser/Node.js support.
- [x] **TypeScript SDK Publishing Infrastructure** ([#311](https://github.com/rshade/finfocus-spec/issues/311)) -
  Automated publishing pipeline for TypeScript SDK to npm.
- [x] **TypeScript SDK Publishing Enhancements**
  ([#313](https://github.com/rshade/finfocus-spec/issues/313)) -
  Additional publishing enhancements from PR #312 review.
- [x] **Plugin Capability Enum** ([#287](https://github.com/rshade/finfocus-spec/issues/287)) -
  Added PluginCapability enum for granular feature discovery.
- [x] **Capability Discovery Robustness**
  ([#297](https://github.com/rshade/finfocus-spec/issues/297),
  [#296](https://github.com/rshade/finfocus-spec/issues/296)) -
  Fixed nil check in inferCapabilities and added test coverage for capability override edge cases.
- [x] **Capability Discovery Bug Fixes**
  ([#294](https://github.com/rshade/finfocus-spec/issues/294),
  [#295](https://github.com/rshade/finfocus-spec/issues/295),
  [#299](https://github.com/rshade/finfocus-spec/issues/299),
  [#300](https://github.com/rshade/finfocus-spec/issues/300),
  [#301](https://github.com/rshade/finfocus-spec/issues/301)) -
  Fixed DryRunHandler interface, proto field consistency, backward compatibility patterns, docs, and slice performance.
- [x] **Backward Compatibility for Environment Variables**
  ([#283](https://github.com/rshade/finfocus-spec/issues/283)) -
  Maintained legacy PULUMICOST_* environment variable support during migration.
- [x] **Migration Documentation** ([#282](https://github.com/rshade/finfocus-spec/issues/282)) -
  Added MIGRATION.md and llm-migration.json for PulumiCost to FinFocus migration.
- [x] **PulumiCost to FinFocus Rename** ([#272](https://github.com/rshade/finfocus-spec/issues/272)) -
  Complete project rename with backward compatibility shims.
- [x] **Log File Security Warning**
  ([#284](https://github.com/rshade/finfocus-spec/issues/284)) -
  Improved security guidance for log file handling.
- [x] **Unclear Exhaustive Nolint Directive**
  ([#298](https://github.com/rshade/finfocus-spec/issues/298)) -
  Clarified the nolint directive in legacyCapabilityMap.
- [x] **Float Validation Harmonization**
  ([#336](https://github.com/rshade/finfocus-spec/issues/336)) -
  Harmonized float validation patterns for spot risk score.
- [x] **TypeScript SDK Documentation**
  ([#334](https://github.com/rshade/finfocus-spec/issues/334)) -
  TypeScript SDK documentation improvements from technical review.
- [x] **Bypass Metadata Security**
  ([#340](https://github.com/rshade/finfocus-spec/issues/340)) -
  Added bypass metadata security and performance enhancements.
- [x] **Documentation Drift Audit**
  ([#346](https://github.com/rshade/finfocus-spec/issues/346),
  [#347](https://github.com/rshade/finfocus-spec/issues/347),
  [#348](https://github.com/rshade/finfocus-spec/issues/348)) -
  Comprehensive audit of SDK READMEs and root README for documentation accuracy.
- [x] **Pagination Hardening**
  ([#364](https://github.com/rshade/finfocus-spec/issues/364),
  [#367](https://github.com/rshade/finfocus-spec/issues/367),
  [#368](https://github.com/rshade/finfocus-spec/issues/368),
  [#369](https://github.com/rshade/finfocus-spec/issues/369)) -
  Upper bound token checks, edge case tests, TypeScript null guards, and page size clamping warnings.
- [x] **Iterator Concurrency Safety Documentation**
  ([#366](https://github.com/rshade/finfocus-spec/issues/366)) -
  Expanded `ActualCostIterator` docs with read-while-write hazard warnings.
- [x] **Expand Pagination SDK Documentation**
  ([#371](https://github.com/rshade/finfocus-spec/issues/371)) -
  Added token opacity guidance, migration guide, and edge case handling to SDK README.
- [x] **Make CapabilitiesToLegacyMetadataWithWarnings Default**
  ([#345](https://github.com/rshade/finfocus-spec/issues/345)) -
  Improved backward compatibility defaults for capability discovery.
- [x] **Upgrade golangci-lint**
  ([#350](https://github.com/rshade/finfocus-spec/issues/350)) -
  Upgraded from v2.6.2 to v2.9.0.

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
