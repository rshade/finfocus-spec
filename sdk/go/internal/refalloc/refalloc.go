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

// Package refalloc is the reference allocator the SDK's own tests serve and
// certify. It is a test fixture only, not a template for production
// allocators: it implements the simplest math that satisfies the
// AllocatorService invariants, through the same public helpers
// (ValidateAllocateRequest, DecodePolicy, ResolveCurrency) a real allocator
// uses.
package refalloc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// PolicyVersion is the only policy version the reference allocator accepts.
const PolicyVersion = 1

// DefaultCPUWeight is the default fraction of node cost attributed to CPU.
const DefaultCPUWeight = 0.5

// resourceKindCluster is the PricedResource tags.kind of a control plane.
const resourceKindCluster = "cluster"

// NodeSplit divides a node's cost between CPU and memory.
type NodeSplit struct {
	// CPUWeight is the fraction of node cost attributed to CPU, in [0, 1].
	// Memory receives the remainder.
	CPUWeight float64 `json:"cpu_weight"`
}

// Policy is the reference allocator's policy document.
type Policy struct {
	// Version is the policy schema version; only PolicyVersion is accepted.
	Version int `json:"version"`
	// NodeSplit configures the CPU/memory split of node cost.
	NodeSplit NodeSplit `json:"node_split"`
}

// DefaultPolicy returns the policy applied when policy_json is empty or "{}".
func DefaultPolicy() Policy {
	return Policy{Version: PolicyVersion, NodeSplit: NodeSplit{CPUWeight: DefaultCPUWeight}}
}

// Allocator is the reference AllocatorProvider.
type Allocator struct{}

// New returns a reference allocator.
func New() *Allocator {
	return &Allocator{}
}

type capacity struct {
	cpu, mem float64
}

type workload struct {
	key      string
	subject  map[string]string
	cpu, mem float64
}

type usageIndex struct {
	capacity  map[string]*capacity
	workloads map[string][]*workload // by node name; "" for workloads without a node
}

// Allocate divides priced nodes across the workloads that request them,
// reporting unclaimed capacity as one idle row per priced node and non-node
// priced resources as cluster rows.
func (a *Allocator) Allocate(_ context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	if err := pluginsdk.ValidateAllocateRequest(req); err != nil {
		return nil, err
	}
	policy, err := decodePolicy(req.GetPolicyJson())
	if err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode effective policy: %v", err)
	}
	digest := sha256.Sum256(canonical)
	currency, err := pluginsdk.ResolveCurrency(req.GetPriced())
	if err != nil {
		return nil, err
	}

	b := &rowBuilder{currency: currency, policy: policy, usage: indexUsage(req.GetUsage())}
	b.allocate(req.GetPriced())

	return &pbc.AllocateResponse{
		Rows:                b.rows,
		EffectivePolicyJson: canonical,
		PolicyDigest:        hex.EncodeToString(digest[:]),
	}, nil
}

func decodePolicy(doc []byte) (Policy, error) {
	policy := DefaultPolicy()
	if err := pluginsdk.DecodePolicy(doc, &policy); err != nil {
		return Policy{}, err
	}
	if policy.Version != PolicyVersion {
		return Policy{}, status.Errorf(codes.InvalidArgument,
			"unsupported policy version %d (supported: %d)", policy.Version, PolicyVersion)
	}
	if w := policy.NodeSplit.CPUWeight; w < 0 || w > 1 {
		return Policy{}, status.Errorf(codes.InvalidArgument, "node_split.cpu_weight %v must be in [0, 1]", w)
	}
	return policy, nil
}

