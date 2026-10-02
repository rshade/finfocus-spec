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

package refalloc_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/internal/refalloc"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const defaultPolicy = `{"version":1,"node_split":{"cpu_weight":0.5}}`

func nodeUsage(node string, cpu, mem float64) []*pbc.UsageRow {
	subject := map[string]string{pluginsdk.SubjectKind: pluginsdk.KindNode, pluginsdk.SubjectNode: node}
	return []*pbc.UsageRow{
		{Subject: subject, Metric: pluginsdk.MetricCPUAllocatable, Amount: cpu, Unit: pluginsdk.UnitCore},
		{Subject: subject, Metric: pluginsdk.MetricMemAllocatable, Amount: mem, Unit: pluginsdk.UnitGiB},
	}
}

func workloadUsage(node, pod string, cpu, mem float64) []*pbc.UsageRow {
	subject := map[string]string{
		pluginsdk.SubjectKind:      pluginsdk.KindWorkload,
		pluginsdk.SubjectNamespace: "default",
		pluginsdk.SubjectPod:       pod,
		pluginsdk.SubjectNode:      node,
	}
	return []*pbc.UsageRow{
		{Subject: subject, Metric: pluginsdk.MetricCPURequest, Amount: cpu, Unit: pluginsdk.UnitCore},
		{Subject: subject, Metric: pluginsdk.MetricMemRequest, Amount: mem, Unit: pluginsdk.UnitGiB},
	}
}

func singleNodeRequest(cpuPerPod, memPerPod float64) *pbc.AllocateRequest {
	usage := nodeUsage("n1", 4, 16)
	usage = append(usage, workloadUsage("n1", "a", cpuPerPod, memPerPod)...)
	usage = append(usage, workloadUsage("n1", "b", cpuPerPod, memPerPod)...)
	return &pbc.AllocateRequest{
		Usage: usage,
		Priced: []*pbc.PricedResource{
			{
				Resource: &pbc.ResourceDescriptor{
					Id:   "n1",
					Tags: map[string]string{pluginsdk.SubjectKind: pluginsdk.KindNode},
				},
				Cost:     10,
				Currency: "USD",
				Priced:   true,
			},
		},
	}
}

func allocate(t *testing.T, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	t.Helper()
	return refalloc.New().Allocate(context.Background(), req)
}

func rowsOfKind(resp *pbc.AllocateResponse, kind string) []*pbc.AllocationRow {
	var rows []*pbc.AllocationRow
	for _, row := range resp.GetRows() {
		if row.GetSubject()[pluginsdk.SubjectKind] == kind {
			rows = append(rows, row)
		}
	}
	return rows
}

func TestAllocateEmptyRequest(t *testing.T) {
	resp, err := allocate(t, &pbc.AllocateRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.GetRows())
	assert.JSONEq(t, defaultPolicy, string(resp.GetEffectivePolicyJson()))

	sum := sha256.Sum256(resp.GetEffectivePolicyJson())
	assert.Equal(t, hex.EncodeToString(sum[:]), resp.GetPolicyDigest())
	assert.Regexp(t, `^[0-9a-f]{64}$`, resp.GetPolicyDigest())
}

func TestAllocateEmptyPolicyEqualsBraces(t *testing.T) {
	empty, err := allocate(t, &pbc.AllocateRequest{})
	require.NoError(t, err)
	braces, err := allocate(t, &pbc.AllocateRequest{PolicyJson: []byte("{}")})
	require.NoError(t, err)
	assert.Equal(t, empty.GetPolicyDigest(), braces.GetPolicyDigest())
}

