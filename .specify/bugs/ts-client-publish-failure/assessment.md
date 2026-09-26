# Bug Assessment: TypeScript client publish fails because `npm ci` rejects the lockfile

- **Slug**: ts-client-publish-failure
- **Created**: 2026-09-25
- **Source**: <https://github.com/rshade/finfocus-spec/issues/498>
  - Host: `github.com`
  - URL policy: allowlisted (read with `gh issue view 498`)
  - Linked Actions run 34295820101 was read with `gh run view --log-failed`, **confirmed by
    the user**
- **Verdict**: valid
- **Severity**: medium

## Report (summarized)

Issue #498, "[ALERT] TypeScript SDK publish failed for finfocus-client-v0.6.5" (labels: `bug`,
`automation`, `typescript-sdk`, `failure`). The `Notify on failure` step of
`publish-ts-client.yml` opened the issue automatically. Its body only links to the failed run.

Details from the failed run:

- Run 34295820101, triggered by `release` for tag `finfocus-client-v0.6.5` (commit `9923c8d`)
  on 2026-09-09.
- The `publish` job failed at **Install dependencies** (`npm ci` in `sdk/typescript`), using
  Node 24.15.0:

```text
npm error code EUSAGE
npm error `npm ci` can only install packages when your package.json and package-lock.json or
npm-shrinkwrap.json are in sync. Please update your lock file with `npm install` before continuing.
npm error Missing: @emnapi/core@1.11.3 from lock file
npm error Missing: @emnapi/runtime@1.11.3 from lock file
```

The build and publish steps never ran, so `@rshade/finfocus-client@0.6.5` was not published.

## Symptom

**Observed**: `npm ci` in `sdk/typescript` exits with `EUSAGE` because two optional peer
dependencies are missing from `package-lock.json`, so the publish workflow fails before it
builds anything.

**Expected**: `npm ci` installs from the committed lockfile, and the client package builds and
publishes for every `finfocus-client-v*` release.

## Reproduction

Reproduced locally at HEAD `1f83adc` with the toolchain CI uses (Node 24.15.0, npm 11.12.1,
pinned in `mise.toml`), on a clean copy made with `git archive`:

1. `git archive HEAD sdk/typescript | tar -x -C <scratch>`
2. `cd <scratch>/sdk/typescript`
3. `mise exec node@24.15.0 -- npm ci --dry-run --ignore-scripts`
4. The command fails with the same `EUSAGE` and the same two `Missing: @emnapi/…@1.11.3` lines
   as CI.

`sdk/typescript/package-lock.json` is identical at `9923c8d`, the failed tag, and at HEAD, so
**the next client release would fail the same way.**

## Suspected Code Paths

- `sdk/typescript/package-lock.json`:
  - `node_modules/@napi-rs/wasm-runtime` (around line 840) is `optional` and declares
    `peerDependencies` `@emnapi/core ^1.7.1` and `@emnapi/runtime ^1.7.1`.
  - The lockfile has **no top-level** `node_modules/@emnapi/core` or
    `node_modules/@emnapi/runtime`. It only has copies nested under
    `node_modules/@rolldown/binding-wasm32-wasi/node_modules/@emnapi/*` at `1.11.1`.
  - `npm ci` resolves the unsatisfied peer to the current `1.11.3`, finds it is not in the
    lockfile, and aborts.
