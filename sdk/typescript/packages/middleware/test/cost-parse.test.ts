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

import { afterEach, describe, expect, it, vi } from "vitest";
import type * as http from "http";
import { PassThrough } from "stream";
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  ParseResponseSchema,
  ParseWarningSeverity,
  type CostParseClient,
  type CostSourceClient,
  type ParseRequest,
  type ParseResponse,
} from "@rshade/finfocus-client";
import { RESTGateway } from "../src/gateway.js";
import { listen, type TestServer } from "./plugin-server.js";

const PARSE_ROUTE = "/finfocus.v1.CostParseService/Parse";
const INPUT = { format: "fixture", files: { files: [{ path: "main.fixture", content: "AP9B" }] } };
const EXPIRY = "2026-01-01T00:00:00.123456789Z";

function parserGateway(parse: (request: ParseRequest) => Promise<ParseResponse>): RESTGateway {
  return new RESTGateway({
    costSourceClient: {} as CostSourceClient,
    costParseClient: { parse } as CostParseClient,
  });
}

function parsedResponse(): ParseResponse {
  return create(ParseResponseSchema, {
    resources: [{
      sourceId: "Instance",
      resource: {
        provider: "custom",
        resourceType: "custom:compute:Instance",
        attributes: { Replicas: 0, Enabled: false },
      },
      unknownProperties: ["InstanceType"],
    }],
    warnings: [{
      severity: ParseWarningSeverity.WARNING,
      code: "property_unknown",
      message: "InstanceType is unresolved",
      sourceId: "Instance",
    }],
    expiresAt: { seconds: BigInt("1767225600"), nanos: 123456789 },
  });
}

