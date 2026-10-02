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
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	// maxUnitScore is the upper bound of risk, false_positive, worth_acting and
	// insufficient_evidence.
	maxUnitScore = 1.0
	// maxPriorityScore is the upper bound of priority.
	maxPriorityScore = 3.0
	// minDuplicateGroupSize is the fewest recommendations a duplicate group may hold.
	minDuplicateGroupSize = 2
	// maxSessionIDLength is the longest session_id a request may carry.
	maxSessionIDLength = 128
)

var (
	// ErrInvalidScoreRequest is wrapped by every ValidateScoreRecommendationsRequest failure.
	ErrInvalidScoreRequest = errors.New("invalid score recommendations request")
	// ErrInvalidScoreResponse is wrapped by every ValidateScoreRecommendationsResponse failure.
	ErrInvalidScoreResponse = errors.New("invalid score recommendations response")
)

// ValidateScoreRecommendationsRequest checks a scorer's input: at least one
// recommendation, none nil, each with a distinct non-empty id, no more than
// maxBatchSize entries when maxBatchSize is positive, only defined signals
// (SCORE_SIGNAL_UNSPECIFIED is invalid), and a defined identifier_mode. Every
// failure wraps ErrInvalidScoreRequest and carries codes.InvalidArgument, so a
// scorer may return it directly. A scorer that also rejects signals it does
// not support does so against its own supported list. A non-empty session_id
// must be at most 128 printable ASCII characters.
func ValidateScoreRecommendationsRequest(req *pbc.ScoreRecommendationsRequest, maxBatchSize int32) error {
	if req == nil {
		return newInvalidArgument(ErrInvalidScoreRequest, "request is nil")
	}
	recs := req.GetRecommendations()
	if len(recs) == 0 {
		return newInvalidArgument(ErrInvalidScoreRequest, "no recommendations to score")
	}
	if maxBatchSize > 0 && len(recs) > int(maxBatchSize) {
		return newInvalidArgument(ErrInvalidScoreRequest,
			"%d recommendations exceed max_batch_size %d", len(recs), maxBatchSize)
	}
	seen := make(map[string]int, len(recs))
	for i, rec := range recs {
		if rec == nil {
			return newInvalidArgument(ErrInvalidScoreRequest, "recommendations[%d] is nil", i)
		}
		if rec.GetId() == "" {
			return newInvalidArgument(ErrInvalidScoreRequest, "recommendations[%d] has an empty id", i)
		}
		if first, dup := seen[rec.GetId()]; dup {
			return newInvalidArgument(ErrInvalidScoreRequest,
				"recommendations[%d] duplicates recommendations[%d] (id %q)", i, first, rec.GetId())
		}
		seen[rec.GetId()] = i
	}
	for i, signal := range req.GetSignals() {
		if !isDefinedScoreSignal(signal) {
			return newInvalidArgument(ErrInvalidScoreRequest, "signals[%d] is %s", i, signal)
		}
	}
	if _, ok := pbc.IdentifierMode_name[int32(req.GetIdentifierMode())]; !ok {
		return newInvalidArgument(ErrInvalidScoreRequest, "identifier_mode %d is not defined", req.GetIdentifierMode())
	}
	if !isValidSessionID(req.GetSessionId()) {
		return newInvalidArgument(ErrInvalidScoreRequest,
			"session_id must be at most %d printable ASCII characters", maxSessionIDLength)
	}
	return nil
}

// isValidSessionID reports whether id is empty or 1 to maxSessionIDLength
// printable ASCII characters (0x21 to 0x7E).
func isValidSessionID(id string) bool {
	if len(id) > maxSessionIDLength {
		return false
	}
	for i := range len(id) {
		if id[i] < '!' || id[i] > '~' {
			return false
		}
	}
	return true
}

