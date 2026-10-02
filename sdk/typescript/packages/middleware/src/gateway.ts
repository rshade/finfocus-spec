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

/** Deadline in milliseconds for receiving the whole request body. */
const BODY_READ_TIMEOUT_MS = 30_000;

/** Codes whose upstream message may carry internal detail, so callers get a generic one. */
const SERVER_SIDE_CODES: ReadonlySet<Code> = new Set([
  Code.Internal,
  Code.Unknown,
  Code.Unavailable,
  Code.DataLoss,
]);

class BodyReadError extends Error {
  constructor(readonly status: number, message: string) {
    super(message);
  }
}

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
      try {
        body = await readBody(req);
      } catch (error) {
        if (error instanceof BodyReadError) {
          send(res, { status: error.status, body: { error: error.message } }, () => req.destroy());
        } else {
          console.error(`finfocus gateway: reading request body failed: ${errorMessage(error)}`);
          send(res, { status: 500, body: { error: "Internal error" } });
        }
        return;
      }
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

/** Reads the request body; rejects with a BodyReadError past MAX_BODY_SIZE or BODY_READ_TIMEOUT_MS. */
function readBody(req: http.IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    let size = 0;
    const cleanup = () => {
      clearTimeout(timer);
      req.off("data", onData);
      req.off("end", onEnd);
      req.off("error", onError);
    };
    const fail = (error: Error) => {
      cleanup();
      req.pause();
      reject(error);
    };
    const onData = (chunk: Buffer) => {
      size += chunk.length;
      if (size > MAX_BODY_SIZE) {
        fail(new BodyReadError(413, "Request body too large"));
        return;
      }
      chunks.push(chunk);
    };
    const onEnd = () => {
      cleanup();
      resolve(Buffer.concat(chunks).toString());
    };
    const onError = (error: Error) => fail(error);
    const timer = setTimeout(() => fail(new BodyReadError(408, "Request body timed out")), BODY_READ_TIMEOUT_MS);
    req.on("data", onData);
    req.on("end", onEnd);
    req.on("error", onError);
  });
}

function errorResult(error: unknown): GatewayResult {
  if (error instanceof ValidationError) {
    return { status: 400, body: { error: error.message, code: codeToString(Code.InvalidArgument) } };
  }
  const err = ConnectError.from(error);
  let message = err.rawMessage;
  if (SERVER_SIDE_CODES.has(err.code)) {
    console.error(`finfocus gateway: upstream ${codeToString(err.code)} error: ${err.rawMessage}`);
    message = "Upstream service error";
  }
  return {
    status: codeToHttpStatus(err.code),
    body: { error: message, code: codeToString(err.code) },
  };
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function send(res: http.ServerResponse, result: GatewayResult, onFlushed?: () => void): void {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (onFlushed) headers.Connection = "close";
  res.writeHead(result.status, headers);
  res.end(JSON.stringify(result.body), onFlushed);
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
