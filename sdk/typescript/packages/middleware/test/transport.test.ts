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
import { ConnectError, Code } from "@connectrpc/connect";
import { CostSourceClient } from "@rshade/finfocus-client";
import { createNodeTransport } from "../src/transport.js";
import { listen, startPlugin, type TestServer } from "./plugin-server.js";

describe("createNodeTransport", () => {
  let plugin: TestServer;
  let silent: TestServer;

  beforeAll(async () => {
    plugin = await startPlugin();
    silent = await listen(() => {});
  });

  afterAll(async () => {
    await plugin.close();
    await silent.close();
  });

  it("carries RPCs to a Connect plugin over HTTP/1.1", async () => {
    const client = new CostSourceClient({
      baseUrl: plugin.baseUrl,
      transport: createNodeTransport({ baseUrl: plugin.baseUrl }),
    });

    const res = await client.name();

    expect(res.name).toBe("test-plugin");
  });

  it("carries RPCs to a Connect plugin over HTTP/2", async () => {
    const h2 = await startPlugin("2");
    try {
      const transport = createNodeTransport({ baseUrl: h2.baseUrl, httpVersion: "2" });
      const client = new CostSourceClient({ baseUrl: h2.baseUrl, transport });

      const res = await client.name();

      expect(res.name).toBe("test-plugin");
    } finally {
      await h2.close();
    }
  });

  it("maps an elapsed timeout to DeadlineExceeded", async () => {
    const client = new CostSourceClient({
      baseUrl: silent.baseUrl,
      transport: createNodeTransport({ baseUrl: silent.baseUrl, timeout: 50 }),
    });

    const err = await client.name().catch((e: unknown) => e);

    expect(err).toBeInstanceOf(ConnectError);
    expect((err as ConnectError).code).toBe(Code.DeadlineExceeded);
  });
});