// ValidateScoreRecommendationsResponse checks a scorer's output against the
// request it answers. Hosts call it on every response before using a score.
// It verifies that results are index-aligned with the request and echo each
// recommendation id; that every result holds scores or a ResourceError whose
// code is not OK; that max_batch_size is positive and covers the request; that
// supported_signals and scorer.calibration hold defined values; that every
// signal set is supported and, when the request named signals, requested; that
// requested signals are supported; that numeric signals are finite and within
// [0, 1] (priority within [0, 3]); and that each non-empty duplicate_group_id
// is shared by at least two recommendations, unless the request carries a
// session_id the response echoes, when a response may hold one member of a
// group because the rest are in other batches. A non-empty response
// session_id must equal the request's. Every failure wraps
// ErrInvalidScoreResponse.
func ValidateScoreRecommendationsResponse(
	req *pbc.ScoreRecommendationsRequest, resp *pbc.ScoreRecommendationsResponse,
) error {
	if resp == nil {
		return fmt.Errorf("%w: response is nil", ErrInvalidScoreResponse)
	}
	if err := validateScoreEnvelope(req, resp); err != nil {
		return err
	}
	supported := scoreSignalSet(resp.GetSupportedSignals())
	for _, signal := range req.GetSignals() {
		if !supported[signal] {
			return fmt.Errorf("%w: %s was requested but is not supported", ErrInvalidScoreResponse, signal)
		}
	}
	allowed := supported
	if len(req.GetSignals()) > 0 {
		allowed = scoreSignalSet(req.GetSignals())
	}

	groups := make(map[string]int)
	for i, result := range resp.GetResults() {
		want := req.GetRecommendations()[i].GetId()
		if result.GetRecommendationId() != want {
			return fmt.Errorf("%w: results[%d].recommendation_id %q, want %q",
				ErrInvalidScoreResponse, i, result.GetRecommendationId(), want)
		}
		if err := validateScoreResult(i, result, supported, allowed, groups); err != nil {
			return err
		}
	}
	if req.GetSessionId() != "" && resp.GetSessionId() == req.GetSessionId() {
		return nil
	}
	for _, result := range resp.GetResults() {
		group := result.GetScores().GetDuplicateGroupId()
		if group != "" && groups[group] < minDuplicateGroupSize {
			return fmt.Errorf("%w: duplicate_group_id %q is used by only one recommendation",
				ErrInvalidScoreResponse, group)
		}
	}
	return nil
}

func validateScoreEnvelope(req *pbc.ScoreRecommendationsRequest, resp *pbc.ScoreRecommendationsResponse) error {
	n := len(req.GetRecommendations())
	if got := len(resp.GetResults()); got != n {
		return fmt.Errorf("%w: %d results for %d recommendations", ErrInvalidScoreResponse, got, n)
	}
	if resp.GetMaxBatchSize() < 1 {
		return fmt.Errorf("%w: max_batch_size %d, want at least 1", ErrInvalidScoreResponse, resp.GetMaxBatchSize())
	}
	if n > int(resp.GetMaxBatchSize()) {
		return fmt.Errorf("%w: %d recommendations exceed the response's max_batch_size %d",
			ErrInvalidScoreResponse, n, resp.GetMaxBatchSize())
	}
	if resp.GetSessionId() != "" && resp.GetSessionId() != req.GetSessionId() {
		return fmt.Errorf("%w: session_id %q does not match the request session_id %q",
			ErrInvalidScoreResponse, resp.GetSessionId(), req.GetSessionId())
	}
	seen := make(map[pbc.ScoreSignal]struct{}, len(resp.GetSupportedSignals()))
	for i, signal := range resp.GetSupportedSignals() {
		if !isDefinedScoreSignal(signal) {
			return fmt.Errorf("%w: supported_signals[%d] is %s", ErrInvalidScoreResponse, i, signal)
		}
		if _, dup := seen[signal]; dup {
			return fmt.Errorf("%w: supported_signals[%d] repeats %s", ErrInvalidScoreResponse, i, signal)
		}
		seen[signal] = struct{}{}
	}
	if _, ok := pbc.ScoreCalibration_name[int32(resp.GetScorer().GetCalibration())]; !ok {
		return fmt.Errorf("%w: scorer.calibration %d is not defined",
			ErrInvalidScoreResponse, resp.GetScorer().GetCalibration())
	}
	return nil
}

