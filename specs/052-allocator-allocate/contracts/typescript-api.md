# TypeScript API: Allocator

Package `sdk/typescript/packages/client`. It mirrors `UsageSourceClient` from 051.

## Generated

`src/generated/finfocus/v1/allocation_pb.ts` (from `make generate`) provides `AllocatorService`,
`AllocateRequest`, `AllocateResponse`, `PricedResource`, and `AllocationRow`. `bytes` fields
(`policyJson`, `effectivePolicyJson`) are `Uint8Array`.
`PluginCapability.ALLOCATION` (15) is added to the generated enums.

## Client wrapper (`src/clients/allocator.ts`)

```ts
/**
 * Client for plugins that serve AllocatorService.
 *
 * Errors propagate as ConnectError with the code the allocator returned
 * (for example Code.InvalidArgument for a bad policy). Requests are not
 * validated client-side, and conservation is not checked client-side.
 */
export class AllocatorClient {
  constructor(config: ClientConfig);

  /** Divides priced resources across workloads according to the policy. */
  allocate(request: AllocateRequest): Promise<AllocateResponse>;
}
```

## Exports (`src/index.ts`)

```ts
export * from "./generated/finfocus/v1/allocation_pb.js";
export { AllocatorClient } from "./clients/allocator.js";
```

`KIND_IDLE` (`"__idle__"`) and `KIND_CLUSTER` (`"__cluster__"`) already exist in
`utils/usage-subjects.ts` and are reused unchanged.

## Tests (`test/allocator.test.ts`, vitest + msw)

- An allocate request round-trips. Rows, effective policy bytes, digest, and warnings arrive intact.
- An allocator error surfaces as `ConnectError` with `Code.InvalidArgument` and the original message.
