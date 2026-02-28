# Changelog

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
