# FinFocus TypeScript SDK

Official TypeScript/JavaScript client library for FinFocus cost source plugins. Provides type-safe
Connect RPC clients for browser and Node.js environments with comprehensive builder patterns and
error handling.

## Features

- **Type-Safe RPC Clients** - Full TypeScript support with generated types from protobuf definitions
- **Universal Runtime** - Works in browsers, Node.js, AWS Lambda, and edge environments
- **Builder Patterns** - Fluent APIs for constructing complex requests
- **Automatic Pagination** - AsyncIterator support for large result sets
- **Comprehensive Error Handling** - Validation errors and Connect RPC error handling
- **Framework Integration** - Ready-to-use adapters for Express, Fastify, and NestJS
- **Usage Sources** - `UsageSourceClient` for workload CPU/memory usage (`UsageSourceService.GetStats`)
- **Allocators** - `AllocatorClient` for splitting priced nodes across workloads (`AllocatorService.Allocate`)
- **Supplemental Datasets** - `SupplementalDatasetClient` for FOCUS contract commitments
  (`SupplementalDatasetService.GetContractCommitments`)

## Packages

This SDK is organized as a monorepo with three packages:

- **[@rshade/finfocus-client](./packages/client)** - Core client SDK (browser + Node.js)
- **[finfocus-middleware](./packages/middleware)** - Node.js HTTP transport and REST gateway
- **[finfocus-framework-plugins](./packages/framework-plugins)** - Express, Fastify, NestJS adapters

Only `@rshade/finfocus-client` is published. `finfocus-middleware` and `finfocus-framework-plugins`
are built and tested in this workspace but not yet released, so use them from a checkout of this repository.

## Installation

### Core Client (Browser & Node.js)

```bash
npm install @rshade/finfocus-client
```

### Node.js Middleware (Server-Side)

