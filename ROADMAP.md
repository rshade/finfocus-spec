# Strategic Roadmap: finfocus-spec

## Vision

To provide the definitive, high-performance gRPC specification and Go SDK for cloud cost observability,
centered around the FinOps Foundation's FOCUS standard.

---

## Immediate Focus (Q2 2026)

### Pagination Hardening

- [ ] **Enforce Token Opacity in Proto Comments**
  ([#363](https://github.com/rshade/finfocus-spec/issues/363)) [S] -
  Add explicit warning that `page_token` is opaque; clients MUST NOT parse or construct tokens.
- [ ] **Warn on TotalCount Change Mid-Iteration**
  ([#365](https://github.com/rshade/finfocus-spec/issues/365)) [S] -
  Log zerolog warning when `total_count` changes between pages during iteration.

### GetProjectedCost Enrichment

- [ ] **Plugin Contract for GetProjectedCost**
  ([#377](https://github.com/rshade/finfocus-spec/issues/377)) [M] -
  Document plugin contract for sparse/old-state property handling in projected cost responses.
- [ ] **Metadata Map for Confidence Signals**
  ([#381](https://github.com/rshade/finfocus-spec/issues/381)) [M] -
  Add metadata map to GetProjectedCostResponse for property confidence signals.

### SDK Enhancements

- [ ] **Add StreamResult.ValidateOutputJSON() Helper**
  ([#344](https://github.com/rshade/finfocus-spec/issues/344)) [M] -
  Helper method for JSON-LD output validation.
- [ ] **Integrate ValidationError Type**
  ([#210](https://github.com/rshade/finfocus-spec/issues/210)) [M] -
  Integrate ValidationError type with validation implementation.
- [ ] **Extract ResourceDescriptor Test Helper**
  ([#204](https://github.com/rshade/finfocus-spec/issues/204)) [S] -
  Refactor test helper for ResourceDescriptor creation.

### Batch RPC Hardening

- [ ] **Document DryRunHandler.HandleDryRun Breaking Change**
  ([#396](https://github.com/rshade/finfocus-spec/issues/396)) [S] -
  Document DryRunHandler.HandleDryRun interface change for plugin authors.
- [ ] **Remove Unnecessary proto.Clone on BatchCostRequest**
  ([#397](https://github.com/rshade/finfocus-spec/issues/397)) [S] -
  Remove unnecessary proto.Clone in Server.BatchCost for performance.
- [ ] **Fix Unbounded Goroutine Creation in batchCostFallback**
  ([#398](https://github.com/rshade/finfocus-spec/issues/398)) [M] -
  Add goroutine pool bounds to prevent unbounded concurrency.
- [ ] **Deduplicate grpcCodeToInt32**
  ([#399](https://github.com/rshade/finfocus-spec/issues/399)) [S] -
  Deduplicate grpcCodeToInt32 between pluginsdk and testing packages.
- [ ] **Improve ValidateBatchCostRequest Return Type**
  ([#400](https://github.com/rshade/finfocus-spec/issues/400)) [M] -
  Improve return type to avoid (nil, nil) sentinel pattern.
- [ ] **Add ResourceDescriptor Field Validation Limits**
  ([#401](https://github.com/rshade/finfocus-spec/issues/401)) [M] -
  Add field validation limits to batch request validation.

### Stability & Maintenance

- [ ] **Dependency Management** ([#13](https://github.com/rshade/finfocus-spec/issues/13)) -
  Automated tracking and updating of core proto and SDK dependencies.

---

## Near-Term Vision

### Batch RPC Code Quality

- [ ] **Avoid Full proto.Clone in BatchCost Normalization**
  ([#394](https://github.com/rshade/finfocus-spec/issues/394)) [M] -
  Avoid full proto.Clone in BatchCost request normalization for performance.
- [ ] **Add PluginInfoProvider + BatchCostHandler Test Coverage**
  ([#395](https://github.com/rshade/finfocus-spec/issues/395)) [S] -
  Add test coverage for PluginInfoProvider + BatchCostHandler combination.
- [ ] **Benchmark descriptorClone Overhead**
  ([#402](https://github.com/rshade/finfocus-spec/issues/402)) [M] -
  Benchmark and evaluate descriptorClone overhead in batch result construction.
- [ ] **Resolve Concurrency TODO in BatchCost Tests**
  ([#403](https://github.com/rshade/finfocus-spec/issues/403)) [M] -
  Resolve TODO in TestBatchCostFallbackWorkerPoolConfig for concurrency assertions.
- [ ] **Replace Magic Constant defaultInternalErrCode**
  ([#404](https://github.com/rshade/finfocus-spec/issues/404)) [S] -
  Replace magic constant with `codes.Internal` from gRPC.
- [ ] **Extract Helper from MockPlugin.BatchCost**
  ([#405](https://github.com/rshade/finfocus-spec/issues/405)) [M] -
  Extract per-resource helper to reduce cognitive complexity.
- [ ] **Improve ResourceError.code Proto Comment**
  ([#406](https://github.com/rshade/finfocus-spec/issues/406)) [S] -
  Add full gRPC status code reference to proto comment.

---

## Future Vision (Long-Term)

### Active Research

- [ ] **Standardized Cost Allocation Lineage Metadata**
  ([#191](https://github.com/rshade/finfocus-spec/issues/191)) [L]
- [ ] **Distributed Tracing Propagation (Contextual Visibility)**
  ([#193](https://github.com/rshade/finfocus-spec/issues/193)) [L]
- [ ] **Authorization Middleware (OIDC/IAM)**
  ([#195](https://github.com/rshade/finfocus-spec/issues/195)) [L] -
  Standardizing how plugins receive and validate identity without violating "Stateless" boundaries.

### Completed Research

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
- [ ] **Multi-Currency Segregation Pattern**
  ([#190](https://github.com/rshade/finfocus-spec/issues/190)) [L]
- [ ] **Per-Request Credential Passing** ([#220](https://github.com/rshade/finfocus-spec/issues/220)) [L] -
  Multi-tenant optimization allowing per-request cloud credentials.

---

## Completed Milestones

### Q1 2026

#### Protocol & Modeling

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

- [x] **GreenOps Standardization** ([#176](https://github.com/rshade/finfocus-spec/issues/176)) -
  Sustainability metrics (carbon footprint, energy utilization) integrated into the core protocol.
- [x] **Recommendation Enhancements**
  ([#173](https://github.com/rshade/finfocus-spec/issues/173),
  [#171](https://github.com/rshade/finfocus-spec/issues/171),
  [#166](https://github.com/rshade/finfocus-spec/issues/166)) -
  Added resource-scoped targets, filtering, and extended action types.
- [x] **FOCUS 1.2 Integration**
  ([#100](https://github.com/rshade/finfocus-spec/issues/100),
  [#99](https://github.com/rshade/finfocus-spec/issues/99)) -
  Full schema coverage and builder API for FOCUS 1.2.
- [x] **RPC Expansion**
  ([#125](https://github.com/rshade/finfocus-spec/issues/125),
  [#123](https://github.com/rshade/finfocus-spec/issues/123),
  [#90](https://github.com/rshade/finfocus-spec/issues/90)) -
  Implemented `EstimateCost`, `GetBudgets`, and `GetRecommendations`.
- [x] **Zero-Allocation Enums**
  ([#63](https://github.com/rshade/finfocus-spec/issues/63),
  [#33](https://github.com/rshade/finfocus-spec/issues/33)) -
  High-performance validation for domain enums.

#### SDK & Tooling

- [x] **Plugin Conformance Suite** ([#109](https://github.com/rshade/finfocus-spec/issues/109)) -
  Automated testing for plugin implementers to ensure spec compliance.
- [x] **SDK Foundation**
  ([#151](https://github.com/rshade/finfocus-spec/issues/151),
  [#148](https://github.com/rshade/finfocus-spec/issues/148),
  [#145](https://github.com/rshade/finfocus-spec/issues/145),
  [#139](https://github.com/rshade/finfocus-spec/issues/139)) -
  Centralized environment handling, Prometheus metrics, and unified file-based logging.
- [x] **Orchestration Support**
  ([#181](https://github.com/rshade/finfocus-spec/issues/181),
  [#143](https://github.com/rshade/finfocus-spec/issues/143),
  [#126](https://github.com/rshade/finfocus-spec/issues/126)) -
  Added reflection, `--port` flags, and `fallback-hints`.
- [x] **Multi-Protocol Support (gRPC-Web/Connect)**
  ([#189](https://github.com/rshade/finfocus-spec/issues/189),
  [#223](https://github.com/rshade/finfocus-spec/pull/223)) -
  Added connect-go integration enabling gRPC, gRPC-Web, and Connect protocols for browser compatibility.

---

## Boundary Safeguards (The "Hard No's")

- **No Orchestration Logic**: Multi-plugin aggregation and "Muxing" logic will stay in `finfocus-core`.
- **No Persistence**: The SDK will not provide built-in database support; it remains stateless.
- **No Native Math Engines**: The SDK will not perform financial amortization;
  it will only define the fields to store the results of such calculations.
