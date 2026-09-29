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
	"slices"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	// DefaultMockScorerBatchSize is the max_batch_size a MockRecommendationScorer
	// advertises unless WithScorerMaxBatchSize overrides it.
	DefaultMockScorerBatchSize = 25

	mockScorerName        = "mock-rules"
	mockRiskBase          = 0.2
	mockRiskDestructive   = 0.5
	mockRiskCommitment    = 0.7
	mockRiskProduction    = 0.2
	mockFalsePositiveHigh = 0.9
	mockFalsePositiveLow  = 0.1
	mockWorthHigh         = 0.9
	mockWorthMedium       = 0.5
	mockWorthLow          = 0.1
	mockSavingsHigh       = 100.0
	mockSavingsMedium     = 10.0
	mockEvidenceThin      = 0.9
	mockEvidenceEnough    = 0.1
)

// MockRecommendationScorer is the reference implementation of
// RecommendationScorerService.ScoreRecommendations. It scores with fixed
// rules over the recommendation's action, tags, savings, priority, utilization
// and reasoning, so results are deterministic and no model or network is
// involved. It passes RunScorerConformance and is safe for concurrent use.
//
// It is a separate type rather than a MockPlugin method so that MockPlugin's
// inferred capabilities and served services do not change.
type MockRecommendationScorer struct {
	maxBatchSize int32
	signals      []pbc.ScoreSignal
}

// MockScorerOption configures a MockRecommendationScorer.
type MockScorerOption func(*MockRecommendationScorer)

// WithScorerMaxBatchSize sets the advertised and enforced max_batch_size. A
// value below one is ignored.
func WithScorerMaxBatchSize(n int32) MockScorerOption {
	return func(m *MockRecommendationScorer) {
		if n >= 1 {
			m.maxBatchSize = n
		}
	}
}

// WithScorerSignals restricts the scorer to the given signals, so tests can
// exercise a partial implementation. Unspecified and repeated values are
// dropped; with no valid signal the scorer keeps all of them.
func WithScorerSignals(signals ...pbc.ScoreSignal) MockScorerOption {
	return func(m *MockRecommendationScorer) {
		var kept []pbc.ScoreSignal
		for _, signal := range signals {
			if isDefinedScoreSignal(signal) && !slices.Contains(kept, signal) {
				kept = append(kept, signal)
			}
		}
		if len(kept) > 0 {
			slices.Sort(kept)
			m.signals = kept
		}
	}
}

