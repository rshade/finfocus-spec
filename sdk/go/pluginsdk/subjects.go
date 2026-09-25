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

package pluginsdk

// Vocabulary for UsageSourceService.GetStats rows. See docs/usage-source.md.
const (
	// SubjectCluster is the subject key naming the cluster a row belongs to.
	SubjectCluster = "cluster"
	// SubjectNamespace is the subject key naming a workload's namespace.
	SubjectNamespace = "namespace"
	// SubjectControllerKind is the subject key naming the owning controller's
	// kind, such as "Deployment" or "StatefulSet".
	SubjectControllerKind = "controller_kind"
	// SubjectController is the subject key naming the owning controller.
	SubjectController = "controller"
	// SubjectPod is the subject key naming a pod.
	SubjectPod = "pod"
	// SubjectNode is the subject key naming a node. It is the join key between
	// usage rows and priceable node entries.
	SubjectNode = "node"
	// SubjectKind is the required subject key holding the row kind
	// (KindWorkload or KindNode).
	SubjectKind = "kind"
	// SubjectLabelPrefix prefixes subject keys that carry workload labels,
	// for example "label.app.kubernetes.io/name".
	SubjectLabelPrefix = "label."

	// KindWorkload marks a row that measures a workload.
	KindWorkload = "workload"
	// KindNode marks a row that measures node capacity.
	KindNode = "node"
	// KindIdle is reserved for allocator output (#506) and is invalid in
	// usage rows.
	KindIdle = "__idle__"
	// KindCluster is reserved for allocator output (#506) and is invalid in
	// usage rows.
	KindCluster = "__cluster__"

	// MetricCPURequest is the CPU requested by a workload.
	MetricCPURequest = "cpu_request"
	// MetricMemRequest is the memory requested by a workload.
	MetricMemRequest = "mem_request"
	// MetricCPUAllocatable is the CPU a node can allocate to workloads.
	MetricCPUAllocatable = "cpu_allocatable"
	// MetricMemAllocatable is the memory a node can allocate to workloads.
	MetricMemAllocatable = "mem_allocatable"
	// MetricCPUUsage is the CPU a workload actually consumed.
	MetricCPUUsage = "cpu_usage"
	// MetricMemUsage is the memory a workload actually consumed.
	MetricMemUsage = "mem_usage"

	// UnitCore is the per-hour CPU unit used in STATS_MODE_RUN_RATE.
	UnitCore = "core"
	// UnitGiB is the per-hour memory unit used in STATS_MODE_RUN_RATE.
	UnitGiB = "GiB"
	// UnitCoreHours is the integrated CPU unit used in STATS_MODE_HISTORICAL.
	UnitCoreHours = "core-hours"
	// UnitGiBHours is the integrated memory unit used in STATS_MODE_HISTORICAL.
	UnitGiBHours = "GiB-hours"
)
