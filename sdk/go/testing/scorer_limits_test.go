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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestScorerLimits_RoundTrip(t *testing.T) {
	signals := []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP}
	md := plugintesting.FormatScorerLimits(40, signals)
	assert.Equal(t, "40", md[plugintesting.ScorerMaxBatchSizeKey])
	assert.Equal(t, "risk,duplicate_group", md[plugintesting.ScorerSupportedSignalsKey])

	got, err := plugintesting.ParseScorerLimits(md)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int32(40), got.MaxBatchSize)
	assert.Equal(t, signals, got.SupportedSignals)
}

func TestParseScorerLimits_Invalid(t *testing.T) {
	const limit, sigs = plugintesting.ScorerMaxBatchSizeKey, plugintesting.ScorerSupportedSignalsKey
	tests := map[string]map[string]string{
		"limit only":      {limit: "5"},
		"signals only":    {sigs: "risk"},
		"zero limit":      {limit: "0", sigs: "risk"},
		"non numeric":     {limit: "many", sigs: "risk"},
		"empty signals":   {limit: "5", sigs: ""},
		"unknown signal":  {limit: "5", sigs: "risk,vibes"},
		"unspecified":     {limit: "5", sigs: "unspecified"},
		"repeated":        {limit: "5", sigs: "risk,risk"},
		"upper case":      {limit: "5", sigs: "RISK"},
		"spaces":          {limit: "5", sigs: "risk, priority"},
		"trailing comma":  {limit: "5", sigs: "risk,"},
		"limit overflows": {limit: "99999999999", sigs: "risk"},
	}
	for name, md := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := plugintesting.ParseScorerLimits(md)
			require.ErrorIs(t, err, plugintesting.ErrInvalidScorerLimits)
			assert.Nil(t, got)
		})
	}
}

func TestParseScorerLimits_NotAdvertised(t *testing.T) {
	got, err := plugintesting.ParseScorerLimits(map[string]string{"other": "x"})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestIsBatchTooLarge(t *testing.T) {
	recs := func(ids ...string) *pbc.ScoreRecommendationsRequest {
		req := &pbc.ScoreRecommendationsRequest{}
		for _, id := range ids {
			req.Recommendations = append(req.Recommendations, &pbc.Recommendation{Id: id})
		}
		return req
	}
	oversize := plugintesting.ValidateScoreRecommendationsRequest(recs("a", "b", "c"), 2)
	require.ErrorIs(t, oversize, plugintesting.ErrInvalidScoreRequest)
	assert.Equal(t, codes.InvalidArgument, status.Code(oversize))
	assert.True(t, plugintesting.IsBatchTooLarge(oversize))

	others := map[string]error{
		"empty":      plugintesting.ValidateScoreRecommendationsRequest(recs(), 2),
		"duplicates": plugintesting.ValidateScoreRecommendationsRequest(recs("a", "a"), 2),
		"plain":      status.Error(codes.InvalidArgument, "batch too large"),
		"other code": status.Error(codes.Unavailable, "down"),
		"nil":        nil,
	}
	for name, err := range others {
		assert.False(t, plugintesting.IsBatchTooLarge(err), name)
	}
}

func TestScorerConformance_AdvertisedLimits(t *testing.T) {
	run := func(t *testing.T, impl plugintesting.ScoreServer, md map[string]string) error {
		t.Helper()
		harness := plugintesting.NewScorerHarness(impl)
		harness.Start(t)
		defer harness.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return plugintesting.RunScorerAdvertisedLimitsForTest(ctx, harness.Client(), md)
	}
	mock := plugintesting.NewMockRecommendationScorer(plugintesting.WithScorerMaxBatchSize(7))

	require.NoError(t, run(t, mock, mock.AdvertisedScorerMetadata()))

	signals := []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK}
	advertised, err := plugintesting.ParseScorerLimits(mock.AdvertisedScorerMetadata())
	require.NoError(t, err)
	all := advertised.SupportedSignals
	require.Error(t, run(t, mock, plugintesting.FormatScorerLimits(8, nil)), "malformed signals")
	require.Error(t, run(t, mock, plugintesting.FormatScorerLimits(8, all)), "limit differs")
	require.Error(t, run(t, mock, plugintesting.FormatScorerLimits(7, signals)), "signals differ")
	require.Error(t, run(t, mock, map[string]string{}), "nothing advertised")
}

type servedInfoScorer struct {
	*plugintesting.MockRecommendationScorer

	metadata map[string]string
}

func (s servedInfoScorer) GetPluginInfo(
	context.Context, *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{Metadata: s.metadata}, nil
}

func TestScorerConformance_ServedLimits(t *testing.T) {
	mock := plugintesting.NewMockRecommendationScorer(plugintesting.WithScorerMaxBatchSize(7))
	run := func(md map[string]string) error {
		impl := servedInfoScorer{MockRecommendationScorer: mock, metadata: md}
		harness := plugintesting.NewScorerHarness(impl)
		harness.Start(t)
		defer harness.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return plugintesting.RunScorerServedLimitsForTest(ctx, harness.Client(), impl)
	}
	require.NoError(t, run(mock.AdvertisedScorerMetadata()), "served metadata matches the response")
	require.NoError(t, run(nil), "advertising is optional")
	mismatch := plugintesting.FormatScorerLimits(8, []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK})
	require.Error(t, run(mismatch), "served metadata differs from the response")
	require.Error(t, run(map[string]string{plugintesting.ScorerMaxBatchSizeKey: "7"}), "half-set pair")
}

func TestScorerConformance_OversizeNeedsDetail(t *testing.T) {
	ref := plugintesting.NewMockRecommendationScorer()
	plain := scoreFunc(func(
		ctx context.Context, req *pbc.ScoreRecommendationsRequest,
	) (*pbc.ScoreRecommendationsResponse, error) {
		if len(req.GetRecommendations()) > plugintesting.DefaultMockScorerBatchSize {
			return nil, status.Error(codes.InvalidArgument, "too many")
		}
		return ref.ScoreRecommendations(ctx, req)
	})
	results := runScorerScenarios(t, plain)
	require.Error(t, results["oversize_batch"])
	assert.Contains(t, results["oversize_batch"].Error(), plugintesting.BatchTooLargeReason)
}

func BenchmarkParseScorerLimits(b *testing.B) {
	md := plugintesting.FormatScorerLimits(40,
		[]pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP})
	b.ReportAllocs()
	for b.Loop() {
		if _, err := plugintesting.ParseScorerLimits(md); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsBatchTooLarge(b *testing.B) {
	req := &pbc.ScoreRecommendationsRequest{Recommendations: []*pbc.Recommendation{{Id: "a"}, {Id: "b"}}}
	err := plugintesting.ValidateScoreRecommendationsRequest(req, 1)
	b.ReportAllocs()
	for b.Loop() {
		if !plugintesting.IsBatchTooLarge(err) {
			b.Fatal("want batch too large")
		}
	}
}
