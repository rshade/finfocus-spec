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
import { Code, ConnectError, createRouterTransport } from "@connectrpc/connect";

import {
  CostParseClient, parseParseLimits,
  PARSE_FORMATS_KEY, PARSE_INPUT_MODES_KEY, PARSE_MAX_INPUT_BYTES_KEY,
  PARSE_ERROR_DOMAIN, UNSUPPORTED_FORMAT_REASON, UNSUPPORTED_INPUT_MODE_REASON,
} from "../src/clients/cost-parse.js";
import type { ParseLimits } from "../src/clients/cost-parse.js";
import {
  CostParseService, ParseRequestSchema, ParseResponseSchema, ParseWarningSeverity,
} from "../src/generated/finfocus/v1/parse_pb.js";
import type { ParseRequest } from "../src/generated/finfocus/v1/parse_pb.js";

const baseUrl = "https://parser.example.com";
const endpoint = `${baseUrl}/finfocus.v1.CostParseService/Parse`;
let lastRequest: Record<string, unknown> | undefined;
let lastContentType: string | null = null;
const responseJSON = {
  resources: [{
    sourceId: "Instance",
    resource: {
      provider: "custom", resourceType: "fixture:compute:Instance",
      attributes: { Replicas: 0, Zone: "", Sizing: {} },
    },
    unknownProperties: ["Sizing.Cpu"],
  }],
  warnings: [{
    severity: "PARSE_WARNING_SEVERITY_WARNING", code: "resource_skipped",
    message: "Unsupported source skipped.", sourceId: "Unsupported",
  }],
  expiresAt: "2026-10-18T12:34:56.123456789Z",
};
const server = setupServer(http.post(endpoint, async ({ request }) => {
  lastContentType = request.headers.get("content-type");
  lastRequest = await request.json() as Record<string, unknown>;
  return HttpResponse.json(responseJSON);
}));
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers(); lastRequest = undefined; lastContentType = null;
});
afterAll(() => server.close());

function contentRequest(): ParseRequest {
  return create(ParseRequestSchema, {
    format: "fixture",
    input: { case: "files", value: { files: [{ path: "main.fixture", content: new Uint8Array([0, 255, 65]) }] } },
    options: { namespace: "payments", replicas: 0 },
  });
}

describe("CostParseClient", () => {
  it("defaults to JSON Connect and encodes arbitrary bytes as base64 AP9B", async () => {
    const result = await new CostParseClient({ baseUrl }).parse(contentRequest());
    expect(lastContentType).toContain("application/json");
    expect(lastRequest).toEqual({
      format: "fixture", files: { files: [{ path: "main.fixture", content: "AP9B" }] },
      options: { namespace: "payments", replicas: 0 },
    });
    expect(lastRequest).not.toHaveProperty("rootPath");
    expect(result.resources[0].sourceId).toBe("Instance");
    expect(result.resources[0].resource?.resourceType).toBe("fixture:compute:Instance");
    expect(result.resources[0].resource?.attributes).toEqual({ Replicas: 0, Zone: "", Sizing: {} });
    expect(result.resources[0].unknownProperties).toEqual(["Sizing.Cpu"]);
    expect(result.warnings[0]).toMatchObject({
      severity: ParseWarningSeverity.WARNING, code: "resource_skipped",
      message: "Unsupported source skipped.", sourceId: "Unsupported",
    });
    expect(result.expiresAt?.seconds).toBe(1792326896n);
    expect(result.expiresAt?.nanos).toBe(123456789);
  });

  it("sends rootPath without files", async () => {
    await new CostParseClient({ baseUrl }).parse(create(ParseRequestSchema, {
      format: "fixture", input: { case: "rootPath", value: "/host/project" },
    }));
    expect(lastRequest).toEqual({ format: "fixture", rootPath: "/host/project" });
    expect(lastRequest).not.toHaveProperty("files");
  });

  it("honors an injected transport for both input arms without HTTP", async () => {
    const seen: ParseRequest[] = [];
    const expected = create(ParseResponseSchema, {
      resources: [{ sourceId: "native/id", resource: { provider: "custom", resourceType: "fixture:compute:Instance" } }],
      expiresAt: { seconds: 1792326896n, nanos: 123456789 },
    });
    const transport = createRouterTransport((router) => {
      router.service(CostParseService, { parse(request) { seen.push(request); return expected; } });
    });
    const client = new CostParseClient({ baseUrl: "https://unused.invalid", transport });
    const files = contentRequest();
    const root = create(ParseRequestSchema, { format: "fixture", input: { case: "rootPath", value: "/host/project" } });
    expect(await client.parse(files)).toEqual(expected);
    expect(await client.parse(root)).toEqual(expected);
    expect(seen).toEqual([files, root]);
    expect(seen[0].input.case === "files" && seen[0].input.value.files[0].content).toEqual(new Uint8Array([0, 255, 65]));
    expect(lastRequest).toBeUndefined();
  });

  it.each(["UNSUPPORTED_FORMAT", "UNSUPPORTED_INPUT_MODE"])("preserves Connect ErrorInfo for %s", async (reason) => {
    const field = (tag: number, value: string) => {
      const encoded = new TextEncoder().encode(value);
      return [(tag << 3) | 2, encoded.length, ...encoded];
    };
    const detail = Uint8Array.from([...field(1, reason), ...field(2, "finfocus.v1.CostParseService")]);
    server.use(http.post(endpoint, () => HttpResponse.json({
      code: "failed_precondition", message: "Requested parser input is unsupported.",
      details: [{ type: "google.rpc.ErrorInfo", value: Buffer.from(detail).toString("base64") }],
    }, { status: 400 })));
    const err: unknown = await new CostParseClient({ baseUrl }).parse(contentRequest()).catch((error: unknown) => error);
    expect(err).toBeInstanceOf(ConnectError);
    const connectError = err as ConnectError;
    expect(connectError.code).toBe(Code.FailedPrecondition);
    expect(connectError.rawMessage).toBe("Requested parser input is unsupported.");
    expect(connectError.details).toEqual([{ type: "google.rpc.ErrorInfo", value: detail }]);
  });

  it("preserves injected transport errors and details", async () => {
    const detail = { type: "google.rpc.ErrorInfo", value: new Uint8Array([10, 1, 120]) };
    const transport = createRouterTransport((router) => {
      router.service(CostParseService, {
        parse() { throw new ConnectError("unsupported", Code.FailedPrecondition, undefined, [detail]); },
      });
    });
    const err: unknown = await new CostParseClient({ baseUrl, transport }).parse(contentRequest()).catch((error: unknown) => error);
    expect(err).toBeInstanceOf(ConnectError);
    expect((err as ConnectError).code).toBe(Code.FailedPrecondition);
    expect((err as ConnectError).details).toEqual([detail]);
  });
});

