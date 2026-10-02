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

package testing_test

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/internal/refalloc"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

//nolint:gochecknoglobals // Expected scenario names, in suite order.
var allocatorScenarioNames = []string{
	"single_node",
	"three_nodes",
	"empty_cluster",
	"fully_packed_node",
	"unpriced_node",
	"control_plane",
	"over_requested_node",
	"policy_unknown_field",
	"policy_unknown_version",
	"empty_request",
	"fingerprint_stable",
	"fingerprint_empty_equals_braces",
	"row_provenance",
	"period_echoed",
	"selector_keeps_invariants",
}

// allocFunc adapts a function to plugintesting.AllocateServer.
type allocFunc func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)

func (f allocFunc) Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	return f(ctx, req)
}

// postProcess wraps the reference allocator and rewrites its successful responses.
func postProcess(mutate func(resp *pbc.AllocateResponse)) allocFunc {
	ref := refalloc.New()
	return func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
		resp, err := ref.Allocate(ctx, req)
		if err == nil {
			mutate(resp)
		}
		return resp, err
	}
}

// rewritePolicy wraps the reference allocator and rewrites the policy it receives.
func rewritePolicy(rewrite func(doc []byte) []byte) allocFunc {
	ref := refalloc.New()
	return func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
		clone := &pbc.AllocateRequest{
			Usage: req.GetUsage(), Priced: req.GetPriced(), Mode: req.GetMode(),
			PolicyJson: rewrite(req.GetPolicyJson()),
		}
		return ref.Allocate(ctx, clone)
	}
}

func firstRowIndex(resp *pbc.AllocateResponse, kind string) int {
	for i, row := range resp.GetRows() {
		if row.GetSubject()["kind"] == kind {
			return i
		}
	}
	return -1
}

func runScenarios(t *testing.T, impl plugintesting.AllocateServer) map[string]error {
	t.Helper()
	harness := plugintesting.NewAllocatorHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return plugintesting.RunAllocatorScenariosForTest(ctx, harness.Client())
}

func TestAllocatorConformance_Reference(t *testing.T) {
	plugintesting.RunAllocatorConformance(t, refalloc.New())
}

func TestAllocatorConformance_ScenarioNames(t *testing.T) {
	results := runScenarios(t, refalloc.New())

	names := make([]string, 0, len(results))
	for name, err := range results {
		names = append(names, name)
		require.NoError(t, err, name)
	}
	sort.Strings(names)
	want := append([]string(nil), allocatorScenarioNames...)
	sort.Strings(want)
	assert.Equal(t, want, names)
}

// mapFieldAllocator wraps the reference allocator with a map-valued policy
// field, "a_labels", that accepts any key and sorts before every struct field.
func mapFieldAllocator() allocFunc {
	ref := refalloc.New()
	return func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
		doc := req.GetPolicyJson()
		var obj map[string]any
		if json.Unmarshal(doc, &obj) == nil && obj != nil {
			delete(obj, "a_labels")
			doc, _ = json.Marshal(obj)
		}
		clone := proto.CloneOf(req)
		clone.PolicyJson = doc
		resp, err := ref.Allocate(ctx, clone)
		if err != nil {
			return nil, err
		}
		var effective map[string]any
		if err = json.Unmarshal(resp.GetEffectivePolicyJson(), &effective); err != nil {
			return nil, err
		}
		effective["a_labels"] = map[string]any{}
		if resp.EffectivePolicyJson, err = json.Marshal(effective); err != nil {
			return nil, err
		}
		return resp, nil
	}
}

func TestAllocatorConformance_MapPolicyFieldPasses(t *testing.T) {
	for name, err := range runScenarios(t, mapFieldAllocator()) {
		require.NoError(t, err, name)
	}
}

// brokenAllocator is a deliberately wrong allocator and the scenarios that
// must catch it.
type brokenAllocator struct {
	name  string
	impl  plugintesting.AllocateServer
	fails []string
	check func(t *testing.T, results map[string]error)
}

func dropRowsOfKind(resp *pbc.AllocateResponse, kind string) {
	kept := resp.GetRows()[:0]
	for _, row := range resp.GetRows() {
		if row.GetSubject()["kind"] != kind {
			kept = append(kept, row)
		}
	}
	resp.Rows = kept
}

func negateIdle(resp *pbc.AllocateResponse) {
	for _, row := range resp.GetRows() {
		if row.GetSubject()["kind"] == "__idle__" {
			row.CpuCost, row.TotalCost = -1, -1
		}
	}
}

// lenientPolicy decodes like encoding/json, silently dropping unknown fields.
func lenientPolicy(doc []byte) []byte {
	policy := refalloc.DefaultPolicy()
	_ = json.Unmarshal(doc, &policy)
	out, _ := json.Marshal(policy)
	return out
}

// currentVersionPolicy rewrites any version to the supported one.
func currentVersionPolicy(doc []byte) []byte {
	var obj map[string]any
	if json.Unmarshal(doc, &obj) != nil || obj == nil {
		return doc
	}
	if _, ok := obj["version"]; ok {
		obj["version"] = refalloc.PolicyVersion
	}
	out, _ := json.Marshal(obj)
	return out
}

