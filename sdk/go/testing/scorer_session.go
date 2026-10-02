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

package testing

import (
	"context"
	"fmt"
	"slices"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	sessionConformanceA = "conformance-session-a"
	sessionConformanceB = "conformance-session-b"
)

// sessionPair is the fixture's duplicate pair: the last two recommendations
// share a resource and action type.
func sessionPair() (*pbc.Recommendation, *pbc.Recommendation) {
	recs := scorerFixture(scorerFixtureSize)
	return recs[scorerFixtureSize-2], recs[scorerFixtureSize-1]
}

// groupIDsInOneCall scores the pair together in one call of the session and
// returns the two group ids. ok is false when the scorer does not take part in
// session grouping here: it supports fewer than two recommendations per call,
// does not support SCORE_SIGNAL_DUPLICATE_GROUP, does not echo the session, or
// does not group the pair.
func groupIDsInOneCall(
	ctx context.Context, client pbc.RecommendationScorerServiceClient, session string,
) (string, bool, error) {
	probe, err := probeScorer(ctx, client)
	if err != nil {
		return "", false, err
	}
	if probe.GetMaxBatchSize() < minDuplicateGroupSize ||
		!slices.Contains(probe.GetSupportedSignals(), pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP) {
		return "", false, nil
	}
	first, second := sessionPair()
	resp, err := scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{
		SessionId:       session,
		Recommendations: []*pbc.Recommendation{first, second},
	})
	if err != nil {
		return "", false, err
	}
	id := resp.GetResults()[0].GetScores().GetDuplicateGroupId()
	if resp.GetSessionId() != session || id == "" ||
		id != resp.GetResults()[1].GetScores().GetDuplicateGroupId() {
		return "", false, nil
	}
	return id, true, nil
}

// groupIDsAcrossCalls scores each member of the pair in its own call of the
// session and returns their group ids.
func groupIDsAcrossCalls(
	ctx context.Context, client pbc.RecommendationScorerServiceClient, session string,
) ([]string, error) {
	first, second := sessionPair()
	ids := make([]string, 0, minDuplicateGroupSize)
	for _, rec := range []*pbc.Recommendation{first, second} {
		resp, err := scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{
			SessionId:       session,
			Recommendations: []*pbc.Recommendation{rec},
		})
		if err != nil {
			return nil, err
		}
		ids = append(ids, resp.GetResults()[0].GetScores().GetDuplicateGroupId())
	}
	return ids, nil
}

// scorerCheckSessionEcho requires the response to echo a valid session_id or
// leave it empty.
func scorerCheckSessionEcho(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	_, err := scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{
		SessionId:       sessionConformanceA,
		Recommendations: scorerFixture(1),
	})
	return err
}

// scorerCheckSessionAcrossBatches requires a pair that one call groups to get
// that same group id when its members are scored in separate calls of one
// session. A scorer that does not group the pair in one call passes.
func scorerCheckSessionAcrossBatches(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	together, ok, err := groupIDsInOneCall(ctx, client, sessionConformanceA)
	if err != nil || !ok {
		return err
	}
	apart, err := groupIDsAcrossCalls(ctx, client, sessionConformanceA)
	if err != nil {
		return err
	}
	for i, id := range apart {
		if id != together {
			return fmt.Errorf("session %q: duplicate_group_id %q for batch %d, want %q as in one call",
				sessionConformanceA, id, i+1, together)
		}
	}
	return nil
}

// scorerCheckSessionIsolation requires the same pair to get a different group
// id in a different session.
func scorerCheckSessionIsolation(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	inA, ok, err := groupIDsInOneCall(ctx, client, sessionConformanceA)
	if err != nil || !ok {
		return err
	}
	inB, ok, err := groupIDsInOneCall(ctx, client, sessionConformanceB)
	if err != nil || !ok {
		return err
	}
	if inA == inB {
		return fmt.Errorf("sessions %q and %q both produced duplicate_group_id %q",
			sessionConformanceA, sessionConformanceB, inA)
	}
	return nil
}
