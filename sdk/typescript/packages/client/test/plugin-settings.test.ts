import { describe, expect, it } from "vitest";
import { create, fromJson, toJson } from "@bufbuild/protobuf";

import {
  PluginSettingSchema,
  PluginSettingType,
} from "../src/generated/finfocus/v1/costsource_pb.js";

describe("PluginSetting", () => {
  it("round-trips key, type, secret, and env", () => {
    const setting = create(PluginSettingSchema, {
      key: "api_key",
      type: PluginSettingType.STRING,
      secret: true,
      env: "TYPESAFE_API_KEY",
    });

    const decoded = fromJson(PluginSettingSchema, toJson(PluginSettingSchema, setting));

    expect(decoded.key).toBe("api_key");
    expect(decoded.type).toBe(PluginSettingType.STRING);
    expect(decoded.secret).toBe(true);
    expect(decoded.env).toBe("TYPESAFE_API_KEY");
  });
});
