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

import { describe, expect, it } from "vitest";
import Fastify from "fastify";
import { createFastifyPlugin, createFastifyRoutes } from "../src/fastify/index.js";
import { RPC_BODY, RPC_PATH, RPC_RESULT, stubConfig } from "./stub-client.js";

describe("Fastify adapter", () => {
  it.each([
    ["createFastifyPlugin", (app: ReturnType<typeof Fastify>) => app.register(createFastifyPlugin(stubConfig))],
    ["createFastifyRoutes", (app: ReturnType<typeof Fastify>) => app.register(createFastifyRoutes, { config: stubConfig })],
  ])("serves RPCs through %s", async (_name, register) => {
    const app = Fastify();
    await register(app);

    const res = await app.inject({ method: "POST", url: RPC_PATH, payload: RPC_BODY });

    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual(RPC_RESULT);
    await app.close();
  });

  it("returns the gateway's 404 for unknown methods", async () => {
    const app = Fastify();
    await app.register(createFastifyPlugin(stubConfig));

    const res = await app.inject({
      method: "POST",
      url: "/finfocus.v1.CostSourceService/Nope",
      payload: {},
    });

    expect(res.statusCode).toBe(404);
    await app.close();
  });
});
