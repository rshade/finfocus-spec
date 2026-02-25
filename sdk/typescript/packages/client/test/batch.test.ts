import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { create } from "@bufbuild/protobuf";

import { CostSourceClient } from "../src/clients/cost-source.js";
import { ValidationError } from "../src/errors/validation-error.js";
import {
  BatchCostRequestSchema,
  GetPluginInfoResponseSchema,
} from "../src/generated/finfocus/v1/costsource_pb.js";
import { CostQueryType, PluginCapability } from "../src/generated/finfocus/v1/enums_pb.js";
import { isBatchSupported, MAX_BATCH_SIZE } from "../src/utils/batch.js";

const batchEndpoint = "https://plugin-aws.example.com/finfocus.v1.CostSourceService/BatchCost";

const server = setupServer(
  http.post(batchEndpoint, () => {
    return HttpResponse.json({
      results: [
        {
          resource: { provider: "aws", resourceType: "ec2" },
          costData: {
            estimate: {
              currency: "USD",
              costMonthly: 120.5,
            },
          },
        },
      ],
      maxBatchSize: 100,
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("Batch client", () => {
  const client = new CostSourceClient({
    baseUrl: "https://plugin-aws.example.com",
  });

  it("performs a basic batchCost call", async () => {
    const req = create(BatchCostRequestSchema, {
      queryType: CostQueryType.ESTIMATE,
      resources: [{ provider: "aws", resourceType: "ec2" }],
    });

    const resp = await client.batchCost(req);
    expect(resp.results.length).toBe(1);
    expect(resp.results[0].result.case).toBe("costData");
    expect(resp.results[0].result.value.data.case).toBe("estimate");
    expect(resp.results[0].result.value.data.value.costMonthly).toBe(120.5);
  });

  it("supports empty batch responses", async () => {
    server.use(
      http.post(batchEndpoint, () =>
        HttpResponse.json({
          results: [],
          maxBatchSize: 100,
        })),
    );

    const req = create(BatchCostRequestSchema, {
      queryType: CostQueryType.ESTIMATE,
      resources: [],
    });

    const resp = await client.batchCost(req);
    expect(resp.results).toEqual([]);
    expect(resp.maxBatchSize).toBe(100);
  });

  it("parses partial-failure batch responses", async () => {
    server.use(
      http.post(batchEndpoint, () =>
        HttpResponse.json({
          results: [
            {
              resource: { provider: "aws", resourceType: "ec2" },
              costData: {
                estimate: {
                  currency: "USD",
                  costMonthly: 42,
                },
              },
            },
            {
              resource: { provider: "aws", resourceType: "unsupported_resource" },
              error: {
                code: 12,
                message: "resource type unsupported",
                resourceTypeUnsupported: true,
              },
            },
          ],
          maxBatchSize: 100,
        })),
    );

    const req = create(BatchCostRequestSchema, {
      queryType: CostQueryType.ESTIMATE,
      resources: [
        { provider: "aws", resourceType: "ec2" },
        { provider: "aws", resourceType: "unsupported_resource" },
      ],
    });

    const resp = await client.batchCost(req);
    expect(resp.results.length).toBe(2);
    expect(resp.results[0].result.case).toBe("costData");
    expect(resp.results[1].result.case).toBe("error");
    expect(resp.results[1].result.value.code).toBe(12);
    expect(resp.results[1].result.value.resourceTypeUnsupported).toBe(true);
  });

  it("validates maximum batch size before calling the RPC", async () => {
    const req = create(BatchCostRequestSchema, {
      queryType: CostQueryType.ESTIMATE,
      resources: Array.from({ length: MAX_BATCH_SIZE + 1 }, () => ({
        provider: "aws",
        resourceType: "ec2",
      })),
    });

    await expect(client.batchCost(req)).rejects.toThrow(ValidationError);
  });
});

describe("Batch utilities", () => {
  it("detects batch support from enum capabilities", () => {
    const info = create(GetPluginInfoResponseSchema, {
      name: "batch-plugin",
      version: "1.0.0",
      specVersion: "v0.0.0",
      capabilities: [PluginCapability.BATCH_COST],
    });

    expect(isBatchSupported(info)).toBe(true);
  });

  it("detects batch support from legacy metadata", () => {
    expect(
      isBatchSupported({
        metadata: {
          supports_batch_cost: "true",
        },
      }),
    ).toBe(true);
  });

  it("returns false when batch support is absent", () => {
    expect(isBatchSupported({ capabilities: [PluginCapability.DRY_RUN] })).toBe(false);
  });
});
