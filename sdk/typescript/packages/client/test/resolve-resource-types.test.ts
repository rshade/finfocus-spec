import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { create } from "@bufbuild/protobuf";

import { CostSourceClient } from "../src/clients/cost-source.js";
import { ValidationError } from "../src/errors/validation-error.js";
import { ResolveResourceTypesRequestSchema } from "../src/generated/finfocus/v1/costsource_pb.js";
import { SourceFormat } from "../src/generated/finfocus/v1/enums_pb.js";
import { MAX_SOURCE_TYPES } from "../src/utils/batch.js";

const resolveEndpoint =
  "https://plugin-aws.example.com/finfocus.v1.CostSourceService/ResolveResourceTypes";

const server = setupServer(
  http.post(resolveEndpoint, () => {
    return HttpResponse.json({
      mappings: {
        aws_instance: {
          pulumiToken: "aws:ec2/instance:Instance",
          supported: true,
        },
      },
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("ResolveResourceTypes client", () => {
  const client = new CostSourceClient({
    baseUrl: "https://plugin-aws.example.com",
  });

  it("performs a basic resolveResourceTypes call", async () => {
    const req = create(ResolveResourceTypesRequestSchema, {
      sourceFormat: SourceFormat.TERRAFORM,
      sourceTypes: ["aws_instance"],
    });

    const resp = await client.resolveResourceTypes(req);
    expect(resp.mappings["aws_instance"].pulumiToken).toBe("aws:ec2/instance:Instance");
    expect(resp.mappings["aws_instance"].supported).toBe(true);
  });

  it("validates maximum source_types size before calling the RPC", async () => {
    const req = create(ResolveResourceTypesRequestSchema, {
      sourceFormat: SourceFormat.TERRAFORM,
      sourceTypes: Array.from({ length: MAX_SOURCE_TYPES + 1 }, (_, i) => `aws_resource_${i}`),
    });

    await expect(client.resolveResourceTypes(req)).rejects.toThrow(ValidationError);
  });

  it("allows a request at the size limit", async () => {
    const req = create(ResolveResourceTypesRequestSchema, {
      sourceFormat: SourceFormat.TERRAFORM,
      sourceTypes: Array.from({ length: MAX_SOURCE_TYPES }, (_, i) => `aws_resource_${i}`),
    });

    await expect(client.resolveResourceTypes(req)).resolves.toBeDefined();
  });
});
