import { PluginCapability } from "../generated/finfocus/v1/enums_pb.js";

export const DEFAULT_MAX_BATCH_SIZE = 100;
export const MAX_BATCH_SIZE = 1000;

type BatchCapabilitySource = {
  capabilities?: PluginCapability[];
  metadata?: Record<string, string | boolean>;
};

/**
 * Determine whether batch-cost operations are supported by the given capability source.
 *
 * @param source - Object containing optional `capabilities` and `metadata` that describe plugin features
 * @returns `true` if the source declares `PluginCapability.BATCH_COST` in `capabilities` or has a `metadata["supports_batch_cost"]` value of `true` or `"true"`, `false` otherwise
 */
export function isBatchSupported(source: BatchCapabilitySource): boolean {
  if (!source) {
    return false;
  }

  if (source.capabilities?.includes(PluginCapability.BATCH_COST)) {
    return true;
  }

  const metadataValue = source.metadata?.["supports_batch_cost"];
  return metadataValue === true || metadataValue === "true";
}
