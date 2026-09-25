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

//nolint:testpackage // Fixtures for tests of the unexported usage-source adapters
package pluginsdk

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

const (
	fixtureNodeA = "ip-10-0-1-5"
	fixtureNodeB = "ip-10-0-2-7"
)

// usageTestPlugin is a usage source that returns a fixed response or error.
type usageTestPlugin struct {
	*BasePlugin

	resp *pbc.GetStatsResponse
	err  error
}

func newUsageTestPlugin() *usageTestPlugin {
	return &usageTestPlugin{BasePlugin: NewBasePlugin("usage-test"), resp: fixtureStatsResponse()}
}

func (p *usageTestPlugin) GetStats(_ context.Context, _ *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.resp, nil
}

func workloadSubject(namespace, controller, pod, node string) map[string]string {
	return map[string]string{
		SubjectKind:           KindWorkload,
		SubjectCluster:        "c1",
		SubjectNamespace:      namespace,
		SubjectControllerKind: "Deployment",
		SubjectController:     controller,
		SubjectPod:            pod,
		SubjectNode:           node,
	}
}

func nodeSubject(node string) map[string]string {
	return map[string]string{SubjectKind: KindNode, SubjectCluster: "c1", SubjectNode: node}
}

func fixturePriceableNode(node, providerID string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Id:           node,
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "m5.large",
		Region:       "us-east-1",
		Tags: map[string]string{
			"kind":          "node",
			"provider_id":   providerID,
			"capacity_type": "on-demand",
		},
	}
}

// fixtureStatsResponse is a two-node cluster in run-rate mode with workloads
// in the payments and web namespaces.
func fixtureStatsResponse() *pbc.GetStatsResponse {
	payments := workloadSubject("payments", "api", "api-7d9f", fixtureNodeA)
	web := workloadSubject("web", "frontend", "frontend-5c2a", fixtureNodeB)
	return &pbc.GetStatsResponse{
		Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
		Rows: []*pbc.UsageRow{
			{Subject: payments, Metric: MetricCPURequest, Amount: 0.5, Unit: UnitCore},
			{Subject: payments, Metric: MetricMemRequest, Amount: 1, Unit: UnitGiB},
			{Subject: web, Metric: MetricCPURequest, Amount: 0.25, Unit: UnitCore},
			{Subject: web, Metric: MetricMemRequest, Amount: 0.5, Unit: UnitGiB},
			{Subject: nodeSubject(fixtureNodeA), Metric: MetricCPUAllocatable, Amount: 1.93, Unit: UnitCore},
			{Subject: nodeSubject(fixtureNodeA), Metric: MetricMemAllocatable, Amount: 7.2, Unit: UnitGiB},
			{Subject: nodeSubject(fixtureNodeB), Metric: MetricCPUAllocatable, Amount: 1.93, Unit: UnitCore},
			{Subject: nodeSubject(fixtureNodeB), Metric: MetricMemAllocatable, Amount: 7.2, Unit: UnitGiB},
		},
		Priceable: []*pbc.ResourceDescriptor{
			fixturePriceableNode(fixtureNodeA, "aws:///us-east-1a/i-0a1b2c3d4e5f60001"),
			fixturePriceableNode(fixtureNodeB, "aws:///us-east-1b/i-0a1b2c3d4e5f60002"),
		},
		Warnings: []string{"mem_usage unavailable: metrics-server not installed"},
	}
}

// referenceUsageSource behaves like a real usage source: it validates the
// request, returns failure (a simulated RBAC or credential error) when set,
// and serves run-rate data (or historical data when historical is true).
type referenceUsageSource struct {
	*BasePlugin

	historical bool
	failure    error
}

func newReferenceUsageSource() *referenceUsageSource {
	return &referenceUsageSource{BasePlugin: NewBasePlugin("usage-reference")}
}

//nolint:gochecknoglobals // Test fixture lookup table.
var documentedMetrics = map[string]bool{
	MetricCPURequest: true, MetricMemRequest: true,
	MetricCPUAllocatable: true, MetricMemAllocatable: true,
	MetricCPUUsage: true, MetricMemUsage: true,
}

//nolint:gochecknoglobals // Test fixture lookup table.
var historicalUnits = map[string]string{UnitCore: UnitCoreHours, UnitGiB: UnitGiBHours}

func (s *referenceUsageSource) GetStats(_ context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
	if err := plugintesting.ValidateGetStatsRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if s.failure != nil {
		return nil, s.failure
	}

	fixture := fixtureStatsResponse()
	resp := &pbc.GetStatsResponse{Mode: fixture.GetMode(), Priceable: fixture.GetPriceable()}

	hours := 0.0
	if req.GetStart() != nil {
		if !s.historical {
			return nil, status.Error(codes.InvalidArgument, "historical mode not supported")
		}
		hours = req.GetEnd().AsTime().Sub(req.GetStart().AsTime()).Hours()
		resp.Mode = pbc.StatsMode_STATS_MODE_HISTORICAL
	}

	namespace, filterNamespace := req.GetSelector()[SubjectNamespace]
	for _, row := range fixture.GetRows() {
		subject := row.GetSubject()
		if filterNamespace && subject[SubjectKind] == KindWorkload && subject[SubjectNamespace] != namespace {
			continue
		}
		if resp.GetMode() == pbc.StatsMode_STATS_MODE_HISTORICAL {
			row.Amount *= hours
			row.Unit = historicalUnits[row.GetUnit()]
		}
		resp.Rows = append(resp.Rows, row)
	}

	for _, metric := range req.GetMetrics() {
		if !documentedMetrics[metric] {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("unknown metric %q ignored", metric))
		}
	}
	return resp, nil
}
