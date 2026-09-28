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
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";

import { AllocatorClient } from "../src/clients/allocator.js";
import { AllocateRequestSchema } from "../src/generated/finfocus/v1/allocation_pb.js";
import { KIND_IDLE, KIND_WORKLOAD, SUBJECT_KIND, SUBJECT_NODE } from "../src/utils/usage-subjects.js";

const baseUrl = "https://allocator.example.com";
const allocateEndpoint = `${baseUrl}/finfocus.v1.AllocatorService/Allocate`;
const digest = "a".repeat(64);

let lastRequest: Record<string, unknown> | undefined;

const server = setupServer(
  http.post(allocateEndpoint, async ({ request }) => {
    lastRequest = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({
      rows: [
        {
          subject: { kind: "workload", namespace: "payments", pod: "api", node: "n1" },
          cpuCost: 1.25,
          memCost: 1.25,
          totalCost: 2.5,
          currency: "USD",
        },
        {
          subject: { kind: "__idle__", node: "n1" },
          cpuCost: 3.75,
          memCost: 3.75,
          totalCost: 7.5,
          currency: "USD",
        },
      ],
      effectivePolicyJson: "eyJ2ZXJzaW9uIjoxfQ==",
      policyDigest: digest,
      warnings: ["node n1 reports no memory capacity"],
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  lastRequest = undefined;
});
afterAll(() => server.close());

describe("AllocatorClient", () => {
  const client = new AllocatorClient({ baseUrl });

  it("sends priced resources and policy and returns rows, policy, digest, and warnings", async () => {
    const resp = await client.allocate(
      create(AllocateRequestSchema, {
        priced: [{ resource: { id: "n1", tags: { kind: "node" } }, cost: 10, currency: "USD", priced: true }],
        policyJson: new TextEncoder().encode('{"node_split":{"cpu_weight":0.25}}'),
      }),
    );

    expect(lastRequest).toMatchObject({
      priced: [{ resource: { id: "n1" }, cost: 10, priced: true }],
      policyJson: "eyJub2RlX3NwbGl0Ijp7ImNwdV93ZWlnaHQiOjAuMjV9fQ==",
    });
    expect(resp.rows).toHaveLength(2);
    expect(resp.rows[0].subject[SUBJECT_KIND]).toBe(KIND_WORKLOAD);
    expect(resp.rows[0].totalCost).toBe(2.5);
    expect(resp.rows[1].subject[SUBJECT_KIND]).toBe(KIND_IDLE);
    expect(resp.rows[1].subject[SUBJECT_NODE]).toBe("n1");
    expect(JSON.parse(new TextDecoder().decode(resp.effectivePolicyJson))).toEqual({ version: 1 });
    expect(resp.policyDigest).toBe(digest);
    expect(resp.warnings).toEqual(["node n1 reports no memory capacity"]);
  });

  it("propagates Connect errors with their code", async () => {
    server.use(
      http.post(allocateEndpoint, () =>
        HttpResponse.json(
          { code: "invalid_argument", message: "unknown field node_split.cpu" },
          { status: 400 },
        ),
      ),
    );

    const err = await client.allocate(create(AllocateRequestSchema)).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ConnectError);
    expect((err as ConnectError).code).toBe(Code.InvalidArgument);
    expect((err as ConnectError).rawMessage).toBe("unknown field node_split.cpu");
  });
});
