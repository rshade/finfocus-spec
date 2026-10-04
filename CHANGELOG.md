# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.3](https://github.com/rshade/finfocus-spec/compare/v0.7.2...v0.7.3) (2026-10-04)


### Added

* **proto:** add ResourceDescriptor attributes and raise tag limit ([7843e7d](https://github.com/rshade/finfocus-spec/commit/7843e7dc831e0e4f10f4809039c55dd0aca42648)), closes [#617](https://github.com/rshade/finfocus-spec/issues/617)


### Fixed

* address PR 618 review and contract suite allocation ([fa0f5e3](https://github.com/rshade/finfocus-spec/commit/fa0f5e39af16875ec3dee41c8d4ee69420944637))

## [0.7.2](https://github.com/rshade/finfocus-spec/compare/v0.7.1...v0.7.2) (2026-10-03)


### Added

* **registry:** write plugin manifests the schema and validator accept ([d15f899](https://github.com/rshade/finfocus-spec/commit/d15f8997d24b65997c90ea545f1b74779e1b6db1)), closes [#611](https://github.com/rshade/finfocus-spec/issues/611)


### Fixed

* **pluginsdk:** reject two keys that name one manifest field ([90debbd](https://github.com/rshade/finfocus-spec/commit/90debbd031e0e341a4527b52fed94dad7aebabd0))


### Performance

* **registry:** group capabilities by length; fix example manifests ([4f81f96](https://github.com/rshade/finfocus-spec/commit/4f81f96827a76ff20ee549b82ed7f920cfe062a9))


### Documentation

* **proto:** define provider as the cloud, not the IaC package ([56ef520](https://github.com/rshade/finfocus-spec/commit/56ef52023eb85df91c8c835ff13fea55bcfe9507))
* state how Pulumi inputs reach plugin tags ([d1641a7](https://github.com/rshade/finfocus-spec/commit/d1641a7396ac732a4eed1900e5e0b5dfdf547b92)), closes [#609](https://github.com/rshade/finfocus-spec/issues/609)

## [0.7.1](https://github.com/rshade/finfocus-spec/compare/v0.7.0...v0.7.1) (2026-10-02)


### Added

* **pluginsdk:** advertise scorer limits and mark oversize batches ([#602](https://github.com/rshade/finfocus-spec/issues/602)) ([db5da5b](https://github.com/rshade/finfocus-spec/commit/db5da5b5e258bbf1e0a804d50b5f8ab2248d9af0))
* **proto:** add billing_account_id to GetActualCostRequest ([#592](https://github.com/rshade/finfocus-spec/issues/592)) ([0458180](https://github.com/rshade/finfocus-spec/commit/04581801101312240f5d3974fe30cd35353c39f4)), closes [#590](https://github.com/rshade/finfocus-spec/issues/590)
* **proto:** add FOCUS 1.3 provenance fields to AllocationRow ([#605](https://github.com/rshade/finfocus-spec/issues/605)) ([c1a07b0](https://github.com/rshade/finfocus-spec/commit/c1a07b02a33ce0d2b1c55969b276846762b56abd)), closes [#578](https://github.com/rshade/finfocus-spec/issues/578)
* **proto:** add omitted_fields to ScoreRecommendationsRequest ([#604](https://github.com/rshade/finfocus-spec/issues/604)) ([9eccf57](https://github.com/rshade/finfocus-spec/commit/9eccf57a87b5c3f63dc84d19566af39fef56c366)), closes [#576](https://github.com/rshade/finfocus-spec/issues/576)
* **proto:** add period and selector to AllocateRequest ([#606](https://github.com/rshade/finfocus-spec/issues/606)) ([0427ae2](https://github.com/rshade/finfocus-spec/commit/0427ae2ba67a96c7dd29de55f9d03bce1a5941f3)), closes [#579](https://github.com/rshade/finfocus-spec/issues/579)
* **proto:** add RegionPrice for per-region retail prices ([#600](https://github.com/rshade/finfocus-spec/issues/600)) ([50d7d95](https://github.com/rshade/finfocus-spec/commit/50d7d95781c5f359c27dbdc1c8316b5ebd9b0c65)), closes [#589](https://github.com/rshade/finfocus-spec/issues/589)
* **proto:** add repeated PriceOption for alternative retail prices ([#599](https://github.com/rshade/finfocus-spec/issues/599)) ([402f84d](https://github.com/rshade/finfocus-spec/commit/402f84de5269052792d695df5591b6924dc39897)), closes [#588](https://github.com/rshade/finfocus-spec/issues/588)
* **proto:** add scorer session_id so duplicate groups span batches ([#603](https://github.com/rshade/finfocus-spec/issues/603)) ([6c078c4](https://github.com/rshade/finfocus-spec/commit/6c078c46da73d34498248b335e18d3ca8c53b8b3)), closes [#574](https://github.com/rshade/finfocus-spec/issues/574)
* **proto:** list every request id and model on ScorerInfo ([#608](https://github.com/rshade/finfocus-spec/issues/608)) ([7a15bd6](https://github.com/rshade/finfocus-spec/commit/7a15bd6fa9858089a6c8ee9c35aabab6ee7dad0a))


### Fixed

* **deps:** require grpc v1.83.2 to avoid GO-2026-6443 ([#583](https://github.com/rshade/finfocus-spec/issues/583)) ([7ecaa46](https://github.com/rshade/finfocus-spec/commit/7ecaa46d31d8b2e8113df022a05b6aaf7b0ab960)), closes [#582](https://github.com/rshade/finfocus-spec/issues/582)
* **pluginsdk:** stamp trace id on a copy of the handler ValidationError ([#595](https://github.com/rshade/finfocus-spec/issues/595)) ([5b0edd8](https://github.com/rshade/finfocus-spec/commit/5b0edd84b10d324694b39df288b9ffde300794f6)), closes [#563](https://github.com/rshade/finfocus-spec/issues/563)
* **sdk:** harden REST gateway error text and body limits ([#598](https://github.com/rshade/finfocus-spec/issues/598)) ([84e00a6](https://github.com/rshade/finfocus-spec/commit/84e00a63a2f6bb8d5472512ca983c5a14dbcb82e)), closes [#565](https://github.com/rshade/finfocus-spec/issues/565)
* **testing:** report first singleton duplicate group in result order ([#585](https://github.com/rshade/finfocus-spec/issues/585)) ([b8f6d53](https://github.com/rshade/finfocus-spec/commit/b8f6d53d68555b7565c7d24e0223e34bffe32808)), closes [#584](https://github.com/rshade/finfocus-spec/issues/584)
* **testing:** validate contract commitment applicability as real JSON ([#597](https://github.com/rshade/finfocus-spec/issues/597)) ([fbc1488](https://github.com/rshade/finfocus-spec/commit/fbc148868770c51e1c089c39f17af02dfc91d3e9)), closes [#562](https://github.com/rshade/finfocus-spec/issues/562)


### Changed

* **testing:** share bufconn harness and duplicate-key scan ([#607](https://github.com/rshade/finfocus-spec/issues/607)) ([0094cac](https://github.com/rshade/finfocus-spec/commit/0094cace40e920e2e766047cfc7e63af29b4dc0a)), closes [#561](https://github.com/rshade/finfocus-spec/issues/561)


### Documentation

* define identifier_mode scope and complete recommendation wording ([#594](https://github.com/rshade/finfocus-spec/issues/594)) ([f8111b9](https://github.com/rshade/finfocus-spec/commit/f8111b95e7e23c1aa6f071c81906e6b8e077a563)), closes [#577](https://github.com/rshade/finfocus-spec/issues/577)
* define recommendation score cache key and validity rules ([#591](https://github.com/rshade/finfocus-spec/issues/591)) ([93eab79](https://github.com/rshade/finfocus-spec/commit/93eab79ed57cf4319889f0d4db8cede1171582b6))
* **pluginsdk:** document failing closed and filtering credential keys ([#593](https://github.com/rshade/finfocus-spec/issues/593)) ([aeeeea4](https://github.com/rshade/finfocus-spec/commit/aeeeea47bf4b24666caf2d482ca571423d995a0d)), closes [#566](https://github.com/rshade/finfocus-spec/issues/566)
* **pluginsdk:** document per-request credentials for optional services ([#596](https://github.com/rshade/finfocus-spec/issues/596)) ([a189386](https://github.com/rshade/finfocus-spec/commit/a18938601158aac69c900663784aa4b9190c7c54)), closes [#575](https://github.com/rshade/finfocus-spec/issues/575)
* sync ROADMAP.md with GitHub after v0.7.0 ([a3ace6b](https://github.com/rshade/finfocus-spec/commit/a3ace6b4258983dd1caec8c0f8364a2a22fa8ee9))
* sync ROADMAP.md with GitHub after v0.7.0 ([949c6ca](https://github.com/rshade/finfocus-spec/commit/949c6ca753523cd59eea1d2ec5ebf46cdc531345))

## [0.7.0](https://github.com/rshade/finfocus-spec/compare/v0.6.2...v0.7.0) (2026-09-29)


### Added

* **pluginsdk:** add FOCUS 1.4 cost and usage columns ([#549](https://github.com/rshade/finfocus-spec/issues/549)) ([48b5dcd](https://github.com/rshade/finfocus-spec/commit/48b5dcd8bf0c1387d30ea752d54965ebf47a6ff9)), closes [#541](https://github.com/rshade/finfocus-spec/issues/541)
* **pluginsdk:** add opt-in per-request credentials ([#550](https://github.com/rshade/finfocus-spec/issues/550)) ([67b47da](https://github.com/rshade/finfocus-spec/commit/67b47da74a541aa11bb42efd6f5f9bac0af4c23f)), closes [#220](https://github.com/rshade/finfocus-spec/issues/220)
* **proto:** add cost allocation lineage metadata ([#546](https://github.com/rshade/finfocus-spec/issues/546)) ([d85660a](https://github.com/rshade/finfocus-spec/commit/d85660ae273be1e1d41c74b77417e8f76bd2f1f8)), closes [#191](https://github.com/rshade/finfocus-spec/issues/191)
* **proto:** add cost_breakdown map to GetProjectedCostResponse ([#537](https://github.com/rshade/finfocus-spec/issues/537)) ([7a29919](https://github.com/rshade/finfocus-spec/commit/7a29919480c2e0d3260262387f32a57d1975afed))
* **proto:** add FOCUS 1.4 billing period and invoice detail ([#554](https://github.com/rshade/finfocus-spec/issues/554)) ([5220a1a](https://github.com/rshade/finfocus-spec/commit/5220a1a44bd6b2406b2e9e5928f4b229fc86ca2a)), closes [#543](https://github.com/rshade/finfocus-spec/issues/543)
* **proto:** add FOCUS 1.4 contract commitment columns ([#553](https://github.com/rshade/finfocus-spec/issues/553)) ([487e2c0](https://github.com/rshade/finfocus-spec/commit/487e2c049704b9cdfb5f073def4c931ece6490b1)), closes [#542](https://github.com/rshade/finfocus-spec/issues/542)
* **proto:** add RecommendationScorerService.ScoreRecommendations ([#559](https://github.com/rshade/finfocus-spec/issues/559)) ([8020f96](https://github.com/rshade/finfocus-spec/commit/8020f968a8b9459a26f1b9f9f4be7991cb9a1fe3)), closes [#556](https://github.com/rshade/finfocus-spec/issues/556)
* **proto:** add SupplementalDatasetService for contract commitments ([#552](https://github.com/rshade/finfocus-spec/issues/552)) ([91c1374](https://github.com/rshade/finfocus-spec/commit/91c13748b4ebe5ceeaee285d097c6dec10901193)), closes [#544](https://github.com/rshade/finfocus-spec/issues/544)
* **proto:** serve FOCUS billing period and invoice detail ([#555](https://github.com/rshade/finfocus-spec/issues/555)) ([31e9006](https://github.com/rshade/finfocus-spec/commit/31e900635acd053ee17854fd02059828a0a9a2d4))


### Fixed

* **pluginsdk:** keep ContractCommitmentBuilder.Build on pre-1.4 rules ([#558](https://github.com/rshade/finfocus-spec/issues/558)) ([b785bcf](https://github.com/rshade/finfocus-spec/commit/b785bcfbca94f73395de0384f5f763d7990049bf))
* **pluginsdk:** log host trace id on validation failures ([#538](https://github.com/rshade/finfocus-spec/issues/538)) ([a5e6790](https://github.com/rshade/finfocus-spec/commit/a5e6790d0cda2ccfad7e9fa72dbbbec90c8255d9)), closes [#193](https://github.com/rshade/finfocus-spec/issues/193)
* **pluginsdk:** preserve lineage when cloning batch resource descriptors ([#557](https://github.com/rshade/finfocus-spec/issues/557)) ([02befb9](https://github.com/rshade/finfocus-spec/commit/02befb9073cc205692468adf2272f6abddbcca18))
* **pluginsdk:** refuse mixed-currency recommendation totals ([#536](https://github.com/rshade/finfocus-spec/issues/536)) ([a7dde3a](https://github.com/rshade/finfocus-spec/commit/a7dde3aa862f4e27b6c58f2e18116c5bc27e4c94)), closes [#190](https://github.com/rshade/finfocus-spec/issues/190)
* **sdk:** make middleware and framework-plugins build and work ([#529](https://github.com/rshade/finfocus-spec/issues/529)) ([b3ad25c](https://github.com/rshade/finfocus-spec/commit/b3ad25c7fe340f4522537eec6fb73306202d5a1d)), closes [#513](https://github.com/rshade/finfocus-spec/issues/513)


### Performance

* **pluginsdk:** replace proto.Clone with manual copy in descriptorClone ([#530](https://github.com/rshade/finfocus-spec/issues/530)) ([6f27a92](https://github.com/rshade/finfocus-spec/commit/6f27a92d165676a90cbff443c6ea2995da100fb3)), closes [#402](https://github.com/rshade/finfocus-spec/issues/402)


### Documentation

* **pluginsdk:** plan trace id on validation failures ([#548](https://github.com/rshade/finfocus-spec/issues/548)) ([8a7668f](https://github.com/rshade/finfocus-spec/commit/8a7668f0f65481682da93d602ce49e238dde4059)), closes [#193](https://github.com/rshade/finfocus-spec/issues/193)
* record cost lineage assessment ([#533](https://github.com/rshade/finfocus-spec/issues/533)) ([79d8b0d](https://github.com/rshade/finfocus-spec/commit/79d8b0d9b21a619ffd2c7b66454518416741f5e3)), closes [#191](https://github.com/rshade/finfocus-spec/issues/191)
* record FOCUS 1.4 support assessment ([#545](https://github.com/rshade/finfocus-spec/issues/545)) ([538daa8](https://github.com/rshade/finfocus-spec/commit/538daa8d4efb1ba1ddcb061f8f8d4a16c5325307)), closes [#540](https://github.com/rshade/finfocus-spec/issues/540)
* record per-request credential assessment ([#535](https://github.com/rshade/finfocus-spec/issues/535)) ([6383526](https://github.com/rshade/finfocus-spec/commit/638352619ca660cb01411c58b3bef2a163710271)), closes [#220](https://github.com/rshade/finfocus-spec/issues/220)
* record trace propagation assessment ([#534](https://github.com/rshade/finfocus-spec/issues/534)) ([90b67ca](https://github.com/rshade/finfocus-spec/commit/90b67ca9de5d14447e00d945ef7e5c6ef62cd3bc)), closes [#193](https://github.com/rshade/finfocus-spec/issues/193)


### Chores

* release 0.7.0 ([edb377a](https://github.com/rshade/finfocus-spec/commit/edb377a03ebb38c67c793fd7cf5d986a153b7c32))

## [0.6.2](https://github.com/rshade/finfocus-spec/compare/v0.6.1...v0.6.2) (2026-09-28)


### Added

* **pluginsdk:** add ValidateAllocateResponse and fix allocator checks ([#518](https://github.com/rshade/finfocus-spec/issues/518)) ([25b3b27](https://github.com/rshade/finfocus-spec/commit/25b3b27630f0c9eb28a195c65445afaa80e543aa))
* **proto:** add AllocatorService.Allocate for cost allocation plugins ([#515](https://github.com/rshade/finfocus-spec/issues/515)) ([063e0b9](https://github.com/rshade/finfocus-spec/commit/063e0b9aabc0c9950564c8859d3288e6eedcb1bc)), closes [#506](https://github.com/rshade/finfocus-spec/issues/506)
* **proto:** add UsageSourceService.GetStats for workload usage plugins ([#508](https://github.com/rshade/finfocus-spec/issues/508)) ([d314c2a](https://github.com/rshade/finfocus-spec/commit/d314c2af06c42e05cdd76f973eb3fd6d7ed1b513)), closes [#505](https://github.com/rshade/finfocus-spec/issues/505)


### Fixed

* **pluginsdk:** make advertised capabilities reachable by hosts ([#510](https://github.com/rshade/finfocus-spec/issues/510)) ([e51f8b4](https://github.com/rshade/finfocus-spec/commit/e51f8b4b0e51901ca1071b0f700cd134528c32e4))

## [0.6.1](https://github.com/rshade/finfocus-spec/compare/v0.6.0...v0.6.1) (2026-09-08)


### Added

* **pluginsdk:** add ax-go CLI entry point via Run() ([#495](https://github.com/rshade/finfocus-spec/issues/495)) ([4dc11db](https://github.com/rshade/finfocus-spec/commit/4dc11dbdb3d84fa6484f5b31352b2b8930ab573a))
* **proto:** add expires_at cache-hint to EstimateCostResponse ([#460](https://github.com/rshade/finfocus-spec/issues/460)) ([d258c1b](https://github.com/rshade/finfocus-spec/commit/d258c1b2cb3f15a20f4ad887aba9c339ead0c6f9)), closes [#434](https://github.com/rshade/finfocus-spec/issues/434)
* **sdk:** harden ResolveResourceTypes RPC ([#493](https://github.com/rshade/finfocus-spec/issues/493)) ([50f73d9](https://github.com/rshade/finfocus-spec/commit/50f73d985d3485fb1c59ad4889d1a3021073562a))

## [0.6.0](https://github.com/rshade/finfocus-spec/compare/v0.5.7...v0.6.0) (2026-03-13)


### ⚠ BREAKING CHANGES

* **migration:** DryRunHandler.HandleDryRun signature changed from HandleDryRun(req *pbc.DryRunRequest) to HandleDryRun(ctx context.Context, req *pbc.DryRunRequest). This is a compile-time breaking change affecting all plugins implementing the DryRunHandler interface.

### Added

* add streamresult validation helper methods ([#425](https://github.com/rshade/finfocus-spec/issues/425)) ([ab458e6](https://github.com/rshade/finfocus-spec/commit/ab458e6b6e9497fe9a76dd02c82329239976c286)), closes [#344](https://github.com/rshade/finfocus-spec/issues/344)
* **pluginsdk:** add ResourceDescriptor field validation limits ([#431](https://github.com/rshade/finfocus-spec/issues/431)) ([f2c07eb](https://github.com/rshade/finfocus-spec/commit/f2c07ebece59884fcbdc3640aebc4a943cf0ae89))
* **pluginsdk:** warn when total_count changes mid-iteration ([#430](https://github.com/rshade/finfocus-spec/issues/430)) ([523fcad](https://github.com/rshade/finfocus-spec/commit/523fcadf23ba687f038e0046903c64e07829cea4)), closes [#365](https://github.com/rshade/finfocus-spec/issues/365)
* **proto:** add metadata map to getprojectedcostresponse ([#427](https://github.com/rshade/finfocus-spec/issues/427)) ([e624864](https://github.com/rshade/finfocus-spec/commit/e62486432c58fccdf5b1d79c5aa13c9a6adf0069)), closes [#381](https://github.com/rshade/finfocus-spec/issues/381)


### Fixed

* **pluginsdk:** harden metadata map validation and add sentinel errors ([#435](https://github.com/rshade/finfocus-spec/issues/435)) ([1ee2b12](https://github.com/rshade/finfocus-spec/commit/1ee2b12c5504162d59265f47272857554ee7511d)), closes [#381](https://github.com/rshade/finfocus-spec/issues/381)


### Performance

* **pluginsdk:** avoid proto.Clone in BatchCost fallback path ([#417](https://github.com/rshade/finfocus-spec/issues/417)) ([73b0f67](https://github.com/rshade/finfocus-spec/commit/73b0f67984730321f34b6e50a90b65a5bfcc4a62)), closes [#394](https://github.com/rshade/finfocus-spec/issues/394)
* **pluginsdk:** fix unbounded goroutine creation in batchCostFallback ([#424](https://github.com/rshade/finfocus-spec/issues/424)) ([827e2d1](https://github.com/rshade/finfocus-spec/commit/827e2d115c1d71b0d91511576abec760cc2336c9)), closes [#398](https://github.com/rshade/finfocus-spec/issues/398)
* **pluginsdk:** remove unnecessary proto.clone in batchcost handler ([#422](https://github.com/rshade/finfocus-spec/issues/422)) ([812f70b](https://github.com/rshade/finfocus-spec/commit/812f70b624915e7888809a0cf1cd2bf93067c92f)), closes [#397](https://github.com/rshade/finfocus-spec/issues/397)


### Changed

* deduplicate grpccodetoint32 into shared internal package ([#426](https://github.com/rshade/finfocus-spec/issues/426)) ([6a02f49](https://github.com/rshade/finfocus-spec/commit/6a02f49282274daf2f486daf2875de032979b154))
* **pluginsdk:** extract newTestDescriptor test helper ([#442](https://github.com/rshade/finfocus-spec/issues/442)) ([5604a57](https://github.com/rshade/finfocus-spec/commit/5604a572f218c20149becbda8ddef6821f5830e5)), closes [#204](https://github.com/rshade/finfocus-spec/issues/204)
* **pluginsdk:** improve ValidateBatchCostRequest return type ([#420](https://github.com/rshade/finfocus-spec/issues/420)) ([3c09962](https://github.com/rshade/finfocus-spec/commit/3c099623b87e184b5e9fdbbd9d444cad0aa1131b)), closes [#400](https://github.com/rshade/finfocus-spec/issues/400)
* **pluginsdk:** integrate ValidationError with all validation sites ([#438](https://github.com/rshade/finfocus-spec/issues/438)) ([2a7382b](https://github.com/rshade/finfocus-spec/commit/2a7382bb39960547451914103fc2f4c1b81f63e5)), closes [#210](https://github.com/rshade/finfocus-spec/issues/210)
* **testing:** extract batchsingleresource helper from mockplugin ([#421](https://github.com/rshade/finfocus-spec/issues/421)) ([c437616](https://github.com/rshade/finfocus-spec/commit/c437616e7f8a5c249cccfbe5dec59db7ed78259b)), closes [#405](https://github.com/rshade/finfocus-spec/issues/405)
* **testing:** replace magic constant with codes.Internal ([#429](https://github.com/rshade/finfocus-spec/issues/429)) ([e0b17f7](https://github.com/rshade/finfocus-spec/commit/e0b17f7ac6faecaddc9d29c2dbc3055aad71c010)), closes [#404](https://github.com/rshade/finfocus-spec/issues/404)


### Documentation

* **migration:** document dryrunhandler.handledryrun breaking change ([#423](https://github.com/rshade/finfocus-spec/issues/423)) ([3933722](https://github.com/rshade/finfocus-spec/commit/3933722c5905ba1706eab152e8e08754cf7826a4)), closes [#396](https://github.com/rshade/finfocus-spec/issues/396)
* **proto:** add grpc code reference to resourceerror.code ([#415](https://github.com/rshade/finfocus-spec/issues/415)) ([587daed](https://github.com/rshade/finfocus-spec/commit/587daed76a5b4ca2496dfd473524adf869346314)), closes [#406](https://github.com/rshade/finfocus-spec/issues/406)
* **proto:** enforce page_token opacity with explicit warnings ([#428](https://github.com/rshade/finfocus-spec/issues/428)) ([6e4af60](https://github.com/rshade/finfocus-spec/commit/6e4af60ddd4ab76008d83e56df2cd7ee7a51e882)), closes [#363](https://github.com/rshade/finfocus-spec/issues/363)
* **sdk:** add plugin contract for getprojectedcost ([#416](https://github.com/rshade/finfocus-spec/issues/416)) ([f4b35b7](https://github.com/rshade/finfocus-spec/commit/f4b35b79995860b16256a71fc7e41dcbf2e8da22)), closes [#377](https://github.com/rshade/finfocus-spec/issues/377)

## [0.5.7](https://github.com/rshade/finfocus-spec/compare/v0.5.6...v0.5.7) (2026-02-28)


### Added

* **proto:** add batch cost operation for multi-resource queries ([#392](https://github.com/rshade/finfocus-spec/issues/392)) ([15addc9](https://github.com/rshade/finfocus-spec/commit/15addc9a3792c186a6dad703c8b608ea674bbebe))
* **proto:** add expires_at caching hint to cost results ([#382](https://github.com/rshade/finfocus-spec/issues/382)) ([6bf8705](https://github.com/rshade/finfocus-spec/commit/6bf8705a12111183a357180cbc19e589673bc636))


### Documentation

* **proto:** add batch cost pagination continuation docs ([#408](https://github.com/rshade/finfocus-spec/issues/408)) ([0a3a47c](https://github.com/rshade/finfocus-spec/commit/0a3a47ca1d73be3e7a3efe444e84eb398358cbb4))

## [0.5.6](https://github.com/rshade/finfocus-spec/compare/v0.5.5...v0.5.6) (2026-02-12)


### Added

* **currency:** add symbol support and amount formatting ([#357](https://github.com/rshade/finfocus-spec/issues/357)) ([698cbe3](https://github.com/rshade/finfocus-spec/commit/698cbe326ef326499b8c6b4d20d451eb9926c1c0))
* **proto:** add pagination support to actual cost retrieval ([#360](https://github.com/rshade/finfocus-spec/issues/360)) ([571a0cd](https://github.com/rshade/finfocus-spec/commit/571a0cdd7c4a822e74dd4a688972bab35fa9afa7)), closes [#353](https://github.com/rshade/finfocus-spec/issues/353)


### Fixed

* **pluginsdk:** harden pagination edge cases and observability ([#376](https://github.com/rshade/finfocus-spec/issues/376)) ([60082d8](https://github.com/rshade/finfocus-spec/commit/60082d843e2fb4d4b9dd7d729daadc561c10d32c)), closes [#364](https://github.com/rshade/finfocus-spec/issues/364) [#367](https://github.com/rshade/finfocus-spec/issues/367) [#368](https://github.com/rshade/finfocus-spec/issues/368) [#369](https://github.com/rshade/finfocus-spec/issues/369)

## [0.5.5](https://github.com/rshade/finfocus-spec/compare/v0.5.4...v0.5.5) (2026-01-29)


### Added

* **pricing:** add validation bypass metadata for audit trails ([#338](https://github.com/rshade/finfocus-spec/issues/338)) ([5abd98c](https://github.com/rshade/finfocus-spec/commit/5abd98c812868566b63a58eac89593bd83746091))
* **proto:** add pricing tier and risk score fields ([#335](https://github.com/rshade/finfocus-spec/issues/335)) ([890f230](https://github.com/rshade/finfocus-spec/commit/890f230a77cb598066ce187e85fcf87e1ebda8b8)), closes [#217](https://github.com/rshade/finfocus-spec/issues/217)
* **proto:** add usage profile enum for workload context signaling ([#349](https://github.com/rshade/finfocus-spec/issues/349)) ([ae82f64](https://github.com/rshade/finfocus-spec/commit/ae82f647590dde04d18a610c3bd3297bef7979c3))
* **sdk:** implement github issues audit plan for [#247](https://github.com/rshade/finfocus-spec/issues/247)-340 ([#341](https://github.com/rshade/finfocus-spec/issues/341)) ([8b5b97d](https://github.com/rshade/finfocus-spec/commit/8b5b97d903ccd1e1f95e975a41e34b2340e7341f))


### Documentation

* **sdk:** fix documentation drift across readme and sdk packages ([#352](https://github.com/rshade/finfocus-spec/issues/352)) ([7e759b4](https://github.com/rshade/finfocus-spec/commit/7e759b423d03f2b702b978716778a4e889c8b75c)), closes [#346](https://github.com/rshade/finfocus-spec/issues/346) [#347](https://github.com/rshade/finfocus-spec/issues/347) [#348](https://github.com/rshade/finfocus-spec/issues/348)

## [0.5.4](https://github.com/rshade/finfocus-spec/compare/v0.5.3...v0.5.4) (2026-01-21)


### Added

* **proto:** add anomaly category and investigate action ([#332](https://github.com/rshade/finfocus-spec/issues/332)) ([318eeb9](https://github.com/rshade/finfocus-spec/commit/318eeb98ff70d2c560d467d59538a3dcd96b6ef9)), closes [#315](https://github.com/rshade/finfocus-spec/issues/315)


### Fixed

* update release workflow for typescript package ([#317](https://github.com/rshade/finfocus-spec/issues/317)) ([d7cdcb5](https://github.com/rshade/finfocus-spec/commit/d7cdcb54c0649208fe386b8105bb42e66065fbd6))


### Documentation

* adding initial typescript sdk readme ([#333](https://github.com/rshade/finfocus-spec/issues/333)) ([b79343d](https://github.com/rshade/finfocus-spec/commit/b79343dcd7eccfd658622f504bdc310299f1e683))

## [0.5.3](https://github.com/rshade/finfocus-spec/compare/v0.5.2...v0.5.3) (2026-01-19)


### Added

* **ci:** implement automated typescript sdk publishing ([#312](https://github.com/rshade/finfocus-spec/issues/312)) ([a12d218](https://github.com/rshade/finfocus-spec/commit/a12d21815c3daa3dafe6cd2db259c230bb34a89f)), closes [#311](https://github.com/rshade/finfocus-spec/issues/311)
* **sdk:** polish capability discovery and optimize performance ([#310](https://github.com/rshade/finfocus-spec/issues/310)) ([f74c8c4](https://github.com/rshade/finfocus-spec/commit/f74c8c4d59b2801ca831ed95e9cc5bf15865b1f4)), closes [#294](https://github.com/rshade/finfocus-spec/issues/294) [#295](https://github.com/rshade/finfocus-spec/issues/295) [#299](https://github.com/rshade/finfocus-spec/issues/299) [#300](https://github.com/rshade/finfocus-spec/issues/300) [#301](https://github.com/rshade/finfocus-spec/issues/301) [#208](https://github.com/rshade/finfocus-spec/issues/208) [#209](https://github.com/rshade/finfocus-spec/issues/209)


### Fixed

* **pluginsdk:** improve capability discovery robustness ([#306](https://github.com/rshade/finfocus-spec/issues/306)) ([873201e](https://github.com/rshade/finfocus-spec/commit/873201eee932ef219efaaadbeca9a92f87e6015c)), closes [#296](https://github.com/rshade/finfocus-spec/issues/296) [#297](https://github.com/rshade/finfocus-spec/issues/297)

## [0.5.2](https://github.com/rshade/finfocus-spec/compare/v0.5.1...v0.5.2) (2026-01-18)


### Added

* **proto:** add standardized reasoning metadata ([#288](https://github.com/rshade/finfocus-spec/issues/288)) ([20f3b14](https://github.com/rshade/finfocus-spec/commit/20f3b146609b84860d480fcc63a7de1ccc2b348e)), closes [#188](https://github.com/rshade/finfocus-spec/issues/188)
* **sdk:** add typescript sdk for browser and node.js ([#302](https://github.com/rshade/finfocus-spec/issues/302)) ([f819c3e](https://github.com/rshade/finfocus-spec/commit/f819c3efa695c65ce4f74e18bdcf2260d56455b3))
* **sdk:** implement granular capability discovery in supports rpc ([#291](https://github.com/rshade/finfocus-spec/issues/291)) ([fea8874](https://github.com/rshade/finfocus-spec/commit/fea88745db9a714b7e1c6457090d12690df8e14d)), closes [#194](https://github.com/rshade/finfocus-spec/issues/194)
* **sdk:** implement granular capability discovery in supports rpc ([#292](https://github.com/rshade/finfocus-spec/issues/292)) ([886cd51](https://github.com/rshade/finfocus-spec/commit/886cd51a54938f9a2d4edf466d204c7c0901b37c)), closes [#194](https://github.com/rshade/finfocus-spec/issues/194)
* **sdk:** implement plugin capability discovery and auto-detection ([#290](https://github.com/rshade/finfocus-spec/issues/290)) ([ee3ed6b](https://github.com/rshade/finfocus-spec/commit/ee3ed6bdd6d4d479c236ba04c10ab258ac44430d)), closes [#287](https://github.com/rshade/finfocus-spec/issues/287)


### Fixed

* **sdk:** complete typescript sdk builder api and improve error handling ([#303](https://github.com/rshade/finfocus-spec/issues/303)) ([4804ded](https://github.com/rshade/finfocus-spec/commit/4804ded9252e01e5b2d6eb934909c08e7ed5cb26))
* **sdk:** complete typescript sdk builder api and improve error handling ([#303](https://github.com/rshade/finfocus-spec/issues/303)) ([#305](https://github.com/rshade/finfocus-spec/issues/305)) ([a1ef832](https://github.com/rshade/finfocus-spec/commit/a1ef8327887376bc39328a288afc2187dad88609))

## [0.5.1](https://github.com/rshade/finfocus-spec/compare/v0.5.0...v0.5.1) (2026-01-13)


### Changed

* **main:** finishing up finfocus polish ([#280](https://github.com/rshade/finfocus-spec/issues/280)) ([2c3827e](https://github.com/rshade/finfocus-spec/commit/2c3827eb559c2e2b7bd3cdcafe16e2a673c544c4))


### Documentation

* add pulumicost to finfocus migration guide ([#286](https://github.com/rshade/finfocus-spec/issues/286)) ([6ed0a71](https://github.com/rshade/finfocus-spec/commit/6ed0a717debc4b64d545133a0e155c855c51f4cd)), closes [#282](https://github.com/rshade/finfocus-spec/issues/282)

## [0.5.0](https://github.com/rshade/finfocus-spec/compare/v0.4.14...v0.5.0) (2026-01-12)

### Changed

- **sdk:** rename project from pulumicost to finfocus ([#273](https://github.com/rshade/finfocus-spec/issues/273)) ([eecdddb](https://github.com/rshade/finfocus-spec/commit/eecdddbdf8ea901f15c46658e8315dadbd4e58a0)), closes [#272](https://github.com/rshade/finfocus-spec/issues/272)

### Migration Guide (PulumiCost -> FinFocus)

This release renames the project to **FinFocus**. Breaking changes include:

- **Environment Variables:**
  - `PULUMICOST_PLUGIN_PORT` → `FINFOCUS_PLUGIN_PORT`
  - `PULUMICOST_LOG_LEVEL` → `FINFOCUS_LOG_LEVEL`
  - `PULUMICOST_LOG_FILE` → `FINFOCUS_LOG_FILE`
- **Plugin Path:** `~/.pulumicost/plugins/` → `~/.finfocus/plugins/`

See [MIGRATION.md](MIGRATION.md) for full details.

### Chores

- release 0.5.0 ([4358f2a](https://github.com/rshade/finfocus-spec/commit/4358f2a060220fc9876d898ecd20136fabb5e3eb))

## [0.4.14](https://github.com/rshade/finfocus-spec/compare/v0.4.13...v0.4.14) (2026-01-10)

### Added

- **pluginsdk:** implement sdk polish features and hardening ([#259](https://github.com/rshade/finfocus-spec/issues/259)) ([2091d19](https://github.com/rshade/finfocus-spec/commit/2091d19cc0e84b46bd14469202e9d896e6fa3cc9))

### Fixed

- **pluginsdk:** use zerolog in health handler ([#268](https://github.com/rshade/finfocus-spec/issues/268)) ([b43443e](https://github.com/rshade/finfocus-spec/commit/b43443e8afedd7bdf69f19b81de0c852c35739f1)), closes [#266](https://github.com/rshade/finfocus-spec/issues/266)

### Changed

- **sdk:** Improve ARN provider type safety ([#269](https://github.com/rshade/finfocus-spec/issues/269)) ([813d98e](https://github.com/rshade/finfocus-spec/commit/813d98e55e55d4efab4ea4802c9ad04dd7ebda85)), closes [#203](https://github.com/rshade/finfocus-spec/issues/203)
- **sdk:** Improve ARN provider type safety ([#270](https://github.com/rshade/finfocus-spec/issues/270)) ([61ea613](https://github.com/rshade/finfocus-spec/commit/61ea6136d9223fb188e862c7d8aff2adda6fd40c)), closes [#203](https://github.com/rshade/finfocus-spec/issues/203)

## [0.4.13](https://github.com/rshade/finfocus-spec/compare/v0.4.12...v0.4.13) (2026-01-05)

### Added

- **pluginsdk:** Add CORS headers, max-age, and security headers ([#256](https://github.com/rshade/finfocus-spec/issues/256)) ([93555f6](https://github.com/rshade/finfocus-spec/commit/93555f6003b3a32b645f36c42bd740f35fef9e97)), closes [#228](https://github.com/rshade/finfocus-spec/issues/228) [#229](https://github.com/rshade/finfocus-spec/issues/229) [#239](https://github.com/rshade/finfocus-spec/issues/239)
- **pluginsdk:** Add DryRun for plugin field mapping discovery ([#248](https://github.com/rshade/finfocus-spec/issues/248)) ([45cea69](https://github.com/rshade/finfocus-spec/commit/45cea69ca004e4d08a0fb9c08a1ea62422c0ec9f)), closes [#186](https://github.com/rshade/finfocus-spec/issues/186)
- **proto:** add growth_type to projected cost response ([#250](https://github.com/rshade/finfocus-spec/issues/250)) ([4ba1da4](https://github.com/rshade/finfocus-spec/commit/4ba1da43a642939f1bceb42c8bf2ebd14b7d327d)), closes [#249](https://github.com/rshade/finfocus-spec/issues/249)
- **sdk:** add jsonld serialization package for focus cost data ([#252](https://github.com/rshade/finfocus-spec/issues/252)) ([6760501](https://github.com/rshade/finfocus-spec/commit/676050119a3605598286238e3f5a8a0b25a6e374)), closes [#187](https://github.com/rshade/finfocus-spec/issues/187)

## [0.4.12](https://github.com/rshade/finfocus-spec/compare/v0.4.11...v0.4.12) (2025-12-31)

### Added

- **pluginsdk:** Add GetPluginInfo RPC for spec version compatibility ([#242](https://github.com/rshade/finfocus-spec/issues/242)) ([d8f4c53](https://github.com/rshade/finfocus-spec/commit/d8f4c539a3f778e323263a531dd08ade274f350d)), closes [#222](https://github.com/rshade/finfocus-spec/issues/222)
- **pluginsdk:** add multi-protocol support with connect-go ([#223](https://github.com/rshade/finfocus-spec/issues/223)) ([26f7549](https://github.com/rshade/finfocus-spec/commit/26f7549da00515932949d8b2503d4ce5166684eb)), closes [#189](https://github.com/rshade/finfocus-spec/issues/189)
- **proto:** add forecasting primitives for cost projections ([#241](https://github.com/rshade/finfocus-spec/issues/241)) ([0e2ab7c](https://github.com/rshade/finfocus-spec/commit/0e2ab7cc84689ae0bf2677e48c8e8e3a788250ed)), closes [#215](https://github.com/rshade/finfocus-spec/issues/215)

### Documentation

- **pluginsdk:** Add thread safety, rate limiting, CORS, and perf docs ([#243](https://github.com/rshade/finfocus-spec/issues/243)) ([91b2ae1](https://github.com/rshade/finfocus-spec/commit/91b2ae1cad4d5209dbfe6236d3dc0ff353e161b0)), closes [#206](https://github.com/rshade/finfocus-spec/issues/206) [#207](https://github.com/rshade/finfocus-spec/issues/207) [#211](https://github.com/rshade/finfocus-spec/issues/211) [#231](https://github.com/rshade/finfocus-spec/issues/231) [#233](https://github.com/rshade/finfocus-spec/issues/233) [#235](https://github.com/rshade/finfocus-spec/issues/235) [#236](https://github.com/rshade/finfocus-spec/issues/236) [#237](https://github.com/rshade/finfocus-spec/issues/237) [#238](https://github.com/rshade/finfocus-spec/issues/238) [#240](https://github.com/rshade/finfocus-spec/issues/240)
- **sdk:** add advanced implementation patterns and examples ([#213](https://github.com/rshade/finfocus-spec/issues/213)) ([922422c](https://github.com/rshade/finfocus-spec/commit/922422c6ccb2d65ff4bfd503c8e543ff3f29dc6a)), closes [#185](https://github.com/rshade/finfocus-spec/issues/185)

## [0.4.11](https://github.com/rshade/finfocus-spec/compare/v0.4.10...v0.4.11) (2025-12-26)

### Added

- **pluginsdk:** Enable gRPC server reflection by default ([#181](https://github.com/rshade/finfocus-spec/issues/181)) ([c058c6e](https://github.com/rshade/finfocus-spec/commit/c058c6e57f2f8e0c983ebe092f6084fc3b3f513a))
- **pluginsdk:** implement contextual finops validation ([#201](https://github.com/rshade/finfocus-spec/issues/201)) ([4a9b808](https://github.com/rshade/finfocus-spec/commit/4a9b80805180b261708a87d02ea1fec41b371d7b)), closes [#184](https://github.com/rshade/finfocus-spec/issues/184)
- **proto:** add focus 1.3 columns and contract commitment dataset ([#199](https://github.com/rshade/finfocus-spec/issues/199)) ([25fcf65](https://github.com/rshade/finfocus-spec/commit/25fcf65591c96435fa3e71e159e42b0f43d5cd6d)), closes [#183](https://github.com/rshade/finfocus-spec/issues/183)
- **proto:** Add id and arn fields to ResourceDescriptor ([#202](https://github.com/rshade/finfocus-spec/issues/202)) ([962db4f](https://github.com/rshade/finfocus-spec/commit/962db4f17621d134eac1a64f1675574ab47b2859)), closes [#200](https://github.com/rshade/finfocus-spec/issues/200)

## [0.4.10](https://github.com/rshade/finfocus-spec/compare/v0.4.9...v0.4.10) (2025-12-19)

### Added

- **proto:** add greenops metrics and utilization modeling ([#176](https://github.com/rshade/finfocus-spec/issues/176)) ([d00dc45](https://github.com/rshade/finfocus-spec/commit/d00dc4504c3fc88cc2a7c21ae294db0ee339116c))

## [0.4.9](https://github.com/rshade/finfocus-spec/compare/v0.4.8...v0.4.9) (2025-12-18)

### Added

- **proto:** add target_resources for resource-scoped recs ([#171](https://github.com/rshade/finfocus-spec/issues/171)) ([4526eb7](https://github.com/rshade/finfocus-spec/commit/4526eb70d93b9e7e05e6d06519c75bd44a80da00))
- **proto:** extend recommendation action types for cost optimization ([#173](https://github.com/rshade/finfocus-spec/issues/173)) ([4abebd1](https://github.com/rshade/finfocus-spec/commit/4abebd1cfabea315a150065a65aca53d3291c6c0)), closes [#170](https://github.com/rshade/finfocus-spec/issues/170)

### Fixed

- fixing test issues and mockplugin ([#172](https://github.com/rshade/finfocus-spec/issues/172)) ([fa0b641](https://github.com/rshade/finfocus-spec/commit/fa0b641e23df8e80652a88a8e1fea513a8e16de8))
- **sdk:** correct strict weak ordering violation in sort recommendations ([#168](https://github.com/rshade/finfocus-spec/issues/168)) ([0860d38](https://github.com/rshade/finfocus-spec/commit/0860d3841b5adb69871f99debe1454bf662820e6)), closes [#167](https://github.com/rshade/finfocus-spec/issues/167)

## [0.4.8](https://github.com/rshade/finfocus-spec/compare/v0.4.7...v0.4.8) (2025-12-16)

### Added

- **proto:** add comprehensive filter and dismissal to recommendations ([#166](https://github.com/rshade/finfocus-spec/issues/166)) ([71240b8](https://github.com/rshade/finfocus-spec/commit/71240b852b590dee01c8063f613ef5dc4ece62e9)), closes [#165](https://github.com/rshade/finfocus-spec/issues/165)

### Documentation

- updating the documentation for the latest changes ([2589c1a](https://github.com/rshade/finfocus-spec/commit/2589c1aff4e06f576d2617ed4b38d7222cca60a4))
- updating the documentation for the latest changes ([#164](https://github.com/rshade/finfocus-spec/issues/164)) ([b54b806](https://github.com/rshade/finfocus-spec/commit/b54b806b27e554fdd384f23a93ca328128eb5b3e))

## [0.4.7](https://github.com/rshade/finfocus-spec/compare/v0.4.6...v0.4.7) (2025-12-15)

### Added

- **proto:** add arn field to actual cost request ([#160](https://github.com/rshade/finfocus-spec/issues/160)) ([a75b42b](https://github.com/rshade/finfocus-spec/commit/a75b42b355395f590b7d211475de05f5083bd82b)), closes [#157](https://github.com/rshade/finfocus-spec/issues/157)

## [0.4.6](https://github.com/rshade/finfocus-spec/compare/v0.4.5...v0.4.6) (2025-12-11)

### Added

- **pluginsdk:** add request validation helpers ([#151](https://github.com/rshade/finfocus-spec/issues/151)) ([ef71ba6](https://github.com/rshade/finfocus-spec/commit/ef71ba6018169b630f3068deab45b97bd1e3522c)), closes [#130](https://github.com/rshade/finfocus-spec/issues/130)

### Fixed

- updating small bugs in spec ([#156](https://github.com/rshade/finfocus-spec/issues/156)) ([83df05f](https://github.com/rshade/finfocus-spec/commit/83df05f11ec2690c9b4e594128a9ba4022c20c8b))

## [0.4.5](https://github.com/rshade/finfocus-spec/compare/v0.4.4...v0.4.5) (2025-12-10)

### Added

- **pluginsdk:** add mapping package for property extraction ([#148](https://github.com/rshade/finfocus-spec/issues/148)) ([8fd1524](https://github.com/rshade/finfocus-spec/commit/8fd1524877218272a7d219239058bfee43c294bc)), closes [#128](https://github.com/rshade/finfocus-spec/issues/128)
- **proto:** add getbudgets rpc for unified budget visibility ([#149](https://github.com/rshade/finfocus-spec/issues/149)) ([b4018d7](https://github.com/rshade/finfocus-spec/commit/b4018d794fd5b2c2f54541102c7625ffd900f26f)), closes [#123](https://github.com/rshade/finfocus-spec/issues/123)
- **sdk:** Support PULUMICOST_LOG_FILE for unified logging ([#145](https://github.com/rshade/finfocus-spec/issues/145)) ([6b9a9b3](https://github.com/rshade/finfocus-spec/commit/6b9a9b38e24f9b49a05095c140bd2500c4e8090b)), closes [#131](https://github.com/rshade/finfocus-spec/issues/131)

### Documentation

- Document pluginsdk.Serve() behavior and configuration ([#146](https://github.com/rshade/finfocus-spec/issues/146)) ([30687f9](https://github.com/rshade/finfocus-spec/commit/30687f9ca34a1c262a0ac6b8f66f98301dce1987))
- **sdk:** add core-plugin interface docs and contract tests ([#150](https://github.com/rshade/finfocus-spec/issues/150)) ([87d4428](https://github.com/rshade/finfocus-spec/commit/87d44289a5a63b25d8baeccb11f7eb9f56ba3128)), closes [#132](https://github.com/rshade/finfocus-spec/issues/132) [#133](https://github.com/rshade/finfocus-spec/issues/133) [#134](https://github.com/rshade/finfocus-spec/issues/134) [#135](https://github.com/rshade/finfocus-spec/issues/135)

## [0.4.4](https://github.com/rshade/finfocus-spec/compare/v0.4.3...v0.4.4) (2025-12-09)

### Added

- **pluginsdk:** add --port flag parsing for multi-plugin orchestration ([#143](https://github.com/rshade/finfocus-spec/issues/143)) ([c0b0528](https://github.com/rshade/finfocus-spec/commit/c0b05288e69dc70ad0105165572a5fa3714ed27f)), closes [#129](https://github.com/rshade/finfocus-spec/issues/129) [#137](https://github.com/rshade/finfocus-spec/issues/137)
- **pluginsdk:** add fallback hint enum for plugin orchestration ([#126](https://github.com/rshade/finfocus-spec/issues/126)) ([ef7aab0](https://github.com/rshade/finfocus-spec/commit/ef7aab0576e3b4815c4d273c33800557734ebb37)), closes [#124](https://github.com/rshade/finfocus-spec/issues/124)
- **pluginsdk:** centralize environment variable handling ([#139](https://github.com/rshade/finfocus-spec/issues/139)) ([4c9e279](https://github.com/rshade/finfocus-spec/commit/4c9e279ad38c58ee28178f61041c387c512654ca)), closes [#127](https://github.com/rshade/finfocus-spec/issues/127)
- **proto:** add getbudgets rpc for unified budget visibility across providers ([#145](https://github.com/rshade/finfocus-spec/issues/145)) ([abc123d](https://github.com/rshade/finfocus-spec/commit/abc123def456ghi789jkl012))
- **proto:** add getrecommendations rpc for finops optimization ([#125](https://github.com/rshade/finfocus-spec/issues/125)) ([ecf92f0](https://github.com/rshade/finfocus-spec/commit/ecf92f0af6c1dbd1d036e92a1e999a7576debaef))

### Fixed

- adding in edge case tests, and benchmark ([#142](https://github.com/rshade/finfocus-spec/issues/142)) ([881132b](https://github.com/rshade/finfocus-spec/commit/881132bd87d1bb69ecd9f3abff01527da16fc08f))

### Documentation

- adding in claude speckit ([#144](https://github.com/rshade/finfocus-spec/issues/144)) ([70a6e78](https://github.com/rshade/finfocus-spec/commit/70a6e78fffba6ac81c518a4225a4da56a34aafdf))

## [0.4.3](https://github.com/rshade/finfocus-spec/compare/v0.4.2...v0.4.3) (2025-12-03)

### Added

- **ci:** Add Lefthook git hooks with commitlint validation ([#120](https://github.com/rshade/finfocus-spec/issues/120)) ([afdf8f7](https://github.com/rshade/finfocus-spec/commit/afdf8f78afb2cac5dfdad95359acb2871c727be7)), closes [#55](https://github.com/rshade/finfocus-spec/issues/55)
- **pluginsdk:** Add conformance testing support for Plugin interface ([#118](https://github.com/rshade/finfocus-spec/issues/118)) ([8df49c0](https://github.com/rshade/finfocus-spec/commit/8df49c041adc671843aa0fa6bda987c50a5bcc7a)), closes [#98](https://github.com/rshade/finfocus-spec/issues/98)
- **pluginsdk:** add Prometheus metrics instrumentation for plugins ([#119](https://github.com/rshade/finfocus-spec/issues/119)) ([9365aef](https://github.com/rshade/finfocus-spec/commit/9365aef0636144f1bcf0695db68681823a889fe0)), closes [#80](https://github.com/rshade/finfocus-spec/issues/80)
- **sdk/go/currency:** extract ISO 4217 validation as reusable package (T101) ([#116](https://github.com/rshade/finfocus-spec/issues/116)) ([97e34f5](https://github.com/rshade/finfocus-spec/commit/97e34f5fba0f1a635ab9651ed2bc80510898b962)), closes [#101](https://github.com/rshade/finfocus-spec/issues/101)

## [0.4.2](https://github.com/rshade/finfocus-spec/compare/v0.4.1...v0.4.2) (2025-11-30)

### Added

- **ci:** add performance regression testing workflow ([8944316](https://github.com/rshade/finfocus-spec/commit/8944316a5337a12652efaa700999b7fd400517de))
- run concurrent benchmark for EstimateCost ([#113](https://github.com/rshade/finfocus-spec/issues/113)) ([0ffcdc4](https://github.com/rshade/finfocus-spec/commit/0ffcdc48e132b1eece31a1c51280cd250c608c23))
- **testing:** add distributed tracing example for EstimateCost (T042) ([#112](https://github.com/rshade/finfocus-spec/issues/112)) ([b14dd3c](https://github.com/rshade/finfocus-spec/commit/b14dd3c2eface44181d13d5add225eff57f53198)), closes [#85](https://github.com/rshade/finfocus-spec/issues/85)
- **testing:** add metrics tracking example for EstimateCost (T041) ([#111](https://github.com/rshade/finfocus-spec/issues/111)) ([944e078](https://github.com/rshade/finfocus-spec/commit/944e0789a67be8e204d92d0d7a35ba181f9dd854)), closes [#84](https://github.com/rshade/finfocus-spec/issues/84)
- **testing:** implement Plugin Conformance Test Suite ([#109](https://github.com/rshade/finfocus-spec/issues/109)) ([03116ce](https://github.com/rshade/finfocus-spec/commit/03116cef17567bdea85ba59e87aca322d2c42efb))

### Documentation

- **006-estimate-cost:** update data-model.md with actual decimal type (T054) ([#114](https://github.com/rshade/finfocus-spec/issues/114)) ([45f4b2e](https://github.com/rshade/finfocus-spec/commit/45f4b2e7c2a37c9414aada68343731a2f0e7913c)), closes [#89](https://github.com/rshade/finfocus-spec/issues/89)

## [0.4.1](https://github.com/rshade/finfocus-spec/compare/v0.4.0...v0.4.1) (2025-11-29)

### Added

- add trace ID validation to TracingUnaryServerInterceptor ([#96](https://github.com/rshade/finfocus-spec/issues/96)) ([dd410cd](https://github.com/rshade/finfocus-spec/commit/dd410cdbc2ca88ecc87dc3bef3e4fa3488efd714)), closes [#94](https://github.com/rshade/finfocus-spec/issues/94)
- **focus:** add complete FOCUS 1.2 column coverage with builder API ([#100](https://github.com/rshade/finfocus-spec/issues/100)) ([2355acf](https://github.com/rshade/finfocus-spec/commit/2355acf206f5d2ed70d9cd47fd9033525f8fe552))
- **focus:** Implement FOCUS 1.2 integration ([#99](https://github.com/rshade/finfocus-spec/issues/99)) ([913b6ef](https://github.com/rshade/finfocus-spec/commit/913b6ef9d9a9ca277058ded46dcf5f7cfadc7aab))
- **sdk:** migrate pluginsdk from core to spec ([#97](https://github.com/rshade/finfocus-spec/issues/97)) ([2e35cbf](https://github.com/rshade/finfocus-spec/commit/2e35cbf548f91f0151901795227dc07d578f3220))
- **testing:** add structured logging example for EstimateCost RPC ([#93](https://github.com/rshade/finfocus-spec/issues/93)) ([4c583c0](https://github.com/rshade/finfocus-spec/commit/4c583c0fc0cc179b57fb919eef6ba349d4cf7187)), closes [#83](https://github.com/rshade/finfocus-spec/issues/83)

## [Unreleased]

### Added

- **ci:** add performance regression tests with benchmark comparison
- **docs:** add EstimateCost cross-provider coverage matrix to examples/README.md
- **sdk:** add trace ID validation to TracingUnaryServerInterceptor for security ([#94](https://github.com/rshade/finfocus-spec/issues/94))

### Security

- **sdk:** prevent log injection attacks through malformed trace IDs by validating and replacing invalid values

## [0.4.0](https://github.com/rshade/finfocus-spec/compare/v0.3.0...v0.4.0) (2025-11-26)

### Added

- **rpc:** implement EstimateCost RPC for what-if cost analysis ([#90](https://github.com/rshade/finfocus-spec/issues/90)) ([d6f3c95](https://github.com/rshade/finfocus-spec/commit/d6f3c9566da8d28550923edfe4ffe34d3c143e0e)), closes [#79](https://github.com/rshade/finfocus-spec/issues/79)

## [0.3.0](https://github.com/rshade/finfocus-spec/compare/v0.2.0...v0.3.0) (2025-11-24)

### Added

- **sdk:** add zerolog logging utilities for plugin standardization ([#76](https://github.com/rshade/finfocus-spec/issues/76)) ([6d5b5ac](https://github.com/rshade/finfocus-spec/commit/6d5b5ac06329dce03a99b595e41d5ca1273b7c40)), closes [#75](https://github.com/rshade/finfocus-spec/issues/75)

### Documentation

- udpate sdk/go/registry/CLAUDE.md for enums ([#78](https://github.com/rshade/finfocus-spec/issues/78)) ([f15ef76](https://github.com/rshade/finfocus-spec/commit/f15ef769ac491056504f7a0376b413727f7969e0)), closes [#3](https://github.com/rshade/finfocus-spec/issues/3)

## [0.2.0](https://github.com/rshade/finfocus-spec/compare/v0.1.0...v0.2.0) (2025-11-24)

### ⚠ BREAKING CHANGES

- **proto:** None - 100% backward compatible (additive proto changes only)
- **registry:** None - 100% backward compatible

### Added

- **proto:** enhance GetPricingSpec with transparent pricing breakdown ([#67](https://github.com/rshade/finfocus-spec/issues/67)) ([336144e](https://github.com/rshade/finfocus-spec/commit/336144e45e2334a677af4a0f5ccb3994126cf22a)), closes [#62](https://github.com/rshade/finfocus-spec/issues/62)
- **schema:** add plugin registry index JSON Schema ([#70](https://github.com/rshade/finfocus-spec/issues/70)) ([79938a7](https://github.com/rshade/finfocus-spec/commit/79938a7fc473cd997465f3a71c734b0b2b3b692b)), closes [#68](https://github.com/rshade/finfocus-spec/issues/68)

### Fixed

- **release:** remove release-as constraint to allow version bumps ([#73](https://github.com/rshade/finfocus-spec/issues/73)) ([706ec65](https://github.com/rshade/finfocus-spec/commit/706ec65dbe779242674aae94c6bc89de0f8c252a))

### Performance

- **registry:** optimize enum validation for zero-allocation performance ([#63](https://github.com/rshade/finfocus-spec/issues/63)) ([6d3c124](https://github.com/rshade/finfocus-spec/commit/6d3c124b4230485ee27288051181a965a31daf50)), closes [#33](https://github.com/rshade/finfocus-spec/issues/33)

### Documentation

- **spec:** document Supports() RPC verification for issue [#64](https://github.com/rshade/finfocus-spec/issues/64) ([#66](https://github.com/rshade/finfocus-spec/issues/66)) ([3a17c4f](https://github.com/rshade/finfocus-spec/commit/3a17c4f516488d033549c26c4bb9ad8ed957e3b0))

## [0.1.0](https://github.com/rshade/finfocus-spec/compare/v0.1.0...v0.1.0) (2025-11-24)

### ⚠ BREAKING CHANGES

- **proto:** None - 100% backward compatible (additive proto changes only)
- **registry:** None - 100% backward compatible

### Added

- **proto:** enhance GetPricingSpec with transparent pricing breakdown ([#67](https://github.com/rshade/finfocus-spec/issues/67)) ([336144e](https://github.com/rshade/finfocus-spec/commit/336144e45e2334a677af4a0f5ccb3994126cf22a)), closes [#62](https://github.com/rshade/finfocus-spec/issues/62)
- **schema:** add plugin registry index JSON Schema ([#70](https://github.com/rshade/finfocus-spec/issues/70)) ([79938a7](https://github.com/rshade/finfocus-spec/commit/79938a7fc473cd997465f3a71c734b0b2b3b692b)), closes [#68](https://github.com/rshade/finfocus-spec/issues/68)

### Performance

- **registry:** optimize enum validation for zero-allocation performance ([#63](https://github.com/rshade/finfocus-spec/issues/63)) ([6d3c124](https://github.com/rshade/finfocus-spec/commit/6d3c124b4230485ee27288051181a965a31daf50)), closes [#33](https://github.com/rshade/finfocus-spec/issues/33)

### Documentation

- **spec:** document Supports() RPC verification for issue [#64](https://github.com/rshade/finfocus-spec/issues/64) ([#66](https://github.com/rshade/finfocus-spec/issues/66)) ([3a17c4f](https://github.com/rshade/finfocus-spec/commit/3a17c4f516488d033549c26c4bb9ad8ed957e3b0))

## [Unreleased]

### Added

- **Schema**: Add plugin registry index JSON Schema (`schemas/plugin_registry.schema.json`)
  - Validates registry.json files for `pulumicost plugin install` discovery
  - Aligns with registry.proto definitions (SecurityLevel, capabilities, providers)
  - Includes `dependentRequired` for deprecation_message when deprecated is true
  - npm validation scripts: `validate:registry`, `validate:registry-schema`
  - Example registry with kubecost and aws-public plugins
  - Closes [#68](https://github.com/rshade/finfocus-spec/issues/68)

### Changed

- **Performance**: Optimized registry package enum validation for zero-allocation performance
  - Converted all 8 enum types (Provider, DiscoverySource, PluginStatus, SecurityLevel, InstallationMethod,
    PluginCapability, SystemPermission, AuthMethod) from function-returned slices to package-level variables
  - Achieved 0 B/op, 0 allocs/op across all validation functions (previously 1 alloc/op)
  - Performance improved to 5-12 ns/op (2x faster than map-based alternatives)
  - Memory footprint reduced to ~608 bytes total for all enums (vs ~3.5 KB for maps)
  - Established validation pattern for future SDK enums (see `specs/001-domain-enum-optimization/`)

## [0.1.0](https://github.com/rshade/finfocus-spec/compare/v0.1.0...v0.1.0) (2025-11-18)

### Added

- Add comprehensive Plugin Registry Specification ([#29](https://github.com/rshade/finfocus-spec/issues/29)) ([5825eaa](https://github.com/rshade/finfocus-spec/commit/5825eaab1f343b1f9fcb966c38effb6a743d3494)), closes [#8](https://github.com/rshade/finfocus-spec/issues/8)
- comprehensive testing framework and enterprise CI/CD pipeline ([#19](https://github.com/rshade/finfocus-spec/issues/19)) ([3a235ef](https://github.com/rshade/finfocus-spec/commit/3a235eff0ce4a172a7b920326066227540c6c8b8))
- enhance Provider enum with String() method and improved error m… ([#39](https://github.com/rshade/finfocus-spec/issues/39)) ([acbaf0c](https://github.com/rshade/finfocus-spec/commit/acbaf0c6db29d986211d3ca8ef19a66e3162e2c2)), closes [#4](https://github.com/rshade/finfocus-spec/issues/4)
- freeze costsource.proto v0.1.0 specification ([#17](https://github.com/rshade/finfocus-spec/issues/17)) ([3b485b9](https://github.com/rshade/finfocus-spec/commit/3b485b96fef7dc2992c166980f324907e4ff06bd)), closes [#3](https://github.com/rshade/finfocus-spec/issues/3)
- freeze costsource.proto v0.1.0 specification ([#18](https://github.com/rshade/finfocus-spec/issues/18)) ([a085bd2](https://github.com/rshade/finfocus-spec/commit/a085bd202e266189efba92a1832fa8f90c0931e6)), closes [#3](https://github.com/rshade/finfocus-spec/issues/3)

### Documentation

- add comprehensive plugin developer guide ([#16](https://github.com/rshade/finfocus-spec/issues/16)) ([b0a5eb3](https://github.com/rshade/finfocus-spec/commit/b0a5eb3396b5c49839d742234666667e5e7b1ee7)), closes [#2](https://github.com/rshade/finfocus-spec/issues/2)
- establish constitution v1.0.0 for gRPC proto specification governance ([#57](https://github.com/rshade/finfocus-spec/issues/57)) ([54578aa](https://github.com/rshade/finfocus-spec/commit/54578aa259cb8907f989196ce2b73e37e57f906f))

## [0.1.0](https://github.com/rshade/finfocus-spec/compare/v0.1.0...v0.1.0) (2025-11-18)

### Added

- Add comprehensive Plugin Registry Specification ([#29](https://github.com/rshade/finfocus-spec/issues/29)) ([5825eaa](https://github.com/rshade/finfocus-spec/commit/5825eaab1f343b1f9fcb966c38effb6a743d3494)), closes [#8](https://github.com/rshade/finfocus-spec/issues/8)
- comprehensive testing framework and enterprise CI/CD pipeline ([#19](https://github.com/rshade/finfocus-spec/issues/19)) ([3a235ef](https://github.com/rshade/finfocus-spec/commit/3a235eff0ce4a172a7b920326066227540c6c8b8))
- enhance Provider enum with String() method and improved error m… ([#39](https://github.com/rshade/finfocus-spec/issues/39)) ([acbaf0c](https://github.com/rshade/finfocus-spec/commit/acbaf0c6db29d986211d3ca8ef19a66e3162e2c2)), closes [#4](https://github.com/rshade/finfocus-spec/issues/4)
- freeze costsource.proto v0.1.0 specification ([#17](https://github.com/rshade/finfocus-spec/issues/17)) ([3b485b9](https://github.com/rshade/finfocus-spec/commit/3b485b96fef7dc2992c166980f324907e4ff06bd)), closes [#3](https://github.com/rshade/finfocus-spec/issues/3)
- freeze costsource.proto v0.1.0 specification ([#18](https://github.com/rshade/finfocus-spec/issues/18)) ([a085bd2](https://github.com/rshade/finfocus-spec/commit/a085bd202e266189efba92a1832fa8f90c0931e6)), closes [#3](https://github.com/rshade/finfocus-spec/issues/3)

### Documentation

- add comprehensive plugin developer guide ([#16](https://github.com/rshade/finfocus-spec/issues/16)) ([b0a5eb3](https://github.com/rshade/finfocus-spec/commit/b0a5eb3396b5c49839d742234666667e5e7b1ee7)), closes [#2](https://github.com/rshade/finfocus-spec/issues/2)
- establish constitution v1.0.0 for gRPC proto specification governance ([#57](https://github.com/rshade/finfocus-spec/issues/57)) ([54578aa](https://github.com/rshade/finfocus-spec/commit/54578aa259cb8907f989196ce2b73e37e57f906f))

## [Unreleased]

### Added

### Changed

### Fixed

## [0.1.0] - 2025-11-18

### Added

- Add comprehensive Plugin Registry Specification
  ([#29](https://github.com/rshade/finfocus-spec/issues/29))
  ([5825eaa](https://github.com/rshade/finfocus-spec/commit/5825eaab1f343b1f9fcb966c38effb6a743d3494)),
  closes [#8](https://github.com/rshade/finfocus-spec/issues/8)
- Comprehensive testing framework and enterprise CI/CD pipeline
  ([#19](https://github.com/rshade/finfocus-spec/issues/19))
  ([3a235ef](https://github.com/rshade/finfocus-spec/commit/3a235eff0ce4a172a7b920326066227540c6c8b8))
- Enhance Provider enum with String() method and improved error handling
  ([#39](https://github.com/rshade/finfocus-spec/issues/39))
  ([acbaf0c](https://github.com/rshade/finfocus-spec/commit/acbaf0c6db29d986211d3ca8ef19a66e3162e2c2)),
  closes [#4](https://github.com/rshade/finfocus-spec/issues/4)
- Freeze costsource.proto v0.1.0 specification
  ([#17](https://github.com/rshade/finfocus-spec/issues/17))
  ([3b485b9](https://github.com/rshade/finfocus-spec/commit/3b485b96fef7dc2992c166980f324907e4ff06bd)),
  closes [#3](https://github.com/rshade/finfocus-spec/issues/3)
- Freeze costsource.proto v0.1.0 specification
  ([#18](https://github.com/rshade/finfocus-spec/issues/18))
  ([a085bd2](https://github.com/rshade/finfocus-spec/commit/a085bd202e266189efba92a1832fa8f90c0931e6)),
  closes [#3](https://github.com/rshade/finfocus-spec/issues/3)

### Documentation

- Add comprehensive plugin developer guide
  ([#16](https://github.com/rshade/finfocus-spec/issues/16))
  ([b0a5eb3](https://github.com/rshade/finfocus-spec/commit/b0a5eb3396b5c49839d742234666667e5e7b1ee7)),
  closes [#2](https://github.com/rshade/finfocus-spec/issues/2)
- Establish constitution v1.0.0 for gRPC proto specification governance
  ([#57](https://github.com/rshade/finfocus-spec/issues/57))
  ([54578aa](https://github.com/rshade/finfocus-spec/commit/54578aa259cb8907f989196ce2b73e37e57f906f))

[0.1.0]: https://github.com/rshade/finfocus-spec/releases/tag/v0.1.0