describe("CostParse REST gateway", () => {
  let server: TestServer | undefined;

  afterEach(async () => {
    await server?.close();
    server = undefined;
    vi.restoreAllMocks();
  });

  it("routes Parse and round-trips base64 bytes, source metadata and nanosecond RFC 3339 expiry", async () => {
    const parse = vi.fn(async (request: ParseRequest) => {
      expect(request.format).toBe("fixture");
      expect(request.input.case).toBe("files");
      if (request.input.case !== "files") throw new Error("expected files");
      expect(request.input.value.files[0].path).toBe("main.fixture");
      expect(request.input.value.files[0].content).toEqual(new Uint8Array([0, 255, 65]));
      expect(request.options).toEqual({ parameters: { Size: "small" } });
      return parsedResponse();
    });
    const gateway = parserGateway(parse);
    server = await listen((req, res) => void gateway.handleRequest(req, res));
    const response = await fetch(`${server.baseUrl}${PARSE_ROUTE}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...INPUT, options: { parameters: { Size: "small" } } }),
    });

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({
      resources: [{
        sourceId: "Instance",
        resource: {
          provider: "custom",
          resourceType: "custom:compute:Instance",
          attributes: { Replicas: 0, Enabled: false },
        },
        unknownProperties: ["InstanceType"],
      }],
      warnings: [{
        severity: "PARSE_WARNING_SEVERITY_WARNING",
        code: "property_unknown",
        message: "InstanceType is unresolved",
        sourceId: "Instance",
      }],
      expiresAt: EXPIRY,
    });
    expect(parse).toHaveBeenCalledTimes(1);
  });

  it("routes the rootPath input arm without interpreting provider-owned options", async () => {
    const parse = vi.fn(async (request: ParseRequest) => {
      expect(request.input).toEqual({ case: "rootPath", value: "/host/fixtures" });
      expect(request.options).toEqual({ release_name: "demo", namespace: "test" });
      return create(ParseResponseSchema);
    });
    const result = await parserGateway(parse).dispatch(PARSE_ROUTE, {
      format: "fixture",
      rootPath: "/host/fixtures",
      options: { release_name: "demo", namespace: "test" },
    });

    expect(result).toEqual({ status: 200, body: {} });
    expect(parse).toHaveBeenCalledTimes(1);
  });

  it("returns the existing 404 response when the parser client is absent", async () => {
    const gateway = new RESTGateway({ costSourceClient: {} as CostSourceClient });
    expect(await gateway.dispatch(PARSE_ROUTE, INPUT)).toEqual({ status: 404, body: { error: "Not found" } });
  });

  it.each([
    { ...INPUT, rootPath: "/host/fixtures" },
    { rootPath: "/host/fixtures", ...INPUT },
    { ...INPUT, rootPath: "" },
    { root_path: "", ...INPUT },
  ])("rejects conflicting JSON input arms before calling the parser: %j", async (body) => {
    const parse = vi.fn(async () => create(ParseResponseSchema));
    const result = await parserGateway(parse).dispatch(PARSE_ROUTE, JSON.stringify(body));

    expect(result.status).toBe(400);
    expect(parse).not.toHaveBeenCalled();
  });

  it.each(["{bad json", JSON.stringify({ ...INPUT, noSuchField: true })])(
    "rejects malformed JSON or unknown request members: %s",
    async (body) => {
      const parse = vi.fn(async () => create(ParseResponseSchema));
      expect((await parserGateway(parse).dispatch(PARSE_ROUTE, body)).status).toBe(400);
      expect(parse).not.toHaveBeenCalled();
    },
  );

  it.each([
    [Code.InvalidArgument, 400, "invalid_argument", "files[0].content: main.fixture:7 malformed declaration"],
    [Code.FailedPrecondition, 400, "failed_precondition", "unsupported format"],
    [Code.ResourceExhausted, 429, "resource_exhausted", "input exceeds parse_max_input_bytes 2"],
    [Code.Unimplemented, 501, "unimplemented", "parsing is not implemented"],
  ] as const)("maps parser category %s to HTTP %s", async (code, status, name, message) => {
    const parse = vi.fn(async () => { throw new ConnectError(message, code); });
    const result = await parserGateway(parse).dispatch(PARSE_ROUTE, INPUT);

    expect(result).toEqual({ status, body: { error: message, code: name } });
    expect(parse).toHaveBeenCalledTimes(1);
  });

  it.each([Code.Internal, Code.Unknown, Code.Unavailable, Code.DataLoss])(
    "sanitizes upstream server-side category %s",
    async (code) => {
      vi.spyOn(console, "error").mockImplementation(() => {});
      const parse = vi.fn(async () => { throw new ConnectError("secret internal-parser:1234", code); });
      const result = await parserGateway(parse).dispatch(PARSE_ROUTE, INPUT);

      expect(result.status).toBeGreaterThanOrEqual(500);
      expect(result.body).toMatchObject({ error: "Upstream service error", code: expect.any(String) });
      expect(JSON.stringify(result.body)).not.toContain("secret");
      expect(parse).toHaveBeenCalledTimes(1);
    },
  );

  it("sanitizes ordinary upstream errors", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const parse = vi.fn(async () => { throw new Error("secret internal credentials"); });
    const result = await parserGateway(parse).dispatch(PARSE_ROUTE, INPUT);

    expect(result).toEqual({ status: 500, body: { error: "Upstream service error", code: "unknown" } });
  });

  it("preserves the independent 1 MiB HTTP body cap for a three-byte parser input", async () => {
    const parse = vi.fn(async () => create(ParseResponseSchema));
    const gateway = parserGateway(parse);
    const json = JSON.stringify(INPUT);
    const exact = json + " ".repeat(1024 * 1024 - Buffer.byteLength(json));
    const accepted = fakeRequest();
    const acceptedReply = capture();
    const acceptedDone = gateway.handleRequest(accepted, acceptedReply.res);
    accepted.end(exact);
    await acceptedDone;

    expect(acceptedReply.status).toBe(200);
    expect(parse).toHaveBeenCalledTimes(1);

    const oversized = fakeRequest();
    const rejected = capture();
    const done = gateway.handleRequest(oversized, rejected.res);
    oversized.end(exact + " ");
    await done;

    expect(rejected.status).toBe(413);
    expect(rejected.headers).toMatchObject({ Connection: "close" });
    expect(oversized.destroyed).toBe(true);
    expect(parse).toHaveBeenCalledTimes(1);
  });
});

function fakeRequest(): PassThrough & http.IncomingMessage {
  return Object.assign(new PassThrough(), { method: "POST", url: PARSE_ROUTE, headers: {} }) as never;
}

function capture() {
  const reply = {
    status: 0,
    headers: {} as Record<string, string>,
    res: undefined as unknown as http.ServerResponse,
  };
  reply.res = {
    writeHead: (code: number, headers?: Record<string, string>) => {
      reply.status = code;
      reply.headers = headers ?? {};
      return reply.res;
    },
    end: (_data?: string, callback?: () => void) => {
      callback?.();
      return reply.res;
    },
  } as unknown as http.ServerResponse;
  return reply;
}