func bracesDiffer() allocFunc {
	ref := refalloc.New()
	return func(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
		resp, err := ref.Allocate(ctx, req)
		if err == nil && string(req.GetPolicyJson()) == "{}" {
			resp.PolicyDigest = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		}
		return resp, err
	}
}

func brokenAllocators() []brokenAllocator {
	var digestCounter atomic.Int64
	return []brokenAllocator{
		{
			name: "drops-window",
			impl: postProcess(func(resp *pbc.AllocateResponse) {
				resp.Start, resp.End = nil, nil
			}),
			fails: []string{"period_echoed"},
		},
		{
			name: "over-allocation",
			impl: postProcess(func(resp *pbc.AllocateResponse) {
				if i := firstRowIndex(resp, "workload"); i >= 0 {
					resp.Rows[i].TotalCost++
					resp.Rows[i].CpuCost++
				}
			}),
			fails: []string{"single_node"},
		},
		{
			name: "under-allocation",
			impl: postProcess(func(resp *pbc.AllocateResponse) {
				if i := firstRowIndex(resp, "workload"); i >= 0 {
					resp.Rows = slices.Delete(resp.GetRows(), i, i+1)
				}
			}),
			fails: []string{"single_node"},
		},
		{
			name:  "dropped idle",
			impl:  postProcess(func(resp *pbc.AllocateResponse) { dropRowsOfKind(resp, "__idle__") }),
			fails: []string{"single_node", "empty_cluster"},
			check: func(t *testing.T, results map[string]error) {
				require.ErrorContains(t, results["empty_cluster"], "__idle__")
				assert.ErrorIs(t, results["empty_cluster"], plugintesting.ErrConservation)
			},
		},
		{
			name:  "negative idle",
			impl:  postProcess(negateIdle),
			fails: []string{"over_requested_node"},
		},
		{
			name:  "ignores unknown fields",
			impl:  rewritePolicy(lenientPolicy),
			fails: []string{"policy_unknown_field"},
		},
		{
			name:  "accepts unknown version",
			impl:  rewritePolicy(currentVersionPolicy),
			fails: []string{"policy_unknown_version"},
		},
		{
			name: "unstable digest",
			impl: postProcess(func(resp *pbc.AllocateResponse) {
				resp.PolicyDigest += strconv.FormatInt(digestCounter.Add(1), 10)
			}),
			fails: []string{"fingerprint_stable"},
		},
		{
			name:  "braces differ",
			impl:  bracesDiffer(),
			fails: []string{"fingerprint_empty_equals_braces"},
		},
	}
}

func TestAllocatorConformance_RejectsBrokenAllocators(t *testing.T) {
	for _, tt := range brokenAllocators() {
		t.Run(tt.name, func(t *testing.T) {
			results := runScenarios(t, tt.impl)
			for _, scenario := range tt.fails {
				require.Contains(t, results, scenario)
				require.Error(t, results[scenario], "%s should fail %s", tt.name, scenario)
			}
			if tt.check != nil {
				tt.check(t, results)
			}
		})
	}
}

func TestAllocatorHarness(t *testing.T) {
	harness := plugintesting.NewAllocatorHarness(refalloc.New())
	harness.Start(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := harness.Client().Allocate(ctx, &pbc.AllocateRequest{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetPolicyDigest())

	harness.Stop()
	harness.Stop()
}

// TestCopiedVocabularyMatchesPluginsdk guards the row kinds and metric names
// this package copies because it cannot import pluginsdk.
func TestCopiedVocabularyMatchesPluginsdk(t *testing.T) {
	assert.Equal(t, []string{
		pluginsdk.KindWorkload, pluginsdk.KindIdle, pluginsdk.KindCluster,
		pluginsdk.MetricCPURequest, pluginsdk.MetricMemRequest,
		pluginsdk.MetricCPUAllocatable, pluginsdk.MetricMemAllocatable,
	}, plugintesting.CopiedVocabularyForTest())
}

func TestAllocatorConformance_RowProvenance(t *testing.T) {
	t.Run("reference passes", func(t *testing.T) {
		require.NoError(t, runScenarios(t, refalloc.New())["row_provenance"])
	})

	t.Run("allocator without provenance passes", func(t *testing.T) {
		impl := postProcess(func(resp *pbc.AllocateResponse) {
			for _, row := range resp.GetRows() {
				row.AllocatedMethodId = ""
				row.AllocatedMethodDetails = ""
				row.AllocatedResourceId = ""
			}
		})
		assert.NoError(t, runScenarios(t, impl)["row_provenance"])
	})

	t.Run("method id without resource id fails by name", func(t *testing.T) {
		impl := postProcess(func(resp *pbc.AllocateResponse) {
			for _, row := range resp.GetRows() {
				row.AllocatedResourceId = ""
			}
		})
		err := runScenarios(t, impl)["row_provenance"]
		require.Error(t, err)
		assert.Contains(t, err.Error(), "allocated_method_id")
		assert.Contains(t, err.Error(), "allocated_resource_id")
	})
}
