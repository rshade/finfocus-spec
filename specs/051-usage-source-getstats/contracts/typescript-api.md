# TypeScript SDK Contract: Usage Source

Package: `sdk/typescript/packages/client`.

## Generated

`src/generated/finfocus/v1/usage_pb.ts` (buf `es` plugin) exports `UsageSourceService`,
`GetStatsRequest`/`GetStatsRequestSchema`, `GetStatsResponse`/`GetStatsResponseSchema`,
`UsageRow`/`UsageRowSchema`, and `StatsMode`. `enums_pb.ts` gains `PluginCapability.USAGE_STATS`.
`src/index.ts` adds `export * from "./generated/finfocus/v1/usage_pb.js";`.

## Client wrapper (`src/clients/usage-source.ts`)

```ts
export class UsageSourceClient {
  constructor(config: ClientConfig); // same ClientConfig as RegistryClient (baseUrl, optional transport)
  getStats(request: GetStatsRequest): Promise<GetStatsResponse>;
}
```

The wrapper uses `createClient(UsageSourceService, transport)`. Errors propagate as `ConnectError`
(no custom wrapping). There is no client-side validation (research R9).

## Constants (`src/utils/usage-subjects.ts`)

These mirror the Go constants in [go-sdk-api.md](./go-sdk-api.md) exactly: `SUBJECT_CLUSTER`,
`SUBJECT_NAMESPACE`, `SUBJECT_CONTROLLER_KIND`, `SUBJECT_CONTROLLER`, `SUBJECT_POD`, `SUBJECT_NODE`,
`SUBJECT_KIND`, `SUBJECT_LABEL_PREFIX`, `KIND_WORKLOAD`, `KIND_NODE`, `KIND_IDLE`, `KIND_CLUSTER`,
`METRIC_CPU_REQUEST`, `METRIC_MEM_REQUEST`, `METRIC_CPU_ALLOCATABLE`, `METRIC_MEM_ALLOCATABLE`,
`METRIC_CPU_USAGE`, `METRIC_MEM_USAGE`, `UNIT_CORE`, `UNIT_GIB`, `UNIT_CORE_HOURS`, `UNIT_GIB_HOURS`.
They are exported from `src/index.ts`.

## Wire endpoint

`POST {baseUrl}/finfocus.v1.UsageSourceService/GetStats` (Connect protocol, JSON or binary).
