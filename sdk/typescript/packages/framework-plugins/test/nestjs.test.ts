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

import "reflect-metadata";
import { afterEach, describe, expect, it } from "vitest";
import { Module, type INestApplication } from "@nestjs/common";
import { NestFactory } from "@nestjs/core";
import { FinFocusModule } from "../src/nestjs/index.js";
import { RPC_BODY, RPC_PATH, RPC_RESULT, stubConfig } from "./stub-client.js";

describe("NestJS adapter", () => {
  let app: INestApplication | undefined;

  afterEach(async () => {
    await app?.close();
    app = undefined;
  });

  it.each([
    ["register", FinFocusModule.register(stubConfig)],
    ["registerAsync", FinFocusModule.registerAsync({ useFactory: async () => stubConfig })],
  ])("serves RPCs through FinFocusModule.%s", async (_name, finfocus) => {
    @Module({ imports: [finfocus] })
    class AppModule {}
    app = await NestFactory.create(AppModule, { logger: false });
    await app.listen(0, "127.0.0.1");

    const res = await fetch(`${await app.getUrl()}${RPC_PATH}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(RPC_BODY),
    });

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual(RPC_RESULT);
  });
});