describe("parseParseLimits", () => {
  const valid = { parse_formats: "fixture,cfn_1,helm-v3", parse_input_modes: "content,path", parse_max_input_bytes: "1024" };
  it("exports canonical metadata and error constants", () => {
    expect([PARSE_FORMATS_KEY, PARSE_INPUT_MODES_KEY, PARSE_MAX_INPUT_BYTES_KEY]).toEqual([
      "parse_formats", "parse_input_modes", "parse_max_input_bytes",
    ]);
    expect(PARSE_ERROR_DOMAIN).toBe("finfocus.v1.CostParseService");
    expect(UNSUPPORTED_FORMAT_REASON).toBe("UNSUPPORTED_FORMAT");
    expect(UNSUPPORTED_INPUT_MODE_REASON).toBe("UNSUPPORTED_INPUT_MODE");
  });
  it.each([{}, { unrelated: "present" }, { parse_other: "unknown" }])("returns undefined without parse metadata %j", (metadata) => {
    expect(parseParseLimits(metadata)).toBeUndefined();
  });
  it("parses ordered canonical names", () => {
    const result: ParseLimits | undefined = parseParseLimits(valid);
    expect(result).toEqual({ formats: ["fixture", "cfn_1", "helm-v3"], inputModes: ["content", "path"], maxInputBytes: 1024n });
    expect(parseParseLimits({ ...valid, parse_input_modes: "path,content" })?.inputModes).toEqual(["path", "content"]);
  });
  it.each(["1", "9007199254740993", "9223372036854775807"])("preserves exact int64 %s as bigint", (raw) => {
    const result = parseParseLimits({ ...valid, parse_max_input_bytes: raw });
    expect(typeof result?.maxInputBytes).toBe("bigint");
    expect(result?.maxInputBytes).toBe(BigInt(raw));
  });
  it.each(["content", "path"])("accepts single mode %s", (mode) => {
    expect(parseParseLimits({ ...valid, parse_input_modes: mode })?.inputModes).toEqual([mode]);
  });
  it.each(["a", "a".repeat(32)])("accepts boundary-length format %s", (format) => {
    expect(parseParseLimits({ ...valid, parse_formats: format })?.formats).toEqual([format]);
  });
  it.each([
    { parse_formats: "fixture" }, { parse_input_modes: "content" }, { parse_max_input_bytes: "1" },
    { parse_formats: "fixture", parse_input_modes: "content" },
    { parse_formats: "fixture", parse_max_input_bytes: "1" },
    { parse_input_modes: "content", parse_max_input_bytes: "1" },
  ])("rejects partial metadata %j", (metadata) => { expect(() => parseParseLimits(metadata)).toThrow(); });
  it.each([
    "", "Fixture", "fixture ", " fixture", "fixture, fixture", "fixture,", ",fixture", "fixture,,cfn",
    "fixture,fixture", "cfn.1", "1fixture", "_fixture", "é", "a".repeat(33), "fixture\n",
  ])("rejects malformed formats %j", (formats) => {
    expect(() => parseParseLimits({ ...valid, parse_formats: formats })).toThrow();
  });
  it.each(["", "CONTENT", "content ", " content", "content,path,", "content,,path", "content,content", "root_path", "files", "both", "path\n"])(
    "rejects malformed modes %j", (modes) => { expect(() => parseParseLimits({ ...valid, parse_input_modes: modes })).toThrow(); },
  );
  it.each([
    "", "0", "00", "01", "+1", "-1", " 1", "1 ", "1\n", "1.0", "1e3", "0x10", "NaN", "Infinity",
    "9223372036854775808", "999999999999999999999999999999999999999999", "١",
  ])("rejects noncanonical or overflowing limits %j", (raw) => {
    expect(() => parseParseLimits({ ...valid, parse_max_input_bytes: raw })).toThrow();
  });
});