- The lockfile was last changed by `4dc11db` (#495) and `ef3e41d` (#464, Renovate "Update NPM
  dev dependencies"). The top-level peer entries were lost in one of these regenerations,
  probably because a different npm version wrote the file.
- `.github/workflows/publish-ts-client.yml:117-121`: the failing `npm ci` step.
- `.github/workflows/ci.yml:35,92,152,178`: **every** pull-request `npm ci` runs at the repo
  root. No CI job installs, builds, or tests `sdk/typescript`, so lockfile drift there
  surfaces only at release time, as an automated issue after the tag is already cut.
- `release-please-config.json`: the `sdk/typescript/packages/client` component has no
  `extra-files`. release-please bumps `packages/client/package.json` (now `0.6.5`) but leaves
  the workspace entry `packages["packages/client"].version` in
  `sdk/typescript/package-lock.json` at `0.6.4`. This drift is cosmetic; it was verified not
  to break `npm ci`. It does churn every regeneration.

## Root Cause Hypothesis

`sdk/typescript/package-lock.json` is missing the top-level entries for the optional peer
dependencies `@emnapi/core` and `@emnapi/runtime`, which `@napi-rs/wasm-runtime` requires
through rolldown's wasm32 binding. Current npm computes those peers during `npm ci`, sees they
are not locked, and refuses to install. The lockfile was most likely written by a different
npm version, from Renovate or a local install, that nested the packages instead of hoisting
them. It shipped because no pull-request CI job runs `npm ci` in `sdk/typescript`.

**Confidence**: high on the mechanism, which was reproduced, and on the fix, which was tried in
scratch. Medium on which commit and which npm version produced the bad lockfile.

## Proposed Remediation

**Preferred**:

1. **Fix the lockfile.** In `sdk/typescript`, run
   `mise exec node@24.15.0 -- npm install --package-lock-only --ignore-scripts` with the pinned
   npm 11.12.1. This was tried in a scratch copy:
   - it adds top-level `node_modules/@emnapi/core@1.11.3` and `node_modules/@emnapi/runtime@1.11.3`
     (both `optional`, `peer`);
   - it syncs the stale `packages/client` version to `0.6.5` and normalizes a few `"peer": true`
     flags, for a 42-line diff;
   - `npm ci --dry-run` then succeeds ("added 294 packages").
2. **Catch drift on pull requests.** Add a `typescript-sdk` job to `ci.yml` that mirrors the
   publish workflow's install and build: Node from `mise.toml`, then `npm ci` and
   `npm run build -w packages/client` in `sdk/typescript`, plus the client's tests if they run
   cleanly. Scope it to the client, because the memory notes a pre-existing TS7006 build error
   in `packages/middleware`; verify that during the fix. Pass explicit `install_args` to
   `jdx/mise-action` so specify-cli is not installed, per `CLAUDE.md`.
3. **Stop version drift in the workspace lockfile.** Add an `extra-files` entry to the
   `sdk/typescript/packages/client` component. Its path is `/sdk/typescript/package-lock.json`
   (release-please treats paths with a leading `/` as repo-root-relative), with type `json` and
   jsonpath `$.packages['packages/client'].version`.

**Alternatives**:

- *Use `npm install` instead of `npm ci` in the publish workflow.* Rejected: the publish would
  no longer be reproducible or pinned to the reviewed lockfile, and the drift would be hidden
  instead of fixed.
- *Add `@emnapi/core` and `@emnapi/runtime` as explicit root `devDependencies`.* This pins the
  peers so any npm writes them at the top level. It is more robust against future
  regenerations, but it adds two dependencies nobody uses directly and makes Renovate bump them.
  Keep it as a fallback if the regenerated lockfile regresses again.
- *Use Renovate's `postUpgradeTasks` or `lockFileMaintenance` with a pinned npm.* It addresses
  the likely origin but needs more setup, and CI item 2 catches the result anyway.

**Files likely to change**:

- `sdk/typescript/package-lock.json`: regenerated
- `.github/workflows/ci.yml`: new `typescript-sdk` job
- `release-please-config.json`: `extra-files` for the client component

**Tests to add or update**:

- The new CI job is the regression test: `npm ci` plus the client build in `sdk/typescript` on
  every pull request.
- Locally: `npm ci` with no `--dry-run`, then `npm run build -w packages/client`, and the
  client tests (`npm test -w packages/client`) with the pinned toolchain.
- `make lint-yaml` for the workflow change.
- Validate `release-please-config.json` against the schema, or at least with `jq`.

## Risks & Considerations

- **The failed tag cannot simply be re-run.**
  - `publish-ts-client.yml` checks out the tag's ref, and `finfocus-client-v0.6.5` points at
    `9923c8d`, which has the bad lockfile. A `workflow_dispatch` for that tag fails the same
    way. 0.6.5 needs a new release, such as 0.6.6, cut after the fix.
  - release-please treats `sdk/typescript/packages/client` as its own component. A commit that
    changes only `sdk/typescript/package-lock.json` or `.github/` is **outside that path, so it
    will not trigger a client release**. It needs a commit touching `packages/client`, or a
    `Release-As:` footer. See Open Questions.
- The new CI job adds about a minute to pull requests. A `paths` filter is possible, but a
  root-level change, such as a Node bump in `mise.toml`, can also break the TypeScript install,
  so running it always is safer.
- Regenerating the lockfile with a different npm can rewrite other metadata. Use the pinned
  toolchain and review the diff; the scratch run changed 42 lines.
- `packages/middleware` may have a pre-existing build error, which is why the CI job builds
  only the client. The fix should confirm this rather than assume it.
- There is no Go SDK, proto, or runtime impact. The blast radius is TypeScript client
  distribution only.

## Open Questions

- [NEEDS CLARIFICATION: how should 0.6.5 be delivered after the fix? Options:
  - (a) Let the fix commit touch `sdk/typescript/packages/client`, for example a no-op
    `package.json` or README change, so release-please proposes `finfocus-client-v0.6.6`.
  - (b) Add a `Release-As: 0.6.6` footer, scoped to the client component.
  - (c) Skip 0.6.5 and ship with the next real client change.

  Also: should #498 be closed by this fix, or once a client version actually publishes?]
- [NEEDS CLARIFICATION: include the release-please `extra-files` change (remediation item 3)
  in this fix, or split it out? It is cosmetic; the missing `@emnapi` entries alone caused the
  failure.]
