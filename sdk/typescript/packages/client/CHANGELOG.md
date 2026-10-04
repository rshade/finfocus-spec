# Changelog

## [0.7.3](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.7.2...finfocus-client-v0.7.3) (2026-10-04)


### Features

* **proto:** add a ResourceDescriptor to GetActualCostRequest ([#621](https://github.com/rshade/finfocus-spec/issues/621)) ([24cec1e](https://github.com/rshade/finfocus-spec/commit/24cec1e90f13c52eac6629a97bb4e807f7f89cd9))

## [0.7.2](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.7.1...finfocus-client-v0.7.2) (2026-10-04)


### Features

* **proto:** add ResourceDescriptor attributes and raise tag limit ([7843e7d](https://github.com/rshade/finfocus-spec/commit/7843e7dc831e0e4f10f4809039c55dd0aca42648)), closes [#617](https://github.com/rshade/finfocus-spec/issues/617)


### Bug Fixes

* address PR 618 review and contract suite allocation ([fa0f5e3](https://github.com/rshade/finfocus-spec/commit/fa0f5e39af16875ec3dee41c8d4ee69420944637))

## [0.7.1](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.7.0...finfocus-client-v0.7.1) (2026-10-02)


### Features

* **pluginsdk:** advertise scorer limits and mark oversize batches ([#602](https://github.com/rshade/finfocus-spec/issues/602)) ([db5da5b](https://github.com/rshade/finfocus-spec/commit/db5da5b5e258bbf1e0a804d50b5f8ab2248d9af0))
* **proto:** add billing_account_id to GetActualCostRequest ([#592](https://github.com/rshade/finfocus-spec/issues/592)) ([0458180](https://github.com/rshade/finfocus-spec/commit/04581801101312240f5d3974fe30cd35353c39f4)), closes [#590](https://github.com/rshade/finfocus-spec/issues/590)
* **proto:** add FOCUS 1.3 provenance fields to AllocationRow ([#605](https://github.com/rshade/finfocus-spec/issues/605)) ([c1a07b0](https://github.com/rshade/finfocus-spec/commit/c1a07b02a33ce0d2b1c55969b276846762b56abd)), closes [#578](https://github.com/rshade/finfocus-spec/issues/578)
* **proto:** add omitted_fields to ScoreRecommendationsRequest ([#604](https://github.com/rshade/finfocus-spec/issues/604)) ([9eccf57](https://github.com/rshade/finfocus-spec/commit/9eccf57a87b5c3f63dc84d19566af39fef56c366)), closes [#576](https://github.com/rshade/finfocus-spec/issues/576)
* **proto:** add period and selector to AllocateRequest ([#606](https://github.com/rshade/finfocus-spec/issues/606)) ([0427ae2](https://github.com/rshade/finfocus-spec/commit/0427ae2ba67a96c7dd29de55f9d03bce1a5941f3)), closes [#579](https://github.com/rshade/finfocus-spec/issues/579)
* **proto:** add RegionPrice for per-region retail prices ([#600](https://github.com/rshade/finfocus-spec/issues/600)) ([50d7d95](https://github.com/rshade/finfocus-spec/commit/50d7d95781c5f359c27dbdc1c8316b5ebd9b0c65)), closes [#589](https://github.com/rshade/finfocus-spec/issues/589)
* **proto:** add repeated PriceOption for alternative retail prices ([#599](https://github.com/rshade/finfocus-spec/issues/599)) ([402f84d](https://github.com/rshade/finfocus-spec/commit/402f84de5269052792d695df5591b6924dc39897)), closes [#588](https://github.com/rshade/finfocus-spec/issues/588)
* **proto:** add scorer session_id so duplicate groups span batches ([#603](https://github.com/rshade/finfocus-spec/issues/603)) ([6c078c4](https://github.com/rshade/finfocus-spec/commit/6c078c46da73d34498248b335e18d3ca8c53b8b3)), closes [#574](https://github.com/rshade/finfocus-spec/issues/574)
* **proto:** list every request id and model on ScorerInfo ([#608](https://github.com/rshade/finfocus-spec/issues/608)) ([7a15bd6](https://github.com/rshade/finfocus-spec/commit/7a15bd6fa9858089a6c8ee9c35aabab6ee7dad0a))

## [0.7.0](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.6...finfocus-client-v0.7.0) (2026-09-29)


### Features

* **pluginsdk:** add FOCUS 1.4 cost and usage columns ([#549](https://github.com/rshade/finfocus-spec/issues/549)) ([48b5dcd](https://github.com/rshade/finfocus-spec/commit/48b5dcd8bf0c1387d30ea752d54965ebf47a6ff9)), closes [#541](https://github.com/rshade/finfocus-spec/issues/541)
* **proto:** add cost allocation lineage metadata ([#546](https://github.com/rshade/finfocus-spec/issues/546)) ([d85660a](https://github.com/rshade/finfocus-spec/commit/d85660ae273be1e1d41c74b77417e8f76bd2f1f8)), closes [#191](https://github.com/rshade/finfocus-spec/issues/191)
* **proto:** add cost_breakdown map to GetProjectedCostResponse ([#537](https://github.com/rshade/finfocus-spec/issues/537)) ([7a29919](https://github.com/rshade/finfocus-spec/commit/7a29919480c2e0d3260262387f32a57d1975afed))
* **proto:** add FOCUS 1.4 billing period and invoice detail ([#554](https://github.com/rshade/finfocus-spec/issues/554)) ([5220a1a](https://github.com/rshade/finfocus-spec/commit/5220a1a44bd6b2406b2e9e5928f4b229fc86ca2a)), closes [#543](https://github.com/rshade/finfocus-spec/issues/543)
* **proto:** add FOCUS 1.4 contract commitment columns ([#553](https://github.com/rshade/finfocus-spec/issues/553)) ([487e2c0](https://github.com/rshade/finfocus-spec/commit/487e2c049704b9cdfb5f073def4c931ece6490b1)), closes [#542](https://github.com/rshade/finfocus-spec/issues/542)
* **proto:** add RecommendationScorerService.ScoreRecommendations ([#559](https://github.com/rshade/finfocus-spec/issues/559)) ([8020f96](https://github.com/rshade/finfocus-spec/commit/8020f968a8b9459a26f1b9f9f4be7991cb9a1fe3)), closes [#556](https://github.com/rshade/finfocus-spec/issues/556)
* **proto:** add SupplementalDatasetService for contract commitments ([#552](https://github.com/rshade/finfocus-spec/issues/552)) ([91c1374](https://github.com/rshade/finfocus-spec/commit/91c13748b4ebe5ceeaee285d097c6dec10901193)), closes [#544](https://github.com/rshade/finfocus-spec/issues/544)
* **proto:** serve FOCUS billing period and invoice detail ([#555](https://github.com/rshade/finfocus-spec/issues/555)) ([31e9006](https://github.com/rshade/finfocus-spec/commit/31e900635acd053ee17854fd02059828a0a9a2d4))


### Miscellaneous Chores

* release 0.7.0 ([edb377a](https://github.com/rshade/finfocus-spec/commit/edb377a03ebb38c67c793fd7cf5d986a153b7c32))

## [0.6.6](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.5...finfocus-client-v0.6.6) (2026-09-28)


### Features

* **pluginsdk:** add ValidateAllocateResponse and fix allocator checks ([#518](https://github.com/rshade/finfocus-spec/issues/518)) ([25b3b27](https://github.com/rshade/finfocus-spec/commit/25b3b27630f0c9eb28a195c65445afaa80e543aa))
* **proto:** add AllocatorService.Allocate for cost allocation plugins ([#515](https://github.com/rshade/finfocus-spec/issues/515)) ([063e0b9](https://github.com/rshade/finfocus-spec/commit/063e0b9aabc0c9950564c8859d3288e6eedcb1bc)), closes [#506](https://github.com/rshade/finfocus-spec/issues/506)
* **proto:** add UsageSourceService.GetStats for workload usage plugins ([#508](https://github.com/rshade/finfocus-spec/issues/508)) ([d314c2a](https://github.com/rshade/finfocus-spec/commit/d314c2af06c42e05cdd76f973eb3fd6d7ed1b513)), closes [#505](https://github.com/rshade/finfocus-spec/issues/505)

## [0.6.5](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.4...finfocus-client-v0.6.5) (2026-09-08)


### Features

* **proto:** add expires_at cache-hint to EstimateCostResponse ([#460](https://github.com/rshade/finfocus-spec/issues/460)) ([d258c1b](https://github.com/rshade/finfocus-spec/commit/d258c1b2cb3f15a20f4ad887aba9c339ead0c6f9)), closes [#434](https://github.com/rshade/finfocus-spec/issues/434)
* **sdk:** harden ResolveResourceTypes RPC ([#493](https://github.com/rshade/finfocus-spec/issues/493)) ([50f73d9](https://github.com/rshade/finfocus-spec/commit/50f73d985d3485fb1c59ad4889d1a3021073562a))

## [0.6.4](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.3...finfocus-client-v0.6.4) (2026-03-13)


### Features

* **proto:** add metadata map to getprojectedcostresponse ([#427](https://github.com/rshade/finfocus-spec/issues/427)) ([e624864](https://github.com/rshade/finfocus-spec/commit/e62486432c58fccdf5b1d79c5aa13c9a6adf0069)), closes [#381](https://github.com/rshade/finfocus-spec/issues/381)


### Bug Fixes

* **pluginsdk:** harden metadata map validation and add sentinel errors ([#435](https://github.com/rshade/finfocus-spec/issues/435)) ([1ee2b12](https://github.com/rshade/finfocus-spec/commit/1ee2b12c5504162d59265f47272857554ee7511d)), closes [#381](https://github.com/rshade/finfocus-spec/issues/381)

## [0.6.3](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.2...finfocus-client-v0.6.3) (2026-02-28)


### Features

* **proto:** add batch cost operation for multi-resource queries ([#392](https://github.com/rshade/finfocus-spec/issues/392)) ([15addc9](https://github.com/rshade/finfocus-spec/commit/15addc9a3792c186a6dad703c8b608ea674bbebe))
* **proto:** add expires_at caching hint to cost results ([#382](https://github.com/rshade/finfocus-spec/issues/382)) ([6bf8705](https://github.com/rshade/finfocus-spec/commit/6bf8705a12111183a357180cbc19e589673bc636))

## [0.6.2](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.1...finfocus-client-v0.6.2) (2026-02-12)


### Features

* **proto:** add pagination support to actual cost retrieval ([#360](https://github.com/rshade/finfocus-spec/issues/360)) ([571a0cd](https://github.com/rshade/finfocus-spec/commit/571a0cdd7c4a822e74dd4a688972bab35fa9afa7)), closes [#353](https://github.com/rshade/finfocus-spec/issues/353)


### Bug Fixes

* **pluginsdk:** harden pagination edge cases and observability ([#376](https://github.com/rshade/finfocus-spec/issues/376)) ([60082d8](https://github.com/rshade/finfocus-spec/commit/60082d843e2fb4d4b9dd7d729daadc561c10d32c)), closes [#364](https://github.com/rshade/finfocus-spec/issues/364) [#367](https://github.com/rshade/finfocus-spec/issues/367) [#368](https://github.com/rshade/finfocus-spec/issues/368) [#369](https://github.com/rshade/finfocus-spec/issues/369)

## [0.6.1](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.6.0...finfocus-client-v0.6.1) (2026-01-29)


### Features

* **proto:** add pricing tier and risk score fields ([#335](https://github.com/rshade/finfocus-spec/issues/335)) ([890f230](https://github.com/rshade/finfocus-spec/commit/890f230a77cb598066ce187e85fcf87e1ebda8b8)), closes [#217](https://github.com/rshade/finfocus-spec/issues/217)
* **proto:** add usage profile enum for workload context signaling ([#349](https://github.com/rshade/finfocus-spec/issues/349)) ([ae82f64](https://github.com/rshade/finfocus-spec/commit/ae82f647590dde04d18a610c3bd3297bef7979c3))
* **sdk:** implement github issues audit plan for [#247](https://github.com/rshade/finfocus-spec/issues/247)-340 ([#341](https://github.com/rshade/finfocus-spec/issues/341)) ([8b5b97d](https://github.com/rshade/finfocus-spec/commit/8b5b97d903ccd1e1f95e975a41e34b2340e7341f))

## [0.6.0](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.5.0...finfocus-client-v0.6.0) (2026-01-21)


### Features

* **proto:** add anomaly category and investigate action ([#332](https://github.com/rshade/finfocus-spec/issues/332)) ([318eeb9](https://github.com/rshade/finfocus-spec/commit/318eeb98ff70d2c560d467d59538a3dcd96b6ef9)), closes [#315](https://github.com/rshade/finfocus-spec/issues/315)


### Bug Fixes

* update release workflow for typescript package ([#317](https://github.com/rshade/finfocus-spec/issues/317)) ([d7cdcb5](https://github.com/rshade/finfocus-spec/commit/d7cdcb54c0649208fe386b8105bb42e66065fbd6))

## [0.5.0](https://github.com/rshade/finfocus-spec/compare/finfocus-client-v0.1.0...finfocus-client-v0.5.0) (2026-01-19)

### Features

- **ci:** implement automated typescript sdk publishing
  ([#312](https://github.com/rshade/finfocus-spec/issues/312))
  ([a12d218](https://github.com/rshade/finfocus-spec/commit/a12d21815c3daa3dafe6cd2db259c230bb34a89f)),
  closes [#311](https://github.com/rshade/finfocus-spec/issues/311)
- **sdk:** add typescript sdk for browser and node.js
  ([#302](https://github.com/rshade/finfocus-spec/issues/302))
  ([f819c3e](https://github.com/rshade/finfocus-spec/commit/f819c3efa695c65ce4f74e18bdcf2260d56455b3))
- **sdk:** polish capability discovery and optimize performance
  ([#310](https://github.com/rshade/finfocus-spec/issues/310))
  ([f74c8c4](https://github.com/rshade/finfocus-spec/commit/f74c8c4d59b2801ca831ed95e9cc5bf15865b1f4)),
  closes [#294](https://github.com/rshade/finfocus-spec/issues/294)
  [#295](https://github.com/rshade/finfocus-spec/issues/295)
  [#299](https://github.com/rshade/finfocus-spec/issues/299)
  [#300](https://github.com/rshade/finfocus-spec/issues/300)
  [#301](https://github.com/rshade/finfocus-spec/issues/301)
  [#208](https://github.com/rshade/finfocus-spec/issues/208)
  [#209](https://github.com/rshade/finfocus-spec/issues/209)

### Bug Fixes

- **sdk:** complete typescript sdk builder api and improve error handling
  ([#303](https://github.com/rshade/finfocus-spec/issues/303))
  ([4804ded](https://github.com/rshade/finfocus-spec/commit/4804ded9252e01e5b2d6eb934909c08e7ed5cb26))
- **sdk:** complete typescript sdk builder api and improve error handling
  ([#303](https://github.com/rshade/finfocus-spec/issues/303))
  ([#305](https://github.com/rshade/finfocus-spec/issues/305))
  ([a1ef832](https://github.com/rshade/finfocus-spec/commit/a1ef8327887376bc39328a288afc2187dad88609))

### Miscellaneous Chores

- release 0.5.0 ([4358f2a](https://github.com/rshade/finfocus-spec/commit/4358f2a060220fc9876d898ecd20136fabb5e3eb))
