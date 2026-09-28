# Bug Assessment: middleware and framework-plugins packages have never compiled

- **Slug**: issue-513-ts-middleware-build
- **Created**: 2026-09-27
- **Source**: <https://github.com/rshade/finfocus-spec/issues/513>
  (host: `github.com`, allowlisted; read via `gh issue view`)
- **Verdict**: valid
- **Severity**: medium
- **Resolution chosen**: option 1 from the issue, "fix and keep" (maintainer decision, 2026-09-27)

## Report (summarized)

`sdk/typescript/packages/middleware` (`finfocus-middleware`) and `sdk/typescript/packages/framework-plugins`
(`finfocus-framework-plugins`) have never compiled since they were added in #302. They are not published, and the
`typescript-sdk` CI job builds and tests only `packages/client`. The issue lists these compile errors: the non-existent
`createNodeHttpTransport` export, an undeclared `@connectrpc/connect-node` dependency, TS2591 for `http`/`https` under
TypeScript 6, TS7006 on the interceptor's `next`, TS6306 on non-composite project references, and a dependency on the
unscoped `finfocus-client` name, which is a dependency-confusion risk. It also notes that the framework adapters have
no tests.

## Symptom

`npm run build` (tsup DTS) and `tsc --noEmit` fail in both packages, and `npm test` fails because there are no test
files. The expected behavior is that both packages build and are tested alongside `packages/client`.

## Reproduction

Toolchain: Node 24.15.0, npm 11.12.1, TypeScript 6.0.3, and tsup 8.5.1, on `origin/main` at `1e80f41`.

1. `cd sdk/typescript && npm ci && npm run build -w packages/client`
2. `cd packages/middleware && npx tsc --noEmit`, which gives TS6306: the referenced project `../client` must set `composite`
3. `npm run build`, which gives a `DTS Build error`
4. `npm test`, which exits 1 with no test files
5. Repeat steps 2 to 4 in `packages/framework-plugins`, which gives TS6306 for both `../client` and `../middleware`

TS6306 stops `tsc` before it reports the source errors the issue lists. Those errors surface once the references are
removed.

## Suspected Code Paths

The compile failures listed in the issue are all confirmed:

- `packages/middleware/src/transport.ts:2`: `createNodeHttpTransport` does not exist in `@connectrpc/connect-node@2.2.0`.
  The `httpClient`/`httpsClient` options are not real transport options.
- `packages/middleware/package.json`: `@connectrpc/connect-node` is missing, and it depends on the unscoped
  `finfocus-client`.
- `packages/{middleware,framework-plugins}/tsconfig.json`: `references` point at non-composite projects (TS6306), and
  `"types": ["node"]` is missing (TS2591 under TypeScript 6).
- `packages/framework-plugins/package.json`: the unscoped `finfocus-client` again.

Reading the code turned up **runtime defects the compile errors were hiding**. Adapter tests would fail on every one
of them:

- `transport.ts:30-46`: the timeout interceptor creates an `AbortController` but never connects it to the request
  signal, so the timeout **never fires**. connect-node already offers `defaultTimeoutMs`, which raises
  `Code.DeadlineExceeded`.
- `gateway.ts:146`: the gateway passes raw parsed JSON to the client, which is not proto3 JSON decoding. `Timestamp`
  fields sent as RFC 3339 strings and `int64` values sent as strings are not converted.
- `gateway.ts:120`: `JSON.stringify(response)` runs on a protobuf-es v2 message. A response that contains an `int64`
  or a `Timestamp` (`seconds` is a `bigint`) throws `TypeError: Do not know how to serialize a BigInt`. Nearly every
  cost response contains one, for example `GetActualCostResponse.results[].timestamp`.
- `gateway.ts:141-146`: methods are resolved by duck typing on the client object. `/finfocus.v1.CostSourceService/Constructor`
  resolves to `client.constructor` and calls a class constructor without `new`.
