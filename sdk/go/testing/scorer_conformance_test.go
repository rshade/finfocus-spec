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
	"fmt"
	"slices"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

//nolint:gochecknoglobals // Expected scenario names, in suite order.
var scorerScenarioNames = []string{
	"single_recommendation",
	"mixed_batch",
	"reversed_order",
	"signal_subset",
	"unsupported_signal",
	"unspecified_signal",
	"identifier_modes",
	"empty_request",
	"duplicate_ids",
	"oversize_batch",
	"session_echo",
	"session_across_batches",
	"session_isolation",
}

// scoreFunc adapts a function to plugintesting.ScoreServer.
type scoreFunc func(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error)

func (f scoreFunc) ScoreRecommendations(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	return f(ctx, req)
}

// scorePostProcess wraps the mock scorer and rewrites its successful responses.
func scorePostProcess(mutate func(resp *pbc.ScoreRecommendationsResponse)) scoreFunc {
	ref := plugintesting.NewMockRecommendationScorer()
	return func(ctx context.Context, req *pbc.ScoreRecommendationsRequest) (*pbc.ScoreRecommendationsResponse, error) {
		resp, err := ref.ScoreRecommendations(ctx, req)
		if err == nil {
			mutate(resp)
		}
		return resp, err
	}
}

func runScorerScenarios(t *testing.T, impl plugintesting.ScoreServer) map[string]error {
	t.Helper()
	harness := plugintesting.NewScorerHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return plugintesting.RunScorerScenariosForTest(ctx, harness.Client())
}

func TestScorerConformance_Reference(t *testing.T) {
	plugintesting.RunScorerConformance(t, plugintesting.NewMockRecommendationScorer())
}

func TestScorerConformance_ScenarioNames(t *testing.T) {
	results := runScorerScenarios(t, plugintesting.NewMockRecommendationScorer())

	names := make([]string, 0, len(results))
	for name, err := range results {
		names = append(names, name)
		require.NoError(t, err, name)
	}
	sort.Strings(names)
	want := slices.Clone(scorerScenarioNames)
	sort.Strings(want)
	assert.Equal(t, want, names)
}

func TestScorerConformance_SubsetScorerPasses(t *testing.T) {
	subset := plugintesting.NewMockRecommendationScorer(
		plugintesting.WithScorerSignals(pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY),
		plugintesting.WithScorerMaxBatchSize(2),
	)
	for name, err := range runScorerScenarios(t, subset) {
		require.NoError(t, err, name)
	}
}

type brokenScorer struct {
	name  string
	impl  plugintesting.ScoreServer
	fails []string
}

func brokenScorers() []brokenScorer {
	firstScores := func(resp *pbc.ScoreRecommendationsResponse) *pbc.RecommendationScores {
		for _, result := range resp.GetResults() {
			if result.GetScores() != nil {
				return result.GetScores()
			}
		}
		return &pbc.RecommendationScores{}
	}
	return []brokenScorer{
		{
			name:  "session ids change per call",
			impl:  &driftingSessionScorer{},
			fails: []string{"session_across_batches"},
		},
		{
			name: "session echo differs",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				resp.SessionId = "other"
			}),
			fails: []string{"session_echo"},
		},
		{
			name: "session ignored in group ids",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				for _, result := range resp.GetResults() {
					if scores := result.GetScores(); scores.GetDuplicateGroupId() != "" {
						scores.DuplicateGroupId = "same-in-every-session"
					}
				}
			}),
			fails: []string{"session_isolation"},
		},
		{
			name: "misaligned results",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				slices.Reverse(resp.GetResults())
			}),
			fails: []string{"mixed_batch"},
		},
		{
			name: "wrong echoed id",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				resp.Results[0].RecommendationId += "-x"
			}),
			fails: []string{"single_recommendation"},
		},
		{
			name: "dropped result",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				resp.Results = resp.GetResults()[:len(resp.GetResults())-1]
			}),
			fails: []string{"single_recommendation"},
		},
		{
			name: "risk out of range",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				firstScores(resp).Risk = proto.Float64(1.5)
			}),
			fails: []string{"single_recommendation"},
		},
		{
			name: "priority out of range",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				firstScores(resp).Priority = proto.Float64(4)
			}),
			fails: []string{"single_recommendation"},
		},
		{
			name: "error with code OK",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				resp.Results[0] = &pbc.RecommendationScoreResult{
					RecommendationId: resp.GetResults()[0].GetRecommendationId(),
					Result: &pbc.RecommendationScoreResult_Error{
						Error: &pbc.ResourceError{Code: int32(codes.OK)},
					},
				}
			}),
			fails: []string{"single_recommendation"},
		},
		{
			name: "singleton duplicate group",
			impl: scorePostProcess(func(resp *pbc.ScoreRecommendationsResponse) {
				firstScores(resp).DuplicateGroupId = "lonely"
			}),
			fails: []string{"single_recommendation"},
		},
		{
			name:  "accepts bad requests and ignores the signal filter",
			impl:  laxScorer{},
			fails: []string{"signal_subset", "empty_request", "duplicate_ids", "oversize_batch", "unspecified_signal"},
		},
	}
}

// driftingSessionScorer honors sessions but numbers its group ids per call, so
// the same group has a different id in every batch.
type driftingSessionScorer struct {
	calls atomic.Int32
}

func (d *driftingSessionScorer) ScoreRecommendations(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	resp, err := plugintesting.NewMockRecommendationScorer().ScoreRecommendations(ctx, req)
	if err != nil {
		return nil, err
	}
	call := d.calls.Add(1)
	for _, result := range resp.GetResults() {
		if scores := result.GetScores(); scores.GetDuplicateGroupId() != "" {
			scores.DuplicateGroupId = fmt.Sprintf("%s-call-%d", scores.GetDuplicateGroupId(), call)
		}
	}
	return resp, nil
}

// laxScorer validates nothing: it answers every recommendation with a risk
// score and always supports all signals with a batch limit of 25.
type laxScorer struct{}

func (laxScorer) ScoreRecommendations(
	_ context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	resp := &pbc.ScoreRecommendationsResponse{MaxBatchSize: 25, SupportedSignals: allSignals()}
	for _, rec := range req.GetRecommendations() {
		resp.Results = append(resp.Results, scoredResult(rec.GetId(), &pbc.RecommendationScores{
			Risk: proto.Float64(0.1),
		}))
	}
	return resp, nil
}

func TestScorerConformance_RejectsBrokenScorers(t *testing.T) {
	for _, tt := range brokenScorers() {
		t.Run(tt.name, func(t *testing.T) {
			results := runScorerScenarios(t, tt.impl)
			for _, scenario := range tt.fails {
				require.Contains(t, results, scenario)
				require.Error(t, results[scenario], "%s should fail %s", tt.name, scenario)
			}
		})
	}
}

func TestScorerHarness(t *testing.T) {
	harness := plugintesting.NewScorerHarness(plugintesting.NewMockRecommendationScorer())
	harness.Start(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := harness.Client().ScoreRecommendations(ctx, scoreRequest("a"))
	require.NoError(t, err)
	assert.Len(t, resp.GetResults(), 1)

	harness.Stop()
	harness.Stop()
}