// NewMockRecommendationScorer returns a scorer that supports every signal with
// a batch limit of DefaultMockScorerBatchSize, adjusted by opts.
func NewMockRecommendationScorer(opts ...MockScorerOption) *MockRecommendationScorer {
	m := &MockRecommendationScorer{
		maxBatchSize: DefaultMockScorerBatchSize,
		signals: []pbc.ScoreSignal{
			pbc.ScoreSignal_SCORE_SIGNAL_RISK,
			pbc.ScoreSignal_SCORE_SIGNAL_FALSE_POSITIVE,
			pbc.ScoreSignal_SCORE_SIGNAL_WORTH_ACTING,
			pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY,
			pbc.ScoreSignal_SCORE_SIGNAL_INSUFFICIENT_EVIDENCE,
			pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP,
		},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ScoreRecommendations validates req with ValidateScoreRecommendationsRequest
// against the scorer's batch limit, rejects requested signals the scorer does
// not support, and scores each recommendation for the requested (or every
// supported) signal. A recommendation without a resource fails on its own with
// a ResourceError of code InvalidArgument. Recommendations that share
// resource.id and action_type receive one duplicate_group_id, unless
// identifier_mode is IDENTIFIER_MODE_OMITTED, when identifiers cannot be
// trusted and no grouping is attempted. Invalid requests fail with
// codes.InvalidArgument.
func (m *MockRecommendationScorer) ScoreRecommendations(
	_ context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	if err := ValidateScoreRecommendationsRequest(req, m.maxBatchSize); err != nil {
		return nil, err
	}
	active := m.signals
	if len(req.GetSignals()) > 0 {
		for _, signal := range req.GetSignals() {
			if !slices.Contains(m.signals, signal) {
				return nil, newInvalidArgument(ErrInvalidScoreRequest, "signal %s is not supported", signal)
			}
		}
		active = req.GetSignals()
	}

	resp := &pbc.ScoreRecommendationsResponse{
		Results:          make([]*pbc.RecommendationScoreResult, len(req.GetRecommendations())),
		MaxBatchSize:     m.maxBatchSize,
		SupportedSignals: slices.Clone(m.signals),
		Scorer: &pbc.ScorerInfo{
			Name:        mockScorerName,
			Calibration: pbc.ScoreCalibration_SCORE_CALIBRATION_RANKING_ONLY,
		},
	}
	for i, rec := range req.GetRecommendations() {
		resp.Results[i] = mockScoreResult(rec, active)
	}
	if slices.Contains(active, pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP) &&
		req.GetIdentifierMode() != pbc.IdentifierMode_IDENTIFIER_MODE_OMITTED {
		assignDuplicateGroups(req.GetRecommendations(), resp.GetResults())
	}
	return resp, nil
}

func mockScoreResult(rec *pbc.Recommendation, active []pbc.ScoreSignal) *pbc.RecommendationScoreResult {
	result := &pbc.RecommendationScoreResult{RecommendationId: rec.GetId()}
	if rec.GetResource() == nil {
		result.Result = &pbc.RecommendationScoreResult_Error{Error: &pbc.ResourceError{
			Code:    int32(codes.InvalidArgument),
			Message: "recommendation has no resource",
		}}
		return result
	}
	scores := &pbc.RecommendationScores{}
	for _, signal := range active {
		switch signal {
		case pbc.ScoreSignal_SCORE_SIGNAL_RISK:
			scores.Risk = proto.Float64(mockRisk(rec))
		case pbc.ScoreSignal_SCORE_SIGNAL_FALSE_POSITIVE:
			scores.FalsePositive = proto.Float64(mockFalsePositive(rec))
		case pbc.ScoreSignal_SCORE_SIGNAL_WORTH_ACTING:
			scores.WorthActing = proto.Float64(mockWorthActing(rec))
		case pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY:
			scores.Priority = proto.Float64(min(float64(rec.GetPriority()), maxPriorityScore))
		case pbc.ScoreSignal_SCORE_SIGNAL_INSUFFICIENT_EVIDENCE:
			scores.InsufficientEvidence = proto.Float64(mockInsufficientEvidence(rec))
		case pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP, pbc.ScoreSignal_SCORE_SIGNAL_UNSPECIFIED:
		}
	}
	result.Result = &pbc.RecommendationScoreResult_Scores{Scores: scores}
	return result
}

func mockRisk(rec *pbc.Recommendation) float64 {
	risk := mockRiskBase
	//nolint:exhaustive // Every other action keeps the base risk.
	switch rec.GetActionType() {
	case pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_PURCHASE_COMMITMENT:
		risk = mockRiskCommitment
	case pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_TERMINATE,
		pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_DELETE_UNUSED:
		risk = mockRiskDestructive
	}
	if rec.GetResource().GetTags()["environment"] == "production" {
		risk += mockRiskProduction
	}
	return min(risk, maxUnitScore)
}

func mockFalsePositive(rec *pbc.Recommendation) float64 {
	switch rec.GetResource().GetTags()["purpose"] {
	case "standby", "seasonal":
		return mockFalsePositiveHigh
	default:
		return mockFalsePositiveLow
	}
}

func mockWorthActing(rec *pbc.Recommendation) float64 {
	switch savings := rec.GetImpact().GetEstimatedSavings(); {
	case savings >= mockSavingsHigh:
		return mockWorthHigh
	case savings >= mockSavingsMedium:
		return mockWorthMedium
	default:
		return mockWorthLow
	}
}

func mockInsufficientEvidence(rec *pbc.Recommendation) float64 {
	if rec.GetResource().GetUtilization() == nil && len(rec.GetReasoning()) == 0 {
		return mockEvidenceThin
	}
	return mockEvidenceEnough
}

// assignDuplicateGroups gives recommendations that share resource.id and
// action_type one "dup-N" id, numbered by first appearance. A key held by a
// single scored recommendation gets no group.
func assignDuplicateGroups(recs []*pbc.Recommendation, results []*pbc.RecommendationScoreResult) {
	type key struct {
		resource string
		action   pbc.RecommendationActionType
	}
	members := make(map[key][]int)
	var order []key
	for i, rec := range recs {
		if results[i].GetScores() == nil || rec.GetResource().GetId() == "" {
			continue
		}
		k := key{rec.GetResource().GetId(), rec.GetActionType()}
		if _, seen := members[k]; !seen {
			order = append(order, k)
		}
		members[k] = append(members[k], i)
	}
	group := 0
	for _, k := range order {
		if len(members[k]) < minDuplicateGroupSize {
			continue
		}
		group++
		id := "dup-" + strconv.Itoa(group)
		for _, i := range members[k] {
			results[i].GetScores().DuplicateGroupId = id
		}
	}
}