func nonNegative(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func subjectKey(subject map[string]string) string {
	keys := make([]string, 0, len(subject))
	for k := range subject {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(subject[k])
		b.WriteByte(0)
	}
	return b.String()
}

func indexUsage(rows []*pbc.UsageRow) *usageIndex {
	idx := &usageIndex{capacity: map[string]*capacity{}, workloads: map[string][]*workload{}}
	byKey := map[string]*workload{}
	for _, row := range rows {
		subject := row.GetSubject()
		switch subject[pluginsdk.SubjectKind] {
		case pluginsdk.KindNode:
			idx.addCapacity(subject[pluginsdk.SubjectNode], row)
		case pluginsdk.KindWorkload:
			key := subjectKey(subject)
			w, ok := byKey[key]
			if !ok {
				w = &workload{key: key, subject: subject}
				byKey[key] = w
				node := subject[pluginsdk.SubjectNode]
				idx.workloads[node] = append(idx.workloads[node], w)
			}
			w.addRequest(row)
		}
	}
	for _, ws := range idx.workloads {
		sort.Slice(ws, func(i, j int) bool { return ws[i].key < ws[j].key })
	}
	return idx
}

func (idx *usageIndex) addCapacity(node string, row *pbc.UsageRow) {
	c, ok := idx.capacity[node]
	if !ok {
		c = &capacity{}
		idx.capacity[node] = c
	}
	switch row.GetMetric() {
	case pluginsdk.MetricCPUAllocatable:
		c.cpu += nonNegative(row.GetAmount())
	case pluginsdk.MetricMemAllocatable:
		c.mem += nonNegative(row.GetAmount())
	}
}

func (w *workload) addRequest(row *pbc.UsageRow) {
	switch row.GetMetric() {
	case pluginsdk.MetricCPURequest:
		w.cpu += nonNegative(row.GetAmount())
	case pluginsdk.MetricMemRequest:
		w.mem += nonNegative(row.GetAmount())
	}
}

type rowBuilder struct {
	currency string
	policy   Policy
	usage    *usageIndex
	rows     []*pbc.AllocationRow
}

func (b *rowBuilder) allocate(priced []*pbc.PricedResource) {
	sorted := make([]*pbc.PricedResource, len(priced))
	copy(sorted, priced)
	sort.SliceStable(sorted, func(i, j int) bool {
		ri, rj := sorted[i].GetResource(), sorted[j].GetResource()
		if ki, kj := ri.GetTags()[pluginsdk.SubjectKind], rj.GetTags()[pluginsdk.SubjectKind]; ki != kj {
			return ki < kj
		}
		return ri.GetId() < rj.GetId()
	})

	handled := map[string]bool{}
	for _, entry := range sorted {
		id := entry.GetResource().GetId()
		switch kind := entry.GetResource().GetTags()[pluginsdk.SubjectKind]; {
		case kind == pluginsdk.KindNode && entry.GetPriced():
			b.allocateNode(id, entry.GetCost())
			handled[id] = true
		case kind == pluginsdk.KindNode:
			b.zeroWorkloads(id, fmt.Sprintf("node %q is not priced: %s", id, entry.GetNote()))
			handled[id] = true
		case entry.GetPriced():
			b.clusterRow(id, kind, entry.GetCost())
		}
	}

	nodes := make([]string, 0, len(b.usage.workloads))
	for node := range b.usage.workloads {
		if !handled[node] {
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		b.zeroWorkloads(node, fmt.Sprintf("node %q has no priced resource", node))
	}
}

// shares divides pool by each request's fraction of allocatable, scaling the
// requests down when together they exceed allocatable.
func shares(pool, allocatable float64, requests []float64) ([]float64, float64) {
	out := make([]float64, len(requests))
	if allocatable <= 0 {
		return out, pool
	}
	total := 0.0
	for _, r := range requests {
		total += r
	}
	scale := 1.0
	if total > allocatable {
		scale = allocatable / total
	}
	claimed := 0.0
	for i, r := range requests {
		out[i] = pool * r / allocatable * scale
		claimed += out[i]
	}
	return out, math.Max(0, pool-claimed)
}

func (b *rowBuilder) allocateNode(node string, cost float64) {
	cpuPool := cost * b.policy.NodeSplit.CPUWeight
	memPool := cost - cpuPool

	var capa capacity
	if c := b.usage.capacity[node]; c != nil {
		capa = *c
	}
	workloads := b.usage.workloads[node]
	cpuReq := make([]float64, len(workloads))
	memReq := make([]float64, len(workloads))
	for i, w := range workloads {
		cpuReq[i], memReq[i] = w.cpu, w.mem
	}
	cpuShares, cpuIdle := shares(cpuPool, capa.cpu, cpuReq)
	memShares, memIdle := shares(memPool, capa.mem, memReq)

	for i, w := range workloads {
		b.rows = append(b.rows, b.row(copySubject(w.subject), cpuShares[i], memShares[i], ""))
	}
	idle := map[string]string{pluginsdk.SubjectKind: pluginsdk.KindIdle, pluginsdk.SubjectNode: node}
	b.rows = append(b.rows, b.row(idle, cpuIdle, memIdle, ""))
}

func (b *rowBuilder) zeroWorkloads(node, note string) {
	for _, w := range b.usage.workloads[node] {
		b.rows = append(b.rows, b.row(copySubject(w.subject), 0, 0, note))
	}
}

func (b *rowBuilder) clusterRow(id, kind string, cost float64) {
	subject := map[string]string{pluginsdk.SubjectKind: pluginsdk.KindCluster}
	note := ""
	if kind == resourceKindCluster {
		subject[pluginsdk.SubjectCluster] = id
	} else {
		note = fmt.Sprintf("priced resource %q has kind %q; allocated as shared cost", id, kind)
	}
	b.rows = append(b.rows, &pbc.AllocationRow{
		Subject:   subject,
		TotalCost: cost,
		Currency:  b.currency,
		Note:      note,
	})
}

func (b *rowBuilder) row(subject map[string]string, cpu, mem float64, note string) *pbc.AllocationRow {
	return &pbc.AllocationRow{
		Subject:   subject,
		CpuCost:   cpu,
		MemCost:   mem,
		TotalCost: cpu + mem,
		Currency:  b.currency,
		Note:      note,
	}
}

func copySubject(subject map[string]string) map[string]string {
	out := make(map[string]string, len(subject))
	for k, v := range subject {
		out[k] = v
	}
	return out
}
