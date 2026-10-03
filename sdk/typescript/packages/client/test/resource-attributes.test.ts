import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { create, fromBinary, fromJson, toBinary, toJson, type JsonObject } from "@bufbuild/protobuf";

import { CostSourceClient } from "../src/clients/cost-source.js";
import { ResourceDescriptorBuilder } from "../src/builders/resource-descriptor.js";
import {
  GetProjectedCostRequestSchema,
  ResourceDescriptorSchema,
} from "../src/generated/finfocus/v1/costsource_pb.js";

const attributes: JsonObject = {
  spec: {
    replicas: 3,
    template: {
      spec: {
        containers: [{ name: "web", resources: { requests: { cpu: "250m", memory: "128Mi" } } }],
      },
    },
  },
  paused: false,
  revisionHistoryLimit: null,
};

const endpoint = "https://plugin-k8s.example.com/finfocus.v1.CostSourceService/GetProjectedCost";
let capturedBody: unknown;

const server = setupServer(
  http.post(endpoint, async ({ request }) => {
    capturedBody = await request.json();
    return HttpResponse.json({ unitPrice: 0.01, currency: "USD", costPerMonth: 7.3 });
  }),
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("ResourceDescriptor.attributes", () => {
  it("round-trips through the binary and JSON encodings", () => {
    const descriptor = create(ResourceDescriptorSchema, {
      provider: "kubernetes",
      resourceType: "kubernetes:apps/v1:Deployment",
      attributes,
    });

    const fromWire = fromBinary(ResourceDescriptorSchema, toBinary(ResourceDescriptorSchema, descriptor));
    expect(fromWire.attributes).toEqual(attributes);

    const fromText = fromJson(ResourceDescriptorSchema, toJson(ResourceDescriptorSchema, descriptor));
    expect(fromText.attributes).toEqual(attributes);
  });

  it("is set by the builder and isolated from later caller changes", () => {
    const input: JsonObject = { spec: { replicas: 2 } };
    const builder = new ResourceDescriptorBuilder()
      .withProvider("kubernetes")
      .withResourceType("kubernetes:apps/v1:Deployment")
      .withAttributes(input);
    (input.spec as JsonObject).replicas = 9;

    expect(builder.build().attributes).toEqual({ spec: { replicas: 2 } });
  });

  it("is sent on the wire by the client unflattened", async () => {
    const client = new CostSourceClient({ baseUrl: "https://plugin-k8s.example.com" });
    const resource = new ResourceDescriptorBuilder()
      .withProvider("kubernetes")
      .withResourceType("kubernetes:apps/v1:Deployment")
      .withTags({ app: "web" })
      .withAttributes(attributes)
      .build();

    await client.getProjectedCost(create(GetProjectedCostRequestSchema, { resource }));

    expect(capturedBody).toMatchObject({ resource: { attributes, tags: { app: "web" } } });
  });
});