func TestAllocateRejectsPolicy(t *testing.T) {
	for _, doc := range []string{`{"version":2}`, `{"node_split":{"cpu_weight":1.5}}`} {
		t.Run(doc, func(t *testing.T) {
			_, err := allocate(t, &pbc.AllocateRequest{PolicyJson: []byte(doc)})
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestAllocateSingleNode(t *testing.T) {
	resp, err := allocate(t, singleNodeRequest(1, 4))
	require.NoError(t, err)

	workloads := rowsOfKind(resp, pluginsdk.KindWorkload)
	require.Len(t, workloads, 2)
	for _, row := range workloads {
		assert.InDelta(t, 2.5, row.GetTotalCost(), 1e-9)
		assert.Equal(t, "USD", row.GetCurrency())
	}

	idle := rowsOfKind(resp, pluginsdk.KindIdle)
	require.Len(t, idle, 1)
	assert.Equal(t, "n1", idle[0].GetSubject()[pluginsdk.SubjectNode])
	assert.InDelta(t, 5, idle[0].GetTotalCost(), 1e-9)
	assert.Equal(t, "USD", idle[0].GetCurrency())
}

func TestAllocateOverRequestedNode(t *testing.T) {
	resp, err := allocate(t, singleNodeRequest(4, 4))
	require.NoError(t, err)

	idle := rowsOfKind(resp, pluginsdk.KindIdle)
	require.Len(t, idle, 1)
	assert.GreaterOrEqual(t, idle[0].GetCpuCost(), 0.0)
	assert.GreaterOrEqual(t, idle[0].GetTotalCost(), 0.0)

	total := 0.0
	for _, row := range resp.GetRows() {
		total += row.GetTotalCost()
	}
	assert.InDelta(t, 10, total, 1e-9)
}

func TestAllocateRowProvenance(t *testing.T) {
	req := singleNodeRequest(1, 4)
	req.Priced = append(req.Priced,
		&pbc.PricedResource{
			Resource: &pbc.ResourceDescriptor{
				Id:   "cp1",
				Tags: map[string]string{pluginsdk.SubjectKind: "cluster"},
			},
			Cost:     3,
			Currency: "USD",
			Priced:   true,
		},
		&pbc.PricedResource{
			Resource: &pbc.ResourceDescriptor{
				Id:   "n2",
				Tags: map[string]string{pluginsdk.SubjectKind: pluginsdk.KindNode},
			},
			Priced: false,
		},
	)
	req.Usage = append(req.Usage, nodeUsage("n2", 4, 16)...)
	req.Usage = append(req.Usage, workloadUsage("n2", "c", 1, 4)...)

	resp, err := allocate(t, req)
	require.NoError(t, err)
	require.NoError(t, pluginsdk.ValidateAllocateResponse(req, resp))

	want := map[string]string{
		pluginsdk.KindWorkload: "n1",
		pluginsdk.KindIdle:     "n1",
		pluginsdk.KindCluster:  "cp1",
	}
	for _, row := range resp.GetRows() {
		assert.Equal(t, refalloc.MethodID, row.GetAllocatedMethodId())
		assert.NotEmpty(t, row.GetAllocatedResourceId())
		if row.GetSubject()[pluginsdk.SubjectPod] == "c" {
			assert.Equal(t, "n2", row.GetAllocatedResourceId())
			continue
		}
		assert.Equal(t, want[row.GetSubject()[pluginsdk.SubjectKind]], row.GetAllocatedResourceId())
	}
}

func TestAllocateEchoesWindow(t *testing.T) {
	plain, err := allocate(t, singleNodeRequest(1, 2))
	require.NoError(t, err)
	assert.Nil(t, plain.GetStart())
	assert.Nil(t, plain.GetEnd())

	req := singleNodeRequest(1, 2)
	req.Start = &timestamppb.Timestamp{Seconds: 1_790_000_000, Nanos: 5}
	req.End = &timestamppb.Timestamp{Seconds: 1_790_086_400}
	windowed, err := allocate(t, req)
	require.NoError(t, err)
	assert.True(t, proto.Equal(req.GetStart(), windowed.GetStart()))
	assert.True(t, proto.Equal(req.GetEnd(), windowed.GetEnd()))
	require.NoError(t, pluginsdk.ValidateAllocateResponse(req, windowed))

	require.Len(t, windowed.GetRows(), len(plain.GetRows()))
	for i := range plain.GetRows() {
		assert.True(t, proto.Equal(plain.GetRows()[i], windowed.GetRows()[i]), "row %d changed with a window", i)
	}
}

func TestAllocateSelectorWarns(t *testing.T) {
	const partial = "selector narrows workloads"

	req := singleNodeRequest(1, 2)
	req.Selector = map[string]string{"namespace": "payments"}
	resp, err := allocate(t, req)
	require.NoError(t, err)
	require.NoError(t, pluginsdk.ValidateAllocateResponse(req, resp))
	require.NoError(t, pluginsdk.CheckConservation(req, resp, pluginsdk.DefaultConservationEpsilon))
	require.Len(t, rowsOfKind(resp, pluginsdk.KindIdle), 1)
	assert.True(t, containsWarning(resp, partial), "warnings: %v", resp.GetWarnings())

	plain, err := allocate(t, singleNodeRequest(1, 2))
	require.NoError(t, err)
	assert.False(t, containsWarning(plain, partial))
}

func containsWarning(resp *pbc.AllocateResponse, substr string) bool {
	for _, w := range resp.GetWarnings() {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
}