For server-side Node.js environments, the middleware package provides the Node.js HTTP transport (unpublished; see
[Packages](#packages)):

```bash
npm install @rshade/finfocus-client finfocus-middleware
```

### Framework Plugins (Optional)

For Express, Fastify, or NestJS integration (unpublished; see [Packages](#packages)). The frameworks are optional
peer dependencies, so install the one you use alongside the adapters:

```bash
npm install @rshade/finfocus-client finfocus-framework-plugins express
```

## Quick Start

### Browser Usage

```typescript
import { CostSourceClient, create } from "@rshade/finfocus-client";
import { GetActualCostRequestSchema } from "@rshade/finfocus-client";

// Create client with default browser transport
const client = new CostSourceClient({
  baseUrl: "https://plugin.example.com"
});

// Wrap in async function for CommonJS compatibility
(async () => {
  // Get plugin name
  const nameResp = await client.name();
  console.log(`Plugin: ${nameResp.name}`);

  // Fetch actual costs
  const request = create(GetActualCostRequestSchema, {
    resourceId: "i-1234567890abcdef0",
    startDate: { year: 2024, month: 1, day: 1 },
    endDate: { year: 2024, month: 1, day: 31 }
  });

  const response = await client.getActualCost(request);
  console.log(`Total cost: ${response.totalCost}`);
})();
```

**Note**: The example above uses top-level `await` inside an async IIFE for CommonJS compatibility.
If you're using ESM (`"type": "module"` in `package.json`), you can use top-level `await` directly.

### Node.js Server Usage

For server-side Node.js environments (Express, Lambda, etc.), use the Node.js HTTP transport:

```typescript
import { CostSourceClient } from "@rshade/finfocus-client";
import { createNodeTransport } from "finfocus-middleware";

const transport = createNodeTransport({
  baseUrl: "https://plugin.example.com",
  timeout: 30000 // 30 second deadline; elapsed calls reject with Code.DeadlineExceeded
});

const client = new CostSourceClient({ baseUrl: "https://plugin.example.com", transport });
```

**Why use Node transport?**

- Supports HTTP/1.1 (default) and HTTP/2
- Uses Node's `http`/`https` modules instead of the browser `fetch` API
- Applies a per-call deadline via `timeout`
- Accepts Node request options, such as a keep-alive `agent`

## Core API

### CostSourceClient

The main client for interacting with FinFocus cost source plugins:

```typescript
import { CostSourceClient, create, ValidationError } from "@rshade/finfocus-client";
import { GetProjectedCostRequestSchema, ResourceDescriptor } from "@rshade/finfocus-client";

const client = new CostSourceClient({
  baseUrl: "https://plugin.example.com"
});

// Get plugin info
const info = await client.getPluginInfo();
console.log(`Plugin: ${info.name} v${info.version}`);
console.log(`Providers: ${info.providers.join(", ")}`);

// Check resource support
const supports = await client.supports({
  resourceType: "aws:ec2:instance"
});
console.log(`Supported: ${supports.supported}`);

// Get projected costs
const resource = create(ResourceDescriptor, {
  resourceType: "aws:ec2:instance",
  instanceType: "t3.medium",
  region: "us-east-1"
});

const projectedReq = create(GetProjectedCostRequestSchema, {
  resource,
  months: 12
});

try {
  const projected = await client.getProjectedCost(projectedReq);
  console.log(`Projected annual cost: ${projected.projectedCost}`);
} catch (error) {
  if (error instanceof ValidationError) {
    console.error(`Validation error: ${error.message} (field: ${error.field})`);
  } else {
    throw error;
  }
}
```

### UsageSourceClient

Calls plugins that serve `UsageSourceService`, which reports per-workload and per-node CPU and
memory usage plus the priceable resources (nodes, control planes) that the workloads run on. See
[docs/usage-source.md](../../docs/usage-source.md) for the row and subject semantics.

```typescript
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  UsageSourceClient,
  GetStatsRequestSchema,
  StatsMode,
  SUBJECT_KIND,
  SUBJECT_NAMESPACE,
  KIND_WORKLOAD,
  METRIC_CPU_REQUEST,
} from "@rshade/finfocus-client";

const usage = new UsageSourceClient({ baseUrl: "https://usage-plugin.example.com" });

try {
  const resp = await usage.getStats(
    create(GetStatsRequestSchema, { scope: "prod-cluster", selector: { namespace: "payments" } }),
  );
  if (resp.mode === StatsMode.RUN_RATE) {
    for (const row of resp.rows) {
      if (row.subject[SUBJECT_KIND] === KIND_WORKLOAD && row.metric === METRIC_CPU_REQUEST) {
        console.log(`${row.subject[SUBJECT_NAMESPACE]}: ${row.amount} ${row.unit}`);
      }
    }
  }
  console.log(`Priceable nodes: ${resp.priceable.map((r) => r.id).join(", ")}`);
} catch (error) {
  if (error instanceof ConnectError && error.code === Code.PermissionDenied) {
    console.error(`Usage source lacks permissions: ${error.rawMessage}`);
  } else {
    throw error;
  }
}
```

Errors arrive as `ConnectError` with the code the source returned. The client does no request
validation. The `SUBJECT_*`, `KIND_*`, `METRIC_*`, and `UNIT_*` constants mirror the Go SDK.

### AllocatorClient

Calls plugins that serve `AllocatorService`, which divides priced nodes and control planes across
workloads and returns workload, idle, and cluster rows. See
[docs/allocator.md](../../docs/allocator.md) for the invariants and policy rules.

```typescript
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  AllocatorClient,
  AllocateRequestSchema,
  UsageSourceClient,
  GetStatsRequestSchema,
  SUBJECT_KIND,
  SUBJECT_NODE,
  KIND_IDLE,
} from "@rshade/finfocus-client";

const usage = new UsageSourceClient({ baseUrl: "https://usage-plugin.example.com" });
const allocator = new AllocatorClient({ baseUrl: "https://allocator-plugin.example.com" });

try {
  const stats = await usage.getStats(create(GetStatsRequestSchema, { scope: "prod-cluster" }));
  const resp = await allocator.allocate(
    create(AllocateRequestSchema, {
      usage: stats.rows,
      priced: [{ resource: { id: "ip-10-0-1-5", tags: { kind: "node" } }, cost: 0.096, currency: "USD", priced: true }],
      policyJson: new TextEncoder().encode('{"node_split":{"cpu_weight":0.6}}'),
    }),
  );
  for (const row of resp.rows) {
    if (row.subject[SUBJECT_KIND] === KIND_IDLE) {
      console.log(`idle on ${row.subject[SUBJECT_NODE]}: ${row.totalCost} ${row.currency}`);
    }
  }
  console.log(`policy ${new TextDecoder().decode(resp.effectivePolicyJson)} (${resp.policyDigest})`);
} catch (error) {
  if (error instanceof ConnectError && error.code === Code.InvalidArgument) {
    console.error(`Allocator rejected the request or policy: ${error.rawMessage}`);
  } else {
    throw error;
  }
}
```

Errors arrive as `ConnectError` with the code the allocator returned; a bad policy is
`Code.InvalidArgument` naming the field's path. The client does not validate requests or check
conservation.

### RecommendationScorerClient

Calls plugins that serve `RecommendationScorerService` and advertise
`PluginCapability.RECOMMENDATION_SCORING`. See
[docs/recommendation-scoring.md](../../docs/recommendation-scoring.md) for signals and trust rules.
Scores are ranking signals and never approval to act.

```typescript
import { create } from "@bufbuild/protobuf";
import {
  RecommendationScorerClient,
  ScoreRecommendationsRequestSchema,
  ScoreSignal,
} from "@rshade/finfocus-client";

const scorer = new RecommendationScorerClient({ baseUrl: "https://scorer-plugin.example.com" });
const resp = await scorer.scoreRecommendations(
  create(ScoreRecommendationsRequestSchema, {
    recommendations,
    signals: [ScoreSignal.RISK],
  }),
);
resp.results.forEach((result, i) => {
  if (result.result.case === "scores") {
    console.log(recommendations[i].id, result.result.value.risk);
  }
});
```

Results are index-aligned with the request. Errors arrive as `ConnectError` with the scorer's code.
The client does not validate requests or responses.

### SupplementalDatasetClient

Calls plugins that serve `SupplementalDatasetService` and advertise
`PluginCapability.CONTRACT_COMMITMENTS`. See
[docs/supplemental-datasets.md](../../docs/supplemental-datasets.md) for window matching and
pagination.

```typescript
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  GetContractCommitmentsRequestSchema,
  SupplementalDatasetClient,
} from "@rshade/finfocus-client";

const client = new SupplementalDatasetClient({ baseUrl: "https://billing-plugin.example.com" });

// One page
const page = await client.getContractCommitments(
  create(GetContractCommitmentsRequestSchema, { pageSize: 100 }),
);

// Every commitment active in June 2025, across all pages
const request = create(GetContractCommitmentsRequestSchema, {
  start: timestampFromDate(new Date("2025-06-01T00:00:00Z")),
  end: timestampFromDate(new Date("2025-07-01T00:00:00Z")),
});
for await (const commitment of client.contractCommitments(request)) {
  console.log(commitment.contractCommitmentId, commitment.contractCommitmentCost);
}
```

`contractCommitments`, `billingPeriods`, and `invoiceDetails` clone the request. A missing or zero
page size becomes 50, and a negative page size is sent unchanged. The iterator throws after 10
consecutive empty pages that still carry a token, and when a non-empty page token repeats. Errors
arrive as `ConnectError` with the plugin's code.

### Pagination

Iterate through large result sets using the async iterator pattern:

```typescript
import { recommendationsIterator, create } from "@rshade/finfocus-client";
import { GetRecommendationsRequestSchema } from "@rshade/finfocus-client";

const request = create(GetRecommendationsRequestSchema, {
  filter: {
    priority: RecommendationPriority.HIGH
  },
  pageSize: 100 // Optional: defaults to server-defined page size
});

// Automatically handles pagination across all pages
for await (const rec of recommendationsIterator(client, request)) {
  console.log(`${rec.id}: ${rec.description}`);
  console.log(`Estimated savings: $${rec.estimatedMonthlySavings}/month`);
}
```

**Resume pagination** from a specific page token:

```typescript
// Resume from a previous page
const resumeRequest = create(GetRecommendationsRequestSchema, {
  filter: { /* same filters */ },
  pageToken: "abc123" // Token from previous response
});

// Continue iterating from that point
for await (const rec of recommendationsIterator(client, resumeRequest)) {
  console.log(rec.description);
}
```

**When to use pagination:**

- Fetching large recommendation lists (100+ items)
- Processing results incrementally (streaming)
- Implementing infinite scroll or load-more UIs
- Resuming interrupted queries

### Builder Patterns

Construct complex requests using fluent builder APIs:

#### ResourceDescriptorBuilder

```typescript
import { ResourceDescriptorBuilder } from "@rshade/finfocus-client";

const resource = new ResourceDescriptorBuilder()
  .withResourceType("aws:ec2:instance")
  .withInstanceType("t3.medium")
  .withRegion("us-east-1")
  .withAvailabilityZone("us-east-1a")
  .withTags({ Environment: "production", Team: "platform" })
  .build();
```

#### RecommendationFilterBuilder

```typescript
import { RecommendationFilterBuilder, RecommendationPriority } from "@rshade/finfocus-client";

const filter = new RecommendationFilterBuilder()
  .withPriority(RecommendationPriority.HIGH)
  .withCategory(RecommendationCategory.COST_OPTIMIZATION)
  .withResourceTypes(["aws:ec2:instance", "aws:rds:db-instance"])
  .build();
```

#### FocusRecordBuilder

```typescript
import { FocusRecordBuilder } from "@rshade/finfocus-client";

const record = new FocusRecordBuilder()
  .withBillingAccountId("123456789012")
  .withBillingPeriodStart({ year: 2024, month: 1, day: 1 })
  .withBillingPeriodEnd({ year: 2024, month: 1, day: 31 })
  .withChargeCategory(FocusChargeCategory.USAGE)
  .withChargeClass(FocusChargeClass.REGULAR)
  .withResourceId("i-1234567890abcdef0")
  .withServiceName("Amazon Elastic Compute Cloud")
  .withBilledCost(150.25)
  .build();
```

## Error Handling

### Comprehensive Error Handling Pattern

The SDK provides two types of errors to handle:

```typescript
import { CostSourceClient, ValidationError } from "@rshade/finfocus-client";
import { ConnectError, Code } from "@connectrpc/connect";

const client = new CostSourceClient({
  baseUrl: "https://plugin.example.com"
});

try {
  await client.dismissRecommendation({
    recommendationId: "rec-123",
    reason: "Already implemented"
  });
} catch (error) {
  if (error instanceof ValidationError) {
    // Client-side validation failure (before request is sent)
    console.error("Validation error:", error.message);
    console.error("Field:", error.field);
    console.error("Code:", error.code);
  } else if (error instanceof ConnectError) {
    // Server or transport error (from Connect RPC)
    switch (error.code) {
      case Code.InvalidArgument:
        console.error("Server validation error:", error.message);
        break;
      case Code.NotFound:
        console.error("Resource not found:", error.message);
        break;
      case Code.DeadlineExceeded:
        console.error("Request timeout");
        break;
      case Code.Unauthenticated:
        console.error("Authentication required");
        break;
      case Code.PermissionDenied:
        console.error("Permission denied");
        break;
      case Code.Unavailable:
        console.error("Service unavailable - retry later");
        break;
      default:
        console.error(`RPC error [${error.code}]: ${error.message}`);
    }

    // Access error metadata
    if (error.metadata) {
      console.error("Error metadata:", error.metadata);
    }
  } else {
    // Unknown error - re-throw
    throw error;
  }
}
```

### ValidationError

Client-side validation errors thrown **before** making the RPC call:

- `message` - Human-readable error description
- `field` - The field that failed validation (optional)
- `code` - Machine-readable error code (optional)

### ConnectError

Server-side errors from the Connect RPC protocol:

- `code` - gRPC status code (use `Code` enum for matching)
- `message` - Error message from the server
- `metadata` - Additional error context (headers)
- `rawMessage` - Original error message

**Common Connect error codes:**

- `InvalidArgument` - Server rejected request parameters
- `NotFound` - Requested resource doesn't exist
- `DeadlineExceeded` - Request timeout
- `Unauthenticated` - Authentication required
- `PermissionDenied` - Insufficient permissions
- `Unavailable` - Service temporarily unavailable (retry)
- `Internal` - Server internal error

## TypeScript Best Practices

### Type-Safe Enums

The SDK exports TypeScript enums for all protobuf enumerations, providing autocomplete and type safety:

```typescript
import {
  RecommendationPriority,
  RecommendationCategory,
  RecommendationActionType,
  FocusServiceCategory,
  FocusChargeCategory,
  PluginCapability
} from "@rshade/finfocus-client";

// Type-safe enum values with IDE autocomplete
const filter = new RecommendationFilterBuilder()
  .withPriority(RecommendationPriority.HIGH)  // Type-safe
  .withCategory(RecommendationCategory.COST_OPTIMIZATION)
  .withActionType(RecommendationActionType.RESIZE)
  .build();

// Service category classification
const category = FocusServiceCategory.COMPUTE;  // 1
const categoryName = FocusServiceCategory[category];  // "COMPUTE"

// Check plugin capabilities
const hasRecommendations = info.capabilities.includes(
  PluginCapability.PLUGIN_CAPABILITY_RECOMMENDATIONS
);
```

**Available enums:**

- `FocusServiceCategory` - COMPUTE, STORAGE, NETWORK, DATABASE, etc.
- `FocusChargeCategory` - USAGE, PURCHASE, CREDIT, TAX, REFUND, ADJUSTMENT
- `FocusPricingCategory` - STANDARD, COMMITTED, DYNAMIC, OTHER
- `FocusChargeClass` - REGULAR, CORRECTION
- `FocusChargeFrequency` - ONE_TIME, RECURRING, USAGE_BASED
- `RecommendationPriority` - CRITICAL, HIGH, MEDIUM, LOW
- `RecommendationCategory` - COST_OPTIMIZATION, PERFORMANCE, SECURITY, etc.
- `RecommendationActionType` - RESIZE, TERMINATE, MIGRATE, SCHEDULE, etc.
- `PluginCapability` - Feature flags for plugin capabilities
- `GrowthType` - NONE, LINEAR, EXPONENTIAL (for cost projections)
- `FieldSupportStatus` - SUPPORTED, UNSUPPORTED, CONDITIONAL, DYNAMIC

### Type Inference

Let TypeScript infer types from the `create` helper:

```typescript
import { create } from "@rshade/finfocus-client";
import { GetActualCostRequestSchema } from "@rshade/finfocus-client";

// Type is inferred as GetActualCostRequest
const request = create(GetActualCostRequestSchema, {
  resourceId: "i-1234567890abcdef0",
  startDate: { year: 2024, month: 1, day: 1 },
  endDate: { year: 2024, month: 1, day: 31 }
});
```

### Null Safety

Protobuf optional fields are represented as TypeScript optional properties:

```typescript
// Check optional fields
if (response.totalCost !== undefined) {
  console.log(`Cost: $${response.totalCost}`);
}

// Use optional chaining
console.log(`Cost: $${response.totalCost ?? 0}`);
```

## Transport Configuration

### Browser Transport (Default)

The default transport uses `fetch` API and works in all modern browsers:

```typescript
const client = new CostSourceClient({
  baseUrl: "https://plugin.example.com"
});
```

### Node.js Transport

For server-side Node.js environments, use the Node.js HTTP transport from `finfocus-middleware`:

```typescript
import { createNodeTransport } from "finfocus-middleware";
import * as https from "https";

const transport = createNodeTransport({
  baseUrl: "https://plugin.example.com",
  timeout: 30000,  // 30 second deadline

  // HTTP/1.1 request options, e.g. a keep-alive agent for connection pooling
  nodeOptions: {
    agent: new https.Agent({ keepAlive: true, maxSockets: 50 })
  }
});

const client = new CostSourceClient({ baseUrl: "https://plugin.example.com", transport });
```

Set `httpVersion: "2"` to use HTTP/2 instead; `nodeOptions` applies to HTTP/1.1 only.

**Node transport features:**

- HTTP/1.1 (default) or HTTP/2
- Connection pooling and keep-alive through `nodeOptions.agent`
- Per-call deadline via `timeout`, surfaced as `Code.DeadlineExceeded`
- Works in AWS Lambda, Google Cloud Functions, etc.
- No dependency on browser `fetch` API

### Custom Transport

Implement custom transport for advanced use cases:

```typescript
import { Transport } from "@connectrpc/connect";

// Custom transport with retry logic, auth, etc.
const customTransport: Transport = {
  // Implementation details...
};

const client = new CostSourceClient({ transport: customTransport });
```

## Testing

### Unit Testing with Vitest

```typescript
import { describe, it, expect } from "vitest";
import { CostSourceClient, ValidationError, create } from "@rshade/finfocus-client";
import { GetActualCostRequestSchema } from "@rshade/finfocus-client";

describe("CostSourceClient", () => {
  it("validates required fields", async () => {
    const client = new CostSourceClient({ baseUrl: "http://test" });

    // Missing required resourceId should throw ValidationError
    await expect(
      client.getActualCost({} as any)
    ).rejects.toThrow(ValidationError);
  });

  it("creates valid requests with create helper", () => {
    const request = create(GetActualCostRequestSchema, {
      resourceId: "i-1234567890abcdef0",
      startDate: { year: 2024, month: 1, day: 1 },
      endDate: { year: 2024, month: 1, day: 31 }
    });

    expect(request.resourceId).toBe("i-1234567890abcdef0");
    expect(request.startDate).toEqual({ year: 2024, month: 1, day: 1 });
  });
});
```

### Mocking with MSW (Mock Service Worker)

```typescript
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { CostSourceClient } from "@rshade/finfocus-client";

// Mock server
const server = setupServer(
  http.post("https://plugin.example.com/finfocus.v1.CostSourceService/GetActualCost", () => {
    return HttpResponse.json({
      totalCost: 150.25,
      currency: "USD",
      records: []
    });
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("CostSourceClient Integration", () => {
  it("fetches actual costs", async () => {
    const client = new CostSourceClient({
      baseUrl: "https://plugin.example.com"
    });

    const response = await client.getActualCost({
      resourceId: "i-1234567890abcdef0",
      startDate: { year: 2024, month: 1, day: 1 },
      endDate: { year: 2024, month: 1, day: 31 }
    });

    expect(response.totalCost).toBe(150.25);
    expect(response.currency).toBe("USD");
  });
});
```

### Testing Pagination

```typescript
import { describe, it, expect } from "vitest";
import { recommendationsIterator, create } from "@rshade/finfocus-client";
import { GetRecommendationsRequestSchema } from "@rshade/finfocus-client";

describe("Pagination", () => {
  it("iterates through all pages", async () => {
    const client = new CostSourceClient({ baseUrl: "http://test" });
    const request = create(GetRecommendationsRequestSchema, {});

    const recommendations = [];
    for await (const rec of recommendationsIterator(client, request)) {
      recommendations.push(rec);
    }

    expect(recommendations.length).toBeGreaterThan(0);
  });

  it("resumes from page token", async () => {
    const client = new CostSourceClient({ baseUrl: "http://test" });

    // Get first page
    const page1 = await client.getRecommendations({});
    const token = page1.nextPageToken;

    // Resume from token
    const request = create(GetRecommendationsRequestSchema, {
      pageToken: token
    });

    const recommendations = [];
    for await (const rec of recommendationsIterator(client, request)) {
      recommendations.push(rec);
    }

    expect(recommendations.length).toBeGreaterThan(0);
  });
});
```

### Testing Error Handling

```typescript
import { describe, it, expect } from "vitest";
import { CostSourceClient, ValidationError } from "@rshade/finfocus-client";
import { ConnectError, Code } from "@connectrpc/connect";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

const server = setupServer(
  http.post("*/DismissRecommendation", () => {
    return HttpResponse.json(
      { code: "invalid_argument", message: "Recommendation not found" },
      { status: 400 }
    );
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("Error Handling", () => {
  it("throws ValidationError for missing required fields", async () => {
    const client = new CostSourceClient({ baseUrl: "http://test" });

    await expect(
      client.dismissRecommendation({ recommendationId: "" })
    ).rejects.toThrow(ValidationError);
  });

  it("handles ConnectError from server", async () => {
    const client = new CostSourceClient({ baseUrl: "http://test" });

    try {
      await client.dismissRecommendation({ recommendationId: "rec-123" });
      expect.fail("Should have thrown ConnectError");
    } catch (error) {
      expect(error).toBeInstanceOf(ConnectError);
      if (error instanceof ConnectError) {
        expect(error.code).toBe(Code.InvalidArgument);
      }
    }
  });
});
```

## API Reference

### Client Methods

#### `name(): Promise<NameResponse>`

Get the plugin name.

#### `supports(req: SupportsRequest): Promise<SupportsResponse>`

Check if a resource type is supported.

#### `getActualCost(req: GetActualCostRequest): Promise<GetActualCostResponse>`

Fetch actual historical costs for a resource.

**Validation**: Requires `resourceId` or `arn`.

#### `getProjectedCost(req: GetProjectedCostRequest): Promise<GetProjectedCostResponse>`

Get projected future costs for a resource configuration.

**Validation**: Requires `resource`.

#### `getPricingSpec(req?: GetPricingSpecRequest): Promise<GetPricingSpecResponse>`

Retrieve the plugin's pricing specification (JSON schema).

#### `estimateCost(req: EstimateCostRequest): Promise<EstimateCostResponse>`

Estimate costs for hypothetical resource configurations.

#### `getRecommendations(req?: GetRecommendationsRequest): Promise<GetRecommendationsResponse>`

Fetch cost optimization recommendations with optional filtering and pagination.

#### `dismissRecommendation(req: DismissRecommendationRequest): Promise<DismissRecommendationResponse>`

Dismiss a recommendation with a reason.

**Validation**: Requires `recommendationId`.

#### `getBudgets(req?: GetBudgetsRequest): Promise<GetBudgetsResponse>`

Retrieve budget information and status.

#### `getPluginInfo(req?: GetPluginInfoRequest): Promise<GetPluginInfoResponse>`

Get plugin metadata (name, version, spec version, providers, capabilities).

#### `dryRun(req?: DryRunRequest): Promise<DryRunResponse>`

Query plugin field mapping capabilities without fetching cost data.

## Framework Integration

The framework adapters mount a REST gateway that proxies JSON requests to a FinFocus plugin through the clients you
supply. Each RPC is served at `POST /finfocus.v1.<Service>/<Method>`, for example
`POST /finfocus.v1.CostSourceService/GetActualCost`. Bodies use the proto3 JSON mapping: `Timestamp` fields are
RFC 3339 strings and 64-bit integers are strings. Plugin errors map to the matching HTTP status with a body of
`{ "error": "...", "code": "not_found" }`. Messages for `internal`, `unknown`, `unavailable`, and `data_loss`
errors are replaced with a generic one (the original is logged with `console.error`). Request bodies over 1 MiB get
`413`, and a body not fully received within 30 seconds gets `408`; both close the connection.

> **Security:** The gateway has no authentication of its own. Mount it behind your own authentication and
> authorization.

### Express

```typescript
import express from "express";
import { CostSourceClient } from "@rshade/finfocus-client";
import { createExpressRouter } from "finfocus-framework-plugins";

const app = express();
const costSourceClient = new CostSourceClient({ baseUrl: "https://plugin.example.com" });

// Works with or without express.json(); other paths fall through to later routes.
app.use(createExpressRouter({ costSourceClient }));

app.listen(3000);
```

### Fastify

```typescript
import Fastify from "fastify";
import { CostSourceClient } from "@rshade/finfocus-client";
import { createFastifyPlugin } from "finfocus-framework-plugins";

const fastify = Fastify();
const costSourceClient = new CostSourceClient({ baseUrl: "https://plugin.example.com" });

await fastify.register(createFastifyPlugin({ costSourceClient }));

await fastify.listen({ port: 3000 });
```

### NestJS

Requires `@nestjs/platform-express`.

```typescript
import { Module } from "@nestjs/common";
import { CostSourceClient } from "@rshade/finfocus-client";
import { FinFocusModule } from "finfocus-framework-plugins";

@Module({
  imports: [
    FinFocusModule.register({
      costSourceClient: new CostSourceClient({ baseUrl: "https://plugin.example.com" })
    })
  ]
})
export class AppModule {}
```

Use `FinFocusModule.registerAsync({ useFactory, inject })` when the clients depend on other providers.

## License

MIT

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development setup and contribution guidelines.

## Support

- Documentation: <https://github.com/rshade/finfocus-spec>
- Issues: <https://github.com/rshade/finfocus-spec/issues>
- Discussions: <https://github.com/rshade/finfocus-spec/discussions>