func validateScoreResult(
	i int,
	result *pbc.RecommendationScoreResult,
	supported, allowed map[pbc.ScoreSignal]bool,
	groups map[string]int,
) error {
	switch outcome := result.GetResult().(type) {
	case *pbc.RecommendationScoreResult_Error:
		if outcome.Error == nil || outcome.Error.GetCode() == int32(codes.OK) {
			return fmt.Errorf("%w: results[%d].error.code must not be OK", ErrInvalidScoreResponse, i)
		}
		return nil
	case *pbc.RecommendationScoreResult_Scores:
		if outcome.Scores == nil {
			return fmt.Errorf("%w: results[%d] has neither scores nor error", ErrInvalidScoreResponse, i)
		}
		return validateScores(i, outcome.Scores, supported, allowed, groups)
	default:
		return fmt.Errorf("%w: results[%d] has neither scores nor error", ErrInvalidScoreResponse, i)
	}
}

func validateScores(
	i int, scores *pbc.RecommendationScores, supported, allowed map[pbc.ScoreSignal]bool, groups map[string]int,
) error {
	//nolint:protogetter // Optional signals need the pointer to tell unset from zero.
	numeric := []struct {
		name   string
		signal pbc.ScoreSignal
		value  *float64
		max    float64
	}{
		{"risk", pbc.ScoreSignal_SCORE_SIGNAL_RISK, scores.Risk, maxUnitScore},
		{"false_positive", pbc.ScoreSignal_SCORE_SIGNAL_FALSE_POSITIVE, scores.FalsePositive, maxUnitScore},
		{"worth_acting", pbc.ScoreSignal_SCORE_SIGNAL_WORTH_ACTING, scores.WorthActing, maxUnitScore},
		{"priority", pbc.ScoreSignal_SCORE_SIGNAL_PRIORITY, scores.Priority, maxPriorityScore},
		{
			"insufficient_evidence", pbc.ScoreSignal_SCORE_SIGNAL_INSUFFICIENT_EVIDENCE,
			scores.InsufficientEvidence, maxUnitScore,
		},
	}
	for _, field := range numeric {
		if field.value == nil {
			continue
		}
		if err := checkScoreSignalAllowed(i, field.name, field.signal, supported, allowed); err != nil {
			return err
		}
		if v := *field.value; !isFinite(v) || v < 0 || v > field.max {
			return fmt.Errorf("%w: results[%d].%s is %v, want a value in [0, %v]",
				ErrInvalidScoreResponse, i, field.name, v, field.max)
		}
	}
	if group := scores.GetDuplicateGroupId(); group != "" {
		err := checkScoreSignalAllowed(i, "duplicate_group_id", pbc.ScoreSignal_SCORE_SIGNAL_DUPLICATE_GROUP,
			supported, allowed)
		if err != nil {
			return err
		}
		groups[group]++
	}
	return nil
}

func checkScoreSignalAllowed(
	i int, field string, signal pbc.ScoreSignal, supported, allowed map[pbc.ScoreSignal]bool,
) error {
	if !supported[signal] {
		return fmt.Errorf("%w: results[%d].%s is set but %s is not supported",
			ErrInvalidScoreResponse, i, field, signal)
	}
	if !allowed[signal] {
		return fmt.Errorf("%w: results[%d].%s is set but %s was not requested",
			ErrInvalidScoreResponse, i, field, signal)
	}
	return nil
}

func isDefinedScoreSignal(signal pbc.ScoreSignal) bool {
	if signal == pbc.ScoreSignal_SCORE_SIGNAL_UNSPECIFIED {
		return false
	}
	_, ok := pbc.ScoreSignal_name[int32(signal)]
	return ok
}

func scoreSignalSet(signals []pbc.ScoreSignal) map[pbc.ScoreSignal]bool {
	set := make(map[pbc.ScoreSignal]bool, len(signals))
	for _, signal := range signals {
		set[signal] = true
	}
	return set
}
