# Bug Verification: middleware and framework-plugins packages have never compiled

- **Slug**: issue-513-ts-middleware-build
- **Tested**: 2026-09-27
- **Assessment**: ./assessment.md
- **Fix**: ./fix.md
- **Result**: verified

## Summary

The reproduction from the assessment no longer fails, even from a clean `node_modules`. Both packages type-check,
build (cjs, esm, and dts), and pass their tests, and the client suite is unchanged. The repository gates found no
regression.

## Checks Performed

| Check | Command / Action | Result | Notes |
|-------|------------------|--------|-------|
| Reproduction step 1 | `rm -rf packages/*/dist node_modules && npm ci && npm run build -w packages/client` | pass | 0 vulnerabilities; client DTS built |
| Reproduction step 2 (middleware) | `npx tsc --noEmit` | pass | exit 0; TS6306 gone |
| Reproduction step 3 (middleware) | `npm run build` | pass | DTS build success |
| Reproduction step 4 (middleware) | `npm test` | pass | 2 files, 17 tests (after the review added an HTTP/2 case) |
| Reproduction step 5 (framework-plugins) | `tsc --noEmit` / `npm run build` / `npm test` | pass | exit 0; DTS success; 3 files, 8 tests |
| CI-equivalent job | `npm run build --workspaces && npm test --workspaces` | pass | build exit 0; tests 78 + 17 + 8 |
| Type-check including tests | `npm run lint --workspaces` | pass | exit 0 |
| No unscoped client dependency | `grep '"finfocus-client"' packages/*/package.json`; lockfile `node_modules/finfocus-client` | pass | 0 matches in both |
| Tests are red without the fix | Ran against the unfixed sources during `/speckit-bug-fix` | pass | Transport: `createNodeHttpTransport is not a function`. Express/NestJS: path-to-regexp errors. Fastify: timeouts |
| Generated code current | `make generate && git diff --exit-code -- sdk/` | pass | No diff in Go or TS bindings |
| Go regression suite | `make test` | pass | exit 0; no Go files changed |
| Markdown lint | `make lint-markdown` | pass | 510 files, 0 issues |
| YAML lint | `uvx yamllint .github/` (the command `make lint-yaml` runs) | pass | `yamllint` is not on PATH locally, so run through `uvx` |
| Schema and examples | `make validate-npm` | pass | All examples valid |
| Go lint / buf lint / buf breaking | `make lint-go`, `buf breaking` | skipped | The diff touches no `.go` or `.proto` file |

## Output Excerpts

```text
== middleware test
 Test Files  2 passed (2)
      Tests  17 passed (17)
== framework-plugins test
 Test Files  3 passed (3)
      Tests  8 passed (8)
== workspaces
build=0
      Tests  78 passed (78)
lint=0
```

## Residual Risks

- The NestJS smoke test covers only `@nestjs/platform-express`. The Fastify platform for NestJS is not exercised.
- The adapters are verified against the stub `CostSourceClient`, and the middleware tests verify the real
  transport-to-plugin path separately. No single test runs framework → gateway → network → plugin.
- Neither package is published yet, so no consumer install path (`npm install finfocus-middleware`) has been tested.
- The CI job change is verified locally with the same commands. The first GitHub run will confirm it.

## Recommendation

Close the bug once the PR merges. It is verified end to end locally, and the reproduction commands from the
assessment now succeed.
