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

import { afterEach, describe, expect, it } from "vitest";
import express from "express";
import type { Server } from "http";
import type { AddressInfo } from "net";
import { createExpressMiddleware, createExpressRouter } from "../src/express/index.js";
import { RPC_BODY, RPC_PATH, RPC_RESULT, stubConfig } from "./stub-client.js";

describe("Express adapter", () => {
  let server: Server | undefined;

  afterEach(() => {
    server?.close();
    server = undefined;
  });

  async function serve(app: express.Express): Promise<string> {
    server = app.listen(0, "127.0.0.1");
    await new Promise((resolve) => server!.once("listening", resolve));
    return `http://127.0.0.1:${(server!.address() as AddressInfo).port}`;
  }

  const post = (base: string, path: string) =>
    fetch(`${base}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(RPC_BODY),
    });

  it.each([
    ["middleware", false],
    ["middleware after express.json()", true],
  ])("serves RPCs through the %s", async (_name, parseJson) => {
    const app = express();
    if (parseJson) app.use(express.json());
    app.use(createExpressMiddleware(stubConfig));
    const base = await serve(app);

    const res = await post(base, RPC_PATH);

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual(RPC_RESULT);
  });

  it("serves RPCs through the router and passes other paths on", async () => {
    const app = express();
    app.use(createExpressRouter(stubConfig));
    app.post("/other", (_req, res) => {
      res.status(204).end();
    });
    const base = await serve(app);

    const rpc = await post(base, RPC_PATH);
    const other = await post(base, "/other");

    expect(rpc.status).toBe(200);
    expect(await rpc.json()).toEqual(RPC_RESULT);
    expect(other.status).toBe(204);
  });
});
