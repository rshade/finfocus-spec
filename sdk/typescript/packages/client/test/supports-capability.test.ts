// Copyright 2026 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { create, fromJson, toJson } from "@bufbuild/protobuf";

import { CostSourceClient } from "../src/clients/cost-source.js";
import { SupportsRequestSchema } from "../src/generated/finfocus/v1/costsource_pb.js";
import { PluginCapability } from "../src/generated/finfocus/v1/enums_pb.js";

const supportsEndpoint = "https://plugin-aws.example.com/finfocus.v1.CostSourceService/Supports";

let captured: Record<string, unknown> | undefined;

const server = setupServer(
  http.post(supportsEndpoint, async ({ request }) => {
    captured = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({ supported: false, reason: "projected cost not priced" });
  }),
);

beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  captured = undefined;
});
afterAll(() => server.close());

describe("Supports capability", () => {
  it("defaults to an unspecified capability", () => {
    const req = create(SupportsRequestSchema);
    expect(req.capability).toBe(PluginCapability.UNSPECIFIED);
  });

  it("round trips a capability through proto3 JSON", () => {
    const req = create(SupportsRequestSchema, { capability: PluginCapability.PROJECTED_COSTS });
    const decoded = fromJson(SupportsRequestSchema, toJson(SupportsRequestSchema, req));
    expect(decoded.capability).toBe(PluginCapability.PROJECTED_COSTS);
  });

  it("forwards the capability to the plugin", async () => {
    const client = new CostSourceClient({ baseUrl: "https://plugin-aws.example.com" });
    const resp = await client.supports(
      create(SupportsRequestSchema, {
        resource: { provider: "custom", resourceType: "custom:compute:Instance" },
        capability: PluginCapability.PROJECTED_COSTS,
      }),
    );

    expect(captured?.capability).toBe("PLUGIN_CAPABILITY_PROJECTED_COSTS");
    expect(resp.supported).toBe(false);
    expect(resp.reason).toBe("projected cost not priced");
  });
});