- `gateway.ts:75-99`: the gateway always reads the raw request stream. When a framework has already consumed the body
  (`express.json()`, NestJS's default body parser, or Fastify), `end` never fires and **the request hangs forever**.
- `framework-plugins/src/fastify/index.ts:119-145`: the mock `IncomingMessage` has no-op `on`/`once`, so the gateway
  never sees `end`. **Every Fastify request hangs.**
- `framework-plugins/src/express/index.ts:74-78`: `require('express')` fails in the ESM build, and `router.post('*')`
  is invalid in Express 5 (path-to-regexp v8 needs a named wildcard).
- `framework-plugins/src/nestjs/index.ts`: NestJS parses the body by default, so it hits the hang above.
  `@All('finfocus.v1.*')` uses the legacy wildcard syntax.
- Root `package.json`: `"workspaces": ["packages/*"]` runs `--workspaces` scripts in alphabetical order, so
  `framework-plugins` builds before `middleware`, which is its dependency.
- `.github/workflows/ci.yml:180-186`: CI builds and tests only `packages/client`.

## Root Cause Hypothesis

The packages were generated in #302 against an imagined connect-node API and were never compiled or exercised. CI
only covered the client, so nothing caught it. Confidence: **high**. Every failure above can be read directly from
the source or reproduces with the commands listed.

## Proposed Remediation

**Preferred** (option 1, fix and keep):

1. **Dependencies and config.** Depend on `@rshade/finfocus-client` (scoped) and `@connectrpc/connect-node@^2.2.0`, and
   drop the unused `@bufbuild/protobuf` duplication only where it is truly unused. Remove `references` from both
   tsconfigs, add `"types": ["node"]`, and enable `experimentalDecorators` for framework-plugins. List the workspaces
   in dependency order in the root `package.json` so `--workspaces` builds client, then middleware, then
   framework-plugins.
2. **Transport.** Rewrite `createNodeTransport` on `createConnectTransport` from connect-node, with `httpVersion`
   (default `"1.1"`), `nodeOptions`, and `timeout` mapped to `defaultTimeoutMs`.
3. **Gateway.** Resolve methods from the service descriptors (`CostSourceService`, `ObservabilityService`,
   `PluginRegistryService`) instead of duck typing. Decode with `fromJson(method.input, body)` and encode with
   `toJson(method.output, response)`. Split out a transport-agnostic `dispatch(path, body)` that returns
   `{ status, body }`. `handleRequest` should use `req.body` when a framework has already parsed it. Map a
   `ConnectError` to an HTTP status instead of a flat 500.
4. **Adapters.** Fastify calls `dispatch` with `request.body` instead of mock objects. Express uses a static `Router`
   import with a regex route. NestJS relies on the pre-parsed body and a valid wildcard route.
5. **CI.** Build and test every workspace.

**Alternatives**:

- Keep the raw-JSON pass-through and use a `bigint`-aware `JSON.stringify` replacer. This is simpler, but the gateway
  would still not speak proto3 JSON (timestamps, enums as names, and so on), and it would disagree with the Connect
  JSON wire format.
- Option 2, removing both packages. The maintainer rejected it.

**Files likely to change**:

- `sdk/typescript/package.json`, `sdk/typescript/package-lock.json`
- `sdk/typescript/packages/middleware/{package.json,tsconfig.json,src/transport.ts,src/gateway.ts}`
- `sdk/typescript/packages/framework-plugins/{package.json,tsconfig.json,src/**}`
- New tests: `packages/middleware/test/{transport,gateway}.test.ts` and `packages/framework-plugins/test/*.test.ts`
- `.github/workflows/ci.yml`
- `sdk/typescript/README.md`, to fix package names, the install commands, and the transport examples

**Tests to add or update**:

- Transport: a server that never responds, with `timeout: 50`, rejects with `ConnectError` `Code.DeadlineExceeded`.
- Gateway: a round trip through an in-process Connect server (`connectNodeAdapter`) to prove proto3 JSON in and out,
  including a `bigint`/`Timestamp` field. Also check 404 for an unknown service or method (including `Constructor`),
  405 for non-POST, 400 for malformed JSON, 413 for oversized bodies, and use of a pre-parsed `req.body`.
- One smoke test each for Express, Fastify, and NestJS: POST a request through the adapter to a mock plugin and
  assert a 200 JSON response.

## Risks & Considerations

- The exported surface of the unpublished packages changes. `NodeTransportConfig` swaps `httpClient`/`httpsClient`
  for `httpVersion`/`nodeOptions`, and gateway JSON becomes proto3 JSON. Nothing ever compiled or published these
  packages, so there are no consumers to break. No `.proto` or Go change is involved.
- NestJS smoke testing needs `@nestjs/platform-express` as a devDependency. It is a new lockfile entry.
- Renovate PRs #521 and #526 (NestJS bumps) touch the same lockfile and will need a rebase.
- Widening CI adds roughly 30 seconds to the `typescript-sdk` job.

## Open Questions

None. The maintainer chose the resolution.
