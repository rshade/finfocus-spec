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

// Vocabulary for UsageSourceService.GetStats rows. Values mirror
// sdk/go/pluginsdk/subjects.go exactly; see docs/usage-source.md.

/** Subject key naming the cluster a row belongs to. */
export const SUBJECT_CLUSTER = "cluster";
/** Subject key naming a workload's namespace. */
export const SUBJECT_NAMESPACE = "namespace";
/** Subject key naming the owning controller's kind, such as "Deployment". */
export const SUBJECT_CONTROLLER_KIND = "controller_kind";
/** Subject key naming the owning controller. */
export const SUBJECT_CONTROLLER = "controller";
/** Subject key naming a pod. */
export const SUBJECT_POD = "pod";
/** Subject key naming a node; the join key to priceable node entries. */
export const SUBJECT_NODE = "node";
/** Required subject key holding the row kind (KIND_WORKLOAD or KIND_NODE). */
export const SUBJECT_KIND = "kind";
/** Prefix for subject keys that carry workload labels, e.g. "label.app". */
export const SUBJECT_LABEL_PREFIX = "label.";

/** Row kind for workload measurements. */
export const KIND_WORKLOAD = "workload";
/** Row kind for node capacity measurements. */
export const KIND_NODE = "node";
/** Reserved for allocator output (#506); invalid in usage rows. */
export const KIND_IDLE = "__idle__";
/** Reserved for allocator output (#506); invalid in usage rows. */
export const KIND_CLUSTER = "__cluster__";

/** CPU requested by a workload. */
export const METRIC_CPU_REQUEST = "cpu_request";
/** Memory requested by a workload. */
export const METRIC_MEM_REQUEST = "mem_request";
/** CPU a node can allocate to workloads. */
export const METRIC_CPU_ALLOCATABLE = "cpu_allocatable";
/** Memory a node can allocate to workloads. */
export const METRIC_MEM_ALLOCATABLE = "mem_allocatable";
/** CPU a workload actually consumed. */
export const METRIC_CPU_USAGE = "cpu_usage";
/** Memory a workload actually consumed. */
export const METRIC_MEM_USAGE = "mem_usage";

/** Per-hour CPU unit used in StatsMode.RUN_RATE. */
export const UNIT_CORE = "core";
/** Per-hour memory unit used in StatsMode.RUN_RATE. */
export const UNIT_GIB = "GiB";
/** Integrated CPU unit used in StatsMode.HISTORICAL. */
export const UNIT_CORE_HOURS = "core-hours";
/** Integrated memory unit used in StatsMode.HISTORICAL. */
export const UNIT_GIB_HOURS = "GiB-hours";
