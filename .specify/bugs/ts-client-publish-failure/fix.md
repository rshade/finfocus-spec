# Bug Fix: TypeScript client publish fails because `npm ci` rejects the lockfile

- **Slug**: ts-client-publish-failure
- **Fixed**: 2026-09-25
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

The regenerated `sdk/typescript/package-lock.json` now includes the missing optional peers
`@emnapi/core` and `@emnapi/runtime`. Fixing it exposed a second failure that would also have
blocked the publish: tsup's type-declaration build under TypeScript 6. That is fixed as well.
A new pull-request CI job runs the publish workflow's install, build, and test steps, so either
failure now surfaces before a release tag exists. release-please now keeps the client's version
in the workspace lockfile in sync.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `sdk/typescript/package-lock.json` | modified | Regenerated with the pinned toolchain (Node 24.15.0, npm 11.12.1) using `npm install --package-lock-only`. Adds top-level `@emnapi/core@1.11.3` and `@emnapi/runtime@1.11.3`, syncs `packages/client` to 0.6.5, and normalizes `peer` flags. 42 lines changed |
| `sdk/typescript/tsconfig.base.json` | modified | Adds `"ignoreDeprecations": "6.0"`, which fixes TS5101 in tsup's declaration build; also adds the missing final newline |
| `.github/workflows/ci.yml` | modified | New `typescript-sdk` job: `npm ci`, client build, and client tests in `sdk/typescript`, installing only `node` through mise |
| `release-please-config.json` | modified | The client component gets an `extra-files` entry that updates `$.packages['packages/client'].version` in `/sdk/typescript/package-lock.json` |

## Diff Highlights

```json
"node_modules/@emnapi/core": { "version": "1.11.3", "optional": true, "peer": true, ... },
"node_modules/@emnapi/runtime": { "version": "1.11.3", "optional": true, "peer": true, ... },
```

```json
"extra-files": [
  { "type": "json", "path": "/sdk/typescript/package-lock.json",
    "jsonpath": "$.packages['packages/client'].version" }
]
```

## Tests Added or Updated

- The new `ci.yml` job `typescript-sdk` is the regression test. On every pull request it runs
  the same `npm ci` and client build as `publish-ts-client.yml`, and also the client tests.
- The client's existing vitest suite runs in CI for the first time: 5 files, 73 tests.

## Local Verification

All commands ran in `sdk/typescript` with `mise exec node@24.15.0` (npm 11.12.1):

- `npm ci`: "added 294 packages". Before the fix it failed with `EUSAGE` and
  `Missing: @emnapi/core@1.11.3`.
- `npm run build -w packages/client`: CJS, ESM, and DTS all build. Before the
  `tsconfig.base.json` change, the DTS step failed with
  `error TS5101: Option 'baseUrl' is deprecated`.
- `npm test -w packages/client`: 73/73 pass.
- `npm publish -w packages/client --dry-run`: `+ @rshade/finfocus-client@0.6.5`, 7 files.
- `npx tsc --noEmit -p packages/client/tsconfig.json --ignoreDeprecations 5.0`: exit 0. This
  runs without the 6.0 suppression and shows that no other TypeScript 6 deprecation is being
  hidden in the client.
- `make lint-yaml`: pass. `actionlint .github/workflows/ci.yml`: pass.
- `release-please-config.json` parses as JSON. The `addPath()` code in release-please's source
  confirms that a leading `/` resolves from the repo root. The bracket JSONPath form matches
  the root component's existing `$.packages[''].version`.

## Deviations from Assessment

- **Scope expansion: the TypeScript 6 build failure.**
  - The assessment covered only the lockfile. Once `npm ci` passed, the next publish step,
    `npm run build -w packages/client`, failed as well.
  - Cause: tsup 8.5.1 injects `baseUrl: "."` into its DTS compiler options
    (`node_modules/tsup/dist/rollup.js:6837`), and TypeScript 6.0.3 rejects that with TS5101.
    TypeScript came in at `^6.0.x` through Renovate #464, with the same lockfile as the failed
    tag. So the v0.6.5 publish would have failed at the build step even with a good lockfile.
  - The fix is in the shared `tsconfig.base.json`, not in each package's `tsup.config.ts`: one
    change fixes all three workspaces, and it stays outside `packages/client`, so it does not
    trigger a client release (the user chose option 1c).
  - Trade-off: `ignoreDeprecations: "6.0"` also silences other TypeScript 6 deprecations in
    `tsc`. None currently exist in the client, as verified above. It must be removed before
    TypeScript 7, where `baseUrl` stops working entirely, unless tsup has fixed the injection
    by then.
- **The CI job builds and tests only the client.** `packages/middleware` and
  `packages/framework-plugins` still fail, for separate pre-existing reasons: TS2307 for the
  missing `@connectrpc/connect-node` and `finfocus-middleware`, TS2591 for missing
  `@types/node`, and TS7006. Neither package is published, so they are out of scope.
- **No release is triggered.** Per the user's answer to open question 1 (option c), nothing
  under `sdk/typescript/packages/client` changed. 0.6.5 remains unpublished and the next real
  client change will release it. The `extra-files` item was included per open question 2.

## Follow-ups

- Fix the builds of `packages/middleware` and `packages/framework-plugins` (missing
  dependencies and types), or remove them from the workspace if they are abandoned. After
  that, widen the CI job to `npm run build --workspaces`.
- Remove `ignoreDeprecations` when tsup stops injecting `baseUrl`, or before upgrading to
  TypeScript 7.
- Consider having Renovate regenerate lockfiles with the npm pinned in `mise.toml`, which is
  the likely origin of the dropped peer entries.
- Decide whether to close #498 now or when the next client version publishes. The publish
  path is fixed, but no package has actually been published yet.
