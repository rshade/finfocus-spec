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

## Packages

This SDK is organized as a monorepo with three packages:

- **[@rshade/finfocus-client](./packages/client)** - Core client SDK (browser + Node.js)
- **[finfocus-middleware](./packages/middleware)** - Node.js HTTP transport and REST gateway
- **[finfocus-framework-plugins](./packages/framework-plugins)** - Express, Fastify, NestJS adapters

## Installation

### Core Client (Browser & Node.js)

```bash
npm install @rshade/finfocus-client
```

### Node.js Middleware (Server-Side)

For server-side Node.js environments, install the middleware package to access the Node.js HTTP transport:

```bash
npm install @rshade/finfocus-client finfocus-middleware
```

### Framework Plugins (Optional)

For Express, Fastify, or NestJS integration:

```bash
npm install @rshade/finfocus-client finfocus-framework-plugins
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
  timeout: 30000 // 30 second timeout
});

const client = new CostSourceClient({ transport });
```

**Why use Node transport?**

- Supports HTTP/2 and connection pooling
- Required for environments without browser `fetch` API
- Enables custom timeout and retry logic
- Better performance for server-to-server communication

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
  timeout: 30000,  // 30 second timeout

  // Custom HTTPS agent for connection pooling
  httpsClient: new https.Agent({
    keepAlive: true,
    maxSockets: 50
  })
});

const client = new CostSourceClient({ transport });
```

**Node transport features:**

- HTTP/2 support (when server supports it)
- Connection pooling and keep-alive
- Custom timeout handling
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

### Express

```typescript
import express from "express";
import { createExpressAdapter } from "finfocus-framework-plugins";
import { MyPlugin } from "./my-plugin.js";

const app = express();

app.use("/finfocus", createExpressAdapter(new MyPlugin()));

app.listen(3000);
```

### Fastify

```typescript
import Fastify from "fastify";
import { createFastifyPlugin } from "finfocus-framework-plugins";
import { MyPlugin } from "./my-plugin.js";

const fastify = Fastify();

await fastify.register(createFastifyPlugin(new MyPlugin()), {
  prefix: "/finfocus"
});

await fastify.listen({ port: 3000 });
```

### NestJS

```typescript
import { Module } from "@nestjs/common";
import { FinFocusModule } from "finfocus-framework-plugins";
import { MyPlugin } from "./my-plugin.js";

@Module({
  imports: [
    FinFocusModule.forRoot({
      plugin: new MyPlugin(),
      path: "/finfocus"
    })
  ]
})
export class AppModule {}
```

## License

MIT

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development setup and contribution guidelines.

## Support

- Documentation: <https://github.com/rshade/finfocus-spec>
- Issues: <https://github.com/rshade/finfocus-spec/issues>
- Discussions: <https://github.com/rshade/finfocus-spec/discussions>
