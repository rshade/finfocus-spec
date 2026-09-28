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

import type * as http from "http";
import { fromJson, toJson, type DescService, type JsonValue } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { codeToHttpStatus, codeToString } from "@connectrpc/connect/protocol-connect";
import {
  CostSourceService,
  ObservabilityService,
  PluginRegistryService,
  ValidationError,
  type CostSourceClient,
  type ObservabilityClient,
  type RegistryClient,
} from "@rshade/finfocus-client";

export interface RESTGatewayConfig {
  costSourceClient: CostSourceClient;
  observabilityClient?: ObservabilityClient;
  registryClient?: RegistryClient;
}

/** HTTP status and JSON body produced for one gateway request. */
export interface GatewayResult {
  status: number;
  body: JsonValue;
}

/** Maximum request body size in bytes (1MB) */
const MAX_BODY_SIZE = 1024 * 1024;

const RPC_PATH = /^\/finfocus\.v1\.([A-Za-z]+)\/([A-Za-z]+)$/;

/**
 * A request whose body a framework (express.json(), NestJS, Fastify) may already have parsed.
 */
type GatewayRequest = http.IncomingMessage & { body?: unknown };

interface ServiceEntry {
  service: DescService;
  client?: object;
}

/**
 * Generic REST Gateway that translates JSON HTTP requests to Connect RPC calls.
 * Request and response bodies use the proto3 JSON mapping, so `Timestamp` fields are
 * RFC 3339 strings and 64-bit integers are strings.
 *
 * Maps HTTP POST requests to RPC methods:
 * - POST /finfocus.v1.CostSourceService/MethodName -> RPC call
 * - POST /finfocus.v1.ObservabilityService/MethodName -> RPC call
 * - POST /finfocus.v1.PluginRegistryService/MethodName -> RPC call
 *
 * @example
 * ```typescript
 * const gateway = new RESTGateway({
 *   costSourceClient: new CostSourceClient({ baseUrl: 'https://plugin.example.com' })
 * });
 *
 * const server = http.createServer((req, res) => {
 *   gateway.handleRequest(req, res);
 * });
 * ```
 */
export class RESTGateway {
  private readonly services: ReadonlyMap<string, ServiceEntry>;

  constructor(config: RESTGatewayConfig) {
    this.services = new Map<string, ServiceEntry>([
      ["CostSourceService", { service: CostSourceService, client: config.costSourceClient }],
      ["ObservabilityService", { service: ObservabilityService, client: config.observabilityClient }],
      ["PluginRegistryService", { service: PluginRegistryService, client: config.registryClient }],
    ]);
  }

  /**
   * Main HTTP request handler.
   * Uses `req.body` when a framework has already parsed it; otherwise reads the request stream.
   * Returns a Promise that resolves after the response is fully sent.
   */
  async handleRequest(req: GatewayRequest, res: http.ServerResponse): Promise<void> {
    if (req.method !== "POST") {
      send(res, { status: 405, body: { error: "Method not allowed" } });
      return;
    }

    let body: unknown;
    if (req.body !== undefined) {
      body = req.body;
    } else if (req.readableEnded) {
      send(res, { status: 400, body: { error: "Request body was already consumed" } });
      return;
    } else {
      let text: string | undefined;
      try {
        text = await readBody(req);
      } catch (error) {
        send(res, { status: 500, body: { error: errorMessage(error) } });
        return;
      }
      if (text === undefined) {
        send(res, { status: 413, body: { error: "Request body too large" } });
        return;
      }
      body = text;
    }

    send(res, await this.dispatch(req.url ?? "", body));
  }

  /**
   * Transport-agnostic entry point for adapters that own the HTTP exchange.
   *
   * @param path - Request path, e.g. `/finfocus.v1.CostSourceService/GetActualCost`; a query string is ignored.
   * @param body - Raw JSON text or an already-parsed JSON value; empty means the default request.
   */
  async dispatch(path: string, body: unknown): Promise<GatewayResult> {
    const match = RPC_PATH.exec(path.split("?")[0]);
    const entry = match ? this.services.get(match[1]) : undefined;
    const method = entry?.service.methods.find((m) => m.name === match![2]);
    const call = method && (entry!.client as Record<string, unknown> | undefined)?.[method.localName];
    if (!method || typeof call !== "function") {
      return { status: 404, body: { error: "Not found" } };
    }

    let json: JsonValue;
    try {
      json = parseBody(body);
    } catch {
      return { status: 400, body: { error: "Malformed JSON" } };
    }

    let request;
    try {
      request = fromJson(method.input, json);
    } catch (error) {
      return { status: 400, body: { error: errorMessage(error) } };
    }

    try {
      const response = await call.call(entry!.client, request);
      return { status: 200, body: toJson(method.output, response) };
    } catch (error) {
      return errorResult(error);
    }
  }
}

function parseBody(body: unknown): JsonValue {
  if (typeof body === "string" || Buffer.isBuffer(body)) {
    const text = body.toString();
    return text.trim() === "" ? {} : (JSON.parse(text) as JsonValue);
  }
  return (body ?? {}) as JsonValue;
}

/** Reads the request body, or resolves undefined once it exceeds MAX_BODY_SIZE. */
function readBody(req: http.IncomingMessage): Promise<string | undefined> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    let size = 0;
    const onData = (chunk: Buffer) => {
      size += chunk.length;
      if (size > MAX_BODY_SIZE) {
        req.off("data", onData);
        req.resume();
        resolve(undefined);
        return;
      }
      chunks.push(chunk);
    };
    req.on("data", onData);
    req.on("end", () => resolve(Buffer.concat(chunks).toString()));
    req.on("error", reject);
  });
}

function errorResult(error: unknown): GatewayResult {
  if (error instanceof ValidationError) {
    return { status: 400, body: { error: error.message, code: codeToString(Code.InvalidArgument) } };
  }
  const err = ConnectError.from(error);
  return {
    status: codeToHttpStatus(err.code),
    body: { error: err.rawMessage, code: codeToString(err.code) },
  };
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function send(res: http.ServerResponse, result: GatewayResult): void {
  res.writeHead(result.status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(result.body));
}

/**
 * Express-style middleware wrapper for RESTGateway.
 *
 * @example
 * ```typescript
 * const gateway = new RESTGateway({...});
 * app.use(createRESTMiddleware(gateway));
 * ```
 */
export function createRESTMiddleware(gateway: RESTGateway) {
  return (req: http.IncomingMessage, res: http.ServerResponse, next?: (err?: Error) => void) => {
    gateway.handleRequest(req, res).catch(error => {
      if (next) next(error instanceof Error ? error : new Error(String(error)));
    });
  };
}
