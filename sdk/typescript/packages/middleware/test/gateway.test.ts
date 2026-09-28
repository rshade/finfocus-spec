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

import { afterAll, beforeAll, describe, expect, it } from "vitest";
import * as http from "http";
import { PassThrough } from "stream";
import { CostSourceClient } from "@rshade/finfocus-client";
import { createNodeTransport } from "../src/transport.js";
import { RESTGateway } from "../src/gateway.js";
import { listen, startPlugin, RESULT_TIME, type TestServer } from "./plugin-server.js";

describe("RESTGateway", () => {
  let plugin: TestServer;
  let gatewayServer: TestServer;
  let gateway: RESTGateway;

  beforeAll(async () => {
    plugin = await startPlugin();
    gateway = new RESTGateway({
      costSourceClient: new CostSourceClient({
        baseUrl: plugin.baseUrl,
        transport: createNodeTransport({ baseUrl: plugin.baseUrl }),
      }),
    });
    gatewayServer = await listen((req, res) => void gateway.handleRequest(req, res));
  });

  afterAll(async () => {
    await gatewayServer.close();
    await plugin.close();
  });

  const post = (path: string, body: string) =>
    fetch(`${gatewayServer.baseUrl}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });

  it("round-trips proto3 JSON, including Timestamp fields", async () => {
    const res = await post(
      "/finfocus.v1.CostSourceService/GetActualCost",
      JSON.stringify({ resourceId: "i-123", start: RESULT_TIME, end: RESULT_TIME }),
    );

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({
      results: [{ timestamp: RESULT_TIME, cost: 12.5, source: "i-123" }],
    });
  });

  it("accepts an empty body as the default request", async () => {
    const res = await post("/finfocus.v1.CostSourceService/Name", "");

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ name: "test-plugin" });
  });

  it.each([
    ["an unknown service", "/finfocus.v1.NopeService/Name"],
    ["an unknown method", "/finfocus.v1.CostSourceService/Nope"],
    ["a non-RPC client member", "/finfocus.v1.CostSourceService/Constructor"],
    ["an unconfigured service", "/finfocus.v1.ObservabilityService/HealthCheck"],
    ["a path outside the API", "/elsewhere"],
  ])("returns 404 for %s", async (_name, path) => {
    const res = await post(path, "{}");

    expect(res.status).toBe(404);
  });

  it("returns 405 for non-POST requests", async () => {
    const res = await fetch(`${gatewayServer.baseUrl}/finfocus.v1.CostSourceService/Name`);

    expect(res.status).toBe(405);
  });

  it("returns 400 for malformed JSON", async () => {
    const res = await post("/finfocus.v1.CostSourceService/Name", "{not json");

    expect(res.status).toBe(400);
  });

  it("returns 400 for JSON that does not match the request message", async () => {
    const res = await post(
      "/finfocus.v1.CostSourceService/GetActualCost",
      JSON.stringify({ resourceId: "i-123", noSuchField: true }),
    );

    expect(res.status).toBe(400);
  });

  it("returns 400 when client-side validation rejects the request", async () => {
    const res = await post("/finfocus.v1.CostSourceService/GetActualCost", "{}");

    expect(res.status).toBe(400);
  });

  it("maps plugin Connect errors to their HTTP status", async () => {
    const res = await post(
      "/finfocus.v1.CostSourceService/GetActualCost",
      JSON.stringify({ resourceId: "missing" }),
    );

    expect(res.status).toBe(404);
    expect(await res.json()).toMatchObject({ code: "not_found" });
  });

  it("returns 413 for bodies over 1MB", async () => {
    const req = fakeRequest("/finfocus.v1.CostSourceService/Name");
    const reply = capture();
    const done = gateway.handleRequest(req, reply.res);
    req.end(Buffer.alloc(1024 * 1024 + 1, "x"));
    await done;

    expect(reply.status).toBe(413);
  });

  it("uses a body a framework has already parsed", async () => {
    const req = Object.assign(fakeRequest("/finfocus.v1.CostSourceService/GetActualCost"), {
      body: { resourceId: "i-9" },
    });
    const reply = capture();

    await gateway.handleRequest(req, reply.res);

    expect(reply.status).toBe(200);
    expect(JSON.parse(reply.payload).results[0].source).toBe("i-9");
  });
});

function fakeRequest(url: string): PassThrough & http.IncomingMessage {
  return Object.assign(new PassThrough(), { method: "POST", url, headers: {} }) as never;
}

function capture() {
  const reply = { status: 0, payload: "", res: undefined as unknown as http.ServerResponse };
  reply.res = {
    writeHead: (code: number) => {
      reply.status = code;
      return reply.res;
    },
    end: (data?: string) => {
      reply.payload = data ?? "";
      return reply.res;
    },
  } as unknown as http.ServerResponse;
  return reply;
}
