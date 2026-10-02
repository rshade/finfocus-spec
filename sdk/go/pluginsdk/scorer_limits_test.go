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

package pluginsdk_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestWithScorerLimits_GetPluginInfo(t *testing.T) {
	signals := []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY}
	info := pluginsdk.NewPluginInfo("scorer", "v1.0.0",
		pluginsdk.WithMetadata("build", "x"), pluginsdk.WithScorerLimits(12, signals...))
	require.NoError(t, info.Validate())

	logger := zerolog.Nop()
	server := pluginsdk.NewServerWithOptions(pluginsdk.NewBasePlugin("scorer"), nil, &logger, info)
	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Equal(t, "x", resp.GetMetadata()["build"])

	got, err := pluginsdk.ParseScorerLimits(resp.GetMetadata())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int32(12), got.MaxBatchSize)
	assert.Equal(t, signals, got.SupportedSignals)
}

func TestPluginInfoValidate_RejectsBadScorerLimits(t *testing.T) {
	bad := []pluginsdk.PluginInfoOption{
		pluginsdk.WithScorerLimits(0, pbc.ScoreSignal_SCORE_SIGNAL_RISK),
		pluginsdk.WithScorerLimits(5),
		pluginsdk.WithScorerLimits(5, pbc.ScoreSignal_SCORE_SIGNAL_UNSPECIFIED),
		pluginsdk.WithScorerLimits(5, pbc.ScoreSignal_SCORE_SIGNAL_RISK, pbc.ScoreSignal_SCORE_SIGNAL_RISK),
	}
	for i, opt := range bad {
		err := pluginsdk.NewPluginInfo("scorer", "v1.0.0", opt).Validate()
		require.ErrorIs(t, err, plugintesting.ErrInvalidScorerLimits, "case %d", i)
	}
	require.NoError(t, pluginsdk.NewPluginInfo("scorer", "v1.0.0").Validate())
}

func TestScorer_BatchTooLargeParity(t *testing.T) {
	for _, tr := range allocTransports {
		scorer := plugintesting.NewMockRecommendationScorer(plugintesting.WithScorerMaxBatchSize(2))
		plugin := newScorerTestPlugin(scorer)
		_, call := startScorerServer(t, plugin, tr.web)

		_, err := callScore(t, call, scorerFixtureRequest())
		require.Error(t, err, tr.name)
		if tr.web {
			var connectErr *connect.Error
			require.ErrorAs(t, err, &connectErr, tr.name)
			assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
			require.Len(t, connectErr.Details(), 1, tr.name)
		} else {
			assert.True(t, pluginsdk.IsBatchTooLarge(err), tr.name)
		}

		dup := &pbc.ScoreRecommendationsRequest{Recommendations: []*pbc.Recommendation{{Id: "a"}, {Id: "a"}}}
		_, err = callScore(t, call, dup)
		require.Error(t, err, tr.name)
		assert.False(t, pluginsdk.IsBatchTooLarge(err), tr.name)
	}
}
