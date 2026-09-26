// Generated proto types and services
// In Connect-ES v2, services are generated directly in *_pb.ts files
export * from "./generated/finfocus/v1/enums_pb.js";
export * from "./generated/finfocus/v1/budget_pb.js";
export * from "./generated/finfocus/v1/costsource_pb.js";
export * from "./generated/finfocus/v1/focus_pb.js";
export * from "./generated/finfocus/v1/registry_pb.js";
export * from "./generated/finfocus/v1/usage_pb.js";

// Error handling - our custom ValidationError takes precedence
export { ValidationError } from "./errors/validation-error.js";

// Client implementations
export { CostSourceClient, CostSourceClientConfig } from "./clients/cost-source.js";
export { RegistryClient, ObservabilityClient, ClientConfig } from "./clients/auxiliary.js";
export { UsageSourceClient } from "./clients/usage-source.js";

// Builder patterns
export { ResourceDescriptorBuilder } from "./builders/resource-descriptor.js";
export { RecommendationFilterBuilder } from "./builders/recommendation-filter.js";
export { FocusRecordBuilder } from "./builders/focus-record.js";

// Utilities
export { recommendationsIterator } from "./utils/pagination.js";
export {
  DEFAULT_MAX_BATCH_SIZE,
  MAX_BATCH_SIZE,
  DEFAULT_MAX_SOURCE_TYPES,
  MAX_SOURCE_TYPES,
  isBatchSupported,
} from "./utils/batch.js";
export {
  getAllUsageProfiles,
  isValidUsageProfile,
  parseUsageProfile,
  usageProfileString,
  normalizeUsageProfile,
  defaultMonthlyHours,
} from "./utils/usage-profile.js";
export {
  SUBJECT_CLUSTER,
  SUBJECT_NAMESPACE,
  SUBJECT_CONTROLLER_KIND,
  SUBJECT_CONTROLLER,
  SUBJECT_POD,
  SUBJECT_NODE,
  SUBJECT_KIND,
  SUBJECT_LABEL_PREFIX,
  KIND_WORKLOAD,
  KIND_NODE,
  KIND_IDLE,
  KIND_CLUSTER,
  METRIC_CPU_REQUEST,
  METRIC_MEM_REQUEST,
  METRIC_CPU_ALLOCATABLE,
  METRIC_MEM_ALLOCATABLE,
  METRIC_CPU_USAGE,
  METRIC_MEM_USAGE,
  UNIT_CORE,
  UNIT_GIB,
  UNIT_CORE_HOURS,
  UNIT_GIB_HOURS,
} from "./utils/usage-subjects.js";
