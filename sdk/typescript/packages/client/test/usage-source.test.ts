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

import { UsageSourceClient } from "../src/clients/usage-source.js";
import { GetStatsRequestSchema, StatsMode } from "../src/generated/finfocus/v1/usage_pb.js";
import * as vocab from "../src/utils/usage-subjects.js";

const baseUrl = "https://usage.example.com";
const getStatsEndpoint = `${baseUrl}/finfocus.v1.UsageSourceService/GetStats`;

let lastRequest: Record<string, unknown> | undefined;

const server = setupServer(
  http.post(getStatsEndpoint, async ({ request }) => {
    lastRequest = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({
      rows: [
        {
          subject: { kind: "workload", namespace: "payments", pod: "api-7d9f", node: "ip-10-0-1-5" },
          metric: "cpu_request",
          amount: 0.5,
          unit: "core",
        },
        {
          subject: { kind: "node", node: "ip-10-0-1-5" },
          metric: "cpu_allocatable",
          amount: 1.93,
          unit: "core",
        },
      ],
      priceable: [
        {
          id: "ip-10-0-1-5",
          provider: "aws",
          resourceType: "ec2",
          tags: { kind: "node", capacity_type: "on-demand" },
        },
      ],
      mode: "STATS_MODE_RUN_RATE",
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  lastRequest = undefined;
});
afterAll(() => server.close());

describe("UsageSourceClient", () => {
  const client = new UsageSourceClient({ baseUrl });

  it("sends scope and selector and returns rows, priceable entries, and mode", async () => {
    const resp = await client.getStats(
      create(GetStatsRequestSchema, { scope: "c1", selector: { namespace: "payments" } }),
    );

    expect(lastRequest).toMatchObject({ scope: "c1", selector: { namespace: "payments" } });
    expect(resp.rows).toHaveLength(2);
    expect(resp.rows[0].subject[vocab.SUBJECT_NAMESPACE]).toBe("payments");
    expect(resp.rows[1].metric).toBe(vocab.METRIC_CPU_ALLOCATABLE);
    expect(resp.priceable).toHaveLength(1);
    expect(resp.priceable[0].id).toBe("ip-10-0-1-5");
    expect(resp.mode).toBe(StatsMode.RUN_RATE);
  });

  it("propagates Connect errors with their code", async () => {
    server.use(
      http.post(getStatsEndpoint, () =>
        HttpResponse.json({ code: "permission_denied", message: "cannot list pods" }, { status: 403 }),
      ),
    );

    const err = await client.getStats(create(GetStatsRequestSchema)).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ConnectError);
    expect((err as ConnectError).code).toBe(Code.PermissionDenied);
    expect((err as ConnectError).rawMessage).toBe("cannot list pods");
  });
});

describe("usage vocabulary", () => {
  it("matches the Go constants in sdk/go/pluginsdk/subjects.go", () => {
    expect(vocab).toMatchObject({
      SUBJECT_CLUSTER: "cluster",
      SUBJECT_NAMESPACE: "namespace",
      SUBJECT_CONTROLLER_KIND: "controller_kind",
      SUBJECT_CONTROLLER: "controller",
      SUBJECT_POD: "pod",
      SUBJECT_NODE: "node",
      SUBJECT_KIND: "kind",
      SUBJECT_LABEL_PREFIX: "label.",
      KIND_WORKLOAD: "workload",
      KIND_NODE: "node",
      KIND_IDLE: "__idle__",
      KIND_CLUSTER: "__cluster__",
      METRIC_CPU_REQUEST: "cpu_request",
      METRIC_MEM_REQUEST: "mem_request",
      METRIC_CPU_ALLOCATABLE: "cpu_allocatable",
      METRIC_MEM_ALLOCATABLE: "mem_allocatable",
      METRIC_CPU_USAGE: "cpu_usage",
      METRIC_MEM_USAGE: "mem_usage",
      UNIT_CORE: "core",
      UNIT_GIB: "GiB",
      UNIT_CORE_HOURS: "core-hours",
      UNIT_GIB_HOURS: "GiB-hours",
    });
  });
});
