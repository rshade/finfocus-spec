# Bug Verification: TypeScript client publish fails because `npm ci` rejects the lockfile

- **Slug**: ts-client-publish-failure
- **Tested**: 2026-09-25
- **Assessment**: ./assessment.md
- **Fix**: ./fix.md
- **Result**: verified

## Summary

Both failures reproduce on clean copies of HEAD (`1f83adc`). The first is `npm ci` failing
with `EUSAGE` and `Missing: @emnapi/core@1.11.3`, the original #498 symptom. The second is the
client declaration build failing with TS5101 once the lockfile is fixed. On a clean copy of
the fixed tree, the full sequence passes: the CI job's steps, then the publish workflow's
steps up to a dry-run publish. No regressions were found.

## Checks Performed

| Check | Command / Action | Result | Notes |
|-------|------------------|--------|-------|
| Reproduction A (pre-fix) | `git archive HEAD sdk/typescript` into scratch; `npm ci` (Node 24.15.0 / npm 11.12.1) | fail (expected) | Same `EUSAGE` and the same two `Missing: @emnapi/…@1.11.3` lines as CI run 34295820101 |
| Reproduction B (pre-fix) | Same HEAD copy with only the fixed lockfile; `npm ci`, then `npm run build -w packages/client` | fail (expected) | `npm ci` passes, then `TS5101: Option 'baseUrl' is deprecated` and `DTS Build error` |
| Post-fix install | Clean copy of the fixed tree's tracked files (no `node_modules` or `dist`); `npm ci` | pass | "added 294 packages" |
| Post-fix build | `npm run build -w packages/client` | pass | CJS, ESM, and DTS all build |
| Client tests | `npm test -w packages/client` | pass | 5 files, 73 tests |
| Publish (simulated) | `npm publish -w packages/client --dry-run --registry https://npm.pkg.github.com` | pass | `+ @rshade/finfocus-client@0.6.5`, 7 files; nothing was actually published |
| Lockfile stability | `cmp` against the committed lockfile after `npm ci`, the build, and the publish dry-run | pass | Byte-identical; `npm ci` does not rewrite it |
| release-please JSONPath | Evaluated `$.packages['packages/client'].version` with `jsonpath-plus`, the library release-please uses, against the lockfile | pass | Exactly one match: `"0.6.5"` |
| YAML lint | `make lint-yaml` | pass | |
| Workflow lint | `actionlint .github/workflows/ci.yml .github/workflows/publish-ts-client.yml` | pass for `ci.yml` | One info-level SC2086 in the unchanged `publish-ts-client.yml` (pre-existing) |
| Markdown lint | `npx markdownlint-cli2 .specify/bugs/ts-client-publish-failure/*.md` | pass | 0 issues |
| Go unaffected | `git diff --name-only`, then `go build ./...` | pass | 0 `.go` files changed; the build is ok |
| CI job on GitHub | Push and let `typescript-sdk` run | not-run | Runs when the branch is pushed; it was exercised locally with the same toolchain |
| Real publish | The publish workflow for a new client tag | not-run | Deliberately deferred: 0.6.5 is skipped and the next client change will release it (user decision 1c) |

## Output Excerpts

Pre-fix:

```text
npm error code EUSAGE
npm error Missing: @emnapi/core@1.11.3 from lock file
npm error Missing: @emnapi/runtime@1.11.3 from lock file
---
error TS5101: Option 'baseUrl' is deprecated and will stop functioning in TypeScript 7.0.
DTS Build error
```

Post-fix:

```text
added 294 packages in 5s
DTS ⚡️ Build success in 984ms
 Tests  73 passed (73)
+ @rshade/finfocus-client@0.6.5
matches: ["0.6.5"]
```

## Residual Risks

- **The reproduction ran locally, not on GitHub Actions.** It used the same pinned toolchain
  as CI and the publish workflow (Node 24.15.0 from `mise.toml`, npm 11.12.1). The real
  confirmation comes from the first `typescript-sdk` CI run on this branch's PR and, later,
  the next client release.
- **The `ignoreDeprecations: "6.0"` workaround** silences all TypeScript 6 deprecations in
  `tsc`. None currently exist in the client. It must be removed before TypeScript 7.
- **The lockfile can drift again** when Renovate or another npm version regenerates it. The
  new CI job now catches that on the pull request instead of at publish time.
- **Not covered:** `packages/middleware` and `packages/framework-plugins` still fail to build,
  for pre-existing, unrelated reasons, and are unpublished. The CI job covers only the client.
- **The release-please change was validated statically** (JSONPath evaluation, and the path
  rule in `addPath()`), not by a real release-please run.
- **Pre-existing, out of scope:** actionlint SC2086 in `publish-ts-client.yml` (unquoted
  `$GITHUB_OUTPUT`, info level).
- **An initial lockfile-stability check misfired.** `git -C <root> diff --no-index` resolved
  the relative path against the repo root. It was re-run with `cmp` on the correct files, which
  confirmed no change.

## Recommendation

Verified locally end to end, so merge the fix. The publish path now works through a dry-run
publish, and pull-request CI will catch recurrence. Keep #498 open until the first
`typescript-sdk` CI run passes on GitHub, or until the next client release publishes. Also
consider closing it with a note that 0.6.5 was intentionally skipped. File the follow-ups
from `fix.md` separately: the broken middleware and framework-plugins builds, removing
`ignoreDeprecations` before TypeScript 7, and Renovate using the pinned npm.
