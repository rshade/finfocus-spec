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

import * as http from "http";
import * as http2 from "http2";
import type { AddressInfo } from "net";
import { ConnectError, Code, type ConnectRouter } from "@connectrpc/connect";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  CostSourceService,
  ActualCostResultSchema,
  GetActualCostResponseSchema,
  NameResponseSchema,
} from "@rshade/finfocus-client";

export interface TestServer {
  baseUrl: string;
  close(): Promise<void>;
}

export async function listen(handler: http.RequestListener): Promise<TestServer> {
  const server = http.createServer(handler);
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as AddressInfo;
  return {
    baseUrl: `http://127.0.0.1:${port}`,
    close: () =>
      new Promise<void>((resolve) => {
        server.closeAllConnections();
        server.close(() => resolve());
      }),
  };
}

export const RESULT_TIME = "2026-01-01T00:00:00Z";

async function listenH2c(handler: (req: http2.Http2ServerRequest, res: http2.Http2ServerResponse) => void): Promise<TestServer> {
  const server = http2.createServer(handler);
  const sessions = new Set<http2.ServerHttp2Session>();
  server.on("session", (session) => sessions.add(session));
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as AddressInfo;
  return {
    baseUrl: `http://127.0.0.1:${port}`,
    close: () =>
      new Promise<void>((resolve) => {
        sessions.forEach((session) => session.close());
        server.close(() => resolve());
      }),
  };
}

/** Starts an in-process CostSource plugin speaking the Connect protocol over HTTP/1.1 or cleartext HTTP/2. */
export function startPlugin(httpVersion: "1.1" | "2" = "1.1"): Promise<TestServer> {
  const routes = (router: ConnectRouter) =>
    router.service(CostSourceService, {
      name: () => create(NameResponseSchema, { name: "test-plugin" }),
      getActualCost: (req) => {
        if (req.resourceId === "missing") {
          throw new ConnectError("no such resource", Code.NotFound);
        }
        return create(GetActualCostResponseSchema, {
          results: [
            create(ActualCostResultSchema, {
              timestamp: timestampFromDate(new Date(RESULT_TIME)),
              cost: 12.5,
              source: req.resourceId,
            }),
          ],
        });
      },
    });
  const handler = connectNodeAdapter({ routes });
  return httpVersion === "2" ? listenH2c(handler) : listen(handler);
}
