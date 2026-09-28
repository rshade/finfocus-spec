# Bug Fix: middleware and framework-plugins packages have never compiled

- **Slug**: issue-513-ts-middleware-build
- **Fixed**: 2026-09-27
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

Both packages now build, type-check, and are tested. Every workspace package depends on the scoped
`@rshade/finfocus-client`. The Node transport is rebuilt on the real connect-node API. The REST gateway now speaks
proto3 JSON through the service descriptors, and all three framework adapters work. CI builds and tests every
workspace.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `sdk/typescript/package.json` | modified | Workspaces listed in dependency order so `--workspaces` builds client → middleware → framework-plugins |
| `sdk/typescript/package-lock.json` | regenerated | `connect-node`, `@nestjs/platform-express`; unscoped `finfocus-client` link gone |
| `packages/middleware/package.json` | modified | `@rshade/finfocus-client`, `@connectrpc/connect-node@^2.2.0` |
| `packages/{middleware,framework-plugins}/tsconfig.json` | modified | Drop `references` and `rootDir`, add `types: ["node"]`, include `test/`; `experimentalDecorators` for NestJS |
| `packages/middleware/src/transport.ts` | rewritten | `createConnectTransport` from connect-node; `timeout` → `defaultTimeoutMs`; `httpVersion`, `nodeOptions` |
| `packages/middleware/src/gateway.ts` | rewritten | Descriptor dispatch, `fromJson`/`toJson`, public `dispatch()`, pre-parsed `req.body`, Connect code → HTTP status |
| `packages/framework-plugins/package.json` | modified | Scoped client; frameworks moved to optional `peerDependencies`; unused `@connectrpc/connect` dropped, `@bufbuild/protobuf` moved to dev |
| `packages/framework-plugins/src/express/index.ts` | modified | Static `Router` import, `router.use()`, no `as any` |
| `packages/framework-plugins/src/fastify/index.ts` | modified | Calls `gateway.dispatch(request.url, request.body)` instead of mock req/res |
| `packages/framework-plugins/src/nestjs/index.ts` | modified | `@All('finfocus.v1.*path')` (path-to-regexp v8 named wildcard) |
| `packages/middleware/test/*.ts` | added | Plugin server helper, transport and gateway tests |
| `packages/framework-plugins/test/*.ts` | added | Stub client, Express, Fastify, NestJS smoke tests |
| `.github/workflows/ci.yml` | modified | `typescript-sdk` job builds, tests, and type-checks (`lint`) `--workspaces`; renamed "TypeScript SDK" |
| `CLAUDE.md` | modified | Stale tsup DTS note corrected; TypeScript workspace learnings added |
| `sdk/typescript/README.md` | modified | Real transport options and adapter APIs; packages marked unpublished |

## Diff Highlights

```ts
// gateway.ts: the method allowlist is the service descriptor, and JSON is proto3 JSON
const method = entry?.service.methods.find((m) => m.name === match![2]);
const call = method && (entry!.client as Record<string, unknown> | undefined)?.[method.localName];
...
request = fromJson(method.input, json);
return { status: 200, body: toJson(method.output, response) };
```

```ts
// transport.ts: connect-node owns the deadline, which maps to Code.DeadlineExceeded
createConnectTransport({ baseUrl, defaultTimeoutMs: config.timeout, httpVersion: "1.1", nodeOptions })
```

## Tests Added or Updated

- `middleware/test/transport.test.ts`: an RPC over HTTP/1.1 and over cleartext HTTP/2 to an in-process Connect
  plugin; an elapsed `timeout` rejects with `Code.DeadlineExceeded`
- `middleware/test/gateway.test.ts`: a proto3 JSON round trip including a `Timestamp`; an empty body as the default
  request; 404 for an unknown or unconfigured service, an unknown method, `Constructor`, or a non-API path; 405, 400
  for malformed JSON, 400 for an unknown field, 400 for `ValidationError`; a Connect `NotFound` → 404 with
  `code: not_found`; 413 above 1 MB; a pre-parsed `req.body`
- `framework-plugins/test/express.test.ts`: the middleware with and without `express.json()`; the router serves RPCs
  and passes other paths on
- `framework-plugins/test/fastify.test.ts`: `createFastifyPlugin`, `createFastifyRoutes`, and the gateway's 404
- `framework-plugins/test/nestjs.test.ts`: `FinFocusModule.register` and `registerAsync` on a real Nest app

These ran red first. The transport tests failed with `createNodeHttpTransport is not a function`. The Express router
failed on `Missing parameter name at index 1: *`, NestJS on `Missing parameter name at index 14`, and all three
Fastify tests timed out because the request hung.

## Local Verification

The toolchain is Node 24.15.0 and npm 11.12.1.

- `rm -rf packages/*/dist && npm ci`: 0 vulnerabilities
- `npm run build --workspaces`: 9/9 bundle outputs (cjs, esm, and dts × 3 packages)
- `npm test --workspaces`: client 78, middleware 17, framework-plugins 8, all passed
- `npm run lint --workspaces` (`tsc --noEmit`, including `test/`): clean
- Built `dist` loads in plain Node through both `import` and `require`

## Deviations from Assessment

- **Peer dependencies.** The assessment did not list this change. With the build finally running, tsup tried to
  bundle NestJS (`class-transformer`/`class-validator` unresolved), because it externalizes only `dependencies` and
  `peerDependencies`. The frameworks are now optional `peerDependencies`. That is the correct model for adapters,
  since the old `optionalDependencies` would have installed all three frameworks for every consumer.
  `@nestjs/platform-express` is a devDependency and an optional peer.
- **`createFastifyRoutes`** called the plugin without its required options argument (TS2554), a pre-existing error that
  the compile errors had been hiding. It now passes `{}`.
- **The README** also corrected APIs that never existed (`createExpressAdapter`, `FinFocusModule.forRoot`, adapters
  taking a plugin instance, `httpsClient`) and added the required `baseUrl` to `CostSourceClient` examples.
- **Express router.** The assessment proposed a regex route. `createExpressRouter` uses `router.use()` instead,
  because `createExpressMiddleware` already filters to POST `/finfocus.v1.*` and calls `next()` otherwise. A test
  covers the fall-through.
- **Review follow-ups** (Phase 4a `/code-review`): the Apache header was added to the rewritten `gateway.ts` and
  `transport.ts`, and the gateway's service map entry type was named `ServiceEntry`. An HTTP/2 transport test was
  added. CI now runs `npm run lint --workspaces`, since vitest does not type-check tests. Unused framework-plugins
  runtime dependencies were removed.
- **`assessment.md` line 5** was rewrapped to satisfy MD013. This is a whitespace-only change with no content change.

## Follow-ups

- `finfocus-framework-plugins`' index re-exports all three adapters, so importing it loads every framework.
  Per-framework subpath `exports` would let an Express user skip installing NestJS.
- Neither package has a publish workflow. Publishing needs a scoped name, like the client.
- The README "Browser Usage" example uses `startDate`/`endDate`/`totalCost`, which are not fields of the current
  messages. That is pre-existing and outside this fix.
- Renovate PRs #521 and #526 (NestJS) will need a lockfile rebase.
